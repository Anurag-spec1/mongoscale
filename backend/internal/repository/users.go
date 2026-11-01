package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mongoscale/internal/database"
	"mongoscale/internal/models"
	"mongoscale/internal/utils"
	"mongoscale/pkg/bsonx"
)

type UserRepository struct {
	db *database.Manager
}

func NewUserRepository(db *database.Manager) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	result, err := r.db.Writer("users", true).InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return utils.Conflict("an account with this email already exists")
		}
		return err
	}
	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	var user models.User
	err := r.db.Plain("users").FindOne(ctx, bsonx.D(bsonx.E("email", email))).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	var user models.User
	err := r.db.Plain("users").FindOne(ctx, bsonx.D(bsonx.E("_id", id))).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, utils.NotFound("user not found")
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpsertAdmin(ctx context.Context, name, email, passwordHash string) error {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	now := time.Now().UTC()
	update := bsonx.D(
		bsonx.E("$set", bsonx.D(
			bsonx.E("name", name),
			bsonx.E("passwordHash", passwordHash),
			bsonx.E("role", models.RoleAdmin),
			bsonx.E("updatedAt", now),
		)),
		bsonx.E("$setOnInsert", bsonx.D(bsonx.E("createdAt", now))),
	)
	_, err := r.db.Plain("users").UpdateOne(ctx, bsonx.D(bsonx.E("email", email)), update, options.Update().SetUpsert(true))
	return err
}

func (r *UserRepository) SampleCustomers(ctx context.Context, limit int) ([]models.UserRef, error) {
	ctx, cancel := r.db.OpContext(ctx)
	defer cancel()
	findOptions := options.Find().SetLimit(int64(limit)).SetProjection(bsonx.D(bsonx.E("role", 1)))
	cursor, err := r.db.Plain("users").Find(ctx, bsonx.D(bsonx.E("role", models.RoleCustomer)), findOptions)
	if err != nil {
		return nil, err
	}
	var refs []models.UserRef
	if err := cursor.All(ctx, &refs); err != nil {
		return nil, err
	}
	return refs, nil
}
