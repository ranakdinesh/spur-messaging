package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

func TestConversationServiceRejectsInvalidStatus(t *testing.T) {
	svc := NewConversationService(&conversationServiceRepoStub{})
	status := domain.ConversationStatus("busy")

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), ports.UpdateConversationRequest{Status: &status})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestConversationServiceAddsNote(t *testing.T) {
	tenantID := uuid.New()
	conversationID := uuid.New()
	authorID := uuid.New()
	repo := &conversationServiceRepoStub{
		conversation: &domain.Conversation{ID: conversationID, TenantID: tenantID},
	}
	svc := NewConversationService(repo)

	conversation, err := svc.AddNote(context.Background(), tenantID, conversationID, authorID, "  call customer back  ")
	if err != nil {
		t.Fatalf("AddNote returned error: %v", err)
	}
	if len(conversation.Notes) != 1 {
		t.Fatalf("notes = %d", len(conversation.Notes))
	}
	if conversation.Notes[0].Body != "call customer back" {
		t.Fatalf("note body = %q", conversation.Notes[0].Body)
	}
	if conversation.Notes[0].AuthorID != authorID {
		t.Fatalf("author = %s", conversation.Notes[0].AuthorID)
	}
}

type conversationServiceRepoStub struct {
	conversation *domain.Conversation
}

func (r *conversationServiceRepoStub) GetActiveByRecipient(context.Context, uuid.UUID, domain.Channel, string, time.Time) (*domain.Conversation, error) {
	return nil, domain.ErrNotFound
}

func (r *conversationServiceRepoStub) UpsertInbound(context.Context, uuid.UUID, domain.Channel, string, time.Time) (*domain.Conversation, error) {
	return nil, nil
}

func (r *conversationServiceRepoStub) UpsertOutbound(context.Context, uuid.UUID, domain.Channel, string, time.Time) (*domain.Conversation, error) {
	return nil, nil
}

func (r *conversationServiceRepoStub) GetConversationByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Conversation, error) {
	if r.conversation == nil || r.conversation.TenantID != tenantID || r.conversation.ID != id {
		return nil, domain.ErrNotFound
	}
	conversation := *r.conversation
	return &conversation, nil
}

func (r *conversationServiceRepoStub) ListConversations(context.Context, uuid.UUID, ports.ConversationFilter) ([]domain.Conversation, int, error) {
	return nil, 0, nil
}

func (r *conversationServiceRepoStub) UpdateConversation(_ context.Context, tenantID, id uuid.UUID, _ ports.ConversationUpdate) (*domain.Conversation, error) {
	return r.GetConversationByID(context.Background(), tenantID, id)
}

func (r *conversationServiceRepoStub) AddConversationNote(_ context.Context, tenantID, id uuid.UUID, note domain.ConversationNote) (*domain.Conversation, error) {
	conversation, err := r.GetConversationByID(context.Background(), tenantID, id)
	if err != nil {
		return nil, err
	}
	conversation.Notes = append(conversation.Notes, note)
	r.conversation = conversation
	return conversation, nil
}
