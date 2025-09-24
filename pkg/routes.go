package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"

	"github.com/jensilo/harmony/pkg/types"
)

const (
	CodeEmailInvalid       = "EMAIL_INVALID"
	CodeValueMissing       = "FIELD_MISSING"
	CodePasswordTooShort   = "PASSWORD_TOO_SHORT"
	CodeEmailExists        = "EMAIL_EXISTS"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
)

func addRoutes(mux *http.ServeMux, logger *slog.Logger, services *Services) error {
	mux.Handle("GET /api/v1/ping", pingHandler())

	mux.Handle("POST /api/v1/user/register", HandleJSON(registerController(logger.WithGroup("POST /api/v1/user/register"), services.User)))
	mux.Handle("POST /api/v1/user/login", HandleJSON(loginController(logger.WithGroup("POST /api/v1/user/login"), services.User)))

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

func (p registerPayload) Validate() *WebError {
	var fieldErrors []FieldError

	if _, err := mail.ParseAddress(p.Email); err != nil {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeEmailInvalid, Field: "email"})
	}
	if p.FirstName == "" {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeValueMissing, Field: "first_name"})
	}
	if p.LastName == "" {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeValueMissing, Field: "last_name"})
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

func registerController(logger *slog.Logger, service *UserService) ControllerLogic[registerPayload, *registerResponse] {
	return func(ctx context.Context, in registerPayload) (*registerResponse, int, *WebError) {
		toCreate := types.UserToCreate{
			Email:         in.Email,
			FirstName:     in.FirstName,
			LastName:      in.LastName,
			PlainPassword: in.PlainPassword,
		}
		id, err := service.Register(ctx, toCreate)
		if errors.Is(err, ErrEmailExists) {
			return nil, 0, &WebError{Code: CodeEmailExists, Status: http.StatusConflict}
		}
		if err != nil {
			logger.Error("could not register user", "err", err)
			return nil, 0, NewInternalError()
		}

		return &registerResponse{ID: id}, http.StatusCreated, nil
	}
}

type loginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (p loginPayload) Validate() *WebError {
	var fieldErrors []FieldError

	if p.Email == "" {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeValueMissing, Field: "email"})
	}
	if p.Password == "" {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeValueMissing, Field: "password"})
	}

	if len(fieldErrors) > 0 {
		return NewValidationError(fieldErrors)
	}

	return nil
}

type loginResponse struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

func loginController(logger *slog.Logger, service *UserService) ControllerLogic[loginPayload, *loginResponse] {
	return func(ctx context.Context, in loginPayload) (*loginResponse, int, *WebError) {
		usrID, sessionID, err := service.Login(ctx, in.Email, in.Password)
		if errors.Is(err, ErrInvalidLogin) {
			return nil, 0, &WebError{Code: CodeInvalidCredentials, Status: http.StatusBadRequest}
		}
		if err != nil {
			logger.Error("could not login user", "err", err)
			return nil, 0, NewInternalError()
		}

		return &loginResponse{UserID: usrID, SessionID: sessionID}, http.StatusOK, nil
	}
}
