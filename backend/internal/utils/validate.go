package utils

import (
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"
)

func NormalizeEmail(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" || len(trimmed) > 254 {
		return "", BadRequest("a valid email is required")
	}
	address, err := mail.ParseAddress(trimmed)
	if err != nil || address.Address != trimmed {
		return "", BadRequest("a valid email is required")
	}
	return trimmed, nil
}

func RequireText(field, value string, minLength, maxLength int) (string, error) {
	trimmed := strings.TrimSpace(value)
	length := utf8.RuneCountInString(trimmed)
	if length < minLength || length > maxLength {
		return "", BadRequest(field + " must be between " + strconv.Itoa(minLength) + " and " + strconv.Itoa(maxLength) + " characters")
	}
	return trimmed, nil
}
