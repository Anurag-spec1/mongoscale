package utils

import (
	"encoding/json"
	"net/http"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *errorBody `json:"error,omitempty"`
}

func writeEnvelope(w http.ResponseWriter, status int, body envelope) {
	payload, err := json.Marshal(body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"encoding_error","message":"response could not be encoded"}}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	writeEnvelope(w, status, envelope{Success: true, Data: data})
}

func WriteError(w http.ResponseWriter, appErr *AppError) {
	writeEnvelope(w, appErr.Status, envelope{
		Success: false,
		Error:   &errorBody{Code: appErr.Code, Message: appErr.Message},
	})
}
