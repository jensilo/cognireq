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
	ErrJSONParsing = errors.New("json parsing error")
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

	for key, session := range sessions {
		var contextData types.IGASessionContextData
		errContextData := json.Unmarshal(session.ContextData, &contextData)
		if errContextData != nil {
			err = errors.Join(err, fmt.Errorf("%w: %s", errContextData, session.ContextData))
		}

		var systemMetadata types.IGASessionSystemMetadata
		errSystemMetadata := json.Unmarshal(session.SystemMetadata, &systemMetadata)
		if errSystemMetadata != nil {
			err = errors.Join(err, fmt.Errorf("%w: %s", errSystemMetadata, session.SystemMetadata))
		}

		igaSessions[key] = types.IGASession{
			ID:             session.ID,
			UserID:         session.UserID,
			ProjectName:    session.ProjectName,
			ContextData:    contextData,
			SystemMetadata: systemMetadata,
			CreatedAt:      session.CreatedAt.Time,
			UpdatedAt:      session.UpdatedAt.Time,
		}
	}
}
