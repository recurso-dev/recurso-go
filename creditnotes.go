package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// CreditNote is a customer credit.
type CreditNote struct {
	ID         string `json:"id"`
	TenantID   string `json:"tenant_id"`
	CustomerID string `json:"customer_id"`
	InvoiceID  string `json:"invoice_id"`
	Reference  string `json:"reference"`
	Amount     int64  `json:"amount"`
	Balance    int64  `json:"balance"`
	Currency   string `json:"currency"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	// Tax breakdown (present when the note recorded one at creation; zero /
	// empty on legacy rows and standalone goodwill credits). Amount is gross;
	// Subtotal is the taxable value. See the credit-note document (CDN).
	Subtotal   int64     `json:"subtotal,omitempty"`
	TaxAmount  int64     `json:"tax_amount,omitempty"`
	IGSTAmount int64     `json:"igst_amount,omitempty"`
	CGSTAmount int64     `json:"cgst_amount,omitempty"`
	SGSTAmount int64     `json:"sgst_amount,omitempty"`
	TaxType    string    `json:"tax_type,omitempty"`
	HSNCode    string    `json:"hsn_code,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Customer   *Customer `json:"customer,omitempty"`
}

// CreditNoteCreateParams is the body for issuing a credit note. Amount is in the
// currency's smallest unit.
type CreditNoteCreateParams struct {
	CustomerID string `json:"customer_id"`
	InvoiceID  string `json:"invoice_id,omitempty"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	Reason     string `json:"reason,omitempty"`
	// Type is "adjustment" (default, issues account credit) or "refund"
	// (calls the gateway's refund API against the paid invoice in InvoiceID
	// and posts a Refunds-vs-Cash ledger reversal).
	Type string `json:"type,omitempty"`
}

// CreditNoteListParams filters the credit-note list.
type CreditNoteListParams struct {
	CustomerID string
	// Limit/Offset page the list; the API clamps at 1000 rows per call.
	Limit  int
	Offset int
}

// CreditNotesService groups the credit-note endpoints.
type CreditNotesService struct{ client *Client }

// Create issues a credit note.
func (s *CreditNotesService) Create(ctx context.Context, params *CreditNoteCreateParams) (*CreditNote, error) {
	out, err := getData[CreditNote](ctx, s.client, http.MethodPost, "/credit-notes", params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's credit notes.
func (s *CreditNotesService) List(ctx context.Context, params *CreditNoteListParams) ([]CreditNote, error) {
	path := "/credit-notes"
	if params != nil {
		path = newQuery().str("customer_id", params.CustomerID).int("limit", params.Limit).int("offset", params.Offset).apply(path)
	}
	return getData[[]CreditNote](ctx, s.client, http.MethodGet, path, nil)
}

// Get returns one credit note by id, scoped to the authenticated tenant. A
// foreign or missing credit note is a flat 404.
func (s *CreditNotesService) Get(ctx context.Context, id string) (*CreditNote, error) {
	return getData[*CreditNote](ctx, s.client, http.MethodGet, fmt.Sprintf("/credit-notes/%s", id), nil)
}

// JournalEntry is one posted double-entry ledger transaction, resolved with
// the debit and credit account codes and names. Amount is in minor units.
// EntityID/EntityName are populated when the entry is tagged to a legal
// entity (Multi-Entity Books).
type JournalEntry struct {
	TransactionID     string    `json:"transaction_id"`
	Timestamp         time.Time `json:"timestamp"`
	Code              int       `json:"code"`
	DebitAccountID    string    `json:"debit_account_id"`
	DebitAccountCode  int       `json:"debit_account_code"`
	DebitAccountName  string    `json:"debit_account_name"`
	CreditAccountID   string    `json:"credit_account_id"`
	CreditAccountCode int       `json:"credit_account_code"`
	CreditAccountName string    `json:"credit_account_name"`
	Amount            int64     `json:"amount"`
	ReferenceID       string    `json:"reference_id"`
	Description       string    `json:"description"`
	AccountingVersion int       `json:"accounting_version"`
	EntityID          *string   `json:"entity_id,omitempty"`
	EntityName        string    `json:"entity_name,omitempty"`
}

// CreditNoteJournalEntries is the ledger drill-down for a credit note.
type CreditNoteJournalEntries struct {
	CreditNoteID string         `json:"credit_note_id"`
	Entries      []JournalEntry `json:"entries"`
}

// Approve approves a credit note awaiting approval, issuing the credit.
func (s *CreditNotesService) Approve(ctx context.Context, id string) (*CreditNote, error) {
	return getData[*CreditNote](ctx, s.client, http.MethodPost, fmt.Sprintf("/credit-notes/%s/approve", id), nil)
}

// Reject rejects a credit note awaiting approval.
func (s *CreditNotesService) Reject(ctx context.Context, id string) (*CreditNote, error) {
	return getData[*CreditNote](ctx, s.client, http.MethodPost, fmt.Sprintf("/credit-notes/%s/reject", id), nil)
}

// Void voids an issued account-credit note, reversing its ledger posting.
func (s *CreditNotesService) Void(ctx context.Context, id string) (*CreditNote, error) {
	return getData[*CreditNote](ctx, s.client, http.MethodPost, fmt.Sprintf("/credit-notes/%s/void", id), nil)
}

// JournalEntries returns the ledger transactions posted for a credit note.
func (s *CreditNotesService) JournalEntries(ctx context.Context, id string) (*CreditNoteJournalEntries, error) {
	return getData[*CreditNoteJournalEntries](ctx, s.client, http.MethodGet, fmt.Sprintf("/credit-notes/%s/journal-entries", id), nil)
}

// DownloadPDF returns the credit note's printable document (HTML).
func (s *CreditNotesService) DownloadPDF(ctx context.Context, id string) ([]byte, error) {
	return s.client.doRaw(ctx, http.MethodGet, fmt.Sprintf("/credit-notes/%s/pdf", id), "text/html")
}
