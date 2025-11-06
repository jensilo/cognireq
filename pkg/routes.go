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

type APIIgaSessionContextData types.IGASessionContextData
type APIIgaScriptQuestionMetadata types.IGAScriptQuestionMetadata
type APIIgaChatLogMetadata types.IGAChatLogMetadata
type APIIgaChatLogRole types.IGAChatLogRole

const (
	CodeEmailInvalid       = "EMAIL_INVALID"
	CodeValueMissing       = "FIELD_MISSING"
	CodeInvalidValue       = "FIELD_INVALID"
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

	createIGAPath := "POST /api/v1/iga"
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
	mux.Handle(deleteQuestionPath, deleteQuestionAuth(deleteQuestionHandler(deleteQuestionLogger, services.IGA)))

	reorderQuestionsPath := "PATCH /api/v1/iga/{id}/questions/reorder"
	reorderQuestionsLogger := logger.With("route", reorderQuestionsPath)
	reorderQuestionsAuth := AuthMiddleware(reorderQuestionsLogger, services.User)
	mux.Handle(reorderQuestionsPath, reorderQuestionsAuth(HandleJSON(reorderQuestionsController(reorderQuestionsLogger, services.IGA))))

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
		user, err := service.Register(r.Context(), toCreate)
		if errors.Is(err, ErrEmailExists) {
			return nil, 0, &WebError{Code: CodeEmailExists, Status: http.StatusConflict}
		}
		if err != nil {
			logger.Error("could not register user", "err", err)
			return nil, 0, NewInternalError()
		}

		return &registerResponse{ID: user.ID.String()}, http.StatusCreated, nil
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

		SetSessionCookie(w, sessionID.String())

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
	ID          uuid.UUID                `json:"id"`
	UserID      uuid.UUID                `json:"user_id"`
	ProjectName string                   `json:"project_name"`
	ContextData APIIgaSessionContextData `json:"context_data"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   *time.Time               `json:"updated_at"`
}

type getIGAResponse []igaSessionResponse

func intoGetIGAResponse(igaSessions []types.IGASession) getIGAResponse {
	res := make(getIGAResponse, len(igaSessions))

	for i, session := range igaSessions {
		res[i] = igaSessionResponse{
			ID:          session.ID,
			UserID:      session.UserID,
			ProjectName: session.ProjectName,
			ContextData: APIIgaSessionContextData(session.ContextData),
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

type createIGAPayload struct {
	ProjectName string                   `json:"project_name"`
	ContextData APIIgaSessionContextData `json:"context_data"`
}

func (p createIGAPayload) Validate() *WebError {
	var fieldErrors []FieldError

	if p.ProjectName == "" {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeValueMissing, Field: "project_name"})
	}

	if len(fieldErrors) > 0 {
		return NewValidationError(fieldErrors)
	}

	return nil
}

type createIGAResponse igaSessionResponse

func createIGAController(logger *slog.Logger, service *IGAService) ControllerLogic[createIGAPayload, *createIGAResponse] {
	return func(w http.ResponseWriter, r *http.Request, in createIGAPayload) (*createIGAResponse, int, *WebError) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)

		toCreate := types.IGASessionToCreate{
			UserID:      user.ID,
			ProjectName: in.ProjectName,
			ContextData: types.IGASessionContextData(in.ContextData),
		}

		session, err := service.CreateSession(ctx, toCreate)
		if err != nil {
			logger.Error("could not create IGA session", "err", err)
			return nil, 0, NewInternalError()
		}

		res := createIGAResponse{
			ID:          session.ID,
			UserID:      session.UserID,
			ProjectName: session.ProjectName,
			ContextData: APIIgaSessionContextData(session.ContextData),
			CreatedAt:   session.CreatedAt,
			UpdatedAt:   session.UpdatedAt,
		}

		return &res, http.StatusCreated, nil
	}
}

func getIGAMetaHandler(logger *slog.Logger, service *IGAService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			WriteJSONError(w, NewMalformedRequestError())
			return
		}

		session, err := service.SessionByID(ctx, user.ID, id)
		if errors.Is(err, ErrNotFound) {
			WriteJSONError(w, NewNotFoundError())
			return
		}
		if err != nil {
			logger.Error("could not get IGA session", "err", err)
			WriteJSONError(w, NewInternalError())
			return
		}

		res := igaSessionResponse{
			ID:          session.ID,
			UserID:      session.UserID,
			ProjectName: session.ProjectName,
			ContextData: APIIgaSessionContextData(session.ContextData),
			CreatedAt:   session.CreatedAt,
			UpdatedAt:   session.UpdatedAt,
		}

		if err := EncodeJSON(w, http.StatusOK, res); err != nil {
			WriteJSONError(w, NewInternalError())
		}
	})
}

type patchIGAMetaPayload struct {
	ProjectName *string                   `json:"project_name"`
	ContextData *APIIgaSessionContextData `json:"context_data"`
}

func (p patchIGAMetaPayload) Validate() *WebError {
	if p.ProjectName != nil && *p.ProjectName == "" {
		return NewValidationError([]FieldError{{Code: CodeValueMissing, Field: "project_name"}})
	}

	return nil
}

type patchIGAMetaResponse igaSessionResponse

func patchIGAMetaController(logger *slog.Logger, service *IGAService) ControllerLogic[patchIGAMetaPayload, *patchIGAMetaResponse] {
	return func(w http.ResponseWriter, r *http.Request, in patchIGAMetaPayload) (*patchIGAMetaResponse, int, *WebError) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			return nil, 0, NewMalformedRequestError()
		}

		toUpdate := types.IGASessionToUpdate{
			ProjectName: in.ProjectName,
			ContextData: (*types.IGASessionContextData)(in.ContextData),
		}

		session, err := service.PatchSession(ctx, user.ID, id, toUpdate)
		if errors.Is(err, ErrNotFound) {
			return nil, 0, NewNotFoundError()
		}
		if err != nil {
			logger.Error("could not update IGA session", "err", err)
			return nil, 0, NewInternalError()
		}

		res := patchIGAMetaResponse{
			ID:          session.ID,
			UserID:      session.UserID,
			ProjectName: session.ProjectName,
			ContextData: APIIgaSessionContextData(session.ContextData),
			CreatedAt:   session.CreatedAt,
			UpdatedAt:   session.UpdatedAt,
		}

		return &res, http.StatusOK, nil
	}
}

type chatMessageResponse struct {
	ID           uuid.UUID             `json:"id"`
	IGASessionID uuid.UUID             `json:"iga_session_id"`
	Role         APIIgaChatLogRole     `json:"role"`
	MessageText  string                `json:"message_text"`
	Metadata     APIIgaChatLogMetadata `json:"metadata"`
	CreatedAt    time.Time             `json:"created_at"`
}

type getChatResponse []chatMessageResponse

func intoGetChatResponse(logs []types.IGAChatLog) getChatResponse {
	res := make(getChatResponse, len(logs))

	for i, log := range logs {
		res[i] = chatMessageResponse{
			ID:           log.ID,
			IGASessionID: log.IGASessionID,
			Role:         APIIgaChatLogRole(log.Role),
			MessageText:  log.MessageText,
			Metadata:     APIIgaChatLogMetadata(log.Metadata),
			CreatedAt:    log.CreatedAt,
		}
	}

	return res
}

func getChatHandler(logger *slog.Logger, service *IGAService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			WriteJSONError(w, NewMalformedRequestError())
			return
		}

		logs, err := service.ChatLogBySessionID(ctx, user.ID, id)
		if errors.Is(err, ErrNotFound) {
			WriteJSONError(w, NewNotFoundError())
			return
		}
		if err != nil {
			logger.Error("could not get chat logs", "err", err)
			WriteJSONError(w, NewInternalError())
			return
		}

		res := intoGetChatResponse(logs)
		if err := EncodeJSON(w, http.StatusOK, res); err != nil {
			WriteJSONError(w, NewInternalError())
		}
	})
}

type sendChatPayload struct {
	MessageText string                          `json:"message_text"`
	Instruction types.IGAChatLogInstructionType `json:"instruction"`
}

func (p sendChatPayload) Validate() *WebError {
	var fieldErrors []FieldError

	if p.MessageText == "" {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeValueMissing, Field: "message_text"})
	}

	if _, err := types.IntoIGAChatLogInstructionType(string(p.Instruction)); err != nil {
		fieldErrors = append(fieldErrors, FieldError{Code: CodeInvalidValue, Field: "instruction"})
	}

	if len(fieldErrors) > 0 {
		return NewValidationError(fieldErrors)
	}

	return nil
}

type sendChatResponse chatMessageResponse

func sendChatController(logger *slog.Logger, service *IGAService) ControllerLogic[sendChatPayload, *sendChatResponse] {
	return func(w http.ResponseWriter, r *http.Request, in sendChatPayload) (*sendChatResponse, int, *WebError) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			return nil, 0, NewMalformedRequestError()
		}

		log, err := service.SendMessage(ctx, user.ID, id, in.MessageText, types.IGAChatLogMetadata{Type: in.Instruction})
		if errors.Is(err, ErrNotFound) {
			return nil, 0, NewNotFoundError()
		}
		if err != nil {
			logger.Error("could not send message", "err", err)
			return nil, 0, NewInternalError()
		}

		res := sendChatResponse{
			ID:           log.ID,
			IGASessionID: log.IGASessionID,
			Role:         APIIgaChatLogRole(log.Role),
			MessageText:  log.MessageText,
			Metadata:     APIIgaChatLogMetadata(log.Metadata),
			CreatedAt:    log.CreatedAt,
		}

		return &res, http.StatusCreated, nil
	}
}

// TODO add that retry can contain altered messages
type retryChatPayload struct{}

func (p retryChatPayload) Validate() *WebError {
	return nil
}

type retryChatResponse chatMessageResponse

func retryChatController(logger *slog.Logger, service *IGAService) ControllerLogic[retryChatPayload, *retryChatResponse] {
	return func(w http.ResponseWriter, r *http.Request, in retryChatPayload) (*retryChatResponse, int, *WebError) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			return nil, 0, NewMalformedRequestError()
		}

		log, err := service.RetryMessage(ctx, user.ID, id)
		if errors.Is(err, ErrNotFound) {
			return nil, 0, NewNotFoundError()
		}
		if err != nil {
			logger.Error("could not retry message", "err", err)
			return nil, 0, NewInternalError()
		}

		res := retryChatResponse{
			ID:           log.ID,
			IGASessionID: log.IGASessionID,
			Role:         APIIgaChatLogRole(log.Role),
			MessageText:  log.MessageText,
			Metadata:     APIIgaChatLogMetadata(log.Metadata),
			CreatedAt:    log.CreatedAt,
		}

		return &res, http.StatusCreated, nil
	}
}

type questionResponse struct {
	ID           uuid.UUID                    `json:"id"`
	IGASessionID uuid.UUID                    `json:"iga_session_id"`
	ReadableID   string                       `json:"readable_id"`
	QuestionText string                       `json:"question_text"`
	NoteText     string                       `json:"note_text"`
	SortOrder    int16                        `json:"sort_order"`
	Metadata     APIIgaScriptQuestionMetadata `json:"metadata,omitempty"`
}

type getQuestionsResponse []questionResponse

func intoGetQuestionsResponse(questions []types.IGAScriptQuestion) getQuestionsResponse {
	res := make(getQuestionsResponse, len(questions))

	for i, q := range questions {
		res[i] = questionResponse{
			ID:           q.ID,
			IGASessionID: q.IGASessionID,
			ReadableID:   q.ReadableID,
			QuestionText: q.QuestionText,
			NoteText:     q.NoteText,
			SortOrder:    q.SortOrder,
			Metadata:     APIIgaScriptQuestionMetadata(q.Metadata),
		}
	}

	return res
}

func getQuestionsHandler(logger *slog.Logger, service *IGAService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			WriteJSONError(w, NewMalformedRequestError())
			return
		}

		questions, err := service.QuestionsBySessionID(ctx, user.ID, id)
		if errors.Is(err, ErrNotFound) {
			WriteJSONError(w, NewNotFoundError())
			return
		}
		if err != nil {
			logger.Error("could not get questions", "err", err)
			WriteJSONError(w, NewInternalError())
			return
		}

		res := intoGetQuestionsResponse(questions)
		if err := EncodeJSON(w, http.StatusOK, res); err != nil {
			WriteJSONError(w, NewInternalError())
		}
	})
}

type postQuestionPayload struct {
	QuestionText string `json:"question_text"`
	NoteText     string `json:"note_text"`
	SortOrder    int16  `json:"sort_order"`
}

func (p postQuestionPayload) Validate() *WebError {
	if p.QuestionText == "" {
		return NewValidationError([]FieldError{{Code: CodeValueMissing, Field: "question_text"}})
	}

	return nil
}

type postQuestionResponse questionResponse

func postQuestionController(logger *slog.Logger, service *IGAService) ControllerLogic[postQuestionPayload, *postQuestionResponse] {
	return func(w http.ResponseWriter, r *http.Request, in postQuestionPayload) (*postQuestionResponse, int, *WebError) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			return nil, 0, NewMalformedRequestError()
		}

		toCreate := types.IGAScriptQuestionToCreate{
			IGASessionID: id,
			QuestionText: in.QuestionText,
			NoteText:     in.NoteText,
			SortOrder:    in.SortOrder,
		}

		q, err := service.CreateQuestion(ctx, user.ID, toCreate)
		if errors.Is(err, ErrNotFound) {
			return nil, 0, NewNotFoundError()
		}
		if err != nil {
			logger.Error("could not create question", "err", err)
			return nil, 0, NewInternalError()
		}

		res := postQuestionResponse{
			ID:           q.ID,
			IGASessionID: q.IGASessionID,
			ReadableID:   q.ReadableID,
			QuestionText: q.QuestionText,
			NoteText:     q.NoteText,
			SortOrder:    q.SortOrder,
			Metadata:     APIIgaScriptQuestionMetadata(q.Metadata),
		}

		return &res, http.StatusCreated, nil
	}
}

type patchQuestionPayload struct {
	QuestionText *string `json:"question_text"`
	NoteText     *string `json:"note_text"`
	SortOrder    *int16  `json:"sort_order"`
}

func (p patchQuestionPayload) Validate() *WebError {
	if p.QuestionText != nil && *p.QuestionText == "" {
		return NewValidationError([]FieldError{{Code: CodeValueMissing, Field: "question_text"}})
	}

	return nil
}

type patchQuestionResponse questionResponse

func patchQuestionController(logger *slog.Logger, service *IGAService) ControllerLogic[patchQuestionPayload, *patchQuestionResponse] {
	return func(w http.ResponseWriter, r *http.Request, in patchQuestionPayload) (*patchQuestionResponse, int, *WebError) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			return nil, 0, NewMalformedRequestError()
		}

		qid, err := uuid.Parse(r.PathValue("qid"))
		if err != nil {
			return nil, 0, NewMalformedRequestError()
		}

		toUpdate := types.IGAScriptQuestionToUpdate{
			QuestionText: in.QuestionText,
			NoteText:     in.NoteText,
			SortOrder:    in.SortOrder,
		}

		q, err := service.PatchQuestion(ctx, user.ID, id, qid, toUpdate)
		if errors.Is(err, ErrNotFound) {
			return nil, 0, NewNotFoundError()
		}
		if err != nil {
			logger.Error("could not update question", "err", err)
			return nil, 0, NewInternalError()
		}

		res := patchQuestionResponse{
			ID:           q.ID,
			IGASessionID: q.IGASessionID,
			ReadableID:   q.ReadableID,
			QuestionText: q.QuestionText,
			NoteText:     q.NoteText,
			SortOrder:    q.SortOrder,
			Metadata:     APIIgaScriptQuestionMetadata(q.Metadata),
		}

		return &res, http.StatusOK, nil
	}
}

func deleteQuestionHandler(logger *slog.Logger, service *IGAService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			WriteJSONError(w, NewMalformedRequestError())
			return
		}

		qid, err := uuid.Parse(r.PathValue("qid"))
		if err != nil {
			WriteJSONError(w, NewMalformedRequestError())
			return
		}

		if err := service.DeleteQuestion(ctx, user.ID, id, qid); err != nil {
			logger.Error("could not delete question", "err", err)
			if errors.Is(err, ErrNotFound) {
				WriteJSONError(w, NewNotFoundError())
				return
			}
			WriteJSONError(w, NewInternalError())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

type reorderQuestionsPayload struct {
	OrderedQuestionIDs []uuid.UUID `json:"ordered_question_ids"`
}

func (p reorderQuestionsPayload) Validate() *WebError {
	if len(p.OrderedQuestionIDs) == 0 {
		return NewValidationError([]FieldError{{Code: CodeValueMissing, Field: "ordered_question_ids"}})
	}

	return nil
}

type reorderQuestionsResponse struct{}

func reorderQuestionsController(logger *slog.Logger, service *IGAService) ControllerLogic[reorderQuestionsPayload, *reorderQuestionsResponse] {
	return func(w http.ResponseWriter, r *http.Request, in reorderQuestionsPayload) (*reorderQuestionsResponse, int, *WebError) {
		ctx := r.Context()
		user := MustUserFromCtx(ctx)
		sessionID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			return nil, 0, NewMalformedRequestError()
		}

		err = service.ReorderQuestions(ctx, user.ID, sessionID, in.OrderedQuestionIDs)
		if errors.Is(err, ErrNotFound) {
			return nil, 0, NewNotFoundError()
		}
		if err != nil {
			logger.Error("could not reorder questions", "err", err)
			return nil, 0, NewInternalError()
		}

		return &reorderQuestionsResponse{}, http.StatusNoContent, nil
	}
}
