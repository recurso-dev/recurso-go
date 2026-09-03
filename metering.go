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
	ID              string `json:"id"`
	TenantID        string `json:"tenant_id"`
	Name            string `json:"name"`
	Code            string `json:"code"`
	AggregationType string `json:"aggregation_type"` // count | sum | max | unique | latest | percentile | weighted_sum | custom (field_name holds the percentile 1-99)
	FieldName       string `json:"field_name,omitempty"`
	// Expression is the per-event formula for the "custom" aggregation (e.g.
	// "quantity * properties.multiplier"); its results are summed over the
	// period. Set only for custom, empty otherwise.
	Expression string    `json:"expression,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BillableMetricParams creates or updates a metric. Code is immutable after
// creation. FieldName is required for the "unique" aggregation (the event
// property whose distinct values are counted) and the "percentile" aggregation
// (the percentile 1-99), and forbidden otherwise. Expression is required for
// the "custom" aggregation (a sandboxed per-event formula over "quantity" and
// numeric "properties.*", summed over the period) and forbidden otherwise.
// "weighted_sum" (time-weighted average of a running level built from per-event
// deltas) takes neither.
type BillableMetricParams struct {
	Name            string `json:"name"`
	Code            string `json:"code"`
	AggregationType string `json:"aggregation_type"`
	FieldName       string `json:"field_name,omitempty"`
	Expression      string `json:"expression,omitempty"`
}

// ChargeTier is one band of a graduated, volume, or graduated_percentage
// charge. UpTo nil means unbounded (last tier only). UnitAmount (per-unit
// tiers) is a decimal string in MAJOR currency units (e.g. "0.0035"); Rate
// (graduated_percentage) is a percent decimal (e.g. "2.5"); FlatAmount is
// minor units.
type ChargeTier struct {
	UpTo       *int64 `json:"up_to"`
	UnitAmount string `json:"unit_amount,omitempty"` // graduated | volume
	Rate       string `json:"rate,omitempty"`        // graduated_percentage
	FlatAmount int64  `json:"flat_amount,omitempty"`
}

// ChargeAmounts is a charge's pricing for one currency; which fields apply
// depends on the charge model. The percentage model prices a percentage of
// the aggregated monetary base (minor units); dynamic carries no pricing (the
// price is supplied per event as UsageRecordParams.DynamicAmount).
type ChargeAmounts struct {
	UnitAmount    string       `json:"unit_amount,omitempty"`    // per_unit
	PackageAmount int64        `json:"package_amount,omitempty"` // package, minor units per bundle
	PackageSize   int64        `json:"package_size,omitempty"`   // package, units per bundle
	Tiers         []ChargeTier `json:"tiers,omitempty"`          // graduated | volume | graduated_percentage
	Rate          string       `json:"rate,omitempty"`           // percentage, percent decimal e.g. "2.5"
	FixedAmount   int64        `json:"fixed_amount,omitempty"`   // percentage, flat fee (minor units)
	FreeUnits     int64        `json:"free_units,omitempty"`     // percentage, base exempt (minor units)
	MinAmount     int64        `json:"min_amount,omitempty"`     // percentage, floor (minor units; 0 = none)
	MaxAmount     int64        `json:"max_amount,omitempty"`     // percentage, cap (minor units; 0 = none)
}

// Charge attaches usage pricing for a billable metric to a plan. Usage is
// billed in arrears at period close; flat plan prices are unaffected.
type Charge struct {
	ID          string                   `json:"id"`
	PlanID      string                   `json:"plan_id"`
	MetricID    string                   `json:"metric_id"`
	ChargeModel string                   `json:"charge_model"` // per_unit | graduated | volume | package | percentage | graduated_percentage | dynamic
	Amounts     map[string]ChargeAmounts `json:"amounts"`
	// PayInAdvance rates the charge per usage event at ingestion time (captured
	// as an unbilled charge, folded onto the next invoice). Only per_unit,
	// percentage, and dynamic may set it.
	PayInAdvance bool            `json:"pay_in_advance,omitempty"`
	HSNCode      string          `json:"hsn_code,omitempty"`
	Metric       *BillableMetric `json:"metric,omitempty"`
}

// ChargeFilter prices one value of a charge's FilterKey property with its own
// amounts (e.g. per-region rates).
type ChargeFilter struct {
	Value   string                   `json:"value"`
	Amounts map[string]ChargeAmounts `json:"amounts"`
}

// ChargeParams is one charge in a plan's charge set (PUT replace semantics).
type ChargeParams struct {
	MetricID    string                   `json:"metric_id"`
	ChargeModel string                   `json:"charge_model"`
	Amounts     map[string]ChargeAmounts `json:"amounts"`
	// FilterKey names the usage-event property whose values Filters price
	// separately; both are optional.
	FilterKey string         `json:"filter_key,omitempty"`
	Filters   []ChargeFilter `json:"filters,omitempty"`
	// PayInAdvance: non-cumulative models only (per_unit/percentage/dynamic).
	PayInAdvance bool   `json:"pay_in_advance,omitempty"`
	HSNCode      string `json:"hsn_code,omitempty"`
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

// MetricCharge is one plan charge priced on a billable metric (reverse
// lookup from the meter).
type MetricCharge struct {
	ChargeID     string `json:"charge_id"`
	PlanID       string `json:"plan_id"`
	PlanName     string `json:"plan_name"`
	PlanCode     string `json:"plan_code"`
	PlanActive   bool   `json:"plan_active"`
	ChargeModel  string `json:"charge_model"`
	PayInAdvance bool   `json:"pay_in_advance"`
}

// Charges lists every plan charge priced on the metric.
func (s *BillableMetricsService) Charges(ctx context.Context, id string) ([]MetricCharge, error) {
	return getData[[]MetricCharge](ctx, s.client, http.MethodGet, fmt.Sprintf("/billable-metrics/%s/charges", id), nil)
}

// SimulateUsage is a hypothetical usage quantity for one metric.
type SimulateUsage struct {
	MetricID string `json:"metric_id"`
	Quantity int64  `json:"quantity"`
}

// SimulateChargesParams is a proposed charge set plus the usage to rate it
// against. SubscriptionID, when set, uses that subscription's current-period
// usage instead of Usage.
type SimulateChargesParams struct {
	Currency       string          `json:"currency,omitempty"`
	SubscriptionID string          `json:"subscription_id,omitempty"`
	Charges        []ChargeParams  `json:"charges"`
	Usage          []SimulateUsage `json:"usage,omitempty"`
}

// SimulatedCharge is one charge's rated amount in a simulation (minor
// units).
type SimulatedCharge struct {
	MetricID    string `json:"metric_id"`
	MetricCode  string `json:"metric_code"`
	MetricName  string `json:"metric_name"`
	ChargeModel string `json:"charge_model"`
	Quantity    int64  `json:"quantity"`
	Amount      int64  `json:"amount"`
}

// SimulatedGLLine is one line of the journal entry the simulated invoice
// would post (minor units).
type SimulatedGLLine struct {
	AccountCode int    `json:"account_code"`
	AccountName string `json:"account_name"`
	Debit       int64  `json:"debit"`
	Credit      int64  `json:"credit"`
}

// ChargeSimulation is the read-only result of rating a proposed charge set:
// per-charge amounts, the subtotal, and the GL preview. Nothing is
// persisted.
type ChargeSimulation struct {
	PlanID    string            `json:"plan_id"`
	Currency  string            `json:"currency"`
	Charges   []SimulatedCharge `json:"charges"`
	Subtotal  int64             `json:"subtotal"`
	GLPreview []SimulatedGLLine `json:"gl_preview"`
	Balanced  bool              `json:"balanced"`
	Note      string            `json:"note"`
}

// SimulateCharges rates a proposed charge set for a plan without saving it.
func (s *PlansService) SimulateCharges(ctx context.Context, planID string, params *SimulateChargesParams) (*ChargeSimulation, error) {
	return getData[*ChargeSimulation](ctx, s.client, http.MethodPost, fmt.Sprintf("/plans/%s/simulate-charges", planID), params)
}
