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

const (
	ProviderMetaCloud = "meta_cloud"
	ProviderMSG91     = "msg91"
	ProviderTwilio    = "twilio"
	ProviderSendGrid  = "sendgrid"
	ProviderMailgun   = "mailgun"
	ProviderPostmark  = "postmark"
	ProviderSMTP      = "smtp"
	ProviderDevEmail  = "dev_email"
)

type ProviderConfig struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Channel       Channel
	Provider      string // "meta_cloud", "msg91", "twilio", "sendgrid", "mailgun", "postmark", "smtp", "dev_email"
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
	TrackOpens    *bool  // Email: override open tracking
	TrackClicks   *bool  // Email: override click tracking
	SMSSenderID   string // SMS: sender ID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// WhatsAppCredentials is the decrypted form of Credentials for WhatsApp
type WhatsAppCredentials struct {
	AccessToken            string `json:"access_token"`
	AppSecret              string `json:"app_secret"`                         // for webhook signature verification
	WebhookSignatureBypass bool   `json:"webhook_signature_bypass,omitempty"` // development only; never enabled by default
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
	APIKey       string `json:"api_key"`                 // All providers
	Domain       string `json:"domain,omitempty"`        // Mailgun: sending domain
	ServerToken  string `json:"server_token,omitempty"`  // Postmark: server token
	WebhookToken string `json:"webhook_token,omitempty"` // For webhook signature verification

	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUsername string `json:"smtp_username,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	SMTPAuth     string `json:"smtp_auth,omitempty"`     // plain, login, none
	SMTPTLSMode  string `json:"smtp_tls_mode,omitempty"` // none, starttls, tls
}
