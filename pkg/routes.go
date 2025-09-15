package pkg

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"

	"github.com/jensilo/harmony/pkg/types"
)

const (
	CodeEmailInvalid       = "EMAIL_INVALID"
	CodeFirstNameMissing   = "FIRST_NAME_MISSING"
	CodeLastNameMissing    = "LAST_NAME_MISSING"
	CodePasswordTooShort   = "PASSWORD_TOO_SHORT"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
)

func addRoutes(mux *http.ServeMux, logger *slog.Logger, services *Services) error {
	mux.Handle("GET /api/v1/ping", pingHandler())

	mux.Handle("POST /api/v1/user/register", registerHandler(logger.WithGroup("/api/v1/user/register"), services.User))
	mux.Handle("POST /api/v1/user/login", loginHandler(logger.WithGroup("/api/v1/user/login"), services.User))

	return nil
}

func pingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode("pong")
		if err != nil {
			WriteJSONError(w, NewInternalError())
			return
		}
	})
}

type registerPayload struct {
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	PlainPassword string `json:"plain_password"`
}

func (p *registerPayload) validate() *WebError {
	var fieldErrors []FieldError

	if _, err := mail.ParseAddress(p.Email); err != nil {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeEmailInvalid, Field: "email"})
	}
	if p.FirstName == "" {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeFirstNameMissing, Field: "first_name"})
	}
	if p.LastName == "" {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeLastNameMissing, Field: "last_name"})
	}
	if len(p.PlainPassword) < 8 {
		fieldErrors = append(fieldErrors, FieldError{Code: CodePasswordTooShort, Field: "plain_password"})
	}

	if len(fieldErrors) > 0 {
		return NewValidationError(fieldErrors)
	}

	return nil
}

type registerResponse struct {
	ID string `json:"id"`
}

// todo user already exists different error
func registerHandler(logger *slog.Logger, service *UserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := DecodeJSON[registerPayload](r)
		if err != nil {
			WriteJSONError(w, NewInternalError())
			return
		}
		if err := p.validate(); err != nil {
			WriteJSONError(w, err)
			return
		}

		toCreate := types.UserToCreate{
			Email:         p.Email,
			FirstName:     p.FirstName,
			LastName:      p.LastName,
			PlainPassword: p.PlainPassword,
		}
		id, err := service.Register(r.Context(), toCreate)
		if err != nil {
			logger.Error("could not register user", "err", err)
			WriteJSONError(w, NewInternalError())
			return
		}

		err = EncodeJSON(w, http.StatusCreated, registerResponse{ID: id})
		if err != nil {
			logger.Error("unexpected json encoding error", "err", err)
			WriteJSONError(w, NewInternalError())
			return
		}
	})
}

type loginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

// todo rework login handler
func loginHandler(logger *slog.Logger, service *UserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := DecodeJSON[loginPayload](r)
		if err != nil {
			WriteJSONError(w, NewInternalError())
			return
		}

		usrID, sessionID, err := service.Login(r.Context(), p.Email, p.Password)
		if err != nil && errors.Is(err, ErrInvalidLogin) {
			WriteJSONError(w, &WebError{Code: CodeInvalidCredentials, Status: http.StatusBadRequest})
			return
		}
		if err != nil {
			WriteJSONError(w, NewInternalError())
			return
		}

		err = EncodeJSON(w, http.StatusOK, loginResponse{UserID: usrID, SessionID: sessionID})
		if err != nil {
			logger.Error("unexpected json encoding error", "err", err)
			WriteJSONError(w, NewInternalError())
			return
		}
	})
}
