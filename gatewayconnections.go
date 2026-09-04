package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// GatewayConnection is a bring-your-own payment-gateway connection. Secret
// keys are vaulted server-side and never serialized; HasWebhookSecret
// reports whether a signing secret is stored.
type GatewayConnection struct {
	ID               string    `json:"id"`
	Provider         string    `json:"provider"` // stripe | razorpay | gocardless
	Mode             string    `json:"mode"`     // test | live
	PublicKey        string    `json:"public_key"`
	HasWebhookSecret bool      `json:"has_webhook_secret"`
	WebhookPath      string    `json:"webhook_path"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// GatewayConnectionList is the tenant's gateway connections plus whether
// the secrets vault is available for new connections.
type GatewayConnectionList struct {
	Connections []GatewayConnection `json:"connections"`
	VaultReady  bool                `json:"vault_ready"`
}

// GatewayConnectionCreateParams connects (or replaces) a payment gateway.
// Provider is "stripe" or "razorpay"; Mode is "test" or "live".
type GatewayConnectionCreateParams struct {
	Provider      string `json:"provider"`
	Mode          string `json:"mode"`
	PublicKey     string `json:"public_key"`
	SecretKey     string `json:"secret_key"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

// GatewayConnectionsService groups the BYO payment-gateway endpoints.
type GatewayConnectionsService struct{ client *Client }

// List returns the tenant's gateway connections.
func (s *GatewayConnectionsService) List(ctx context.Context) (*GatewayConnectionList, error) {
	return getData[*GatewayConnectionList](ctx, s.client, http.MethodGet, "/gateway-connections", nil)
}

// Create connects a payment gateway, replacing any existing connection for
// the same provider.
func (s *GatewayConnectionsService) Create(ctx context.Context, params *GatewayConnectionCreateParams) (*GatewayConnection, error) {
	return getData[*GatewayConnection](ctx, s.client, http.MethodPost, "/gateway-connections", params)
}

// Delete disconnects a payment gateway ("stripe" or "razorpay").
func (s *GatewayConnectionsService) Delete(ctx context.Context, provider string) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/gateway-connections/%s", provider), nil, nil)
}

// SetWebhookSecret stores the webhook signing secret on the provider's
// active connection.
func (s *GatewayConnectionsService) SetWebhookSecret(ctx context.Context, provider, webhookSecret string) error {
	body := map[string]string{"webhook_secret": webhookSecret}
	return s.client.do(ctx, http.MethodPut, fmt.Sprintf("/gateway-connections/%s/webhook-secret", provider), body, nil)
}
