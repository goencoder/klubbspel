package email

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Service interface for sending emails
type Service interface {
	SendMagicLink(ctx context.Context, toEmail, token, returnURL string) error
	SendClubInvitationMagicLink(ctx context.Context, toEmail, token, returnURL, clubName, inviterName, inviterEmail string) error
	SendInvitation(ctx context.Context, toEmail, clubName, inviterName string) error
	SendEmail(ctx context.Context, toEmail, subject, body string) error
}

// SendGridService sends emails using the SendGrid API
type SendGridService struct {
	client  *sendgrid.Client
	from    *mail.Email
	baseURL string
}

// NewSendGridService creates a new SendGridService instance
func NewSendGridService(fromName, fromEmail, baseURL string) *SendGridService {
	apiKey := os.Getenv("SENDGRID_API_KEY")

	log.Info().
		Str("from_name", fromName).
		Str("from_email", fromEmail).
		Str("base_url", baseURL).
		Bool("api_key_present", apiKey != "").
		Msg("Initializing SendGrid service")

	if apiKey == "" {
		log.Warn().Msg("SENDGRID_API_KEY is not set - emails will be logged to console only")
		return &SendGridService{
			client:  nil,
			from:    mail.NewEmail(fromName, fromEmail),
			baseURL: baseURL,
		}
	}

	sg := sendgrid.NewSendClient(apiKey)
	from := mail.NewEmail(fromName, fromEmail)

	return &SendGridService{
		client:  sg,
		from:    from,
		baseURL: baseURL,
	}
}

// SendMagicLink sends a magic link email for authentication
func (s *SendGridService) SendMagicLink(ctx context.Context, toEmail, token, returnURL string) error {
	log.Info().
		Str("to_email", toEmail).
		Msg("Sending magic link email")

	to := mail.NewEmail("", toEmail)

	// Default to leaderboard page if returnURL is /login or empty
	if returnURL == "" || returnURL == "/login" {
		returnURL = "/leaderboard"
	}

	// Build magic link URL
	magicURL := fmt.Sprintf("%s/auth/login?apikey=%s", s.baseURL, token)
	if returnURL != "" {
		magicURL += fmt.Sprintf("&return_url=%s", returnURL)
	}

	subject := "Logga in på Klubbspel"

	// Plain text version (Swedish)
	plainText := fmt.Sprintf(`
Hej!

Klicka på länken nedan för att logga in på Klubbspel:

%s

Denna länk går ut om 15 minuter av säkerhetsskäl.

Om du inte begärt denna inloggning kan du bortse från detta mejl.

Med vänliga hälsningar,
Klubbspel-teamet
`, magicURL)

	// HTML version (Swedish)
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Logga in på Klubbspel</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 8px;">
        <h1 style="color: #2c3e50; margin-bottom: 30px;">🏓 Logga in på Klubbspel</h1>
        
        <p style="font-size: 16px; margin-bottom: 25px;">
            Hej!
        </p>
        
        <p style="font-size: 16px; margin-bottom: 25px;">
            Klicka på knappen nedan för att logga in på ditt Klubbspel-konto:
        </p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" 
               style="background-color: #3498db; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">
                Logga in på Klubbspel
            </a>
        </div>
        
        <p style="font-size: 14px; color: #666; margin-bottom: 20px;">
            Eller kopiera och klistra in denna länk i din webbläsare:
        </p>
        
        <p style="font-size: 14px; color: #666; word-break: break-all; background-color: #e9ecef; padding: 10px; border-radius: 4px;">
            %s
        </p>
        
        <div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #dee2e6;">
            <p style="font-size: 14px; color: #666; margin-bottom: 10px;">
                <strong>Säkerhetsmeddelande:</strong> Denna länk går ut om 15 minuter.
            </p>
            
            <p style="font-size: 14px; color: #666;">
                Om du inte begärt denna inloggning kan du bortse från detta mejl.
            </p>
        </div>
        
        <div style="margin-top: 30px; text-align: center;">
            <p style="font-size: 14px; color: #666;">
                Med vänliga hälsningar,<br>
                Klubbspel-teamet
            </p>
        </div>
    </div>
</body>
</html>
`, magicURL, magicURL)

	if s.client == nil {
		log.Warn().
			Str("to_email", toEmail).
			Msg("Development mode: Email logged to console instead of sending")

		// Development mode - log instead of sending
		fmt.Printf("🔗 MAGIC LINK EMAIL (Development Mode)\n")
		fmt.Printf("To: %s\n", toEmail)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Magic Link: %s\n", magicURL)
		fmt.Printf("---\n")
		return nil
	}

	msg := mail.NewSingleEmail(s.from, subject, to, plainText, htmlContent)

	response, err := s.client.SendWithContext(ctx, msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("to_email", toEmail).
			Msg("SendGrid email sending failed")
		return fmt.Errorf("failed to send magic link email: %w", err)
	}

	// Check for SendGrid API errors (e.g., 403 for unverified sender)
	if response.StatusCode >= 400 {
		log.Error().
			Str("to_email", toEmail).
			Int("status_code", response.StatusCode).
			Str("response_body", response.Body).
			Msg("SendGrid returned error status")
		return fmt.Errorf("email delivery failed: SendGrid returned status %d: %s", response.StatusCode, response.Body)
	}

	log.Info().
		Str("to_email", toEmail).
		Int("status_code", response.StatusCode).
		Msg("Email sent successfully")

	return nil
}

// SendClubInvitationMagicLink sends a club invitation email with magic link
func (s *SendGridService) SendClubInvitationMagicLink(ctx context.Context, toEmail, token, returnURL, clubName, inviterName, inviterEmail string) error {
	log.Info().
		Str("to_email", toEmail).
		Str("club_name", clubName).
		Str("inviter_name", inviterName).
		Msg("Sending club invitation magic link email")

	to := mail.NewEmail("", toEmail)

	// Default to root page if returnURL is /login or empty
	if returnURL == "" || returnURL == "/login" {
		returnURL = "/"
	}

	// Build magic link URL
	magicURL := fmt.Sprintf("%s/auth/login?apikey=%s", s.baseURL, token)
	if returnURL != "" {
		magicURL += fmt.Sprintf("&return_url=%s", returnURL)
	}

	subject := fmt.Sprintf("Du har blivit tillagd som medlem i %s", clubName)

	// Plain text version (Swedish)
	plainText := fmt.Sprintf(`
Hej!

%s (%s) har lagt till dig som medlem i klubben "%s" på Klubbspel.

Klicka på länken nedan för att logga in och se klubbdetaljer:

%s

Denna länk går ut om 24 timmar av säkerhetsskäl.

Om du har frågor kan du kontakta %s på %s.

Med vänliga hälsningar,
Klubbspel-teamet
`, inviterName, inviterEmail, clubName, magicURL, inviterName, inviterEmail)

	// HTML version (Swedish)
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Välkommen till %s</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 8px;">
        <h1 style="color: #2c3e50; margin-bottom: 30px;">🏓 Välkommen till %s!</h1>
        
        <p style="font-size: 16px; margin-bottom: 25px;">
            Hej!
        </p>
        
        <p style="font-size: 16px; margin-bottom: 25px;">
            <strong>%s</strong> (%s) har lagt till dig som medlem i klubben <strong>"%s"</strong> på Klubbspel.
        </p>
        
        <p style="font-size: 16px; margin-bottom: 25px;">
            Klicka på knappen nedan för att logga in och se klubbdetaljer:
        </p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" 
               style="background-color: #27ae60; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">
                Se klubbdetaljer
            </a>
        </div>
        
        <p style="font-size: 14px; color: #666; margin-bottom: 20px;">
            Eller kopiera och klistra in denna länk i din webbläsare:
        </p>
        
        <p style="font-size: 14px; color: #666; word-break: break-all; background-color: #e9ecef; padding: 10px; border-radius: 4px;">
            %s
        </p>
        
        <div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #dee2e6;">
            <p style="font-size: 14px; color: #666; margin-bottom: 10px;">
                <strong>Säkerhetsmeddelande:</strong> Denna länk går ut om 24 timmar.
            </p>
            
            <p style="font-size: 14px; color: #666; margin-bottom: 15px;">
                Om du har frågor kan du kontakta <strong>%s</strong> på <a href="mailto:%s" style="color: #3498db;">%s</a>.
            </p>
        </div>
        
        <div style="margin-top: 30px; text-align: center;">
            <p style="font-size: 14px; color: #666;">
                Med vänliga hälsningar,<br>
                Klubbspel-teamet
            </p>
        </div>
    </div>
</body>
</html>
`, clubName, clubName, inviterName, inviterEmail, clubName, magicURL, magicURL, inviterName, inviterEmail, inviterEmail)

	if s.client == nil {
		log.Warn().
			Str("to_email", toEmail).
			Str("club_name", clubName).
			Msg("Development mode: Club invitation email logged to console instead of sending")

		// Development mode - log instead of sending
		fmt.Printf("🎾 CLUB INVITATION EMAIL (Development Mode)\n")
		fmt.Printf("To: %s\n", toEmail)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Club: %s\n", clubName)
		fmt.Printf("Inviter: %s (%s)\n", inviterName, inviterEmail)
		fmt.Printf("Magic Link: %s\n", magicURL)
		fmt.Printf("---\n")
		return nil
	}

	msg := mail.NewSingleEmail(s.from, subject, to, plainText, htmlContent)

	response, err := s.client.SendWithContext(ctx, msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("to_email", toEmail).
			Str("club_name", clubName).
			Msg("SendGrid club invitation email sending failed")
		return fmt.Errorf("failed to send club invitation email: %w", err)
	}

	// Check for SendGrid API errors
	if response.StatusCode >= 400 {
		log.Error().
			Str("to_email", toEmail).
			Str("club_name", clubName).
			Int("status_code", response.StatusCode).
			Str("response_body", response.Body).
			Msg("SendGrid returned error status for club invitation")
		return fmt.Errorf("club invitation email delivery failed: SendGrid returned status %d: %s", response.StatusCode, response.Body)
	}

	log.Info().
		Str("to_email", toEmail).
		Str("club_name", clubName).
		Int("status_code", response.StatusCode).
		Msg("Club invitation email sent successfully")

	return nil
}

// SendInvitation sends a club invitation email
func (s *SendGridService) SendInvitation(ctx context.Context, toEmail, clubName, inviterName string) error {
	to := mail.NewEmail("", toEmail)

	subject := fmt.Sprintf("You've been invited to join %s on Klubbspel", clubName)

	// Build invitation URL
	inviteURL := fmt.Sprintf("%s/invite?club=%s", s.baseURL, clubName)

	// Plain text version
	plainText := fmt.Sprintf(`
Hi there!

%s has invited you to join the "%s" club on Klubbspel.

Click the link below to get started:

%s

If you already have an account, you'll be able to join the club after signing in.
If you don't have an account yet, you can create one using this email address.

Best regards,
The Klubbspel Team
`, inviterName, clubName, inviteURL)

	// HTML version
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Club Invitation - Klubbspel</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 8px;">
        <h1 style="color: #2c3e50; margin-bottom: 30px;">🏓 You're Invited to Join %s</h1>
        
        <p style="font-size: 16px; margin-bottom: 25px;">
            Hi there!
        </p>
        
        <p style="font-size: 16px; margin-bottom: 25px;">
            <strong>%s</strong> has invited you to join the <strong>"%s"</strong> club on Klubbspel.
        </p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" 
               style="background-color: #27ae60; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">
                Join %s
            </a>
        </div>
        
        <p style="font-size: 14px; color: #666; margin-bottom: 20px;">
            If you already have an account, you'll be able to join the club after signing in.<br>
            If you don't have an account yet, you can create one using this email address.
        </p>
        
        <div style="margin-top: 30px; text-align: center;">
            <p style="font-size: 14px; color: #666;">
                Best regards,<br>
                The Klubbspel Team
            </p>
        </div>
    </div>
</body>
</html>
`, clubName, inviterName, clubName, inviteURL, clubName)

	if s.client == nil {
		// Development mode - log instead of sending
		fmt.Printf("📧 INVITATION EMAIL (Development Mode)\n")
		fmt.Printf("To: %s\n", toEmail)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Invite URL: %s\n", inviteURL)
		fmt.Printf("---\n")
		return nil
	}

	msg := mail.NewSingleEmail(s.from, subject, to, plainText, htmlContent)

	_, err := s.client.SendWithContext(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send invitation email: %w", err)
	}

	return nil
}

// SendEmail sends a generic email with custom subject and body
func (s *SendGridService) SendEmail(ctx context.Context, toEmail, subject, body string) error {
	to := mail.NewEmail("", toEmail)

	if s.client == nil {
		// Development mode - log instead of sending
		fmt.Printf("📧 EMAIL (Development Mode)\n")
		fmt.Printf("To: %s\n", toEmail)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Body:\n%s\n", body)
		fmt.Printf("---\n")
		return nil
	}

	msg := mail.NewSingleEmail(s.from, subject, to, body, "")

	_, err := s.client.SendWithContext(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
