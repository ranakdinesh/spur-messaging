package domain

import (
	"time"

	"github.com/google/uuid"
)

type TemplateStatus string

const (
	TemplateStatusDraft    TemplateStatus = "draft"
	TemplateStatusPending  TemplateStatus = "pending"
	TemplateStatusApproved TemplateStatus = "approved"
	TemplateStatusRejected TemplateStatus = "rejected"
)

type TemplateCategory string

const (
	TemplateCategoryMarketing      TemplateCategory = "marketing"
	TemplateCategoryUtility        TemplateCategory = "utility"
	TemplateCategoryAuthentication TemplateCategory = "authentication"
)

type Template struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	Channel            Channel
	Name               string // alphanumeric + underscore, lowercase
	Language           string // BCP 47: en, en_US, hi, ar
	Category           TemplateCategory
	Components         []TemplateComponent // header, body, footer, buttons
	Status             TemplateStatus
	ProviderTemplateID *string // Meta's template ID after submission
	RejectionReason    *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type TemplateComponentType string

const (
	ComponentHeader  TemplateComponentType = "HEADER"
	ComponentBody    TemplateComponentType = "BODY"
	ComponentFooter  TemplateComponentType = "FOOTER"
	ComponentButtons TemplateComponentType = "BUTTONS"
)

type TemplateComponent struct {
	Type    TemplateComponentType `json:"type"`
	Format  *string               `json:"format,omitempty"` // TEXT, IMAGE, VIDEO, DOCUMENT (header only)
	Text    *string               `json:"text,omitempty"`
	Example *TemplateExample      `json:"example,omitempty"`
	Buttons []TemplateButton      `json:"buttons,omitempty"`
}

type TemplateButton struct {
	Type        string  `json:"type"` // QUICK_REPLY, URL, PHONE_NUMBER
	Text        string  `json:"text"`
	URL         *string `json:"url,omitempty"`
	PhoneNumber *string `json:"phone_number,omitempty"`
}

type TemplateExample struct {
	HeaderHandle []string   `json:"header_handle,omitempty"`
	BodyText     [][]string `json:"body_text,omitempty"`
}
