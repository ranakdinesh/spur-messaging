package services

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

func TestMessageServiceSendReturnsExistingMessageForIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	key := "send:order-123"
	phone := "+919810914244"
	existing := &domain.Message{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Channel:        domain.ChannelSMS,
		Direction:      "outbound",
		Recipient:      phone,
		MessageType:    domain.MessageTypeText,
		IdempotencyKey: &key,
		Status:         domain.MessageStatusQueued,
		CreatedAt:      time.Now(),
	}

	repo := newMessageRepoStub()
	repo.seed(existing)
	queue := &messageQueueStub{}
	svc := newSMSMessageService(repo, queue, tenantID, phone)

	text := "hello"
	msg, err := svc.Send(ctx, tenantID, ports.SendMessageRequest{
		Channel:        domain.ChannelSMS,
		Recipient:      phone,
		MessageType:    domain.MessageTypeText,
		Text:           &text,
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if msg.ID != existing.ID {
		t.Fatalf("expected existing message %s, got %s", existing.ID, msg.ID)
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected no new message create, got %d creates", repo.createCalls)
	}
	if len(queue.enqueued) != 0 {
		t.Fatalf("expected no enqueue for replayed idempotency key, got %d", len(queue.enqueued))
	}
}

func TestMessageServiceSendStoresIdempotencyKeyAndDoesNotEnqueueDuplicate(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	key := "send:invoice-456"
	phone := "+919810914244"
	repo := newMessageRepoStub()
	queue := &messageQueueStub{}
	svc := newSMSMessageService(repo, queue, tenantID, phone)

	text := "hello"
	req := ports.SendMessageRequest{
		Channel:        domain.ChannelSMS,
		Recipient:      phone,
		MessageType:    domain.MessageTypeText,
		Text:           &text,
		IdempotencyKey: key,
	}

	first, err := svc.Send(ctx, tenantID, req)
	if err != nil {
		t.Fatalf("first Send returned error: %v", err)
	}
	if first.IdempotencyKey == nil || *first.IdempotencyKey != key {
		t.Fatalf("expected idempotency key %q to be stored, got %#v", key, first.IdempotencyKey)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected one message create, got %d", repo.createCalls)
	}
	if len(queue.enqueued) != 1 {
		t.Fatalf("expected one enqueue, got %d", len(queue.enqueued))
	}

	second, err := svc.Send(ctx, tenantID, req)
	if err != nil {
		t.Fatalf("second Send returned error: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected duplicate idempotency key to return first message %s, got %s", first.ID, second.ID)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected duplicate send not to create a second message, got %d creates", repo.createCalls)
	}
	if len(queue.enqueued) != 1 {
		t.Fatalf("expected duplicate send not to enqueue again, got %d enqueues", len(queue.enqueued))
	}
}

func newSMSMessageService(repo *messageRepoStub, queue *messageQueueStub, tenantID uuid.UUID, phone string) *MessageService {
	configRepo := &providerConfigRepoStub{
		cfg: &domain.ProviderConfig{
			ID:          uuid.New(),
			TenantID:    tenantID,
			Channel:     domain.ChannelSMS,
			Provider:    "fake_sms",
			IsActive:    true,
			SMSSenderID: "SPUR",
		},
	}
	registry := NewProviderRegistry(configRepo)
	registry.RegisterWithName("fake_sms", fakeProvider{channel: domain.ChannelSMS})

	return NewMessageService(
		repo,
		&contactRepoStub{phone: phone},
		nil,
		queue,
		nil,
		nil,
		nil,
		registry,
		Config{SMSSenderID: "SPUR"},
	)
}

type messageRepoStub struct {
	createCalls int
	byID        map[uuid.UUID]*domain.Message
	byKey       map[string]*domain.Message
}

func newMessageRepoStub() *messageRepoStub {
	return &messageRepoStub{
		byID:  make(map[uuid.UUID]*domain.Message),
		byKey: make(map[string]*domain.Message),
	}
}

func (r *messageRepoStub) seed(msg *domain.Message) {
	stored := *msg
	r.byID[msg.ID] = &stored
	if msg.IdempotencyKey != nil {
		r.byKey[*msg.IdempotencyKey] = &stored
	}
}

func (r *messageRepoStub) Create(_ context.Context, msg *domain.Message) error {
	r.createCalls++
	if msg.IdempotencyKey != nil {
		if _, ok := r.byKey[*msg.IdempotencyKey]; ok {
			return domain.ErrAlreadyExists
		}
	}
	r.seed(msg)
	return nil
}

func (r *messageRepoStub) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Message, error) {
	msg, ok := r.byID[id]
	if !ok || msg.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	stored := *msg
	return &stored, nil
}

func (r *messageRepoStub) GetByIdempotencyKey(_ context.Context, tenantID uuid.UUID, key string) (*domain.Message, error) {
	msg, ok := r.byKey[key]
	if !ok || msg.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	stored := *msg
	return &stored, nil
}

func (r *messageRepoStub) List(context.Context, uuid.UUID, ports.MessageFilter) ([]domain.Message, int, error) {
	return nil, 0, nil
}

func (r *messageRepoStub) UpdateStatus(context.Context, uuid.UUID, uuid.UUID, domain.MessageStatus, string) error {
	return nil
}

func (r *messageRepoStub) UpdateStatusByProviderID(context.Context, string, domain.MessageStatus, time.Time) error {
	return nil
}

func (r *messageRepoStub) GetByProviderID(context.Context, string) (*domain.Message, error) {
	return nil, domain.ErrNotFound
}

func (r *messageRepoStub) GetByCampaignID(context.Context, uuid.UUID, uuid.UUID, int, int) ([]domain.Message, int, error) {
	return nil, 0, nil
}

func (r *messageRepoStub) ExistsForCampaign(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}

func (r *messageRepoStub) CountByCampaign(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

type contactRepoStub struct {
	phone string
}

func (r *contactRepoStub) contact(tenantID uuid.UUID) *domain.Contact {
	return &domain.Contact{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Phone:      &r.phone,
		OptInSMS:   domain.OptInStatusOptedIn,
		OptInEmail: domain.OptInStatusOptedIn,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func (r *contactRepoStub) Create(context.Context, *domain.Contact) error { return nil }
func (r *contactRepoStub) GetByID(_ context.Context, tenantID, _ uuid.UUID) (*domain.Contact, error) {
	return r.contact(tenantID), nil
}
func (r *contactRepoStub) GetByPhone(_ context.Context, tenantID uuid.UUID, phone string) (*domain.Contact, error) {
	if phone != r.phone {
		return nil, domain.ErrNotFound
	}
	return r.contact(tenantID), nil
}
func (r *contactRepoStub) GetByEmail(context.Context, uuid.UUID, string) (*domain.Contact, error) {
	return nil, domain.ErrNotFound
}
func (r *contactRepoStub) List(context.Context, uuid.UUID, ports.ContactFilter) ([]domain.Contact, int, error) {
	return nil, 0, nil
}
func (r *contactRepoStub) Update(context.Context, *domain.Contact) error { return nil }
func (r *contactRepoStub) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (r *contactRepoStub) BulkCreate(context.Context, []domain.Contact) (int, error) {
	return 0, nil
}
func (r *contactRepoStub) UpdateOptIn(context.Context, uuid.UUID, uuid.UUID, domain.Channel, domain.OptInStatus) error {
	return nil
}
func (r *contactRepoStub) GetBySegment(context.Context, uuid.UUID, uuid.UUID, int, int) ([]domain.Contact, int, error) {
	return nil, 0, nil
}

type providerConfigRepoStub struct {
	cfg *domain.ProviderConfig
}

func (r *providerConfigRepoStub) Create(context.Context, *domain.ProviderConfig) error { return nil }
func (r *providerConfigRepoStub) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.ProviderConfig, error) {
	return nil, domain.ErrNotFound
}
func (r *providerConfigRepoStub) GetByChannel(_ context.Context, tenantID uuid.UUID, channel domain.Channel) (*domain.ProviderConfig, error) {
	if r.cfg == nil || r.cfg.TenantID != tenantID || r.cfg.Channel != channel {
		return nil, domain.ErrNotFound
	}
	return r.cfg, nil
}
func (r *providerConfigRepoStub) GetByWABAID(context.Context, string) (*domain.ProviderConfig, error) {
	return nil, domain.ErrNotFound
}
func (r *providerConfigRepoStub) List(context.Context, uuid.UUID) ([]domain.ProviderConfig, error) {
	return nil, nil
}
func (r *providerConfigRepoStub) Update(context.Context, *domain.ProviderConfig) error { return nil }
func (r *providerConfigRepoStub) UpdateIsActive(context.Context, uuid.UUID, uuid.UUID, bool) error {
	return nil
}
func (r *providerConfigRepoStub) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type messageQueueStub struct {
	enqueued []ports.QueueMessage
}

func (q *messageQueueStub) Enqueue(_ context.Context, msg ports.QueueMessage) error {
	q.enqueued = append(q.enqueued, msg)
	return nil
}
func (q *messageQueueStub) EnqueueBulk(_ context.Context, msgs []ports.QueueMessage) error {
	q.enqueued = append(q.enqueued, msgs...)
	return nil
}
func (q *messageQueueStub) StartConsumer(context.Context, func(context.Context, ports.QueueMessage) error) error {
	return nil
}
func (q *messageQueueStub) Stop() {}

type fakeProvider struct {
	channel domain.Channel
}

func (p fakeProvider) Channel() domain.Channel { return p.channel }
func (p fakeProvider) Send(context.Context, *domain.ProviderConfig, ports.ProviderSendRequest) (*ports.ProviderSendResult, error) {
	return nil, nil
}
func (p fakeProvider) SubmitTemplate(context.Context, *domain.ProviderConfig, domain.Template) (string, error) {
	return "", nil
}
func (p fakeProvider) GetTemplateStatus(context.Context, *domain.ProviderConfig, string) (domain.TemplateStatus, *string, error) {
	return domain.TemplateStatusApproved, nil, nil
}
func (p fakeProvider) ParseWebhook(context.Context, *domain.ProviderConfig, http.Header, []byte) ([]ports.WebhookEvent, error) {
	return nil, nil
}
func (p fakeProvider) ValidateWebhookSignature(*domain.ProviderConfig, http.Header, []byte) bool {
	return true
}
