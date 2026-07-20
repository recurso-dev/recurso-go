package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Organization groups multiple tenants under one umbrella for consolidated
// reporting.
type Organization struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	OwnerEmail string    `json:"owner_email"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TenantMRR is one tenant's contribution to an organization's MRR.
type TenantMRR struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	MRR        int64  `json:"mrr"`
	Currency   string `json:"currency"`
}

// CurrencyMRR is an organization's MRR in one currency, broken down by tenant.
type CurrencyMRR struct {
	TotalMRR     int64       `json:"total_mrr"`
	Currency     string      `json:"currency"`
	ConvertedMRR int64       `json:"converted_mrr"`
	Rate         float64     `json:"rate"`
	FXError      string      `json:"fx_error,omitempty"`
	ByTenant     []TenantMRR `json:"by_tenant"`
}

// OrgMRRMetrics is the consolidated MRR report across an organization's
// tenants.
type OrgMRRMetrics struct {
	ByCurrency        []CurrencyMRR `json:"by_currency"`
	NormalizedMRR     int64         `json:"normalized_mrr"`
	ReportingCurrency string        `json:"reporting_currency"`
	FX                FXSnapshot    `json:"fx"`
}

// OrganizationCreateParams is the body for creating an organization.
type OrganizationCreateParams struct {
	Name       string `json:"name"`
	OwnerEmail string `json:"owner_email"`
}

// OrganizationUpdateParams is the body for updating an organization. Omitted
// fields are left unchanged.
type OrganizationUpdateParams struct {
	Name       string `json:"name,omitempty"`
	OwnerEmail string `json:"owner_email,omitempty"`
}

// OrganizationsService groups the organization endpoints.
type OrganizationsService struct{ client *Client }

// Create creates an organization.
func (s *OrganizationsService) Create(ctx context.Context, params *OrganizationCreateParams) (*Organization, error) {
	var out Organization
	if err := s.client.do(ctx, http.MethodPost, "/organizations", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns all organizations.
func (s *OrganizationsService) List(ctx context.Context) ([]Organization, error) {
	return getData[[]Organization](ctx, s.client, http.MethodGet, "/organizations", nil)
}

// Get retrieves an organization by ID.
func (s *OrganizationsService) Get(ctx context.Context, id string) (*Organization, error) {
	out, err := getData[Organization](ctx, s.client, http.MethodGet, "/organizations/"+id, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates an organization.
func (s *OrganizationsService) Update(ctx context.Context, id string, params *OrganizationUpdateParams) (*Organization, error) {
	out, err := getData[Organization](ctx, s.client, http.MethodPut, "/organizations/"+id, params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes an organization.
func (s *OrganizationsService) Delete(ctx context.Context, id string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodDelete, "/organizations/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Tenants lists an organization's tenants.
func (s *OrganizationsService) Tenants(ctx context.Context, id string) ([]Tenant, error) {
	return getData[[]Tenant](ctx, s.client, http.MethodGet, fmt.Sprintf("/organizations/%s/tenants", id), nil)
}

// AddTenant attaches a tenant to an organization.
func (s *OrganizationsService) AddTenant(ctx context.Context, id, tenantID string) (*StatusResponse, error) {
	body := map[string]any{"tenant_id": tenantID}
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/organizations/%s/tenants", id), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveTenant detaches a tenant from an organization.
func (s *OrganizationsService) RemoveTenant(ctx context.Context, id, tenantID string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/organizations/%s/tenants/%s", id, tenantID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MRR returns the consolidated MRR across the organization's tenants, grouped
// by currency and tenant.
func (s *OrganizationsService) MRR(ctx context.Context, id string) (*OrgMRRMetrics, error) {
	out, err := getData[OrgMRRMetrics](ctx, s.client, http.MethodGet, fmt.Sprintf("/organizations/%s/analytics/mrr", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
