package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"mongoscale/internal/cache"
	"mongoscale/internal/models"
	"mongoscale/internal/repository"
	"mongoscale/internal/utils"
)

const (
	productCacheNamespace = "products"
	maxStockDelta         = 1000000
)

type TuningSource interface {
	Tuning() models.Tuning
}

type ProductService struct {
	products *repository.ProductRepository
	cache    *cache.Cache
	tuning   TuningSource
	ttl      time.Duration
}

func NewProductService(products *repository.ProductRepository, cacheClient *cache.Cache, tuning TuningSource, ttl time.Duration) *ProductService {
	return &ProductService{products: products, cache: cacheClient, tuning: tuning, ttl: ttl}
}

func (s *ProductService) List(ctx context.Context, filter repository.ProductFilter) (*models.Page[models.Product], error) {
	tuning := s.tuning.Tuning()
	if !tuning.Cache {
		return s.products.List(ctx, filter, tuning)
	}
	key := s.listKey(ctx, filter)
	var cached models.Page[models.Product]
	if s.cache.GetJSON(ctx, key, &cached) {
		return &cached, nil
	}
	page, err := s.products.List(ctx, filter, tuning)
	if err != nil {
		return nil, err
	}
	s.cache.SetJSON(ctx, key, page, s.ttl)
	return page, nil
}

func (s *ProductService) listKey(ctx context.Context, filter repository.ProductFilter) string {
	lowPrice, highPrice := -1.0, -1.0
	if filter.MinPrice != nil {
		lowPrice = *filter.MinPrice
	}
	if filter.MaxPrice != nil {
		highPrice = *filter.MaxPrice
	}
	raw := fmt.Sprintf("%s|%s|%f|%f|%s|%d|%d|%s",
		filter.Category, strings.ToLower(filter.Query), lowPrice, highPrice, filter.Sort, filter.Limit, filter.Page, filter.Cursor)
	digest := sha256.Sum256([]byte(raw))
	return "products:list:" + s.cache.Version(ctx, productCacheNamespace) + ":" + hex.EncodeToString(digest[:12])
}

func (s *ProductService) Get(ctx context.Context, id primitive.ObjectID) (*models.Product, error) {
	tuning := s.tuning.Tuning()
	if !tuning.Cache {
		return s.products.GetByID(ctx, id, tuning)
	}
	key := "products:item:" + s.cache.Version(ctx, productCacheNamespace) + ":" + id.Hex()
	var cached models.Product
	if s.cache.GetJSON(ctx, key, &cached) {
		return &cached, nil
	}
	product, err := s.products.GetByID(ctx, id, tuning)
	if err != nil {
		return nil, err
	}
	s.cache.SetJSON(ctx, key, product, s.ttl)
	return product, nil
}

func (s *ProductService) Categories(ctx context.Context) ([]string, error) {
	tuning := s.tuning.Tuning()
	if !tuning.Cache {
		return s.products.Categories(ctx, tuning)
	}
	key := "products:categories:" + s.cache.Version(ctx, productCacheNamespace)
	var cached []string
	if s.cache.GetJSON(ctx, key, &cached) {
		return cached, nil
	}
	categories, err := s.products.Categories(ctx, tuning)
	if err != nil {
		return nil, err
	}
	s.cache.SetJSON(ctx, key, categories, s.ttl)
	return categories, nil
}

func (s *ProductService) Create(ctx context.Context, request models.CreateProductRequest) (*models.Product, error) {
	name, err := utils.RequireText("name", request.Name, 2, 140)
	if err != nil {
		return nil, err
	}
	category, err := utils.RequireText("category", request.Category, 2, 60)
	if err != nil {
		return nil, err
	}
	description, err := utils.RequireText("description", request.Description, 0, 4000)
	if err != nil {
		return nil, err
	}
	if request.Price <= 0 || request.Price > 10000000 {
		return nil, utils.BadRequest("price must be greater than zero")
	}
	if request.Stock < 0 || request.Stock > maxStockDelta {
		return nil, utils.BadRequest("stock must be between 0 and 1000000")
	}
	product := &models.Product{
		Name:        name,
		Description: description,
		Category:    category,
		Price:       request.Price,
		Stock:       request.Stock,
		ImageURL:    strings.TrimSpace(request.ImageURL),
	}
	if err := s.products.Create(ctx, product); err != nil {
		return nil, err
	}
	s.cache.BumpVersion(ctx, productCacheNamespace)
	return product, nil
}

func (s *ProductService) AdjustStock(ctx context.Context, id primitive.ObjectID, request models.StockAdjustmentRequest) (*models.Product, error) {
	if request.Delta == 0 || request.Delta > maxStockDelta || request.Delta < -maxStockDelta {
		return nil, utils.BadRequest("delta must be a non-zero integer within one million")
	}
	product, err := s.products.AdjustStock(ctx, id, request.Delta)
	if err != nil {
		return nil, err
	}
	s.cache.BumpVersion(ctx, productCacheNamespace)
	return product, nil
}
