package pkg

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jensilo/cognireq/pkg/db"
	"github.com/jensilo/cognireq/pkg/types"
	"github.com/jensilo/cognireq/test"
	"github.com/stretchr/testify/require"
)

type testServer struct {
	router *http.ServeMux
	server *httptest.Server
	user   *UserService
	iga    *IGAService
	db     *pgxpool.Pool
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	logger := test.InitLogger(t)
	dbConn := test.InitDB(t)
	queries := db.New(dbConn)

	userService := NewUserService(queries)
	igaService := NewIGAService(queries)
	services := NewServices(userService, igaService)

	router, err := NewServer(logger, services)
	require.NoError(t, err)

	s := httptest.NewServer(router)
	t.Cleanup(func() {
		s.Close()
	})

	return &testServer{
		router: router,
		server: s,
		user:   userService,
		iga:    igaService,
		db:     dbConn,
	}
}

func (ts *testServer) get(t *testing.T, urlPath string) *http.Response {
	t.Helper()

	res, err := ts.server.Client().Get(ts.server.URL + urlPath)
	require.NoError(t, err)

	return res
}

func (ts *testServer) createTestUser(t *testing.T, email, password string) *types.User {
	t.Helper()

	user, err := ts.user.Register(t.Context(), types.UserToCreate{
		Email:         email,
		FirstName:     "Test",
		LastName:      "User",
		PlainPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, user)

	activated := ts.user.ActivateUser(t.Context(), user.ID)
	require.NoError(t, activated)

	return user
}

func (ts *testServer) newAuthenticatedRequest(t *testing.T, method, urlPath, body string) *http.Request {
	t.Helper()

	user := ts.createTestUser(t, "test@test.com", "password123")

	_, sessionID, err := ts.user.Login(t.Context(), user.Email, "password123")
	require.NoError(t, err)

	req, err := http.NewRequest(method, ts.server.URL+urlPath, bytes.NewReader([]byte(body)))
	require.NoError(t, err)

	req.AddCookie(&http.Cookie{
		Name:    "session_id",
		Value:   sessionID.String(),
		Expires: time.Now().Add(time.Hour),
	})

	return req
}

func TestPing(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	res := ts.get(t, "/api/v1/ping")
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode, "invalid status code")
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err, "can not read body")
	pong, err := DecodeJSONFromBytes[string](data)
	require.NoError(t, err, "can not decode json")
	require.Equal(t, "pong", pong, "ping did not pong")
	t.Log("ping ?", pong)
}

func TestRegister(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	payload, err := json.Marshal(registerPayload{
		Email:         "foo@bar.de",
		FirstName:     "Foo",
		LastName:      "Bar",
		PlainPassword: "test1234",
	})
	require.NoError(t, err, "could not marshal json payload")

	req, err := http.NewRequest(http.MethodPost, ts.server.URL+"/api/v1/user/register", bytes.NewReader(payload))
	require.NoError(t, err)

	res, err := ts.server.Client().Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode, "invalid status code")
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err, "can not read body")
	reg, err := DecodeJSONFromBytes[registerResponse](data)
	require.NoError(t, err, "can not decode json")
	require.NotEmpty(t, reg.ID, "empty registration id")
	t.Log("User ID", reg.ID)
}

func TestCreateIGASession(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	payload := `{"project_name": "Test Project", "context_data": {"stakeholder_summary": "summary", "project_summary": "summary", "interview_goal": "goal", "interview_advice": "advice"}}`
	req := ts.newAuthenticatedRequest(t, http.MethodPost, "/api/v1/iga", payload)
	req.Header.Set("Content-Type", "application/json")

	res, err := ts.server.Client().Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode)

	var igaResp createIGAResponse
	err = json.NewDecoder(res.Body).Decode(&igaResp)
	require.NoError(t, err)

	require.NotEqual(t, uuid.Nil, igaResp.ID)
	require.Equal(t, "Test Project", igaResp.ProjectName)
}
