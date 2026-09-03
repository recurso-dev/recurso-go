package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// APIKey is a tenant API key. KeyValue (the raw secret) is populated only in
// the creation response.
type APIKey struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	KeyValue  string    `json:"key_value"`
	KeyPrefix string    `json:"key_prefix"`
	Type      string    `json:"type"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// DeveloperService groups the API-key management endpoints.
type DeveloperService struct{ client *Client }

// ListKeys returns the tenant's API keys (raw key values are never returned).
func (s *DeveloperService) ListKeys(ctx context.Context) ([]APIKey, error) {
	return getData[[]APIKey](ctx, s.client, http.MethodGet, "/developer/keys", nil)
}

// CreateKey generates a new secret API key. The raw key_value is returned only
// in this response.
func (s *DeveloperService) CreateKey(ctx context.Context) (*APIKey, error) {
	var out APIKey
	if err := s.client.do(ctx, http.MethodPost, "/developer/keys", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RevokeKey revokes an API key. Requests signed with it fail from the next
// call onward.
func (s *DeveloperService) RevokeKey(ctx context.Context, id string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/developer/keys/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
