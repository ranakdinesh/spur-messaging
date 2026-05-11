package services

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type ContactService struct {
	repo ports.ContactRepository
}

func NewContactService(repo ports.ContactRepository) *ContactService {
	return &ContactService{repo: repo}
}

func (s *ContactService) Create(ctx context.Context, tenantID uuid.UUID, req ports.CreateContactRequest) (*domain.Contact, error) {
	// Section 10A.2: Phone or email required
	if (req.Phone == nil || *req.Phone == "") && (req.Email == nil || *req.Email == "") {
		return nil, domain.NewValidationError("contact", "phone or email is required")
	}

	// Section 10A.2: Uniqueness checks
	if req.Phone != nil && *req.Phone != "" {
		existing, err := s.repo.GetByPhone(ctx, tenantID, *req.Phone)
		if err == nil && existing != nil {
			return nil, domain.NewConflictError("contact with this phone exists")
		}
	}
	if req.Email != nil && *req.Email != "" {
		existing, err := s.repo.GetByEmail(ctx, tenantID, *req.Email)
		if err == nil && existing != nil {
			return nil, domain.NewConflictError("contact with this email exists")
		}
	}

	contact := &domain.Contact{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Phone:         req.Phone,
		Email:         req.Email,
		Name:          req.Name,
		Attributes:    req.Attributes,
		Tags:          req.Tags,
		OptInWhatsApp: domain.OptInStatusPending,
		OptInSMS:      domain.OptInStatusPending,
		OptInEmail:    domain.OptInStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := s.repo.Create(ctx, contact)
	if err != nil {
		return nil, err
	}

	return contact, nil
}

func (s *ContactService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Contact, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *ContactService) List(ctx context.Context, tenantID uuid.UUID, filter ports.ContactFilter) ([]domain.Contact, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

func (s *ContactService) Update(ctx context.Context, tenantID, id uuid.UUID, req ports.UpdateContactRequest) (*domain.Contact, error) {
	contact, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Section 10A.2: Create/Update: phone unique per tenant
	if req.Phone != nil && *req.Phone != "" && (contact.Phone == nil || *req.Phone != *contact.Phone) {
		existing, err := s.repo.GetByPhone(ctx, tenantID, *req.Phone)
		if err == nil && existing != nil {
			return nil, domain.NewConflictError("contact with this phone exists")
		}
	}

	// Section 10A.2: Create/Update: email unique per tenant
	if req.Email != nil && *req.Email != "" && (contact.Email == nil || *req.Email != *contact.Email) {
		existing, err := s.repo.GetByEmail(ctx, tenantID, *req.Email)
		if err == nil && existing != nil {
			return nil, domain.NewConflictError("contact with this email exists")
		}
	}

	if req.Phone != nil {
		contact.Phone = req.Phone
	}
	if req.Email != nil {
		contact.Email = req.Email
	}
	if req.Name != nil {
		contact.Name = req.Name
	}
	if req.Attributes != nil {
		contact.Attributes = *req.Attributes
	}
	if req.Tags != nil {
		contact.Tags = *req.Tags
	}
	contact.UpdatedAt = time.Now()

	err = s.repo.Update(ctx, contact)
	if err != nil {
		return nil, err
	}

	return contact, nil
}

func (s *ContactService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *ContactService) BulkImport(ctx context.Context, tenantID uuid.UUID, reqs []ports.CreateContactRequest) (ports.BulkImportResult, error) {
	result := ports.BulkImportResult{
		Total: len(reqs),
	}

	for i, req := range reqs {
		rowNum := i + 1

		// 1. Validation
		if (req.Phone == nil || *req.Phone == "") && (req.Email == nil || *req.Email == "") {
			result.Errors = append(result.Errors, ports.ImportRowError{
				Row:     rowNum,
				Field:   "phone/email",
				Message: "phone or email is required",
			})
			continue
		}

		// Phone validation
		if req.Phone != nil && *req.Phone != "" {
			// Basic E.164 check: ^\+[1-9]\d{6,14}$
			matched, _ := regexp.MatchString(`^\+[1-9]\d{6,14}$`, *req.Phone)
			if !matched {
				result.Errors = append(result.Errors, ports.ImportRowError{
					Row:     rowNum,
					Field:   "phone",
					Message: "phone must be E.164 format (e.g. +919810914244)",
				})
				continue
			}
		}

		// 2. Duplicate Check
		if req.Phone != nil && *req.Phone != "" {
			existing, err := s.repo.GetByPhone(ctx, tenantID, *req.Phone)
			if err == nil && existing != nil {
				result.Duplicates++
				continue
			}
		}
		if req.Email != nil && *req.Email != "" {
			existing, err := s.repo.GetByEmail(ctx, tenantID, *req.Email)
			if err == nil && existing != nil {
				result.Duplicates++
				continue
			}
		}

		// 3. Create
		contact := &domain.Contact{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Phone:         req.Phone,
			Email:         req.Email,
			Name:          req.Name,
			Attributes:    req.Attributes,
			Tags:          req.Tags,
			OptInWhatsApp: domain.OptInStatusPending,
			OptInSMS:      domain.OptInStatusPending,
			OptInEmail:    domain.OptInStatusPending,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := s.repo.Create(ctx, contact)
		if err != nil {
			result.Errors = append(result.Errors, ports.ImportRowError{
				Row:     rowNum,
				Message: err.Error(),
			})
			continue
		}

		result.Imported++
	}

	return result, nil
}

func (s *ContactService) OptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, consent ports.ConsentEvidence) error {
	contact, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Section 10A.2: Idempotent — return success if already opted in
	currentStatus, err := contactOptInStatus(contact, channel)
	if err != nil {
		return err
	}

	if currentStatus == domain.OptInStatusOptedIn {
		return nil
	}
	if consent.ExpiresAt != nil && consent.ExpiresAt.Before(time.Now()) {
		return domain.NewValidationError("expires_at", "consent expiry must be in the future")
	}
	targetStatus := domain.OptInStatusOptedIn
	if consent.DoubleOptInRequired {
		targetStatus = domain.OptInStatusDoubleOptInPending
	}

	if err := s.repo.UpdateOptIn(ctx, tenantID, id, channel, targetStatus); err != nil {
		return err
	}
	return s.repo.CreateConsentRecord(ctx, buildConsentRecord(tenantID, id, channel, targetStatus, consent))
}

func (s *ContactService) ConfirmOptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, consent ports.ConsentEvidence) error {
	contact, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	currentStatus, err := contactOptInStatus(contact, channel)
	if err != nil {
		return err
	}
	if currentStatus == domain.OptInStatusOptedIn {
		return nil
	}
	if currentStatus != domain.OptInStatusDoubleOptInPending && currentStatus != domain.OptInStatusPending {
		return domain.NewValidationError("status", "contact is not waiting for opt-in confirmation")
	}
	now := time.Now()
	if expired, err := s.pendingConsentExpired(ctx, tenantID, id, channel, now); err != nil {
		return err
	} else if expired {
		return domain.NewValidationError("expires_at", "opt-in confirmation has expired")
	}
	consent.DoubleOptInRequired = false
	consent.ConfirmedAt = &now
	if consent.Source == "" {
		consent.Source = "double_opt_in_confirmed"
	}
	if err := s.repo.UpdateOptIn(ctx, tenantID, id, channel, domain.OptInStatusOptedIn); err != nil {
		return err
	}
	return s.repo.CreateConsentRecord(ctx, buildConsentRecord(tenantID, id, channel, domain.OptInStatusOptedIn, consent))
}

func (s *ContactService) pendingConsentExpired(ctx context.Context, tenantID, contactID uuid.UUID, channel domain.Channel, now time.Time) (bool, error) {
	records, err := s.repo.ListConsentRecords(ctx, tenantID, contactID, 1, 10)
	if err != nil {
		return false, err
	}
	for _, record := range records {
		if record.Channel != channel || record.Status != domain.OptInStatusDoubleOptInPending {
			continue
		}
		return record.ExpiresAt != nil && record.ExpiresAt.Before(now), nil
	}
	return false, nil
}

func (s *ContactService) OptOut(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, consent ports.ConsentEvidence) error {
	contact, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Section 10A.2: Idempotent — return success if already opted out
	currentStatus, err := contactOptInStatus(contact, channel)
	if err != nil {
		return err
	}

	if currentStatus == domain.OptInStatusOptedOut {
		return nil
	}

	if err := s.repo.UpdateOptIn(ctx, tenantID, id, channel, domain.OptInStatusOptedOut); err != nil {
		return err
	}
	return s.repo.CreateConsentRecord(ctx, buildConsentRecord(tenantID, id, channel, domain.OptInStatusOptedOut, consent))
}

func (s *ContactService) HandleInboundConsentKeyword(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient, text string, consent ports.ConsentEvidence) (domain.ConsentKeywordAction, error) {
	action := domain.DetectConsentKeyword(text)
	if action == domain.ConsentKeywordUnknown {
		return action, nil
	}
	contact, err := s.contactByRecipient(ctx, tenantID, channel, recipient)
	if err != nil {
		return action, err
	}
	consent.Keyword = strings.TrimSpace(text)
	if consent.Source == "" {
		consent.Source = "inbound_keyword"
	}
	switch action {
	case domain.ConsentKeywordOptIn:
		consent.DoubleOptInRequired = false
		return action, s.OptIn(ctx, tenantID, contact.ID, channel, consent)
	case domain.ConsentKeywordOptOut:
		return action, s.OptOut(ctx, tenantID, contact.ID, channel, consent)
	default:
		return action, nil
	}
}

func (s *ContactService) ListConsentRecords(ctx context.Context, tenantID, contactID uuid.UUID, page, perPage int) ([]domain.ConsentRecord, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return s.repo.ListConsentRecords(ctx, tenantID, contactID, page, perPage)
}

func contactOptInStatus(contact *domain.Contact, channel domain.Channel) (domain.OptInStatus, error) {
	switch channel {
	case domain.ChannelWhatsApp:
		return contact.OptInWhatsApp, nil
	case domain.ChannelSMS:
		return contact.OptInSMS, nil
	case domain.ChannelEmail:
		return contact.OptInEmail, nil
	default:
		return "", domain.NewValidationError("channel", "channel must be whatsapp, sms, or email")
	}
}

func (s *ContactService) contactByRecipient(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string) (*domain.Contact, error) {
	recipient = strings.TrimSpace(recipient)
	switch channel {
	case domain.ChannelWhatsApp, domain.ChannelSMS:
		contact, err := s.repo.GetByPhone(ctx, tenantID, recipient)
		if err == nil {
			return contact, nil
		}
		if !strings.HasPrefix(recipient, "+") {
			return s.repo.GetByPhone(ctx, tenantID, "+"+recipient)
		}
		return nil, err
	case domain.ChannelEmail:
		return s.repo.GetByEmail(ctx, tenantID, recipient)
	default:
		return nil, domain.NewValidationError("channel", "channel must be whatsapp, sms, or email")
	}
}

func buildConsentRecord(tenantID, contactID uuid.UUID, channel domain.Channel, status domain.OptInStatus, evidence ports.ConsentEvidence) *domain.ConsentRecord {
	source := evidence.Source
	if source == "" {
		source = "manual"
	}
	return &domain.ConsentRecord{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ContactID:   contactID,
		Channel:     channel,
		Status:      status,
		Source:      source,
		Purpose:     evidence.Purpose,
		Proof:       evidence.Proof,
		IPAddress:   evidence.IPAddress,
		UserAgent:   evidence.UserAgent,
		Brand:       evidence.Brand,
		Keyword:     evidence.Keyword,
		Locale:      evidence.Locale,
		ExpiresAt:   evidence.ExpiresAt,
		ConfirmedAt: evidence.ConfirmedAt,
		CreatedAt:   time.Now(),
	}
}
