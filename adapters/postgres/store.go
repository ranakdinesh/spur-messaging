package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ranakdinesh/spur-messaging/adapters/postgres/gen"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type Store struct {
	db *pgxpool.Pool
	q  *gen.Queries
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db: db,
		q:  gen.New(db),
	}
}

// MessageRepository
func (s *Store) Create(ctx context.Context, msg *domain.Message) error {
	params := toMessageCreateSQLC(msg)
	m, err := s.q.CreateMessage(ctx, params)
	if err != nil {
		if isIdempotencyConflict(err) {
			return domain.ErrAlreadyExists
		}
		return err
	}
	*msg = toMessageDomain(m)
	return nil
}

func (s *Store) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error) {
	m, err := s.q.GetMessageByID(ctx, gen.GetMessageByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toMessageDomain(m)
	return &res, nil
}

func (s *Store) GetByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*domain.Message, error) {
	m, err := s.q.GetMessageByIdempotencyKey(ctx, gen.GetMessageByIdempotencyKeyParams{
		TenantID:       tenantID,
		IdempotencyKey: fromString(key),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toMessageDomain(m)
	return &res, nil
}

func (s *Store) List(ctx context.Context, tenantID uuid.UUID, filter ports.MessageFilter) ([]domain.Message, int, error) {
	arg := gen.ListMessagesParams{
		TenantID: tenantID,
		Column2:  derefString(fromStringPtr((*string)(filter.Channel))),
		Column3:  derefString(fromStringPtr((*string)(filter.Status))),
		Column4:  derefString(fromStringPtr(filter.Recipient)),
		Column5:  fromUUIDPtr(filter.CampaignID).Bytes,
		Column6:  fromTimePtr(filter.DateFrom).Time,
		Column7:  fromTimePtr(filter.DateTo).Time,
		Limit:    int32(filter.PerPage),
		Offset:   int32(filter.Page * filter.PerPage),
	}
	rows, err := s.q.ListMessages(ctx, arg)
	if err != nil {
		return nil, 0, err
	}
	var res []domain.Message
	total := 0
	for _, r := range rows {
		res = append(res, toMessageDomain(gen.MessagingMessage{
			ID:                r.ID,
			TenantID:          r.TenantID,
			CampaignID:        r.CampaignID,
			ConversationID:    r.ConversationID,
			Channel:           r.Channel,
			Direction:         r.Direction,
			Recipient:         r.Recipient,
			Sender:            r.Sender,
			MessageType:       r.MessageType,
			TemplateID:        r.TemplateID,
			TemplateName:      r.TemplateName,
			TemplateParams:    r.TemplateParams,
			TextBody:          r.TextBody,
			MediaUrl:          r.MediaUrl,
			MediaType:         r.MediaType,
			ProviderMessageID: r.ProviderMessageID,
			IdempotencyKey:    r.IdempotencyKey,
			Status:            r.Status,
			ErrorCode:         r.ErrorCode,
			ErrorMessage:      r.ErrorMessage,
			Cost:              r.Cost,
			SentAt:            r.SentAt,
			DeliveredAt:       r.DeliveredAt,
			ReadAt:            r.ReadAt,
			FailedAt:          r.FailedAt,
			CreatedAt:         r.CreatedAt,
			Metadata:          r.Metadata,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.MessageStatus, providerMsgID string) error {
	return s.q.UpdateMessageStatus(ctx, gen.UpdateMessageStatusParams{
		TenantID:          tenantID,
		ID:                id,
		Status:            string(status),
		ProviderMessageID: fromString(providerMsgID),
	})
}

func (s *Store) UpdateStatusByProviderID(ctx context.Context, providerMsgID string, status domain.MessageStatus, timestamp time.Time) error {
	return s.q.UpdateMessageStatusByProviderID(ctx, gen.UpdateMessageStatusByProviderIDParams{
		ProviderMessageID: fromString(providerMsgID),
		Status:            string(status),
		DeliveredAt:       fromTimePtr(&timestamp),
	})
}

func (s *Store) GetByProviderID(ctx context.Context, providerMsgID string) (*domain.Message, error) {
	m, err := s.q.GetMessageByProviderID(ctx, fromString(providerMsgID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toMessageDomain(m)
	return &res, nil
}

func (s *Store) GetByCampaignID(ctx context.Context, tenantID, campaignID uuid.UUID, page, perPage int) ([]domain.Message, int, error) {
	rows, err := s.q.GetMessagesByCampaignID(ctx, gen.GetMessagesByCampaignIDParams{
		TenantID:   tenantID,
		CampaignID: fromUUIDPtr(&campaignID),
		Limit:      int32(perPage),
		Offset:     int32(page * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.Message
	total := 0
	for _, r := range rows {
		res = append(res, toMessageDomain(gen.MessagingMessage{
			ID:                r.ID,
			TenantID:          r.TenantID,
			CampaignID:        r.CampaignID,
			ConversationID:    r.ConversationID,
			Channel:           r.Channel,
			Direction:         r.Direction,
			Recipient:         r.Recipient,
			Sender:            r.Sender,
			MessageType:       r.MessageType,
			TemplateID:        r.TemplateID,
			TemplateName:      r.TemplateName,
			TemplateParams:    r.TemplateParams,
			TextBody:          r.TextBody,
			MediaUrl:          r.MediaUrl,
			MediaType:         r.MediaType,
			ProviderMessageID: r.ProviderMessageID,
			IdempotencyKey:    r.IdempotencyKey,
			Status:            r.Status,
			ErrorCode:         r.ErrorCode,
			ErrorMessage:      r.ErrorMessage,
			Cost:              r.Cost,
			SentAt:            r.SentAt,
			DeliveredAt:       r.DeliveredAt,
			ReadAt:            r.ReadAt,
			FailedAt:          r.FailedAt,
			CreatedAt:         r.CreatedAt,
			Metadata:          r.Metadata,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) ExistsForCampaign(ctx context.Context, campaignID uuid.UUID, recipient string) (bool, error) {
	return s.q.CheckMessageExistsForCampaign(ctx, gen.CheckMessageExistsForCampaignParams{
		CampaignID: fromUUID(campaignID),
		Recipient:  recipient,
	})
}

func (s *Store) CountByCampaign(ctx context.Context, campaignID uuid.UUID) (int, error) {
	count, err := s.q.CountMessagesByCampaign(ctx, fromUUID(campaignID))
	return int(count), err
}

func isIdempotencyConflict(err error) bool {
	pgErr, ok := err.(*pgconn.PgError)
	return ok && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_messages_tenant_idempotency"
}

// TemplateRepository
func (s *Store) CreateTemplate(ctx context.Context, tmpl *domain.Template) error {
	comp, _ := json.Marshal(tmpl.Components)
	t, err := s.q.CreateTemplate(ctx, gen.CreateTemplateParams{
		TenantID:           tmpl.TenantID,
		Channel:            string(tmpl.Channel),
		Name:               tmpl.Name,
		Language:           tmpl.Language,
		Category:           string(tmpl.Category),
		Components:         comp,
		Status:             string(tmpl.Status),
		ProviderTemplateID: fromStringPtr(tmpl.ProviderTemplateID),
	})
	if err != nil {
		return err
	}
	*tmpl = toTemplateDomain(t)
	return nil
}

func (s *Store) GetTemplateByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	t, err := s.q.GetTemplateByID(ctx, gen.GetTemplateByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toTemplateDomain(t)
	return &res, nil
}

func (s *Store) GetTemplateByName(ctx context.Context, tenantID uuid.UUID, name, language string) (*domain.Template, error) {
	t, err := s.q.GetTemplateByName(ctx, gen.GetTemplateByNameParams{TenantID: tenantID, Name: name, Language: language})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toTemplateDomain(t)
	return &res, nil
}

func (s *Store) ListTemplates(ctx context.Context, tenantID uuid.UUID, channel *domain.Channel, status *domain.TemplateStatus, page, perPage int) ([]domain.Template, int, error) {
	rows, err := s.q.ListTemplates(ctx, gen.ListTemplatesParams{
		TenantID: tenantID,
		Column2:  derefString(fromStringPtr((*string)(channel))),
		Column3:  derefString(fromStringPtr((*string)(status))),
		Limit:    int32(perPage),
		Offset:   int32(page * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.Template
	total := 0
	for _, r := range rows {
		res = append(res, toTemplateDomain(gen.MessagingTemplate{
			ID:                 r.ID,
			TenantID:           r.TenantID,
			Channel:            r.Channel,
			Name:               r.Name,
			Language:           r.Language,
			Category:           r.Category,
			Components:         r.Components,
			Status:             r.Status,
			ProviderTemplateID: r.ProviderTemplateID,
			RejectionReason:    r.RejectionReason,
			CreatedAt:          r.CreatedAt,
			UpdatedAt:          r.UpdatedAt,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) UpdateTemplate(ctx context.Context, tmpl *domain.Template) error {
	comp, _ := json.Marshal(tmpl.Components)
	t, err := s.q.UpdateTemplate(ctx, gen.UpdateTemplateParams{
		TenantID:           tmpl.TenantID,
		ID:                 tmpl.ID,
		Category:           string(tmpl.Category),
		Components:         comp,
		Status:             string(tmpl.Status),
		ProviderTemplateID: fromStringPtr(tmpl.ProviderTemplateID),
	})
	if err != nil {
		return err
	}
	*tmpl = toTemplateDomain(t)
	return nil
}

func (s *Store) UpdateTemplateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.TemplateStatus, providerID *string, rejectionReason *string) error {
	return s.q.UpdateTemplateStatus(ctx, gen.UpdateTemplateStatusParams{
		TenantID:           tenantID,
		ID:                 id,
		Status:             string(status),
		ProviderTemplateID: fromStringPtr(providerID),
		RejectionReason:    fromStringPtr(rejectionReason),
	})
}

func (s *Store) DeleteTemplate(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteTemplate(ctx, gen.DeleteTemplateParams{TenantID: tenantID, ID: id})
}

// ContactRepository
func (s *Store) CreateContact(ctx context.Context, contact *domain.Contact) error {
	attr, _ := json.Marshal(contact.Attributes)
	c, err := s.q.CreateContact(ctx, gen.CreateContactParams{
		TenantID:      contact.TenantID,
		Phone:         fromStringPtr(contact.Phone),
		Email:         fromStringPtr(contact.Email),
		Name:          fromStringPtr(contact.Name),
		Attributes:    attr,
		Tags:          contact.Tags,
		OptInWhatsapp: string(contact.OptInWhatsApp),
		OptInSms:      string(contact.OptInSMS),
		OptInEmail:    string(contact.OptInEmail),
	})
	if err != nil {
		return err
	}
	*contact = toContactDomain(c)
	return nil
}

func (s *Store) GetContactByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Contact, error) {
	c, err := s.q.GetContactByID(ctx, gen.GetContactByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toContactDomain(c)
	return &res, nil
}

func (s *Store) GetContactByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*domain.Contact, error) {
	c, err := s.q.GetContactByPhone(ctx, gen.GetContactByPhoneParams{TenantID: tenantID, Phone: fromString(phone)})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toContactDomain(c)
	return &res, nil
}

func (s *Store) GetContactByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.Contact, error) {
	c, err := s.q.GetContactByEmail(ctx, gen.GetContactByEmailParams{TenantID: tenantID, Email: fromString(email)})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toContactDomain(c)
	return &res, nil
}

func (s *Store) ListContacts(ctx context.Context, tenantID uuid.UUID, filter ports.ContactFilter) ([]domain.Contact, int, error) {
	rows, err := s.q.ListContacts(ctx, gen.ListContactsParams{
		TenantID: tenantID,
		Column2:  derefString(fromStringPtr(filter.Phone)),
		Column3:  derefString(fromStringPtr(filter.Email)),
		Column4:  derefString(fromStringPtr(filter.Tag)),
		Column5:  derefString(fromStringPtr((*string)(filter.OptedIn))),
		Limit:    int32(filter.PerPage),
		Offset:   int32(filter.Page * filter.PerPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.Contact
	total := 0
	for _, r := range rows {
		res = append(res, toContactDomain(gen.MessagingContact{
			ID:            r.ID,
			TenantID:      r.TenantID,
			Phone:         r.Phone,
			Email:         r.Email,
			Name:          r.Name,
			Attributes:    r.Attributes,
			Tags:          r.Tags,
			OptInWhatsapp: r.OptInWhatsapp,
			OptInSms:      r.OptInSms,
			OptInEmail:    r.OptInEmail,
			OptedInAt:     r.OptedInAt,
			OptedOutAt:    r.OptedOutAt,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) UpdateContact(ctx context.Context, contact *domain.Contact) error {
	attr, _ := json.Marshal(contact.Attributes)
	c, err := s.q.UpdateContact(ctx, gen.UpdateContactParams{
		TenantID:      contact.TenantID,
		ID:            contact.ID,
		Phone:         fromStringPtr(contact.Phone),
		Email:         fromStringPtr(contact.Email),
		Name:          fromStringPtr(contact.Name),
		Attributes:    attr,
		Tags:          contact.Tags,
		OptInWhatsapp: string(contact.OptInWhatsApp),
		OptInSms:      string(contact.OptInSMS),
		OptInEmail:    string(contact.OptInEmail),
	})
	if err != nil {
		return err
	}
	*contact = toContactDomain(c)
	return nil
}

func (s *Store) DeleteContact(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteContact(ctx, gen.DeleteContactParams{TenantID: tenantID, ID: id})
}

func (s *Store) BulkCreate(ctx context.Context, contacts []domain.Contact) (int, error) {
	var params []gen.BulkCreateContactsParams
	for _, c := range contacts {
		attr, _ := json.Marshal(c.Attributes)
		params = append(params, gen.BulkCreateContactsParams{
			TenantID:      c.TenantID,
			Phone:         fromStringPtr(c.Phone),
			Email:         fromStringPtr(c.Email),
			Name:          fromStringPtr(c.Name),
			Attributes:    attr,
			Tags:          c.Tags,
			OptInWhatsapp: string(c.OptInWhatsApp),
			OptInSms:      string(c.OptInSMS),
			OptInEmail:    string(c.OptInEmail),
		})
	}
	count, err := s.q.BulkCreateContacts(ctx, params)
	return int(count), err
}

func (s *Store) UpdateOptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, status domain.OptInStatus) error {
	return s.q.UpdateOptIn(ctx, gen.UpdateOptInParams{
		TenantID: tenantID,
		ID:       id,
		Column3:  string(channel),
		Column4:  string(status),
	})
}

// CampaignRepository
func (s *Store) CreateCampaign(ctx context.Context, campaign *domain.Campaign) error {
	params, _ := json.Marshal(campaign.TemplateParams)
	c, err := s.q.CreateCampaign(ctx, gen.CreateCampaignParams{
		TenantID:       campaign.TenantID,
		Name:           campaign.Name,
		Channel:        string(campaign.Channel),
		TemplateID:     campaign.TemplateID,
		TemplateParams: params,
		SegmentID:      fromUUIDPtr(campaign.SegmentID),
		ContactIds:     campaign.ContactIDs,
		ScheduledAt:    fromTimePtr(campaign.ScheduledAt),
		Status:         string(campaign.Status),
	})
	if err != nil {
		return err
	}
	*campaign = toCampaignDomain(c)
	return nil
}

func (s *Store) GetCampaignByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Campaign, error) {
	c, err := s.q.GetCampaignByID(ctx, gen.GetCampaignByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toCampaignDomain(c)
	return &res, nil
}

func (s *Store) ListCampaigns(ctx context.Context, tenantID uuid.UUID, status *domain.CampaignStatus, page, perPage int) ([]domain.Campaign, int, error) {
	rows, err := s.q.ListCampaigns(ctx, gen.ListCampaignsParams{
		TenantID: tenantID,
		Column2:  derefString(fromStringPtr((*string)(status))),
		Limit:    int32(perPage),
		Offset:   int32(page * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.Campaign
	total := 0
	for _, r := range rows {
		res = append(res, toCampaignDomain(gen.MessagingCampaign{
			ID:             r.ID,
			TenantID:       r.TenantID,
			Name:           r.Name,
			Channel:        r.Channel,
			TemplateID:     r.TemplateID,
			TemplateParams: r.TemplateParams,
			SegmentID:      r.SegmentID,
			ContactIds:     r.ContactIds,
			ScheduledAt:    r.ScheduledAt,
			StartedAt:      r.StartedAt,
			CompletedAt:    r.CompletedAt,
			Status:         r.Status,
			Stats:          r.Stats,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) UpdateCampaign(ctx context.Context, campaign *domain.Campaign) error {
	params, _ := json.Marshal(campaign.TemplateParams)
	c, err := s.q.UpdateCampaign(ctx, gen.UpdateCampaignParams{
		TenantID:       campaign.TenantID,
		ID:             campaign.ID,
		Name:           campaign.Name,
		TemplateID:     campaign.TemplateID,
		TemplateParams: params,
		SegmentID:      fromUUIDPtr(campaign.SegmentID),
		ContactIds:     campaign.ContactIDs,
		ScheduledAt:    fromTimePtr(campaign.ScheduledAt),
		Status:         string(campaign.Status),
	})
	if err != nil {
		return err
	}
	*campaign = toCampaignDomain(c)
	return nil
}

func (s *Store) UpdateCampaignStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CampaignStatus) error {
	return s.q.UpdateCampaignStatus(ctx, gen.UpdateCampaignStatusParams{
		TenantID: tenantID,
		ID:       id,
		Status:   string(status),
	})
}

func (s *Store) UpdateCampaignStats(ctx context.Context, tenantID, id uuid.UUID, stats domain.CampaignStats) error {
	st, _ := json.Marshal(stats)
	return s.q.UpdateCampaignStats(ctx, gen.UpdateCampaignStatsParams{
		TenantID: tenantID,
		ID:       id,
		Stats:    st,
	})
}

func (s *Store) DeleteCampaign(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteCampaign(ctx, gen.DeleteCampaignParams{TenantID: tenantID, ID: id})
}

func (s *Store) GetScheduledCampaigns(ctx context.Context, before time.Time) ([]domain.Campaign, error) {
	rows, err := s.q.GetScheduledCampaigns(ctx, fromTimePtr(&before))
	if err != nil {
		return nil, err
	}
	var res []domain.Campaign
	for _, r := range rows {
		res = append(res, toCampaignDomain(r))
	}
	return res, nil
}

func (s *Store) GetRunningCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	rows, err := s.q.GetRunningCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	var res []domain.Campaign
	for _, r := range rows {
		res = append(res, toCampaignDomain(r))
	}
	return res, nil
}

// ProviderConfigRepository
func (s *Store) CreateProviderConfig(ctx context.Context, cfg *domain.ProviderConfig) error {
	p, err := s.q.CreateProviderConfig(ctx, gen.CreateProviderConfigParams{
		TenantID:      cfg.TenantID,
		Channel:       string(cfg.Channel),
		Provider:      cfg.Provider,
		Credentials:   cfg.Credentials,
		WebhookSecret: fromString(cfg.WebhookSecret),
		IsActive:      cfg.IsActive,
		PhoneNumberID: fromString(cfg.PhoneNumberID),
		WabaID:        fromString(cfg.WABAID),
		BusinessID:    fromString(cfg.BusinessID),
		DisplayPhone:  fromString(cfg.DisplayPhone),
		FromEmail:     fromString(cfg.FromEmail),
		FromName:      fromString(cfg.FromName),
		ReplyToEmail:  fromString(cfg.ReplyToEmail),
	})
	if err != nil {
		return err
	}
	*cfg = toProviderConfigDomain(p)
	return nil
}

func (s *Store) GetProviderConfigByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ProviderConfig, error) {
	p, err := s.q.GetProviderConfigByID(ctx, gen.GetProviderConfigByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toProviderConfigDomain(p)
	return &res, nil
}

func (s *Store) GetProviderConfigByChannel(ctx context.Context, tenantID uuid.UUID, channel domain.Channel) (*domain.ProviderConfig, error) {
	p, err := s.q.GetProviderConfigByChannel(ctx, gen.GetProviderConfigByChannelParams{TenantID: tenantID, Channel: string(channel)})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toProviderConfigDomain(p)
	return &res, nil
}

func (s *Store) GetProviderConfigByWABAID(ctx context.Context, wabaID string) (*domain.ProviderConfig, error) {
	p, err := s.q.GetProviderConfigByWABAID(ctx, fromString(wabaID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toProviderConfigDomain(p)
	return &res, nil
}

func (s *Store) ListProviderConfigs(ctx context.Context, tenantID uuid.UUID) ([]domain.ProviderConfig, error) {
	rows, err := s.q.ListProviderConfigs(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var res []domain.ProviderConfig
	for _, r := range rows {
		res = append(res, toProviderConfigDomain(r))
	}
	return res, nil
}

func (s *Store) UpdateProviderConfig(ctx context.Context, cfg *domain.ProviderConfig) error {
	p, err := s.q.UpdateProviderConfig(ctx, gen.UpdateProviderConfigParams{
		TenantID:      cfg.TenantID,
		ID:            cfg.ID,
		Provider:      cfg.Provider,
		Credentials:   cfg.Credentials,
		WebhookSecret: fromString(cfg.WebhookSecret),
		IsActive:      cfg.IsActive,
		PhoneNumberID: fromString(cfg.PhoneNumberID),
		WabaID:        fromString(cfg.WABAID),
		BusinessID:    fromString(cfg.BusinessID),
		DisplayPhone:  fromString(cfg.DisplayPhone),
		FromEmail:     fromString(cfg.FromEmail),
		FromName:      fromString(cfg.FromName),
		ReplyToEmail:  fromString(cfg.ReplyToEmail),
	})
	if err != nil {
		return err
	}
	*cfg = toProviderConfigDomain(p)
	return nil
}

func (s *Store) UpdateIsActive(ctx context.Context, tenantID, id uuid.UUID, isActive bool) error {
	return s.q.UpdateProviderConfigIsActive(ctx, gen.UpdateProviderConfigIsActiveParams{
		TenantID: tenantID,
		ID:       id,
		IsActive: isActive,
	})
}

func (s *Store) DeleteProviderConfig(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteProviderConfig(ctx, gen.DeleteProviderConfigParams{TenantID: tenantID, ID: id})
}

// SegmentRepository
func (s *Store) CreateSegment(ctx context.Context, segment *domain.Segment) error {
	rules, _ := json.Marshal(segment.Rules)
	p, err := s.q.CreateSegment(ctx, gen.CreateSegmentParams{
		TenantID:  segment.TenantID,
		Name:      segment.Name,
		IsDynamic: segment.IsDynamic,
		Rules:     rules,
	})
	if err != nil {
		return err
	}
	*segment = toSegmentDomain(p)
	return nil
}

func (s *Store) GetSegmentByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Segment, error) {
	p, err := s.q.GetSegmentByID(ctx, gen.GetSegmentByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toSegmentDomain(p)
	return &res, nil
}

func (s *Store) ListSegments(ctx context.Context, tenantID uuid.UUID) ([]domain.Segment, error) {
	rows, err := s.q.ListSegments(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var res []domain.Segment
	for _, r := range rows {
		res = append(res, toSegmentDomain(r))
	}
	return res, nil
}

func (s *Store) UpdateSegment(ctx context.Context, segment *domain.Segment) error {
	rules, _ := json.Marshal(segment.Rules)
	p, err := s.q.UpdateSegment(ctx, gen.UpdateSegmentParams{
		TenantID:  segment.TenantID,
		ID:        segment.ID,
		Name:      segment.Name,
		IsDynamic: segment.IsDynamic,
		Rules:     rules,
	})
	if err != nil {
		return err
	}
	*segment = toSegmentDomain(p)
	return nil
}

func (s *Store) DeleteSegment(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteSegment(ctx, gen.DeleteSegmentParams{TenantID: tenantID, ID: id})
}

func (s *Store) ResolveContacts(ctx context.Context, tenantID, segmentID uuid.UUID, page, perPage int) ([]domain.Contact, int, error) {
	rows, err := s.q.ResolveContacts(ctx, gen.ResolveContactsParams{
		TenantID:  tenantID,
		SegmentID: segmentID,
		Limit:     int32(perPage),
		Offset:    int32(page * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.Contact
	total := 0
	for _, r := range rows {
		res = append(res, toContactDomain(gen.MessagingContact{
			ID:            r.ID,
			TenantID:      r.TenantID,
			Phone:         r.Phone,
			Email:         r.Email,
			Name:          r.Name,
			Attributes:    r.Attributes,
			Tags:          r.Tags,
			OptInWhatsapp: r.OptInWhatsapp,
			OptInSms:      r.OptInSms,
			OptInEmail:    r.OptInEmail,
			OptedInAt:     r.OptedInAt,
			OptedOutAt:    r.OptedOutAt,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

// AnalyticsRepository
func (s *Store) GetMessageStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time, channel *domain.Channel) (*domain.MessageAnalytics, error) {
	row, err := s.q.GetMessageStats(ctx, gen.GetMessageStatsParams{
		TenantID: tenantID,
		Column2:  from,
		Column3:  to,
		Column4:  derefString(fromStringPtr((*string)(channel))),
	})
	if err != nil {
		return nil, err
	}
	return &domain.MessageAnalytics{
		TotalSent: int(row.Total),
		Delivered: int(row.Delivered),
		Read:      int(row.Read),
		Failed:    int(row.Failed),
	}, nil
}

func (s *Store) GetCampaignStats(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.CampaignStats, error) {
	data, err := s.q.GetCampaignStats(ctx, gen.GetCampaignStatsParams{TenantID: tenantID, ID: campaignID})
	if err != nil {
		return nil, err
	}
	var stats domain.CampaignStats
	_ = json.Unmarshal(data, &stats)
	return &stats, nil
}

func derefString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// EmailTemplateRepository
func (s *Store) CreateEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	t, err := s.q.CreateEmailTemplate(ctx, gen.CreateEmailTemplateParams{
		TenantID:    tmpl.TenantID,
		Name:        tmpl.Name,
		Subject:     tmpl.Subject,
		PreviewText: tmpl.PreviewText,
		HtmlBody:    tmpl.HTMLBody,
		TextBody:    tmpl.TextBody,
		Category:    string(tmpl.Category),
		Variables:   tmpl.Variables,
		IsActive:    tmpl.IsActive,
	})
	if err != nil {
		return err
	}
	*tmpl = toEmailTemplateDomain(t)
	return nil
}

func (s *Store) GetEmailTemplateByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmailTemplate, error) {
	t, err := s.q.GetEmailTemplateByID(ctx, gen.GetEmailTemplateByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toEmailTemplateDomain(t)
	return &res, nil
}

func (s *Store) GetEmailTemplateByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.EmailTemplate, error) {
	t, err := s.q.GetEmailTemplateByName(ctx, gen.GetEmailTemplateByNameParams{TenantID: tenantID, Name: name})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	res := toEmailTemplateDomain(t)
	return &res, nil
}

func (s *Store) ListEmailTemplates(ctx context.Context, tenantID uuid.UUID, category *domain.EmailCategory, page, perPage int) ([]domain.EmailTemplate, int, error) {
	rows, err := s.q.ListEmailTemplates(ctx, gen.ListEmailTemplatesParams{
		TenantID: tenantID,
		Column2:  derefString(fromStringPtr((*string)(category))),
		Limit:    int32(perPage),
		Offset:   int32(page * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.EmailTemplate
	total := 0
	for _, r := range rows {
		res = append(res, toEmailTemplateDomain(gen.MessagingEmailTemplate{
			ID:          r.ID,
			TenantID:    r.TenantID,
			Name:        r.Name,
			Subject:     r.Subject,
			PreviewText: r.PreviewText,
			HtmlBody:    r.HtmlBody,
			TextBody:    r.TextBody,
			Category:    r.Category,
			Variables:   r.Variables,
			IsActive:    r.IsActive,
			Version:     r.Version,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) UpdateEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	t, err := s.q.UpdateEmailTemplate(ctx, gen.UpdateEmailTemplateParams{
		TenantID:    tmpl.TenantID,
		ID:          tmpl.ID,
		Subject:     tmpl.Subject,
		PreviewText: tmpl.PreviewText,
		HtmlBody:    tmpl.HTMLBody,
		TextBody:    tmpl.TextBody,
		Category:    string(tmpl.Category),
		Variables:   tmpl.Variables,
		IsActive:    tmpl.IsActive,
	})
	if err != nil {
		return err
	}
	*tmpl = toEmailTemplateDomain(t)
	return nil
}

func (s *Store) DeleteEmailTemplate(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteEmailTemplate(ctx, gen.DeleteEmailTemplateParams{TenantID: tenantID, ID: id})
}

// EmailEventRepository
func (s *Store) CreateEmailEvent(ctx context.Context, event *domain.EmailEvent) error {
	params, _ := json.Marshal(event.RawPayload)
	e, err := s.q.CreateEmailEvent(ctx, gen.CreateEmailEventParams{
		TenantID:          event.TenantID,
		MessageID:         event.MessageID,
		CampaignID:        fromUUIDPtr(event.CampaignID),
		EventType:         string(event.EventType),
		Recipient:         event.Recipient,
		Timestamp:         event.Timestamp,
		ProviderEventID:   fromString(event.ProviderEventID),
		UserAgent:         fromString(event.UserAgent),
		IpAddress:         fromString(event.IPAddress),
		Url:               fromString(event.URL),
		BounceType:        fromStringPtr(event.BounceType),
		BounceReason:      fromStringPtr(event.BounceReason),
		ComplaintFeedback: fromStringPtr(event.ComplaintFeedback),
		RawPayload:        params,
	})
	if err != nil {
		return err
	}
	*event = toEmailEventDomain(e)
	return nil
}

func (s *Store) CreateBatch(ctx context.Context, events []domain.EmailEvent) error {
	var params []gen.CreateEmailEventBatchParams
	for _, event := range events {
		p, _ := json.Marshal(event.RawPayload)
		params = append(params, gen.CreateEmailEventBatchParams{
			TenantID:          event.TenantID,
			MessageID:         event.MessageID,
			CampaignID:        fromUUIDPtr(event.CampaignID),
			EventType:         string(event.EventType),
			Recipient:         event.Recipient,
			Timestamp:         event.Timestamp,
			ProviderEventID:   fromString(event.ProviderEventID),
			UserAgent:         fromString(event.UserAgent),
			IpAddress:         fromString(event.IPAddress),
			Url:               fromString(event.URL),
			BounceType:        fromStringPtr(event.BounceType),
			BounceReason:      fromStringPtr(event.BounceReason),
			ComplaintFeedback: fromStringPtr(event.ComplaintFeedback),
			RawPayload:        p,
		})
	}
	_, err := s.q.CreateEmailEventBatch(ctx, params)
	return err
}

func (s *Store) GetByMessageID(ctx context.Context, tenantID, messageID uuid.UUID) ([]domain.EmailEvent, error) {
	rows, err := s.q.GetEmailEventsByMessageID(ctx, gen.GetEmailEventsByMessageIDParams{TenantID: tenantID, MessageID: messageID})
	if err != nil {
		return nil, err
	}
	var res []domain.EmailEvent
	for _, r := range rows {
		res = append(res, toEmailEventDomain(r))
	}
	return res, nil
}

func (s *Store) GetEmailEventsByCampaignID(ctx context.Context, tenantID, campaignID uuid.UUID, eventType *domain.EmailEventType, page, perPage int) ([]domain.EmailEvent, int, error) {
	rows, err := s.q.GetEmailEventsByCampaignID(ctx, gen.GetEmailEventsByCampaignIDParams{
		TenantID:   tenantID,
		CampaignID: fromUUIDPtr(&campaignID),
		Column3:    derefString(fromStringPtr((*string)(eventType))),
		Limit:      int32(perPage),
		Offset:     int32(page * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.EmailEvent
	total := 0
	for _, r := range rows {
		res = append(res, toEmailEventDomain(gen.MessagingEmailEvent{
			ID:                r.ID,
			TenantID:          r.TenantID,
			MessageID:         r.MessageID,
			CampaignID:        r.CampaignID,
			EventType:         r.EventType,
			Recipient:         r.Recipient,
			Timestamp:         r.Timestamp,
			ProviderEventID:   r.ProviderEventID,
			UserAgent:         r.UserAgent,
			IpAddress:         r.IpAddress,
			Url:               r.Url,
			BounceType:        r.BounceType,
			BounceReason:      r.BounceReason,
			ComplaintFeedback: r.ComplaintFeedback,
			RawPayload:        r.RawPayload,
			CreatedAt:         r.CreatedAt,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) ExistsByProviderEventID(ctx context.Context, providerEventID string) (bool, error) {
	return s.q.ExistsByProviderEventID(ctx, fromString(providerEventID))
}

func (s *Store) GetStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.EmailStats, error) {
	row, err := s.q.GetEmailStats(ctx, gen.GetEmailStatsParams{TenantID: tenantID, Timestamp: from, Timestamp_2: to})
	if err != nil {
		return nil, err
	}
	return &domain.EmailStats{
		Delivered:    int(row.Delivered),
		Opens:        int(row.Opens),
		UniqueOpens:  int(row.UniqueOpens),
		Clicks:       int(row.Clicks),
		UniqueClicks: int(row.UniqueClicks),
		Bounces:      int(row.Bounces),
		HardBounces:  int(row.HardBounces),
		SoftBounces:  int(row.SoftBounces),
		Complaints:   int(row.Complaints),
		Unsubscribes: int(row.Unsubscribes),
	}, nil
}

func (s *Store) GetEmailCampaignStats(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.EmailCampaignStats, error) {
	row, err := s.q.GetEmailCampaignStats(ctx, gen.GetEmailCampaignStatsParams{TenantID: tenantID, CampaignID: fromUUIDPtr(&campaignID)})
	if err != nil {
		return nil, err
	}
	return &domain.EmailCampaignStats{
		CampaignID: campaignID,
		EmailStats: domain.EmailStats{
			Delivered:    int(row.Delivered),
			Opens:        int(row.Opens),
			UniqueOpens:  int(row.UniqueOpens),
			Clicks:       int(row.Clicks),
			UniqueClicks: int(row.UniqueClicks),
			Bounces:      int(row.Bounces),
			HardBounces:  int(row.HardBounces),
			SoftBounces:  int(row.SoftBounces),
			Complaints:   int(row.Complaints),
			Unsubscribes: int(row.Unsubscribes),
		},
	}, nil
}

// UnsubscribeRepository
func (s *Store) CreateUnsubscribe(ctx context.Context, unsub *domain.Unsubscribe) error {
	u, err := s.q.CreateUnsubscribe(ctx, gen.CreateUnsubscribeParams{
		TenantID:   unsub.TenantID,
		Email:      unsub.Email,
		Scope:      string(unsub.Scope),
		CampaignID: fromUUIDPtr(unsub.CampaignID),
		Reason:     unsub.Reason,
	})
	if err != nil {
		return err
	}
	*unsub = toUnsubscribeDomain(u)
	return nil
}

func (s *Store) IsUnsubscribed(ctx context.Context, tenantID uuid.UUID, email string, scope domain.UnsubscribeScope, campaignID *uuid.UUID) (bool, error) {
	return s.q.IsUnsubscribed(ctx, gen.IsUnsubscribedParams{
		TenantID:   tenantID,
		Email:      email,
		Column3:    string(scope),
		CampaignID: fromUUIDPtr(campaignID),
	})
}

func (s *Store) ListUnsubscribes(ctx context.Context, tenantID uuid.UUID, scope *domain.UnsubscribeScope, page, perPage int) ([]domain.Unsubscribe, int, error) {
	rows, err := s.q.ListUnsubscribes(ctx, gen.ListUnsubscribesParams{
		TenantID: tenantID,
		Column2:  derefString(fromStringPtr((*string)(scope))),
		Limit:    int32(perPage),
		Offset:   int32(page * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.Unsubscribe
	total := 0
	for _, r := range rows {
		res = append(res, toUnsubscribeDomain(gen.MessagingUnsubscribe{
			ID:         r.ID,
			TenantID:   r.TenantID,
			Email:      r.Email,
			Scope:      r.Scope,
			CampaignID: r.CampaignID,
			Reason:     r.Reason,
			CreatedAt:  r.CreatedAt,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) DeleteUnsubscribe(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteUnsubscribe(ctx, gen.DeleteUnsubscribeParams{TenantID: tenantID, ID: id})
}

func (s *Store) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) ([]domain.Unsubscribe, error) {
	rows, err := s.q.GetUnsubscribesByEmail(ctx, gen.GetUnsubscribesByEmailParams{TenantID: tenantID, Email: email})
	if err != nil {
		return nil, err
	}
	var res []domain.Unsubscribe
	for _, r := range rows {
		res = append(res, toUnsubscribeDomain(r))
	}
	return res, nil
}

// SuppressionRepository
func (s *Store) CreateSuppression(ctx context.Context, entry *domain.SuppressionEntry) error {
	supp, err := s.q.CreateSuppression(ctx, gen.CreateSuppressionParams{
		TenantID: entry.TenantID,
		Email:    entry.Email,
		Reason:   string(entry.Reason),
		Source:   entry.Source,
	})
	if err != nil {
		return err
	}
	*entry = toSuppressionEntryDomain(supp)
	return nil
}

func (s *Store) IsSuppressed(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	return s.q.IsSuppressed(ctx, gen.IsSuppressedParams{TenantID: tenantID, Email: email})
}

func (s *Store) ListSuppressions(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error) {
	rows, err := s.q.ListSuppressions(ctx, gen.ListSuppressionsParams{
		TenantID: tenantID,
		Column2:  derefString(fromStringPtr((*string)(reason))),
		Limit:    int32(perPage),
		Offset:   int32(page * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	var res []domain.SuppressionEntry
	total := 0
	for _, r := range rows {
		res = append(res, toSuppressionEntryDomain(gen.MessagingSuppression{
			ID:        r.ID,
			TenantID:  r.TenantID,
			Email:     r.Email,
			Reason:    r.Reason,
			Source:    r.Source,
			CreatedAt: r.CreatedAt,
		}))
		total = int(r.TotalCount)
	}
	return res, total, nil
}

func (s *Store) DeleteSuppression(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteSuppression(ctx, gen.DeleteSuppressionParams{TenantID: tenantID, ID: id})
}

func (s *Store) BulkCheck(ctx context.Context, tenantID uuid.UUID, emails []string) ([]string, error) {
	return s.q.BulkCheckSuppression(ctx, gen.BulkCheckSuppressionParams{TenantID: tenantID, Column2: emails})
}
