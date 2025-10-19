package pkg

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jensilo/cognireq/pkg/types"
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

	//
	//	--- User Routes ---
	//
	registerPath := "POST /api/v1/user/register"
	mux.Handle(registerPath, HandleJSON(registerController(logger.With("route", registerPath), services.User)))

	loginPath := "POST /api/v1/user/login"
	mux.Handle(loginPath, HandleJSON(loginController(logger.With("route", loginPath), services.User)))

	logoutPath := "POST /api/v1/user/logout"
	mux.Handle(logoutPath, HandleJSON(logoutController(logger.With("route", logoutPath), services.User)))

	mePath := "GET /api/v1/user/me"
	meAuth := AuthMiddleware(logger.With("route", mePath), services.User)
	mux.Handle(mePath, meAuth(meController())) // todo rename to handler

	//
	//	--- IGA Routes ---
	//
	getIGAPath := "GET /api/v1/iga"
	getIGALogger := logger.With("route", getIGAPath)
	getIGAAuth := AuthMiddleware(getIGALogger, services.User)
	mux.Handle(getIGAPath, getIGAAuth(getIGAHandler(getIGALogger, services.IGA)))

	/*createIGAPath := "POST /api/v1/iga"
	createIGALogger := logger.With("route", createIGAPath)
	createIGAAuth := AuthMiddleware(createIGALogger, services.User)
	mux.Handle(createIGAPath, createIGAAuth(HandleJSON(createIGAController(createIGALogger, services.IGA))))

	getIGAMetaPath := "GET /api/v1/iga/{id}"
	getIGAMetaLogger := logger.With("route", getIGAMetaPath)
	getIGAMetaAuth := AuthMiddleware(getIGAMetaLogger, services.User)
	mux.Handle(getIGAMetaPath, getIGAMetaAuth(getIGAMetaHandler(getIGAMetaLogger, services.IGA)))

	patchIGAMetaPath := "PATCH /api/v1/iga/{id}"
	patchIGAMetaLogger := logger.With("route", patchIGAMetaPath)
	patchIGAMetaAuth := AuthMiddleware(patchIGAMetaLogger, services.User)
	mux.Handle(patchIGAMetaPath, patchIGAMetaAuth(HandleJSON(patchIGAMetaController(patchIGAMetaLogger, services.IGA))))

	//
	// --- IGA Chat Routes ---
	//
	getChatPath := "GET /api/v1/iga/{id}/chat"
	getChatLogger := logger.With("route", getChatPath)
	getChatAuth := AuthMiddleware(getChatLogger, services.User)
	mux.Handle(getChatPath, getChatAuth(getChatHandler(getChatLogger, services.IGA)))

	sendChatPath := "POST /api/v1/iga/{id}/chat"
	sendChatLogger := logger.With("route", sendChatPath)
	sendChatAuth := AuthMiddleware(sendChatLogger, services.User)
	mux.Handle(sendChatPath, sendChatAuth(HandleJSON(sendChatController(sendChatLogger, services.IGA))))

	// retries/regenerates the latest message with potentially varying content (POST /api/v1/iga/{id}/chat/retry)
	retryChatPath := "POST /api/v1/iga/{id}/chat/retry"
	retryChatLogger := logger.With("route", retryChatPath)
	retryChatAuth := AuthMiddleware(retryChatLogger, services.User)
	mux.Handle(retryChatPath, retryChatAuth(HandleJSON(retryChatController(retryChatLogger, services.IGA))))

	//
	// --- IGA Questionnaire Routes ---
	//
	getQuestionsPath := "GET /api/v1/iga/{id}/questions"
	getQuestionsLogger := logger.With("route", getQuestionsPath)
	getQuestionsAuth := AuthMiddleware(getQuestionsLogger, services.User)
	mux.Handle(getQuestionsPath, getQuestionsAuth(getQuestionsHandler(getQuestionsLogger, services.IGA)))

	postQuestionPath := "POST /api/v1/iga/{id}/questions"
	postQuestionLogger := logger.With("route", postQuestionPath)
	postQuestionAuth := AuthMiddleware(postQuestionLogger, services.User)
	mux.Handle(postQuestionPath, postQuestionAuth(HandleJSON(postQuestionController(postQuestionLogger, services.IGA))))

	patchQuestionPath := "PATCH /api/v1/iga/{id}/questions/{qid}"
	patchQuestionLogger := logger.With("route", patchQuestionPath)
	patchQuestionAuth := AuthMiddleware(patchQuestionLogger, services.User)
	mux.Handle(patchQuestionPath, patchQuestionAuth(HandleJSON(patchQuestionController(patchQuestionLogger, services.IGA))))

	deleteQuestionPath := "DELETE /api/v1/iga/{id}/questions/{qid}"
	deleteQuestionLogger := logger.With("route", deleteQuestionPath)
	deleteQuestionAuth := AuthMiddleware(deleteQuestionLogger, services.User)
	mux.Handle(deleteQuestionPath, deleteQuestionAuth(deleteQuestionHandler(deleteQuestionLogger, services.IGA)))*/

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

		if err := service.Logout(r.Context(), sessionID); err != nil {
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

type igaSessionResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	ProjectName string    `json:"project_name"`
	ContextData struct {
		StakeholderSummary string `json:"stakeholder_summary"`
		ProjectSummary     string `json:"project_summary"`
		InterviewGoal      string `json:"interview_goal"`
		InterviewAdvice    string `json:"interview_advice"`
	} `json:"context_data"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type getIGAResponse []igaSessionResponse

func intoGetIGAResponse(igaSessions []types.IGASession) getIGAResponse {
	res := make(getIGAResponse, len(igaSessions))

	for key, session := range igaSessions {
		res[key] = igaSessionResponse{
			ID:          session.ID,
			UserID:      session.UserID,
			ProjectName: session.ProjectName,
			ContextData: session.ContextData,
			CreatedAt:   session.CreatedAt,
			UpdatedAt:   session.UpdatedAt,
		}
	}

	return res
}

func getIGAHandler(logger *slog.Logger, service *IGAService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		sessions, err := service.SessionsByUser(ctx, user.ID)
		if err != nil {
			logger.Error("error retrieving IGA", "err", err)
			WriteJSONError(w, NewInternalError())
			return
		}

		res := intoGetIGAResponse(sessions)

		if err := EncodeJSON(w, http.StatusOK, res); err != nil {
			WriteJSONError(w, NewInternalError())
		}
	})
}
