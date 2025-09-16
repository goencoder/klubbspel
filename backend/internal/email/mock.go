package email

import (
	"context"
	"fmt"
	"log"
	"time"
)

// MockEmailService provides a mock implementation for testing
type MockEmailService struct {
	config     EmailConfig
	sentEmails []MockEmail
}

// MockEmail represents an email that was "sent" via the mock service
type MockEmail struct {
	To        string
	Subject   string
	Body      string
	Timestamp time.Time
	Type      string // "magic_link" or "invitation"
}

// NewMockEmailService creates a new mock email service instance
func NewMockEmailService(config EmailConfig) (*MockEmailService, error) {
	return &MockEmailService{
		config:     config,
		sentEmails: make([]MockEmail, 0),
	}, nil
}

// SendMagicLink logs a magic link email (mock implementation)
func (s *MockEmailService) SendMagicLink(ctx context.Context, toEmail, token, returnURL string) error {
	// Default to root page if returnURL is /login or empty
	if returnURL == "" || returnURL == "/login" {
		returnURL = "/"
	}

	// Build magic link URL
	magicURL := fmt.Sprintf("%s/auth/login?apikey=%s", s.config.BaseURL, token)
	if returnURL != "" {
		magicURL += fmt.Sprintf("&return_url=%s", returnURL)
	}

	subject := "Logga in på Klubbspel"
	body := fmt.Sprintf(`Hej!

Klicka på länken nedan för att logga in på Klubbspel:

%s

Denna länk går ut om 15 minuter av säkerhetsskäl.

Om du inte begärt denna inloggning kan du bortse från detta mejl.

Med vänliga hälsningar,
Klubbspel-teamet`, magicURL)

	// Store the mock email
	mockEmail := MockEmail{
		To:        toEmail,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now(),
		Type:      "magic_link",
	}
	s.sentEmails = append(s.sentEmails, mockEmail)

	// Log for debugging
	log.Printf("[MOCK EMAIL] Magic Link sent to %s: %s", toEmail, magicURL)

	return nil
}

// SendInvitation logs an invitation email (mock implementation)
func (s *MockEmailService) SendInvitation(ctx context.Context, toEmail, clubName, inviterName string) error {
	subject := fmt.Sprintf("Invitation att gå med i %s på Klubbspel", clubName)
	body := fmt.Sprintf(`Hej!

%s har bjudit in dig att gå med i klubben "%s" på Klubbspel, plattformen för hantering av bordtennisturneringar.

För att acceptera denna inbjudan och skapa ditt konto, klicka på länken nedan:

%s/join?club=%s

Om Klubbspel:
Klubbspel hjälper bordtennisklubbar att organisera turneringar, följa spelarranking och hantera medlemskap i klubben. Gå med för att delta i turneringar och följa din utveckling!

Om du har några frågor, tveka inte att kontakta %s eller besök vårt hjälpcenter.

Med vänliga hälsningar,
Klubbspel-teamet`, inviterName, clubName, s.config.BaseURL, clubName, inviterName)

	// Store the mock email
	mockEmail := MockEmail{
		To:        toEmail,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now(),
		Type:      "invitation",
	}
	s.sentEmails = append(s.sentEmails, mockEmail)

	// Log for debugging
	log.Printf("[MOCK EMAIL] Invitation sent to %s for club %s", toEmail, clubName)

	return nil
}

// SendEmail logs a generic email (mock implementation)
func (s *MockEmailService) SendEmail(ctx context.Context, toEmail, subject, body string) error {
	// Store the mock email
	mockEmail := MockEmail{
		To:        toEmail,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now(),
		Type:      "generic",
	}
	s.sentEmails = append(s.sentEmails, mockEmail)

	// Log for debugging
	log.Printf("[MOCK EMAIL] Generic email sent to %s: %s", toEmail, subject)

	return nil
}

// GetSentEmails returns all emails that were "sent" via this mock service
func (s *MockEmailService) GetSentEmails() []MockEmail {
	return s.sentEmails
}

// GetLastEmail returns the most recently sent email, or nil if none
func (s *MockEmailService) GetLastEmail() *MockEmail {
	if len(s.sentEmails) == 0 {
		return nil
	}
	return &s.sentEmails[len(s.sentEmails)-1]
}

// GetEmailsForRecipient returns all emails sent to a specific recipient
func (s *MockEmailService) GetEmailsForRecipient(email string) []MockEmail {
	var result []MockEmail
	for _, sent := range s.sentEmails {
		if sent.To == email {
			result = append(result, sent)
		}
	}
	return result
}

// ClearSentEmails clears the sent emails list (useful for tests)
func (s *MockEmailService) ClearSentEmails() {
	s.sentEmails = make([]MockEmail, 0)
}

// CountSentEmails returns the total number of emails sent
func (s *MockEmailService) CountSentEmails() int {
	return len(s.sentEmails)
}
