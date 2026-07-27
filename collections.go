package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// CollectionsQueueItem is one currently-failing invoice on the collections
// worklist: who is failing, why, what the recovery engine will do next, and
// the latest ACH payment-attempt state.
type CollectionsQueueItem struct {
	ID               string     `json:"id"`
	CustomerID       string     `json:"customer_id"`
	CustomerName     string     `json:"customer_name"`
	CustomerEmail    string     `json:"customer_email"`
	InvoiceNumber    string     `json:"invoice_number"`
	Status           string     `json:"status"` // "past_due" | "uncollectible"
	Currency         string     `json:"currency"`
	AmountRemaining  int64      `json:"amount_remaining"` // minor units
	DueDate          time.Time  `json:"due_date"`
	DaysOverdue      int        `json:"days_overdue"`
	RetryCount       int        `json:"retry_count"`
	LastPaymentError string     `json:"last_payment_error"`
	NextRetryAt      *time.Time `json:"next_retry_at,omitempty"`
	ManagedBy        string     `json:"managed_by"` // "scheduler" | "worker" | "campaign"
	AttemptStatus    string     `json:"attempt_status,omitempty"`
	DunningPaused    bool       `json:"dunning_paused"`
}

// CollectionsQueueParams filters the worklist. Zero values mean "no filter".
type CollectionsQueueParams struct {
	Status    string // "past_due" | "uncollectible"
	ManagedBy string // "scheduler" | "worker" | "campaign"
	Page      int
	PerPage   int
}

// CollectionsBucket is one stage of the recovery funnel, FX-normalized to the
// reporting currency (minor units).
type CollectionsBucket struct {
	Count  int   `json:"count"`
	Amount int64 `json:"amount"`
}

// CollectionsFunnel is the failed → resolved journey of billed revenue.
// RecoveryRate is a windowed cohort: of the cases concluded (recovered or
// written off) in the trailing RateWindowDays, the fraction recovered.
type CollectionsFunnel struct {
	ReportingCurrency    string            `json:"reporting_currency"`
	PastDue              CollectionsBucket `json:"past_due"`
	Uncollectible        CollectionsBucket `json:"uncollectible"`
	Recovered            CollectionsBucket `json:"recovered"`
	RecoveryRate         float64           `json:"recovery_rate"`
	RateWindowDays       int               `json:"rate_window_days"`
	FXExcludedCurrencies []string          `json:"fx_excluded_currencies,omitempty"`
}

// CollectionsFailureBucket is one failure reason ranked by money at risk.
type CollectionsFailureBucket struct {
	ErrorCode    string `json:"error_code"`
	Count        int    `json:"count"`
	AmountAtRisk int64  `json:"amount_at_risk"`
}

// CollectionsActionResult reports the outcome of a manual collections action.
type CollectionsActionResult struct {
	Status        string `json:"status,omitempty"`
	DunningPaused bool   `json:"dunning_paused"`
}

// CollectionsService is the operator layer over payment recovery: the worklist
// of failing invoices, the recovery funnel, and manual controls. Guarded
// actions return 409 (APIError) when they would double-charge — e.g. retrying
// a paused, mandate, or still-settling (ACH in-flight) invoice.
type CollectionsService struct{ client *Client }

// Queue lists invoices currently in recovery, oldest due-date first.
func (s *CollectionsService) Queue(ctx context.Context, params *CollectionsQueueParams) ([]CollectionsQueueItem, error) {
	path := "/collections/queue"
	if params != nil {
		path = newQuery().
			str("status", params.Status).
			str("managed_by", params.ManagedBy).
			int("page", params.Page).
			int("per_page", params.PerPage).
			apply(path)
	}
	return getData[[]CollectionsQueueItem](ctx, s.client, http.MethodGet, path, nil)
}

// Funnel returns the recovery funnel with revenue at risk and the windowed
// recovery rate, FX-normalized to the reporting currency.
func (s *CollectionsService) Funnel(ctx context.Context) (*CollectionsFunnel, error) {
	return getData[*CollectionsFunnel](ctx, s.client, http.MethodGet, "/analytics/collections/funnel", nil)
}

// Failures ranks failure reasons by the amount of billed revenue each is
// currently holding, descending.
func (s *CollectionsService) Failures(ctx context.Context) ([]CollectionsFailureBucket, error) {
	return getData[[]CollectionsFailureBucket](ctx, s.client, http.MethodGet, "/analytics/collections/failures", nil)
}

// RetryNow queues an immediate smart-retry charge for a past-due invoice.
// Refused with a 409 APIError when the invoice is paused, a UPI-mandate
// invoice, still settling an ACH debit, or no longer past due.
func (s *CollectionsService) RetryNow(ctx context.Context, invoiceID string) (*CollectionsActionResult, error) {
	return getData[*CollectionsActionResult](ctx, s.client, http.MethodPost,
		fmt.Sprintf("/collections/invoices/%s/retry-now", invoiceID), nil)
}

// PauseDunning pauses (true) or resumes (false) ALL automated dunning for one
// invoice — retry charges, escalation emails, and campaign steps alike.
func (s *CollectionsService) PauseDunning(ctx context.Context, invoiceID string, paused bool) (*CollectionsActionResult, error) {
	body := map[string]bool{"paused": paused}
	return getData[*CollectionsActionResult](ctx, s.client, http.MethodPost,
		fmt.Sprintf("/collections/invoices/%s/pause", invoiceID), body)
}

// MarkUncollectible writes an invoice off (status change only): it stops being
// chased and its balance stops counting as collectible.
func (s *CollectionsService) MarkUncollectible(ctx context.Context, invoiceID string) (*CollectionsActionResult, error) {
	return getData[*CollectionsActionResult](ctx, s.client, http.MethodPost,
		fmt.Sprintf("/collections/invoices/%s/mark-uncollectible", invoiceID), nil)
}
