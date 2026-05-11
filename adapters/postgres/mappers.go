package postgres

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ranakdinesh/spur-messaging/adapters/postgres/gen"
	"github.com/ranakdinesh/spur-messaging/core/domain"
)

// Message mappers
func toMessageDomain(m gen.MessagingMessage) domain.Message {
	var templateParams map[string]string
	if m.TemplateParams != nil {
		_ = json.Unmarshal(m.TemplateParams, &templateParams)
	}

	var metadata map[string]string
	if m.Metadata != nil {
		_ = json.Unmarshal(m.Metadata, &metadata)
	}

	var cost *float64
	if m.Cost.Valid {
		c, _ := m.Cost.Float64Value()
		cost = &c.Float64
	}

	return domain.Message{
		ID:                m.ID,
		TenantID:          m.TenantID,
		CampaignID:        pgUUIDToPtr(m.CampaignID),
		ConversationID:    pgUUIDToPtr(m.ConversationID),
		Channel:           domain.Channel(m.Channel),
		Direction:         m.Direction,
		Recipient:         m.Recipient,
		Sender:            pgTextToString(m.Sender),
		MessageType:       domain.MessageType(m.MessageType),
		TemplateID:        pgUUIDToPtr(m.TemplateID),
		TemplateName:      pgTextToStringPtr(m.TemplateName),
		TemplateParams:    templateParams,
		TextBody:          pgTextToStringPtr(m.TextBody),
		MediaURL:          pgTextToStringPtr(m.MediaUrl),
		MediaType:         pgTextToStringPtr(m.MediaType),
		ProviderMessageID: pgTextToString(m.ProviderMessageID),
		IdempotencyKey:    pgTextToStringPtr(m.IdempotencyKey),
		Status:            domain.MessageStatus(m.Status),
		ErrorCode:         pgTextToStringPtr(m.ErrorCode),
		ErrorMessage:      pgTextToStringPtr(m.ErrorMessage),
		Cost:              cost,
		SentAt:            pgTimestamptzToPtr(m.SentAt),
		DeliveredAt:       pgTimestamptzToPtr(m.DeliveredAt),
		ReadAt:            pgTimestamptzToPtr(m.ReadAt),
		FailedAt:          pgTimestamptzToPtr(m.FailedAt),
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		Metadata:          metadata,
	}
}

// Template mappers
func toTemplateDomain(t gen.MessagingTemplate) domain.Template {
	var components []domain.TemplateComponent
	if t.Components != nil {
		_ = json.Unmarshal(t.Components, &components)
	}

	return domain.Template{
		ID:                 t.ID,
		TenantID:           t.TenantID,
		Channel:            domain.Channel(t.Channel),
		Name:               t.Name,
		Language:           t.Language,
		Category:           domain.TemplateCategory(t.Category),
		Components:         components,
		Status:             domain.TemplateStatus(t.Status),
		ProviderTemplateID: pgTextToStringPtr(t.ProviderTemplateID),
		RejectionReason:    pgTextToStringPtr(t.RejectionReason),
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
	}
}

// Contact mappers
func toContactDomain(c gen.MessagingContact) domain.Contact {
	var attributes map[string]string
	if c.Attributes != nil {
		_ = json.Unmarshal(c.Attributes, &attributes)
	}

	return domain.Contact{
		ID:            c.ID,
		TenantID:      c.TenantID,
		Phone:         pgTextToStringPtr(c.Phone),
		Email:         pgTextToStringPtr(c.Email),
		Name:          pgTextToStringPtr(c.Name),
		Attributes:    attributes,
		Tags:          c.Tags,
		OptInWhatsApp: domain.OptInStatus(c.OptInWhatsapp),
		OptInSMS:      domain.OptInStatus(c.OptInSms),
		OptInEmail:    domain.OptInStatus(c.OptInEmail),
		OptedInAt:     pgTimestamptzToPtr(c.OptedInAt),
		OptedOutAt:    pgTimestamptzToPtr(c.OptedOutAt),
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func toConsentRecordDomain(r gen.MessagingConsentRecord) domain.ConsentRecord {
	return domain.ConsentRecord{
		ID:        r.ID,
		TenantID:  r.TenantID,
		ContactID: r.ContactID,
		Channel:   domain.Channel(r.Channel),
		Status:    domain.OptInStatus(r.Status),
		Source:    r.Source,
		Purpose:   r.Purpose,
		Proof:     r.Proof,
		IPAddress: r.IpAddress,
		UserAgent: r.UserAgent,
		Brand:     r.Brand,
		CreatedAt: r.CreatedAt,
	}
}

func toConversationDomain(c gen.MessagingConversation) domain.Conversation {
	return domain.Conversation{
		ID:                 c.ID,
		TenantID:           c.TenantID,
		Channel:            domain.Channel(c.Channel),
		Recipient:          c.Recipient,
		Status:             domain.ConversationStatus(c.Status),
		HandoffStatus:      domain.ConversationHandoffStatus(c.HandoffStatus),
		LastInboundAt:      pgTimestamptzToPtr(c.LastInboundAt),
		LastOutboundAt:     pgTimestamptzToPtr(c.LastOutboundAt),
		ServiceWindowUntil: pgTimestamptzToPtr(c.ServiceWindowUntil),
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

// Campaign mappers
func toCampaignDomain(c gen.MessagingCampaign) domain.Campaign {
	var templateParams map[string]string
	if c.TemplateParams != nil {
		_ = json.Unmarshal(c.TemplateParams, &templateParams)
	}

	var stats domain.CampaignStats
	if c.Stats != nil {
		_ = json.Unmarshal(c.Stats, &stats)
	}

	return domain.Campaign{
		ID:             c.ID,
		TenantID:       c.TenantID,
		Name:           c.Name,
		Channel:        domain.Channel(c.Channel),
		TemplateID:     c.TemplateID,
		TemplateParams: templateParams,
		SegmentID:      pgUUIDToPtr(c.SegmentID),
		ContactIDs:     c.ContactIds,
		ScheduledAt:    pgTimestamptzToPtr(c.ScheduledAt),
		StartedAt:      pgTimestamptzToPtr(c.StartedAt),
		CompletedAt:    pgTimestamptzToPtr(c.CompletedAt),
		Status:         domain.CampaignStatus(c.Status),
		Stats:          stats,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

// ProviderConfig mappers
func toProviderConfigDomain(p gen.MessagingProviderConfig) domain.ProviderConfig {
	return domain.ProviderConfig{
		ID:            p.ID,
		TenantID:      p.TenantID,
		Channel:       domain.Channel(p.Channel),
		Provider:      p.Provider,
		Credentials:   p.Credentials,
		WebhookSecret: pgTextToString(p.WebhookSecret),
		IsActive:      p.IsActive,
		PhoneNumberID: pgTextToString(p.PhoneNumberID),
		WABAID:        pgTextToString(p.WabaID),
		BusinessID:    pgTextToString(p.BusinessID),
		DisplayPhone:  pgTextToString(p.DisplayPhone),
		FromEmail:     pgTextToString(p.FromEmail),
		FromName:      pgTextToString(p.FromName),
		ReplyToEmail:  pgTextToString(p.ReplyToEmail),
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

// Segment mappers
func toSegmentDomain(s gen.MessagingSegment) domain.Segment {
	var rules []domain.SegmentRule
	if s.Rules != nil {
		_ = json.Unmarshal(s.Rules, &rules)
	}

	return domain.Segment{
		ID:        s.ID,
		TenantID:  s.TenantID,
		Name:      s.Name,
		IsDynamic: s.IsDynamic,
		Rules:     rules,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// EmailTemplate mappers
func toEmailTemplateDomain(t gen.MessagingEmailTemplate) domain.EmailTemplate {
	return domain.EmailTemplate{
		ID:          t.ID,
		TenantID:    t.TenantID,
		Name:        t.Name,
		Subject:     t.Subject,
		PreviewText: t.PreviewText,
		HTMLBody:    t.HtmlBody,
		TextBody:    t.TextBody,
		Category:    domain.EmailCategory(t.Category),
		Variables:   t.Variables,
		IsActive:    t.IsActive,
		Version:     int(t.Version),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// EmailEvent mappers
func toEmailEventDomain(e gen.MessagingEmailEvent) domain.EmailEvent {
	var rawPayload map[string]string
	if e.RawPayload != nil {
		_ = json.Unmarshal(e.RawPayload, &rawPayload)
	}

	return domain.EmailEvent{
		ID:                e.ID,
		TenantID:          e.TenantID,
		MessageID:         e.MessageID,
		CampaignID:        pgUUIDToPtr(e.CampaignID),
		EventType:         domain.EmailEventType(e.EventType),
		Recipient:         e.Recipient,
		Timestamp:         e.Timestamp,
		ProviderEventID:   pgTextToString(e.ProviderEventID),
		UserAgent:         pgTextToString(e.UserAgent),
		IPAddress:         pgTextToString(e.IpAddress),
		URL:               pgTextToString(e.Url),
		BounceType:        pgTextToStringPtr(e.BounceType),
		BounceReason:      pgTextToStringPtr(e.BounceReason),
		ComplaintFeedback: pgTextToStringPtr(e.ComplaintFeedback),
		RawPayload:        rawPayload,
		CreatedAt:         e.CreatedAt,
	}
}

// Unsubscribe mappers
func toUnsubscribeDomain(u gen.MessagingUnsubscribe) domain.Unsubscribe {
	return domain.Unsubscribe{
		ID:         u.ID,
		TenantID:   u.TenantID,
		Email:      u.Email,
		Scope:      domain.UnsubscribeScope(u.Scope),
		CampaignID: pgUUIDToPtr(u.CampaignID),
		Reason:     u.Reason,
		CreatedAt:  u.CreatedAt,
	}
}

// SuppressionEntry mappers
func toSuppressionEntryDomain(s gen.MessagingSuppression) domain.SuppressionEntry {
	return domain.SuppressionEntry{
		ID:        s.ID,
		TenantID:  s.TenantID,
		Channel:   domain.Channel(s.Channel),
		Recipient: s.Recipient,
		Email:     pgTextToString(s.Email),
		Reason:    domain.SuppressionReason(s.Reason),
		Source:    s.Source,
		CreatedAt: s.CreatedAt,
	}
}

// PG Helper functions
func pgUUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	res := uuid.UUID(u.Bytes)
	return &res
}

func pgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func pgTextToStringPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	res := t.String
	return &res
}

func nullableStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func pgTimestamptzToPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	res := t.Time
	return &res
}

func fromUUIDPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func fromStringPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func fromTimePtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func toMessageCreateSQLC(m *domain.Message) gen.CreateMessageParams {
	tp, _ := json.Marshal(m.TemplateParams)
	meta, _ := json.Marshal(m.Metadata)

	return gen.CreateMessageParams{
		TenantID:          m.TenantID,
		CampaignID:        fromUUIDPtr(m.CampaignID),
		ConversationID:    fromUUIDPtr(m.ConversationID),
		Channel:           string(m.Channel),
		Direction:         m.Direction,
		Recipient:         m.Recipient,
		Sender:            fromString(m.Sender),
		MessageType:       string(m.MessageType),
		TemplateID:        fromUUIDPtr(m.TemplateID),
		TemplateName:      fromStringPtr(m.TemplateName),
		TemplateParams:    tp,
		TextBody:          fromStringPtr(m.TextBody),
		MediaUrl:          fromStringPtr(m.MediaURL),
		MediaType:         fromStringPtr(m.MediaType),
		ProviderMessageID: fromString(m.ProviderMessageID),
		IdempotencyKey:    fromStringPtr(m.IdempotencyKey),
		Status:            string(m.Status),
		ErrorCode:         fromStringPtr(m.ErrorCode),
		ErrorMessage:      fromStringPtr(m.ErrorMessage),
		Metadata:          meta,
	}
}
