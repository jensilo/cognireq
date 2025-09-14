package pkg

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jensilo/harmony/pkg/db"
	"github.com/jensilo/harmony/test"
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
	log.Println("ping ?", pong)
}

func TestRegister(t *testing.T) {
	t.Parallel()
	logger := test.InitLogger(t)
	dbConn := test.InitDB(t)
	queries := db.New(dbConn)
	payload, err := json.Marshal(registerPayload{
		Email:         "foo@bar.de",
		FirstName:     "Foo",
		LastName:      "Bar",
		PlainPassword: "test",
	})
	if err != nil {
		t.Fatalf("could not marshal json payload %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	registerHandler(logger, NewUserService(queries)).ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("user not registered, status code: %d", res.StatusCode)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	reg, err := DecodeDataToJSON[registerResponse](data)
	if err != nil {
		t.Errorf("got error decoding json %v", err)
	}
	if reg.ID == "" {
		t.Errorf("empty registration id")
	}
	t.Log("User ID", reg.ID)
}
