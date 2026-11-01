package utils

import (
	"net/http"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ParseObjectID(value string) (primitive.ObjectID, error) {
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		return primitive.NilObjectID, BadRequest("identifier is not a valid ObjectId")
	}
	return id, nil
}

func IntQuery(r *http.Request, key string, fallback, minimum, maximum int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, BadRequest(key + " must be an integer between " + strconv.Itoa(minimum) + " and " + strconv.Itoa(maximum))
	}
	return value, nil
}

func FloatQuery(r *http.Request, key string) (*float64, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		return nil, BadRequest(key + " must be a non-negative number")
	}
	return &value, nil
}

func TrimmedQuery(r *http.Request, key string, maxLength int) (string, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if len(value) > maxLength {
		return "", BadRequest(key + " is too long")
	}
	return value, nil
}
