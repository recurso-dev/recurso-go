package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Wallet is a customer's prepaid balance in one currency (minor units).
// Invoices drain the wallet before credit notes and the payment gateway.
type Wallet struct {
	ID                    string    `json:"id"`
	CustomerID            string    `json:"customer_id"`
	Currency              string    `json:"currency"`
	Balance               int64     `json:"balance"`
	AutoRechargeThreshold *int64    `json:"auto_recharge_threshold,omitempty"`
	AutoRechargeAmount    *int64    `json:"auto_recharge_amount,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// WalletTransaction is one append-only wallet movement.
type WalletTransaction struct {
	ID           string     `json:"id"`
	WalletID     string     `json:"wallet_id"`
	Type         string     `json:"type"` // "top_up" | "drain" | "expiry"
	Source       string     `json:"source,omitempty"`
	Amount       int64      `json:"amount"`
	Remaining    *int64     `json:"remaining,omitempty"`
	BalanceAfter int64      `json:"balance_after"`
	InvoiceID    *string    `json:"invoice_id,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// WalletCreateParams creates a wallet (one per customer+currency).
type WalletCreateParams struct {
	CustomerID            string `json:"customer_id"`
	Currency              string `json:"currency"`
	AutoRechargeThreshold *int64 `json:"auto_recharge_threshold,omitempty"`
	AutoRechargeAmount    *int64 `json:"auto_recharge_amount,omitempty"`
}

// WalletTopUpParams adds balance. Source "manual" records money already
// received; "promotional" grants credit and may carry ExpiresAt.
type WalletTopUpParams struct {
	Amount    int64      `json:"amount"`
	Source    string     `json:"source,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// WalletAutoRechargeParams sets (both fields) or clears (both nil) the rule.
type WalletAutoRechargeParams struct {
	AutoRechargeThreshold *int64 `json:"auto_recharge_threshold"`
	AutoRechargeAmount    *int64 `json:"auto_recharge_amount"`
}

// WalletsService groups the prepaid-wallet endpoints.
type WalletsService struct{ client *Client }

// Create creates a prepaid wallet.
func (s *WalletsService) Create(ctx context.Context, params *WalletCreateParams) (*Wallet, error) {
	return getData[*Wallet](ctx, s.client, http.MethodPost, "/wallets", params)
}

// Get fetches one wallet.
func (s *WalletsService) Get(ctx context.Context, id string) (*Wallet, error) {
	return getData[*Wallet](ctx, s.client, http.MethodGet, fmt.Sprintf("/wallets/%s", id), nil)
}

// ForCustomer lists a customer's wallets across currencies.
func (s *WalletsService) ForCustomer(ctx context.Context, customerID string) ([]Wallet, error) {
	return getData[[]Wallet](ctx, s.client, http.MethodGet, fmt.Sprintf("/customers/%s/wallets", customerID), nil)
}

// TopUp adds balance to a wallet.
func (s *WalletsService) TopUp(ctx context.Context, id string, params *WalletTopUpParams) (*WalletTransaction, error) {
	return getData[*WalletTransaction](ctx, s.client, http.MethodPost, fmt.Sprintf("/wallets/%s/top-up", id), params)
}

// Transactions lists the wallet's movement history, newest first.
func (s *WalletsService) Transactions(ctx context.Context, id string) ([]WalletTransaction, error) {
	return getData[[]WalletTransaction](ctx, s.client, http.MethodGet, fmt.Sprintf("/wallets/%s/transactions", id), nil)
}

// SetAutoRecharge sets or clears the auto-recharge rule.
func (s *WalletsService) SetAutoRecharge(ctx context.Context, id string, params *WalletAutoRechargeParams) (*Wallet, error) {
	return getData[*Wallet](ctx, s.client, http.MethodPut, fmt.Sprintf("/wallets/%s/auto-recharge", id), params)
}

// SetCommitment sets the subscription's per-period minimum in minor units
// (0 clears it): shortfalls bill a true-up line at period close.
func (s *SubscriptionsService) SetCommitment(ctx context.Context, id string, amount int64) (*Subscription, error) {
	body := map[string]int64{"amount": amount}
	return getData[*Subscription](ctx, s.client, http.MethodPut, fmt.Sprintf("/subscriptions/%s/commitment", id), body)
}

// UsageAlert is a threshold on a metric that fires once per billing period.
type UsageAlert struct {
	ID                   string     `json:"id"`
	SubscriptionID       string     `json:"subscription_id"`
	MetricCode           string     `json:"metric_code"`
	ThresholdType        string     `json:"threshold_type"` // "quantity" | "percent_of_limit"
	Threshold            int64      `json:"threshold"`
	LastFiredPeriodStart *time.Time `json:"last_fired_period_start,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

// UsageAlertCreateParams creates an alert.
type UsageAlertCreateParams struct {
	SubscriptionID string `json:"subscription_id"`
	MetricCode     string `json:"metric_code"`
	ThresholdType  string `json:"threshold_type"`
	Threshold      int64  `json:"threshold"`
}

// UsageAlertsService groups the usage-alert endpoints.
type UsageAlertsService struct{ client *Client }

// Create creates a usage threshold alert.
func (s *UsageAlertsService) Create(ctx context.Context, params *UsageAlertCreateParams) (*UsageAlert, error) {
	return getData[*UsageAlert](ctx, s.client, http.MethodPost, "/usage-alerts", params)
}

// List lists alerts, optionally scoped to one subscription.
func (s *UsageAlertsService) List(ctx context.Context, subscriptionID string) ([]UsageAlert, error) {
	path := "/usage-alerts"
	if subscriptionID != "" {
		path = newQuery().str("subscription_id", subscriptionID).apply(path)
	}
	return getData[[]UsageAlert](ctx, s.client, http.MethodGet, path, nil)
}

// Delete removes an alert.
func (s *UsageAlertsService) Delete(ctx context.Context, id string) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/usage-alerts/%s", id), nil, nil)
}

// UsageBatchItemResult is one event's outcome in a batch ingest.
type UsageBatchItemResult struct {
	Index   int    `json:"index"`
	Status  string `json:"status"` // "recorded" | "duplicate" | "error"
	EventID string `json:"event_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// RecordBatch ingests up to 500 events with per-item results. Events with
// a TransactionID are idempotent: duplicates collapse to the original.
func (s *UsageService) RecordBatch(ctx context.Context, events []UsageRecordParams) ([]UsageBatchItemResult, error) {
	body := map[string]any{"events": events}
	return getData[[]UsageBatchItemResult](ctx, s.client, http.MethodPost, "/usage/events/batch", body)
}

// AuditLog is one immutable record of a config-grade mutation.
type AuditLog struct {
	ID          string    `json:"id"`
	Actor       string    `json:"actor"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id,omitempty"`
	Status      int       `json:"status"`
	RequestBody string    `json:"request_body,omitempty"`
	IP          string    `json:"ip,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuditLogListParams filters the audit trail.
type AuditLogListParams struct {
	EntityType string
	EntityID   string
	Actor      string
	Limit      int
	Offset     int
}

// AuditLogsService exposes the append-only audit trail.
type AuditLogsService struct{ client *Client }

// List returns audit entries, newest first.
func (s *AuditLogsService) List(ctx context.Context, params *AuditLogListParams) ([]AuditLog, error) {
	path := "/audit-logs"
	if params != nil {
		path = newQuery().
			str("entity_type", params.EntityType).
			str("entity_id", params.EntityID).
			str("actor", params.Actor).
			int("limit", params.Limit).
			int("offset", params.Offset).
			apply(path)
	}
	return getData[[]AuditLog](ctx, s.client, http.MethodGet, path, nil)
}
