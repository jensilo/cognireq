package cfg

import "os"

var (
	KeyDatabaseUrl = "PG_DATABASE_URL"
	WebHost        = "WEB_HOST"
	WebPort        = "WEB_PORT"
)

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
		return "user=user password=password host=db port=5432 dbname=harmony"
	case WebHost:
		return ""
	case WebPort:
		return "8080"
	default:
		return ""
	}
}

func WebConfigFromEnv(getenv func(string) string) *WebConfig {
	return &WebConfig{Host: getenv(WebHost), Port: getenv(WebPort)}
}
