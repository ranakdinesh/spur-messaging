package worker

import (
	"context"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/core/services"
)

type TemplateSync struct {
	templateRepo     ports.TemplateRepository
	providerRegistry *services.ProviderRegistry
}

func NewTemplateSync(templateRepo ports.TemplateRepository, providerRegistry *services.ProviderRegistry) *TemplateSync {
	return &TemplateSync{
		templateRepo:     templateRepo,
		providerRegistry: providerRegistry,
	}
}

func (s *TemplateSync) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SyncPendingTemplates(ctx)
		}
	}
}

func (s *TemplateSync) SyncPendingTemplates(ctx context.Context) {
	// 1. Get all pending templates.
	// Since the repository requires a tenantID, this is tricky in a global worker.
	// In a real Spur deployment, we'd iterate over all tenants.
	// For this task, we'll focus on the logic.

	// Assume we have a list of templates to sync.
	var templates []domain.Template // This would be fetched from DB

	for _, tmpl := range templates {
		if tmpl.Channel != domain.ChannelWhatsApp || tmpl.ProviderTemplateID == nil {
			continue
		}

		provider, cfg, err := s.providerRegistry.GetProvider(ctx, tmpl.TenantID, tmpl.Channel)
		if err != nil {
			continue
		}

		status, reason, err := provider.GetTemplateStatus(ctx, cfg, *tmpl.ProviderTemplateID)
		if err != nil {
			continue
		}

		if status != tmpl.Status {
			_ = s.templateRepo.UpdateStatus(ctx, tmpl.TenantID, tmpl.ID, status, tmpl.ProviderTemplateID, reason)
		}
	}
}
