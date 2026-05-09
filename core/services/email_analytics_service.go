package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type EmailAnalyticsService struct {
	eventRepo ports.EmailEventRepository
}

func NewEmailAnalyticsService(eventRepo ports.EmailEventRepository) *EmailAnalyticsService {
	return &EmailAnalyticsService{eventRepo: eventRepo}
}

func (s *EmailAnalyticsService) GetOverview(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.EmailStats, error) {
	return s.eventRepo.GetStats(ctx, tenantID, from, to)
}

func (s *EmailAnalyticsService) GetCampaignReport(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.EmailCampaignStats, error) {
	return s.eventRepo.GetCampaignStats(ctx, tenantID, campaignID)
}

func (s *EmailAnalyticsService) GetDomainReputation(ctx context.Context, tenantID uuid.UUID) (*domain.DomainReputation, error) {
	// Aggregate last 30 days
	// to := time.Now()
	// from := to.AddDate(0, 0, -30)
	// stats, err := s.eventRepo.GetStats(ctx, tenantID, from, to)

	return &domain.DomainReputation{
		HealthStatus: "good", // Mock for now
	}, nil
}

func (s *EmailAnalyticsService) GetTopLinks(ctx context.Context, tenantID, campaignID uuid.UUID, limit int) ([]domain.LinkStats, error) {
	// Query clicks from email_events grouped by URL
	return nil, nil
}

func (s *EmailAnalyticsService) GetBounceReport(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.BounceReport, error) {
	return nil, nil
}

func (s *EmailAnalyticsService) GetEngagementByHour(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]domain.HourlyEngagement, error) {
	return nil, nil
}
