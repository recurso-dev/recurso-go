package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// LedgerAccount is a double-entry ledger account.
type LedgerAccount struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Code          int       `json:"code"`
	LedgerID      int       `json:"ledger_id"`
	CreditsPosted int64     `json:"credits_posted"`
	DebitsPosted  int64     `json:"debits_posted"`
	Currency      string    `json:"currency"`
	Balance       int64     `json:"balance"`
	CreatedAt     time.Time `json:"created_at"`
}

// LedgerTransaction is a ledger transfer touching an account.
type LedgerTransaction struct {
	ID              string    `json:"id"`
	DebitAccountID  string    `json:"debit_account_id"`
	CreditAccountID string    `json:"credit_account_id"`
	Amount          int64     `json:"amount"`
	LedgerID        int       `json:"ledger_id"`
	Code            int       `json:"code"`
	ReferenceID     string    `json:"reference_id"`
	Description     string    `json:"description"`
	Timestamp       time.Time `json:"timestamp"`
}

// LedgerEntriesParams selects the account whose entries to list.
type LedgerEntriesParams struct {
	AccountID string
}

// LedgerService groups the finance ledger endpoints.
type LedgerService struct{ client *Client }

// Accounts lists the tenant's ledger accounts.
func (s *LedgerService) Accounts(ctx context.Context) ([]LedgerAccount, error) {
	return getData[[]LedgerAccount](ctx, s.client, http.MethodGet, "/ledger/accounts", nil)
}

// Entries lists ledger transfers touching the given account.
func (s *LedgerService) Entries(ctx context.Context, params *LedgerEntriesParams) ([]LedgerTransaction, error) {
	path := "/ledger/entries"
	if params != nil {
		path = newQuery().str("account_id", params.AccountID).apply(path)
	}
	return getData[[]LedgerTransaction](ctx, s.client, http.MethodGet, path, nil)
}

// Transaction returns one posted journal entry by id, resolved with account
// codes and names.
func (s *LedgerService) Transaction(ctx context.Context, id string) (*JournalEntry, error) {
	return getData[*JournalEntry](ctx, s.client, http.MethodGet, fmt.Sprintf("/ledger/transactions/%s", id), nil)
}

// TrialBalanceLine is one account on the trial balance (minor units).
// Balance is signed on the account's normal side; Abnormal flags a
// wrong-sign balance. Type is the account type code.
type TrialBalanceLine struct {
	AccountID  string  `json:"account_id"`
	Code       int     `json:"code"`
	Name       string  `json:"name"`
	Type       int     `json:"type"`
	Debits     int64   `json:"debits"`
	Credits    int64   `json:"credits"`
	Balance    int64   `json:"balance"`
	Abnormal   bool    `json:"abnormal"`
	EntityID   *string `json:"entity_id,omitempty"`
	EntityName string  `json:"entity_name,omitempty"`
}

// TrialBalance is every account with its posted totals and the double-entry
// invariant (Balanced: total debits == total credits).
type TrialBalance struct {
	TenantID          string             `json:"tenant_id"`
	Lines             []TrialBalanceLine `json:"lines"`
	TotalDebits       int64              `json:"total_debits"`
	TotalCredits      int64              `json:"total_credits"`
	Balanced          bool               `json:"balanced"`
	AsOf              time.Time          `json:"as_of"`
	ReportingCurrency string             `json:"reporting_currency"`
}

// TrialBalanceParams scopes the trial balance: EntityID to one legal entity,
// or Consolidated to roll every entity up by account code.
type TrialBalanceParams struct {
	EntityID     string
	Consolidated bool
}

// TrialBalance returns the trial balance.
func (s *LedgerService) TrialBalance(ctx context.Context, params *TrialBalanceParams) (*TrialBalance, error) {
	path := "/ledger/trial-balance"
	if params != nil {
		path = newQuery().str("entity_id", params.EntityID).boolean("consolidated", params.Consolidated).apply(path)
	}
	return getData[*TrialBalance](ctx, s.client, http.MethodGet, path, nil)
}

// LedgerExportParams selects the month to export and optionally one legal
// entity. Month/Year 0 export the current month.
type LedgerExportParams struct {
	Month    int
	Year     int
	EntityID string
}

// Export returns the general ledger for a month as CSV.
func (s *LedgerService) Export(ctx context.Context, params *LedgerExportParams) ([]byte, error) {
	path := "/ledger/export"
	if params != nil {
		path = newQuery().int("month", params.Month).int("year", params.Year).str("entity_id", params.EntityID).apply(path)
	}
	return s.client.doRaw(ctx, http.MethodGet, path, "text/csv")
}

// DeferredRollforward is the deferred-revenue account's movement over a
// period (minor units): Closing = Opening + Added - Released.
type DeferredRollforward struct {
	TenantID          string    `json:"tenant_id"`
	PeriodStart       time.Time `json:"period_start"`
	PeriodEnd         time.Time `json:"period_end"`
	Opening           int64     `json:"opening"`
	Added             int64     `json:"added"`
	Released          int64     `json:"released"`
	Closing           int64     `json:"closing"`
	ReportingCurrency string    `json:"reporting_currency"`
}

// DeferredRollforward returns the deferred-revenue rollforward for a month.
func (s *LedgerService) DeferredRollforward(ctx context.Context, month, year int) (*DeferredRollforward, error) {
	path := newQuery().int("month", month).int("year", year).apply("/ledger/deferred-rollforward")
	return getData[*DeferredRollforward](ctx, s.client, http.MethodGet, path, nil)
}
