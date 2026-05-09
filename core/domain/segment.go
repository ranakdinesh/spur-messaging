package domain

import (
	"time"

	"github.com/google/uuid"
)

type Segment struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	IsDynamic bool
	Rules     []SegmentRule // for dynamic segments
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SegmentRule struct {
	Field    string `json:"field"`    // "tags", "opt_in_whatsapp", "attributes.city"
	Operator string `json:"operator"` // "eq", "neq", "contains", "gt", "lt", "in"
	Value    string `json:"value"`
}
