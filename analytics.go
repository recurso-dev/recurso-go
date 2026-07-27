package recurso

import (
	"context"
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
