package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type TemplateService struct {
	repo             ports.TemplateRepository
	providerRegistry *ProviderRegistry
}

func NewTemplateService(repo ports.TemplateRepository, providerRegistry *ProviderRegistry) *TemplateService {
	return &TemplateService{
		repo:             repo,
		providerRegistry: providerRegistry,
	}
}

func (s *TemplateService) Create(ctx context.Context, tenantID uuid.UUID, req ports.CreateTemplateRequest) (*domain.Template, error) {
	tmpl := &domain.Template{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Channel:    req.Channel,
		Name:       req.Name,
		Language:   req.Language,
		Category:   req.Category,
		Components: req.Components,
		Status:     domain.TemplateStatusDraft,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := s.repo.Create(ctx, tmpl)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s *TemplateService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *TemplateService) List(ctx context.Context, tenantID uuid.UUID, channel *domain.Channel, status *domain.TemplateStatus, page, perPage int) ([]domain.Template, int, error) {
	return s.repo.List(ctx, tenantID, channel, status, page, perPage)
}

func (s *TemplateService) Update(ctx context.Context, tenantID, id uuid.UUID, req ports.UpdateTemplateRequest) (*domain.Template, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Section 10A.2: approved/pending templates cannot be edited
	if tmpl.Status == domain.TemplateStatusApproved || tmpl.Status == domain.TemplateStatusPending {
		return nil, domain.NewValidationError("status", "approved/pending templates cannot be edited")
	}

	if req.Category != nil {
		tmpl.Category = *req.Category
	}
	if req.Components != nil {
		tmpl.Components = *req.Components
	}
	tmpl.UpdatedAt = time.Now()

	err = s.repo.Update(ctx, tmpl)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s *TemplateService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	// Section 10A.2: check in-use before delete
	// This would require checking if any campaign uses this template.
	// For now, placeholder check.
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *TemplateService) SubmitForApproval(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Section 10A.2: only draft templates can be submitted
	if tmpl.Status != domain.TemplateStatusDraft {
		return nil, domain.NewValidationError("status", "only draft templates can be submitted")
	}

	provider, cfg, err := s.providerRegistry.GetProvider(ctx, tenantID, tmpl.Channel)
	if err != nil {
		return nil, err
	}

	providerTmplID, err := provider.SubmitTemplate(ctx, cfg, *tmpl)
	if err != nil {
		return nil, err
	}

	tmpl.Status = domain.TemplateStatusPending
	tmpl.ProviderTemplateID = &providerTmplID
	tmpl.UpdatedAt = time.Now()

	err = s.repo.UpdateStatus(ctx, tenantID, id, tmpl.Status, tmpl.ProviderTemplateID, nil)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s *TemplateService) SyncStatus(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if tmpl.ProviderTemplateID == nil {
		return tmpl, nil
	}

	provider, cfg, err := s.providerRegistry.GetProvider(ctx, tenantID, tmpl.Channel)
	if err != nil {
		return nil, err
	}

	status, reason, err := provider.GetTemplateStatus(ctx, cfg, *tmpl.ProviderTemplateID)
	if err != nil {
		return nil, err
	}

	if status != tmpl.Status {
		tmpl.Status = status
		tmpl.RejectionReason = reason
		tmpl.UpdatedAt = time.Now()
		err = s.repo.UpdateStatus(ctx, tenantID, id, tmpl.Status, tmpl.ProviderTemplateID, tmpl.RejectionReason)
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}
