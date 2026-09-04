package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Subscription is a customer's recurring subscription.
type Subscription struct {
	ID                     string    `json:"id"`
	TenantID               string    `json:"tenant_id"`
	CustomerID             string    `json:"customer_id"`
	PlanID                 string    `json:"plan_id"`
	Status                 string    `json:"status"`
	CurrentPeriodStart     time.Time `json:"current_period_start"`
	CurrentPeriodEnd       time.Time `json:"current_period_end"`
	CancelAtPeriodEnd      bool      `json:"cancel_at_period_end"`
	CanceledAt             time.Time `json:"canceled_at"`
	CancellationReason     string    `json:"cancellation_reason"`
	CancellationFeedback   string    `json:"cancellation_feedback"`
	BillingAnchor          time.Time `json:"billing_anchor"`
	BillingAnchorType      string    `json:"billing_anchor_type"`
	BillingAnchorDay       int       `json:"billing_anchor_day"`
	PaymentTerms           string    `json:"payment_terms"`
	CouponID               string    `json:"coupon_id"`
	ReferenceID            string    `json:"reference_id"`
	MandateID              string    `json:"mandate_id"`
	RazorpaySubscriptionID string    `json:"razorpay_subscription_id"`
	StripeSubscriptionID   string    `json:"stripe_subscription_id"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// SubscriptionCreateParams is the body for creating a subscription.
type SubscriptionCreateParams struct {
	CustomerID        string `json:"customer_id"`
	PlanID            string `json:"plan_id"`
	CouponCode        string `json:"coupon_code,omitempty"`
	StartDate         string `json:"start_date,omitempty"`
	BillingAnchorType string `json:"billing_anchor_type,omitempty"`
	PaymentTerms      string `json:"payment_terms,omitempty"`
	// TrialDays, when greater than zero, starts the subscription in
	// "trialing" and converts it to "active" (issuing the first invoice)
	// when the trial ends.
	TrialDays int `json:"trial_days,omitempty"`
}

// SubscriptionListParams filters the subscription list.
type SubscriptionListParams struct {
	Status string
	Q      string
	// PlanID filters to one plan's subscriptions.
	PlanID string
	// CustomerID filters to one customer's subscriptions.
	CustomerID string
	// StartedAfter keeps subscriptions whose current period started at or
	// after this instant. The zero value means no lower bound.
	StartedAfter time.Time
	Limit        int
	Page         int
}

// SubscriptionUpdateParams changes a subscription's plan (upgrade/downgrade).
type SubscriptionUpdateParams struct {
	PlanID string `json:"plan_id"`
}

// SubscriptionCancelParams controls how a subscription is canceled. When
// Immediately is nil/false the cancellation takes effect at period end.
type SubscriptionCancelParams struct {
	CancelAtPeriodEnd *bool  `json:"cancel_at_period_end,omitempty"`
	Immediately       *bool  `json:"immediately,omitempty"`
	Reason            string `json:"reason"`
	Feedback          string `json:"feedback,omitempty"`
	RevokeConsent     *bool  `json:"revoke_consent,omitempty"`
}

// CancelResult is returned by Cancel.
type CancelResult struct {
	ID                 string    `json:"id"`
	Status             string    `json:"status"`
	CancelAtPeriodEnd  bool      `json:"cancel_at_period_end"`
	CancelledAt        time.Time `json:"cancelled_at"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CancellationReason string    `json:"cancellation_reason"`
	Message            string    `json:"message"`
}

// ReactivateResult is returned by Reactivate.
type ReactivateResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// UnbilledCharge is an ad-hoc charge queued onto a subscription's next invoice.
type UnbilledCharge struct {
	ID             string    `json:"id"`
	SubscriptionID string    `json:"subscription_id"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description"`
	HSNCode        string    `json:"hsn_code"`
	Status         string    `json:"status"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChargeCreateParams adds an unbilled charge.
type ChargeCreateParams struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	HSNCode     string `json:"hsn_code,omitempty"`
}

// AdvanceParams bills a subscription ahead by the given number of periods.
type AdvanceParams struct {
	Periods int `json:"periods"`
}

// SubscriptionAddon is a plan attached to a subscription as a priced add-on.
type SubscriptionAddon struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SubscriptionID string    `json:"subscription_id"`
	PlanID         string    `json:"plan_id"`
	Quantity       int       `json:"quantity"`
	CreatedAt      time.Time `json:"created_at"`
}

// AddonCreateParams attaches an add-on plan to a subscription.
type AddonCreateParams struct {
	PlanID   string `json:"plan_id"`
	Quantity int    `json:"quantity"`
}

// PlanChangePreview is the read-only proration breakdown for a plan change.
type PlanChangePreview struct {
	SubscriptionID    string    `json:"subscription_id"`
	CurrentPlanID     string    `json:"current_plan_id"`
	NewPlanID         string    `json:"new_plan_id"`
	Currency          string    `json:"currency"`
	CreditAmount      int64     `json:"credit_amount"`
	ChargeAmount      int64     `json:"charge_amount"`
	NetAmount         int64     `json:"net_amount"`
	TaxAmount         int64     `json:"tax_amount"`
	TotalAmount       int64     `json:"total_amount"`
	EffectiveDate     time.Time `json:"effective_date"`
	NextInvoiceAmount int64     `json:"next_invoice_amount"`
	IsUpgrade         bool      `json:"is_upgrade"`
}

// SubscriptionDimensionUsage is one dimension's usage for a subscription.
type SubscriptionDimensionUsage struct {
	Dimension        string `json:"dimension"`
	PeriodQuantity   int64  `json:"period_quantity"`
	LifetimeQuantity int64  `json:"lifetime_quantity"`
	LimitValue       *int64 `json:"limit_value"`
	Remaining        *int64 `json:"remaining"`
}

// SubscriptionUsage is the current-period usage report for a subscription.
type SubscriptionUsage struct {
	SubscriptionID     string                       `json:"subscription_id"`
	CustomerID         string                       `json:"customer_id"`
	CurrentPeriodStart time.Time                    `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                    `json:"current_period_end"`
	Dimensions         []SubscriptionDimensionUsage `json:"dimensions"`
}

// SubscriptionsService groups the subscription lifecycle endpoints.
type SubscriptionsService struct{ client *Client }

// Create creates a subscription and generates its first invoice.
func (s *SubscriptionsService) Create(ctx context.Context, params *SubscriptionCreateParams) (*Subscription, error) {
	var out Subscription
	if err := s.client.do(ctx, http.MethodPost, "/subscriptions", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's subscriptions.
func (s *SubscriptionsService) List(ctx context.Context, params *SubscriptionListParams) ([]Subscription, error) {
	path := "/subscriptions"
	if params != nil {
		q := newQuery().
			str("status", params.Status).
			str("q", params.Q).
			str("plan_id", params.PlanID).
			str("customer_id", params.CustomerID).
			int("limit", params.Limit).
			int("page", params.Page)
		if !params.StartedAfter.IsZero() {
			q.str("started_after", params.StartedAfter.UTC().Format(time.RFC3339))
		}
		path = q.apply(path)
	}
	return getData[[]Subscription](ctx, s.client, http.MethodGet, path, nil)
}

// Get returns one subscription by id, scoped to the authenticated tenant.
func (s *SubscriptionsService) Get(ctx context.Context, id string) (*Subscription, error) {
	return getData[*Subscription](ctx, s.client, http.MethodGet, fmt.Sprintf("/subscriptions/%s", id), nil)
}

// Update changes the subscription's plan.
func (s *SubscriptionsService) Update(ctx context.Context, id string, params *SubscriptionUpdateParams) (*Subscription, error) {
	var out Subscription
	if err := s.client.do(ctx, http.MethodPut, fmt.Sprintf("/subscriptions/%s", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PreviewChange previews the proration for switching to newPlanID. Nothing is
// charged or persisted.
func (s *SubscriptionsService) PreviewChange(ctx context.Context, id, newPlanID string) (*PlanChangePreview, error) {
	path := newQuery().str("plan_id", newPlanID).apply(fmt.Sprintf("/subscriptions/%s/preview-change", id))
	var out PlanChangePreview
	if err := s.client.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Cancel cancels a subscription (at period end by default).
func (s *SubscriptionsService) Cancel(ctx context.Context, id string, params *SubscriptionCancelParams) (*CancelResult, error) {
	var out CancelResult
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/cancel", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Reactivate reactivates a cancelled subscription.
func (s *SubscriptionsService) Reactivate(ctx context.Context, id string) (*ReactivateResult, error) {
	var out ReactivateResult
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/reactivate", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Pause pauses a subscription.
func (s *SubscriptionsService) Pause(ctx context.Context, id string) (*Subscription, error) {
	out, err := getData[Subscription](ctx, s.client, http.MethodPost, fmt.Sprintf("/subscriptions/%s/pause", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Resume resumes a paused subscription.
func (s *SubscriptionsService) Resume(ctx context.Context, id string) (*Subscription, error) {
	out, err := getData[Subscription](ctx, s.client, http.MethodPost, fmt.Sprintf("/subscriptions/%s/resume", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Charges lists a subscription's pending unbilled charges.
func (s *SubscriptionsService) Charges(ctx context.Context, id string) ([]UnbilledCharge, error) {
	return getData[[]UnbilledCharge](ctx, s.client, http.MethodGet, fmt.Sprintf("/subscriptions/%s/charges", id), nil)
}

// AddCharge records an ad-hoc charge for the subscription's next invoice.
func (s *SubscriptionsService) AddCharge(ctx context.Context, id string, params *ChargeCreateParams) (*UnbilledCharge, error) {
	var out UnbilledCharge
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/charges", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Advance bills the subscription ahead for the given number of periods.
func (s *SubscriptionsService) Advance(ctx context.Context, id string, params *AdvanceParams) (*Invoice, error) {
	var out Invoice
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/advance", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Usage returns the subscription's current-period usage report.
func (s *SubscriptionsService) Usage(ctx context.Context, id string) (*SubscriptionUsage, error) {
	var out SubscriptionUsage
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/subscriptions/%s/usage", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddAddon attaches an add-on plan to a subscription.
func (s *SubscriptionsService) AddAddon(ctx context.Context, id string, params *AddonCreateParams) (*SubscriptionAddon, error) {
	var out SubscriptionAddon
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/addons", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAddons lists a subscription's add-ons.
func (s *SubscriptionsService) ListAddons(ctx context.Context, id string) ([]SubscriptionAddon, error) {
	return getData[[]SubscriptionAddon](ctx, s.client, http.MethodGet, fmt.Sprintf("/subscriptions/%s/addons", id), nil)
}

// RemoveAddon detaches an add-on from a subscription.
func (s *SubscriptionsService) RemoveAddon(ctx context.Context, id, addonID string) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/subscriptions/%s/addons/%s", id, addonID), nil, nil)
}

// SubscriptionFinancialSummary is a subscription's recurring economics and
// per-currency outstanding position (minor units).
type SubscriptionFinancialSummary struct {
	SubscriptionID        string               `json:"subscription_id"`
	Status                string               `json:"status"`
	Currency              string               `json:"currency"`
	MRR                   int64                `json:"mrr"`
	RecurringAmount       int64                `json:"recurring_amount"`
	IntervalUnit          string               `json:"interval_unit"`
	IntervalCount         int                  `json:"interval_count"`
	CurrentPeriodStart    time.Time            `json:"current_period_start"`
	CurrentPeriodEnd      time.Time            `json:"current_period_end"`
	NextInvoiceDate       *time.Time           `json:"next_invoice_date"`
	NextInvoiceBaseAmount int64                `json:"next_invoice_base_amount"`
	CouponID              *string              `json:"coupon_id"`
	DiscountActive        bool                 `json:"discount_active"`
	Outstanding           []CurrencyFinancials `json:"outstanding"`
}

// CancelPreview is the read-only financial consequence of canceling a
// subscription now or at period end (minor units). Nothing is changed.
type CancelPreview struct {
	SubscriptionID           string    `json:"subscription_id"`
	Immediately              bool      `json:"immediately"`
	EffectiveDate            time.Time `json:"effective_date"`
	ResultingStatus          string    `json:"resulting_status"`
	CancelAtPeriodEnd        bool      `json:"cancel_at_period_end"`
	Currency                 string    `json:"currency"`
	DeferredRevenueForfeited int64     `json:"deferred_revenue_forfeited"`
	RecognizedAsBreakage     int64     `json:"recognized_as_breakage"`
	AvoidedFutureRecurring   int64     `json:"avoided_future_recurring"`
	FlatFeeRefund            int64     `json:"flat_fee_refund"`
}

// SubscriptionChange is one lifecycle event: a "status" or "plan" change.
type SubscriptionChange struct {
	ID             string    `json:"id"`
	SubscriptionID string    `json:"subscription_id"`
	ChangeType     string    `json:"change_type"`
	FromValue      *string   `json:"from_value"`
	ToValue        *string   `json:"to_value"`
	ChangedAt      time.Time `json:"changed_at"`
}

// SubscriptionHistory is a subscription's lifecycle timeline.
type SubscriptionHistory struct {
	SubscriptionID string               `json:"subscription_id"`
	History        []SubscriptionChange `json:"history"`
}

// CancellationReason is one preset reason customers can pick when
// canceling.
type CancellationReason struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	AllowsFeedback bool   `json:"allows_feedback"`
}

// FinancialSummary returns the subscription's MRR, next invoice, and
// outstanding position.
func (s *SubscriptionsService) FinancialSummary(ctx context.Context, id string) (*SubscriptionFinancialSummary, error) {
	return getData[*SubscriptionFinancialSummary](ctx, s.client, http.MethodGet, fmt.Sprintf("/subscriptions/%s/financial-summary", id), nil)
}

// CancelPreview previews the financial consequence of canceling.
// immediately selects an immediate cancellation instead of period end.
func (s *SubscriptionsService) CancelPreview(ctx context.Context, id string, immediately bool) (*CancelPreview, error) {
	path := newQuery().boolean("immediately", immediately).apply(fmt.Sprintf("/subscriptions/%s/cancel-preview", id))
	return getData[*CancelPreview](ctx, s.client, http.MethodGet, path, nil)
}

// History returns the subscription's status and plan change timeline.
func (s *SubscriptionsService) History(ctx context.Context, id string) (*SubscriptionHistory, error) {
	return getData[*SubscriptionHistory](ctx, s.client, http.MethodGet, fmt.Sprintf("/subscriptions/%s/history", id), nil)
}

// BillUsage generates an interim invoice for usage accrued so far in the
// current period (progressive billing).
func (s *SubscriptionsService) BillUsage(ctx context.Context, id string) (*Invoice, error) {
	var out Invoice
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/bill-usage", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Consent returns the recurring-billing consent tied to the subscription.
func (s *SubscriptionsService) Consent(ctx context.Context, id string) (*Consent, error) {
	var out Consent
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/subscriptions/%s/consent", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancellationReasons lists the preset cancellation reasons.
func (s *SubscriptionsService) CancellationReasons(ctx context.Context) ([]CancellationReason, error) {
	return getData[[]CancellationReason](ctx, s.client, http.MethodGet, "/cancellation-reasons", nil)
}
