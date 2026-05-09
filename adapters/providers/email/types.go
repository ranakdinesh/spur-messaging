package email

// Shared email types could go here if they are not already in domain or ports.
// Based on AGENTS.md, Section 12.6 and 12.7, most structs are already in domain/ports.
// We'll use this file for provider-specific response types if needed or internal helpers.

type SendGridResponse struct {
	Errors []struct {
		Message string `json:"message"`
		Field   string `json:"field"`
		Help    string `json:"help"`
	} `json:"errors"`
}

type MailgunResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type PostmarkResponse struct {
	To          string `json:"To"`
	SubmittedAt string `json:"SubmittedAt"`
	MessageID   string `json:"MessageID"`
	ErrorCode   int    `json:"ErrorCode"`
	Message     string `json:"Message"`
}
