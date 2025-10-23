package pkg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	return nil, nil
}

func (is *IGAService) ChatLogBySessionID(ctx context.Context, userID, sessionID uuid.UUID) ([]types.IGAChatLog, error) {
	return nil, nil
}

func (is *IGAService) SendMessage(ctx context.Context, userID, sessionID uuid.UUID, messageText string, messageMetaData types.IGAChatLogMetadata) (*types.IGAChatLog, error) {
	return nil, nil
}

func (is *IGAService) RetryMessage(ctx context.Context, userID, sessionID uuid.UUID) (*types.IGAChatLog, error) {
	return nil, nil
}

func (is *IGAService) QuestionsBySessionID(ctx context.Context, userID, sessionID uuid.UUID) ([]types.IGAScriptQuestion, error) {
	return nil, nil
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
