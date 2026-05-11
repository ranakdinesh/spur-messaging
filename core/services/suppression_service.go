package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type SuppressionService struct {
	repo ports.SuppressionRepository
}

func NewSuppressionService(repo ports.SuppressionRepository) *SuppressionService {
	return &SuppressionService{repo: repo}
}

func (s *SuppressionService) IsSuppressed(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string) (bool, error) {
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		return false, nil
	}
	return s.repo.IsSuppressed(ctx, tenantID, channel, recipient)
}

func (s *SuppressionService) AddToSuppression(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string, reason domain.SuppressionReason) error {
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		return domain.NewValidationError("recipient", "recipient is required")
	}
	if reason == "" {
		reason = domain.SuppressionManual
	}
	entry := &domain.SuppressionEntry{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Channel:   channel,
		Recipient: recipient,
		Reason:    reason,
		Source:    "manual",
		CreatedAt: time.Now(),
	}
	if channel == domain.ChannelEmail {
		entry.Email = recipient
	}

	return s.repo.Create(ctx, entry)
}

func (s *SuppressionService) RemoveFromSuppression(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *SuppressionService) List(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error) {
	return s.repo.List(ctx, tenantID, reason, page, perPage)
}

func (s *SuppressionService) BulkCheck(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipients []string) ([]string, error) {
	return s.repo.BulkCheck(ctx, tenantID, channel, recipients)
}
