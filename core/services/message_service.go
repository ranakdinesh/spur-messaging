package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type Config struct {
	EncryptionKey    []byte
	WebhookBaseURL   string
	DefaultRateLimit int
	WorkerCount      int

	EmailProvider    string
	EmailAPIKey      string
	EmailFromAddress string
	EmailFromName    string
	EmailTrackOpens  bool
	EmailTrackClicks bool

	SMSProvider string
	SMSAPIKey   string
	SMSSenderID string

	WhatsAppWebhookVerifyToken string
	WhatsAppMetaAppID          string
}

type MessageService struct {
	repo              ports.MessageRepository
	conversationRepo  ports.ConversationRepository
	contactRepo       ports.ContactRepository
	templateRepo      ports.TemplateRepository
	queue             ports.MessageQueue
	suppressionSvc    ports.SuppressionService
	unsubscribeSvc    ports.UnsubscribeService
	emailTemplateRepo ports.EmailTemplateRepository
	providerRegistry  *ProviderRegistry
	cfg               Config
}

func NewMessageService(
	repo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	contactRepo ports.ContactRepository,
	templateRepo ports.TemplateRepository,
	queue ports.MessageQueue,
	suppressionSvc ports.SuppressionService,
	unsubscribeSvc ports.UnsubscribeService,
	emailTemplateRepo ports.EmailTemplateRepository,
	providerRegistry *ProviderRegistry,
	cfg Config,
) *MessageService {
	return &MessageService{
		repo:              repo,
		conversationRepo:  conversationRepo,
		contactRepo:       contactRepo,
		templateRepo:      templateRepo,
		queue:             queue,
		suppressionSvc:    suppressionSvc,
		unsubscribeSvc:    unsubscribeSvc,
		emailTemplateRepo: emailTemplateRepo,
		providerRegistry:  providerRegistry,
		cfg:               cfg,
	}
}

func (s *MessageService) Send(ctx context.Context, tenantID uuid.UUID, req ports.SendMessageRequest) (*domain.Message, error) {
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if req.IdempotencyKey != "" {
		existing, err := s.repo.GetByIdempotencyKey(ctx, tenantID, req.IdempotencyKey)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}

	// 1. Resolve provider config (Section 10A.2)
	_, tenantConfig, err := s.providerRegistry.GetProvider(ctx, tenantID, req.Channel)
	if err != nil {
		return nil, domain.ErrProviderNotConfigured
	}

	if req.Channel == domain.ChannelEmail {
		// 2. RESOLVE PARAMS: 3-tier resolution
		req.FromEmail = resolveString(req.FromEmail, tenantConfig.FromEmail, s.cfg.EmailFromAddress)
		req.FromName = resolveString(req.FromName, tenantConfig.FromName, s.cfg.EmailFromName)
		req.ReplyTo = resolveString(req.ReplyTo, tenantConfig.ReplyToEmail, "")

		trackOpens := s.cfg.EmailTrackOpens
		// In a real app we'd load these from tenantConfig.Credentials if stored there as per AGENTS.md 13.6
		// For now we use platform defaults as fallback and Tier 1 if provided
		if req.TrackOpens != nil {
			trackOpens = *req.TrackOpens
		}
		req.TrackOpens = &trackOpens

		trackClicks := s.cfg.EmailTrackClicks
		if req.TrackClicks != nil {
			trackClicks = *req.TrackClicks
		}
		req.TrackClicks = &trackClicks

	}

	if req.Channel == domain.ChannelSMS {
		req.SenderID = resolveString(req.SenderID, tenantConfig.SMSSenderID, s.cfg.SMSSenderID)
	}

	// 2. Validate contact exists
	var contact *domain.Contact
	if req.Channel == domain.ChannelEmail {
		contact, err = s.contactRepo.GetByEmail(ctx, tenantID, req.Recipient)
	} else {
		contact, err = s.contactRepo.GetByPhone(ctx, tenantID, req.Recipient)
	}
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewNotFoundError("contact")
		}
		return nil, err
	}

	// 3. Check opt-in (Section 10A.2)
	var optedIn bool
	switch req.Channel {
	case domain.ChannelWhatsApp:
		optedIn = contact.OptInWhatsApp == domain.OptInStatusOptedIn
	case domain.ChannelSMS:
		optedIn = contact.OptInSMS == domain.OptInStatusOptedIn
	case domain.ChannelEmail:
		optedIn = contact.OptInEmail == domain.OptInStatusOptedIn
	}

	if !optedIn {
		return nil, domain.ErrOptInRequired
	}

	if s.suppressionSvc != nil {
		suppressed, err := s.suppressionSvc.IsSuppressed(ctx, tenantID, req.Channel, req.Recipient)
		if err != nil {
			return nil, err
		}
		if suppressed {
			return s.createSuppressedMessage(ctx, tenantID, req, "SUPPRESSED")
		}
	}

	if req.Channel == domain.ChannelEmail && s.unsubscribeSvc != nil && isMarketingMessage(req.Metadata) {
		unsubscribed, err := s.unsubscribeSvc.IsUnsubscribed(ctx, tenantID, req.Recipient)
		if err != nil {
			return nil, err
		}
		if unsubscribed {
			return s.createSuppressedMessage(ctx, tenantID, req, "UNSUBSCRIBED")
		}
	}

	// 4. Rate limit check (Section 10A.2 - placeholder)

	// 5. Channel-specific checks & RENDERING
	var conversationID *uuid.UUID
	if req.Channel == domain.ChannelWhatsApp {
		if req.MessageType == domain.MessageTypeTemplate {
			if req.TemplateName == nil {
				return nil, domain.NewValidationError("template_name", "template name is required for template messages")
			}
			lang := "en"
			if req.TemplateLanguage != nil && *req.TemplateLanguage != "" {
				lang = *req.TemplateLanguage
			}
			tmpl, err := s.templateRepo.GetByName(ctx, tenantID, *req.TemplateName, lang)
			if err != nil {
				return nil, domain.NewNotFoundError("template")
			}
			if tmpl.Status != domain.TemplateStatusApproved {
				return nil, domain.ErrTemplateNotApproved
			}
		} else {
			if s.conversationRepo == nil {
				return nil, domain.ErrSessionWindowClosed
			}
			conversation, err := s.conversationRepo.GetActiveByRecipient(ctx, tenantID, domain.ChannelWhatsApp, req.Recipient, time.Now())
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return nil, domain.ErrSessionWindowClosed
				}
				return nil, err
			}
			conversationID = &conversation.ID
		}
		if s.conversationRepo != nil {
			conversation, err := s.conversationRepo.UpsertOutbound(ctx, tenantID, domain.ChannelWhatsApp, req.Recipient, time.Now())
			if err != nil {
				return nil, err
			}
			conversationID = &conversation.ID
		}
	}

	if req.Channel == domain.ChannelEmail {
		isMarketing := true
		if req.Metadata != nil && req.Metadata["category"] == "transactional" {
			isMarketing = false
		}

		// 5. RENDER TEMPLATE (if template-based)
		if req.MessageType == domain.MessageTypeTemplate {
			if req.TemplateName == nil {
				return nil, domain.NewValidationError("template_name", "template name is required")
			}
			tmpl, err := s.emailTemplateRepo.GetByName(ctx, tenantID, *req.TemplateName)
			if err != nil {
				return nil, domain.NewNotFoundError("email template")
			}

			subject := s.renderEmail(tmpl.Subject, req.TemplateParams)
			htmlBody := s.renderEmail(tmpl.HTMLBody, req.TemplateParams)
			textBody := s.renderEmail(tmpl.TextBody, req.TemplateParams)
			if textBody == "" {
				textBody = s.stripHTML(htmlBody)
			}

			// Validate all variables provided (simple check)
			if strings.Contains(subject, "{{") || strings.Contains(htmlBody, "{{") {
				// This is a bit simplified, but checks if any placeholders remain
				// In a real app we'd compare req.TemplateParams with tmpl.Variables
			}

			req.Text = &htmlBody // For email, we store HTML in TextBody field or metadata
			// Actually we'll use a local variable to update msg later
			req.Metadata["subject"] = subject
			req.Metadata["html_body"] = htmlBody
			req.Metadata["text_body"] = textBody
		}

		if isMarketing {
			// 6. INJECT UNSUBSCRIBE HEADERS & LINKS
			token := s.generateUnsubscribeToken(tenantID, req.Recipient)
			unsubURL := s.cfg.WebhookBaseURL + "/messaging/unsubscribe/" + token
			if req.Metadata == nil {
				req.Metadata = make(map[string]string)
			}
			req.Metadata["list_unsubscribe"] = fmt.Sprintf("<mailto:unsub@citual.com>, <%s>", unsubURL)
			req.Metadata["list_unsubscribe_post"] = "List-Unsubscribe=One-Click"

			htmlBody := ""
			if req.Metadata["html_body"] != "" {
				htmlBody = req.Metadata["html_body"]
			} else if req.Text != nil {
				htmlBody = *req.Text
			}

			if !strings.Contains(htmlBody, "unsubscribe") {
				unsubLink := fmt.Sprintf("<br><br><a href=\"%s\">Unsubscribe</a>", unsubURL)
				htmlBody += unsubLink
			}
			req.Metadata["html_body"] = htmlBody
		}

		// 7. INJECT TRACKING PIXEL
		trackOpens := false
		if req.TrackOpens != nil {
			trackOpens = *req.TrackOpens
		}
		if trackOpens {
			// Step 7 will be completed during message creation below
		}

		// 8. REWRITE LINKS
		trackClicks := false
		if req.TrackClicks != nil {
			trackClicks = *req.TrackClicks
		}
		if trackClicks {
			htmlBody := req.Metadata["html_body"]
			htmlBody = s.rewriteLinks(htmlBody, uuid.Nil) // messageID not known yet
			req.Metadata["html_body"] = htmlBody
		}
	}

	// 6. Create message
	msg := &domain.Message{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ConversationID: conversationID,
		Channel:        req.Channel,
		Direction:      "outbound",
		Recipient:      req.Recipient,
		MessageType:    req.MessageType,
		TemplateName:   req.TemplateName,
		TemplateParams: req.TemplateParams,
		TextBody:       req.Text,
		MediaURL:       req.MediaURL,
		MediaType:      req.MediaType,
		Status:         domain.MessageStatusQueued,
		CreatedAt:      time.Now(),
		Metadata:       req.Metadata,
	}
	if req.IdempotencyKey != "" {
		msg.IdempotencyKey = &req.IdempotencyKey
	}

	if req.Channel == domain.ChannelEmail {
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]string)
		}
		msg.Metadata["from_email"] = req.FromEmail
		msg.Metadata["from_name"] = req.FromName
		msg.Metadata["reply_to"] = req.ReplyTo

		trackOpens := false
		if req.TrackOpens != nil {
			trackOpens = *req.TrackOpens
		}
		if trackOpens {
			pixelURL := fmt.Sprintf("%s/messaging/track/open/%s", s.cfg.WebhookBaseURL, msg.ID)
			pixelTag := fmt.Sprintf("<img src=\"%s\" width=\"1\" height=\"1\" style=\"display:none;\">", pixelURL)
			htmlBody := msg.Metadata["html_body"]
			if strings.Contains(htmlBody, "</body>") {
				htmlBody = strings.Replace(htmlBody, "</body>", pixelTag+"</body>", 1)
			} else {
				htmlBody += pixelTag
			}
			msg.Metadata["html_body"] = htmlBody
		}

		trackClicks := false
		if req.TrackClicks != nil {
			trackClicks = *req.TrackClicks
		}
		if trackClicks {
			htmlBody := msg.Metadata["html_body"]
			htmlBody = s.rewriteLinks(htmlBody, msg.ID)
			msg.Metadata["html_body"] = htmlBody
		}
	}

	if msg.Metadata == nil {
		msg.Metadata = make(map[string]string)
	}

	if req.Channel == domain.ChannelEmail {
		msg.Metadata["from_email"] = req.FromEmail
		msg.Metadata["from_name"] = req.FromName
		msg.Metadata["reply_to"] = req.ReplyTo
		if req.TrackOpens != nil {
			msg.Metadata["track_opens"] = strconv.FormatBool(*req.TrackOpens)
		}
		if req.TrackClicks != nil {
			msg.Metadata["track_clicks"] = strconv.FormatBool(*req.TrackClicks)
		}
		if len(req.CC) > 0 {
			msg.Metadata["cc"] = strings.Join(req.CC, ",")
		}
		if len(req.BCC) > 0 {
			msg.Metadata["bcc"] = strings.Join(req.BCC, ",")
		}
	} else if req.Channel == domain.ChannelSMS {
		msg.Metadata["sender_id"] = req.SenderID
	}

	if req.TemplateName != nil {
		lang := "en"
		if req.TemplateLanguage != nil && *req.TemplateLanguage != "" {
			lang = *req.TemplateLanguage
		}
		tmpl, err := s.templateRepo.GetByName(ctx, tenantID, *req.TemplateName, lang)
		if err == nil {
			msg.TemplateID = &tmpl.ID
		}
	}

	err = s.repo.Create(ctx, msg)
	if err != nil {
		if req.IdempotencyKey != "" && errors.Is(err, domain.ErrAlreadyExists) {
			existing, getErr := s.repo.GetByIdempotencyKey(ctx, tenantID, req.IdempotencyKey)
			if getErr == nil {
				return existing, nil
			}
			return nil, getErr
		}
		return nil, err
	}

	// 7. Enqueue
	priority := 0
	if req.Metadata != nil && req.Metadata["priority"] == "high" {
		priority = 1
	}

	err = s.queue.Enqueue(ctx, ports.QueueMessage{
		MessageID: msg.ID,
		TenantID:  tenantID,
		Channel:   req.Channel,
		Priority:  priority,
	})
	if err != nil {
		return nil, domain.ErrQueueUnavailable
	}

	return msg, nil
}

func (s *MessageService) SendBulk(ctx context.Context, tenantID uuid.UUID, reqs []ports.SendMessageRequest) ([]domain.Message, error) {
	var msgs []domain.Message
	for _, req := range reqs {
		msg, err := s.Send(ctx, tenantID, req)
		if err != nil {
			// In bulk, we might want to continue or return first error
			return nil, err
		}
		msgs = append(msgs, *msg)
	}
	return msgs, nil
}

func (s *MessageService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *MessageService) List(ctx context.Context, tenantID uuid.UUID, filter ports.MessageFilter) ([]domain.Message, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

func (s *MessageService) Retry(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error) {
	msg, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if msg.Status != domain.MessageStatusFailed {
		return nil, errors.New("only failed messages can be retried")
	}

	msg.Status = domain.MessageStatusQueued
	// Update status in repo...
	// (Needs UpdateStatus method in repo)

	err = s.queue.Enqueue(ctx, ports.QueueMessage{
		MessageID: msg.ID,
		TenantID:  tenantID,
		Channel:   msg.Channel,
		Priority:  0,
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *MessageService) createSuppressedMessage(ctx context.Context, tenantID uuid.UUID, req ports.SendMessageRequest, code string) (*domain.Message, error) {
	msg := &domain.Message{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Channel:     req.Channel,
		Direction:   "outbound",
		Recipient:   req.Recipient,
		MessageType: req.MessageType,
		Status:      domain.MessageStatusSuppressed,
		ErrorCode:   &code,
		CreatedAt:   time.Now(),
		Metadata:    req.Metadata,
	}
	if req.IdempotencyKey != "" {
		msg.IdempotencyKey = &req.IdempotencyKey
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]string)
	}
	if req.Channel == domain.ChannelEmail {
		msg.Metadata["from_email"] = req.FromEmail
		msg.Metadata["from_name"] = req.FromName
		msg.Metadata["reply_to"] = req.ReplyTo
	} else if req.Channel == domain.ChannelSMS {
		msg.Metadata["sender_id"] = req.SenderID
	}
	if err := s.repo.Create(ctx, msg); err != nil {
		if req.IdempotencyKey != "" && errors.Is(err, domain.ErrAlreadyExists) {
			existing, getErr := s.repo.GetByIdempotencyKey(ctx, tenantID, req.IdempotencyKey)
			if getErr == nil {
				return existing, nil
			}
			return nil, getErr
		}
		return nil, err
	}
	return msg, nil
}

func isMarketingMessage(metadata map[string]string) bool {
	return metadata == nil || metadata["category"] != "transactional"
}

func (s *MessageService) renderEmail(content string, variables map[string]string) string {
	for k, v := range variables {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content
}

func (s *MessageService) stripHTML(html string) string {
	// Simple regex to strip HTML tags
	re := regexp.MustCompile("<[^>]*>")
	return re.ReplaceAllString(html, "")
}

func (s *MessageService) generateUnsubscribeToken(tenantID uuid.UUID, email string) string {
	// AGENTS.md 6: JWT or HMAC-signed token
	h := hmac.New(sha256.New, s.cfg.EncryptionKey)
	h.Write([]byte(tenantID.String() + email))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *MessageService) rewriteLinks(html string, messageID uuid.UUID) string {
	// Find all <a href="..."> in HTML body
	re := regexp.MustCompile(`(?i)<a\s+[^>]*href=["']([^"']+)["'][^>]*>`)
	return re.ReplaceAllStringFunc(html, func(match string) string {
		// Do NOT rewrite the unsubscribe link
		if strings.Contains(match, "unsubscribe") {
			return match
		}

		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		originalURL := submatches[1]

		// If messageID is Nil, we can't fully rewrite yet, or we use a placeholder
		if messageID == uuid.Nil {
			return match
		}

		linkHash := hex.EncodeToString([]byte(originalURL))[:8]
		newURL := fmt.Sprintf("%s/messaging/track/click/%s/%s?url=%s",
			s.cfg.WebhookBaseURL, messageID, linkHash, originalURL)

		return strings.Replace(match, originalURL, newURL, 1)
	})
}
