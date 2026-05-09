package whatsapp

// Meta Webhook Payloads

type WebhookPayload struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

type WebhookEntry struct {
	ID      string          `json:"id"` // WABA ID
	Changes []WebhookChange `json:"changes"`
}

type WebhookChange struct {
	Value WebhookValue `json:"value"`
	Field string       `json:"field"`
}

type WebhookValue struct {
	MessagingProduct string           `json:"messaging_product"`
	Metadata         WebhookMetadata  `json:"metadata"`
	Statuses         []WebhookStatus  `json:"statuses,omitempty"`
	Messages         []WebhookMessage `json:"messages,omitempty"`
}

type WebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WebhookStatus struct {
	ID          string         `json:"id"`
	Status      string         `json:"status"`
	Timestamp   string         `json:"timestamp"`
	RecipientID string         `json:"recipient_id"`
	Errors      []WebhookError `json:"errors,omitempty"`
}

type WebhookMessage struct {
	From      string       `json:"from"`
	ID        string       `json:"id"`
	Timestamp string       `json:"timestamp"`
	Type      string       `json:"type"`
	Text      *WebhookText `json:"text,omitempty"`
	// Add other types as needed (image, video, etc.)
}

type WebhookText struct {
	Body string `json:"body"`
}

type WebhookError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// Meta API Requests/Responses

type SendMessageRequest struct {
	MessagingProduct string        `json:"messaging_product"`
	To               string        `json:"to"`
	Type             string        `json:"type"`
	Template         *TemplateData `json:"template,omitempty"`
	Text             *TextData     `json:"text,omitempty"`
}

type TemplateData struct {
	Name     string           `json:"name"`
	Language TemplateLanguage `json:"language"`
}

type TemplateLanguage struct {
	Code string `json:"code"`
}

type TextData struct {
	Body string `json:"body"`
}

type MetaResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}
