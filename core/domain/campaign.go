package domain

import (
	"time"

	"github.com/google/uuid"
)

type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusScheduled CampaignStatus = "scheduled"
	CampaignStatusRunning   CampaignStatus = "running"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusFailed    CampaignStatus = "failed"
)

type Campaign struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	Name           string
	Channel        Channel
	TemplateID     uuid.UUID
	TemplateParams map[string]string // static params; per-contact params come from contact attributes
	SegmentID      *uuid.UUID        // target segment
	ContactIDs     []uuid.UUID       // OR explicit contact list
	ScheduledAt    *time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	Status         CampaignStatus
	Stats          CampaignStats
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CampaignStats struct {
	Total     int `json:"total"`
	Queued    int `json:"queued"`
	Sent      int `json:"sent"`
	Delivered int `json:"delivered"`
	Read      int `json:"read"`
	Failed    int `json:"failed"`
}
