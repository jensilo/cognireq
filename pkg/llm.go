package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jensilo/cognireq/pkg/cfg"
	"github.com/jensilo/cognireq/pkg/prmpt"
	"github.com/jensilo/cognireq/pkg/types"
	"google.golang.org/genai"
)

const (
	GeminiAIServiceAPIKey = "GEMINI_AI_SERVICE_API_KEY"
	GeminiAIDefaultModel  = "GEMINI_AI_DEFAULT_MODEL"
)

type GeminiAIService struct {
	client *genai.Client
	model  string
	prompt *prmpt.PromptService
	logger *slog.Logger
}

func NewGeminiAIService(ctx context.Context, envFunc cfg.GetEnvFunc, ps *prmpt.PromptService, l *slog.Logger) (*GeminiAIService, error) {
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  envFunc(GeminiAIServiceAPIKey),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("could not instantiate gemini genai client: %w", err)
	}

	m := envFunc(GeminiAIDefaultModel)
	if m == "" {
		return nil, fmt.Errorf("no gemini ai default model given, consider using gemini-2.5-flash")
	}

	return &GeminiAIService{client: c, model: m, prompt: ps, logger: l}, nil
}

type HandleIGAMessageParams struct {
	UserMessage   types.IGAChatLog
	Session       types.IGASession
	Chat          []types.IGAChatLog
	Questionnaire []types.IGAScriptQuestion
}

type HandleIGAMessageResult struct {
	AssistantResponse  types.IGAChatLogToCreate
	SessionContextData *types.IGASessionContextData
	QuestionsToUpdate  map[uuid.UUID]types.IGAScriptQuestionToUpdate
	QuestionsToCreate  []types.IGAScriptQuestionToCreate
	QuestionsToDelete  []uuid.UUID
}

type igaContextData struct {
	StakeholderSummary string `json:"stakeholder_summary"`
	ProjectSummary     string `json:"project_summary"`
	InterviewGoal      string `json:"interview_goal"`
	InterviewAdvice    string `json:"interview_advice"`
}

type igaChat struct {
	Role        string `json:"role"`
	MessageText string `json:"message_text"`
	Instruction string `json:"instruction"`
}

type igaQuestion struct {
	ReadableID   string `json:"readable_id"`
	QuestionText string `json:"question_text"`
	NoteText     string `json:"note_text"`
	SortOrder    int16  `json:"sort_order"`
}

type igaPrompt struct {
	ModelInstruction string         `json:"model_instruction"`
	UserMessageText  string         `json:"user_message_text"`
	UserInstruction  string         `json:"user_instruction"`
	ContextData      igaContextData `json:"context_data"`
	ChatHistory      []igaChat      `json:"chat_history"`
	Questionnaire    []igaQuestion  `json:"questionnaire"`
}

type igaResponse struct {
	AssistantResponseText string         `json:"assistant_response_text"`
	ContextData           igaContextData `json:"context_data"`
	QuestionsToCreate     []igaQuestion  `json:"questions_to_create"`
	QuestionsToUpdate     []igaQuestion  `json:"questions_to_update"`
	QuestionsToDeleteByID []string       `json:"questions_to_delete_by_id"`
}

// todo review with special focus on igaResponse schema translation into Gemini AI JSON schema
// todo write `iga.go.prmpt`, explaining the AI's role, the input schema, and the task
// todo implement HandleIGAMessage calling in SendMessage (and RetryMessage)
// todo evaluate implementing calling HandleIGAMessage with instruction type "all" on session creation

func (g *GeminiAIService) HandleIGAMessage(ctx context.Context, params HandleIGAMessageParams) (*HandleIGAMessageResult, error) {
	prompt, err := intoIgaPrompt(params)
	if err != nil {
		return nil, err
	}

	prompt.ModelInstruction, err = g.prompt.IgaPrompt(prmpt.IgaPromptParams{
		UserInstruction: prompt.UserInstruction,
	})
	if err != nil {
		return nil, fmt.Errorf("error constructing Gemini AI prompt: %w", err)
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseJsonSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"assistant_response_text": map[string]any{
					"type":        "string",
					"description": "The Requirements Engineering Expert's response for the chat conversation.",
				},
				"context_data": map[string]any{
					"type":        "object",
					"description": "The project's context data.",
					"properties": map[string]any{
						"stakeholder_summary": map[string]any{
							"type":        "string",
							"description": "Summary of the stakeholder who is interviewed, the INTERVIEWEE.",
						},
						"project_summary": map[string]any{
							"type":        "string",
							"description": "Brief summary of the project that will be discussed and explored in the interview.",
						},
						"interview_goal": map[string]any{
							"type":        "string",
							"description": "The goal of the requirements elicitation interview.",
						},
						"interview_advice": map[string]any{
							"type":        "string",
							"description": "General advice for executing the interview, for the INTERVIEWER.",
						},
					},
				},
				"questions_to_create": map[string]any{
					"type":        "array",
					"description": "Questions with notes to create for the INTERVIEWER to conduct the interview.",
					"maxItems":    20,
					"items": map[string]any{
						"type":        "object",
						"description": "Questionnaire item containing the question, a note concerning the question, the sort order and the readable ID.",
						"properties": map[string]any{
							"readable_id": map[string]any{
								"type":        "string",
								"description": "Readable ID in the format of: Q-[XXX]. Can not be changed! MUST BE MAINTAINED!",
							},
							"question_text": map[string]any{
								"type":        "string",
								"description": "The question's content. Must be a question that the INTERVIEWER asks the INTERVIEWEE.",
							},
							"note_text": map[string]any{
								"type":        "string",
								"description": "The note's content. Helpful advice for the INTERVIEWER.",
							},
							"sort_order": map[string]any{
								"type":        "integer",
								"description": "Sort order is 0-index, 1-stepped, contiguous, UI will sort ASCENDING.",
							},
						},
					},
				},
				"questions_to_update": map[string]any{
					"type":        "array",
					"description": "Questions with notes to update for the INTERVIEWER to conduct the interview.",
					"maxItems":    20,
					"items": map[string]any{
						"$ref": "#/properties/questions_to_create/items",
					},
				},
				"questions_to_delete_by_id": map[string]any{
					"type":        "array",
					"description": "Questions to delete from the questionnaire by their READABLE ID.",
					"items": map[string]any{
						"type":        "string",
						"description": "Readable ID of the question in the format Q-[XXX].",
					},
				},
			},
			"required": []string{"assistantResponseText"},
		},
	}

	promptJSON, err := json.Marshal(prompt)
	if err != nil {
		return nil, fmt.Errorf("error marshalling prompt to JSON for Gemini AI request: %w, with prompt: %+v", err, prompt)
	}

	// todo implement sane, logged, capped re-try, using generic function to re-try and re-parse JSON on previous failures
	result, err := g.client.Models.GenerateContent(
		ctx,
		g.model,
		genai.Text(string(promptJSON)),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("error generating content using Gemini AI: %w\nModel: %s\nPrompt JSON: %s\nConfig: %+v", err, g.model, promptJSON, config)
	}

	var igaRes igaResponse
	err = json.Unmarshal([]byte(result.Text()), &igaRes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling Gemini AI response text to struct: %w\nResult: %s\nModel: %s\nPrompt JSON: %s\nConfig: %+v",
			err, result.Text(), g.model, promptJSON, config)
	}

	if igaRes.AssistantResponseText == "" {
		return nil, fmt.Errorf("empty assistant response message from Gemini AI response not expected: %+v\nModel: %s\nPrompt JSON: %s\nConfig: %+v",
			igaRes, g.model, promptJSON, config)
	}
	// until here retry border

	return intoHandleIGAMessageResult(params, igaRes), nil
}

func intoIgaPrompt(params HandleIGAMessageParams) (*igaPrompt, error) {
	prompt := &igaPrompt{}

	prompt.ModelInstruction = "" // will be set later

	prompt.UserMessageText = params.UserMessage.MessageText

	prompt.UserInstruction = string(params.UserMessage.Metadata.Type)
	if _, err := types.IntoIGAChatLogInstructionType(prompt.UserInstruction); err != nil {
		return nil, fmt.Errorf("invalid user instruction type for Gemini AI prompt: %w", err)
	}

	prompt.ContextData = igaContextData{
		StakeholderSummary: params.Session.ContextData.StakeholderSummary,
		ProjectSummary:     params.Session.ContextData.ProjectSummary,
		InterviewGoal:      params.Session.ContextData.InterviewGoal,
		InterviewAdvice:    params.Session.ContextData.InterviewAdvice,
	}

	prompt.ChatHistory = make([]igaChat, len(params.Chat))
	for i, log := range params.Chat {
		prompt.ChatHistory[i] = igaChat{
			Role:        string(log.Role),
			MessageText: log.MessageText,
			Instruction: string(log.Metadata.Type),
		}
	}

	prompt.Questionnaire = make([]igaQuestion, len(params.Questionnaire))
	for i, qstn := range params.Questionnaire {
		prompt.Questionnaire[i] = igaQuestion{
			ReadableID:   qstn.ReadableID,
			QuestionText: qstn.QuestionText,
			NoteText:     qstn.NoteText,
			SortOrder:    qstn.SortOrder,
		}
	}

	return prompt, nil
}

func intoHandleIGAMessageResult(params HandleIGAMessageParams, llm igaResponse) *HandleIGAMessageResult {
	result := &HandleIGAMessageResult{}

	result.AssistantResponse = types.IGAChatLogToCreate{
		IGASessionID: params.Session.ID,
		Role:         string(types.IGAChatLogRoleAssistant),
		MessageText:  llm.AssistantResponseText,
	}

	refineContext := params.UserMessage.Metadata.Type == types.IGAChatLogInstructionRefineContext
	refineAll := params.UserMessage.Metadata.Type == types.IGAChatLogInstructionAll
	if refineContext || refineAll {
		fallbackContextData := params.Session.ContextData

		stakeholderSummary := fallbackContextData.StakeholderSummary
		if llm.ContextData.StakeholderSummary != "" {
			stakeholderSummary = llm.ContextData.StakeholderSummary
		}

		projectSummary := fallbackContextData.ProjectSummary
		if llm.ContextData.ProjectSummary != "" {
			projectSummary = llm.ContextData.ProjectSummary
		}

		interviewGoal := fallbackContextData.InterviewGoal
		if llm.ContextData.InterviewGoal != "" {
			interviewGoal = llm.ContextData.InterviewGoal
		}

		interviewAdvice := fallbackContextData.InterviewAdvice
		if llm.ContextData.InterviewAdvice != "" {
			interviewAdvice = llm.ContextData.InterviewAdvice
		}

		result.SessionContextData = &types.IGASessionContextData{
			StakeholderSummary: stakeholderSummary,
			ProjectSummary:     projectSummary,
			InterviewGoal:      interviewGoal,
			InterviewAdvice:    interviewAdvice,
		}
	}

	if len(llm.QuestionsToCreate) > 0 {
		result.QuestionsToCreate = make([]types.IGAScriptQuestionToCreate, len(llm.QuestionsToCreate))
		for i, qtc := range llm.QuestionsToCreate {
			result.QuestionsToCreate[i] = types.IGAScriptQuestionToCreate{
				IGASessionID: params.Session.ID,
				QuestionText: qtc.QuestionText,
				NoteText:     qtc.NoteText,
				SortOrder:    qtc.SortOrder,
			}
		}
	}

	originalQuestionsByReadableID := make(map[string]types.IGAScriptQuestion, len(params.Questionnaire))
	for _, q := range params.Questionnaire {
		originalQuestionsByReadableID[q.ReadableID] = q
	}

	if len(llm.QuestionsToUpdate) > 0 {
		result.QuestionsToUpdate = make(map[uuid.UUID]types.IGAScriptQuestionToUpdate, len(llm.QuestionsToUpdate))
		for _, qtu := range llm.QuestionsToUpdate {
			original, ok := originalQuestionsByReadableID[qtu.ReadableID]
			if !ok {
				continue // question to update does not exist in original questionnaire
			}

			toUpdate := types.IGAScriptQuestionToUpdate{}

			if qtu.QuestionText != original.QuestionText {
				toUpdate.QuestionText = &qtu.QuestionText
			}

			if qtu.NoteText != original.NoteText {
				toUpdate.NoteText = &qtu.NoteText
			}

			if qtu.SortOrder != original.SortOrder {
				toUpdate.SortOrder = &qtu.SortOrder
			}

			result.QuestionsToUpdate[original.ID] = toUpdate
		}
	}

	if len(llm.QuestionsToDeleteByID) > 0 {
		result.QuestionsToDelete = make([]uuid.UUID, 0, len(llm.QuestionsToDeleteByID))
		for _, qtd := range llm.QuestionsToDeleteByID {
			original, ok := originalQuestionsByReadableID[qtd]
			if !ok {
				continue // question to delete does not exist in original questionnaire
			}

			result.QuestionsToDelete = append(result.QuestionsToDelete, original.ID)
		}
	}

	return result
}
