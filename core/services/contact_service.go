package services

import (
	"context"
	"regexp"
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

func (s *ContactService) OptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel) error {
	contact, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Section 10A.2: Idempotent — return success if already opted in
	var currentStatus domain.OptInStatus
	switch channel {
	case domain.ChannelWhatsApp:
		currentStatus = contact.OptInWhatsApp
	case domain.ChannelSMS:
		currentStatus = contact.OptInSMS
	case domain.ChannelEmail:
		currentStatus = contact.OptInEmail
	}

	if currentStatus == domain.OptInStatusOptedIn {
		return nil
	}

	return s.repo.UpdateOptIn(ctx, tenantID, id, channel, domain.OptInStatusOptedIn)
}

func (s *ContactService) OptOut(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel) error {
	contact, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Section 10A.2: Idempotent — return success if already opted out
	var currentStatus domain.OptInStatus
	switch channel {
	case domain.ChannelWhatsApp:
		currentStatus = contact.OptInWhatsApp
	case domain.ChannelSMS:
		currentStatus = contact.OptInSMS
	case domain.ChannelEmail:
		currentStatus = contact.OptInEmail
	}

	if currentStatus == domain.OptInStatusOptedOut {
		return nil
	}

	return s.repo.UpdateOptIn(ctx, tenantID, id, channel, domain.OptInStatusOptedOut)
}
