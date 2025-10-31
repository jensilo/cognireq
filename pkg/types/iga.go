package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrJSONMarshal   = errors.New("json marshal error")
	ErrJSONUnmarshal = errors.New("json unmarshal error")
)

type IGASessionContextData struct {
	StakeholderSummary string `json:"stakeholder_summary"`
	ProjectSummary     string `json:"project_summary"`
	InterviewGoal      string `json:"interview_goal"`
	InterviewAdvice    string `json:"interview_advice"`
}

func IntoIGASessionContextData(b []byte) (IGASessionContextData, error) {
	var d IGASessionContextData
	err := json.Unmarshal(b, &d)
	if err != nil {
		return d, fmt.Errorf("%w: invalid IGASessionContextData JSON: %s", ErrJSONUnmarshal, b)
	}

	return d, nil
}

func (d IGASessionContextData) JSON() ([]byte, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid IGASessionContextData: %+v", ErrJSONMarshal, d)
	}

	return data, nil
}

type IGASessionSystemMetadata struct {
	LLModel string `json:"ll_model"`
}

func IntoIGASessionSystemMetadata(b []byte) (IGASessionSystemMetadata, error) {
	if len(b) < 1 {
		return IGASessionSystemMetadata{}, nil
	}

	var d IGASessionSystemMetadata
	err := json.Unmarshal(b, &d)
	if err != nil {
		return d, fmt.Errorf("%w: invalid IGASessionSystemMetadata JSON: %s", ErrJSONUnmarshal, b)
	}

	return d, nil
}

func (m IGASessionSystemMetadata) JSON() ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid IGASessionSystemMetadata: %+v", ErrJSONMarshal, m)
	}

	return data, nil
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

func IntoIGAScriptQuestionMetadata(b []byte) (IGAScriptQuestionMetadata, error) {
	var d IGAScriptQuestionMetadata
	err := json.Unmarshal(b, &d)
	if err != nil {
		return d, fmt.Errorf("%w: invalid IGAScriptQuestionMetadata JSON: %s", ErrJSONUnmarshal, b)
	}

	return d, nil
}

func (m IGAScriptQuestionMetadata) JSON() ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid IGAScriptQuestionMetadata: %+v", ErrJSONMarshal, m)
	}

	return data, nil
}

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

func IntoIGAChatLogMetadata(b []byte) (IGAChatLogMetadata, error) {
	var d IGAChatLogMetadata
	err := json.Unmarshal(b, &d)
	if err != nil {
		return d, fmt.Errorf("%w: invalid IGAChatLogMetadata JSON: %s", ErrJSONUnmarshal, b)
	}

	return d, nil
}

func (m IGAChatLogMetadata) JSON() ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid IGAChatLogMetadata: %+v", ErrJSONMarshal, m)
	}

	return data, nil
}

type IGAChatLogRole string

const (
	IGAChatLogRoleUser      IGAChatLogRole = "user"
	IGAChatLogRoleAssistant IGAChatLogRole = "assistant"
)

func IntoIGAChatLogRole(value string) (IGAChatLogRole, error) {
	switch value {
	case string(IGAChatLogRoleUser),
		string(IGAChatLogRoleAssistant):
		return IGAChatLogRole(value), nil
	default:
		return "", errors.New("invalid chat role")
	}
}

type IGAChatLog struct {
	ID           uuid.UUID
	IGASessionID uuid.UUID
	Role         IGAChatLogRole
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
