package pkg

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPing(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	w := httptest.NewRecorder()
	pingHandler().ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	pong, err := DecodeDataToJSON[string](data)
	if err != nil {
		t.Errorf("got error decoding json %v", err)
	}
	if pong != "pong" {
		t.Errorf("expected pong, got: %s", pong)
	}
	fmt.Println(pong)
}

func TestRegister(t *testing.T) {
	t.Parallel()

}
