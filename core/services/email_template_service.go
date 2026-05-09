package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type EmailTemplateService struct {
	repo ports.EmailTemplateRepository
}

func NewEmailTemplateService(repo ports.EmailTemplateRepository) *EmailTemplateService {
	return &EmailTemplateService{repo: repo}
}

func (s *EmailTemplateService) Create(ctx context.Context, tenantID uuid.UUID, req ports.CreateEmailTemplateRequest) (*domain.EmailTemplate, error) {
	tmpl := &domain.EmailTemplate{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        req.Name,
		Subject:     req.Subject,
		PreviewText: req.PreviewText,
		HTMLBody:    req.HTMLBody,
		TextBody:    req.TextBody,
		Category:    req.Category,
		Variables:   req.Variables,
		IsActive:    true,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if tmpl.TextBody == "" {
		tmpl.TextBody = s.stripHTML(tmpl.HTMLBody)
	}

	err := s.repo.Create(ctx, tmpl)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s *EmailTemplateService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmailTemplate, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *EmailTemplateService) List(ctx context.Context, tenantID uuid.UUID, category *domain.EmailCategory, page, perPage int) ([]domain.EmailTemplate, int, error) {
	return s.repo.List(ctx, tenantID, category, page, perPage)
}

func (s *EmailTemplateService) Update(ctx context.Context, tenantID, id uuid.UUID, req ports.UpdateEmailTemplateRequest) (*domain.EmailTemplate, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Subject != nil {
		tmpl.Subject = *req.Subject
	}
	if req.PreviewText != nil {
		tmpl.PreviewText = *req.PreviewText
	}
	if req.HTMLBody != nil {
		tmpl.HTMLBody = *req.HTMLBody
	}
	if req.TextBody != nil {
		tmpl.TextBody = *req.TextBody
	}
	if req.Category != nil {
		tmpl.Category = *req.Category
	}
	if req.Variables != nil {
		tmpl.Variables = *req.Variables
	}
	if req.IsActive != nil {
		tmpl.IsActive = *req.IsActive
	}

	tmpl.Version++
	tmpl.UpdatedAt = time.Now()

	err = s.repo.Update(ctx, tmpl)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s *EmailTemplateService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *EmailTemplateService) Preview(ctx context.Context, tenantID, id uuid.UUID, variables map[string]string) (*ports.EmailPreview, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	subject := s.render(tmpl.Subject, variables)
	html := s.render(tmpl.HTMLBody, variables)
	text := s.render(tmpl.TextBody, variables)

	return &ports.EmailPreview{
		Subject:  subject,
		HTMLBody: html,
		TextBody: text,
	}, nil
}

func (s *EmailTemplateService) Duplicate(ctx context.Context, tenantID, id uuid.UUID, newName string) (*domain.EmailTemplate, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	newTmpl := *tmpl
	newTmpl.ID = uuid.New()
	newTmpl.Name = newName
	newTmpl.CreatedAt = time.Now()
	newTmpl.UpdatedAt = time.Now()
	newTmpl.Version = 1

	err = s.repo.Create(ctx, &newTmpl)
	if err != nil {
		return nil, err
	}

	return &newTmpl, nil
}

func (s *EmailTemplateService) render(content string, variables map[string]string) string {
	for k, v := range variables {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content
}

func (s *EmailTemplateService) stripHTML(html string) string {
	// Very basic HTML stripping for now. 
	// In production, use a proper library like bluemonday or html2text
	return html 
}
