package cfg

import "os"

var (
	KeyDatabaseUrl       = "PG_DATABASE_URL"
	WebHost              = "WEB_HOST"
	WebPort              = "WEB_PORT"
	CORSAllowOrigin      = "CORS_ALLOW_ORIGIN"
	CORSAllowCredentials = "CORS_ALLOW_CREDENTIALS"
	CORSAllowMethods     = "CORS_ALLOW_METHODS"
	CORSAllowHeaders     = "CORS_ALLOW_HEADERS"
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
	default:
		return ""
	}
}

func WebConfigFromEnv(getenv GetEnvFunc) *WebConfig {
	return &WebConfig{Host: getenv(WebHost), Port: getenv(WebPort)}
}
