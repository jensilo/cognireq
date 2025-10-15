package pkg

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/google/uuid"
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

	registerPath := "POST /api/v1/user/register"
	mux.Handle(registerPath, HandleJSON(registerController(logger.WithGroup(registerPath), services.User)))

	loginPath := "POST /api/v1/user/login"
	mux.Handle(loginPath, HandleJSON(loginController(logger.WithGroup(loginPath), services.User)))

	logoutPath := "POST /api/v1/user/logout"
	mux.Handle(logoutPath, HandleJSON(logoutController(logger.WithGroup(logoutPath), services.User)))

	mePath := "GET /api/v1/user/me"
	meAuth := AuthMiddleware(logger.WithGroup(mePath), services.User)
	mux.Handle(mePath, meAuth(meController()))

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
	return func(w http.ResponseWriter, r *http.Request, in registerPayload) (*registerResponse, int, *WebError) {
		toCreate := types.UserToCreate{
			Email:         in.Email,
			FirstName:     in.FirstName,
			LastName:      in.LastName,
			PlainPassword: in.PlainPassword,
		}
		id, err := service.Register(r.Context(), toCreate)
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

type loginResponse struct{}

func loginController(logger *slog.Logger, service *UserService) ControllerLogic[loginPayload, *loginResponse] {
	return func(w http.ResponseWriter, r *http.Request, in loginPayload) (*loginResponse, int, *WebError) {
		_, sessionID, err := service.Login(r.Context(), in.Email, in.Password)
		if errors.Is(err, ErrInvalidLogin) {
			return nil, 0, &WebError{Code: CodeInvalidCredentials, Status: http.StatusBadRequest}
		}
		if err != nil {
			logger.Error("could not login user", "err", err)
			return nil, 0, NewInternalError()
		}

		SetSessionCookie(w, sessionID)

		return &loginResponse{}, http.StatusOK, nil
	}
}

type logoutPayload struct{}

func (p logoutPayload) Validate() *WebError {
	return nil
}

type logoutResponse struct{}

func logoutController(logger *slog.Logger, service *UserService) ControllerLogic[logoutPayload, *logoutResponse] {
	return func(w http.ResponseWriter, r *http.Request, in logoutPayload) (*logoutResponse, int, *WebError) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			return &logoutResponse{}, http.StatusOK, nil
		}

		// check if cookie value is UUID
		sessionID, err := uuid.Parse(cookie.Value)
		if err != nil {
			return nil, 0, NewMalformedRequestError()
		}

		if err := service.Logout(r.Context(), sessionID.String()); err != nil {
			logger.Error("could not logout user", "err", err)
			return nil, 0, NewInternalError()
		}

		RemoveSessionCookie(w)

		return &logoutResponse{}, http.StatusOK, nil
	}
}

type meResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	LastLogin time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func meController() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := MustUserFromCtx(r.Context())
		res := &meResponse{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			LastLogin: user.LastLogin,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}

		if err := EncodeJSON(w, http.StatusOK, res); err != nil {
			WriteJSONError(w, NewInternalError())
		}
	})
}
