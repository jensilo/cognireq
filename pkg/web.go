package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const (
	CodeInternalError         = "INTERNAL_ERROR"
	CodeMalformedRequestError = "MALFORMED_REQUEST_ERROR"
	CodeValidationError       = "VALIDATION_ERROR"
)

// ControllerLogic represents the core business logic of a controller.
type ControllerLogic[In, Out any] func(ctx context.Context, in In) (Out, int, *WebError)

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

func NewMalformedRequestError() *WebError {
	return &WebError{
		Code:   CodeMalformedRequestError,
		Status: http.StatusBadRequest,
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

// Validatable is a payload that can be validated.
type Validatable interface {
	Validate() *WebError
}

// HandleJSON is a generic handler that manages JSON decoding, validation, and error handling.
// It abstracts away HTTP boilerplate, allowing controllers to focus on business logic.
// HandleJSON is minimal and should not fit every use case, it's just a convenience for the standard http handler.
func HandleJSON[In Validatable, Out any](logic ControllerLogic[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in In
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteJSONError(w, NewMalformedRequestError())
			return
		}

		if err := in.Validate(); err != nil {
			WriteJSONError(w, err)
			return
		}

		out, status, err := logic(r.Context(), in)
		if err != nil {
			WriteJSONError(w, err)
			return
		}

		if err := EncodeJSON(w, status, out); err != nil {
			WriteJSONError(w, NewInternalError())
		}
	}
}

// EncodeJSON writes data as a JSON response.
func EncodeJSON[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	return nil
}

// DecodeJSON reads JSON data from a request.
func DecodeJSON[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}

	return v, nil
}

// DecodeJSONFromBytes reads JSON from a byte slice.
func DecodeJSONFromBytes[T any](d []byte) (T, error) {
	var v T
	if err := json.NewDecoder(bytes.NewReader(d)).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}

	return v, nil
}

// WriteJSONError writes an AppError to the HTTP response.
func WriteJSONError(w http.ResponseWriter, err *WebError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(err.Status)
	_ = json.NewEncoder(w).Encode(err)
}
