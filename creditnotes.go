package recurso

import (
	"context"
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
