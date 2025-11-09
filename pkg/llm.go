package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jensilo/cognireq/pkg/cfg"
	"github.com/jensilo/cognireq/pkg/prmpt"
	"github.com/jensilo/cognireq/pkg/types"
	"google.golang.org/genai"
)

type IGAMessageHandler interface {
	HandleIGAMessage(context.Context, HandleIGAMessageParams) (*HandleIGAMessageResult, error)
}

type GeminiAIService struct {
	client   *genai.Client
	model    string
	prompt   *prmpt.PromptService
	logger   *slog.Logger
	testMode bool
}

func NewGeminiAIService(ctx context.Context, envFunc cfg.GetEnvFunc, ps *prmpt.PromptService, l *slog.Logger) (*GeminiAIService, error) {
	if cfg.EvaluateBool(envFunc(cfg.AITestMode)) {
		return &GeminiAIService{client: nil, model: "test-1", prompt: ps, logger: l, testMode: true}, nil
	}

	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  envFunc(cfg.GeminiAIServiceAPIKey),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("could not instantiate gemini genai client: %w", err)
	}

	m := envFunc(cfg.GeminiAIDefaultModel)
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

// todo evaluate implementing calling HandleIGAMessage with instruction type "all" on session creation

func (g *GeminiAIService) HandleIGAMessage(ctx context.Context, params HandleIGAMessageParams) (*HandleIGAMessageResult, error) {
	var client geminiContentGenerator
	if g.testMode {
		client = testAIClient{logger: g.logger}
	} else {
		client = g.client.Models
	}

	return g.geminiHandleIGAMessage(ctx, client, params)
}

type geminiContentGenerator interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

func (g *GeminiAIService) geminiHandleIGAMessage(ctx context.Context, client geminiContentGenerator, params HandleIGAMessageParams) (*HandleIGAMessageResult, error) {
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
					"description": "AI's chat response.",
				},
				"context_data": map[string]any{
					"type":        "object",
					"description": "Project context data.",
					"properties": map[string]any{
						"stakeholder_summary": map[string]any{
							"type":        "string",
							"description": "Summary of interviewee.",
						},
						"project_summary": map[string]any{
							"type":        "string",
							"description": "Brief project summary.",
						},
						"interview_goal": map[string]any{
							"type":        "string",
							"description": "Goal of interview.",
						},
						"interview_advice": map[string]any{
							"type":        "string",
							"description": "Advice for interviewer.",
						},
					},
				},
				"questions_to_create": map[string]any{
					"type":        "array",
					"description": "Questions to create.",
					"items": map[string]any{
						"type":        "object",
						"description": "Questionnaire item.",
						"properties": map[string]any{
							"readable_id": map[string]any{
								"type":        "string",
								"description": "Q-[XXX] ID. MUST BE MAINTAINED!",
							},
							"question_text": map[string]any{
								"type":        "string",
								"description": "Question content.",
							},
							"note_text": map[string]any{
								"type":        "string",
								"description": "Note content.",
							},
							"sort_order": map[string]any{
								"type":        "integer",
								"description": "0-indexed, 1-stepped, contiguous sort order.",
							},
						},
					},
				},
				"questions_to_update": map[string]any{
					"type":        "array",
					"description": "Questions to update.",
					"items": map[string]any{
						"$ref": "#/properties/questions_to_create/items",
					},
				},
				"questions_to_delete_by_id": map[string]any{
					"type":        "array",
					"description": "Questions to delete by readable ID.",
					"items": map[string]any{
						"type":        "string",
						"description": "Readable ID (Q-[XXX]).",
					},
				},
			},
			"required": []string{"assistant_response_text"},
		},
	}

	promptJSON, err := json.Marshal(prompt)
	if err != nil {
		return nil, fmt.Errorf("error marshalling prompt to JSON for Gemini AI request: %w, with prompt: %+v", err, prompt)
	}

	logger := g.logger.With(
		"function-trace", "llm.go/intoHandleIGAMessageResult",
		"params", params,
		"model", g.model,
		"prompt", prompt,
		"config", config,
	)

	// todo implement sane, logged, capped re-try, using generic function to re-try and re-parse JSON on previous failures
	result, err := client.GenerateContent(
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

	logger.Debug("gemini AI responded", "iga-response", igaRes)

	return intoHandleIGAMessageResult(logger, params, igaRes), nil
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

func intoHandleIGAMessageResult(logger *slog.Logger, params HandleIGAMessageParams, llm igaResponse) *HandleIGAMessageResult {
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
				logger.Info("skipping LLM question to update bc not found in original", "question-to-update", qtu)
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
				logger.Info("skipping LLM question to delete bc not found in original", "question-to-delete", qtd)
				continue // question to delete does not exist in original questionnaire
			}

			result.QuestionsToDelete = append(result.QuestionsToDelete, original.ID)
		}
	}

	return result
}

type testAIClient struct {
	logger *slog.Logger
}

func (c testAIClient) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	var message string
	var res igaResponse

	if len(contents) < 1 || len(contents[0].Parts) < 1 || contents[0].Parts[0].Text == "" {
		return nil, fmt.Errorf("test AI client request to GenerateContent: non-empty contents expected")
	}
	var input igaPrompt
	err := json.Unmarshal([]byte(contents[0].Parts[0].Text), &input)
	if err != nil {
		return nil, fmt.Errorf("test AI client can not unmarshal into iga prompt: %w", err)
	}
	lastMessage := input.UserMessageText
	commands := parseCommands(lastMessage)

	if commands.questionsToCreate != nil {
		message = fmt.Sprintf("[read command: question create, param amount: %d]\n", commands.questionsToCreate.amount)
		igaQuestions := make([]igaQuestion, commands.questionsToCreate.amount)
		for i := range commands.questionsToCreate.amount {
			str := uuid.NewString()
			igaQuestions[i] = igaQuestion{
				QuestionText: fmt.Sprintf("Q no. %d: %s", i, str),
				NoteText:     fmt.Sprintf("N no. %d: %s", i, str),
				SortOrder:    int16(i),
			}
		}
		res.QuestionsToCreate = igaQuestions
	}

	if commands.questionsToUpdate != nil {
		message = fmt.Sprintf("%s[read command: question update, param readable ids: %s]\n", message, strings.Join(commands.questionsToUpdate.readableIds, " "))
		igaQuestions := make([]igaQuestion, len(commands.questionsToUpdate.readableIds))
		for i, id := range commands.questionsToUpdate.readableIds {
			str := uuid.NewString()
			igaQuestions[i] = igaQuestion{
				ReadableID:   id,
				QuestionText: fmt.Sprintf("Q (for %s) updated no. %d: %s", id, i, str),
				NoteText:     fmt.Sprintf("N (for %s) updated no. %d: %s", id, i, str),
				SortOrder:    int16(i),
			}
		}
		res.QuestionsToUpdate = igaQuestions
	}

	if commands.questionsToDelete != nil {
		message = fmt.Sprintf("%s[read command: question delete, param readable ids: %s]\n", message, strings.Join(commands.questionsToDelete.readableIds, " "))
		res.QuestionsToDeleteByID = commands.questionsToDelete.readableIds
	}

	if commands.contextRefine != nil {
		message = fmt.Sprintf("%s[read command: context refine, param fields: %s]\n", message, strings.Join(commands.contextRefine.fields, " "))
		for i, field := range commands.contextRefine.fields {
			switch field {
			case "project_summary":
				res.ContextData.ProjectSummary = fmt.Sprintf("refined by mock LLM: %s (%d)", uuid.NewString(), i)
			case "stakeholder_summary":
				res.ContextData.StakeholderSummary = fmt.Sprintf("refined by mock LLM: %s (%d)", uuid.NewString(), i)
			case "interview_goal":
				res.ContextData.InterviewGoal = fmt.Sprintf("refined by mock LLM: %s (%d)", uuid.NewString(), i)
			case "interview_advice":
				res.ContextData.InterviewAdvice = fmt.Sprintf("refined by mock LLM: %s (%d)", uuid.NewString(), i)
			default:
				message = fmt.Sprintf("%s[could not match context refine field: %s (%d)]\n", message, field, i)
			}
		}
	}

	if message == "" {
		message = fmt.Sprintf("[no commands read]\n")
	}

	res.AssistantResponseText = fmt.Sprintf("%s\nThis is a stubbed response from the mock LLM.", message)
	respJSON, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("test ai client failed to parse iga response: %w", err)
	}

	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{
							Text: string(respJSON),
						},
					},
				},
			},
		},
	}, nil
}

type questionCreateCommand struct {
	amount int
}

type questionUpdateCommand struct {
	readableIds []string
}

type questionDeleteCommand struct {
	readableIds []string
}

type contextRefineCommand struct {
	// fields possible values are "project_summary", "stakeholder_summary", "interview_goal", "interview_advice"
	fields []string
}

type generateContentCommand struct {
	questionsToCreate *questionCreateCommand
	questionsToUpdate *questionUpdateCommand
	questionsToDelete *questionDeleteCommand
	contextRefine     *contextRefineCommand
}

// Use four separate, focused regular expressions.
// The `(?s)` flag enables dotall mode so '.' matches newlines, allowing commands to span lines.
var (
	// Matches [/question create X] and captures X (the amount).
	createRegexp = regexp.MustCompile(`(?s)\[/question\s+create\s+(\d+)\]`)

	// Matches [/question update Q-ABC Q-CDE ...] and captures all IDs.
	// Requires at least one ID (Q-[A-Z]{3}) and captures the space-separated list.
	updateRegexp = regexp.MustCompile(`(?s)\[/question\s+update\s+((?:Q-[A-Z]{3}(?:\s+Q-[A-Z]{3})*))\]`)

	// Matches [/question delete Q-CDE Q-ABC ...] and captures all IDs.
	// Requires at least one ID and captures the space-separated list.
	deleteRegexp = regexp.MustCompile(`(?s)\[/question\s+delete\s+((?:Q-[A-Z]{3}(?:\s+Q-[A-Z]{3})*))\]`)

	// Matches [/context refine field1 field2 ...] and captures all fields.
	// Matches one to four valid field names, space-separated.
	refineRegexp = regexp.MustCompile(`(?s)\[/context\s+refine\s+((?:project_summary|stakeholder_summary|interview_goal|interview_advice)(?:\s+(?:project_summary|stakeholder_summary|interview_goal|interview_advice)){0,3})\]`)
)

// parseCommands extracts the first occurrence of each command type from an input string.
func parseCommands(input string) generateContentCommand {
	result := generateContentCommand{}

	// 'question create'
	if match := createRegexp.FindStringSubmatch(input); len(match) > 1 {
		if amount, err := strconv.Atoi(match[1]); err == nil {
			result.questionsToCreate = &questionCreateCommand{amount: amount}
		}
	}

	// 'question update'
	if match := updateRegexp.FindStringSubmatch(input); len(match) > 1 {
		idsStr := match[1]
		ids := strings.Fields(idsStr)
		if len(ids) > 0 {
			result.questionsToUpdate = &questionUpdateCommand{readableIds: ids}
		}
	}

	// 'question delete'
	if match := deleteRegexp.FindStringSubmatch(input); len(match) > 1 {
		idsStr := match[1]
		ids := strings.Fields(idsStr)
		if len(ids) > 0 {
			result.questionsToDelete = &questionDeleteCommand{readableIds: ids}
		}
	}

	// 'context refine'
	if match := refineRegexp.FindStringSubmatch(input); len(match) > 1 {
		fieldsStr := match[1]
		fields := strings.Fields(fieldsStr)
		if len(fields) > 0 {
			result.contextRefine = &contextRefineCommand{fields: fields}
		}
	}

	return result
}
