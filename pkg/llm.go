package pkg

import (
	"context"
	"fmt"

	"github.com/jensilo/cognireq/pkg/cfg"
	"google.golang.org/genai"
)

const (
	GeminiAIServiceAPIKey = "GEMINI_AI_SERVICE_API_KEY"
)

type GeminiAIService struct {
	client *genai.Client
}

func NewGeminiAIService(ctx context.Context, envFunc cfg.GetEnvFunc) (*GeminiAIService, error) {
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  envFunc(GeminiAIServiceAPIKey),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("could not instantiate gemini genai client: %w", err)
	}

	return &GeminiAIService{client: c}, nil
}
