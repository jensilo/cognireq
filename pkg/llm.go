package pkg

import (
	"context"
	"fmt"

	"github.com/jensilo/cognireq/pkg/cfg"
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
}

func NewGeminiAIService(ctx context.Context, envFunc cfg.GetEnvFunc) (*GeminiAIService, error) {
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

	return &GeminiAIService{client: c, model: m}, nil
}

type HandleIGAMessageParams struct {
	userMessage   types.IGAChatLog
	session       types.IGASession
	chat          []types.IGAChatLog
	questionnaire []types.IGAScriptQuestion
}

type HandleIGAMessageResult struct {
	assistantResponse types.IGAChatLog
	session           types.IGASession
	questionnaire     []types.IGAScriptQuestion
}

type igaContextData struct {
	StakeholderSummary string `json:"stakeholder_summary"`
	ProjectSummary     string `json:"project_summary"`
	InterviewGoal      string `json:"interview_goal"`
	InterviewAdvice    string `json:"interview_advice"`
}

type igaChatHistory struct {
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
	ChatHistory      igaChatHistory `json:"chat_history"`
	Questionnaire    []igaQuestion  `json:"questionnaire"`
}

type igaResponse struct {
	AssistantResponseText string         `json:"assistant_response_text"`
	ContextData           igaContextData `json:"context_data"`
	Questionnaire         []igaQuestion  `json:"questionnaire"`
}

func (*GeminiAIService) HandleIGAMessage(ctx context.Context, params HandleIGAMessageParams) {
	config := genai.GenerateContentConfig{
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
				"questionnaire": map[string]any{
					"type":        "array",
					"description": "Questionnaire with questions and notes for the INTERVIEWER to conduct the interview.",
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
			},
			"required": []string{"assistantResponseText"},
		},
	}

	fmt.Println(config)
}
