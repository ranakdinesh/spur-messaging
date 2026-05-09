package domain

import "errors"

var (
	// Generic
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")

	// Provider
	ErrProviderError         = errors.New("provider error")
	ErrProviderNotConfigured = errors.New("provider not configured")
	ErrProviderTimeout       = errors.New("provider timeout")
	ErrCredentialsExpired    = errors.New("credentials expired")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")

	// WhatsApp
	ErrTemplateNotApproved = errors.New("template not approved")
	ErrSessionWindowClosed = errors.New("session window closed")

	// Email
	ErrSuppressed   = errors.New("email suppressed")
	ErrUnsubscribed = errors.New("recipient unsubscribed")

	// Contacts
	ErrOptInRequired = errors.New("opt-in required")

	// Campaigns
	ErrCampaignNotExecutable = errors.New("campaign not executable")
	ErrTemplateInUse         = errors.New("template in use")

	// Infrastructure
	ErrQueueUnavailable = errors.New("queue unavailable")
)

// DomainError wraps a sentinel error with context
type DomainError struct {
	Err     error
	Message string // human-readable detail
	Field   string // which field caused the error (optional)
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Err }

// Convenience constructors
func NewValidationError(field, message string) *DomainError {
	return &DomainError{Err: ErrInvalidInput, Message: message, Field: field}
}

func NewNotFoundError(resource string) *DomainError {
	return &DomainError{Err: ErrNotFound, Message: resource + " not found"}
}

func NewConflictError(message string) *DomainError {
	return &DomainError{Err: ErrAlreadyExists, Message: message}
}

func NewProviderError(message string) *DomainError {
	return &DomainError{Err: ErrProviderError, Message: message}
}
