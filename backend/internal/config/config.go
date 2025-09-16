package config

import "os"

type Config struct {
	MongoURI      string
	MongoDB       string
	GRPCAddr      string
	HTTPAddr      string // grpc-gateway REST
	SiteAddr      string // chi mux for /healthz and serving openapi json
	DefaultLocale string
	Environment   string // development, staging, production

	// Email configuration
	EmailProvider    string // sendgrid, mailhog, mock
	EmailFromName    string
	EmailFromAddress string
	EmailBaseURL     string

	// SendGrid specific
	SendGridAPIKey string

	// SMTP specific (for MailHog)
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPTLSMode  string

	// GDPR configuration
	GDPREncryptionKey string
}

func FromEnv() Config {
	return Config{
		MongoURI:      getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:       getenv("MONGO_DB", "pingis"),
		GRPCAddr:      getenv("GRPC_ADDR", ":9090"),
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		SiteAddr:      getenv("SITE_ADDR", ":8081"),
		DefaultLocale: getenv("DEFAULT_LOCALE", "sv"),
		Environment:   getenv("ENVIRONMENT", "development"),

		// Email configuration
		EmailProvider:    getenv("EMAIL_PROVIDER", "mailhog"), // Default to MailHog for development
		EmailFromName:    getenv("EMAIL_FROM_NAME", "Klubbspel"),
		EmailFromAddress: getenv("EMAIL_FROM_ADDRESS", "noreply@klubbspel.se"),
		EmailBaseURL:     getenv("EMAIL_BASE_URL", "http://localhost:5000"),

		// SendGrid
		SendGridAPIKey: getenv("SENDGRID_API_KEY", ""),

		// SMTP (MailHog defaults)
		SMTPHost:     getenv("SMTP_HOST", "localhost"),
		SMTPPort:     getenv("SMTP_PORT", "1025"),
		SMTPUsername: getenv("SMTP_USERNAME", ""),
		SMTPPassword: getenv("SMTP_PASSWORD", ""),
		SMTPTLSMode:  getenv("SMTP_TLS_MODE", "none"),

		// GDPR configuration
		GDPREncryptionKey: getenv("GDPR_ENCRYPTION_KEY", ""),
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
