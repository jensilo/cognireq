package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jensilo/cognireq/pkg/cfg"
	"github.com/jensilo/cognireq/pkg/types"
)

type ctxKey string

const (
	authUserCtxKey ctxKey = "AuthUser"

	CodeInternalError         = "INTERNAL_ERROR"
	CodeMalformedRequestError = "MALFORMED_REQUEST_ERROR"
	CodeValidationError       = "VALIDATION_ERROR"
	CodeAuthError             = "AUTHORIZATION_ERROR"
)

// ControllerLogic represents the core business logic of a controller.
type ControllerLogic[In, Out any] func(w http.ResponseWriter, r *http.Request, in In) (Out, int, *WebError)

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

func NewAuthError() *WebError {
	return &WebError{
		Code:   CodeAuthError,
		Status: http.StatusUnauthorized,
	}
}

func NewServer(getenv cfg.GetEnvFunc, logger *slog.Logger, services *Services) (http.Handler, error) {
	mux := http.NewServeMux()
	err := addRoutes(mux, logger, services)
	if err != nil {
		return nil, err
	}

	finalMux := CORSMiddleware(getenv)(mux)

	return finalMux, nil
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

		out, status, err := logic(w, r, in)
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

// SetSessionCookie sets a secure, HTTP-only session cookie.
func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// RemoveSessionCookie sets session cookie to expired and empty.
func RemoveSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Set expiry to the past
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// CORSMiddleware is a middleware adding CORS allow headers.
func CORSMiddleware(getenv cfg.GetEnvFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			h.Set("Access-Control-Allow-Origin", getenv(cfg.CORSAllowOrigin))
			h.Set("Access-Control-Allow-Credentials", getenv(cfg.CORSAllowCredentials))
			h.Set("Access-Control-Allow-Methods", getenv(cfg.CORSAllowMethods))
			h.Set("Access-Control-Allow-Headers", getenv(cfg.CORSAllowHeaders))

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware checks the user's login status using their session cookie, adding the user to the request's context.
// If the user can not be authenticated (no session cookie, session expired) the middleware returns a 401 Unauthorized.
func AuthMiddleware(logger *slog.Logger, service *UserService) func(http.Handler) http.Handler {
	logger = logger.With("middleware", "AuthMiddleware")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := r.Cookie("session_id")
			if err != nil {
				logger.Info("no cookie found for request")
				WriteJSONError(w, NewAuthError())
				return
			}

			err = sess.Valid()
			id := sess.Value
			if err != nil || id == "" {
				logger.Info("invalid session cookie", "id", id)
				RemoveSessionCookie(w)
				WriteJSONError(w, NewAuthError())
				return
			}

			// check if cookie value is UUID
			sessionID, err := uuid.Parse(id)
			if err != nil {
				logger.Info("invalid uuid in session cookie", "id", id)
				RemoveSessionCookie(w)
				WriteJSONError(w, NewAuthError())
				return
			}

			ctx := r.Context()
			user, err := service.LoggedInUser(ctx, sessionID)
			if err != nil {
				logger.Info("authentication failed", "id", id, "err", err)
				RemoveSessionCookie(w)
				WriteJSONError(w, NewAuthError())
				return
			}

			newCtx := context.WithValue(ctx, authUserCtxKey, user)
			r = r.WithContext(newCtx)

			next.ServeHTTP(w, r)
		})
	}
}

func MustUserFromCtx(ctx context.Context) *types.User {
	user, ok := ctx.Value(authUserCtxKey).(*types.User)
	if !ok {
		panic("must use auth middleware before trying to retrieve user from context")
	}

	return user
}
