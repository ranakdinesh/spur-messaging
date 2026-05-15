package whatsapp

import "encoding/json"

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
	From        string              `json:"from"`
	ID          string              `json:"id"`
	Timestamp   string              `json:"timestamp"`
	Type        string              `json:"type"`
	Text        *WebhookText        `json:"text,omitempty"`
	Image       *WebhookMedia       `json:"image,omitempty"`
	Document    *WebhookDocument    `json:"document,omitempty"`
	Audio       *WebhookMedia       `json:"audio,omitempty"`
	Video       *WebhookMedia       `json:"video,omitempty"`
	Button      *WebhookButton      `json:"button,omitempty"`
	Interactive *WebhookInteractive `json:"interactive,omitempty"`
	Location    *WebhookLocation    `json:"location,omitempty"`
	Raw         json.RawMessage     `json:"-"`
}

type WebhookText struct {
	Body string `json:"body"`
}

type WebhookMedia struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

type WebhookDocument struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type WebhookButton struct {
	Text    string `json:"text,omitempty"`
	Payload string `json:"payload,omitempty"`
}

type WebhookInteractive struct {
	Type        string                     `json:"type"`
	ButtonReply *WebhookInteractiveReply   `json:"button_reply,omitempty"`
	ListReply   *WebhookInteractiveReply   `json:"list_reply,omitempty"`
	Raw         map[string]json.RawMessage `json:"-"`
}

type WebhookInteractiveReply struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type WebhookLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
	URL       string  `json:"url,omitempty"`
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
