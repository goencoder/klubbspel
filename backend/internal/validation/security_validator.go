package validation

import (
	"context"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SecurityValidator provides enhanced input validation with security protections
type SecurityValidator struct {
	// Regular expressions for validation
	emailRegex       *regexp.Regexp
	nameRegex        *regexp.Regexp
	clubNameRegex    *regexp.Regexp
	descriptionRegex *regexp.Regexp

	// Security patterns
	sqlInjectionPatterns     []*regexp.Regexp
	xssPatterns              []*regexp.Regexp
	commandInjectionPatterns []*regexp.Regexp
}

// ValidationConfig defines validation parameters
type ValidationConfig struct {
	MaxNameLength        int
	MaxDescriptionLength int
	MaxEmailLength       int
	AllowUnicode         bool
	StrictMode           bool
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(config ValidationConfig) *SecurityValidator {
	validator := &SecurityValidator{
		// Email validation - RFC 5322 compliant
		emailRegex: regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),

		// Name validation - letters, spaces, hyphens, apostrophes
		nameRegex: regexp.MustCompile(`^[a-zA-ZÀ-ÿ\s\-'\.]+$`),

		// Club name validation - alphanumeric, spaces, common punctuation
		clubNameRegex: regexp.MustCompile(`^[a-zA-ZÀ-ÿ0-9\s\-_\.&]+$`),

		// Description validation - printable characters
		descriptionRegex: regexp.MustCompile(`^[\p{L}\p{N}\p{P}\p{Z}]*$`),
	}

	// Initialize security patterns
	validator.initSecurityPatterns()

	return validator
}

// initSecurityPatterns initializes patterns for detecting malicious input
func (sv *SecurityValidator) initSecurityPatterns() {
	// SQL injection patterns
	sv.sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|drop\s+table|delete\s+from|insert\s+into|update\s+set)`),
		regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|sp_|xp_)`),
		regexp.MustCompile(`(?i)(script\s*:|\bjavascript\b|vbscript\b)`),
		regexp.MustCompile(`(['"][\s]*;|;[\s]*--|/\*|\*/)`),
		regexp.MustCompile(`(?i)(0x[0-9a-f]+|char\s*\(|ascii\s*\()`),
	}

	// XSS patterns
	sv.xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(<script|</script>|javascript:|vbscript:)`),
		regexp.MustCompile(`(?i)(on\w+\s*=|expression\s*\(|@import)`),
		regexp.MustCompile(`(?i)(<iframe|<object|<embed|<form|<input)`),
		regexp.MustCompile(`(?i)(eval\s*\(|setTimeout\s*\(|setInterval\s*\()`),
		regexp.MustCompile(`(?i)(document\.|window\.|alert\s*\(|confirm\s*\()`),
	}

	// Command injection patterns
	sv.commandInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(\||&|;|\$\(|` + "`" + `)`),
		regexp.MustCompile(`(?i)(cmd|powershell|bash|sh|nc|netcat)`),
		regexp.MustCompile(`(\.\./|\.\.\\|/etc/|/proc/|/sys/)`),
		regexp.MustCompile(`(?i)(wget|curl|ping|nslookup|dig)`),
	}
}

// ValidateEmail validates email addresses with security checks
func (sv *SecurityValidator) ValidateEmail(ctx context.Context, email string) error {
	if email == "" {
		return status.Error(codes.InvalidArgument, "EMPTY_EMAIL")
	}

	// Length check
	if len(email) > 254 { // RFC 5321 limit
		return status.Error(codes.InvalidArgument, "EMAIL_TOO_LONG")
	}

	// Basic format validation
	if !sv.emailRegex.MatchString(email) {
		return status.Error(codes.InvalidArgument, "INVALID_EMAIL_FORMAT")
	}

	// RFC 5322 validation using standard library
	if _, err := mail.ParseAddress(email); err != nil {
		return status.Error(codes.InvalidArgument, "INVALID_EMAIL_FORMAT")
	}

	// Security checks
	if err := sv.checkSecurityThreats(email); err != nil {
		return err
	}

	// Domain validation
	if err := sv.validateEmailDomain(email); err != nil {
		return err
	}

	return nil
}

// ValidateName validates person/player names with security checks
func (sv *SecurityValidator) ValidateName(ctx context.Context, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "EMPTY_NAME")
	}

	// Trim whitespace
	name = strings.TrimSpace(name)
	if name == "" {
		return status.Error(codes.InvalidArgument, "EMPTY_NAME")
	}

	// Length check
	if len(name) > 100 {
		return status.Error(codes.InvalidArgument, "NAME_TOO_LONG")
	}

	if len(name) < 2 {
		return status.Error(codes.InvalidArgument, "NAME_TOO_SHORT")
	}

	// UTF-8 validation
	if !utf8.ValidString(name) {
		return status.Error(codes.InvalidArgument, "INVALID_UTF8_NAME")
	}

	// Character validation
	if !sv.nameRegex.MatchString(name) {
		return status.Error(codes.InvalidArgument, "INVALID_NAME_CHARACTERS")
	}

	// Security checks
	if err := sv.checkSecurityThreats(name); err != nil {
		return err
	}

	// Additional name-specific checks
	if err := sv.validateNameContent(name); err != nil {
		return err
	}

	return nil
}

// ValidateClubName validates club names with security checks
func (sv *SecurityValidator) ValidateClubName(ctx context.Context, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "EMPTY_CLUB_NAME")
	}

	// Trim whitespace
	name = strings.TrimSpace(name)
	if name == "" {
		return status.Error(codes.InvalidArgument, "EMPTY_CLUB_NAME")
	}

	// Length check
	if len(name) > 200 {
		return status.Error(codes.InvalidArgument, "CLUB_NAME_TOO_LONG")
	}

	if len(name) < 3 {
		return status.Error(codes.InvalidArgument, "CLUB_NAME_TOO_SHORT")
	}

	// UTF-8 validation
	if !utf8.ValidString(name) {
		return status.Error(codes.InvalidArgument, "INVALID_UTF8_CLUB_NAME")
	}

	// Character validation
	if !sv.clubNameRegex.MatchString(name) {
		return status.Error(codes.InvalidArgument, "INVALID_CLUB_NAME_CHARACTERS")
	}

	// Security checks
	if err := sv.checkSecurityThreats(name); err != nil {
		return err
	}

	return nil
}

// ValidateDescription validates descriptions with security checks
func (sv *SecurityValidator) ValidateDescription(ctx context.Context, description string) error {
	// Description is optional
	if description == "" {
		return nil
	}

	// Length check
	if len(description) > 1000 {
		return status.Error(codes.InvalidArgument, "DESCRIPTION_TOO_LONG")
	}

	// UTF-8 validation
	if !utf8.ValidString(description) {
		return status.Error(codes.InvalidArgument, "INVALID_UTF8_DESCRIPTION")
	}

	// Character validation - allow Unicode for international users
	if !sv.descriptionRegex.MatchString(description) {
		return status.Error(codes.InvalidArgument, "INVALID_DESCRIPTION_CHARACTERS")
	}

	// Security checks
	if err := sv.checkSecurityThreats(description); err != nil {
		return err
	}

	return nil
}

// checkSecurityThreats checks input for common security threats
func (sv *SecurityValidator) checkSecurityThreats(input string) error {
	// Convert to lowercase for pattern matching
	lowerInput := strings.ToLower(input)

	// Check for SQL injection patterns
	for _, pattern := range sv.sqlInjectionPatterns {
		if pattern.MatchString(lowerInput) {
			return status.Error(codes.InvalidArgument, "POTENTIALLY_MALICIOUS_INPUT")
		}
	}

	// Check for XSS patterns
	for _, pattern := range sv.xssPatterns {
		if pattern.MatchString(lowerInput) {
			return status.Error(codes.InvalidArgument, "POTENTIALLY_MALICIOUS_INPUT")
		}
	}

	// Check for command injection patterns
	for _, pattern := range sv.commandInjectionPatterns {
		if pattern.MatchString(input) { // Don't lowercase for these patterns
			return status.Error(codes.InvalidArgument, "POTENTIALLY_MALICIOUS_INPUT")
		}
	}

	return nil
}

// validateEmailDomain performs additional email domain validation
func (sv *SecurityValidator) validateEmailDomain(email string) error {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return status.Error(codes.InvalidArgument, "INVALID_EMAIL_FORMAT")
	}

	domain := parts[1]

	// Check for obviously invalid domains
	invalidDomains := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"example.com",
		"test.com",
		"invalid.invalid",
	}

	for _, invalid := range invalidDomains {
		if strings.EqualFold(domain, invalid) {
			return status.Error(codes.InvalidArgument, "INVALID_EMAIL_DOMAIN")
		}
	}

	// Check domain length
	if len(domain) > 253 {
		return status.Error(codes.InvalidArgument, "EMAIL_DOMAIN_TOO_LONG")
	}

	return nil
}

// validateNameContent performs additional name content validation
func (sv *SecurityValidator) validateNameContent(name string) error {
	// Check for excessive whitespace
	if strings.Contains(name, "  ") { // Double spaces
		return status.Error(codes.InvalidArgument, "EXCESSIVE_WHITESPACE_IN_NAME")
	}

	// Check for names that are all symbols
	hasLetter := false
	for _, r := range name {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}

	if !hasLetter {
		return status.Error(codes.InvalidArgument, "NAME_MUST_CONTAIN_LETTERS")
	}

	// Check for obviously fake names
	suspiciousNames := []string{
		"test", "testing", "admin", "administrator", "root", "user",
		"null", "undefined", "delete", "drop", "select", "insert",
		"update", "script", "hack", "exploit",
	}

	lowerName := strings.ToLower(name)
	for _, suspicious := range suspiciousNames {
		if lowerName == suspicious {
			return status.Error(codes.InvalidArgument, "INVALID_NAME")
		}
	}

	return nil
}

// SanitizeInput sanitizes input by removing potentially dangerous characters
func (sv *SecurityValidator) SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters except tab, newline, carriage return
	var result strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			continue
		}
		result.WriteRune(r)
	}

	return strings.TrimSpace(result.String())
}

// ValidateToken validates authentication tokens
func (sv *SecurityValidator) ValidateToken(ctx context.Context, token string) error {
	if token == "" {
		return status.Error(codes.InvalidArgument, "EMPTY_TOKEN")
	}

	// Token length check (assuming UUID format: 36 characters)
	if len(token) < 10 || len(token) > 100 {
		return status.Error(codes.InvalidArgument, "INVALID_TOKEN_LENGTH")
	}

	// Check for valid characters (alphanumeric + hyphens for UUID)
	validTokenRegex := regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)
	if !validTokenRegex.MatchString(token) {
		return status.Error(codes.InvalidArgument, "INVALID_TOKEN_FORMAT")
	}

	// Security checks
	if err := sv.checkSecurityThreats(token); err != nil {
		return err
	}

	return nil
}

// GetDefaultValidationConfig returns default validation configuration
func GetDefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxNameLength:        100,
		MaxDescriptionLength: 1000,
		MaxEmailLength:       254,
		AllowUnicode:         true,
		StrictMode:           true,
	}
}
