package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"mongoscale/internal/config"
	"mongoscale/internal/database"
	"mongoscale/internal/metrics"
	"mongoscale/internal/models"
	"mongoscale/pkg/bsonx"
)

const (
	seedPoolSize     = 20
	historyDays      = 365
	operationTimeout = 10 * time.Minute
	maxOrderLines    = 4
)

type seedProduct struct {
	id       primitive.ObjectID
	name     string
	price    float64
	category string
}

type seeder struct {
	cfg    *config.Config
	db     *database.Manager
	random *rand.Rand
	logger *slog.Logger
	now    time.Time
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	db := database.NewManager(cfg, logger, metrics.NewMongoOps(time.Hour))
	if err := connectWithRetry(ctx, db, logger); err != nil {
		return err
	}
	defer db.Close(context.Background())

	worker := &seeder{
		cfg:    cfg,
		db:     db,
		random: rand.New(rand.NewSource(cfg.Seed.RandomSeed)),
		logger: logger,
		now:    time.Now().UTC(),
	}
	return worker.execute(ctx)
}

func connectWithRetry(ctx context.Context, db *database.Manager, logger *slog.Logger) error {
	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		lastErr = db.Connect(ctx, database.Profile{MaxPoolSize: seedPoolSize, MinPoolSize: 2})
		if lastErr == nil {
			return nil
		}
		logger.Warn("waiting for mongodb", "attempt", attempt, "error", lastErr.Error())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	return lastErr
}

func (s *seeder) execute(ctx context.Context) error {
	started := time.Now()
	settings := s.cfg.Seed

	if settings.Reset {
		s.logger.Info("resetting collections")
		for _, name := range []string{"orders", "reviews", "products", "users", "loadtest_results"} {
			if err := s.db.Plain(name).Drop(ctx); err != nil {
				return fmt.Errorf("drop %s: %w", name, err)
			}
		}
	}
	if err := s.db.EnsureCoreIndexes(ctx); err != nil {
		return err
	}
	if settings.ShardOrders && s.db.Topology().Sharded() {
		if err := s.db.ShardOrders(ctx); err != nil {
			return err
		}
		s.logger.Info("orders collection sharded on hashed userId")
	}

	userIDs, usersRebuilt, err := s.ensureUsers(ctx)
	if err != nil {
		return err
	}
	products, productsRebuilt, err := s.ensureProducts(ctx)
	if err != nil {
		return err
	}
	rebuildDependents := usersRebuilt || productsRebuilt
	if err := s.ensureOrders(ctx, userIDs, products, rebuildDependents); err != nil {
		return err
	}
	if err := s.ensureReviews(ctx, userIDs, products, rebuildDependents); err != nil {
		return err
	}
	s.logger.Info("seed complete", "durationSeconds", time.Since(started).Seconds())
	return nil
}

func (s *seeder) insertInBatches(ctx context.Context, collection string, total int, build func(index int) any) error {
	batch := s.cfg.Seed.BatchSize
	for offset := 0; offset < total; offset += batch {
		end := min(offset+batch, total)
		operations := make([]mongo.WriteModel, 0, end-offset)
		for index := offset; index < end; index++ {
			operations = append(operations, mongo.NewInsertOneModel().SetDocument(build(index)))
		}
		if _, err := s.db.Writer(collection, false).BulkWrite(ctx, operations, options.BulkWrite().SetOrdered(false)); err != nil {
			return fmt.Errorf("bulk insert into %s: %w", collection, err)
		}
		s.logger.Info("seeding progress", "collection", collection, "inserted", end, "total", total)
	}
	return nil
}

func (s *seeder) randomPast() time.Time {
	seconds := s.random.Int63n(int64(historyDays) * 24 * 3600)
	return s.now.Add(-time.Duration(seconds) * time.Second)
}

func (s *seeder) ensureUsers(ctx context.Context) ([]primitive.ObjectID, bool, error) {
	target := s.cfg.Seed.Users
	filter := bsonx.D(bsonx.E("role", models.RoleCustomer))
	collection := s.db.Plain("users")
	existing, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, false, err
	}
	rebuilt := false
	if int(existing) < target {
		if existing > 0 {
			s.logger.Info("user collection incomplete, rebuilding", "existing", existing, "target", target)
			if _, err := collection.DeleteMany(ctx, filter); err != nil {
				return nil, false, err
			}
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.Seed.UserPassword), s.cfg.Seed.BcryptCost)
		if err != nil {
			return nil, false, err
		}
		err = s.insertInBatches(ctx, "users", target, func(index int) any {
			created := s.randomPast()
			return models.User{
				Name:         firstNames[s.random.Intn(len(firstNames))] + " " + lastNames[s.random.Intn(len(lastNames))],
				Email:        fmt.Sprintf("user%05d@example.com", index+1),
				PasswordHash: string(hash),
				Role:         models.RoleCustomer,
				CreatedAt:    created,
				UpdatedAt:    created,
			}
		})
		if err != nil {
			return nil, false, err
		}
		rebuilt = true
	} else {
		s.logger.Info("users already seeded", "count", existing)
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetProjection(bsonx.D(bsonx.E("_id", 1))).SetLimit(int64(target)))
	if err != nil {
		return nil, false, err
	}
	var refs []models.UserRef
	if err := cursor.All(ctx, &refs); err != nil {
		return nil, false, err
	}
	ids := make([]primitive.ObjectID, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.ID)
	}
	return ids, rebuilt, nil
}

func (s *seeder) ensureProducts(ctx context.Context) ([]seedProduct, bool, error) {
	target := s.cfg.Seed.Products
	collection := s.db.Plain("products")
	existing, err := collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		return nil, false, err
	}
	rebuilt := false
	if int(existing) < target {
		if existing > 0 {
			s.logger.Info("product collection incomplete, rebuilding", "existing", existing, "target", target)
			if err := collection.Drop(ctx); err != nil {
				return nil, false, err
			}
		}
		err := s.insertInBatches(ctx, "products", target, func(index int) any {
			spec := categories[index%len(categories)]
			noun := spec.nouns[s.random.Intn(len(spec.nouns))]
			adjective := adjectives[s.random.Intn(len(adjectives))]
			price := spec.minPrice + s.random.Float64()*(spec.maxPrice-spec.minPrice)
			created := s.randomPast()
			return models.Product{
				Name:        fmt.Sprintf("%s %s %c%d", adjective, noun, 'A'+rune(s.random.Intn(26)), 100+s.random.Intn(900)),
				Description: fmt.Sprintf("%s %s from the %s range. Built for everyday use with a focus on durability, value and design.", adjective, noun, spec.name),
				Category:    spec.name,
				Price:       math.Round(price*100) / 100,
				Stock:       50000 + s.random.Intn(150000),
				ImageURL:    fmt.Sprintf("https://picsum.photos/seed/mongoscale-%d/480/360", index+1),
				CreatedAt:   created,
				UpdatedAt:   created,
			}
		})
		if err != nil {
			return nil, false, err
		}
		rebuilt = true
	} else {
		s.logger.Info("products already seeded", "count", existing)
	}

	cursor, err := collection.Find(ctx, bson.D{}, options.Find().SetLimit(int64(target)))
	if err != nil {
		return nil, false, err
	}
	var stored []models.Product
	if err := cursor.All(ctx, &stored); err != nil {
		return nil, false, err
	}
	products := make([]seedProduct, 0, len(stored))
	for _, product := range stored {
		products = append(products, seedProduct{id: product.ID, name: product.Name, price: product.Price, category: product.Category})
	}
	return products, rebuilt, nil
}

func (s *seeder) weightedIndex(weights []int) int {
	total := 0
	for _, weight := range weights {
		total += weight
	}
	roll := s.random.Intn(total)
	for index, weight := range weights {
		if roll < weight {
			return index
		}
		roll -= weight
	}
	return len(weights) - 1
}

func (s *seeder) ensureOrders(ctx context.Context, userIDs []primitive.ObjectID, products []seedProduct, force bool) error {
	target := s.cfg.Seed.Orders
	collection := s.db.Plain("orders")
	existing, err := collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	if int(existing) >= target && !force {
		s.logger.Info("orders already seeded", "count", existing)
		return nil
	}
	if existing > 0 {
		s.logger.Info("rebuilding orders", "existing", existing, "target", target)
		if _, err := collection.DeleteMany(ctx, bson.D{}); err != nil {
			return err
		}
	}

	statusWeights := make([]int, len(orderStatusMix))
	for index, entry := range orderStatusMix {
		statusWeights[index] = entry.weight
	}
	return s.insertInBatches(ctx, "orders", target, func(int) any {
		lineCount := 1 + s.random.Intn(maxOrderLines)
		items := make([]models.OrderItem, 0, lineCount)
		used := make(map[int]bool, lineCount)
		total := 0.0
		for len(items) < lineCount {
			position := s.random.Intn(len(products))
			if used[position] {
				continue
			}
			used[position] = true
			quantity := 1 + s.random.Intn(3)
			product := products[position]
			items = append(items, models.OrderItem{ProductID: product.id, Name: product.name, Price: product.price, Quantity: quantity})
			total += product.price * float64(quantity)
		}
		mix := orderStatusMix[s.weightedIndex(statusWeights)]
		location := cities[s.random.Intn(len(cities))]
		created := s.randomPast()
		return models.Order{
			UserID:        userIDs[s.random.Intn(len(userIDs))],
			Items:         items,
			TotalAmount:   math.Round(total*100) / 100,
			Status:        mix.status,
			PaymentStatus: mix.payment,
			ShippingAddress: models.ShippingAddress{
				FullName:   firstNames[s.random.Intn(len(firstNames))] + " " + lastNames[s.random.Intn(len(lastNames))],
				Line1:      fmt.Sprintf("%d Market Road", 1+s.random.Intn(300)),
				City:       location.city,
				State:      location.state,
				PostalCode: location.zip,
				Country:    "India",
			},
			CreatedAt: created,
			UpdatedAt: created.Add(time.Duration(s.random.Intn(72)) * time.Hour),
		}
	})
}

func (s *seeder) ensureReviews(ctx context.Context, userIDs []primitive.ObjectID, products []seedProduct, force bool) error {
	target := s.cfg.Seed.Reviews
	collection := s.db.Plain("reviews")
	existing, err := collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	if int(existing) >= target && !force {
		s.logger.Info("reviews already seeded", "count", existing)
		return nil
	}
	if existing > 0 {
		s.logger.Info("rebuilding reviews", "existing", existing, "target", target)
		if _, err := collection.DeleteMany(ctx, bson.D{}); err != nil {
			return err
		}
	}

	ratingSums := make([]int, len(products))
	ratingCounts := make([]int, len(products))
	err = s.insertInBatches(ctx, "reviews", target, func(int) any {
		position := s.random.Intn(len(products))
		rating := 1 + s.weightedIndex(ratingWeights)
		ratingSums[position] += rating
		ratingCounts[position]++
		created := s.randomPast()
		return models.Review{
			ProductID: products[position].id,
			UserID:    userIDs[s.random.Intn(len(userIDs))],
			Rating:    rating,
			Comment:   reviewComments[s.random.Intn(len(reviewComments))],
			CreatedAt: created,
			UpdatedAt: created,
		}
	})
	if err != nil {
		return err
	}

	operations := make([]mongo.WriteModel, 0, len(products))
	for position, product := range products {
		average := 0.0
		if ratingCounts[position] > 0 {
			average = math.Round(float64(ratingSums[position])/float64(ratingCounts[position])*100) / 100
		}
		operations = append(operations, mongo.NewUpdateOneModel().
			SetFilter(bsonx.D(bsonx.E("_id", product.id))).
			SetUpdate(bsonx.D(bsonx.E("$set", bsonx.D(
				bsonx.E("rating", average),
				bsonx.E("ratingCount", ratingCounts[position]),
			)))))
	}
	if _, err := s.db.Plain("products").BulkWrite(ctx, operations, options.BulkWrite().SetOrdered(false)); err != nil {
		return fmt.Errorf("update product ratings: %w", err)
	}
	s.logger.Info("product ratings aggregated from reviews", "products", len(products))
	return nil
}
