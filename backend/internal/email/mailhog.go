package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
)

// MailHogService sends emails using SMTP (primarily for MailHog testing)
type MailHogService struct {
	config EmailConfig
}

// NewMailHogService creates a new MailHog SMTP service instance
func NewMailHogService(config EmailConfig) (*MailHogService, error) {
	return &MailHogService{
		config: config,
	}, nil
}

// SendMagicLink sends a magic link email via SMTP
func (s *MailHogService) SendMagicLink(ctx context.Context, toEmail, token, returnURL string) error {
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

	// Plain text version for SMTP (Swedish)
	body := fmt.Sprintf(`Hej!

Klicka på länken nedan för att logga in på Klubbspel:

%s

Denna länk går ut om 15 minuter av säkerhetsskäl.

Om du inte begärt denna inloggning kan du bortse från detta mejl.

Med vänliga hälsningar,
Klubbspel-teamet`, magicURL)

	return s.SendEmail(ctx, toEmail, subject, body)
}

// SendInvitation sends an invitation email via SMTP
func (s *MailHogService) SendInvitation(ctx context.Context, toEmail, clubName, inviterName string) error {
	subject := fmt.Sprintf("Invitation to join %s on Klubbspel", clubName)

	body := fmt.Sprintf(`Hi there!

%s has invited you to join the club "%s" on Klubbspel, the table tennis tournament management platform.

To accept this invitation and create your account, click the link below:

%s/join?club=%s

About Klubbspel:
Klubbspel helps table tennis clubs organize tournaments, track player rankings, and manage club memberships. Join to participate in tournaments and track your progress!

If you have any questions, feel free to contact %s or visit our help center.

Best regards,
The Klubbspel Team`, inviterName, clubName, s.config.BaseURL, clubName, inviterName)

	return s.SendEmail(ctx, toEmail, subject, body)
}

// SendEmail sends a generic email via SMTP
func (s *MailHogService) SendEmail(ctx context.Context, toEmail, subject, body string) error {
	// For development/testing, if no SMTP is configured, just log
	if s.config.SMTPHost == "" || s.config.SMTPPort == 0 {
		fmt.Printf("[EMAIL] To: %s\nSubject: %s\nBody:\n%s\n\n", toEmail, subject, body)
		return nil
	}

	// Create the message
	msg := s.buildSMTPMessage(toEmail, subject, body)

	// Determine auth
	var auth smtp.Auth
	if s.config.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	// Address
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Send based on TLS mode
	switch s.config.SMTPTLSMode {
	case "tls":
		return s.sendWithTLS(addr, auth, []string{toEmail}, msg)
	case "starttls":
		return s.sendWithStartTLS(addr, auth, []string{toEmail}, msg)
	default:
		// Plain SMTP (for MailHog)
		return smtp.SendMail(addr, auth, s.config.FromEmail, []string{toEmail}, msg)
	}
}

// buildSMTPMessage builds a properly formatted SMTP message
func (s *MailHogService) buildSMTPMessage(toEmail, subject, body string) []byte {
	message := fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromEmail)
	message += fmt.Sprintf("To: %s\r\n", toEmail)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += body

	return []byte(message)
}

// sendWithTLS sends email over TLS connection
func (s *MailHogService) sendWithTLS(addr string, auth smtp.Auth, to []string, msg []byte) error {
	// Create TLS connection
	tlsConfig := &tls.Config{
		ServerName:         s.config.SMTPHost,
		InsecureSkipVerify: false,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to establish TLS connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			// Log but don't return error as it's in cleanup
		}
	}()

	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() {
		if err := client.Quit(); err != nil {
			// Log but don't return error as it's in cleanup
		}
	}()

	// Authenticate if credentials provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send DATA command: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return w.Close()
}

// sendWithStartTLS sends email using STARTTLS
func (s *MailHogService) sendWithStartTLS(addr string, auth smtp.Auth, to []string, msg []byte) error {
	// Connect to server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() {
		if err := client.Quit(); err != nil {
			// Log but don't return error as it's in cleanup
		}
	}()

	// Start TLS if supported
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName:         s.config.SMTPHost,
			InsecureSkipVerify: false,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate if credentials provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send DATA command: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return w.Close()
}
