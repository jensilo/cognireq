package pkg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jensilo/cognireq/pkg/db"
	"github.com/jensilo/cognireq/pkg/types"
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

	return types.IGASession{
		ID:             session.ID,
		UserID:         session.UserID,
		ProjectName:    session.ProjectName,
		ContextData:    contextData,
		SystemMetadata: systemMetadata,
		CreatedAt:      session.CreatedAt.Time,
		UpdatedAt:      session.UpdatedAt.Time,
	}, nil
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
		UpdatedAt:      session.CreatedAt.Time,
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
	metadata, err := types.IntoIGAChatLogMetadata(log.Metadata)
	if err != nil {
		return types.IGAChatLog{}, fmt.Errorf("failed to parse metadata for chat log %s: %w", log.ID, err)
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

	if len(logs) == 0 {
		return nil, fmt.Errorf("no message found to retry")
	}

	// TODO: NLP integration to regenerate the last response.
	//  potentially delete previous "assistant" response, however if there is no "assistant" message the previous might
	//  have been interrupted prematurely.
	//  - The history to send to the model would be `logs[1:]` or `logs` (excluding the last assistant message if available).
	//  - For now, we'll just create a new stubbed response.

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

	converted, err := intoIGAChatLog(assistantLog)
	if err != nil {
		return nil, err
	}

	return &converted, nil
}

func (is *IGAService) QuestionsBySessionID(ctx context.Context, userID, sessionID uuid.UUID) ([]types.IGAScriptQuestion, error) {
	questions, err := is.queries.IGAScriptQuestionsBySessionID(ctx, db.IGAScriptQuestionsBySessionIDParams{
		UserID: userID,
		ID:     sessionID,
	})
	if err != nil {
		return nil, err
	}

	return intoIGAScriptQuestions(questions)
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

func (is *IGAService) CreateQuestion(ctx context.Context, userID uuid.UUID, toCreate types.IGAScriptQuestionToCreate) (*types.IGAScriptQuestion, error) {
	return nil, nil
}

func (is *IGAService) PatchQuestion(ctx context.Context, userID, sessionID, questionID uuid.UUID, toUpdate types.IGAScriptQuestionToUpdate) (*types.IGAScriptQuestion, error) {
	return nil, nil
}

func (is *IGAService) DeleteQuestion(ctx context.Context, userID, sessionID, questionID uuid.UUID) error {
	return nil
}
