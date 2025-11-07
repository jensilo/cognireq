package pkg

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jensilo/cognireq/pkg/db"
	"github.com/jensilo/cognireq/pkg/types"
)

const (
	readableIDChars      = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	readableIDLength     = 3
	readableIDFormat     = "Q-%s"
	maxReadableIDRetries = 10
)

var (
	ErrNotFound = errors.New("not found")
)

type IGAService struct {
	queries *db.Queries
}

func NewIGAService(queries *db.Queries) *IGAService {
	return &IGAService{queries: queries}
}

func (is *IGAService) SessionsByUser(ctx context.Context, userID uuid.UUID) ([]types.IGASession, error) {
	sessions, err := is.queries.IGASessionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	igaSessions, err := intoIGASessions(sessions)
	if err != nil {
		return nil, err
	}

	return igaSessions, nil
}

func intoIGASession(session db.IgaSession) (types.IGASession, error) {
	contextData, err := types.IntoIGASessionContextData(session.ContextData)
	if err != nil {
		return types.IGASession{}, fmt.Errorf("failed to parse context data for IGA session %s: %w", session.ID, err)
	}

	systemMetadata, err := types.IntoIGASessionSystemMetadata(session.SystemMetadata)
	if err != nil {
		return types.IGASession{}, fmt.Errorf("failed to parse system metadata for IGA session %s: %w", session.ID, err)
	}

	sess := types.IGASession{
		ID:             session.ID,
		UserID:         session.UserID,
		ProjectName:    session.ProjectName,
		ContextData:    contextData,
		SystemMetadata: systemMetadata,
		CreatedAt:      session.CreatedAt.Time,
	}

	if session.UpdatedAt.Valid {
		sess.UpdatedAt = &session.UpdatedAt.Time
	}

	return sess, nil
}

func intoIGASessions(sessions []db.IgaSession) ([]types.IGASession, error) {
	igaSessions := make([]types.IGASession, len(sessions))
	var allErrors error

	for i, session := range sessions {
		converted, err := intoIGASession(session)
		if err != nil {
			allErrors = errors.Join(allErrors, err)
			continue
		}
		igaSessions[i] = converted
	}

	return igaSessions, allErrors
}

func (is *IGAService) CreateSession(ctx context.Context, toCreate types.IGASessionToCreate) (*types.IGASession, error) {
	contextDataJSON, err := toCreate.ContextData.JSON()
	if err != nil {
		return nil, err
	}
	systemMetadataJSON, err := toCreate.SystemMetadata.JSON()
	if err != nil {
		return nil, err
	}

	session, err := is.queries.CreateIGASession(ctx, db.CreateIGASessionParams{
		UserID:         toCreate.UserID,
		ProjectName:    toCreate.ProjectName,
		ContextData:    contextDataJSON,
		SystemMetadata: systemMetadataJSON,
	})
	if err != nil {
		return nil, err
	}

	contextData, err := types.IntoIGASessionContextData(session.ContextData)
	if err != nil {
		return nil, err
	}
	systemMetadata, err := types.IntoIGASessionSystemMetadata(session.SystemMetadata)
	if err != nil {
		return nil, err
	}

	return &types.IGASession{
		ID:             session.ID,
		UserID:         session.UserID,
		ProjectName:    session.ProjectName,
		ContextData:    contextData,
		SystemMetadata: systemMetadata,
		CreatedAt:      session.CreatedAt.Time,
		UpdatedAt:      nil,
	}, nil
}

func (is *IGAService) SessionByID(ctx context.Context, userID, sessionID uuid.UUID) (*types.IGASession, error) {
	session, err := is.queries.IGASessionByID(ctx, db.IGASessionByIDParams{
		UserID: userID,
		ID:     sessionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	converted, err := intoIGASession(session)
	if err != nil {
		return nil, err
	}

	return &converted, nil
}

func (is *IGAService) PatchSession(ctx context.Context, userID, sessionID uuid.UUID, toUpdate types.IGASessionToUpdate) (*types.IGASession, error) {
	params := db.PatchIGASessionParams{
		UserID: userID,
		ID:     sessionID,
	}

	if toUpdate.ProjectName != nil {
		params.ProjectName = pgtype.Text{String: *toUpdate.ProjectName, Valid: true}
	}

	if toUpdate.ContextData != nil {
		jsonData, err := toUpdate.ContextData.JSON()
		if err != nil {
			return nil, err
		}
		params.ContextData = jsonData
	}

	if toUpdate.SystemMetadata != nil {
		jsonData, err := toUpdate.SystemMetadata.JSON()
		if err != nil {
			return nil, err
		}
		params.SystemMetadata = jsonData
	}

	updatedSession, err := is.queries.PatchIGASession(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	converted, err := intoIGASession(updatedSession)
	if err != nil {
		return nil, err
	}

	return &converted, nil
}

func (is *IGAService) ChatLogBySessionID(ctx context.Context, userID, sessionID uuid.UUID) ([]types.IGAChatLog, error) {
	sessionExists, err := is.queries.IGASessionExistsForUser(ctx, db.IGASessionExistsForUserParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return nil, err
	}
	if !sessionExists {
		return nil, ErrNotFound
	}

	logs, err := is.queries.IGAChatLogsBySessionID(ctx, db.IGAChatLogsBySessionIDParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return nil, err
	}

	return intoIGAChatLogs(logs)
}

func intoIGAChatLog(log db.IgaChatLog) (types.IGAChatLog, error) {
	var metadata types.IGAChatLogMetadata
	if len(log.Metadata) > 0 {
		var err error
		metadata, err = types.IntoIGAChatLogMetadata(log.Metadata)
		if err != nil {
			return types.IGAChatLog{}, fmt.Errorf("failed to parse metadata for chat log %s: %w", log.ID, err)
		}
	}

	role, err := types.IntoIGAChatLogRole(log.Role)
	if err != nil {
		return types.IGAChatLog{}, fmt.Errorf("invalid chat log role %s: %w", log.Role, err)
	}

	return types.IGAChatLog{
			ID:           log.ID,
			IGASessionID: log.IgaSessionID,
			Role:         role,
			MessageText:  log.MessageText,
			Metadata:     metadata,
			CreatedAt:    log.CreatedAt.Time,
		},
		nil
}

func intoIGAChatLogs(logs []db.IgaChatLog) ([]types.IGAChatLog, error) {
	chatLogs := make([]types.IGAChatLog, len(logs))
	var allErrors error

	for i, log := range logs {
		converted, err := intoIGAChatLog(log)
		if err != nil {
			allErrors = errors.Join(allErrors, err)
			continue
		}
		chatLogs[i] = converted
	}

	return chatLogs, allErrors
}

func (is *IGAService) CreateMessage(
	ctx context.Context,
	userID, sessionID uuid.UUID,
	messageText string,
	role types.IGAChatLogRole,
	messageMetadata *types.IGAChatLogMetadata,
) (*types.IGAChatLog, error) {
	params := db.CreateIGAChatLogParams{
		UserID:       userID,
		IgaSessionID: sessionID,
		Role:         string(role),
		MessageText:  messageText,
	}

	if messageMetadata != nil {
		metadataJSON, err := messageMetadata.JSON()
		if err != nil {
			return nil, err
		}
		params.Metadata = metadataJSON
	}

	log, err := is.queries.CreateIGAChatLog(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	converted, err := intoIGAChatLog(log)
	if err != nil {
		return nil, err
	}

	_, err = is.queries.TouchIGASessionUpdatedAt(ctx, db.TouchIGASessionUpdatedAtParams{ID: sessionID, UserID: userID})
	if err != nil {
		return nil, err
	}

	return &converted, nil
}

type SendMessageParams struct {
	UserID, SessionID uuid.UUID
	MessageText       string
	MessageMetaData   types.IGAChatLogMetadata
	MessageHandler    IGAMessageHandler
}

func (is *IGAService) SendMessage(ctx context.Context, params SendMessageParams) (*types.IGAChatLog, error) {
	existingLogs, err := is.ChatLogBySessionID(ctx, params.UserID, params.SessionID)
	if err != nil {
		return nil, err
	}

	userMetadataJSON, err := params.MessageMetaData.JSON()
	if err != nil {
		return nil, err
	}

	userChatLog, err := is.queries.CreateIGAChatLog(ctx, db.CreateIGAChatLogParams{
		UserID:       params.UserID,
		IgaSessionID: params.SessionID,
		Role:         string(types.IGAChatLogRoleUser),
		MessageText:  params.MessageText,
		Metadata:     userMetadataJSON,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	convertedUserChatLog, err := intoIGAChatLog(userChatLog)
	if err != nil {
		return nil, err
	}

	session, err := is.SessionByID(ctx, params.UserID, params.SessionID)
	if err != nil {
		return nil, err
	}
	questionnaire, err := is.QuestionsBySessionID(ctx, params.UserID, params.SessionID)
	if err != nil {
		return nil, err
	}

	res, err := params.MessageHandler.HandleIGAMessage(ctx, HandleIGAMessageParams{
		UserMessage:   convertedUserChatLog,
		Session:       *session,
		Chat:          existingLogs,
		Questionnaire: questionnaire,
	})
	if err != nil {
		return nil, err
	}

	return is.handleAIResponse(ctx, params.UserID, params.SessionID, res)
}

// handleAIResponse processes the AI's response, updating the session context,
// questionnaire, and creating the assistant's chat log.
func (is *IGAService) handleAIResponse(
	ctx context.Context,
	userID, sessionID uuid.UUID,
	res *HandleIGAMessageResult,
) (*types.IGAChatLog, error) {
	if res.SessionContextData != nil {
		_, err := is.PatchSession(ctx, userID, sessionID, types.IGASessionToUpdate{ContextData: res.SessionContextData})
		if err != nil {
			return nil, fmt.Errorf("error patching session context data after LLM response: %w", err)
		}
	}

	if len(res.QuestionsToDelete) > 0 {
		deleteParams := make([]db.BatchDeleteIGAScriptQuestionsParams, len(res.QuestionsToDelete))
		for i, questionID := range res.QuestionsToDelete {
			deleteParams[i] = db.BatchDeleteIGAScriptQuestionsParams{
				ID:           questionID,
				IgaSessionID: sessionID,
				UserID:       userID,
			}
		}
		batch := is.queries.BatchDeleteIGAScriptQuestions(ctx, deleteParams)
		var batchErr error
		batch.Exec(func(i int, err error) {
			if err != nil {
				batchErr = errors.Join(batchErr, fmt.Errorf("failed to delete question %s: %w", deleteParams[i].ID, err))
			}
		})
		if batchErr != nil {
			return nil, batchErr
		}
	}

	if len(res.QuestionsToUpdate) > 0 {
		patchParams := make([]db.BatchPatchIGAScriptQuestionsParams, 0, len(res.QuestionsToUpdate))
		for questionID, toUpdate := range res.QuestionsToUpdate {
			var metadataJSON []byte
			var err error
			if toUpdate.Metadata != nil {
				metadataJSON, err = toUpdate.Metadata.JSON()
				if err != nil {
					return nil, fmt.Errorf("failed to marshal question metadata for update: %w", err)
				}
			}

			var questionText pgtype.Text
			if toUpdate.QuestionText != nil {
				questionText = pgtype.Text{String: *toUpdate.QuestionText, Valid: true}
			}

			var noteText pgtype.Text
			if toUpdate.NoteText != nil {
				noteText = pgtype.Text{String: *toUpdate.NoteText, Valid: true}
			}

			var sortOrder pgtype.Int2
			if toUpdate.SortOrder != nil {
				sortOrder = pgtype.Int2{Int16: *toUpdate.SortOrder, Valid: true}
			}

			patchParams = append(patchParams, db.BatchPatchIGAScriptQuestionsParams{
				ID:           questionID,
				IgaSessionID: sessionID,
				UserID:       userID,
				QuestionText: questionText,
				NoteText:     noteText,
				SortOrder:    sortOrder,
				Metadata:     metadataJSON,
			})
		}
		batch := is.queries.BatchPatchIGAScriptQuestions(ctx, patchParams)
		var batchErr error
		batch.Exec(func(i int, err error) {
			if err != nil {
				batchErr = errors.Join(batchErr, fmt.Errorf("failed to update question %s: %w", patchParams[i].ID, err))
			}
		})
		if batchErr != nil {
			return nil, batchErr
		}
	}

	if len(res.QuestionsToCreate) > 0 {
		createParams := make([]db.BatchCreateIGAScriptQuestionsParams, len(res.QuestionsToCreate))
		for i, toCreate := range res.QuestionsToCreate {
			readableID, err := is.generateUniqueReadableID(ctx, sessionID)
			if err != nil {
				return nil, fmt.Errorf("failed to generate unique readable ID for new question: %w", err)
			}
			metadataJSON, err := toCreate.Metadata.JSON()
			if err != nil {
				return nil, fmt.Errorf("failed to marshal question metadata for create: %w", err)
			}
			createParams[i] = db.BatchCreateIGAScriptQuestionsParams{
				IgaSessionID: sessionID,
				ReadableID:   readableID,
				QuestionText: toCreate.QuestionText,
				NoteText:     pgtype.Text{String: toCreate.NoteText, Valid: toCreate.NoteText != ""},
				SortOrder:    toCreate.SortOrder,
				Metadata:     metadataJSON,
				UserID:       userID,
			}
		}
		batch := is.queries.BatchCreateIGAScriptQuestions(ctx, createParams)
		var batchErr error
		batch.Exec(func(i int, err error) {
			if err != nil {
				batchErr = errors.Join(batchErr, fmt.Errorf("failed to create question %s: %w", createParams[i].ReadableID, err))
			}
		})
		if batchErr != nil {
			return nil, batchErr
		}
	}

	assistantLog, err := is.queries.CreateIGAChatLog(ctx, db.CreateIGAChatLogParams{
		UserID:       userID,
		IgaSessionID: sessionID,
		Role:         res.AssistantResponse.Role,
		MessageText:  res.AssistantResponse.MessageText,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := is.NormalizeQuestionnaireSorting(ctx, userID, sessionID); err != nil {
		return nil, fmt.Errorf("failed to normalize questionnaire sorting after AI response: %w", err)
	}

	_, err = is.queries.TouchIGASessionUpdatedAt(ctx, db.TouchIGASessionUpdatedAtParams{ID: sessionID, UserID: userID})
	if err != nil {
		return nil, err
	}

	converted, err := intoIGAChatLog(assistantLog)
	if err != nil {
		return nil, err
	}

	return &converted, nil
}

func (is *IGAService) RetryMessage(ctx context.Context, userID, sessionID uuid.UUID, messageHandler IGAMessageHandler) (*types.IGAChatLog, error) {
	logs, err := is.queries.IGAChatLogsBySessionID(ctx, db.IGAChatLogsBySessionIDParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return nil, err
	}

	if len(logs) == 0 {
		return nil, ErrNotFound
	}

	// find the last user message and separate it from the rest of the history
	// iterate backwards to find the last user message
	var dbLastUserMessage db.IgaChatLog
	var dbChatHistory []db.IgaChatLog
	foundLastUserMessage := false
	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].Role == string(types.IGAChatLogRoleUser) && !foundLastUserMessage {
			dbLastUserMessage = logs[i]
			foundLastUserMessage = true
		} else {
			// Add all other messages to chatHistory, prepending to maintain chronological order
			dbChatHistory = append([]db.IgaChatLog{logs[i]}, dbChatHistory...)
		}
	}
	if dbLastUserMessage.ID == uuid.Nil {
		return nil, fmt.Errorf("no user message found to retry in session %s", sessionID)
	}
	lastUserMessage, err := intoIGAChatLog(dbLastUserMessage)
	if err != nil {
		return nil, err
	}
	chatHistory, err := intoIGAChatLogs(dbChatHistory)
	if err != nil {
		return nil, err
	}

	session, err := is.SessionByID(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	questionnaire, err := is.QuestionsBySessionID(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	res, err := messageHandler.HandleIGAMessage(ctx, HandleIGAMessageParams{
		UserMessage:   lastUserMessage,
		Session:       *session,
		Chat:          chatHistory,
		Questionnaire: questionnaire,
	})
	if err != nil {
		return nil, err
	}

	return is.handleAIResponse(ctx, userID, sessionID, res)
}

func (is *IGAService) QuestionsBySessionID(ctx context.Context, userID, sessionID uuid.UUID) ([]types.IGAScriptQuestion, error) {
	sessionExists, err := is.queries.IGASessionExistsForUser(ctx, db.IGASessionExistsForUserParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return nil, err
	}
	if !sessionExists {
		return nil, ErrNotFound
	}

	questions, err := is.queries.IGAScriptQuestionsBySessionID(ctx, db.IGAScriptQuestionsBySessionIDParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return nil, err
	}

	return intoIGAScriptQuestions(questions)
}

func (is *IGAService) NormalizeQuestionnaireSorting(ctx context.Context, userID, sessionID uuid.UUID) error {
	questions, err := is.QuestionsBySessionID(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get questions for session %s: %w", sessionID, err)
	}

	// extract IDs in their current sorted order
	orderedQuestionIDs := make([]uuid.UUID, len(questions))
	for i, q := range questions {
		orderedQuestionIDs[i] = q.ID
	}

	// use ReorderQuestions to apply the normalized sort order
	if err := is.ReorderQuestions(ctx, userID, sessionID, orderedQuestionIDs); err != nil {
		return fmt.Errorf("failed to normalize questionnaire sorting: %w", err)
	}

	return nil
}

func intoIGAScriptQuestion(q db.IgaScriptQuestion) (types.IGAScriptQuestion, error) {
	metadata, err := types.IntoIGAScriptQuestionMetadata(q.Metadata)
	if err != nil {
		return types.IGAScriptQuestion{}, fmt.Errorf("failed to parse metadata for script question %s: %w", q.ID, err)
	}

	var noteText string
	if q.NoteText.Valid {
		noteText = q.NoteText.String
	}

	return types.IGAScriptQuestion{
		ID:           q.ID,
		IGASessionID: q.IgaSessionID,
		ReadableID:   q.ReadableID,
		QuestionText: q.QuestionText,
		NoteText:     noteText,
		SortOrder:    q.SortOrder,
		Metadata:     metadata,
	}, nil
}

func intoIGAScriptQuestions(questions []db.IgaScriptQuestion) ([]types.IGAScriptQuestion, error) {
	scriptQuestions := make([]types.IGAScriptQuestion, len(questions))
	var allErrors error

	for i, q := range questions {
		converted, err := intoIGAScriptQuestion(q)
		if err != nil {
			allErrors = errors.Join(allErrors, err)
			continue
		}
		scriptQuestions[i] = converted
	}

	return scriptQuestions, allErrors
}

func (is *IGAService) generateUniqueReadableID(ctx context.Context, sessionID uuid.UUID) (string, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < maxReadableIDRetries; i++ {
		b := make([]byte, readableIDLength)
		for j := range b {
			b[j] = readableIDChars[r.Intn(len(readableIDChars))]
		}
		id := fmt.Sprintf(readableIDFormat, string(b))

		exists, err := is.queries.IGAScriptQuestionExistsByReadableID(ctx, db.IGAScriptQuestionExistsByReadableIDParams{
			IgaSessionID: sessionID,
			ReadableID:   id,
		})
		if err != nil {
			return "", fmt.Errorf("failed to check for existing readable ID: %w", err)
		}

		if !exists {
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique readable ID after %d retries", maxReadableIDRetries)
}

func (is *IGAService) CreateQuestion(ctx context.Context, userID uuid.UUID, toCreate types.IGAScriptQuestionToCreate) (*types.IGAScriptQuestion, error) {
	readableID, err := is.generateUniqueReadableID(ctx, toCreate.IGASessionID)
	if err != nil {
		return nil, err
	}

	metadataJSON, err := toCreate.Metadata.JSON()
	if err != nil {
		return nil, err
	}

	params := db.CreateIGAScriptQuestionParams{
		UserID:       userID,
		IgaSessionID: toCreate.IGASessionID,
		ReadableID:   readableID,
		QuestionText: toCreate.QuestionText,
		NoteText:     pgtype.Text{String: toCreate.NoteText, Valid: toCreate.NoteText != ""},
		SortOrder:    toCreate.SortOrder,
		Metadata:     metadataJSON,
	}

	question, err := is.queries.CreateIGAScriptQuestion(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	_, err = is.queries.TouchIGASessionUpdatedAt(ctx, db.TouchIGASessionUpdatedAtParams{ID: question.IgaSessionID, UserID: userID})
	if err != nil {
		return nil, err
	}

	if err := is.NormalizeQuestionnaireSorting(ctx, userID, toCreate.IGASessionID); err != nil {
		return nil, err
	}

	refetchedQuestion, err := is.queries.IGAScriptQuestionByID(ctx, db.IGAScriptQuestionByIDParams{
		UserID:              userID,
		IgaSessionID:        toCreate.IGASessionID,
		IgaScriptQuestionID: question.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to refetch created question after normalization: %w", err)
	}

	convertedRefetched, err := intoIGAScriptQuestion(refetchedQuestion)
	if err != nil {
		return nil, err
	}

	return &convertedRefetched, nil
}

func (is *IGAService) PatchQuestion(ctx context.Context, userID, sessionID, questionID uuid.UUID, toUpdate types.IGAScriptQuestionToUpdate) (*types.IGAScriptQuestion, error) {
	params := db.PatchIGAScriptQuestionParams{
		UserID:       userID,
		IgaSessionID: sessionID,
		ID:           questionID,
	}

	if toUpdate.QuestionText != nil {
		params.QuestionText = pgtype.Text{String: *toUpdate.QuestionText, Valid: true}
	}
	if toUpdate.NoteText != nil {
		params.NoteText = pgtype.Text{String: *toUpdate.NoteText, Valid: true}
	}
	if toUpdate.SortOrder != nil {
		params.SortOrder = pgtype.Int2{Int16: *toUpdate.SortOrder, Valid: true}
	}
	if toUpdate.Metadata != nil {
		jsonData, err := toUpdate.Metadata.JSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal question metadata: %w", err)
		}
		params.Metadata = jsonData
	}

	question, err := is.queries.PatchIGAScriptQuestion(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	_, err = is.queries.TouchIGASessionUpdatedAt(ctx, db.TouchIGASessionUpdatedAtParams{ID: question.IgaSessionID, UserID: userID})
	if err != nil {
		return nil, err
	}

	if err := is.NormalizeQuestionnaireSorting(ctx, userID, sessionID); err != nil {
		return nil, err
	}

	refetchedQuestion, err := is.queries.IGAScriptQuestionByID(ctx, db.IGAScriptQuestionByIDParams{
		UserID:              userID,
		IgaSessionID:        sessionID,
		IgaScriptQuestionID: questionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to refetch patched question after normalization: %w", err)
	}

	convertedRefetched, err := intoIGAScriptQuestion(refetchedQuestion)
	if err != nil {
		return nil, err
	}

	return &convertedRefetched, nil
}

func (is *IGAService) DeleteQuestion(ctx context.Context, userID, sessionID, questionID uuid.UUID) error {
	rowsAffected, err := is.queries.DeleteIGAScriptQuestion(ctx, db.DeleteIGAScriptQuestionParams{
		UserID:       userID,
		IgaSessionID: sessionID,
		ID:           questionID,
	})
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	_, err = is.queries.TouchIGASessionUpdatedAt(ctx, db.TouchIGASessionUpdatedAtParams{ID: sessionID, UserID: userID})
	if err != nil {
		return err
	}

	if err := is.NormalizeQuestionnaireSorting(ctx, userID, sessionID); err != nil {
		return err
	}

	return nil
}

func (is *IGAService) ReorderQuestions(ctx context.Context, userID, sessionID uuid.UUID, orderedQuestionIDs []uuid.UUID) error {
	sessionExists, err := is.queries.IGASessionExistsForUser(ctx, db.IGASessionExistsForUserParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return fmt.Errorf("failed to check session existence for reorder: %w", err)
	}
	if !sessionExists {
		return ErrNotFound
	}

	existingQuestionIDs, err := is.queries.IGAScriptQuestionIDsBySessionID(ctx, db.IGAScriptQuestionIDsBySessionIDParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return fmt.Errorf("failed to get existing question IDs for session %s: %w", sessionID, err)
	}

	if len(orderedQuestionIDs) != len(existingQuestionIDs) {
		return ErrNotFound
	}

	existingIDMap := make(map[uuid.UUID]struct{}, len(existingQuestionIDs))
	for _, id := range existingQuestionIDs {
		existingIDMap[id] = struct{}{}
	}

	for _, id := range orderedQuestionIDs {
		if _, found := existingIDMap[id]; !found {
			return ErrNotFound // an ID in the reorder list does not exist in the session
		}
	}

	batchParams := make([]db.ReorderIGAScriptQuestionsParams, len(orderedQuestionIDs))
	for i, qID := range orderedQuestionIDs {
		batchParams[i] = db.ReorderIGAScriptQuestionsParams{
			ID:        qID,
			SortOrder: int16(i), // assign 0-indexed, contiguous sort order
		}
	}

	batch := is.queries.ReorderIGAScriptQuestions(ctx, batchParams)
	batch.Exec(nil)
	if err := batch.Close(); err != nil {
		return fmt.Errorf("failed to reorder questions: %w", err)
	}

	_, err = is.queries.TouchIGASessionUpdatedAt(ctx, db.TouchIGASessionUpdatedAtParams{ID: sessionID, UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to update session timestamp after reordering questions: %w", err)
	}

	return nil
}
