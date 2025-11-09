package cfg

import (
	"os"
	"strings"
)

const (
	KeyDatabaseUrl       = "PG_DATABASE_URL"
	WebHost              = "WEB_HOST"
	WebPort              = "WEB_PORT"
	CORSAllowOrigin      = "CORS_ALLOW_ORIGIN"
	CORSAllowCredentials = "CORS_ALLOW_CREDENTIALS"
	CORSAllowMethods     = "CORS_ALLOW_METHODS"
	CORSAllowHeaders     = "CORS_ALLOW_HEADERS"

	GeminiAIServiceAPIKey = "GEMINI_AI_SERVICE_API_KEY"
	GeminiAIDefaultModel  = "GEMINI_AI_DEFAULT_MODEL"
	AITestMode            = "AI_TEST_MODE" // AITestMode evaluated with EvaluateBool
)

type GetEnvFunc func(string) string

type WebConfig struct {
	Host string
	Port string
}

func DefaultGetEnv(key string) string {
	if s, ok := os.LookupEnv(key); ok {
		return s
	}

	switch key {
	case KeyDatabaseUrl:
		return "user=user password=password host=db port=5432 dbname=cognireq"
	case WebHost:
		return ""
	case WebPort:
		return "8080"
	case CORSAllowOrigin:
		return "http://localhost:5173"
	case CORSAllowCredentials:
		return "true"
	case CORSAllowMethods:
		return "GET, POST, PATCH, DELETE, OPTIONS"
	case CORSAllowHeaders:
		return "Content-Type"
	case AITestMode:
		return "true"
	default:
		return ""
	}
}

func EvaluateBool(input string) bool {
	input = strings.ToUpper(input)

	switch input {
	case "", "FALSE", "F":
		return false
	default:
		return true
	}
}

func WebConfigFromEnv(getenv GetEnvFunc) *WebConfig {
	return &WebConfig{Host: getenv(WebHost), Port: getenv(WebPort)}
}
