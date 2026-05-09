package services

import (
	"context"
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

func (s *SuppressionService) IsSuppressed(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	return s.repo.IsSuppressed(ctx, tenantID, email)
}

func (s *SuppressionService) AddToSuppression(ctx context.Context, tenantID uuid.UUID, email string, reason domain.SuppressionReason) error {
	entry := &domain.SuppressionEntry{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Email:     email,
		Reason:    reason,
		Source:    "manual",
		CreatedAt: time.Now(),
	}

	return s.repo.Create(ctx, entry)
}

func (s *SuppressionService) RemoveFromSuppression(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *SuppressionService) List(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error) {
	return s.repo.List(ctx, tenantID, reason, page, perPage)
}

func (s *SuppressionService) BulkCheck(ctx context.Context, tenantID uuid.UUID, emails []string) ([]string, error) {
	return s.repo.BulkCheck(ctx, tenantID, emails)
}
