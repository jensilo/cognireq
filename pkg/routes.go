package pkg

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jensilo/harmony/pkg/types"
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
type registerResponse struct {
	ID string `json:"id"`
}

func registerHandler(logger *slog.Logger, service *UserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := DecodeJSON[registerPayload](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = EncodeJSON(w, http.StatusCreated, registerResponse{ID: id.String()})
		if err != nil {
			logger.Error("unexpected json encoding error", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

func loginHandler(logger *slog.Logger, service *UserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := DecodeJSON[loginPayload](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		usrID, sessionID, err := service.Login(r.Context(), p.Email, p.Password)
		if err != nil && errors.Is(err, ErrInvalidLogin) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = EncodeJSON(w, http.StatusOK, loginResponse{UserID: usrID, SessionID: sessionID})
		if err != nil {
			logger.Error("unexpected json encoding error", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}
