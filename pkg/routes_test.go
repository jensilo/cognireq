package pkg

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jensilo/cognireq/pkg/db"
	"github.com/jensilo/cognireq/test"
	"github.com/stretchr/testify/require"
)

func TestPing(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	w := httptest.NewRecorder()
	pingHandler().ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode, "invalid status code")
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err, "can not read body")
	pong, err := DecodeJSONFromBytes[string](data)
	require.NoError(t, err, "can not decode json")
	require.Equal(t, "pong", pong, "ping did not pong")
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
		PlainPassword: "test1234",
	})
	require.NoError(t, err, "could not marshal json payload")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	HandleJSON(registerController(logger, NewUserService(queries))).ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode, "invalid status code")
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err, "can not read body")
	reg, err := DecodeJSONFromBytes[registerResponse](data)
	require.NoError(t, err, "can not decode json")
	require.NotEmpty(t, reg.ID, "empty registration id")
	t.Log("User ID", reg.ID)
}
