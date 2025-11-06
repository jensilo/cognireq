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

	_, err = is.queries.UpdatedIGASession(ctx, db.UpdatedIGASessionParams{ID: sessionID, UserID: userID})
	if err != nil {
		return nil, err
	}

	return &converted, nil
}

func (is *IGAService) SendMessage(ctx context.Context, userID, sessionID uuid.UUID, messageText string, messageMetaData types.IGAChatLogMetadata) (*types.IGAChatLog, error) {
	userMetadataJSON, err := messageMetaData.JSON()
	if err != nil {
		return nil, err
	}

	_, err = is.queries.CreateIGAChatLog(ctx, db.CreateIGAChatLogParams{
		UserID:       userID,
		IgaSessionID: sessionID,
		Role:         string(types.IGAChatLogRoleUser),
		MessageText:  messageText,
		Metadata:     userMetadataJSON,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// TODO: NLP integration to process the message and generate a response.
	//  - Fetch conversation history.
	//  - Call NLP service.
	//  - For now, we'll just create a stubbed response.
	time.Sleep(2000 * time.Millisecond)

	assistantLog, err := is.queries.CreateIGAChatLog(ctx, db.CreateIGAChatLogParams{
		UserID:       userID,
		IgaSessionID: sessionID,
		Role:         string(types.IGAChatLogRoleAssistant),
		MessageText:  "This is a stubbed response.",
	})
	// 2nd should not fail
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	_, err = is.queries.UpdatedIGASession(ctx, db.UpdatedIGASessionParams{ID: sessionID, UserID: userID})
	if err != nil {
		return nil, err
	}

	converted, err := intoIGAChatLog(assistantLog)
	if err != nil {
		return nil, err
	}

	return &converted, nil
}

func (is *IGAService) RetryMessage(ctx context.Context, userID, sessionID uuid.UUID) (*types.IGAChatLog, error) {
	logs, err := is.queries.IGAChatLogsBySessionID(ctx, db.IGAChatLogsBySessionIDParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return nil, err
	}

	if len(logs) == 0 { // either no messages found, lacking permission, or session not found
		return nil, ErrNotFound
	}

	// TODO: NLP integration to regenerate the last response.
	//  potentially delete previous "assistant" response, however if there is no "assistant" message the previous might
	//  have been interrupted prematurely.
	//  - The history to send to the model would be `logs[1:]` or `logs` (excluding the last assistant message if available).
	//  - For now, we'll just create a new stubbed response.
	time.Sleep(100 * time.Millisecond)

	assistantLog, err := is.queries.CreateIGAChatLog(ctx, db.CreateIGAChatLogParams{
		UserID:       userID,
		IgaSessionID: sessionID,
		Role:         string(types.IGAChatLogRoleAssistant),
		MessageText:  "This is a stubbed *retry* response.",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	_, err = is.queries.UpdatedIGASession(ctx, db.UpdatedIGASessionParams{ID: sessionID, UserID: userID})
	if err != nil {
		return nil, err
	}

	converted, err := intoIGAChatLog(assistantLog)
	if err != nil {
		return nil, err
	}

	return &converted, nil
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

	_, err = is.queries.UpdatedIGASession(ctx, db.UpdatedIGASessionParams{ID: question.IgaSessionID, UserID: userID})
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

	_, err = is.queries.UpdatedIGASession(ctx, db.UpdatedIGASessionParams{ID: question.IgaSessionID, UserID: userID})
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

	_, err = is.queries.UpdatedIGASession(ctx, db.UpdatedIGASessionParams{ID: sessionID, UserID: userID})
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

	_, err = is.queries.UpdatedIGASession(ctx, db.UpdatedIGASessionParams{ID: sessionID, UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to update session timestamp after reordering questions: %w", err)
	}

	return nil
}
