package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

const (
	defaultWebhookPerPage = 25
	maxWebhookPerPage     = 100
	maxWebhookAttempts    = 5
	maxResponseBodyBytes  = 4096
)

type TenantWebhookService struct {
	repo   ports.WebhookRepository
	client *http.Client
	log    Logger
	now    func() time.Time
}

func NewTenantWebhookService(repo ports.WebhookRepository, client *http.Client, log Logger) *TenantWebhookService {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &TenantWebhookService{
		repo:   repo,
		client: client,
		log:    log,
		now:    time.Now,
	}
}

func (s *TenantWebhookService) CreateEndpoint(ctx context.Context, tenantID uuid.UUID, req ports.CreateWebhookEndpointRequest) (*domain.WebhookEndpoint, error) {
	if !domain.IsValidWebhookURL(req.URL) {
		return nil, domain.NewValidationError("url", "webhook URL must be HTTPS")
	}
	if err := validateWebhookEvents(req.Events); err != nil {
		return nil, err
	}
	secret := strings.TrimSpace(req.Secret)
	if secret == "" {
		var err error
		secret, err = generateWebhookSecret()
		if err != nil {
			return nil, err
		}
	}
	if len(secret) < 32 {
		return nil, domain.NewValidationError("secret", "webhook secret must be at least 32 characters")
	}

	endpoint := &domain.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: tenantID,
		URL:      req.URL,
		Secret:   secret,
		Events:   req.Events,
		IsActive: true,
	}
	if err := s.repo.CreateWebhookEndpoint(ctx, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (s *TenantWebhookService) GetEndpoint(ctx context.Context, tenantID, id uuid.UUID) (*domain.WebhookEndpoint, error) {
	return s.repo.GetWebhookEndpoint(ctx, tenantID, id)
}

func (s *TenantWebhookService) ListEndpoints(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WebhookEndpoint, int, error) {
	page, perPage = normalizePage(page, perPage, defaultWebhookPerPage, maxWebhookPerPage)
	return s.repo.ListWebhookEndpoints(ctx, tenantID, page, perPage)
}

func (s *TenantWebhookService) UpdateEndpoint(ctx context.Context, tenantID, id uuid.UUID, req ports.UpdateWebhookEndpointRequest) (*domain.WebhookEndpoint, error) {
	endpoint, err := s.repo.GetWebhookEndpoint(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if req.URL != nil {
		if !domain.IsValidWebhookURL(*req.URL) {
			return nil, domain.NewValidationError("url", "webhook URL must be HTTPS")
		}
		endpoint.URL = *req.URL
	}
	if req.Events != nil {
		if err := validateWebhookEvents(*req.Events); err != nil {
			return nil, err
		}
		endpoint.Events = *req.Events
	}
	if req.Secret != nil {
		secret := strings.TrimSpace(*req.Secret)
		if secret == "" {
			var err error
			secret, err = generateWebhookSecret()
			if err != nil {
				return nil, err
			}
		}
		if len(secret) < 32 {
			return nil, domain.NewValidationError("secret", "webhook secret must be at least 32 characters")
		}
		endpoint.Secret = secret
	}
	if req.IsActive != nil {
		endpoint.IsActive = *req.IsActive
	}
	if err := s.repo.UpdateWebhookEndpoint(ctx, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (s *TenantWebhookService) DeleteEndpoint(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteWebhookEndpoint(ctx, tenantID, id)
}

func (s *TenantWebhookService) ListDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID *uuid.UUID, page, perPage int) ([]domain.WebhookDelivery, int, error) {
	page, perPage = normalizePage(page, perPage, defaultWebhookPerPage, maxWebhookPerPage)
	return s.repo.ListWebhookDeliveries(ctx, tenantID, webhookID, page, perPage)
}

func (s *TenantWebhookService) TestEndpoint(ctx context.Context, tenantID, id uuid.UUID) (*domain.WebhookDelivery, error) {
	endpoint, err := s.repo.GetWebhookEndpoint(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		"event_id":    uuid.New().String(),
		"event_type":  string(domain.WebhookEventTest),
		"tenant_id":   tenantID.String(),
		"webhook_id":  id.String(),
		"occurred_at": s.now().UTC().Format(time.RFC3339Nano),
		"data": map[string]string{
			"message": "Webhook test event",
		},
	})
	if err != nil {
		return nil, err
	}
	delivery := newWebhookDelivery(tenantID, endpoint.ID, domain.WebhookEventTest, payload)
	if err := s.repo.CreateWebhookDelivery(ctx, &delivery); err != nil {
		return nil, err
	}
	if err := s.attemptDelivery(ctx, endpoint, &delivery); err != nil {
		return nil, err
	}
	return &delivery, nil
}

func (s *TenantWebhookService) ReplayDelivery(ctx context.Context, tenantID, id uuid.UUID) (*domain.WebhookDelivery, error) {
	delivery, err := s.repo.GetWebhookDelivery(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	endpoint, err := s.repo.GetWebhookEndpoint(ctx, tenantID, delivery.WebhookID)
	if err != nil {
		return nil, err
	}
	if err := s.attemptDelivery(ctx, endpoint, delivery); err != nil {
		return nil, err
	}
	return delivery, nil
}

func (s *TenantWebhookService) DeliverEvent(ctx context.Context, tenantID uuid.UUID, eventType domain.WebhookEventType, payload json.RawMessage) ([]domain.WebhookDelivery, error) {
	if !domain.IsValidWebhookEvent(eventType) {
		return nil, domain.NewValidationError("event_type", "unsupported webhook event type")
	}
	endpoints, _, err := s.repo.ListWebhookEndpoints(ctx, tenantID, 0, maxWebhookPerPage)
	if err != nil {
		return nil, err
	}

	deliveries := make([]domain.WebhookDelivery, 0, len(endpoints))
	for i := range endpoints {
		endpoint := endpoints[i]
		if !endpoint.SubscribesTo(eventType) {
			continue
		}
		delivery := newWebhookDelivery(tenantID, endpoint.ID, eventType, payload)
		if err := s.repo.CreateWebhookDelivery(ctx, &delivery); err != nil {
			return deliveries, err
		}
		if err := s.attemptDelivery(ctx, &endpoint, &delivery); err != nil {
			return deliveries, err
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries, nil
}

func (s *TenantWebhookService) ProcessDueDeliveries(ctx context.Context, limit int) error {
	if limit <= 0 || limit > maxWebhookPerPage {
		limit = maxWebhookPerPage
	}
	deliveries, err := s.repo.ListDueWebhookDeliveries(ctx, s.now(), limit)
	if err != nil {
		return err
	}
	for i := range deliveries {
		delivery := deliveries[i]
		endpoint, err := s.repo.GetWebhookEndpoint(ctx, delivery.TenantID, delivery.WebhookID)
		if err != nil {
			if s.log != nil {
				s.log.Warn("webhook endpoint missing for retry", "delivery_id", delivery.ID, "error", err)
			}
			continue
		}
		if err := s.attemptDelivery(ctx, endpoint, &delivery); err != nil && s.log != nil {
			s.log.Warn("webhook retry failed", "delivery_id", delivery.ID, "error", err)
		}
	}
	return nil
}

func (s *TenantWebhookService) attemptDelivery(ctx context.Context, endpoint *domain.WebhookEndpoint, delivery *domain.WebhookDelivery) error {
	now := s.now().UTC()
	signature := domain.SignWebhookPayload(endpoint.Secret, now, delivery.Payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(delivery.Payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "spur-messaging-webhooks/1.0")
	req.Header.Set("X-Spur-Event-ID", delivery.EventID.String())
	req.Header.Set("X-Spur-Event-Type", string(delivery.EventType))
	req.Header.Set("X-Spur-Timestamp", fmt.Sprintf("%d", now.Unix()))
	req.Header.Set("X-Spur-Signature", signature)

	delivery.AttemptCount++
	delivery.LastAttemptAt = &now
	delivery.Signature = signature

	resp, err := s.client.Do(req)
	if err != nil {
		msg := err.Error()
		delivery.ErrorMessage = &msg
		delivery.ResponseStatus = nil
		delivery.ResponseBody = nil
		setNextWebhookAttempt(s.now, delivery)
		return s.repo.UpdateWebhookDelivery(ctx, delivery)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	bodyText := string(body)
	status := resp.StatusCode
	delivery.ResponseStatus = &status
	delivery.ResponseBody = &bodyText
	delivery.ErrorMessage = nil
	if status >= 200 && status < 300 {
		delivery.Status = domain.WebhookDeliverySucceeded
		delivery.NextAttemptAt = nil
	} else {
		msg := fmt.Sprintf("webhook endpoint returned HTTP %d", status)
		delivery.ErrorMessage = &msg
		setNextWebhookAttempt(s.now, delivery)
	}
	return s.repo.UpdateWebhookDelivery(ctx, delivery)
}

func setNextWebhookAttempt(now func() time.Time, delivery *domain.WebhookDelivery) {
	if delivery.AttemptCount >= maxWebhookAttempts {
		delivery.Status = domain.WebhookDeliveryFailed
		delivery.NextAttemptAt = nil
		return
	}
	delay := time.Duration(1<<max(delivery.AttemptCount-1, 0)) * time.Minute
	next := now().UTC().Add(delay)
	delivery.Status = domain.WebhookDeliveryRetrying
	delivery.NextAttemptAt = &next
}

func newWebhookDelivery(tenantID, webhookID uuid.UUID, eventType domain.WebhookEventType, payload json.RawMessage) domain.WebhookDelivery {
	eventID := uuid.New()
	var envelope struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil && envelope.EventID != "" {
		if parsed, err := uuid.Parse(envelope.EventID); err == nil {
			eventID = parsed
		}
	}
	return domain.WebhookDelivery{
		ID:           uuid.New(),
		TenantID:     tenantID,
		WebhookID:    webhookID,
		EventID:      eventID,
		EventType:    eventType,
		Payload:      append(json.RawMessage(nil), payload...),
		Status:       domain.WebhookDeliveryPending,
		AttemptCount: 0,
	}
}

func validateWebhookEvents(events []domain.WebhookEventType) error {
	if len(events) == 0 {
		return domain.NewValidationError("events", "at least one webhook event is required")
	}
	for _, event := range events {
		if !domain.IsValidWebhookEvent(event) {
			return domain.NewValidationError("events", "unsupported webhook event type")
		}
	}
	return nil
}

func generateWebhookSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func normalizePage(page, perPage, defaultPerPage, maxPerPage int) (int, int) {
	if page < 0 {
		page = 0
	}
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return page, perPage
}
