package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jensilo/cognireq/pkg/db"
	"github.com/jensilo/cognireq/pkg/types"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrMissingPermission = errors.New("missing permission")
	ErrJSONParsing       = errors.New("json parsing error")
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
		return nil, errors.Join(ErrJSONParsing, err)
	}

	return igaSessions, nil
}

func intoIGASessions(sessions []db.IgaSession) ([]types.IGASession, error) {
	igaSessions := make([]types.IGASession, len(sessions))
	var err error

	for i, session := range sessions {
		var contextData types.IGASessionContextData
		errContextData := json.Unmarshal(session.ContextData, &contextData)
		if errContextData != nil {
			err = errors.Join(err, fmt.Errorf("%w: %s", errContextData, session.ContextData))
		}

		var systemMetadata types.IGASessionSystemMetadata
		if len(session.SystemMetadata) > 0 {
			errSystemMetadata := json.Unmarshal(session.SystemMetadata, &systemMetadata)
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
	return nil, nil
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
