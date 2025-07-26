package pkg

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func NewServer(logger *slog.Logger) (http.Handler, error) {
	mux := http.NewServeMux()
	err := addRoutes(mux, logger)
	if err != nil {
		return nil, err
	}

	return mux, nil
}

func addRoutes(mux *http.ServeMux, logger *slog.Logger) error {
	mux.Handle("GET /api/v1/ping", pingHandler())

	return nil
}

func pingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode("pong")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
