package email

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

// EmailProvider represents different email service providers
type EmailProvider string

const (
	ProviderSendGrid EmailProvider = "sendgrid"
	ProviderMailHog  EmailProvider = "mailhog"
	ProviderMock     EmailProvider = "mock"
)

// EmailConfig holds configuration for email services
type EmailConfig struct {
	Provider  EmailProvider
	FromName  string
	FromEmail string
	BaseURL   string

	// SendGrid specific
	SendGridAPIKey string

	// MailHog/SMTP specific
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPTLSMode  string // none, tls, starttls
}

// EmailAdapter provides a unified interface for different email providers
type EmailAdapter struct {
	config  EmailConfig
	service Service
}

// NewEmailAdapter creates the appropriate email service based on configuration
func NewEmailAdapter(config EmailConfig) (*EmailAdapter, error) {
	log.Info().
		Str("provider", string(config.Provider)).
		Str("from_email", config.FromEmail).
		Msg("Creating email adapter")

	var service Service

	switch config.Provider {
	case "sendgrid":
		service = NewSendGridService(config.FromName, config.FromEmail, config.BaseURL)
	case "mailhog":
		mailhogService, err := NewMailHogService(config)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create MailHog service")
			return nil, fmt.Errorf("failed to create MailHog service: %w", err)
		}
		service = mailhogService
	case "mock":
		mockService, err := NewMockEmailService(config)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Mock service")
			return nil, fmt.Errorf("failed to create Mock service: %w", err)
		}
		service = mockService
	default:
		log.Error().
			Str("provider", string(config.Provider)).
			Msg("Unknown email provider")
		return nil, fmt.Errorf("unknown email provider: %s", config.Provider)
	}

	return &EmailAdapter{
		config:  config,
		service: service,
	}, nil
}

// SendMagicLink sends a magic link email
func (a *EmailAdapter) SendMagicLink(ctx context.Context, toEmail, token, returnURL string) error {
	return a.service.SendMagicLink(ctx, toEmail, token, returnURL)
}

// SendInvitation sends an invitation email
func (a *EmailAdapter) SendInvitation(ctx context.Context, toEmail, clubName, inviterName string) error {
	return a.service.SendInvitation(ctx, toEmail, clubName, inviterName)
}

// SendEmail sends a generic email
func (a *EmailAdapter) SendEmail(ctx context.Context, toEmail, subject, body string) error {
	return a.service.SendEmail(ctx, toEmail, subject, body)
}

// GetProvider returns the current email provider
func (a *EmailAdapter) GetProvider() EmailProvider {
	return a.config.Provider
}

// FromEnv creates an EmailConfig from environment variables
func FromEnv() EmailConfig {
	provider := strings.ToLower(getenv("EMAIL_PROVIDER", "mock"))

	config := EmailConfig{
		Provider:  EmailProvider(provider),
		FromName:  getenv("EMAIL_FROM_NAME", "Klubbspel"),
		FromEmail: getenv("EMAIL_FROM_ADDRESS", "noreply@klubbspel.se"),
		BaseURL:   getenv("EMAIL_BASE_URL", "http://localhost:5000"),

		// SendGrid
		SendGridAPIKey: os.Getenv("SENDGRID_API_KEY"),

		// SMTP/MailHog
		SMTPHost:     getenv("SMTP_HOST", "localhost"),
		SMTPPort:     parseInt(getenv("SMTP_PORT", "1025")),
		SMTPUsername: getenv("SMTP_USERNAME", ""),
		SMTPPassword: getenv("SMTP_PASSWORD", ""),
		SMTPTLSMode:  getenv("SMTP_TLS_MODE", "none"),
	}

	return config
}

// Helper function to get environment variable with default
func getenv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to parse integer from environment variable
func parseInt(s string) int {
	if s == "" {
		return 0
	}

	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
