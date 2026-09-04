package recurso

import (
	"context"
	"net/http"
)

// Version is the API build version. GatewayMode is "test", "live", or
// "none" depending on how the deployment's payment gateway is configured.
type Version struct {
	Version     string `json:"version"`
	GatewayMode string `json:"gateway_mode"`
}

// SystemService groups the unversioned deployment endpoints that live
// beside the /v1 prefix.
type SystemService struct{ client *Client }

// Version returns the API build version. The endpoint is unauthenticated,
// which makes it a cheap connectivity check.
func (s *SystemService) Version(ctx context.Context) (*Version, error) {
	var out Version
	if err := s.client.doRoot(ctx, http.MethodGet, "/version", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
