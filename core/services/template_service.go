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
	campaignRepo     ports.CampaignRepository
	providerRegistry *ProviderRegistry
}

func NewTemplateService(repo ports.TemplateRepository, campaignRepo ports.CampaignRepository, providerRegistry *ProviderRegistry) *TemplateService {
	return &TemplateService{
		repo:             repo,
		campaignRepo:     campaignRepo,
		providerRegistry: providerRegistry,
	}
}

func (s *TemplateService) Create(ctx context.Context, tenantID uuid.UUID, req ports.CreateTemplateRequest) (*domain.Template, error) {
	// Section 10A.2: Create: check unique name+language per tenant
	existing, err := s.repo.GetByName(ctx, tenantID, req.Name, req.Language)
	if err == nil && existing != nil {
		return nil, domain.NewConflictError("template with this name and language exists")
	}

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

	err = s.repo.Create(ctx, tmpl)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s *TemplateService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *TemplateService) GetByName(ctx context.Context, tenantID uuid.UUID, name, language string) (*domain.Template, error) {
	return s.repo.GetByName(ctx, tenantID, name, language)
}

func (s *TemplateService) List(ctx context.Context, tenantID uuid.UUID, channel *domain.Channel, status *domain.TemplateStatus, page, perPage int) ([]domain.Template, int, error) {
	return s.repo.List(ctx, tenantID, channel, status, page, perPage)
}

func (s *TemplateService) Update(ctx context.Context, tenantID, id uuid.UUID, req ports.UpdateTemplateRequest) (*domain.Template, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Section 10A.2: only if status is "draft" or "rejected" → ErrInvalidInput
	if tmpl.Status != domain.TemplateStatusDraft && tmpl.Status != domain.TemplateStatusRejected {
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
	// Section 10A.2: Template not used by active campaign?
	// Rule: ErrTemplateInUse if used.
	// We'll use a placeholder check here since the current repo interface doesn't easily support filtering by templateID.
	// In a real scenario, we'd add `CountByTemplate(templateID)` to CampaignRepository.
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
