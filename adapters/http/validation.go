package http

import (
	"regexp"
	"strings"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
)

var (
	phoneRegex        = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
	emailRegex        = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	templateNameRegex = regexp.MustCompile(`^[a-z0-9_]+$`)
	idempotencyRegex  = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)
)

func validatePhone(phone string) (string, error) {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	if !phoneRegex.MatchString(phone) {
		return "", domain.NewValidationError("phone", "phone must be E.164 format (e.g. +919810914244)")
	}
	return phone, nil
}

func validateEmail(email string) (string, error) {
	if len(email) > 254 {
		return "", domain.NewValidationError("email", "invalid email address")
	}
	if !emailRegex.MatchString(email) {
		return "", domain.NewValidationError("email", "invalid email address")
	}
	return strings.ToLower(email), nil
}

func validateTags(tags []string) error {
	if len(tags) > 10 {
		return domain.NewValidationError("tags", "max 10 tags allowed, each max 50 chars")
	}
	for _, t := range tags {
		if len(t) > 50 {
			return domain.NewValidationError("tags", "max 10 tags allowed, each max 50 chars")
		}
	}
	return nil
}

func validateMetadata(metadata map[string]string) error {
	if len(metadata) > 20 {
		return domain.NewValidationError("metadata", "metadata: max 20 keys, key max 50, value max 500 chars")
	}
	for k, v := range metadata {
		if len(k) > 50 || len(v) > 500 {
			return domain.NewValidationError("metadata", "metadata: max 20 keys, key max 50, value max 500 chars")
		}
	}
	return nil
}

func validateChannel(channel domain.Channel) error {
	switch channel {
	case domain.ChannelWhatsApp, domain.ChannelSMS, domain.ChannelEmail:
		return nil
	default:
		return domain.NewValidationError("channel", "channel must be whatsapp, sms, or email")
	}
}

func validateIdempotencyKey(key string) error {
	if key == "" {
		return nil
	}
	if len(key) < 8 || len(key) > 128 {
		return domain.NewValidationError("idempotency_key", "idempotency key must be between 8 and 128 characters")
	}
	if !idempotencyRegex.MatchString(key) {
		return domain.NewValidationError("idempotency_key", "idempotency key may contain letters, numbers, dots, underscores, colons, and hyphens")
	}
	return nil
}

func validateTemplateName(name string) error {
	if len(name) < 1 || len(name) > 512 {
		return domain.NewValidationError("name", "template name must be lowercase alphanumeric with underscores")
	}
	if !templateNameRegex.MatchString(name) {
		return domain.NewValidationError("name", "template name must be lowercase alphanumeric with underscores")
	}
	return nil
}

func validateTemplateCategory(cat domain.TemplateCategory) error {
	switch cat {
	case domain.TemplateCategoryMarketing, domain.TemplateCategoryUtility, domain.TemplateCategoryAuthentication:
		return nil
	default:
		return domain.NewValidationError("category", "category must be marketing, utility, or authentication")
	}
}

func validateCampaignName(name string) error {
	if len(name) < 1 || len(name) > 255 {
		return domain.NewValidationError("name", "campaign name is required (max 255 chars)")
	}
	return nil
}

func validateScheduledAt(scheduledAt *time.Time) error {
	if scheduledAt == nil {
		return nil
	}
	if scheduledAt.Before(time.Now().Add(5 * time.Minute)) {
		return domain.NewValidationError("scheduled_at", "scheduled_at must be at least 5 minutes in the future")
	}
	return nil
}

func validatePagination(page, perPage int) (int, int, error) {
	if page < 1 {
		return 1, 20, domain.NewValidationError("page", "page must be >= 1")
	}
	if perPage < 1 || perPage > 100 {
		return 1, 20, domain.NewValidationError("per_page", "per_page must be between 1 and 100")
	}
	return page, perPage, nil
}

func validateEmailTemplate(name, subject, htmlBody string) error {
	if name == "" {
		return domain.NewValidationError("name", "template name is required")
	}
	if len(subject) < 1 || len(subject) > 998 {
		return domain.NewValidationError("subject", "subject is required (max 998 chars)")
	}
	if len(htmlBody) < 1 {
		return domain.NewValidationError("html_body", "html_body is required (max 5MB)")
	}
	if len(htmlBody) > 5*1024*1024 {
		return domain.NewValidationError("html_body", "html_body is required (max 5MB)")
	}
	return nil
}

func validateAttachments(attachments []domain.EmailAttachment) error {
	if len(attachments) > 10 {
		return domain.NewValidationError("attachments", "max 10 attachments allowed")
	}
	for _, a := range attachments {
		// content is base64
		// 10MB limit. 10MB in base64 is roughly 13.3MB
		if len(a.Content) > 10*1024*1024*4/3 {
			return domain.NewValidationError("attachment", "attachment exceeds 10MB limit")
		}
		if a.ContentType == "" {
			return domain.NewValidationError("content_type", "content_type must be a valid MIME type")
		}
	}
	return nil
}

func validateSegmentRules(rules []domain.SegmentRule) error {
	if len(rules) > 10 {
		return domain.NewValidationError("rules", "max 10 rules per segment")
	}
	validOps := map[string]bool{
		"eq":       true,
		"neq":      true,
		"contains": true,
		"gt":       true,
		"lt":       true,
		"in":       true,
	}
	for _, r := range rules {
		if !validOps[r.Operator] {
			return domain.NewValidationError("operator", "operator must be: eq, neq, contains, gt, lt, in")
		}
	}
	return nil
}
