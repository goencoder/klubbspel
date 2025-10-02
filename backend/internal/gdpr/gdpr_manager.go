package gdpr

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/mongo"
)

// GDPRManager provides GDPR compliance functionality
type GDPRManager struct {
	logger     zerolog.Logger
	config     GDPRConfig
	encryption *EncryptionService
	db         *mongo.Database
}

// GDPRConfig defines GDPR compliance configuration
type GDPRConfig struct {
	ServiceName      string
	DataController   string
	DPOContact       string
	PrivacyPolicyURL string

	// Data retention policies
	DefaultRetentionDays  int
	InactiveRetentionDays int
	LegalHoldDays         int

	// Encryption settings
	EncryptionKey        string
	EncryptPII           bool
	EncryptSensitiveData bool

	// Anonymization settings
	AnonymizeAfterDays  int
	EnableRightToForget bool

	// Consent management
	RequireExplicitConsent bool
	ConsentVersion         string

	// Data minimization
	EnableDataMinimization bool
	AutoCleanupEnabled     bool
}

// PersonalDataCategory defines categories of personal data
type PersonalDataCategory string

const (
	// GDPR Article 9 - Special categories
	CategoryBiometric PersonalDataCategory = "biometric"
	CategoryGenetic   PersonalDataCategory = "genetic"
	CategoryHealth    PersonalDataCategory = "health"
	CategoryReligious PersonalDataCategory = "religious"
	CategoryPolitical PersonalDataCategory = "political"
	CategoryUnion     PersonalDataCategory = "union"
	CategorySexual    PersonalDataCategory = "sexual"
	CategoryRacial    PersonalDataCategory = "racial"

	// GDPR Article 4 - Regular personal data
	CategoryIdentity     PersonalDataCategory = "identity"
	CategoryContact      PersonalDataCategory = "contact"
	CategoryLocation     PersonalDataCategory = "location"
	CategoryBehavioral   PersonalDataCategory = "behavioral"
	CategoryFinancial    PersonalDataCategory = "financial"
	CategoryProfessional PersonalDataCategory = "professional"
	CategoryTechnical    PersonalDataCategory = "technical"
)

// DataSubject represents a data subject (person)
type DataSubject struct {
	ID              string               `json:"id" bson:"_id"`
	Email           string               `json:"email" bson:"email"`
	CreatedAt       time.Time            `json:"created_at" bson:"created_at"`
	LastActive      time.Time            `json:"last_active" bson:"last_active"`
	ConsentStatus   ConsentStatus        `json:"consent_status" bson:"consent_status"`
	DataProcessing  []ProcessingActivity `json:"data_processing" bson:"data_processing"`
	RetentionPolicy string               `json:"retention_policy" bson:"retention_policy"`
	LegalBasis      string               `json:"legal_basis" bson:"legal_basis"`
	IsAnonymized    bool                 `json:"is_anonymized" bson:"is_anonymized"`
	DeletedAt       *time.Time           `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
}

// ConsentStatus tracks consent for different processing activities
type ConsentStatus struct {
	ConsentVersion   string                     `json:"consent_version" bson:"consent_version"`
	ConsentTimestamp time.Time                  `json:"consent_timestamp" bson:"consent_timestamp"`
	Activities       map[string]ActivityConsent `json:"activities" bson:"activities"`
	WithdrawnAt      *time.Time                 `json:"withdrawn_at,omitempty" bson:"withdrawn_at,omitempty"`
}

// ActivityConsent represents consent for a specific processing activity
type ActivityConsent struct {
	Granted       bool      `json:"granted" bson:"granted"`
	Timestamp     time.Time `json:"timestamp" bson:"timestamp"`
	LegalBasis    string    `json:"legal_basis" bson:"legal_basis"`
	Purpose       string    `json:"purpose" bson:"purpose"`
	DataTypes     []string  `json:"data_types" bson:"data_types"`
	RetentionDays int       `json:"retention_days" bson:"retention_days"`
}

// ProcessingActivity represents a data processing activity
type ProcessingActivity struct {
	ID            string                 `json:"id" bson:"id"`
	Name          string                 `json:"name" bson:"name"`
	Purpose       string                 `json:"purpose" bson:"purpose"`
	LegalBasis    string                 `json:"legal_basis" bson:"legal_basis"`
	DataTypes     []PersonalDataCategory `json:"data_types" bson:"data_types"`
	Recipients    []string               `json:"recipients" bson:"recipients"`
	Transfers     []DataTransfer         `json:"transfers" bson:"transfers"`
	RetentionDays int                    `json:"retention_days" bson:"retention_days"`
	StartDate     time.Time              `json:"start_date" bson:"start_date"`
	EndDate       *time.Time             `json:"end_date,omitempty" bson:"end_date,omitempty"`
}

// DataTransfer represents a data transfer to third countries
type DataTransfer struct {
	Country          string    `json:"country" bson:"country"`
	Organization     string    `json:"organization" bson:"organization"`
	SafeguardType    string    `json:"safeguard_type" bson:"safeguard_type"` // adequacy, SCC, BCR, etc.
	SafeguardDetails string    `json:"safeguard_details" bson:"safeguard_details"`
	TransferDate     time.Time `json:"transfer_date" bson:"transfer_date"`
}

// DataSubjectRequest represents a data subject request
type DataSubjectRequest struct {
	ID          string                 `json:"id" bson:"_id"`
	SubjectID   string                 `json:"subject_id" bson:"subject_id"`
	Type        DataSubjectRequestType `json:"type" bson:"type"`
	Description string                 `json:"description" bson:"description"`
	Status      RequestStatus          `json:"status" bson:"status"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	Response    string                 `json:"response" bson:"response"`
	ProcessedBy string                 `json:"processed_by" bson:"processed_by"`
}

// DataSubjectRequestType defines types of data subject requests
type DataSubjectRequestType string

const (
	RequestTypeAccess          DataSubjectRequestType = "access"           // Article 15
	RequestTypeRectification   DataSubjectRequestType = "rectification"    // Article 16
	RequestTypeErasure         DataSubjectRequestType = "erasure"          // Article 17
	RequestTypeRestriction     DataSubjectRequestType = "restriction"      // Article 18
	RequestTypePortability     DataSubjectRequestType = "portability"      // Article 20
	RequestTypeObjection       DataSubjectRequestType = "objection"        // Article 21
	RequestTypeWithdrawConsent DataSubjectRequestType = "withdraw_consent" // Article 7
)

// RequestStatus defines the status of a data subject request
type RequestStatus string

const (
	StatusPending    RequestStatus = "pending"
	StatusProcessing RequestStatus = "processing"
	StatusCompleted  RequestStatus = "completed"
	StatusRejected   RequestStatus = "rejected"
	StatusApproved   RequestStatus = "approved"
)

// EncryptionService provides encryption/decryption for personal data
type EncryptionService struct {
	key    []byte
	cipher cipher.AEAD
}

// NewGDPRManager creates a new GDPR manager
func NewGDPRManager(logger zerolog.Logger, config GDPRConfig, db *mongo.Database) (*GDPRManager, error) {
	encryption, err := NewEncryptionService(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encryption: %w", err)
	}

	return &GDPRManager{
		logger:     logger,
		config:     config,
		encryption: encryption,
		db:         db,
	}, nil
}

// NewEncryptionService creates a new encryption service
func NewEncryptionService(key string) (*EncryptionService, error) {
	if key == "" {
		return nil, fmt.Errorf("encryption key cannot be empty, must be set via GDPR_ENCRYPTION_KEY environment variable")
	}
	
	if len(key) < 16 {
		return nil, fmt.Errorf("encryption key must be at least 16 characters long")
	}

	// Derive a consistent key from the provided key
	hash := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &EncryptionService{
		key:    hash[:],
		cipher: gcm,
	}, nil
}

// Encrypt encrypts sensitive data
func (es *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, es.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := es.cipher.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts sensitive data
func (es *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	nonceSize := es.cipher.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := es.cipher.Open(nil, nonce, ciphertext_bytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// RegisterDataSubject registers a new data subject
func (gm *GDPRManager) RegisterDataSubject(ctx context.Context, email string, initialConsent ConsentStatus) (*DataSubject, error) {
	subject := &DataSubject{
		ID:              generateDataSubjectID(),
		Email:           email,
		CreatedAt:       time.Now(),
		LastActive:      time.Now(),
		ConsentStatus:   initialConsent,
		DataProcessing:  []ProcessingActivity{},
		RetentionPolicy: "default",
		LegalBasis:      "consent",
		IsAnonymized:    false,
	}

	// Log registration
	gm.logger.Info().
		Str("subject_id", subject.ID).
		Str("email", maskEmail(email)).
		Str("consent_version", initialConsent.ConsentVersion).
		Msg("Data subject registered")

	return subject, nil
}

// ProcessDataSubjectRequest processes a data subject request
func (gm *GDPRManager) ProcessDataSubjectRequest(ctx context.Context, request DataSubjectRequest) error {
	gm.logger.Info().
		Str("request_id", request.ID).
		Str("subject_id", request.SubjectID).
		Str("type", string(request.Type)).
		Msg("Processing data subject request")

	switch request.Type {
	case RequestTypeAccess:
		return gm.processAccessRequest(ctx, request)
	case RequestTypeRectification:
		return gm.processRectificationRequest(ctx, request)
	case RequestTypeErasure:
		return gm.processErasureRequest(ctx, request)
	case RequestTypeRestriction:
		return gm.processRestrictionRequest(ctx, request)
	case RequestTypePortability:
		return gm.processPortabilityRequest(ctx, request)
	case RequestTypeObjection:
		return gm.processObjectionRequest(ctx, request)
	case RequestTypeWithdrawConsent:
		return gm.processWithdrawConsentRequest(ctx, request)
	default:
		return fmt.Errorf("unknown request type: %s", request.Type)
	}
}

// processAccessRequest handles data access requests
func (gm *GDPRManager) processAccessRequest(ctx context.Context, request DataSubjectRequest) error {
	// Implementation would collect all personal data for the subject
	// and provide it in a structured format

	gm.logger.Info().
		Str("request_id", request.ID).
		Str("subject_id", request.SubjectID).
		Msg("Processing access request")

	// This is a placeholder - implement actual data collection
	return nil
}

// processErasureRequest handles right to be forgotten requests
func (gm *GDPRManager) processErasureRequest(ctx context.Context, request DataSubjectRequest) error {
	gm.logger.Info().
		Str("request_id", request.ID).
		Str("subject_id", request.SubjectID).
		Msg("Processing erasure request")

	// Implementation would:
	// 1. Verify the request is valid
	// 2. Check for legal obligations to retain data
	// 3. Anonymize or delete personal data
	// 4. Notify third parties if necessary

	return nil
}

// Other request processing methods...
func (gm *GDPRManager) processRectificationRequest(ctx context.Context, request DataSubjectRequest) error {
	return nil // Placeholder
}

func (gm *GDPRManager) processRestrictionRequest(ctx context.Context, request DataSubjectRequest) error {
	return nil // Placeholder
}

func (gm *GDPRManager) processPortabilityRequest(ctx context.Context, request DataSubjectRequest) error {
	return nil // Placeholder
}

func (gm *GDPRManager) processObjectionRequest(ctx context.Context, request DataSubjectRequest) error {
	return nil // Placeholder
}

func (gm *GDPRManager) processWithdrawConsentRequest(ctx context.Context, request DataSubjectRequest) error {
	return nil // Placeholder
}

// AnonymizePersonalData anonymizes personal data while preserving utility
func (gm *GDPRManager) AnonymizePersonalData(ctx context.Context, subjectID string) error {
	gm.logger.Info().
		Str("subject_id", subjectID).
		Msg("Anonymizing personal data")

	// Implementation would:
	// 1. Replace identifiers with anonymous IDs
	// 2. Remove or generalize quasi-identifiers
	// 3. Add noise to sensitive attributes
	// 4. Ensure k-anonymity or differential privacy

	return nil
}

// CleanupExpiredData removes data that has exceeded retention periods
func (gm *GDPRManager) CleanupExpiredData(ctx context.Context) error {
	gm.logger.Info().Msg("Starting expired data cleanup")

	cutoffDate := time.Now().AddDate(0, 0, -gm.config.DefaultRetentionDays)

	// Implementation would:
	// 1. Identify data past retention period
	// 2. Check for legal holds
	// 3. Anonymize or delete expired data
	// 4. Log cleanup actions

	gm.logger.Info().
		Time("cutoff_date", cutoffDate).
		Int("retention_days", gm.config.DefaultRetentionDays).
		Msg("Data cleanup completed")

	return nil
}

// GeneratePrivacyReport generates a privacy compliance report
func (gm *GDPRManager) GeneratePrivacyReport(ctx context.Context) (*PrivacyReport, error) {
	report := &PrivacyReport{
		GeneratedAt:    time.Now(),
		ServiceName:    gm.config.ServiceName,
		DataController: gm.config.DataController,
		ReportPeriod:   "last_30_days",
	}

	// Implementation would collect compliance metrics

	return report, nil
}

// PrivacyReport contains privacy compliance metrics
type PrivacyReport struct {
	GeneratedAt    time.Time `json:"generated_at"`
	ServiceName    string    `json:"service_name"`
	DataController string    `json:"data_controller"`
	ReportPeriod   string    `json:"report_period"`

	DataSubjects      int `json:"data_subjects"`
	ActiveConsents    int `json:"active_consents"`
	WithdrawnConsents int `json:"withdrawn_consents"`

	Requests struct {
		Access        int `json:"access"`
		Rectification int `json:"rectification"`
		Erasure       int `json:"erasure"`
		Restriction   int `json:"restriction"`
		Portability   int `json:"portability"`
		Objection     int `json:"objection"`
	} `json:"requests"`

	DataBreaches    int `json:"data_breaches"`
	ComplianceScore int `json:"compliance_score"` // 0-100
}

// Helper functions

func generateDataSubjectID() string {
	return fmt.Sprintf("ds_%d", time.Now().UnixNano())
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "[MASKED]"
	}

	username := parts[0]
	domain := parts[1]

	if len(username) <= 2 {
		return "[MASKED]@" + domain
	}

	return string(username[0]) + "***" + string(username[len(username)-1]) + "@" + domain
}

// GetDefaultGDPRConfig returns default GDPR configuration
func GetDefaultGDPRConfig() GDPRConfig {
	return GDPRConfig{
		ServiceName:      "Klubbspel Tournament Management",
		DataController:   "Klubbspel AB",
		DPOContact:       "dpo@klubbspel.se",
		PrivacyPolicyURL: "https://klubbspel.se/privacy",

		DefaultRetentionDays:  1095, // 3 years
		InactiveRetentionDays: 365,  // 1 year for inactive users
		LegalHoldDays:         2555, // 7 years for legal requirements

		EncryptionKey:        "", // Must be set via environment variable GDPR_ENCRYPTION_KEY
		EncryptPII:           true,
		EncryptSensitiveData: true,

		AnonymizeAfterDays:  1095, // 3 years
		EnableRightToForget: true,

		RequireExplicitConsent: true,
		ConsentVersion:         "1.0",

		EnableDataMinimization: true,
		AutoCleanupEnabled:     true,
	}
}
