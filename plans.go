package recurso

import (
	"context"
	"net/http"
	"time"
)

// Price is a currency-specific price attached to a plan.
type Price struct {
	ID        string    `json:"id"`
	PlanID    string    `json:"plan_id"`
	Currency  string    `json:"currency"`
	Amount    int64     `json:"amount"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// Plan is a product-catalog plan.
type Plan struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	IntervalUnit  string    `json:"interval_unit"`
	IntervalCount int       `json:"interval_count"`
	Active        bool      `json:"active"`
	HSNCode       string    `json:"hsn_code"`
	CreatedAt     time.Time `json:"created_at"`
	Prices        []Price   `json:"prices"`
}

// PlanCreateParams is the body for creating a plan. Amount is in the currency's
// smallest unit.
type PlanCreateParams struct {
	Name          string `json:"name"`
	Code          string `json:"code"`
	IntervalUnit  string `json:"interval_unit"`
	IntervalCount int    `json:"interval_count"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	HSNCode       string `json:"hsn_code,omitempty"`
}

// PlanUpdateParams is the body for updating a plan. Omitted fields are left
// unchanged. Set Active to false to archive the plan (hides it from new
// subscriptions without affecting existing ones) and to true to restore it.
// The plan's price is a separate versioned entity and is not editable here.
type PlanUpdateParams struct {
	Name          string `json:"name,omitempty"`
	HSNCode       string `json:"hsn_code,omitempty"`
	IntervalUnit  string `json:"interval_unit,omitempty"`
	IntervalCount int    `json:"interval_count,omitempty"`
	Active        *bool  `json:"active,omitempty"`
}

// PlanListParams filters the plan list.
type PlanListParams struct {
	Q     string
	Limit int
	Page  int
}

// PlansService groups the plan endpoints.
type PlansService struct{ client *Client }

// Create creates a plan.
func (s *PlansService) Create(ctx context.Context, params *PlanCreateParams) (*Plan, error) {
	var out Plan
	if err := s.client.do(ctx, http.MethodPost, "/plans", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get retrieves a plan by ID.
func (s *PlansService) Get(ctx context.Context, id string) (*Plan, error) {
	out, err := getData[Plan](ctx, s.client, http.MethodGet, "/plans/"+id, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update partially updates a plan. Set Active in params to archive or restore
// it.
func (s *PlansService) Update(ctx context.Context, id string, params *PlanUpdateParams) (*Plan, error) {
	var out Plan
	if err := s.client.do(ctx, http.MethodPut, "/plans/"+id, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's plans.
func (s *PlansService) List(ctx context.Context, params *PlanListParams) ([]Plan, error) {
	path := "/plans"
	if params != nil {
		path = newQuery().str("q", params.Q).int("limit", params.Limit).int("page", params.Page).apply(path)
	}
	return getData[[]Plan](ctx, s.client, http.MethodGet, path, nil)
}
