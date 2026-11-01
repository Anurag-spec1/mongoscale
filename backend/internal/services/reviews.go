package services

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"mongoscale/internal/cache"
	"mongoscale/internal/models"
	"mongoscale/internal/repository"
	"mongoscale/internal/utils"
)

type ReviewService struct {
	reviews  *repository.ReviewRepository
	products *repository.ProductRepository
	cache    *cache.Cache
	tuning   TuningSource
	ttl      time.Duration
}

func NewReviewService(
	reviews *repository.ReviewRepository,
	products *repository.ProductRepository,
	cacheClient *cache.Cache,
	tuning TuningSource,
	ttl time.Duration,
) *ReviewService {
	return &ReviewService{reviews: reviews, products: products, cache: cacheClient, tuning: tuning, ttl: ttl}
}

func (s *ReviewService) Create(ctx context.Context, userID primitive.ObjectID, request models.CreateReviewRequest) (*models.Review, error) {
	productID, err := utils.ParseObjectID(request.ProductID)
	if err != nil {
		return nil, err
	}
	if request.Rating < 1 || request.Rating > 5 {
		return nil, utils.BadRequest("rating must be between 1 and 5")
	}
	comment, err := utils.RequireText("comment", request.Comment, 0, 1000)
	if err != nil {
		return nil, err
	}
	exists, err := s.products.Exists(ctx, productID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, utils.NotFound("product not found")
	}
	review := &models.Review{ProductID: productID, UserID: userID, Rating: request.Rating, Comment: comment}
	if err := s.reviews.Create(ctx, review); err != nil {
		return nil, err
	}
	if err := s.products.ApplyRating(ctx, productID, request.Rating); err != nil {
		return nil, err
	}
	s.cache.Delete(ctx, summaryKey(productID))
	return review, nil
}

func (s *ReviewService) ListByProduct(
	ctx context.Context,
	productID primitive.ObjectID,
	limit, page int,
	cursor string,
) (*models.Page[models.Review], error) {
	return s.reviews.ListByProduct(ctx, productID, limit, page, cursor, s.tuning.Tuning())
}

func (s *ReviewService) Summary(ctx context.Context, productID primitive.ObjectID) (*models.ReviewSummary, error) {
	tuning := s.tuning.Tuning()
	if !tuning.Cache {
		return s.reviews.Summary(ctx, productID, tuning)
	}
	key := summaryKey(productID)
	var cached models.ReviewSummary
	if s.cache.GetJSON(ctx, key, &cached) {
		return &cached, nil
	}
	summary, err := s.reviews.Summary(ctx, productID, tuning)
	if err != nil {
		return nil, err
	}
	s.cache.SetJSON(ctx, key, summary, s.ttl)
	return summary, nil
}

func summaryKey(productID primitive.ObjectID) string {
	return "reviews:summary:" + productID.Hex()
}
