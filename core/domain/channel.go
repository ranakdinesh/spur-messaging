package domain

import (
	"time"

	"github.com/google/uuid"
)

type Channel string

const (
	ChannelWhatsApp Channel = "whatsapp"
	ChannelSMS      Channel = "sms"
	ChannelEmail    Channel = "email"
)

type ProviderConfig struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Channel       Channel
	Provider      string // "meta_cloud", "msg91", "twilio", "sendgrid", "mailgun", "postmark"
	Credentials   []byte // AES-256-GCM encrypted JSON
	WebhookSecret string
	IsActive      bool
	PhoneNumberID string // WhatsApp-specific: Meta phone number ID
	WABAID        string // WhatsApp-specific: WhatsApp Business Account ID
	BusinessID    string // Meta Business ID
	DisplayPhone  string // The actual phone number (for display only)
	FromEmail     string // Email-specific: verified sender address
	FromName      string // Email-specific: sender display name
	ReplyToEmail  string // Email-specific: reply-to address
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// WhatsAppCredentials is the decrypted form of Credentials for WhatsApp
type WhatsAppCredentials struct {
	AccessToken string `json:"access_token"`
	AppSecret   string `json:"app_secret"` // for webhook signature verification
}

// SMSCredentials for MSG91 or Twilio
type SMSCredentials struct {
	AuthKey    string `json:"auth_key"`    // MSG91
	SenderID   string `json:"sender_id"`   // MSG91
	AccountSID string `json:"account_sid"` // Twilio
	AuthToken  string `json:"auth_token"`  // Twilio
	FromNumber string `json:"from_number"` // Twilio
}

// EmailCredentials — provider-specific API keys
// The platform default provider is set via MESSAGING_EMAIL_PROVIDER env var.
// Tenants can override by creating their own provider_config with their own keys.
type EmailCredentials struct {
	APIKey       string `json:"api_key"`                // All providers
	Domain       string `json:"domain,omitempty"`       // Mailgun: sending domain
	ServerToken  string `json:"server_token,omitempty"` // Postmark: server token
	WebhookToken string `json:"webhook_token,omitempty"` // For webhook signature verification
}
