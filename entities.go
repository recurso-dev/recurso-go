package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Entity is a legal entity under the workspace (Multi-Entity Books): its own
// general ledger, gapless invoice series, and tax identity. Every workspace
// has exactly one primary entity, which cannot be deleted.
type Entity struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	LegalName     string    `json:"legal_name"`
	IsPrimary     bool      `json:"is_primary"`
	TBLedgerID    int       `json:"tb_ledger_id"`
	InvoicePrefix string    `json:"invoice_prefix"`
	CountryCode   string    `json:"country_code"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EntityParams creates or updates an entity. InvoicePrefix must be unique in
// the workspace; it defaults to a slug of the name when empty on create.
type EntityParams struct {
	Name          string `json:"name"`
	LegalName     string `json:"legal_name,omitempty"`
	InvoicePrefix string `json:"invoice_prefix,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
}

// EntityOverviewRow is one entity's headline health in the reporting currency
// (minor units).
type EntityOverviewRow struct {
	EntityID      string `json:"entity_id"`
	EntityName    string `json:"entity_name"`
	IsPrimary     bool   `json:"is_primary"`
	MRR           int64  `json:"mrr"`
	ARR           int64  `json:"arr"`
	AROutstanding int64  `json:"ar_outstanding"`
	Subscriptions int    `json:"subscriptions"`
}

// EntitiesOverview is the multi-entity control tower: every entity's MRR and
// open receivables side by side, plus consolidated totals. Empty for
// single-entity workspaces.
type EntitiesOverview struct {
	ReportingCurrency  string              `json:"reporting_currency"`
	TotalMRR           int64               `json:"total_mrr"`
	TotalAROutstanding int64               `json:"total_ar_outstanding"`
	Entities           []EntityOverviewRow `json:"entities"`
}

// EntitiesService manages legal entities and their overview.
type EntitiesService struct{ client *Client }

// List returns the workspace's legal entities.
func (s *EntitiesService) List(ctx context.Context) ([]Entity, error) {
	return getData[[]Entity](ctx, s.client, http.MethodGet, "/entities", nil)
}

// Create adds a legal entity.
func (s *EntitiesService) Create(ctx context.Context, params *EntityParams) (*Entity, error) {
	return getData[*Entity](ctx, s.client, http.MethodPost, "/entities", params)
}

// Get fetches one entity.
func (s *EntitiesService) Get(ctx context.Context, id string) (*Entity, error) {
	return getData[*Entity](ctx, s.client, http.MethodGet, fmt.Sprintf("/entities/%s", id), nil)
}

// Update edits an entity's names, prefix, or country.
func (s *EntitiesService) Update(ctx context.Context, id string, params *EntityParams) (*Entity, error) {
	return getData[*Entity](ctx, s.client, http.MethodPut, fmt.Sprintf("/entities/%s", id), params)
}

// Delete removes a non-primary entity (the primary cannot be deleted).
func (s *EntitiesService) Delete(ctx context.Context, id string) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/entities/%s", id), nil, nil)
}

// Overview returns per-entity MRR + open AR with consolidated totals,
// FX-normalized to the reporting currency.
func (s *EntitiesService) Overview(ctx context.Context) (*EntitiesOverview, error) {
	return getData[*EntitiesOverview](ctx, s.client, http.MethodGet, "/analytics/entities-overview", nil)
}
