package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type ConversationService struct {
	repo ports.ConversationRepository
}

func NewConversationService(repo ports.ConversationRepository) *ConversationService {
	return &ConversationService{repo: repo}
}

func (s *ConversationService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Conversation, error) {
	return s.repo.GetConversationByID(ctx, tenantID, id)
}

func (s *ConversationService) List(ctx context.Context, tenantID uuid.UUID, filter ports.ConversationFilter) ([]domain.Conversation, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	return s.repo.ListConversations(ctx, tenantID, filter)
}

func (s *ConversationService) Update(ctx context.Context, tenantID, id uuid.UUID, req ports.UpdateConversationRequest) (*domain.Conversation, error) {
	if req.Status != nil && !domain.IsValidConversationStatus(*req.Status) {
		return nil, domain.NewValidationError("status", "invalid conversation status")
	}
	if req.HandoffStatus != nil && !domain.IsValidConversationHandoffStatus(*req.HandoffStatus) {
		return nil, domain.NewValidationError("handoff_status", "invalid handoff status")
	}
	if req.Priority != nil && !domain.IsValidConversationPriority(*req.Priority) {
		return nil, domain.NewValidationError("priority", "invalid priority")
	}
	if req.Tags != nil {
		for _, tag := range *req.Tags {
			if strings.TrimSpace(tag) == "" {
				return nil, domain.NewValidationError("tags", "tags must not be empty")
			}
		}
	}

	return s.repo.UpdateConversation(ctx, tenantID, id, ports.ConversationUpdate{
		Status:             req.Status,
		HandoffStatus:      req.HandoffStatus,
		AssignedAgentID:    req.AssignedAgentID,
		AssignedTeam:       req.AssignedTeam,
		Priority:           req.Priority,
		Tags:               req.Tags,
		FirstResponseDueAt: req.FirstResponseDueAt,
		ResolutionDueAt:    req.ResolutionDueAt,
	})
}

func (s *ConversationService) AddNote(ctx context.Context, tenantID, id, authorID uuid.UUID, body string) (*domain.Conversation, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, domain.NewValidationError("body", "note body is required")
	}
	if len(body) > 5000 {
		return nil, domain.NewValidationError("body", "note body must be 5000 characters or fewer")
	}
	if authorID == uuid.Nil {
		return nil, domain.NewValidationError("author_id", "note author is required")
	}
	return s.repo.AddConversationNote(ctx, tenantID, id, domain.ConversationNote{
		ID:        uuid.New(),
		AuthorID:  authorID,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	})
}
