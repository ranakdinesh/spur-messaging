package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type UnsubscribeService struct {
	repo ports.UnsubscribeRepository
}

func NewUnsubscribeService(repo ports.UnsubscribeRepository) *UnsubscribeService {
	return &UnsubscribeService{repo: repo}
}

func (s *UnsubscribeService) Unsubscribe(ctx context.Context, tenantID uuid.UUID, email string, scope domain.UnsubscribeScope, campaignID *uuid.UUID, reason string) error {
	unsub := &domain.Unsubscribe{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Email:      email,
		Scope:      scope,
		CampaignID: campaignID,
		Reason:     reason,
		CreatedAt:  time.Now(),
	}

	return s.repo.Create(ctx, unsub)
}

func (s *UnsubscribeService) Resubscribe(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *UnsubscribeService) IsUnsubscribed(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	// By default, check global unsubscribe.
	// In a real implementation, you might want to check campaign-level too if provided.
	return s.repo.IsUnsubscribed(ctx, tenantID, email, domain.UnsubscribeScopeGlobal, nil)
}

func (s *UnsubscribeService) List(ctx context.Context, tenantID uuid.UUID, scope *domain.UnsubscribeScope, page, perPage int) ([]domain.Unsubscribe, int, error) {
	return s.repo.List(ctx, tenantID, scope, page, perPage)
}

func (s *UnsubscribeService) HandleUnsubscribeLink(ctx context.Context, token string) error {
	// 1. Decode token to get tenantID, email, scope, campaignID
	// (Token logic needs to be implemented)
	return errors.New("not implemented")
}
