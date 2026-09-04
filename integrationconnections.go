package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// IntegrationConnection is a bring-your-own tax, CRM, or storage
// integration. Config holds the provider's non-secret settings; secrets are
// vaulted server-side and never serialized.
type IntegrationConnection struct {
	ID         string            `json:"id"`
	Category   string            `json:"category"` // tax | crm | storage
	Provider   string            `json:"provider"`
	Config     map[string]string `json:"config"`
	HasSecrets bool              `json:"has_secrets"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// IntegrationConnectionList is the tenant's integrations plus whether the
// secrets vault is available for new connections.
type IntegrationConnectionList struct {
	Connections []IntegrationConnection `json:"connections"`
	VaultReady  bool                    `json:"vault_ready"`
}

// IntegrationConnectionCreateParams connects (or replaces) an integration.
// Config carries the provider's settings and credentials.
type IntegrationConnectionCreateParams struct {
	Category string            `json:"category"`
	Provider string            `json:"provider"`
	Config   map[string]string `json:"config"`
}

// CRMSyncResult reports a CRM contact sync.
type CRMSyncResult struct {
	ContactsSynced    int `json:"contacts_synced"`
	ContactsRemaining int `json:"contacts_remaining"`
}

// IntegrationConnectionsService groups the BYO tax/CRM/storage integration
// endpoints.
type IntegrationConnectionsService struct{ client *Client }

// List returns the tenant's integration connections.
func (s *IntegrationConnectionsService) List(ctx context.Context) (*IntegrationConnectionList, error) {
	return getData[*IntegrationConnectionList](ctx, s.client, http.MethodGet, "/integration-connections", nil)
}

// Create connects an integration, replacing any existing connection for the
// same category and provider.
func (s *IntegrationConnectionsService) Create(ctx context.Context, params *IntegrationConnectionCreateParams) (*IntegrationConnection, error) {
	return getData[*IntegrationConnection](ctx, s.client, http.MethodPost, "/integration-connections", params)
}

// Delete disconnects an integration.
func (s *IntegrationConnectionsService) Delete(ctx context.Context, category, provider string) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/integration-connections/%s/%s", category, provider), nil, nil)
}

// SyncCRM pushes the workspace's customers to its connected CRM now.
func (s *IntegrationConnectionsService) SyncCRM(ctx context.Context) (*CRMSyncResult, error) {
	return getData[*CRMSyncResult](ctx, s.client, http.MethodPost, "/crm/sync", nil)
}
