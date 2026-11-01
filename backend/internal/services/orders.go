package services

import (
	"context"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"mongoscale/internal/models"
	"mongoscale/internal/repository"
	"mongoscale/internal/utils"
)

const (
	maxOrderLines    = 10
	maxLineQuantity  = 20
	compensationTime = 5 * time.Second
)

var allowedTransitions = map[models.OrderStatus][]models.OrderStatus{
	models.OrderPending:   {models.OrderPaid, models.OrderCancelled},
	models.OrderPaid:      {models.OrderShipped, models.OrderCancelled},
	models.OrderShipped:   {models.OrderDelivered},
	models.OrderDelivered: {},
	models.OrderCancelled: {},
}

var customerTransitions = map[models.OrderStatus][]models.OrderStatus{
	models.OrderPending: {models.OrderPaid, models.OrderCancelled},
	models.OrderPaid:    {models.OrderCancelled},
}

type OrderService struct {
	orders   *repository.OrderRepository
	products *repository.ProductRepository
	tuning   TuningSource
}

func NewOrderService(orders *repository.OrderRepository, products *repository.ProductRepository, tuning TuningSource) *OrderService {
	return &OrderService{orders: orders, products: products, tuning: tuning}
}

func (s *OrderService) Create(ctx context.Context, userID primitive.ObjectID, request models.CreateOrderRequest) (*models.Order, error) {
	if len(request.Items) == 0 || len(request.Items) > maxOrderLines {
		return nil, utils.BadRequest("an order needs between 1 and 10 distinct items")
	}
	address, err := validateAddress(request.ShippingAddress)
	if err != nil {
		return nil, err
	}

	quantities := make(map[primitive.ObjectID]int, len(request.Items))
	order := make([]primitive.ObjectID, 0, len(request.Items))
	for _, line := range request.Items {
		productID, parseErr := utils.ParseObjectID(line.ProductID)
		if parseErr != nil {
			return nil, parseErr
		}
		if line.Quantity < 1 || line.Quantity > maxLineQuantity {
			return nil, utils.BadRequest("quantity must be between 1 and 20")
		}
		if _, seen := quantities[productID]; !seen {
			order = append(order, productID)
		}
		quantities[productID] += line.Quantity
	}

	found, err := s.products.FindByIDs(ctx, order)
	if err != nil {
		return nil, err
	}
	items := make([]models.OrderItem, 0, len(order))
	total := 0.0
	for _, productID := range order {
		product, exists := found[productID]
		if !exists {
			return nil, utils.NotFound("product " + productID.Hex() + " does not exist")
		}
		quantity := quantities[productID]
		if product.Stock < quantity {
			return nil, utils.Conflict("not enough stock for " + product.Name)
		}
		items = append(items, models.OrderItem{ProductID: productID, Name: product.Name, Price: product.Price, Quantity: quantity})
		total += product.Price * float64(quantity)
	}

	token := primitive.NewObjectID().Hex()
	reserved, err := s.products.ReserveStock(ctx, token, items)
	if err != nil {
		s.compensate(token, items)
		return nil, err
	}
	if reserved != int64(len(items)) {
		s.compensate(token, items)
		return nil, utils.Conflict("stock changed while the order was processed, please retry")
	}

	tuning := s.tuning.Tuning()
	created := &models.Order{
		UserID:          userID,
		Items:           items,
		TotalAmount:     math.Round(total*100) / 100,
		Status:          models.OrderPending,
		PaymentStatus:   models.PaymentPending,
		ShippingAddress: address,
	}
	if err := s.orders.Insert(ctx, created, tuning); err != nil {
		s.restoreAll(items)
		return nil, err
	}
	return created, nil
}

func (s *OrderService) compensate(token string, items []models.OrderItem) {
	ctx, cancel := context.WithTimeout(context.Background(), compensationTime)
	defer cancel()
	_ = s.products.ReleaseReservation(ctx, token, items)
}

func (s *OrderService) restoreAll(items []models.OrderItem) {
	ctx, cancel := context.WithTimeout(context.Background(), compensationTime)
	defer cancel()
	_ = s.products.RestoreStock(ctx, items, nil)
}

func validateAddress(address models.ShippingAddress) (models.ShippingAddress, error) {
	var err error
	if address.FullName, err = utils.RequireText("shippingAddress.fullName", address.FullName, 2, 120); err != nil {
		return address, err
	}
	if address.Line1, err = utils.RequireText("shippingAddress.line1", address.Line1, 3, 200); err != nil {
		return address, err
	}
	if address.City, err = utils.RequireText("shippingAddress.city", address.City, 2, 80); err != nil {
		return address, err
	}
	if address.State, err = utils.RequireText("shippingAddress.state", address.State, 2, 80); err != nil {
		return address, err
	}
	if address.PostalCode, err = utils.RequireText("shippingAddress.postalCode", address.PostalCode, 3, 12); err != nil {
		return address, err
	}
	address.Country, err = utils.RequireText("shippingAddress.country", address.Country, 2, 80)
	return address, err
}

type ListOrdersQuery struct {
	Status string
	UserID string
	Limit  int
	Page   int
	Cursor string
}

func (s *OrderService) List(ctx context.Context, who Identity, query ListOrdersQuery) (*models.Page[models.Order], error) {
	filter := repository.OrderFilter{Limit: query.Limit, Page: query.Page, Cursor: query.Cursor}
	if query.Status != "" {
		status := models.OrderStatus(query.Status)
		if !status.Valid() {
			return nil, utils.BadRequest("status is not a valid order status")
		}
		filter.Status = status
	}
	switch {
	case who.Role != models.RoleAdmin:
		filter.UserID = &who.UserID
	case query.UserID != "":
		userID, err := utils.ParseObjectID(query.UserID)
		if err != nil {
			return nil, err
		}
		filter.UserID = &userID
	}
	return s.orders.List(ctx, filter, s.tuning.Tuning())
}

func (s *OrderService) Get(ctx context.Context, who Identity, id primitive.ObjectID) (*models.Order, error) {
	order, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if who.Role != models.RoleAdmin && order.UserID != who.UserID {
		return nil, utils.NotFound("order not found")
	}
	return order, nil
}

func (s *OrderService) Summary(ctx context.Context, who Identity) ([]models.OrderSummaryRow, error) {
	var userID *primitive.ObjectID
	if who.Role != models.RoleAdmin {
		userID = &who.UserID
	}
	return s.orders.Summary(ctx, userID, s.tuning.Tuning())
}

func (s *OrderService) UpdateStatus(ctx context.Context, who Identity, id primitive.ObjectID, next models.OrderStatus) (*models.Order, error) {
	if !next.Valid() {
		return nil, utils.BadRequest("status is not a valid order status")
	}
	order, err := s.Get(ctx, who, id)
	if err != nil {
		return nil, err
	}
	rules := allowedTransitions
	if who.Role != models.RoleAdmin {
		rules = customerTransitions
	}
	if !contains(rules[order.Status], next) {
		return nil, utils.Conflict("order cannot move from " + string(order.Status) + " to " + string(next))
	}

	payment := order.PaymentStatus
	switch next {
	case models.OrderPaid:
		payment = models.PaymentPaid
	case models.OrderCancelled:
		if order.PaymentStatus == models.PaymentPaid {
			payment = models.PaymentRefunded
		} else {
			payment = models.PaymentFailed
		}
	}
	return s.orders.Transition(ctx, order, next, payment, next == models.OrderCancelled, s.products)
}

func contains(statuses []models.OrderStatus, candidate models.OrderStatus) bool {
	for _, status := range statuses {
		if status == candidate {
			return true
		}
	}
	return false
}
