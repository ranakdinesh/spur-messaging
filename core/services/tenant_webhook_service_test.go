package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

func TestTenantWebhookTestEndpointSignsDelivery(t *testing.T) {
	ctx := context.Background()
	repo := newTenantWebhookRepoStub()

	var receivedSignature string
	var receivedTimestamp string
	var receivedBody []byte
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		receivedSignature = r.Header.Get("X-Spur-Signature")
		receivedTimestamp = r.Header.Get("X-Spur-Timestamp")
		receivedBody, _ = io.ReadAll(r.Body)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	service := NewTenantWebhookService(repo, client, nil)
	endpoint, err := service.CreateEndpoint(ctx, uuid.New(), ports.CreateWebhookEndpointRequest{
		URL:    "https://hooks.example.test/messaging",
		Secret: "12345678901234567890123456789012",
		Events: []domain.WebhookEventType{
			domain.WebhookEventTest,
		},
	})
	if err != nil {
		t.Fatalf("CreateEndpoint() error = %v", err)
	}

	delivery, err := service.TestEndpoint(ctx, endpoint.TenantID, endpoint.ID)
	if err != nil {
		t.Fatalf("TestEndpoint() error = %v", err)
	}
	if delivery.Status != domain.WebhookDeliverySucceeded {
		t.Fatalf("delivery status = %s, want %s", delivery.Status, domain.WebhookDeliverySucceeded)
	}
	if receivedSignature == "" || receivedTimestamp == "" {
		t.Fatal("expected signature and timestamp headers")
	}
	if delivery.Signature != receivedSignature {
		t.Fatalf("stored signature = %q, received %q", delivery.Signature, receivedSignature)
	}
	if len(receivedBody) == 0 {
		t.Fatal("expected webhook payload body")
	}
}

func TestTenantWebhookReplayRetriesFailedDelivery(t *testing.T) {
	ctx := context.Background()
	repo := newTenantWebhookRepoStub()
	attempts := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("failed")),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})}

	service := NewTenantWebhookService(repo, client, nil)
	endpoint, err := service.CreateEndpoint(ctx, uuid.New(), ports.CreateWebhookEndpointRequest{
		URL:    "https://hooks.example.test/messaging",
		Secret: "12345678901234567890123456789012",
		Events: []domain.WebhookEventType{domain.WebhookEventTest},
	})
	if err != nil {
		t.Fatalf("CreateEndpoint() error = %v", err)
	}

	delivery, err := service.TestEndpoint(ctx, endpoint.TenantID, endpoint.ID)
	if err != nil {
		t.Fatalf("TestEndpoint() error = %v", err)
	}
	if delivery.Status != domain.WebhookDeliveryRetrying {
		t.Fatalf("first status = %s, want %s", delivery.Status, domain.WebhookDeliveryRetrying)
	}
	if delivery.NextAttemptAt == nil {
		t.Fatal("expected next retry time")
	}

	replayed, err := service.ReplayDelivery(ctx, endpoint.TenantID, delivery.ID)
	if err != nil {
		t.Fatalf("ReplayDelivery() error = %v", err)
	}
	if replayed.Status != domain.WebhookDeliverySucceeded {
		t.Fatalf("replay status = %s, want %s", replayed.Status, domain.WebhookDeliverySucceeded)
	}
	if replayed.AttemptCount != 2 {
		t.Fatalf("attempt count = %d, want 2", replayed.AttemptCount)
	}
}

type tenantWebhookRepoStub struct {
	mu         sync.Mutex
	endpoints  map[uuid.UUID]domain.WebhookEndpoint
	deliveries map[uuid.UUID]domain.WebhookDelivery
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTenantWebhookRepoStub() *tenantWebhookRepoStub {
	return &tenantWebhookRepoStub{
		endpoints:  make(map[uuid.UUID]domain.WebhookEndpoint),
		deliveries: make(map[uuid.UUID]domain.WebhookDelivery),
	}
}

func (r *tenantWebhookRepoStub) CreateWebhookEndpoint(_ context.Context, endpoint *domain.WebhookEndpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	endpoint.CreatedAt = now
	endpoint.UpdatedAt = now
	r.endpoints[endpoint.ID] = *endpoint
	return nil
}

func (r *tenantWebhookRepoStub) GetWebhookEndpoint(_ context.Context, tenantID, id uuid.UUID) (*domain.WebhookEndpoint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	endpoint, ok := r.endpoints[id]
	if !ok || endpoint.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return cloneEndpoint(endpoint), nil
}

func (r *tenantWebhookRepoStub) ListWebhookEndpoints(_ context.Context, tenantID uuid.UUID, _, _ int) ([]domain.WebhookEndpoint, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var endpoints []domain.WebhookEndpoint
	for _, endpoint := range r.endpoints {
		if endpoint.TenantID == tenantID {
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints, len(endpoints), nil
}

func (r *tenantWebhookRepoStub) UpdateWebhookEndpoint(_ context.Context, endpoint *domain.WebhookEndpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.endpoints[endpoint.ID]; !ok {
		return domain.ErrNotFound
	}
	endpoint.UpdatedAt = time.Now().UTC()
	r.endpoints[endpoint.ID] = *endpoint
	return nil
}

func (r *tenantWebhookRepoStub) DeleteWebhookEndpoint(_ context.Context, tenantID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	endpoint, ok := r.endpoints[id]
	if !ok || endpoint.TenantID != tenantID {
		return domain.ErrNotFound
	}
	delete(r.endpoints, id)
	return nil
}

func (r *tenantWebhookRepoStub) CreateWebhookDelivery(_ context.Context, delivery *domain.WebhookDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	delivery.CreatedAt = now
	delivery.UpdatedAt = now
	delivery.Payload = append(json.RawMessage(nil), delivery.Payload...)
	r.deliveries[delivery.ID] = *delivery
	return nil
}

func (r *tenantWebhookRepoStub) GetWebhookDelivery(_ context.Context, tenantID, id uuid.UUID) (*domain.WebhookDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delivery, ok := r.deliveries[id]
	if !ok || delivery.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return cloneDelivery(delivery), nil
}

func (r *tenantWebhookRepoStub) ListWebhookDeliveries(_ context.Context, tenantID uuid.UUID, webhookID *uuid.UUID, _, _ int) ([]domain.WebhookDelivery, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var deliveries []domain.WebhookDelivery
	for _, delivery := range r.deliveries {
		if delivery.TenantID != tenantID {
			continue
		}
		if webhookID != nil && delivery.WebhookID != *webhookID {
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries, len(deliveries), nil
}

func (r *tenantWebhookRepoStub) ListDueWebhookDeliveries(_ context.Context, before time.Time, limit int) ([]domain.WebhookDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var deliveries []domain.WebhookDelivery
	for _, delivery := range r.deliveries {
		if delivery.Status == domain.WebhookDeliveryRetrying && delivery.NextAttemptAt != nil && !delivery.NextAttemptAt.After(before) {
			deliveries = append(deliveries, delivery)
			if len(deliveries) == limit {
				break
			}
		}
	}
	return deliveries, nil
}

func (r *tenantWebhookRepoStub) UpdateWebhookDelivery(_ context.Context, delivery *domain.WebhookDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.deliveries[delivery.ID]; !ok {
		return domain.ErrNotFound
	}
	delivery.UpdatedAt = time.Now().UTC()
	delivery.Payload = append(json.RawMessage(nil), delivery.Payload...)
	r.deliveries[delivery.ID] = *delivery
	return nil
}

func cloneEndpoint(endpoint domain.WebhookEndpoint) *domain.WebhookEndpoint {
	endpoint.Events = append([]domain.WebhookEventType(nil), endpoint.Events...)
	return &endpoint
}

func cloneDelivery(delivery domain.WebhookDelivery) *domain.WebhookDelivery {
	delivery.Payload = append(json.RawMessage(nil), delivery.Payload...)
	return &delivery
}
