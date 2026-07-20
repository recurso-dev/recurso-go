package recurso

import (
	"context"
	"net/http"
	"time"
)

// AccountingConnection is a link to an external accounting system. OAuth
// access/refresh tokens are stored server-side and never serialized.
type AccountingConnection struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Provider       string     `json:"provider"`
	TokenExpiresAt *time.Time `json:"token_expires_at"`
	RealmID        string     `json:"realm_id"`
	LastSyncAt     *time.Time `json:"last_sync_at"`
	SyncStatus     string     `json:"sync_status"`
	LastError      string     `json:"last_error"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
}

// AccountingSyncLog is one per-entity result from an accounting sync run.
type AccountingSyncLog struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	ConnectionID string    `json:"connection_id"`
	EntityType   string    `json:"entity_type"`
	EntityID     string    `json:"entity_id"`
	ExternalID   string    `json:"external_id"`
	Action       string    `json:"action"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message"`
	SyncedAt     time.Time `json:"synced_at"`
}

// AccountingConnectTokenParams are the credentials for a token-based
// accounting provider. NetSuite requires AccountID and a SuiteTalk OAuth 2.0
// AccessToken minted in NetSuite; Tally takes no credentials.
type AccountingConnectTokenParams struct {
	AccountID   string `json:"account_id,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
}

// AccountingService groups the accounting-integration endpoints.
type AccountingService struct{ client *Client }

// Connections lists the tenant's accounting connections (OAuth tokens are
// never serialized).
func (s *AccountingService) Connections(ctx context.Context) ([]AccountingConnection, error) {
	return getData[[]AccountingConnection](ctx, s.client, http.MethodGet, "/accounting/connections", nil)
}

// ConnectToken creates or refreshes a connection for a token-based provider
// ("netsuite" or "tally") outside the browser OAuth flow. params may be nil
// for Tally.
func (s *AccountingService) ConnectToken(ctx context.Context, provider string, params *AccountingConnectTokenParams) (*AccountingConnection, error) {
	out, err := getData[AccountingConnection](ctx, s.client, http.MethodPost, "/accounting/connect-token/"+provider, params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Disconnect removes an accounting connection.
func (s *AccountingService) Disconnect(ctx context.Context, id string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodDelete, "/accounting/connections/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Sync triggers a sync to the connected accounting systems.
func (s *AccountingService) Sync(ctx context.Context) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodPost, "/accounting/sync", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SyncStatus returns the 50 most recent per-entity sync results.
func (s *AccountingService) SyncStatus(ctx context.Context) ([]AccountingSyncLog, error) {
	return getData[[]AccountingSyncLog](ctx, s.client, http.MethodGet, "/accounting/sync/status", nil)
}
