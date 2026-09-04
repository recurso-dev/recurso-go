package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Gift is a gifted subscription.
type Gift struct {
	ID                   string     `json:"id"`
	TenantID             string     `json:"tenant_id"`
	Code                 string     `json:"code"`
	PlanID               string     `json:"plan_id"`
	BuyerCustomerID      string     `json:"buyer_customer_id"`
	RecipientEmail       string     `json:"recipient_email"`
	Status               string     `json:"status"`
	RedeemedByCustomerID string     `json:"redeemed_by_customer_id"`
	RedeemedAt           *time.Time `json:"redeemed_at"`
	DurationMonths       int        `json:"duration_months"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// GiftPurchaseParams buys a plan as a gift.
type GiftPurchaseParams struct {
	BuyerCustomerID string `json:"buyer_customer_id"`
	PlanID          string `json:"plan_id"`
	RecipientEmail  string `json:"recipient_email,omitempty"`
	DurationMonths  int    `json:"duration_months"`
}

// GiftRedeemParams redeems a gift code for a customer.
type GiftRedeemParams struct {
	Code                string `json:"code"`
	RecipientCustomerID string `json:"recipient_customer_id"`
}

// GiftListParams paginates the gift list.
type GiftListParams struct {
	Page    int
	PerPage int
}

// GiftsService groups the gift-subscription endpoints.
type GiftsService struct{ client *Client }

// Purchase buys a plan as a gift and returns the redemption code.
func (s *GiftsService) Purchase(ctx context.Context, params *GiftPurchaseParams) (*Gift, error) {
	var out Gift
	if err := s.client.do(ctx, http.MethodPost, "/gifts/purchase", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Redeem redeems a gift code and starts the gifted subscription.
func (s *GiftsService) Redeem(ctx context.Context, params *GiftRedeemParams) (*Subscription, error) {
	var out Subscription
	if err := s.client.do(ctx, http.MethodPost, "/gifts/redeem", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GiftCancelResult reports what canceling a gift did on the money side:
// a PAID purchase issues the buyer an account credit (CreditNote set), an
// unpaid purchase invoice is voided (InvoiceVoided true).
type GiftCancelResult struct {
	Gift          Gift        `json:"gift"`
	CreditNote    *CreditNote `json:"credit_note,omitempty"`
	InvoiceVoided bool        `json:"invoice_voided"`
}

// Cancel cancels a purchased, unredeemed gift. The buyer is made whole
// automatically; redeemed or already-canceled gifts return a conflict error.
func (s *GiftsService) Cancel(ctx context.Context, id string) (*GiftCancelResult, error) {
	out, err := getData[GiftCancelResult](ctx, s.client, http.MethodPost, fmt.Sprintf("/gifts/%s/cancel", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's gift subscriptions.
func (s *GiftsService) List(ctx context.Context, params *GiftListParams) ([]Gift, error) {
	path := "/gifts"
	if params != nil {
		path = newQuery().int("page", params.Page).int("per_page", params.PerPage).apply(path)
	}
	return getData[[]Gift](ctx, s.client, http.MethodGet, path, nil)
}
