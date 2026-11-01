package handlers

import (
	"log/slog"
	"net/http"

	"mongoscale/internal/config"
	"mongoscale/internal/metrics"
	"mongoscale/internal/middleware"
	"mongoscale/internal/services"
)

type Dependencies struct {
	Config       *config.Config
	Logger       *slog.Logger
	Auth         *services.AuthService
	Products     *services.ProductService
	Orders       *services.OrderService
	Reviews      *services.ReviewService
	System       *services.SystemService
	LoadTest     *services.LoadTestService
	Optimization *services.OptimizationService
	Collector    *metrics.Collector
	Limiter      *middleware.RateLimiter
}

func NewRouter(deps Dependencies) http.Handler {
	h := &Handlers{
		logger:       deps.Logger,
		auth:         deps.Auth,
		products:     deps.Products,
		orders:       deps.Orders,
		reviews:      deps.Reviews,
		system:       deps.System,
		loadTest:     deps.LoadTest,
		optimization: deps.Optimization,
	}

	mux := http.NewServeMux()
	authenticated := middleware.Authenticate(deps.Auth)
	optional := middleware.OptionalAuthenticate(deps.Auth)
	admin := middleware.RequireAdmin()
	rateLimitEnabled := func() bool { return deps.Optimization.Tuning().RateLimit }
	limitAlways := middleware.RateLimit(rateLimitEnabled, deps.Limiter, nil)
	limitSearches := middleware.RateLimit(rateLimitEnabled, deps.Limiter, func(r *http.Request) bool {
		return r.URL.Query().Get("q") != ""
	})

	route := func(pattern string, handler Handler, chain ...middleware.Middleware) {
		mux.Handle(pattern, middleware.Chain(wrap(deps.Logger, handler), chain...))
	}

	route("GET /health", h.health)
	route("GET /metrics", h.prometheus)

	route("POST /api/v1/auth/register", h.register)
	route("POST /api/v1/auth/login", h.login)
	route("GET /api/v1/auth/me", h.me, authenticated)

	route("GET /api/v1/products", h.listProducts, optional, limitSearches)
	route("GET /api/v1/products/categories", h.listCategories)
	route("GET /api/v1/products/category/{category}", h.listProductsByCategory, optional)
	route("GET /api/v1/products/{id}", h.getProduct)
	route("POST /api/v1/products", h.createProduct, authenticated, admin)
	route("PATCH /api/v1/products/{id}/stock", h.adjustStock, authenticated, admin)

	route("POST /api/v1/orders", h.createOrder, authenticated, limitAlways)
	route("GET /api/v1/orders", h.listOrders, authenticated)
	route("GET /api/v1/orders/summary", h.orderSummary, authenticated)
	route("GET /api/v1/orders/{id}", h.getOrder, authenticated)
	route("PATCH /api/v1/orders/{id}/status", h.updateOrderStatus, authenticated)

	route("POST /api/v1/reviews", h.createReview, authenticated, limitAlways)
	route("GET /api/v1/reviews/product/{productId}", h.listReviews)
	route("GET /api/v1/reviews/product/{productId}/summary", h.reviewSummary, optional, limitAlways)

	route("GET /api/v1/system/health", h.health)
	route("GET /api/v1/system/metrics", h.metrics, authenticated, admin)
	route("GET /api/v1/system/mongodb-status", h.mongoStatus, authenticated, admin)
	route("GET /api/v1/system/optimizations", h.optimizations, authenticated, admin)
	route("POST /api/v1/system/optimizations/{feature}/toggle", h.toggleOptimization, authenticated, admin)
	route("POST /api/v1/system/load-test/start", h.startLoadTest, authenticated, admin)
	route("POST /api/v1/system/load-test/stop", h.stopLoadTest, authenticated, admin)
	route("GET /api/v1/system/load-test/status", h.loadTestStatus, authenticated, admin)
	route("GET /api/v1/system/load-test/results", h.loadTestResults, authenticated, admin)
	route("POST /api/v1/system/replica-set/step-down", h.stepDown, authenticated, admin)

	route("/", h.notFound)

	return middleware.Chain(mux,
		middleware.Recover(deps.Logger),
		middleware.RequestID(),
		middleware.CORS(deps.Config.CORSOrigin),
		middleware.Metrics(deps.Collector),
		middleware.AccessLog(deps.Logger),
		middleware.Timeout(deps.Config.RequestTimeout),
	)
}
