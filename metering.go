package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// BillableMetric is a tenant-defined meter over usage events. Code doubles
// as the usage event dimension it aggregates and is unique per tenant.
type BillableMetric struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	Name            string    `json:"name"`
	Code            string    `json:"code"`
	AggregationType string    `json:"aggregation_type"` // "count" | "sum" | "max" | "unique"
	FieldName       string    `json:"field_name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BillableMetricParams creates or updates a metric. Code is immutable after
// creation. FieldName is required for the "unique" aggregation (the event
// property whose distinct values are counted) and forbidden otherwise.
type BillableMetricParams struct {
	Name            string `json:"name"`
	Code            string `json:"code"`
	AggregationType string `json:"aggregation_type"`
	FieldName       string `json:"field_name,omitempty"`
}

// ChargeTier is one band of a graduated or volume charge. UpTo nil means
// unbounded (last tier only). UnitAmount is a decimal string in MAJOR
// currency units (e.g. "0.0035"); FlatAmount is minor units.
type ChargeTier struct {
	UpTo       *int64 `json:"up_to"`
	UnitAmount string `json:"unit_amount"`
	FlatAmount int64  `json:"flat_amount,omitempty"`
}

// ChargeAmounts is a charge's pricing for one currency; which fields apply
// depends on the charge model.
type ChargeAmounts struct {
	UnitAmount    string       `json:"unit_amount,omitempty"`    // per_unit
	PackageAmount int64        `json:"package_amount,omitempty"` // package, minor units per bundle
	PackageSize   int64        `json:"package_size,omitempty"`   // package, units per bundle
	Tiers         []ChargeTier `json:"tiers,omitempty"`          // graduated | volume
}

// Charge attaches usage pricing for a billable metric to a plan. Usage is
// billed in arrears at period close; flat plan prices are unaffected.
type Charge struct {
	ID          string                   `json:"id"`
	PlanID      string                   `json:"plan_id"`
	MetricID    string                   `json:"metric_id"`
	ChargeModel string                   `json:"charge_model"` // "per_unit" | "graduated" | "volume" | "package"
	Amounts     map[string]ChargeAmounts `json:"amounts"`
	HSNCode     string                   `json:"hsn_code,omitempty"`
	Metric      *BillableMetric          `json:"metric,omitempty"`
}

// ChargeParams is one charge in a plan's charge set (PUT replace semantics).
type ChargeParams struct {
	MetricID    string                   `json:"metric_id"`
	ChargeModel string                   `json:"charge_model"`
	Amounts     map[string]ChargeAmounts `json:"amounts"`
	HSNCode     string                   `json:"hsn_code,omitempty"`
}

// UsageAmountCharge is one charge's live preview entry.
type UsageAmountCharge struct {
	MetricCode      string `json:"metric_code"`
	MetricName      string `json:"metric_name"`
	AggregationType string `json:"aggregation_type"`
	ChargeModel     string `json:"charge_model"`
	Quantity        int64  `json:"quantity"`
	Amount          int64  `json:"amount"` // minor currency units
}

// UsageAmount is the live preview of what the current period's usage would
// rate to if invoiced now.
type UsageAmount struct {
	SubscriptionID     string              `json:"subscription_id"`
	Currency           string              `json:"currency"`
	CurrentPeriodStart time.Time           `json:"current_period_start"`
	AsOf               time.Time           `json:"as_of"`
	Charges            []UsageAmountCharge `json:"charges"`
	TotalAmount        int64               `json:"total_amount"`
}

// BillableMetricsService groups the billable-metric endpoints
// (usage-based billing v1).
type BillableMetricsService struct{ client *Client }

// Create creates a billable metric.
func (s *BillableMetricsService) Create(ctx context.Context, params *BillableMetricParams) (*BillableMetric, error) {
	return getData[*BillableMetric](ctx, s.client, http.MethodPost, "/billable-metrics", params)
}

// List returns the tenant's billable metrics.
func (s *BillableMetricsService) List(ctx context.Context) ([]BillableMetric, error) {
	return getData[[]BillableMetric](ctx, s.client, http.MethodGet, "/billable-metrics", nil)
}

// Get fetches one billable metric.
func (s *BillableMetricsService) Get(ctx context.Context, id string) (*BillableMetric, error) {
	return getData[*BillableMetric](ctx, s.client, http.MethodGet, fmt.Sprintf("/billable-metrics/%s", id), nil)
}

// Update changes a metric's name/aggregation/field; Code is immutable.
func (s *BillableMetricsService) Update(ctx context.Context, id string, params *BillableMetricParams) (*BillableMetric, error) {
	return getData[*BillableMetric](ctx, s.client, http.MethodPut, fmt.Sprintf("/billable-metrics/%s", id), params)
}

// Delete removes a metric. The API answers 409 while a plan charge still
// references it.
func (s *BillableMetricsService) Delete(ctx context.Context, id string) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/billable-metrics/%s", id), nil, nil)
}

// SetCharges replaces a plan's full usage-charge set (charges absent from
// the list are removed), mirroring SetForPlan entitlement semantics.
func (s *PlansService) SetCharges(ctx context.Context, planID string, charges []ChargeParams) ([]Charge, error) {
	return getData[[]Charge](ctx, s.client, http.MethodPut, fmt.Sprintf("/plans/%s/charges", planID), charges)
}

// GetCharges lists a plan's usage charges with their metrics joined.
func (s *PlansService) GetCharges(ctx context.Context, planID string) ([]Charge, error) {
	return getData[[]Charge](ctx, s.client, http.MethodGet, fmt.Sprintf("/plans/%s/charges", planID), nil)
}

// UsageAmount previews what the subscription's current-period usage would
// rate to if invoiced now.
func (s *SubscriptionsService) UsageAmount(ctx context.Context, id string) (*UsageAmount, error) {
	return getData[*UsageAmount](ctx, s.client, http.MethodGet, fmt.Sprintf("/subscriptions/%s/usage-amount", id), nil)
}
