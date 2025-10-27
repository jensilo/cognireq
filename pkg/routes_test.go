package pkg

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jensilo/cognireq/pkg/db"
	"github.com/jensilo/cognireq/pkg/types"
	"github.com/jensilo/cognireq/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testServer struct {
	router *http.ServeMux
	server *httptest.Server
	user   *UserService
	iga    *IGAService
	db     *pgxpool.Pool

	once       sync.Once
	cookie     *http.Cookie
	authedUser *types.User
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	logger := test.InitLogger()
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

func (ts *testServer) newAuthenticatedRequest(t *testing.T, method, urlPath string, body []byte) *http.Request {
	t.Helper()

	ts.doAuth(t)

	req, err := http.NewRequest(method, ts.server.URL+urlPath, bytes.NewReader(body))
	require.NoError(t, err)

	req.AddCookie(ts.cookie)

	return req
}

func (ts *testServer) doAuth(t *testing.T) {
	t.Helper()

	ts.once.Do(func() {
		user := ts.createTestUser(t, "test@test.com", "password123")

		_, sessionID, err := ts.user.Login(t.Context(), user.Email, "password123")
		require.NoError(t, err)

		ts.cookie = &http.Cookie{
			Name:    "session_id",
			Value:   sessionID.String(),
			Expires: time.Now().Add(time.Hour),
		}
		ts.authedUser = user
	})
}

func (ts *testServer) createTestSession(t *testing.T, userID uuid.UUID, projectName string) *types.IGASession {
	t.Helper()

	session, err := ts.iga.CreateSession(t.Context(), types.IGASessionToCreate{
		UserID:      userID,
		ProjectName: projectName,
	})
	require.NoError(t, err)
	require.NotNil(t, session)

	return session
}

func (ts *testServer) createTestQuestion(t *testing.T, userID, sessionID uuid.UUID, questionText string) *types.IGAScriptQuestion {
	t.Helper()

	question, err := ts.iga.CreateQuestion(t.Context(), userID, types.IGAScriptQuestionToCreate{
		IGASessionID: sessionID,
		QuestionText: questionText,
	})
	require.NoError(t, err)
	require.NotNil(t, question)

	return question
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
	payload := createIGAPayload{
		ProjectName: "Test Project",
		ContextData: APIIgaSessionContextData{
			StakeholderSummary: "Stakeholder Summary",
			ProjectSummary:     "Project Summary",
			InterviewGoal:      "Interview Goal",
			InterviewAdvice:    "Interview Advice",
		},
	}
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	req := ts.newAuthenticatedRequest(t, http.MethodPost, "/api/v1/iga", payloadJSON)
	req.Header.Set("Content-Type", "application/json")

	res, err := ts.server.Client().Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode)

	var igaResp createIGAResponse
	err = json.NewDecoder(res.Body).Decode(&igaResp)
	require.NoError(t, err)

	require.NotEqual(t, uuid.Nil, igaResp.ID)
	assert.Equal(t, payload.ProjectName, igaResp.ProjectName)
	assert.Equal(t, payload.ContextData, igaResp.ContextData)

	t.Log("iga create response", igaResp)
}

func TestListIGASessions(t *testing.T) {
	t.Parallel()

	t.Run("no sessions", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga", nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusOK, res.StatusCode)

		var sessions getIGAResponse
		err = json.NewDecoder(res.Body).Decode(&sessions)
		require.NoError(t, err)
		assert.Len(t, sessions, 0)
	})

	t.Run("one session", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		// Create a session first
		payload := createIGAPayload{
			ProjectName: "My First Project",
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, "/api/v1/iga", payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		createRes, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, createRes.StatusCode)

		// Now list the sessions
		listReq := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga", nil)
		listRes, err := ts.server.Client().Do(listReq)
		require.NoError(t, err)
		defer listRes.Body.Close()

		require.Equal(t, http.StatusOK, listRes.StatusCode)

		var sessions getIGAResponse
		err = json.NewDecoder(listRes.Body).Decode(&sessions)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
		assert.Equal(t, "My First Project", sessions[0].ProjectName)
		t.Log("sessions", sessions)
	})
}

func TestGetIGASession(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		createdSession := ts.createTestSession(t, ts.authedUser.ID, "Get Me")

		req := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga/"+createdSession.ID.String(), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusOK, res.StatusCode)

		var fetchedSession igaSessionResponse
		err = json.NewDecoder(res.Body).Decode(&fetchedSession)
		require.NoError(t, err)

		assert.Equal(t, createdSession.ID, fetchedSession.ID)
		assert.Equal(t, "Get Me", fetchedSession.ProjectName)
		t.Log("session", fetchedSession)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga/"+uuid.NewString(), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		// Create a session for the default authenticated user
		ts.doAuth(t)
		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User Session")

		// Create a new, separate user and try to fetch the session
		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodGet, ts.server.URL+"/api/v1/iga/"+otherUserSession.ID.String(), nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})

		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode) // Expect Not Found, not Forbidden
		t.Log("Not Found", res.StatusCode)
	})
}

func TestPatchIGASession(t *testing.T) {
	t.Parallel()

	t.Run("patch project name success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		createdSession := ts.createTestSession(t, ts.authedUser.ID, "Original Project Name")

		newProjectName := "Updated Project Name"
		payload := patchIGAMetaPayload{ProjectName: &newProjectName}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, "/api/v1/iga/"+createdSession.ID.String(), payloadJSON)
		req.Header.Set("Content-Type", "application/json")

		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusOK, res.StatusCode)

		var patchedSession patchIGAMetaResponse
		err = json.NewDecoder(res.Body).Decode(&patchedSession)
		require.NoError(t, err)

		assert.Equal(t, createdSession.ID, patchedSession.ID)
		assert.Equal(t, newProjectName, patchedSession.ProjectName)
		assert.Equal(t, APIIgaSessionContextData(createdSession.ContextData), patchedSession.ContextData)
		t.Log("created session name", createdSession.ProjectName, "patched session name", patchedSession.ProjectName)
	})

	t.Run("patch context data success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		createdSession := ts.createTestSession(t, ts.authedUser.ID, "Project with Context")

		newContextData := APIIgaSessionContextData{
			StakeholderSummary: "New Stakeholder Summary",
			ProjectSummary:     "New Project Summary",
			InterviewGoal:      "New Interview Goal",
			InterviewAdvice:    "New Interview Advice",
		}
		payload := patchIGAMetaPayload{ContextData: &newContextData}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, "/api/v1/iga/"+createdSession.ID.String(), payloadJSON)
		req.Header.Set("Content-Type", "application/json")

		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusOK, res.StatusCode)

		var patchedSession patchIGAMetaResponse
		err = json.NewDecoder(res.Body).Decode(&patchedSession)
		require.NoError(t, err)

		assert.Equal(t, createdSession.ID, patchedSession.ID)
		assert.Equal(t, createdSession.ProjectName, patchedSession.ProjectName) // Should remain unchanged
		assert.Equal(t, newContextData, patchedSession.ContextData)

		t.Log("new context data", newContextData)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		newProjectName := "Non Existent Project"
		payload := patchIGAMetaPayload{ProjectName: &newProjectName}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, "/api/v1/iga/"+uuid.NewString(), payloadJSON)
		req.Header.Set("Content-Type", "application/json")

		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		// Create a session for the default authenticated user
		ts.doAuth(t)
		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User Session")

		// Create a new, separate user and try to patch the session
		intruder := ts.createTestUser(t, "intruder2@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		newProjectName := "Intruder Project Name"
		payload := patchIGAMetaPayload{ProjectName: &newProjectName}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPatch, ts.server.URL+"/api/v1/iga/"+otherUserSession.ID.String(), bytes.NewReader(payloadJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})

		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})
}
