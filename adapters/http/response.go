package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ranakdinesh/spur-messaging/core/domain"
)

// APIResponse is the standard success response envelope
type APIResponse struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Meta    *Pagination `json:"meta,omitempty"`
}

// APIError is the standard error response envelope
type APIError struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func RespondOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func RespondCreated(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func RespondList(w http.ResponseWriter, data any, meta Pagination) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
		Meta:    &meta,
	})
}

var errorCodeMap = map[error]struct {
	code   string
	status int
}{
	// Generic
	domain.ErrNotFound:      {"NOT_FOUND", 404},
	domain.ErrAlreadyExists: {"ALREADY_EXISTS", 409},
	domain.ErrInvalidInput:  {"INVALID_INPUT", 400},
	domain.ErrUnauthorized:  {"UNAUTHORIZED", 401},
	domain.ErrForbidden:     {"FORBIDDEN", 403},

	// Provider
	domain.ErrProviderError:         {"PROVIDER_ERROR", 502},
	domain.ErrProviderNotConfigured: {"PROVIDER_NOT_CONFIGURED", 422},
	domain.ErrProviderTimeout:       {"PROVIDER_TIMEOUT", 504},
	domain.ErrCredentialsExpired:    {"CREDENTIALS_EXPIRED", 422},
	domain.ErrRateLimitExceeded:     {"RATE_LIMIT_EXCEEDED", 429},

	// WhatsApp
	domain.ErrTemplateNotApproved: {"TEMPLATE_NOT_APPROVED", 422},
	domain.ErrSessionWindowClosed: {"SESSION_WINDOW_CLOSED", 422},

	// Email
	domain.ErrSuppressed:   {"EMAIL_SUPPRESSED", 422},
	domain.ErrUnsubscribed: {"RECIPIENT_UNSUBSCRIBED", 422},

	// Contacts
	domain.ErrOptInRequired: {"OPT_IN_REQUIRED", 422},

	// Campaigns
	domain.ErrCampaignNotExecutable: {"CAMPAIGN_NOT_EXECUTABLE", 422},
	domain.ErrTemplateInUse:         {"TEMPLATE_IN_USE", 409},

	// Infrastructure
	domain.ErrQueueUnavailable: {"QUEUE_UNAVAILABLE", 503},
}

func RespondError(w http.ResponseWriter, err error) {
	var domainErr *domain.DomainError
	statusCode := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := err.Error()
	field := ""

	if errors.As(err, &domainErr) {
		message = domainErr.Message
		field = domainErr.Field
		err = domainErr.Err
	}

	// Try to find the error in our map
	for sentinel, entry := range errorCodeMap {
		if errors.Is(err, sentinel) {
			code = entry.code
			statusCode = entry.status
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(APIError{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Field:   field,
		},
	})
}

func RespondValidationError(w http.ResponseWriter, field, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(APIError{
		Success: false,
		Error: ErrorDetail{
			Code:    "VALIDATION_ERROR",
			Message: message,
			Field:   field,
		},
	})
}
