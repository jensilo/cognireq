package types

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type IGASessionContextData struct {
	StakeholderSummary string `json:"stakeholder_summary"`
	ProjectSummary     string `json:"project_summary"`
	InterviewGoal      string `json:"interview_goal"`
	InterviewAdvice    string `json:"interview_advice"`
}

type IGASessionSystemMetadata struct {
	LLModel string `json:"ll_model"`
}

type IGASession struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	ProjectName    string
	ContextData    IGASessionContextData
	SystemMetadata IGASessionSystemMetadata
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type IGASessionToCreate struct {
	UserID         uuid.UUID
	ProjectName    string
	ContextData    IGASessionContextData
	SystemMetadata IGASessionSystemMetadata
}

type IGASessionToUpdate struct {
	ProjectName    *string
	ContextData    *IGASessionContextData
	SystemMetadata *IGASessionSystemMetadata
}

type IGAScriptQuestionMetadata struct{}

type IGAScriptQuestion struct {
	ID           uuid.UUID
	IGASessionID uuid.UUID
	ReadableID   string
	QuestionText string
	NoteText     string
	SortOrder    int16
	Metadata     IGAScriptQuestionMetadata
}

type IGAScriptQuestionToCreate struct {
	IGASessionID uuid.UUID
	QuestionText string
	NoteText     string
	SortOrder    int16
	Metadata     IGAScriptQuestionMetadata
}

type IGAScriptQuestionToUpdate struct {
	QuestionText *string
	NoteText     *string
	SortOrder    *int16
	Metadata     *IGAScriptQuestionMetadata
}

type IGAChatLogInstructionType string

const (
	IGAChatLogInstructionMessage        IGAChatLogInstructionType = "message"
	IGAChatLogInstructionRefineQuestion IGAChatLogInstructionType = "refine-question"
	IGAChatLogInstructionRefineContext  IGAChatLogInstructionType = "refine-context"
)

func IntoIGAChatLogInstructionType(value string) (IGAChatLogInstructionType, error) {
	switch value {
	case string(IGAChatLogInstructionMessage),
		string(IGAChatLogInstructionRefineQuestion),
		string(IGAChatLogInstructionRefineContext):
		return IGAChatLogInstructionType(value), nil
	default:
		return "", errors.New("invalid chat instruction type")
	}
}

type IGAChatLogMetadata struct {
	Type IGAChatLogInstructionType `json:"type"`
}

type IGAChatLog struct {
	ID           uuid.UUID
	IGASessionID uuid.UUID
	Role         string
	MessageText  string
	Metadata     IGAChatLogMetadata
	CreatedAt    time.Time
}

type IGAChatLogToCreate struct {
	IGASessionID uuid.UUID
	Role         string
	MessageText  string
	Metadata     IGAChatLogMetadata
}

type IGAChatLogToUpdate struct {
	Role        string
	MessageText string
	Metadata    IGAChatLogMetadata
}
