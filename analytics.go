package recurso

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// MRRCurrencyBreakdown is per-currency MRR before and after FX normalization.
type MRRCurrencyBreakdown struct {
	Currency        string  `json:"currency"`
	Amount          int64   `json:"amount"`
	ConvertedAmount int64   `json:"converted_amount"`
	Rate            float64 `json:"rate"`
	Subscriptions   int     `json:"subscriptions"`
	Error           string  `json:"error,omitempty"`
}

// FXSnapshot is the FX rate set used to normalize MRR.
type FXSnapshot struct {
	Rates  map[string]float64 `json:"rates"`
	Source string             `json:"source"`
	AsOf   time.Time          `json:"as_of"`
}

// MRRMetrics is the monthly-recurring-revenue report.
type MRRMetrics struct {
	Currency          string                 `json:"currency"`
	Amount            int64                  `json:"amount"`
	MRR               int64                  `json:"mrr"`
	NormalizedMRR     int64                  `json:"normalized_mrr"`
	ReportingCurrency string                 `json:"reporting_currency"`
	Breakdown         []MRRCurrencyBreakdown `json:"breakdown"`
	FX                FXSnapshot             `json:"fx"`
}

// AnalyticsService groups the revenue-analytics endpoints.
type AnalyticsService struct{ client *Client }

// MRR returns the tenant's current monthly recurring revenue, FX-normalized to
// the reporting currency.
func (s *AnalyticsService) MRR(ctx context.Context) (*MRRMetrics, error) {
	// The API wraps responses in {"data": ...}; decoding without the envelope
	// (the previous implementation) silently returned zeros.
	return getData[*MRRMetrics](ctx, s.client, http.MethodGet, "/analytics/mrr", nil)
}

// MRRForEntity returns MRR scoped to one legal entity (Multi-Entity Books).
func (s *AnalyticsService) MRRForEntity(ctx context.Context, entityID string) (*MRRMetrics, error) {
	path := newQuery().str("entity_id", entityID).apply("/analytics/mrr")
	return getData[*MRRMetrics](ctx, s.client, http.MethodGet, path, nil)
}

// MRREntityBreakdown is one legal entity's MRR contribution in the reporting
// currency (minor units).
type MRREntityBreakdown struct {
	EntityID      string `json:"entity_id"`
	EntityName    string `json:"entity_name"`
	IsPrimary     bool   `json:"is_primary"`
	NormalizedMRR int64  `json:"normalized_mrr"`
	ARR           int64  `json:"arr"`
	Subscriptions int    `json:"subscriptions"`
}

// MRRByEntity is the per-entity MRR breakdown; TotalMRR equals the
// consolidated MRR figure.
type MRRByEntity struct {
	ReportingCurrency string               `json:"reporting_currency"`
	TotalMRR          int64                `json:"total_mrr"`
	Entities          []MRREntityBreakdown `json:"entities"`
}

// MRRByEntity breaks MRR down across every legal entity, sorted by MRR
// descending. Single-entity workspaces get one row.
func (s *AnalyticsService) MRRByEntity(ctx context.Context) (*MRRByEntity, error) {
	return getData[*MRRByEntity](ctx, s.client, http.MethodGet, "/analytics/mrr/by-entity", nil)
}

// InvoiceAgingBucket is one age band of outstanding receivables.
type InvoiceAgingBucket struct {
	Label  string `json:"label"` // "current" | "1-30" | "31-60" | "61-90" | "90+"
	Count  int    `json:"count"`
	Amount int64  `json:"amount"`
}

// InvoiceAgingReport is outstanding AR bucketed by how far past due each open
// invoice is, FX-normalized to the reporting currency.
type InvoiceAgingReport struct {
	ReportingCurrency string               `json:"reporting_currency"`
	TotalOutstanding  int64                `json:"total_outstanding"`
	TotalCount        int                  `json:"total_count"`
	Buckets           []InvoiceAgingBucket `json:"buckets"`
}

// InvoiceAging returns the AR aging report. entityID scopes to one legal
// entity; empty = the whole workspace.
func (s *AnalyticsService) InvoiceAging(ctx context.Context, entityID string) (*InvoiceAgingReport, error) {
	path := "/analytics/invoice-aging"
	if entityID != "" {
		path = newQuery().str("entity_id", entityID).apply(path)
	}
	return getData[*InvoiceAgingReport](ctx, s.client, http.MethodGet, path, nil)
}

// DunningTimingRate is one time bucket's retry success rate (UTC).
type DunningTimingRate struct {
	Bucket      int     `json:"bucket"` // hour 0-23 or day-of-week 0-6 (Sunday=0)
	Total       int     `json:"total"`
	Successes   int     `json:"successes"`
	SuccessRate float64 `json:"success_rate"`
}

// DunningTimingInsights answers "when do retries succeed?" from historical
// outcomes. BestHour/BestDay are nil until enough history accrues.
type DunningTimingInsights struct {
	ByHour      []DunningTimingRate `json:"by_hour"`
	ByDayOfWeek []DunningTimingRate `json:"by_day_of_week"`
	BestHour    *int                `json:"best_hour,omitempty"`
	BestDay     *int                `json:"best_day,omitempty"`
	SampleSize  int                 `json:"sample_size"`
}

// DunningTiming returns best-time-to-retry insights (read-only; the live
// retry engine learns independently).
func (s *AnalyticsService) DunningTiming(ctx context.Context) (*DunningTimingInsights, error) {
	return getData[*DunningTimingInsights](ctx, s.client, http.MethodGet, "/analytics/dunning/timing", nil)
}

// AnalyticsAnswer is the result of a natural-language analytics question:
// the rows the generated query returned and the SQL that produced them.
type AnalyticsAnswer struct {
	Data  json.RawMessage `json:"data"`
	Query string          `json:"query"`
}

// Ask answers a natural-language question about the tenant's billing data.
func (s *AnalyticsService) Ask(ctx context.Context, question string) (*AnalyticsAnswer, error) {
	body := map[string]string{"question": question}
	var out AnalyticsAnswer
	if err := s.client.do(ctx, http.MethodPost, "/analytics/ask", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DunningOverview is the aggregate dunning retry success rate.
type DunningOverview struct {
	TotalRetries   int     `json:"total_retries"`
	TotalSuccesses int     `json:"total_successes"`
	SuccessRate    float64 `json:"success_rate"`
}

// DunningOverview returns the aggregate retry success rate.
func (s *AnalyticsService) DunningOverview(ctx context.Context) (*DunningOverview, error) {
	var out DunningOverview
	if err := s.client.do(ctx, http.MethodGet, "/analytics/dunning/overview", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DunningWeight is one learned (context, retry interval) reward.
type DunningWeight struct {
	ContextKey    string    `json:"context_key"`
	ActionID      string    `json:"action_id"`
	AverageReward float64   `json:"average_reward"`
	SampleCount   int64     `json:"sample_count"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DunningWeights returns the learned retry-timing weights.
func (s *AnalyticsService) DunningWeights(ctx context.Context) ([]DunningWeight, error) {
	return getData[[]DunningWeight](ctx, s.client, http.MethodGet, "/analytics/dunning/weights", nil)
}

// DunningAttempt is one recorded dunning retry and its outcome.
type DunningAttempt struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	InvoiceID     string    `json:"invoice_id"`
	ContextKey    string    `json:"context_key"`
	ActionID      string    `json:"action_id"`
	RetryInterval int64     `json:"retry_interval"`
	Outcome       string    `json:"outcome"` // success | failure
	Reward        float64   `json:"reward"`
	CreatedAt     time.Time `json:"created_at"`
}

// DunningHistory returns recent retry attempts, newest first. limit 0 uses
// the API default.
func (s *AnalyticsService) DunningHistory(ctx context.Context, limit int) ([]DunningAttempt, error) {
	path := newQuery().int("limit", limit).apply("/analytics/dunning/history")
	return getData[[]DunningAttempt](ctx, s.client, http.MethodGet, path, nil)
}

// DunningRecoveredMonth is revenue recovered by dunning in one month and
// currency (minor units).
type DunningRecoveredMonth struct {
	Month    string `json:"month"`
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
	Count    int    `json:"count"`
}

// DunningRecovered attributes recovered revenue to dunning: totals per
// currency (minor units), how many invoices, and how long recovery took.
type DunningRecovered struct {
	RecoveredAmountTotal map[string]int64        `json:"recovered_amount_total"`
	RecoveredCount       int                     `json:"recovered_count"`
	AvgAttempts          float64                 `json:"avg_attempts"`
	AvgDaysToRecover     float64                 `json:"avg_days_to_recover"`
	Monthly              []DunningRecoveredMonth `json:"monthly"`
}

// DunningRecovered returns recovered-revenue attribution.
func (s *AnalyticsService) DunningRecovered(ctx context.Context) (*DunningRecovered, error) {
	var out DunningRecovered
	if err := s.client.do(ctx, http.MethodGet, "/analytics/dunning/recovered", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MRRWaterfall is MRR movement between two dates (minor units in the
// reporting currency). Contraction and Churned are positive magnitudes.
// HasStartHistory is false when the start predates MRR snapshots, in which
// case movement is not accurate.
type MRRWaterfall struct {
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	StartingMRR          int64     `json:"starting_mrr"`
	New                  int64     `json:"new"`
	Expansion            int64     `json:"expansion"`
	Contraction          int64     `json:"contraction"`
	Churned              int64     `json:"churned"`
	Reactivation         int64     `json:"reactivation"`
	EndingMRR            int64     `json:"ending_mrr"`
	ReportingCurrency    string    `json:"reporting_currency"`
	NetDollarRetention   float64   `json:"net_dollar_retention"`
	GrossDollarRetention float64   `json:"gross_dollar_retention"`
	HasStartHistory      bool      `json:"has_start_history"`
}

// MRRWaterfallParams selects the period (YYYY-MM-DD; defaults to the
// trailing month) and optionally one legal entity.
type MRRWaterfallParams struct {
	Start    string
	End      string
	EntityID string
}

// MRRWaterfall returns MRR movement (new / expansion / contraction / churn /
// reactivation) plus net and gross dollar retention.
func (s *AnalyticsService) MRRWaterfall(ctx context.Context, params *MRRWaterfallParams) (*MRRWaterfall, error) {
	path := "/analytics/mrr/waterfall"
	if params != nil {
		path = newQuery().str("start", params.Start).str("end", params.End).str("entity_id", params.EntityID).apply(path)
	}
	return getData[*MRRWaterfall](ctx, s.client, http.MethodGet, path, nil)
}

// RevenueSegment is one slice of MRR (reporting currency, minor units).
type RevenueSegment struct {
	Key           string  `json:"key"`
	Label         string  `json:"label"`
	MRR           int64   `json:"mrr"`
	Subscriptions int     `json:"subscriptions"`
	SharePct      float64 `json:"share_pct"`
}

// RevenueBreakdown is MRR split into segments (by plan or by country).
type RevenueBreakdown struct {
	ReportingCurrency string           `json:"reporting_currency"`
	TotalMRR          int64            `json:"total_mrr"`
	Segments          []RevenueSegment `json:"segments"`
}

// RevenueByPlan breaks MRR down by plan.
func (s *AnalyticsService) RevenueByPlan(ctx context.Context) (*RevenueBreakdown, error) {
	return getData[*RevenueBreakdown](ctx, s.client, http.MethodGet, "/analytics/revenue-by-plan", nil)
}

// RevenueByGeography breaks MRR down by customer country.
func (s *AnalyticsService) RevenueByGeography(ctx context.Context) (*RevenueBreakdown, error) {
	return getData[*RevenueBreakdown](ctx, s.client, http.MethodGet, "/analytics/revenue-by-geography", nil)
}

// UnitEconomics is ARPA / ARPU / LTV in the reporting currency (minor
// units). MonthlyChurnRate is a percentage; HasLTV is false when churn is
// zero and LTV is undefined.
type UnitEconomics struct {
	ReportingCurrency   string  `json:"reporting_currency"`
	MRR                 int64   `json:"mrr"`
	ActiveCustomers     int     `json:"active_customers"`
	ActiveSubscriptions int     `json:"active_subscriptions"`
	ARPA                int64   `json:"arpa"`
	ARPU                int64   `json:"arpu"`
	MonthlyChurnRate    float64 `json:"monthly_churn_rate"`
	LTV                 int64   `json:"ltv"`
	HasLTV              bool    `json:"has_ltv"`
}

// UnitEconomics returns ARPA, ARPU, churn, and LTV.
func (s *AnalyticsService) UnitEconomics(ctx context.Context) (*UnitEconomics, error) {
	return getData[*UnitEconomics](ctx, s.client, http.MethodGet, "/analytics/unit-economics", nil)
}

// UsageDimensionTotal is aggregate usage for one dimension.
type UsageDimensionTotal struct {
	Dimension     string `json:"dimension"`
	TotalQuantity int64  `json:"total_quantity"`
}

// UsageStats is aggregate metered usage by dimension.
type UsageStats struct {
	Dimensions       []UsageDimensionTotal `json:"data"`
	CustomersMetered int                   `json:"customers_metered"`
}

// UsageStats returns aggregate usage by dimension.
func (s *AnalyticsService) UsageStats(ctx context.Context) (*UsageStats, error) {
	var out UsageStats
	if err := s.client.do(ctx, http.MethodGet, "/analytics/usage", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
