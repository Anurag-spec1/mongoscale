package utils

import (
	"encoding/base64"
	"encoding/json"
)

type Cursor struct {
	Number float64 `json:"n,omitempty"`
	Millis int64   `json:"m,omitempty"`
	ID     string  `json:"id"`
}

func EncodeCursor(cursor Cursor) string {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func DecodeCursor(token string) (Cursor, error) {
	var cursor Cursor
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return cursor, BadRequest("cursor is not valid")
	}
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.ID == "" {
		return cursor, BadRequest("cursor is not valid")
	}
	return cursor, nil
}
