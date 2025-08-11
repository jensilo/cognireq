package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

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

func DecodeDataToJSON[T any](d []byte) (T, error) {
	var v T
	if err := json.NewDecoder(bytes.NewReader(d)).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}
