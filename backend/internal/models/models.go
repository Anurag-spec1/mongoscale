package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role string

const (
	RoleCustomer Role = "customer"
	RoleAdmin    Role = "admin"
)

type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderPaid      OrderStatus = "paid"
	OrderShipped   OrderStatus = "shipped"
	OrderDelivered OrderStatus = "delivered"
	OrderCancelled OrderStatus = "cancelled"
)

type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "pending"
	PaymentPaid     PaymentStatus = "paid"
	PaymentRefunded PaymentStatus = "refunded"
	PaymentFailed   PaymentStatus = "failed"
)

func (s OrderStatus) Valid() bool {
	switch s {
	case OrderPending, OrderPaid, OrderShipped, OrderDelivered, OrderCancelled:
		return true
	}
	return false
}

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Role         Role               `bson:"role" json:"role"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type Product struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name            string             `bson:"name" json:"name"`
	Description     string             `bson:"description,omitempty" json:"description,omitempty"`
	Category        string             `bson:"category" json:"category"`
	Price           float64            `bson:"price" json:"price"`
	Stock           int                `bson:"stock" json:"stock"`
	Rating          float64            `bson:"rating" json:"rating"`
	RatingCount     int                `bson:"ratingCount" json:"ratingCount"`
	ImageURL        string             `bson:"imageUrl" json:"imageUrl"`
	LastReservation string             `bson:"lastReservation,omitempty" json:"-"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type ProductRef struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Category string             `bson:"category" json:"category"`
	Price    float64            `bson:"price" json:"price"`
}

type UserRef struct {
	ID   primitive.ObjectID `bson:"_id"`
	Role Role               `bson:"role"`
}

type OrderItem struct {
	ProductID primitive.ObjectID `bson:"productId" json:"productId"`
	Name      string             `bson:"name" json:"name"`
	Price     float64            `bson:"price" json:"price"`
	Quantity  int                `bson:"quantity" json:"quantity"`
}

type ShippingAddress struct {
	FullName   string `bson:"fullName" json:"fullName"`
	Line1      string `bson:"line1" json:"line1"`
	City       string `bson:"city" json:"city"`
	State      string `bson:"state" json:"state"`
	PostalCode string `bson:"postalCode" json:"postalCode"`
	Country    string `bson:"country" json:"country"`
}

type Order struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          primitive.ObjectID `bson:"userId" json:"userId"`
	Items           []OrderItem        `bson:"items" json:"items"`
	TotalAmount     float64            `bson:"totalAmount" json:"totalAmount"`
	Status          OrderStatus        `bson:"status" json:"status"`
	PaymentStatus   PaymentStatus      `bson:"paymentStatus" json:"paymentStatus"`
	ShippingAddress ShippingAddress    `bson:"shippingAddress" json:"shippingAddress"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type OrderSummaryRow struct {
	Status      OrderStatus `bson:"_id" json:"status"`
	Count       int64       `bson:"count" json:"count"`
	TotalAmount float64     `bson:"totalAmount" json:"totalAmount"`
}

type Review struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID primitive.ObjectID `bson:"productId" json:"productId"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Rating    int                `bson:"rating" json:"rating"`
	Comment   string             `bson:"comment" json:"comment"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type ReviewSummary struct {
	ProductID     string        `json:"productId"`
	AverageRating float64       `json:"averageRating"`
	TotalReviews  int64         `json:"totalReviews"`
	Distribution  map[int]int64 `json:"distribution"`
}

type Page[T any] struct {
	Items      []T    `json:"items"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit"`
	Total      *int64 `json:"total,omitempty"`
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor,omitempty"`
	Pagination string `json:"pagination"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	ImageURL    string  `json:"imageUrl"`
}

type StockAdjustmentRequest struct {
	Delta int `json:"delta"`
}

type OrderItemRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type CreateOrderRequest struct {
	Items           []OrderItemRequest `json:"items"`
	ShippingAddress ShippingAddress    `json:"shippingAddress"`
}

type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status"`
}

type CreateReviewRequest struct {
	ProductID string `json:"productId"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
}

type LoadTestRequest struct {
	Users           int `json:"users"`
	RampUpSeconds   int `json:"rampUpSeconds"`
	HoldSeconds     int `json:"holdSeconds"`
	RampDownSeconds int `json:"rampDownSeconds"`
	ThinkTimeMinMs  int `json:"thinkTimeMinMs"`
	ThinkTimeMaxMs  int `json:"thinkTimeMaxMs"`
}

type LoadTestResult struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Mode            Mode               `bson:"mode" json:"mode"`
	Topology        string             `bson:"topology" json:"topology"`
	TargetUsers     int                `bson:"targetUsers" json:"targetUsers"`
	PeakUsers       int                `bson:"peakUsers" json:"peakUsers"`
	DurationSeconds float64            `bson:"durationSeconds" json:"durationSeconds"`
	TotalRequests   int64              `bson:"totalRequests" json:"totalRequests"`
	FailedRequests  int64              `bson:"failedRequests" json:"failedRequests"`
	Throttled       int64              `bson:"throttled" json:"throttled"`
	ErrorRate       float64            `bson:"errorRate" json:"errorRate"`
	RPS             float64            `bson:"rps" json:"rps"`
	P50             float64            `bson:"p50" json:"p50"`
	P95             float64            `bson:"p95" json:"p95"`
	P99             float64            `bson:"p99" json:"p99"`
	StartedAt       time.Time          `bson:"startedAt" json:"startedAt"`
	FinishedAt      time.Time          `bson:"finishedAt" json:"finishedAt"`
}

type LoadTestStatus struct {
	Running         bool      `json:"running"`
	Phase           string    `json:"phase"`
	Mode            Mode      `json:"mode"`
	TargetUsers     int       `json:"targetUsers"`
	ActiveUsers     int       `json:"activeUsers"`
	PeakUsers       int       `json:"peakUsers"`
	Requests        int64     `json:"requests"`
	Failures        int64     `json:"failures"`
	Throttled       int64     `json:"throttled"`
	ElapsedSeconds  float64   `json:"elapsedSeconds"`
	TotalSeconds    float64   `json:"totalSeconds"`
	ProgressPercent float64   `json:"progressPercent"`
	StartedAt       time.Time `json:"startedAt"`
	Message         string    `json:"message,omitempty"`
}

type ModeChange struct {
	Mode       Mode      `json:"mode"`
	Tuning     Tuning    `json:"tuning"`
	Topology   Topology  `json:"topology"`
	Warnings   []string  `json:"warnings"`
	AppliedAt  time.Time `json:"appliedAt"`
	DurationMs int64     `json:"durationMs"`
}
