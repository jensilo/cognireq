package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jensilo/cognireq/pkg/cfg"
	"github.com/jensilo/cognireq/pkg/db"
	"github.com/jensilo/cognireq/pkg/prmpt"
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
	promptService := prmpt.NewPromptService()
	geminiService, err := NewGeminiAIService(context.Background(), cfg.DefaultGetEnv, promptService, logger)
	assert.NoError(t, err)
	services := NewServices(userService, igaService, promptService, geminiService)

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

func (ts *testServer) newUnauthenticatedRequest(t *testing.T, method, urlPath string, body []byte) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, ts.server.URL+urlPath, bytes.NewReader(body))
	require.NoError(t, err)

	return req
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
			Name:  "session_id",
			Value: sessionID.String(),
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

func (ts *testServer) createTestChatLog(t *testing.T, userID, sessionID uuid.UUID, messageText string, role types.IGAChatLogRole) *types.IGAChatLog {
	t.Helper()

	var metadata *types.IGAChatLogMetadata
	if role == types.IGAChatLogRoleUser {
		metadata = &types.IGAChatLogMetadata{Type: types.IGAChatLogInstructionMessage}
	}
	chatLog, err := ts.iga.CreateMessage(t.Context(), userID, sessionID, messageText, role, metadata)
	require.NoError(t, err)
	require.NotNil(t, chatLog)

	return chatLog
}

func assertSortedQuestions(t *testing.T, questions []types.IGAScriptQuestion) bool {

	for i, q := range questions {
		if q.SortOrder != int16(i) {
			return assert.Failf(t, "Sort order mismatch", "Question at index %d has SortOrder %d, expected %d", i, q.SortOrder, i)
		}
	}
	return true
}

func TestPing(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	res := ts.get(t, "/api/v1/ping")
	defer func() { assert.NoError(t, res.Body.Close()) }()
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
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", ts.server.URL, "/api/v1/user/register"), bytes.NewReader(payload))
	require.NoError(t, err)
	res, err := ts.server.Client().Do(req)
	require.NoError(t, err)
	defer func() { assert.NoError(t, res.Body.Close()) }()
	require.Equal(t, http.StatusCreated, res.StatusCode, "invalid status code")
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err, "can not read body")

	reg, err := DecodeJSONFromBytes[registerResponse](data)
	require.NoError(t, err, "can not decode json")
	require.NotEmpty(t, reg.ID, "empty registration id")
	t.Log("User ID", reg.ID)
}

func TestLogin(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		email := "login@test.com"
		password := "password123"
		ts.createTestUser(t, email, password)

		payload, err := json.Marshal(loginPayload{
			Email:    email,
			Password: password,
		})
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", ts.server.URL, "/api/v1/user/login"), bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.NotNil(t, res.Header.Get("Set-Cookie"))
	})

	t.Run("invalid credentials", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		email := "invalid@test.com"
		password := "password123"
		ts.createTestUser(t, email, password)

		payload, err := json.Marshal(loginPayload{
			Email:    email,
			Password: "wrongpassword",
		})
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", ts.server.URL, "/api/v1/user/login"), bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Equal(t, CodeInvalidCredentials, webErr.Code)
	})

	t.Run("non-existent user", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		payload, err := json.Marshal(loginPayload{
			Email:    "nonexistent@test.com",
			Password: "password123",
		})
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", ts.server.URL, "/api/v1/user/login"), bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Equal(t, CodeInvalidCredentials, webErr.Code)
	})

	t.Run("validation error (missing email)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		payload, err := json.Marshal(loginPayload{
			Email:    "",
			Password: "password123",
		})
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", ts.server.URL, "/api/v1/user/login"), bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Len(t, webErr.Fields, 1)
		assert.Equal(t, CodeValueMissing, webErr.Fields[0].Code)
		assert.Equal(t, "email", webErr.Fields[0].Field)
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t) // this creates a session and sets a cookie

		req := ts.newAuthenticatedRequest(t, http.MethodPost, "/api/v1/user/logout", []byte("{}"))
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// verify that the session cookie is cleared
		cookies := res.Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "session_id", cookies[0].Name)
		assert.Equal(t, "", cookies[0].Value)
		assert.True(t, cookies[0].MaxAge < 0 || cookies[0].Expires.Before(time.Now()))

		// verify that the session is deleted from the database
		_, err = ts.user.LoggedInUser(t.Context(), uuid.MustParse(ts.cookie.Value))
		assert.Error(t, err) // session should be expired/not found
	})

	t.Run("no session cookie", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newUnauthenticatedRequest(t, http.MethodPost, "/api/v1/user/logout", []byte("{}"))
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusOK, res.StatusCode)
		// no Set-Cookie header should be present to clear a non-existent cookie
		assert.Empty(t, res.Header.Get("Set-Cookie"))
	})

	t.Run("malformed session ID in cookie", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", ts.server.URL, "/api/v1/user/logout"), bytes.NewReader([]byte("{}")))
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "invalid-uuid"})
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Equal(t, CodeMalformedRequestError, webErr.Code)
	})
}

func TestMe(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/user/me", nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		var meResp meResponse
		err = json.NewDecoder(res.Body).Decode(&meResp)
		require.NoError(t, err)
		assert.Equal(t, ts.authedUser.ID, meResp.ID)
		assert.Equal(t, ts.authedUser.Email, meResp.Email)
		assert.Equal(t, ts.authedUser.FirstName, meResp.FirstName)
		assert.Equal(t, ts.authedUser.LastName, meResp.LastName)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newUnauthenticatedRequest(t, http.MethodGet, "/api/v1/user/me", nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})
}

func TestCreateIGASession(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
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
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var igaResp createIGAResponse
		err = json.NewDecoder(res.Body).Decode(&igaResp)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, igaResp.ID)
		assert.Equal(t, payload.ProjectName, igaResp.ProjectName)
		assert.Equal(t, payload.ContextData, igaResp.ContextData)
		t.Log("iga create response", igaResp)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		payload := createIGAPayload{
			ProjectName: "Test Project",
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newUnauthenticatedRequest(t, http.MethodPost, "/api/v1/iga", payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("validation error (empty project name)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		payload := createIGAPayload{
			ProjectName: "", // empty project name
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, "/api/v1/iga", payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Len(t, webErr.Fields, 1)
		assert.Equal(t, CodeValueMissing, webErr.Fields[0].Code)
		assert.Equal(t, "project_name", webErr.Fields[0].Field)
	})
}

func TestListIGASessions(t *testing.T) {
	t.Parallel()

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newUnauthenticatedRequest(t, http.MethodGet, "/api/v1/iga", nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("no sessions", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga", nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var sessions getIGAResponse
		err = json.NewDecoder(res.Body).Decode(&sessions)
		require.NoError(t, err)
		assert.Len(t, sessions, 0)
	})

	t.Run("one session", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

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

		listReq := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga", nil)
		listRes, err := ts.server.Client().Do(listReq)
		require.NoError(t, err)
		defer func() { assert.NoError(t, listRes.Body.Close()) }()
		require.Equal(t, http.StatusOK, listRes.StatusCode)

		var sessions getIGAResponse
		err = json.NewDecoder(listRes.Body).Decode(&sessions)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
		assert.Equal(t, "My First Project", sessions[0].ProjectName)
		t.Log("sessions", sessions)
	})

	t.Run("sessions from other users are not returned", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		selfSession := ts.createTestSession(t, ts.authedUser.ID, "My Session")
		otherUser := ts.createTestUser(t, "other@test.com", "password123")
		_ = ts.createTestSession(t, otherUser.ID, "Other User's Session")

		listReq := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga", nil)
		listRes, err := ts.server.Client().Do(listReq)
		require.NoError(t, err)
		defer func() { assert.NoError(t, listRes.Body.Close()) }()
		require.Equal(t, http.StatusOK, listRes.StatusCode)

		var sessions getIGAResponse
		err = json.NewDecoder(listRes.Body).Decode(&sessions)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
		assert.Equal(t, selfSession.ProjectName, sessions[0].ProjectName)
		assert.Equal(t, selfSession.ID, sessions[0].ID)
		t.Log("sessions", sessions)
	})
}

func TestGetIGASession(t *testing.T) {
	t.Parallel()

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newUnauthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s", uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		createdSession := ts.createTestSession(t, ts.authedUser.ID, "Get Me")

		req := ts.newAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s", createdSession.ID.String()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var fetchedSession igaSessionResponse
		err = json.NewDecoder(res.Body).Decode(&fetchedSession)
		require.NoError(t, err)
		assert.Equal(t, createdSession.ID, fetchedSession.ID)
		assert.Equal(t, "Get Me", fetchedSession.ProjectName)
		t.Log("session", fetchedSession)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga/invalid-uuid", nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User Session")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/iga/%s", ts.server.URL, otherUserSession.ID.String()), nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
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

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s", createdSession.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
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
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s", createdSession.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var patchedSession patchIGAMetaResponse
		err = json.NewDecoder(res.Body).Decode(&patchedSession)
		require.NoError(t, err)
		assert.Equal(t, createdSession.ID, patchedSession.ID)
		assert.Equal(t, createdSession.ProjectName, patchedSession.ProjectName)
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
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s", uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User Session")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		newProjectName := "Intruder Project Name"
		payload := patchIGAMetaPayload{ProjectName: &newProjectName}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/v1/iga/%s", ts.server.URL, otherUserSession.ID.String()), bytes.NewReader(payloadJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		newProjectName := "Updated Project Name"
		payload := patchIGAMetaPayload{ProjectName: &newProjectName}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newUnauthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s", uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		newProjectName := "Updated Project Name"
		payload := patchIGAMetaPayload{ProjectName: &newProjectName}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, "/api/v1/iga/invalid-uuid", payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("validation error (empty project name in patch)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		session := ts.createTestSession(t, ts.authedUser.ID, "Validation Patch Session")

		emptyProjectName := "" // empty project name
		payload := patchIGAMetaPayload{ProjectName: &emptyProjectName}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Len(t, webErr.Fields, 1)
		assert.Equal(t, CodeValueMissing, webErr.Fields[0].Code)
		assert.Equal(t, "project_name", webErr.Fields[0].Field)
		t.Log("Bad Request", res.StatusCode)
	})
}

func TestGetChatLog(t *testing.T) {
	t.Parallel()

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newUnauthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s/chat", uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("empty chat log", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session Empty")

		req := ts.newAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var chatLogs getChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatLogs)
		require.NoError(t, err)
		assert.Len(t, chatLogs, 0)
	})

	t.Run("populated chat log", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session Populated")
		ts.createTestChatLog(t, ts.authedUser.ID, session.ID, "Hello", types.IGAChatLogRoleUser)
		ts.createTestChatLog(t, ts.authedUser.ID, session.ID, "Hi there!", types.IGAChatLogRoleAssistant)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var chatLogs getChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatLogs)
		require.NoError(t, err)
		assert.Len(t, chatLogs, 2)
		assert.Equal(t, APIIgaChatLogRole("user"), chatLogs[0].Role)
		assert.Equal(t, "Hello", chatLogs[0].MessageText)
		assert.Equal(t, APIIgaChatLogRole("assistant"), chatLogs[1].Role)
		assert.Equal(t, "Hi there!", chatLogs[1].MessageText)
		t.Log("user", chatLogs[0].MessageText)
		t.Log("assistant", chatLogs[1].MessageText)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga/invalid-uuid/chat", nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s/chat", uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User Chat Session")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/iga/%s/chat", ts.server.URL, otherUserSession.ID.String()), nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})
}

func TestSendChatController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session for Sending")
		messageText := "This is a test message."
		instructionType := types.IGAChatLogInstructionMessage

		payload := sendChatPayload{
			MessageText: messageText,
			Instruction: instructionType,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp sendChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		t.Log("message", chatResp.MessageText, "role", chatResp.Role, "created at", chatResp.CreatedAt)

		// verify the user's message is in the DB
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2)
		assert.Equal(t, messageText, logs[0].MessageText)
		assert.Equal(t, types.IGAChatLogRoleUser, logs[0].Role)
		t.Log("message", logs[0].MessageText, "role", logs[0].Role, "created at", logs[0].CreatedAt)
		t.Log("message", logs[1].MessageText, "role", logs[1].Role, "created at", logs[1].CreatedAt)
	})

	t.Run("create questions via AI", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session for AI Create Questions")
		messageText := "[/question create 3]"
		instructionType := types.IGAChatLogInstructionMessage

		payload := sendChatPayload{
			MessageText: messageText,
			Instruction: instructionType,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp sendChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		assert.Contains(t, chatResp.MessageText, "[read command: question create, param amount: 3]")

		// verify the user's message and assistant's response are in the DB
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2)
		assert.Equal(t, messageText, logs[0].MessageText)
		assert.Equal(t, types.IGAChatLogRoleUser, logs[0].Role)
		assert.Contains(t, logs[1].MessageText, "[read command: question create, param amount: 3]")
		assert.Equal(t, types.IGAChatLogRoleAssistant, logs[1].Role)

		// verify three questions were created
		questions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, questions, 3)
		assertSortedQuestions(t, questions)
		assert.Contains(t, questions[0].QuestionText, "Q no. 0:")
		assert.Contains(t, questions[1].QuestionText, "Q no. 1:")
		assert.Contains(t, questions[2].QuestionText, "Q no. 2:")
	})

	t.Run("update questions via AI", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session for AI Update Questions")

		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Original Q1")
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Original Q2")
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Original Q3")

		// ensure they are in the DB and get their readable IDs
		initialQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, initialQuestions, 3)

		readableIDs := []string{initialQuestions[0].ReadableID, initialQuestions[1].ReadableID, initialQuestions[2].ReadableID}
		messageText := fmt.Sprintf("[/question update %s %s %s]", readableIDs[0], readableIDs[1], readableIDs[2])
		instructionType := types.IGAChatLogInstructionMessage

		payload := sendChatPayload{
			MessageText: messageText,
			Instruction: instructionType,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp sendChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		assert.Contains(t, chatResp.MessageText, fmt.Sprintf("[read command: question update, param readable ids: %s %s %s]", readableIDs[0], readableIDs[1], readableIDs[2]))

		// verify the user's message and assistant's response are in the DB
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2) // original user message + AI response
		assert.Equal(t, messageText, logs[0].MessageText)
		assert.Equal(t, types.IGAChatLogRoleUser, logs[0].Role)
		assert.Contains(t, logs[1].MessageText, "[read command: question update")
		assert.Equal(t, types.IGAChatLogRoleAssistant, logs[1].Role)

		// verify questions were updated
		updatedQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, updatedQuestions, 3)
		assertSortedQuestions(t, updatedQuestions) // sort order should still be normalized

		// check that question texts have changed and contain the readable ID
		for _, q := range updatedQuestions {
			assert.Contains(t, q.QuestionText, "updated")
			assert.Contains(t, q.QuestionText, q.ReadableID)
			assert.Contains(t, q.NoteText, "updated")
			assert.Contains(t, q.NoteText, q.ReadableID)
		}
	})

	t.Run("delete questions via AI", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session for AI Delete Questions")

		// create 3 questions
		q1 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 1 to keep")
		q2 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 2 to delete")
		q3 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 3 to delete")

		// ensure they are in the DB and get their readable IDs
		initialQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, initialQuestions, 3)

		// delete q2 and q3
		readableIDsToDelete := []string{q2.ReadableID, q3.ReadableID}
		messageText := fmt.Sprintf("[/question delete %s %s]", readableIDsToDelete[0], readableIDsToDelete[1])
		instructionType := types.IGAChatLogInstructionMessage

		payload := sendChatPayload{
			MessageText: messageText,
			Instruction: instructionType,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp sendChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		assert.Contains(t, chatResp.MessageText, fmt.Sprintf("[read command: question delete, param readable ids: %s %s]", readableIDsToDelete[0], readableIDsToDelete[1]))

		// verify the user's message and assistant's response are in the DB
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2) // original user message + AI response
		assert.Equal(t, messageText, logs[0].MessageText)
		assert.Equal(t, types.IGAChatLogRoleUser, logs[0].Role)
		assert.Contains(t, logs[1].MessageText, "[read command: question delete")
		assert.Equal(t, types.IGAChatLogRoleAssistant, logs[1].Role)

		// verify only one question remains and it's q1
		remainingQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, remainingQuestions, 1)
		assertSortedQuestions(t, remainingQuestions) // Sort order should be normalized (0)
		assert.Equal(t, q1.ID, remainingQuestions[0].ID)
		assert.Equal(t, "Question 1 to keep", remainingQuestions[0].QuestionText)
	})

	t.Run("refine one context via AI", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		initialContext := types.IGASessionContextData{
			StakeholderSummary: "Initial Stakeholder Summary",
			ProjectSummary:     "Initial Project Summary",
			InterviewGoal:      "Initial Interview Goal",
			InterviewAdvice:    "Initial Interview Advice",
		}
		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session for AI Refine Context")
		session.ContextData = initialContext // manually set context data for the test
		_, err := ts.iga.PatchSession(t.Context(), ts.authedUser.ID, session.ID, types.IGASessionToUpdate{
			ContextData: &initialContext,
		})
		require.NoError(t, err)

		messageText := "[/context refine project_summary]"
		instructionType := types.IGAChatLogInstructionRefineContext

		payload := sendChatPayload{
			MessageText: messageText,
			Instruction: instructionType,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp sendChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		assert.Contains(t, chatResp.MessageText, "[read command: context refine, param fields: project_summary]")

		// verify the user's message and assistant's response are in the DB
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2) // original user message + AI response
		assert.Equal(t, messageText, logs[0].MessageText)
		assert.Equal(t, types.IGAChatLogRoleUser, logs[0].Role)
		assert.Contains(t, logs[1].MessageText, "[read command: context refine")
		assert.Equal(t, types.IGAChatLogRoleAssistant, logs[1].Role)

		// verify session context data was updated
		updatedSession, err := ts.iga.SessionByID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		assert.NotEqual(t, initialContext.ProjectSummary, updatedSession.ContextData.ProjectSummary)
		assert.Contains(t, updatedSession.ContextData.ProjectSummary, "refined by mock LLM")
		assert.Equal(t, initialContext.StakeholderSummary, updatedSession.ContextData.StakeholderSummary) // not changed
		assert.Equal(t, initialContext.InterviewGoal, updatedSession.ContextData.InterviewGoal)           // not changed
		assert.Equal(t, initialContext.InterviewAdvice, updatedSession.ContextData.InterviewAdvice)       // not changed
	})

	t.Run("refine all context via AI", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		initialContext := types.IGASessionContextData{
			StakeholderSummary: "Initial Stakeholder Summary",
			ProjectSummary:     "Initial Project Summary",
			InterviewGoal:      "Initial Interview Goal",
			InterviewAdvice:    "Initial Interview Advice",
		}
		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session for AI Refine Context")
		session.ContextData = initialContext // manually set context data for the test
		_, err := ts.iga.PatchSession(t.Context(), ts.authedUser.ID, session.ID, types.IGASessionToUpdate{
			ContextData: &initialContext,
		})
		require.NoError(t, err)

		messageText := "[/context refine project_summary stakeholder_summary interview_goal interview_advice]"
		instructionType := types.IGAChatLogInstructionRefineContext

		payload := sendChatPayload{
			MessageText: messageText,
			Instruction: instructionType,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp sendChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		assert.Contains(t, chatResp.MessageText, "[read command: context refine, param fields: project_summary stakeholder_summary interview_goal interview_advice]")

		// verify the user's message and assistant's response are in the DB
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2) // original user message + AI response
		assert.Equal(t, messageText, logs[0].MessageText)
		assert.Equal(t, types.IGAChatLogRoleUser, logs[0].Role)
		assert.Contains(t, logs[1].MessageText, "[read command: context refine")
		assert.Equal(t, types.IGAChatLogRoleAssistant, logs[1].Role)

		// verify session context data was updated
		updatedSession, err := ts.iga.SessionByID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		assert.NotEqual(t, initialContext.ProjectSummary, updatedSession.ContextData.ProjectSummary)
		assert.Contains(t, updatedSession.ContextData.ProjectSummary, "refined by mock LLM")
		assert.NotEqual(t, initialContext.StakeholderSummary, updatedSession.ContextData.StakeholderSummary)
		assert.Contains(t, updatedSession.ContextData.StakeholderSummary, "refined by mock LLM")
		assert.NotEqual(t, initialContext.InterviewGoal, updatedSession.ContextData.InterviewGoal)
		assert.Contains(t, updatedSession.ContextData.InterviewGoal, "refined by mock LLM")
		assert.NotEqual(t, initialContext.InterviewAdvice, updatedSession.ContextData.InterviewAdvice)
		assert.Contains(t, updatedSession.ContextData.InterviewAdvice, "refined by mock LLM")
	})

	t.Run("refine no context via AI bc insufficient instruction type", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		initialContext := types.IGASessionContextData{
			StakeholderSummary: "Initial Stakeholder Summary",
		}
		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session For No AI Refine Context With Insufficient Instruction Type")
		session.ContextData = initialContext // manually set context data for the test
		_, err := ts.iga.PatchSession(t.Context(), ts.authedUser.ID, session.ID, types.IGASessionToUpdate{
			ContextData: &initialContext,
		})
		require.NoError(t, err)

		messageText := "[/context refine stakeholder_summary]"
		instructionType := types.IGAChatLogInstructionMessage

		payload := sendChatPayload{
			MessageText: messageText,
			Instruction: instructionType,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp sendChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		assert.Contains(t, chatResp.MessageText, "[read command: context refine, param fields: stakeholder_summary]")

		// verify the user's message and assistant's response are in the DB
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2) // original user message + AI response
		assert.Equal(t, messageText, logs[0].MessageText)
		assert.Equal(t, types.IGAChatLogRoleUser, logs[0].Role)
		assert.Contains(t, logs[1].MessageText, "[read command: context refine")
		assert.Equal(t, types.IGAChatLogRoleAssistant, logs[1].Role)

		// verify session context data was NOT updated
		updatedSession, err := ts.iga.SessionByID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		assert.Equal(t, initialContext.ProjectSummary, updatedSession.ContextData.ProjectSummary) // not changed bc of message instruction type
	})

	t.Run("multiple commands via AI", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		// Initial setup: session with context and some questions
		initialContext := types.IGASessionContextData{
			StakeholderSummary: "Initial Stakeholder Summary for Multi",
			ProjectSummary:     "Initial Project Summary for Multi",
			InterviewGoal:      "Initial Interview Goal for Multi",
			InterviewAdvice:    "Initial Interview Advice for Multi",
		}
		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session for Multiple AI Commands")
		session.ContextData = initialContext
		_, err := ts.iga.PatchSession(t.Context(), ts.authedUser.ID, session.ID, types.IGASessionToUpdate{
			ContextData: &initialContext,
		})
		require.NoError(t, err)

		qToUpdate := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question to Update")
		qToDelete := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question to Delete")
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Another Question to Keep") // keep one question

		// construct message with multiple commands
		messageText := fmt.Sprintf(
			"[/question create 1]\n[/question update %s]\n[/question delete %s]\n[/context refine project_summary]",
			qToUpdate.ReadableID,
			qToDelete.ReadableID,
		)
		instructionType := types.IGAChatLogInstructionAll

		payload := sendChatPayload{
			MessageText: messageText,
			Instruction: instructionType,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp sendChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		assert.Contains(t, chatResp.MessageText, "[read command: question create, param amount: 1]")
		assert.Contains(t, chatResp.MessageText, fmt.Sprintf("[read command: question update, param readable ids: %s]", qToUpdate.ReadableID))
		assert.Contains(t, chatResp.MessageText, fmt.Sprintf("[read command: question delete, param readable ids: %s]", qToDelete.ReadableID))
		assert.Contains(t, chatResp.MessageText, "[read command: context refine, param fields: project_summary]")

		// verify chat logs
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2) // user message + AI response

		// verify questions: 3 initial - 1 deleted + 1 created = 3 questions
		questions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, questions, 3)
		assertSortedQuestions(t, questions)

		// check updated question
		foundUpdated := false
		for _, q := range questions {
			if q.ID == qToUpdate.ID {
				assert.Contains(t, q.QuestionText, "updated")
				foundUpdated = true
			}
		}
		assert.True(t, foundUpdated, "Question to update was not found or not updated")

		// check deleted question is gone
		foundDeleted := false
		for _, q := range questions {
			if q.ID == qToDelete.ID {
				foundDeleted = true
				break
			}
		}
		assert.False(t, foundDeleted, "Question to delete was still found")

		// verify context data was updated
		updatedSession, err := ts.iga.SessionByID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		assert.NotEqual(t, initialContext.ProjectSummary, updatedSession.ContextData.ProjectSummary)
		assert.Contains(t, updatedSession.ContextData.ProjectSummary, "refined by mock LLM")
		assert.Equal(t, initialContext.StakeholderSummary, updatedSession.ContextData.StakeholderSummary) // not refined
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		payload := sendChatPayload{
			MessageText: "test",
			Instruction: types.IGAChatLogInstructionMessage,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newUnauthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		payload := sendChatPayload{
			MessageText: "test",
			Instruction: types.IGAChatLogInstructionMessage,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, "/api/v1/iga/invalid-uuid/chat", payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("not found (non-existent session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		payload := sendChatPayload{
			MessageText: "test",
			Instruction: types.IGAChatLogInstructionMessage,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden (session owned by another user)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User's Chat Session")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		payload := sendChatPayload{
			MessageText: "I'm an intruder!",
			Instruction: types.IGAChatLogInstructionMessage,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/iga/%s/chat", ts.server.URL, otherUserSession.ID.String()), bytes.NewReader(payloadJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("validation error (empty message text)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Validation Test Session")

		payload := sendChatPayload{
			MessageText: "",
			Instruction: types.IGAChatLogInstructionMessage,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Len(t, webErr.Fields, 1)
		assert.Equal(t, CodeValueMissing, webErr.Fields[0].Code)
		assert.Equal(t, "message_text", webErr.Fields[0].Field)

		t.Log("Bad Request", res.StatusCode, "err", webErr.Code)
	})

	t.Run("validation error (invalid instruction type)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Validation Test Session")

		payload := sendChatPayload{
			MessageText: "Valid message",
			Instruction: "INVALID_INSTRUCTION_TYPE",
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Len(t, webErr.Fields, 1)
		assert.Equal(t, CodeInvalidValue, webErr.Fields[0].Code)
		assert.Equal(t, "instruction", webErr.Fields[0].Field)
		t.Log("Bad Request", res.StatusCode, "err", webErr.Code)
	})
}

func TestRetryChatController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Chat Session for Retry")
		ts.createTestChatLog(t, ts.authedUser.ID, session.ID, "Initial user message", types.IGAChatLogRoleUser)
		// service will NOT create and assistant response

		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat/retry", session.ID.String()), []byte("{}"))
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var chatResp retryChatResponse
		err = json.NewDecoder(res.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, chatResp.ID)
		assert.Equal(t, session.ID, chatResp.IGASessionID)
		assert.Equal(t, APIIgaChatLogRole(types.IGAChatLogRoleAssistant), chatResp.Role)
		assert.NotEmpty(t, chatResp.MessageText)
		assert.NotZero(t, chatResp.CreatedAt)
		t.Log("message", chatResp.MessageText, "role", chatResp.Role, "created at", chatResp.CreatedAt)

		// verify total messages in DB
		logs, err := ts.iga.ChatLogBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, logs, 2) // original user (no assistant reply), new assistant
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newUnauthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat/retry", uuid.NewString()), []byte("{}"))
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodPost, "/api/v1/iga/invalid-uuid/chat/retry", []byte("{}"))
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("not found (non-existent session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat/retry", uuid.NewString()), []byte("{}"))
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden (session owned by another user)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User's Chat Session")
		ts.createTestChatLog(t, ts.authedUser.ID, otherUserSession.ID, "User message for other session", types.IGAChatLogRoleUser)

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/iga/%s/chat/retry", ts.server.URL, otherUserSession.ID.String()), bytes.NewReader([]byte("{}")))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("no messages to retry", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Empty Chat Session")

		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/chat/retry", session.ID.String()), []byte("{}"))
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})
}

func TestGetQuestionsHandler(t *testing.T) {
	t.Parallel()

	t.Run("empty questions list", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Questions Session Empty")

		req := ts.newAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s/questions", session.ID.String()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var questions getQuestionsResponse
		err = json.NewDecoder(res.Body).Decode(&questions)
		require.NoError(t, err)
		assert.Len(t, questions, 0)
	})

	t.Run("populated questions list", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Questions Session Populated")
		q1 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 1")
		q2 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 2")

		req := ts.newAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s/questions", session.ID.String()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var questions getQuestionsResponse
		err = json.NewDecoder(res.Body).Decode(&questions)
		require.NoError(t, err)
		assert.Len(t, questions, 2)
		assert.Equal(t, q1.ID, questions[0].ID)
		assert.Equal(t, q1.QuestionText, questions[0].QuestionText)
		assert.Equal(t, q2.ID, questions[1].ID)
		assert.Equal(t, q2.QuestionText, questions[1].QuestionText)
		t.Log("question", questions[0].QuestionText)
		t.Log("question", questions[1].QuestionText)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newUnauthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s/questions", uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("forbidden (session owned by another user)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User's Questions Session")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/iga/%s/questions", ts.server.URL, otherUserSession.ID.String()), nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, "/api/v1/iga/invalid-uuid/questions", nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("not found (non-existent session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/iga/%s/questions", uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})
}

func TestPostQuestionController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Questions Session for Posting")
		questionText := "What is your favorite color?"
		noteText := "Optional note for the question."
		sortOrder := int16(0)

		payload := postQuestionPayload{
			QuestionText: questionText,
			NoteText:     noteText,
			SortOrder:    sortOrder,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/questions", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusCreated, res.StatusCode)

		var questionResp postQuestionResponse
		err = json.NewDecoder(res.Body).Decode(&questionResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, questionResp.ID)
		assert.Equal(t, session.ID, questionResp.IGASessionID)
		assert.NotEmpty(t, questionResp.ReadableID)
		assert.Equal(t, questionText, questionResp.QuestionText)
		assert.Equal(t, noteText, questionResp.NoteText)
		assert.Equal(t, sortOrder, questionResp.SortOrder)

		// verify the question is in the DB
		questions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, questions, 1)
		assert.Equal(t, questionResp.ID, questions[0].ID)
		t.Log("question readable id", questionResp.ReadableID,
			"question text", questionResp.QuestionText,
			"note text", questionResp.NoteText,
			"sort order", questionResp.SortOrder)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		payload := postQuestionPayload{
			QuestionText: "test",
			SortOrder:    1,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, "/api/v1/iga/invalid-uuid/questions", payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("not found (non-existent session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		payload := postQuestionPayload{
			QuestionText: "test",
			SortOrder:    1,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/questions", uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden (session owned by another user)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User's Questions Session")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		payload := postQuestionPayload{
			QuestionText: "Intruder question",
			SortOrder:    1,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/iga/%s/questions", ts.server.URL, otherUserSession.ID.String()), bytes.NewReader(payloadJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("validation error (empty question text)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		session := ts.createTestSession(t, ts.authedUser.ID, "Validation Test Session")

		payload := postQuestionPayload{
			QuestionText: "", // empty question text
			SortOrder:    1,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/questions", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Len(t, webErr.Fields, 1)
		assert.Equal(t, CodeValueMissing, webErr.Fields[0].Code)
		assert.Equal(t, "question_text", webErr.Fields[0].Field)
		t.Log("Bad Request", res.StatusCode, "err", webErr.Code)
	})

	t.Run("sort order after multiple creations", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Session for Sort Order Test")

		// create multiple questions
		for i := 0; i < 5; i++ {
			questionText := fmt.Sprintf("Question %d", i)
			payload := postQuestionPayload{
				QuestionText: questionText,
				SortOrder:    int16(i + 10), // initial sort order with + 10 offset, will be normalized
			}
			payloadJSON, err := json.Marshal(payload)
			require.NoError(t, err)
			req := ts.newAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/questions", session.ID.String()), payloadJSON)
			req.Header.Set("Content-Type", "application/json")
			res, err := ts.server.Client().Do(req)
			require.NoError(t, err)
			defer func() { assert.NoError(t, res.Body.Close()) }()
			require.Equal(t, http.StatusCreated, res.StatusCode)
		}

		// fetch all questions and verify their sort order
		questions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, questions, 5)
		assertSortedQuestions(t, questions)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		payload := postQuestionPayload{
			QuestionText: "test",
			SortOrder:    1,
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newUnauthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/v1/iga/%s/questions", uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})
}

func TestPatchQuestionController(t *testing.T) {
	t.Parallel()

	t.Run("patch question text success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Patch Question Session")
		question := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Original Question Text")

		newQuestionText := "Updated Question Text"
		payload := patchQuestionPayload{QuestionText: &newQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), question.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var patchedQuestion patchQuestionResponse
		err = json.NewDecoder(res.Body).Decode(&patchedQuestion)
		require.NoError(t, err)
		assert.Equal(t, question.ID, patchedQuestion.ID)
		assert.Equal(t, newQuestionText, patchedQuestion.QuestionText)

		// verify the change in DB
		questions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, questions, 1)
		assert.Equal(t, newQuestionText, questions[0].QuestionText)
		t.Log("new question text", questions[0].QuestionText)
	})

	t.Run("patch note text and sort order success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Patch Question Session 2")
		q1 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 1") // sort_order 0
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 2")   // sort_order 1

		newNoteText := "Updated Note Text"
		newSortOrder := int16(2) // set to two, now swaps places, after normalization becomes 1
		payload := patchQuestionPayload{NoteText: &newNoteText, SortOrder: &newSortOrder}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), q1.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var patchedQuestion patchQuestionResponse
		err = json.NewDecoder(res.Body).Decode(&patchedQuestion)
		require.NoError(t, err)
		assert.Equal(t, q1.ID, patchedQuestion.ID)
		assert.Equal(t, newNoteText, patchedQuestion.NoteText)
		assert.Equal(t, int16(1), patchedQuestion.SortOrder)

		// verify the change in DB
		questions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, questions, 2)
		assert.Equal(t, newNoteText, questions[1].NoteText)
		assert.Equal(t, int16(1), questions[1].SortOrder)
		t.Log("question", questions[1].QuestionText, "sort order", questions[1].SortOrder)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		newQuestionText := "Updated Question Text"
		payload := patchQuestionPayload{QuestionText: &newQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/invalid-uuid/questions/%s", uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("malformed question ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Session for Malformed QID")

		newQuestionText := "Updated Question Text"
		payload := patchQuestionPayload{QuestionText: &newQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/invalid-uuid", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("not found (non-existent session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		newQuestionText := "Updated Question Text"
		payload := patchQuestionPayload{QuestionText: &newQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/%s", uuid.NewString(), uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("not found (non-existent question in existing session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		session := ts.createTestSession(t, ts.authedUser.ID, "Session for Non-existent Question")

		newQuestionText := "Updated Question Text"
		payload := patchQuestionPayload{QuestionText: &newQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden (session owned by another user)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User's Session for Patch")
		otherUserQuestion := ts.createTestQuestion(t, ts.authedUser.ID, otherUserSession.ID, "Question for other user")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		newQuestionText := "Intruder's attempt to patch"
		payload := patchQuestionPayload{QuestionText: &newQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/v1/iga/%s/questions/%s", ts.server.URL, otherUserSession.ID.String(), otherUserQuestion.ID.String()), bytes.NewReader(payloadJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("validation error (empty question text in patch)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Validation Patch Session")
		question := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Original Question")

		emptyQuestionText := "" // empty question text
		payload := patchQuestionPayload{QuestionText: &emptyQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), question.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Len(t, webErr.Fields, 1)
		assert.Equal(t, CodeValueMissing, webErr.Fields[0].Code)
		assert.Equal(t, "question_text", webErr.Fields[0].Field)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("sort order after patching", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Session for Patch Sort Order Test")

		// create questions with initial sort orders that will be normalized
		q1 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 1") // sort_order 0
		q2 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 2") // sort_order 1
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 3")   // sort_order 2

		// verify initial normalized state
		initialQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, initialQuestions, 3)
		assertSortedQuestions(t, initialQuestions)

		// patch q1, changing its text. Sort order should remain normalized.
		newQuestionText := "Updated Question 1 Text"
		payload := patchQuestionPayload{QuestionText: &newQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), q1.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		// fetch all questions and verify sort order is still normalized
		questionsAfterPatch, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, questionsAfterPatch, 3)
		assertSortedQuestions(t, questionsAfterPatch)

		// patch q2, explicitly setting a non-sequential sort order. It should be normalized back.
		outOfOrderSort := int16(100)
		payload = patchQuestionPayload{SortOrder: &outOfOrderSort}
		payloadJSON, err = json.Marshal(payload)
		require.NoError(t, err)
		req = ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), q2.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err = ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		require.Equal(t, http.StatusOK, res.StatusCode)

		// fetch all questions and verify sort order is normalized again
		questionsAfterSortPatch, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, questionsAfterSortPatch, 3)
		assertSortedQuestions(t, questionsAfterSortPatch)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		newQuestionText := "Updated Question Text"
		payload := patchQuestionPayload{QuestionText: &newQuestionText}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		req := ts.newUnauthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/%s", uuid.NewString(), uuid.NewString()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})
}

func TestDeleteQuestionHandler(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Delete Question Session")
		question := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question to Delete")

		req := ts.newAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), question.ID.String()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNoContent, res.StatusCode)
		t.Log("No Content", res.StatusCode)

		// verify the question is no longer in the DB
		questions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		assert.Len(t, questions, 0)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/iga/invalid-uuid/questions/%s", uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("malformed question ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		session := ts.createTestSession(t, ts.authedUser.ID, "Session for Malformed QID Delete")

		req := ts.newAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/iga/%s/questions/invalid-uuid", session.ID.String()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		t.Log("Bad Request", res.StatusCode)
	})

	t.Run("not found (non-existent session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		req := ts.newAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/iga/%s/questions/%s", uuid.NewString(), uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("not found (non-existent question in existing session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)
		session := ts.createTestSession(t, ts.authedUser.ID, "Session for Non-existent Question Delete")

		req := ts.newAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("forbidden (session owned by another user)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User's Session for Delete")
		otherUserQuestion := ts.createTestQuestion(t, ts.authedUser.ID, otherUserSession.ID, "Question for other user to delete")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/iga/%s/questions/%s", ts.server.URL, otherUserSession.ID.String(), otherUserQuestion.ID.String()), nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		t.Log("Not Found", res.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		req := ts.newUnauthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/iga/%s/questions/%s", uuid.NewString(), uuid.NewString()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("sort order after deletion", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Session for Delete Sort Order Test")

		// create multiple questions
		q1 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 1")
		q2 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 2")
		q3 := ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 3")

		// verify initial normalized state
		initialQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, initialQuestions, 3)
		assertSortedQuestions(t, initialQuestions)

		// delete the middle question (q2)
		req := ts.newAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/v1/iga/%s/questions/%s", session.ID.String(), q2.ID.String()), nil)
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNoContent, res.StatusCode)

		// fetch remaining questions and verify sort order is normalized
		remainingQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, remainingQuestions, 2)
		assertSortedQuestions(t, remainingQuestions)
		assert.Equal(t, q1.ID, remainingQuestions[0].ID) // q1 should now be at index 0
		assert.Equal(t, q3.ID, remainingQuestions[1].ID) // q3 should now be at index 1
	})
}

func TestReorderQuestionsController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Reorder Questions Session")

		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question A")
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question B")
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question C")

		initialQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, initialQuestions, 3)
		assertSortedQuestions(t, initialQuestions)
		// can not assert ABC because `createTestQuestion` is non-deterministic (with 0 sort order)

		// define a new order
		newOrder := []uuid.UUID{initialQuestions[2].ID, initialQuestions[0].ID, initialQuestions[1].ID}
		payload := reorderQuestionsPayload{OrderedQuestionIDs: newOrder}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/reorder", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNoContent, res.StatusCode)

		// fetch questions again and verify the new order and normalized sort_order
		reorderedQuestions, err := ts.iga.QuestionsBySessionID(t.Context(), ts.authedUser.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, reorderedQuestions, 3)
		assertSortedQuestions(t, reorderedQuestions) // checks 0, 1, 2 sort order
		assert.Equal(t, initialQuestions[2].ID, reorderedQuestions[0].ID)
		assert.Equal(t, initialQuestions[0].ID, reorderedQuestions[1].ID)
		assert.Equal(t, initialQuestions[1].ID, reorderedQuestions[2].ID)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)

		sessionID := uuid.New()
		payload := reorderQuestionsPayload{OrderedQuestionIDs: []uuid.UUID{uuid.New()}}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newUnauthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/reorder", sessionID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("malformed session ID", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		payload := reorderQuestionsPayload{OrderedQuestionIDs: []uuid.UUID{uuid.New()}}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, "/api/v1/iga/invalid-uuid/questions/reorder", payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("not found (non-existent session)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		sessionID := uuid.New()
		payload := reorderQuestionsPayload{OrderedQuestionIDs: []uuid.UUID{uuid.New()}}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/reorder", sessionID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("forbidden (session owned by another user)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		otherUserSession := ts.createTestSession(t, ts.authedUser.ID, "Other User's Session")
		_ = ts.createTestQuestion(t, ts.authedUser.ID, otherUserSession.ID, "Question for other user")

		intruder := ts.createTestUser(t, "intruder@test.com", "password123")
		_, sessionID, err := ts.user.Login(t.Context(), intruder.Email, "password123")
		require.NoError(t, err)

		payload := reorderQuestionsPayload{OrderedQuestionIDs: []uuid.UUID{uuid.New()}}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/v1/iga/%s/questions/reorder", ts.server.URL, otherUserSession.ID.String()), bytes.NewReader(payloadJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("validation error (empty ordered_question_ids)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Validation Test Session")

		payload := reorderQuestionsPayload{OrderedQuestionIDs: []uuid.UUID{}} // empty list
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/reorder", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

		var webErr WebError
		err = json.NewDecoder(res.Body).Decode(&webErr)
		require.NoError(t, err)
		assert.Len(t, webErr.Fields, 1)
		assert.Equal(t, CodeValueMissing, webErr.Fields[0].Code)
		assert.Equal(t, "ordered_question_ids", webErr.Fields[0].Field)
	})

	t.Run("validation error (mismatched question IDs)", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ts.doAuth(t)

		session := ts.createTestSession(t, ts.authedUser.ID, "Mismatched IDs Test Session")
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 1")
		_ = ts.createTestQuestion(t, ts.authedUser.ID, session.ID, "Question 2")

		// provide an ordered list with a non-existent question ID
		payload := reorderQuestionsPayload{OrderedQuestionIDs: []uuid.UUID{uuid.New(), uuid.New()}}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)

		req := ts.newAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/iga/%s/questions/reorder", session.ID.String()), payloadJSON)
		req.Header.Set("Content-Type", "application/json")
		res, err := ts.server.Client().Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, res.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})
}
