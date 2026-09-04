package recurso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// InvoiceItem is one itemized line on an invoice.
type InvoiceItem struct {
	ID            string    `json:"id"`
	InvoiceID     string    `json:"invoice_id"`
	Description   string    `json:"description"`
	HSNCode       string    `json:"hsn_code"`
	Quantity      int       `json:"quantity"`
	UnitAmount    int64     `json:"unit_amount"`
	Amount        int64     `json:"amount"`
	TaxRate       float64   `json:"tax_rate"`
	CGSTAmount    int64     `json:"cgst_amount"`
	SGSTAmount    int64     `json:"sgst_amount"`
	IGSTAmount    int64     `json:"igst_amount"`
	TaxableAmount int64     `json:"taxable_amount"`
	CreatedAt     time.Time `json:"created_at"`
}

// Invoice is a billing invoice.
type Invoice struct {
	ID                string        `json:"id"`
	TenantID          string        `json:"tenant_id"`
	SubscriptionID    string        `json:"subscription_id"`
	CustomerID        string        `json:"customer_id"`
	InvoiceNumber     string        `json:"invoice_number"`
	BillingReason     string        `json:"billing_reason"`
	AmountDue         int64         `json:"amount_due"`
	AmountPaid        int64         `json:"amount_paid"`
	Currency          string        `json:"currency"`
	Subtotal          int64         `json:"subtotal"`
	TaxAmount         int64         `json:"tax_amount"`
	Total             int64         `json:"total"`
	IGSTAmount        int64         `json:"igst_amount"`
	CGSTAmount        int64         `json:"cgst_amount"`
	SGSTAmount        int64         `json:"sgst_amount"`
	HSNCode           string        `json:"hsn_code"`
	IRN               string        `json:"irn"`
	AckNo             string        `json:"ack_no"`
	SignedQRCode      string        `json:"signed_qr_code"`
	EInvoiceStatus    string        `json:"e_invoice_status"`
	TDSAmount         int64         `json:"tds_amount"`
	Status            string        `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	DueDate           time.Time     `json:"due_date"`
	PaidAt            time.Time     `json:"paid_at"`
	PaymentTerms      string        `json:"payment_terms"`
	ExchangeRate      float64       `json:"exchange_rate"`
	BaseCurrencyTotal int64         `json:"base_currency_total"`
	BaseCurrency      string        `json:"base_currency"`
	RetryCount        int           `json:"retry_count"`
	PaymentWallActive bool          `json:"payment_wall_active"`
	LineItems         []InvoiceItem `json:"line_items"`
}

// EInvoiceStatus is the IRN (Indian e-invoicing) status for an invoice.
type EInvoiceStatus struct {
	InvoiceID      string     `json:"invoice_id"`
	InvoiceNumber  string     `json:"invoice_number"`
	EInvoiceStatus string     `json:"e_invoice_status"`
	IRN            string     `json:"irn"`
	AckNo          string     `json:"ack_no"`
	AckDate        string     `json:"ack_date"`
	SignedQRCode   string     `json:"signed_qr_code"`
	RetryCount     int        `json:"retry_count"`
	NextRetryAt    *time.Time `json:"next_retry_at"`
	ErrorMessage   string     `json:"error_message"`
}

// EInvoiceRetryResult is returned by RetryEInvoice.
type EInvoiceRetryResult struct {
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

// MessageResponse is a generic {"message": ...} acknowledgement.
type MessageResponse struct {
	Message string `json:"message"`
}

// EInvoiceCancelParams cancels an e-invoice (IRN).
type EInvoiceCancelParams struct {
	CancelCode int    `json:"cancel_code"`
	Reason     string `json:"reason"`
}

// InvoiceListParams filters the invoice list.
type InvoiceListParams struct {
	// CustomerID filters to one customer's invoices (tenant-scoped).
	CustomerID string
	// SubscriptionID filters to one subscription's invoices. Ignored by the
	// API when CustomerID is also set.
	SubscriptionID string
	Limit          int
	Page           int
}

// InvoicesService groups the invoice endpoints.
type InvoicesService struct{ client *Client }

// List returns the tenant's invoices.
func (s *InvoicesService) List(ctx context.Context, params *InvoiceListParams) ([]Invoice, error) {
	path := "/invoices"
	if params != nil {
		path = newQuery().
			str("customer_id", params.CustomerID).
			str("subscription_id", params.SubscriptionID).
			int("limit", params.Limit).
			int("page", params.Page).
			apply(path)
	}
	return getData[[]Invoice](ctx, s.client, http.MethodGet, path, nil)
}

// Get returns one invoice by id, scoped to the authenticated tenant. A
// foreign or missing invoice is a flat 404.
func (s *InvoicesService) Get(ctx context.Context, id string) (*Invoice, error) {
	return getData[*Invoice](ctx, s.client, http.MethodGet, fmt.Sprintf("/invoices/%s", id), nil)
}

// PDFURL returns the public URL for an invoice's printable document. It does
// not perform a request.
func (s *InvoicesService) PDFURL(id string) string {
	return fmt.Sprintf("%s/invoices/%s/pdf", s.client.baseURL, id)
}

// EInvoiceStatus returns the e-invoice (IRN) status for an invoice.
func (s *InvoicesService) EInvoiceStatus(ctx context.Context, id string) (*EInvoiceStatus, error) {
	out, err := getData[EInvoiceStatus](ctx, s.client, http.MethodGet, fmt.Sprintf("/invoices/%s/einvoice", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RetryEInvoice retries IRN generation for a failed e-invoice.
func (s *InvoicesService) RetryEInvoice(ctx context.Context, id string) (*EInvoiceRetryResult, error) {
	var out EInvoiceRetryResult
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/invoices/%s/einvoice/retry", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelEInvoice cancels an e-invoice (IRN).
func (s *InvoicesService) CancelEInvoice(ctx context.Context, id string, params *EInvoiceCancelParams) (*MessageResponse, error) {
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/invoices/%s/einvoice/cancel", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// InvoiceJournalEntries is the ledger drill-down for an invoice.
type InvoiceJournalEntries struct {
	InvoiceID string         `json:"invoice_id"`
	Entries   []JournalEntry `json:"entries"`
}

// InvoicePaymentAttempts is an invoice's settlement history.
type InvoicePaymentAttempts struct {
	InvoiceID string           `json:"invoice_id"`
	Attempts  []PaymentAttempt `json:"attempts"`
}

// InvoiceStatusChange is one transition in an invoice's status timeline.
// FromStatus is nil for the initial status.
type InvoiceStatusChange struct {
	ID         string    `json:"id"`
	InvoiceID  string    `json:"invoice_id"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	ChangedAt  time.Time `json:"changed_at"`
}

// InvoiceStatusHistory is an invoice's status timeline, oldest first.
type InvoiceStatusHistory struct {
	InvoiceID string                `json:"invoice_id"`
	History   []InvoiceStatusChange `json:"history"`
}

// EUEInvoice is the EU e-invoice (EN 16931 / UBL) generated for an invoice.
// Document is the serialized XML.
type EUEInvoice struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	InvoiceID      string     `json:"invoice_id"`
	Syntax         string     `json:"syntax"`
	Status         string     `json:"status"`
	Document       string     `json:"document"`
	RecipientVATID string     `json:"recipient_vat_id"`
	MessageID      string     `json:"message_id"`
	ErrorMessage   string     `json:"error_message"`
	RetryCount     int        `json:"retry_count"`
	NextRetryAt    *time.Time `json:"next_retry_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// EUEInvoiceRetryResult is returned by RetryEUEInvoice. EUEInvoice is nil
// when nothing was generated (Message explains why).
type EUEInvoiceRetryResult struct {
	EUEInvoice *EUEInvoice `json:"data"`
	Message    string      `json:"message"`
}

// PaymentWallStatus reports whether the customer is blocked behind the
// invoice until it is paid.
type PaymentWallStatus struct {
	InvoiceID         string `json:"invoice_id"`
	PaymentWallActive bool   `json:"payment_wall_active"`
}

// JournalEntries returns the ledger transactions posted for an invoice.
func (s *InvoicesService) JournalEntries(ctx context.Context, id string) (*InvoiceJournalEntries, error) {
	return getData[*InvoiceJournalEntries](ctx, s.client, http.MethodGet, fmt.Sprintf("/invoices/%s/journal-entries", id), nil)
}

// PaymentAttempts returns every gateway payment attempt against an invoice.
func (s *InvoicesService) PaymentAttempts(ctx context.Context, id string) (*InvoicePaymentAttempts, error) {
	return getData[*InvoicePaymentAttempts](ctx, s.client, http.MethodGet, fmt.Sprintf("/invoices/%s/payment-attempts", id), nil)
}

// StatusHistory returns an invoice's status timeline.
func (s *InvoicesService) StatusHistory(ctx context.Context, id string) (*InvoiceStatusHistory, error) {
	return getData[*InvoiceStatusHistory](ctx, s.client, http.MethodGet, fmt.Sprintf("/invoices/%s/status-history", id), nil)
}

// DownloadPDF returns the invoice's printable document (HTML). See PDFURL
// for the public link.
func (s *InvoicesService) DownloadPDF(ctx context.Context, id string) ([]byte, error) {
	return s.client.doRaw(ctx, http.MethodGet, fmt.Sprintf("/invoices/%s/pdf", id), "text/html")
}

// PreviewHTML renders the invoice as HTML.
func (s *InvoicesService) PreviewHTML(ctx context.Context, id string) ([]byte, error) {
	return s.client.doRaw(ctx, http.MethodGet, fmt.Sprintf("/invoices/%s/preview", id), "text/html")
}

// Send emails the invoice to the customer.
func (s *InvoicesService) Send(ctx context.Context, id string) (*MessageResponse, error) {
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/invoices/%s/send", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EUEInvoice returns the EU e-invoice for an invoice, or nil when none has
// been generated.
func (s *InvoicesService) EUEInvoice(ctx context.Context, id string) (*EUEInvoice, error) {
	return getData[*EUEInvoice](ctx, s.client, http.MethodGet, fmt.Sprintf("/invoices/%s/eu-einvoice", id), nil)
}

// RetryEUEInvoice regenerates and re-transmits the EU e-invoice.
func (s *InvoicesService) RetryEUEInvoice(ctx context.Context, id string) (*EUEInvoiceRetryResult, error) {
	var out EUEInvoiceRetryResult
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/invoices/%s/eu-einvoice/retry", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PaymentWall returns whether the payment wall is active for an invoice.
func (s *InvoicesService) PaymentWall(ctx context.Context, id string) (*PaymentWallStatus, error) {
	var out PaymentWallStatus
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/invoices/%s/payment-wall", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
