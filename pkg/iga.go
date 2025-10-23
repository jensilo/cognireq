package pkg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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

func intoIGASessions(sessions []db.IgaSession) ([]types.IGASession, error) {
	igaSessions := make([]types.IGASession, len(sessions))
	var err error

	for i, session := range sessions {
		contextData, errContextData := types.IntoIGASessionContextData(session.ContextData)
		if errContextData != nil {
			err = errors.Join(err, fmt.Errorf("%w: %s", errContextData, session.ContextData))
		}

		var systemMetadata types.IGASessionSystemMetadata
		if len(session.SystemMetadata) > 0 {
			var errSystemMetadata error
			systemMetadata, errSystemMetadata = types.IntoIGASessionSystemMetadata(session.SystemMetadata)
			if errSystemMetadata != nil {
				err = errors.Join(err, fmt.Errorf("%w: %s", errSystemMetadata, session.SystemMetadata))
			}
		}

		igaSessions[i] = types.IGASession{
			ID:             session.ID,
			UserID:         session.UserID,
			ProjectName:    session.ProjectName,
			ContextData:    contextData,
			SystemMetadata: systemMetadata,
			CreatedAt:      session.CreatedAt.Time,
			UpdatedAt:      session.UpdatedAt.Time,
		}
	}

	return igaSessions, err
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
	return nil, nil
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
