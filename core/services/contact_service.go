package services

import (
	"context"
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

func (s *ContactService) BulkImport(ctx context.Context, tenantID uuid.UUID, reqs []ports.CreateContactRequest) (int, error) {
	// Section 10A.2: Max 10,000 contacts per request (Handled in Handler)

	// Section 10A.3: Bulk import error handling
	// Process ALL rows even if some fail

	var contacts []domain.Contact
	importedCount := 0

	for _, req := range reqs {
		// Basic validation
		if (req.Phone == nil || *req.Phone == "") && (req.Email == nil || *req.Email == "") {
			continue // skip invalid row
		}

		contacts = append(contacts, domain.Contact{
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
		})
	}

	if len(contacts) > 0 {
		count, err := s.repo.BulkCreate(ctx, contacts)
		if err != nil {
			return 0, err
		}
		importedCount = count
	}

	return importedCount, nil
}

func (s *ContactService) OptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel) error {
	return s.repo.UpdateOptIn(ctx, tenantID, id, channel, domain.OptInStatusOptedIn)
}

func (s *ContactService) OptOut(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel) error {
	return s.repo.UpdateOptIn(ctx, tenantID, id, channel, domain.OptInStatusOptedOut)
}
