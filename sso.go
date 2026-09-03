package recurso

import (
	"context"
	"net/http"
)

// SSOConnection is the tenant's SAML single-sign-on configuration.
// Configured reports whether an IdP has been set up; SPMetadataURL and
// SPACSURL are the values to register with the IdP.
type SSOConnection struct {
	TenantID       string `json:"tenant_id"`
	IdPEntityID    string `json:"idp_entity_id"`
	IdPSSOURL      string `json:"idp_sso_url"`
	IdPCertificate string `json:"idp_certificate"`
	IdPMetadataXML string `json:"idp_metadata_xml"`
	Enabled        bool   `json:"enabled"`
	Configured     bool   `json:"configured"`
	SPMetadataURL  string `json:"sp_metadata_url"`
	SPACSURL       string `json:"sp_acs_url"`
}

// SSOConnectionParams creates or updates the SAML IdP configuration. Supply
// either IdPMetadataXML or the entity id / SSO URL / certificate triple.
type SSOConnectionParams struct {
	IdPEntityID    string `json:"idp_entity_id,omitempty"`
	IdPSSOURL      string `json:"idp_sso_url,omitempty"`
	IdPCertificate string `json:"idp_certificate,omitempty"`
	IdPMetadataXML string `json:"idp_metadata_xml,omitempty"`
	Enabled        bool   `json:"enabled"`
}

// SSOService groups the SAML SSO connection endpoints (owner/admin only).
type SSOService struct{ client *Client }

// Get returns the tenant's SSO connection.
func (s *SSOService) Get(ctx context.Context) (*SSOConnection, error) {
	return getData[*SSOConnection](ctx, s.client, http.MethodGet, "/sso/connection", nil)
}

// Upsert creates or updates the SSO connection.
func (s *SSOService) Upsert(ctx context.Context, params *SSOConnectionParams) (*SSOConnection, error) {
	return getData[*SSOConnection](ctx, s.client, http.MethodPut, "/sso/connection", params)
}

// Delete removes the SSO connection.
func (s *SSOService) Delete(ctx context.Context) (*MessageResponse, error) {
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodDelete, "/sso/connection", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
