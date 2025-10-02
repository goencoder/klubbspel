package audit

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	maskedValue          = "[MASKED]"
	emailPartsExpected   = 2
	minUsernameLength    = 2
	defaultFlushInterval = 5 * time.Second
	defaultRetentionDays = 90
)

// AuditLogger provides comprehensive audit logging functionality
type AuditLogger struct {
	logger   zerolog.Logger
	config   AuditConfig
	enricher *LogEnricher
}

// AuditConfig defines audit logging configuration
type AuditConfig struct {
	// Service information
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Logging levels for different event types
	LogLevels map[EventType]zerolog.Level

	// Sensitive data handling
	MaskSensitiveData bool
	SensitiveFields   []string

	// Performance settings
	BufferSize    int
	FlushInterval time.Duration
	SampleRate    float64 // 0.0 to 1.0

	// Compliance settings
	RetentionDays  int
	EncryptLogs    bool
	ComplianceMode string // "GDPR", "HIPAA", "SOX", etc.
}

// EventType represents different types of audit events
type EventType string

const (
	// Authentication events
	EventAuthLogin          EventType = "auth.login"
	EventAuthLogout         EventType = "auth.logout"
	EventAuthTokenGenerated EventType = "auth.token.generated" // #nosec G101 - token event name, not credential
	EventAuthTokenValidated EventType = "auth.token.validated" // #nosec G101 - token event name, not credential
	EventAuthTokenRevoked   EventType = "auth.token.revoked"   // #nosec G101 - token event name, not credential
	EventAuthFailure        EventType = "auth.failure"

	// Authorization events
	EventAuthzPermissionCheck EventType = "authz.permission.check"
	EventAuthzAccessDenied    EventType = "authz.access.denied"
	EventAuthzRoleChanged     EventType = "authz.role.changed"

	// Data events
	EventDataCreate EventType = "data.create"
	EventDataRead   EventType = "data.read"
	EventDataUpdate EventType = "data.update"
	EventDataDelete EventType = "data.delete"
	EventDataExport EventType = "data.export"
	EventDataImport EventType = "data.import"

	// Administrative events
	EventAdminUserCreate   EventType = "admin.user.create"
	EventAdminUserUpdate   EventType = "admin.user.update"
	EventAdminUserDelete   EventType = "admin.user.delete"
	EventAdminClubCreate   EventType = "admin.club.create"
	EventAdminClubUpdate   EventType = "admin.club.update"
	EventAdminClubDelete   EventType = "admin.club.delete"
	EventAdminConfigChange EventType = "admin.config.change"

	// Security events
	EventSecurityThreatDetected     EventType = "security.threat.detected"
	EventSecurityRateLimited        EventType = "security.rate.limited"
	EventSecuritySuspiciousActivity EventType = "security.suspicious.activity"
	EventSecurityBruteForce         EventType = "security.brute.force"

	// System events
	EventSystemStartup  EventType = "system.startup"
	EventSystemShutdown EventType = "system.shutdown"
	EventSystemError    EventType = "system.error"
	EventSystemAlert    EventType = "system.alert"
)

// AuditEvent represents a single audit event
type AuditEvent struct {
	// Core event information
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      EventType `json:"type"`
	Category  string    `json:"category"`

	// Actor information (who performed the action)
	ActorID    string `json:"actor_id,omitempty"`
	ActorType  string `json:"actor_type,omitempty"`
	ActorEmail string `json:"actor_email,omitempty"`
	ActorIP    string `json:"actor_ip,omitempty"`
	ActorAgent string `json:"actor_agent,omitempty"`

	// Target information (what was acted upon)
	TargetID      string                 `json:"target_id,omitempty"`
	TargetType    string                 `json:"target_type,omitempty"`
	TargetDetails map[string]interface{} `json:"target_details,omitempty"`

	// Action information
	Action   string `json:"action"`
	Method   string `json:"method,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Result   string `json:"result"` // SUCCESS, FAILURE, ERROR

	// Context information
	SessionID      string `json:"session_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	ClubID         string `json:"club_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`

	// Additional metadata
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
	Changes []FieldChange          `json:"changes,omitempty"`

	// Security context
	SecurityLevel string `json:"security_level,omitempty"`
	RiskScore     int    `json:"risk_score,omitempty"`

	// Compliance information
	DataClassification string `json:"data_classification,omitempty"`
	RetentionPeriod    string `json:"retention_period,omitempty"`

	// Technical information
	Service        string `json:"service"`
	ServiceVersion string `json:"service_version"`
	Environment    string `json:"environment"`
}

// FieldChange represents a change to a specific field
type FieldChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

// LogEnricher enriches log entries with additional context
type LogEnricher struct {
	sensitiveFields map[string]bool
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger zerolog.Logger, config AuditConfig) *AuditLogger { //nolint:gocritic
	enricher := &LogEnricher{
		sensitiveFields: make(map[string]bool),
	}

	// Build sensitive fields map for fast lookup
	for _, field := range config.SensitiveFields {
		enricher.sensitiveFields[field] = true
	}

	return &AuditLogger{
		logger:   logger,
		config:   config,
		enricher: enricher,
	}
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(ctx context.Context, event AuditEvent) { //nolint:gocritic
	// Sample if configured
	if al.config.SampleRate < 1.0 && al.config.SampleRate > 0.0 {
		// Simple sampling logic - in production, use more sophisticated sampling
		if time.Now().UnixNano()%100 >= int64(al.config.SampleRate*100) {
			return
		}
	}

	// Enrich event with context
	enrichedEvent := al.enrichEvent(ctx, event)

	// Mask sensitive data if configured
	if al.config.MaskSensitiveData {
		enrichedEvent = al.maskSensitiveData(enrichedEvent)
	}

	// Determine log level
	level := al.getLogLevel(event.Type)

	// Create log entry
	logEvent := al.logger.WithLevel(level).
		Str("event_id", enrichedEvent.ID).
		Str("event_type", string(enrichedEvent.Type)).
		Str("event_category", enrichedEvent.Category).
		Time("timestamp", enrichedEvent.Timestamp).
		Str("action", enrichedEvent.Action).
		Str("result", enrichedEvent.Result)

	// Add actor information
	if enrichedEvent.ActorID != "" {
		logEvent = logEvent.Str("actor_id", enrichedEvent.ActorID)
	}
	if enrichedEvent.ActorType != "" {
		logEvent = logEvent.Str("actor_type", enrichedEvent.ActorType)
	}
	if enrichedEvent.ActorEmail != "" {
		logEvent = logEvent.Str("actor_email", enrichedEvent.ActorEmail)
	}
	if enrichedEvent.ActorIP != "" {
		logEvent = logEvent.Str("actor_ip", enrichedEvent.ActorIP)
	}

	// Add target information
	if enrichedEvent.TargetID != "" {
		logEvent = logEvent.Str("target_id", enrichedEvent.TargetID)
	}
	if enrichedEvent.TargetType != "" {
		logEvent = logEvent.Str("target_type", enrichedEvent.TargetType)
	}

	// Add context information
	if enrichedEvent.SessionID != "" {
		logEvent = logEvent.Str("session_id", enrichedEvent.SessionID)
	}
	if enrichedEvent.RequestID != "" {
		logEvent = logEvent.Str("request_id", enrichedEvent.RequestID)
	}
	if enrichedEvent.ClubID != "" {
		logEvent = logEvent.Str("club_id", enrichedEvent.ClubID)
	}

	// Add security information
	if enrichedEvent.SecurityLevel != "" {
		logEvent = logEvent.Str("security_level", enrichedEvent.SecurityLevel)
	}
	if enrichedEvent.RiskScore > 0 {
		logEvent = logEvent.Int("risk_score", enrichedEvent.RiskScore)
	}

	// Add changes if present
	if len(enrichedEvent.Changes) > 0 {
		logEvent = logEvent.Interface("changes", enrichedEvent.Changes)
	}

	// Add details if present
	if len(enrichedEvent.Details) > 0 {
		logEvent = logEvent.Interface("details", enrichedEvent.Details)
	}

	// Log the event
	logEvent.Msg(enrichedEvent.Message)
}

// enrichEvent enriches an audit event with additional context
func (al *AuditLogger) enrichEvent(ctx context.Context, event AuditEvent) AuditEvent { //nolint:gocritic
	// Set core service information
	event.Service = al.config.ServiceName
	event.ServiceVersion = al.config.ServiceVersion
	event.Environment = al.config.Environment

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Generate ID if not provided
	if event.ID == "" {
		event.ID = fmt.Sprintf("%s_%d", event.Type, time.Now().UnixNano())
	}

	// Extract information from gRPC context
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// Extract request ID
		if reqID := md.Get("x-request-id"); len(reqID) > 0 {
			event.RequestID = reqID[0]
		}

		// Extract session ID
		if sessionID := md.Get("x-session-id"); len(sessionID) > 0 {
			event.SessionID = sessionID[0]
		}

		// Extract user agent
		if userAgent := md.Get("user-agent"); len(userAgent) > 0 {
			event.ActorAgent = userAgent[0]
		}

		// Extract authorization info
		if auth := md.Get("authorization"); len(auth) > 0 {
			// Don't log the actual token, just indicate it's present
			event.Details = map[string]interface{}{
				"authenticated": true,
				"auth_method":   "bearer_token",
			}
		}
	}

	// Extract IP address from peer
	if p, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			event.ActorIP = tcpAddr.IP.String()
		}
	}

	// Set category based on event type
	event.Category = al.getCategoryFromEventType(event.Type)

	// Set security level and risk score
	event.SecurityLevel = al.getSecurityLevel(event.Type)
	event.RiskScore = al.getRiskScore(event.Type, event.Result)

	// Set data classification and retention
	event.DataClassification = al.getDataClassification(event.Type)
	event.RetentionPeriod = fmt.Sprintf("%d days", al.config.RetentionDays)

	return event
}

// maskSensitiveData masks sensitive information in the event
func (al *AuditLogger) maskSensitiveData(event AuditEvent) AuditEvent { //nolint:gocritic
	// Mask actor email if configured
	if al.enricher.sensitiveFields["email"] && event.ActorEmail != "" {
		event.ActorEmail = al.maskEmail(event.ActorEmail)
	}

	// Mask sensitive fields in details
	if event.Details != nil {
		event.Details = al.maskMapValues(event.Details)
	}

	// Mask sensitive fields in target details
	if event.TargetDetails != nil {
		event.TargetDetails = al.maskMapValues(event.TargetDetails)
	}

	// Mask sensitive fields in changes
	for i, change := range event.Changes {
		if al.enricher.sensitiveFields[change.Field] {
			event.Changes[i].OldValue = maskedValue
			event.Changes[i].NewValue = maskedValue
		}
	}

	return event
}

// Helper functions

func (al *AuditLogger) getLogLevel(eventType EventType) zerolog.Level {
	if level, exists := al.config.LogLevels[eventType]; exists {
		return level
	}

	// Default log levels based on event type
	switch eventType { //nolint:exhaustive
	case EventSecurityThreatDetected, EventSecurityBruteForce:
		return zerolog.ErrorLevel
	case EventAuthFailure, EventAuthzAccessDenied:
		return zerolog.WarnLevel
	case EventSystemError, EventSystemAlert:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func (al *AuditLogger) getCategoryFromEventType(eventType EventType) string {
	eventStr := string(eventType)
	parts := strings.Split(eventStr, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

func (al *AuditLogger) getSecurityLevel(eventType EventType) string {
	switch eventType { //nolint:exhaustive
	case EventSecurityThreatDetected, EventSecurityBruteForce:
		return "HIGH"
	case EventAuthFailure, EventAuthzAccessDenied, EventSecurityRateLimited:
		return "MEDIUM"
	case EventAdminUserDelete, EventAdminClubDelete:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func (al *AuditLogger) getRiskScore(eventType EventType, result string) int {
	baseScore := 0

	switch eventType { //nolint:exhaustive
	case EventSecurityThreatDetected:
		baseScore = 90
	case EventSecurityBruteForce:
		baseScore = 85
	case EventAuthFailure:
		baseScore = 30
	case EventAuthzAccessDenied:
		baseScore = 40
	case EventDataDelete:
		baseScore = 60
	case EventAdminUserDelete, EventAdminClubDelete:
		baseScore = 70
	default:
		baseScore = 10
	}

	// Adjust score based on result
	if result == "FAILURE" || result == "ERROR" {
		baseScore += 20
	}

	// Cap at 100
	if baseScore > 100 {
		baseScore = 100
	}

	return baseScore
}

func (al *AuditLogger) getDataClassification(eventType EventType) string {
	switch eventType { //nolint:exhaustive
	case EventAuthLogin, EventAuthTokenGenerated, EventAuthTokenValidated:
		return "CONFIDENTIAL"
	case EventDataCreate, EventDataUpdate, EventDataDelete:
		return "RESTRICTED"
	case EventAdminUserCreate, EventAdminUserUpdate, EventAdminUserDelete:
		return "CONFIDENTIAL"
	default:
		return "INTERNAL"
	}
}

func (al *AuditLogger) maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != emailPartsExpected {
		return maskedValue
	}

	username := parts[0]
	domain := parts[1]

	if len(username) <= minUsernameLength {
		return maskedValue + "@" + domain
	}

	return string(username[0]) + "***" + string(username[len(username)-1]) + "@" + domain
}

func (al *AuditLogger) maskMapValues(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		if al.enricher.sensitiveFields[key] {
			result[key] = maskedValue
		} else {
			result[key] = value
		}
	}

	return result
}

// GetDefaultAuditConfig returns a default audit configuration
func GetDefaultAuditConfig() AuditConfig {
	return AuditConfig{
		ServiceName:    "klubbspel-api",
		ServiceVersion: "1.0.0",
		Environment:    "production",

		LogLevels: map[EventType]zerolog.Level{
			EventSecurityThreatDetected: zerolog.ErrorLevel,
			EventSecurityBruteForce:     zerolog.ErrorLevel,
			EventAuthFailure:            zerolog.WarnLevel,
			EventAuthzAccessDenied:      zerolog.WarnLevel,
			EventSystemError:            zerolog.ErrorLevel,
		},

		MaskSensitiveData: true,
		SensitiveFields: []string{
			"email", "password", "token", "secret", "key",
			"ssn", "phone", "address", "credit_card",
		},

		BufferSize:    1000,
		FlushInterval: defaultFlushInterval,
		SampleRate:    1.0, // Log all events in production

		RetentionDays:  defaultRetentionDays, // 90 days retention
		EncryptLogs:    true,
		ComplianceMode: "GDPR",
	}
}
