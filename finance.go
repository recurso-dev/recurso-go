package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ReconciliationDiscrepancy is one mismatch found by a ledger reconciliation
// run. Amounts are in the currency's smallest unit. AccountCode is set for
// abnormal-account-balance findings.
type ReconciliationDiscrepancy struct {
	Type           string  `json:"type"`
	InvoiceID      *string `json:"invoice_id,omitempty"`
	TransactionID  *string `json:"transaction_id,omitempty"`
	ReferenceID    *string `json:"reference_id,omitempty"`
	AccountCode    int     `json:"account_code,omitempty"`
	ExpectedAmount int64   `json:"expected_amount"`
	FoundAmount    int64   `json:"found_amount"`
}

// ReconciliationReport is the result of an invoice-vs-ledger reconciliation
// pass. TBCompared reports whether the trial balance was also checked;
// TBSkipReason explains why when it was not.
type ReconciliationReport struct {
	TenantID            string                      `json:"tenant_id"`
	StartedAt           time.Time                   `json:"started_at"`
	FinishedAt          time.Time                   `json:"finished_at"`
	InvoicesChecked     int                         `json:"invoices_checked"`
	PaidInvoicesChecked int                         `json:"paid_invoices_checked"`
	TotalDiscrepancies  int                         `json:"total_discrepancies"`
	Discrepancies       []ReconciliationDiscrepancy `json:"discrepancies"`
	Truncated           bool                        `json:"truncated"`
	TBCompared          bool                        `json:"tb_compared"`
	TBSkipReason        string                      `json:"tb_skip_reason,omitempty"`
	TBAccountsChecked   int                         `json:"tb_accounts_checked,omitempty"`
	TBTransfersChecked  int                         `json:"tb_transfers_checked,omitempty"`
	ReportingCurrency   string                      `json:"reporting_currency,omitempty"`
}

// ReconciliationRun is a recorded reconciliation (audit trail). Discrepancies
// are only populated by FinanceService.ReconciliationRun.
type ReconciliationRun struct {
	ID                     string                      `json:"id"`
	RunBy                  *string                     `json:"run_by"`
	RunAt                  time.Time                   `json:"run_at"`
	InvoicesChecked        int                         `json:"invoices_checked"`
	PaidInvoicesChecked    int                         `json:"paid_invoices_checked"`
	TotalDiscrepancies     int                         `json:"total_discrepancies"`
	TBCompared             bool                        `json:"tb_compared"`
	TBAccountsChecked      int                         `json:"tb_accounts_checked"`
	TBTransfersChecked     int                         `json:"tb_transfers_checked"`
	CreatedAt              time.Time                   `json:"created_at"`
	DiscrepanciesTruncated bool                        `json:"discrepancies_truncated,omitempty"`
	Discrepancies          []ReconciliationDiscrepancy `json:"discrepancies,omitempty"`
}

// ClosePackPeriod is the calendar month a close pack covers.
type ClosePackPeriod struct {
	Month int       `json:"month"`
	Year  int       `json:"year"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ClosePackDeferred ties the ledger's deferred-revenue movement to the
// recognition schedule. UnexplainedDelta is ledger − (schedule + awaiting
// payment); Ties is true when it is zero.
type ClosePackDeferred struct {
	Rollforward      *DeferredRollforward `json:"rollforward"`
	Recognition      *RevRecReport        `json:"recognition,omitempty"`
	AwaitingPayment  int64                `json:"awaiting_payment"`
	UnexplainedDelta int64                `json:"unexplained_delta"`
	Ties             bool                 `json:"ties"`
}

// ClosePackGL points at the general-ledger export for the period.
type ClosePackGL struct {
	Format    string `json:"format"`
	ExportURL string `json:"export_url"`
}

// ClosePack is the month-end close bundle: trial balance, reconciliation,
// deferred-revenue tie-out, and the GL export pointer, plus a ready-to-close
// verdict and its blockers.
type ClosePack struct {
	TenantID          string                `json:"tenant_id"`
	Period            ClosePackPeriod       `json:"period"`
	GeneratedAt       time.Time             `json:"generated_at"`
	ReadyToClose      bool                  `json:"ready_to_close"`
	Blockers          []string              `json:"blockers"`
	TrialBalance      *TrialBalance         `json:"trial_balance"`
	Reconciliation    *ReconciliationReport `json:"reconciliation"`
	Deferred          ClosePackDeferred     `json:"deferred_revenue"`
	GeneralLedger     ClosePackGL           `json:"general_ledger"`
	ReportingCurrency string                `json:"reporting_currency"`
}

// RevRecBucket is the deferred balance due to recognize in one month.
type RevRecBucket struct {
	Month  int   `json:"month"`
	Year   int   `json:"year"`
	Amount int64 `json:"amount"`
}

// RevRecCurrencyBalance is the deferred balance in one native currency.
type RevRecCurrencyBalance struct {
	Currency string `json:"currency"`
	Deferred int64  `json:"deferred"`
}

// RevRecReport is the revenue-recognition report for one month: what was
// recognized, what is still deferred, and when the remainder recognizes.
type RevRecReport struct {
	Month            int                     `json:"month"`
	Year             int                     `json:"year"`
	RecognizedAmount int64                   `json:"recognized_amount"`
	DeferredBalance  int64                   `json:"deferred_balance"`
	Upcoming         []RevRecBucket          `json:"upcoming"`
	ByCurrency       []RevRecCurrencyBalance `json:"by_currency"`
}

// RevenueWaterfallBucket is one month of recognized vs still-scheduled
// revenue (minor units).
type RevenueWaterfallBucket struct {
	Year       int   `json:"year"`
	Month      int   `json:"month"`
	Recognized int64 `json:"recognized"`
	Scheduled  int64 `json:"scheduled"`
}

// RevenueWaterfall is the month-by-month recognition schedule.
type RevenueWaterfall struct {
	TenantID          string                   `json:"tenant_id"`
	Buckets           []RevenueWaterfallBucket `json:"buckets"`
	TotalRecognized   int64                    `json:"total_recognized"`
	TotalScheduled    int64                    `json:"total_scheduled"`
	ReportingCurrency string                   `json:"reporting_currency"`
}

// FinanceService groups the reconciliation, close-pack, and revenue
// recognition endpoints.
type FinanceService struct{ client *Client }

// Reconcile runs an on-demand ledger reconciliation without recording it.
func (s *FinanceService) Reconcile(ctx context.Context) (*ReconciliationReport, error) {
	return getData[*ReconciliationReport](ctx, s.client, http.MethodGet, "/finance/reconciliation", nil)
}

// RecordReconciliation runs a reconciliation and records it in the audit
// trail (see ReconciliationRuns).
func (s *FinanceService) RecordReconciliation(ctx context.Context) (*ReconciliationReport, error) {
	return getData[*ReconciliationReport](ctx, s.client, http.MethodPost, "/finance/reconciliation/runs", nil)
}

// ReconciliationRuns lists recorded reconciliation runs, newest first. limit
// 0 uses the API default.
func (s *FinanceService) ReconciliationRuns(ctx context.Context, limit int) ([]ReconciliationRun, error) {
	path := newQuery().int("limit", limit).apply("/finance/reconciliation/runs")
	return getData[[]ReconciliationRun](ctx, s.client, http.MethodGet, path, nil)
}

// ReconciliationRun returns one recorded run with its discrepancies.
func (s *FinanceService) ReconciliationRun(ctx context.Context, id string) (*ReconciliationRun, error) {
	return getData[*ReconciliationRun](ctx, s.client, http.MethodGet, fmt.Sprintf("/finance/reconciliation/runs/%s", id), nil)
}

// ClosePack returns the month-end close pack for the given calendar month.
func (s *FinanceService) ClosePack(ctx context.Context, month, year int) (*ClosePack, error) {
	path := newQuery().int("month", month).int("year", year).apply("/finance/close-pack")
	return getData[*ClosePack](ctx, s.client, http.MethodGet, path, nil)
}

// RevRecReport returns the revenue-recognition report for the given month.
func (s *FinanceService) RevRecReport(ctx context.Context, month, year int) (*RevRecReport, error) {
	path := newQuery().int("month", month).int("year", year).apply("/finance/revrec/report")
	return getData[*RevRecReport](ctx, s.client, http.MethodGet, path, nil)
}

// RevRecWaterfall returns the month-by-month recognition waterfall.
func (s *FinanceService) RevRecWaterfall(ctx context.Context) (*RevenueWaterfall, error) {
	return getData[*RevenueWaterfall](ctx, s.client, http.MethodGet, "/finance/revrec/waterfall", nil)
}
