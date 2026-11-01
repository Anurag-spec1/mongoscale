package handlers

import (
	"log/slog"

	"mongoscale/internal/services"
)

type Handlers struct {
	logger       *slog.Logger
	auth         *services.AuthService
	products     *services.ProductService
	orders       *services.OrderService
	reviews      *services.ReviewService
	system       *services.SystemService
	loadTest     *services.LoadTestService
	optimization *services.OptimizationService
}
