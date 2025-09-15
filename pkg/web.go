package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const (
	CodeInternalError   = "INTERNAL_ERROR"
	CodeValidationError = "VALIDATION_ERROR"
)

type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

type WebError struct {
	Code   string       `json:"code"`
	Fields []FieldError `json:"fields,omitempty"`
	Status int          `json:"-"`
}

func (e *WebError) Error() string {
	return fmt.Sprintf("%s: %+v", e.Code, e.Fields)
}

func NewInternalError() *WebError {
	return &WebError{
		Code:   CodeInternalError,
		Status: http.StatusInternalServerError,
	}
}

func NewValidationError(fields []FieldError) *WebError {
	return &WebError{
		Code:   CodeValidationError,
		Status: http.StatusBadRequest,
		Fields: fields,
	}
}

func NewServer(logger *slog.Logger, services *Services) (http.Handler, error) {
	mux := http.NewServeMux()
	err := addRoutes(mux, logger, services)
	if err != nil {
		return nil, err
	}

	return mux, nil
}

func EncodeJSON[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func DecodeJSON[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

func DecodeJSONFromBytes[T any](d []byte) (T, error) {
	var v T
	if err := json.NewDecoder(bytes.NewReader(d)).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

func WriteJSONError(w http.ResponseWriter, err *WebError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(err.Status)
	_ = json.NewEncoder(w).Encode(err)
}
