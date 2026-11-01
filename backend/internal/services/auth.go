package services

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"mongoscale/internal/models"
	"mongoscale/internal/repository"
	"mongoscale/internal/utils"
)

const (
	minPasswordLength = 8
	maxPasswordBytes  = 72
)

type Claims struct {
	Role models.Role `json:"role"`
	jwt.RegisteredClaims
}

type Identity struct {
	UserID primitive.ObjectID
	Role   models.Role
}

type AuthService struct {
	users      *repository.UserRepository
	secret     []byte
	tokenTTL   time.Duration
	bcryptCost int
}

func NewAuthService(users *repository.UserRepository, secret string, tokenTTL time.Duration, bcryptCost int) *AuthService {
	return &AuthService{users: users, secret: []byte(secret), tokenTTL: tokenTTL, bcryptCost: bcryptCost}
}

func (s *AuthService) Register(ctx context.Context, request models.RegisterRequest) (*models.AuthResponse, error) {
	name, err := utils.RequireText("name", request.Name, 2, 80)
	if err != nil {
		return nil, err
	}
	email, err := utils.NormalizeEmail(request.Email)
	if err != nil {
		return nil, err
	}
	if len(request.Password) < minPasswordLength || len(request.Password) > maxPasswordBytes {
		return nil, utils.BadRequest("password must be between 8 and 72 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), s.bcryptCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{Name: name, Email: email, PasswordHash: string(hash), Role: models.RoleCustomer}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	token, err := s.IssueToken(user.ID.Hex(), user.Role)
	if err != nil {
		return nil, err
	}
	return &models.AuthResponse{Token: token, User: *user}, nil
}

func (s *AuthService) Login(ctx context.Context, request models.LoginRequest) (*models.AuthResponse, error) {
	email, err := utils.NormalizeEmail(request.Email)
	if err != nil {
		return nil, utils.Unauthorized("invalid email or password")
	}
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, utils.Unauthorized("invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		return nil, utils.Unauthorized("invalid email or password")
	}
	token, err := s.IssueToken(user.ID.Hex(), user.Role)
	if err != nil {
		return nil, err
	}
	return &models.AuthResponse{Token: token, User: *user}, nil
}

func (s *AuthService) Me(ctx context.Context, userID primitive.ObjectID) (*models.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *AuthService) IssueToken(userID string, role models.Role) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
			Issuer:    "mongoscale",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *AuthService) Parse(token string) (Identity, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("mongoscale"))
	if err != nil || !parsed.Valid {
		return Identity{}, utils.Unauthorized("token is invalid or expired")
	}
	userID, err := primitive.ObjectIDFromHex(claims.Subject)
	if err != nil {
		return Identity{}, utils.Unauthorized("token subject is invalid")
	}
	return Identity{UserID: userID, Role: claims.Role}, nil
}

func (s *AuthService) EnsureAdmin(ctx context.Context, email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return err
	}
	if err := s.users.UpsertAdmin(ctx, "MongoScale Admin", email, string(hash)); err != nil {
		return errors.Join(errors.New("ensure admin account"), err)
	}
	return nil
}
