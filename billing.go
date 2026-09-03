package recurso

import (
	"context"
	"net/http"
	"time"
)

// BillingStatus is the tenant's own managed-cloud subscription/trial state.
type BillingStatus struct {
	BillingStatus string    `json:"billing_status"` // trialing | active | past_due | canceled
	PlanTier      string    `json:"plan_tier"`
	TrialEndsAt   time.Time `json:"trial_ends_at"`
	TrialDaysLeft int       `json:"trial_days_left"`
	TrialExpired  bool      `json:"trial_expired"`
}

// BillingPlan is one tier of the managed-cloud catalog. Price and Period are
// display strings.
type BillingPlan struct {
	Tier        string   `json:"tier"`
	Name        string   `json:"name"`
	Price       string   `json:"price"`
	Period      string   `json:"period"`
	FreeNote    string   `json:"free_note"`
	Features    []string `json:"features"`
	CTA         string   `json:"cta"`
	Recommended bool     `json:"recommended"`
}

// BillingService groups the tenant's own managed-cloud billing endpoints
// (what Recurso charges the tenant, not what the tenant charges its
// customers).
type BillingService struct{ client *Client }

// Status returns the tenant's managed-cloud billing/trial status.
func (s *BillingService) Status(ctx context.Context) (*BillingStatus, error) {
	var out BillingStatus
	if err := s.client.do(ctx, http.MethodGet, "/billing/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Plans returns the managed-cloud plan catalog.
func (s *BillingService) Plans(ctx context.Context) ([]BillingPlan, error) {
	var out struct {
		Plans []BillingPlan `json:"plans"`
	}
	if err := s.client.do(ctx, http.MethodGet, "/billing/plans", nil, &out); err != nil {
		return nil, err
	}
	return out.Plans, nil
}
