package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// PaymentAttempt is one gateway payment attempt against an invoice. Amount
// is in the currency's smallest unit. CustomerID/SubscriptionID are
// read-time joins off the attempt's invoice and are only populated by Get.
type PaymentAttempt struct {
	ID                     string     `json:"id"`
	InvoiceID              string     `json:"invoice_id"`
	InvoiceNumber          string     `json:"invoice_number"`
	Currency               string     `json:"currency"`
	CustomerID             string     `json:"customer_id,omitempty"`
	SubscriptionID         *string    `json:"subscription_id,omitempty"`
	Gateway                string     `json:"gateway"`
	Method                 string     `json:"method"`
	GatewayPaymentIntentID string     `json:"gateway_payment_intent_id"`
	Status                 string     `json:"status"` // initiated | processing | succeeded | failed | returned
	FailureCode            string     `json:"failure_code"`
	Amount                 int64      `json:"amount"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at,omitempty"`
	SettledAt              *time.Time `json:"settled_at"`
}

// PaymentAttemptListParams filters the payments log. Q is a case-insensitive
// substring search on invoice number or gateway payment reference; the
// status filter is ignored on the search path.
type PaymentAttemptListParams struct {
	Status  string
	Q       string
	Page    int
	PerPage int
}

// Pagination is the page/per_page/total metadata on paged responses.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PaymentAttemptList is one page of the tenant's payments log.
type PaymentAttemptList struct {
	Data       []PaymentAttempt `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

// PaymentAttemptsService groups the tenant-wide payments-log endpoints.
type PaymentAttemptsService struct{ client *Client }

// List returns the tenant's payment attempts, newest first, with pagination
// metadata.
func (s *PaymentAttemptsService) List(ctx context.Context, params *PaymentAttemptListParams) (*PaymentAttemptList, error) {
	path := "/payment-attempts"
	if params != nil {
		path = newQuery().
			str("status", params.Status).
			str("q", params.Q).
			int("page", params.Page).
			int("per_page", params.PerPage).
			apply(path)
	}
	var out PaymentAttemptList
	if err := s.client.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns one payment attempt with its invoice-level context (invoice
// number, currency, customer, and subscription).
func (s *PaymentAttemptsService) Get(ctx context.Context, id string) (*PaymentAttempt, error) {
	return getData[*PaymentAttempt](ctx, s.client, http.MethodGet, fmt.Sprintf("/payment-attempts/%s", id), nil)
}
