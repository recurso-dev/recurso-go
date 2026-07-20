package recurso

import (
	"context"
	"net/http"
	"time"
)

// VirtualAccount is a virtual bank account (provisioned via Razorpay) that a
// customer can wire money to. Amounts are in the currency's smallest unit.
type VirtualAccount struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	CustomerID      string     `json:"customer_id"`
	InvoiceID       *string    `json:"invoice_id"`
	AccountNumber   string     `json:"account_number"`
	IFSCCode        string     `json:"ifsc_code"`
	BankName        string     `json:"bank_name"`
	BeneficiaryName string     `json:"beneficiary_name"`
	RazorpayVAID    string     `json:"razorpay_va_id"`
	Status          string     `json:"status"`
	AmountExpected  int64      `json:"amount_expected"`
	AmountReceived  int64      `json:"amount_received"`
	ClosedAt        *time.Time `json:"closed_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// VirtualAccountCreateParams provisions a virtual account. Amount is the
// expected amount in the currency's smallest unit.
type VirtualAccountCreateParams struct {
	CustomerID string `json:"customer_id"`
	InvoiceID  string `json:"invoice_id,omitempty"`
	Amount     int64  `json:"amount"`
}

// OfflinePayment is a manually recorded bank transfer, cash, or cheque
// payment. Amounts are in the currency's smallest unit.
type OfflinePayment struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	CustomerID      string    `json:"customer_id"`
	InvoiceID       *string   `json:"invoice_id"`
	PaymentType     string    `json:"payment_type"`
	Amount          int64     `json:"amount"`
	TDSAmount       int64     `json:"tds_amount"`
	Currency        string    `json:"currency"`
	ReferenceNumber string    `json:"reference_number"`
	Notes           string    `json:"notes"`
	RecordedBy      string    `json:"recorded_by"`
	RecordedAt      time.Time `json:"recorded_at"`
}

// OfflinePaymentRecordParams records an offline payment, optionally against an
// invoice. PaymentType is "bank_transfer", "cash", or "cheque". TDSAmount is
// tax deducted at source by the customer (India B2B); it requires InvoiceID,
// counts toward settling the invoice, and cannot exceed the invoice's
// outstanding balance.
type OfflinePaymentRecordParams struct {
	CustomerID      string `json:"customer_id"`
	InvoiceID       string `json:"invoice_id,omitempty"`
	PaymentType     string `json:"payment_type"`
	Amount          int64  `json:"amount"`
	TDSAmount       int64  `json:"tds_amount,omitempty"`
	Currency        string `json:"currency,omitempty"`
	ReferenceNumber string `json:"reference_number,omitempty"`
	Notes           string `json:"notes,omitempty"`
	RecordedBy      string `json:"recorded_by,omitempty"`
}

// VirtualAccountsService groups the virtual-account endpoints.
type VirtualAccountsService struct{ client *Client }

// Create provisions a virtual bank account for a customer.
func (s *VirtualAccountsService) Create(ctx context.Context, params *VirtualAccountCreateParams) (*VirtualAccount, error) {
	var out VirtualAccount
	if err := s.client.do(ctx, http.MethodPost, "/virtual-accounts", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's virtual accounts.
func (s *VirtualAccountsService) List(ctx context.Context) ([]VirtualAccount, error) {
	return getData[[]VirtualAccount](ctx, s.client, http.MethodGet, "/virtual-accounts", nil)
}

// OfflinePaymentsService groups the offline-payment endpoints.
type OfflinePaymentsService struct{ client *Client }

// Record manually records an offline payment.
func (s *OfflinePaymentsService) Record(ctx context.Context, params *OfflinePaymentRecordParams) (*OfflinePayment, error) {
	var out OfflinePayment
	if err := s.client.do(ctx, http.MethodPost, "/payments/offline", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's offline payments.
func (s *OfflinePaymentsService) List(ctx context.Context) ([]OfflinePayment, error) {
	return getData[[]OfflinePayment](ctx, s.client, http.MethodGet, "/payments/offline", nil)
}
