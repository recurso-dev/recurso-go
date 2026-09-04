package recurso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// StripeExport is a Stripe data export to preview, compare, or import. Each
// slice holds the raw Stripe API objects (as decoded JSON) of that type.
type StripeExport struct {
	Customers      []map[string]any `json:"customers"`
	Products       []map[string]any `json:"products"`
	Prices         []map[string]any `json:"prices"`
	Subscriptions  []map[string]any `json:"subscriptions"`
	PaymentMethods []map[string]any `json:"payment_methods,omitempty"`
}

// RevenueCatExport is a RevenueCat data export to preview, compare, or
// import.
type RevenueCatExport struct {
	Subscribers []map[string]any `json:"subscribers"`
	Products    []map[string]any `json:"products"`
}

// ChargebeeExport is a Chargebee data export to preview, compare, or import.
type ChargebeeExport struct {
	Customers     []map[string]any `json:"customers"`
	Plans         []map[string]any `json:"plans"`
	Subscriptions []map[string]any `json:"subscriptions"`
}

// ImportPreviewItem is one object in a dry-run import preview and the action
// the import would take on it (create, link_existing, skip_already_imported,
// conflict, unsupported). The source system's id arrives under a
// provider-specific key (StripeID or ChargebeeID).
type ImportPreviewItem struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Action      string `json:"action"`
	Detail      string `json:"detail"`
	StripeID    string `json:"stripe_id,omitempty"`
	ChargebeeID string `json:"chargebee_id,omitempty"`
}

// ImportPreview is a dry-run of an import: per-object actions, a count per
// action, and warnings.
type ImportPreview struct {
	Items    []ImportPreviewItem `json:"items"`
	Summary  map[string]int      `json:"summary"`
	Warnings []string            `json:"warnings"`
}

// ImportFailure is one object an import could not create.
type ImportFailure struct {
	Kind        string `json:"kind"`
	StripeID    string `json:"stripe_id,omitempty"`
	ChargebeeID string `json:"chargebee_id,omitempty"`
	Error       string `json:"error"`
}

// ImportResult is the outcome of a committed import: the plan that was
// executed, created counts per kind, and failures.
type ImportResult struct {
	Plan     json.RawMessage `json:"plan"`
	Created  map[string]int  `json:"created"`
	Failures []ImportFailure `json:"failures"`
}

// CompareCounts is how many source objects of one kind matched a Recurso
// record.
type CompareCounts struct {
	Source  int `json:"source"`
	Matched int `json:"matched"`
	Missing int `json:"missing"`
}

// CompareIssue is one field-level mismatch between the source system and
// Recurso.
type CompareIssue struct {
	Kind       string `json:"kind"`
	ExternalID string `json:"external_id"`
	Field      string `json:"field"`
	Source     string `json:"source"`
	Recurso    string `json:"recurso"`
}

// CompareReport is the migration compare gate: whether every source object
// is present and equal in Recurso. Ready is true when there are no issues.
type CompareReport struct {
	Source        string         `json:"source"`
	Customers     CompareCounts  `json:"customers"`
	Plans         CompareCounts  `json:"plans"`
	Subscriptions CompareCounts  `json:"subscriptions"`
	Issues        []CompareIssue `json:"issues"`
	Ready         bool           `json:"ready"`
	GeneratedAt   time.Time      `json:"generated_at"`
}

// StoredCompareReport is a persisted compare run. Report holds the full
// CompareReport JSON and is only populated by ImportsService.CompareReport.
type StoredCompareReport struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id,omitempty"`
	Source      string          `json:"source"`
	Ready       bool            `json:"ready"`
	Report      json.RawMessage `json:"report,omitempty"`
	GeneratedAt time.Time       `json:"generated_at"`
}

// ImportsService groups the migration endpoints: preview / commit / compare
// for Stripe, RevenueCat, and Chargebee exports, and the stored compare
// reports.
type ImportsService struct{ client *Client }

func (s *ImportsService) preview(ctx context.Context, path string, body any) (*ImportPreview, error) {
	var out ImportPreview
	if err := s.client.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *ImportsService) commit(ctx context.Context, path string, body any) (*ImportResult, error) {
	var out ImportResult
	if err := s.client.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *ImportsService) compare(ctx context.Context, path string, body any) (*CompareReport, error) {
	var out CompareReport
	if err := s.client.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PreviewStripe dry-runs a Stripe import. Nothing is written.
func (s *ImportsService) PreviewStripe(ctx context.Context, export *StripeExport) (*ImportPreview, error) {
	return s.preview(ctx, "/import/stripe/preview", export)
}

// CommitStripe imports a Stripe export (customers, plans, subscriptions).
func (s *ImportsService) CommitStripe(ctx context.Context, export *StripeExport) (*ImportResult, error) {
	return s.commit(ctx, "/import/stripe/commit", export)
}

// CompareStripe proves a Stripe migration before cut-over.
func (s *ImportsService) CompareStripe(ctx context.Context, export *StripeExport) (*CompareReport, error) {
	return s.compare(ctx, "/import/stripe/compare", export)
}

// PreviewRevenueCat dry-runs a RevenueCat import. Nothing is written.
func (s *ImportsService) PreviewRevenueCat(ctx context.Context, export *RevenueCatExport) (*ImportPreview, error) {
	return s.preview(ctx, "/import/revenuecat/preview", export)
}

// CommitRevenueCat imports a RevenueCat export (customers, plans, active
// subscriptions).
func (s *ImportsService) CommitRevenueCat(ctx context.Context, export *RevenueCatExport) (*ImportResult, error) {
	return s.commit(ctx, "/import/revenuecat/commit", export)
}

// CompareRevenueCat proves a RevenueCat migration before cut-over.
func (s *ImportsService) CompareRevenueCat(ctx context.Context, export *RevenueCatExport) (*CompareReport, error) {
	return s.compare(ctx, "/import/revenuecat/compare", export)
}

// PreviewChargebee dry-runs a Chargebee import. Nothing is written.
func (s *ImportsService) PreviewChargebee(ctx context.Context, export *ChargebeeExport) (*ImportPreview, error) {
	return s.preview(ctx, "/import/chargebee/preview", export)
}

// CommitChargebee imports a Chargebee export (customers, plans,
// subscriptions).
func (s *ImportsService) CommitChargebee(ctx context.Context, export *ChargebeeExport) (*ImportResult, error) {
	return s.commit(ctx, "/import/chargebee/commit", export)
}

// CompareChargebee proves a Chargebee migration before cut-over.
func (s *ImportsService) CompareChargebee(ctx context.Context, export *ChargebeeExport) (*CompareReport, error) {
	return s.compare(ctx, "/import/chargebee/compare", export)
}

// CompareReports lists stored compare runs, newest first. limit 0 uses the
// API default.
func (s *ImportsService) CompareReports(ctx context.Context, limit int) ([]StoredCompareReport, error) {
	path := newQuery().int("limit", limit).apply("/import/compare-reports")
	return getData[[]StoredCompareReport](ctx, s.client, http.MethodGet, path, nil)
}

// CompareReport returns one stored compare run with its full report.
func (s *ImportsService) CompareReport(ctx context.Context, id string) (*StoredCompareReport, error) {
	return getData[*StoredCompareReport](ctx, s.client, http.MethodGet, fmt.Sprintf("/import/compare-reports/%s", id), nil)
}

// CompareReportDocument returns the printable (HTML) receipt for a stored
// compare run.
func (s *ImportsService) CompareReportDocument(ctx context.Context, id string) ([]byte, error) {
	return s.client.doRaw(ctx, http.MethodGet, fmt.Sprintf("/import/compare-reports/%s/document", id), "text/html")
}
