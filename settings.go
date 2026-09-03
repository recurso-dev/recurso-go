package recurso

import (
	"context"
	"net/http"
	"time"
)

// GSTConfig is the tenant's Indian GST identity. GSTRate is a percentage.
type GSTConfig struct {
	GSTIN     string  `json:"gstin"`
	StateCode string  `json:"state_code"`
	StateName string  `json:"state_name"`
	SACCode   string  `json:"sac_code"`
	GSTRate   float64 `json:"gst_rate"`
	PAN       string  `json:"pan"`
	LegalName string  `json:"legal_name"`
	TradeName string  `json:"trade_name"`
	Address   string  `json:"address"`
	HasLUT    bool    `json:"has_lut"`
}

// GSTINValidation is the result of validating a GSTIN.
type GSTINValidation struct {
	Valid     bool   `json:"valid"`
	StateCode string `json:"state_code"`
	StateName string `json:"state_name"`
	PAN       string `json:"pan"`
	Message   string `json:"message"`
}

// TaxRegistration is a US sales-tax registration in one state. RegisteredAt
// is a YYYY-MM-DD date.
type TaxRegistration struct {
	StateCode          string  `json:"state_code"`
	RegistrationNumber string  `json:"registration_number,omitempty"`
	Status             string  `json:"status"` // registered | pending | not_registered
	RegisteredAt       *string `json:"registered_at,omitempty"`
}

// TaxRegistrationsParams replaces the tenant's US sales-tax registrations.
type TaxRegistrationsParams struct {
	Registrations []TaxRegistration `json:"registrations"`
}

// TaxNexusState is a US state where the tenant has sales-tax nexus.
type TaxNexusState struct {
	StateCode     string     `json:"state_code"`
	NexusType     string     `json:"nexus_type"` // physical | voluntary | economic
	EstablishedAt *time.Time `json:"established_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at,omitempty"`
}

// TaxNexusParams replaces the tenant's nexus states.
type TaxNexusParams struct {
	States []TaxNexusState `json:"states"`
}

// TaxLiabilityState is one state's sales-tax liability (minor units).
type TaxLiabilityState struct {
	StateCode       string `json:"state_code"`
	GrossSales      int64  `json:"gross_sales"`
	TaxableSales    int64  `json:"taxable_sales"`
	ExemptSales     int64  `json:"exempt_sales"`
	NonTaxableSales int64  `json:"non_taxable_sales"`
	TaxCollected    int64  `json:"tax_collected"`
	InvoiceCount    int    `json:"invoice_count"`
	HasNexus        bool   `json:"has_nexus"`
	NexusType       string `json:"nexus_type,omitempty"`
}

// TaxLiabilityReport is the per-state US sales-tax liability report. Dates
// are YYYY-MM-DD.
type TaxLiabilityReport struct {
	FromDate          string              `json:"from_date"`
	ToDate            string              `json:"to_date"`
	Currency          string              `json:"currency"`
	TotalGrossSales   int64               `json:"total_gross_sales"`
	TotalTaxCollected int64               `json:"total_tax_collected"`
	States            []TaxLiabilityState `json:"states"`
}

// TaxLiabilityParams selects the reporting window: a calendar Year, or an
// explicit From/To (YYYY-MM-DD).
type TaxLiabilityParams struct {
	Year int
	From string
	To   string
}

// NexusThreshold is one state's economic-nexus threshold rule.
type NexusThreshold struct {
	StateCode         string `json:"state_code"`
	SalesThreshold    int64  `json:"sales_threshold"`
	TxnThreshold      int    `json:"txn_threshold"`
	Combinator        string `json:"combinator"` // or | and
	MeasurementPeriod string `json:"measurement_period"`
	Certified         bool   `json:"certified"`
}

// NexusStatusState is one state's progress toward economic nexus.
type NexusStatusState struct {
	StateCode     string         `json:"state_code"`
	NexusType     string         `json:"nexus_type,omitempty"`
	EstablishedAt *time.Time     `json:"established_at,omitempty"`
	TaxableSales  int64          `json:"taxable_sales"`
	TxnCount      int            `json:"txn_count"`
	Threshold     NexusThreshold `json:"threshold"`
	ProximityPct  int            `json:"proximity_pct"`
	Crossed       bool           `json:"crossed"`
}

// NexusStatusReport is the per-state economic-nexus status for a year.
type NexusStatusReport struct {
	Year             int                `json:"year"`
	DatasetCertified bool               `json:"dataset_certified"`
	States           []NexusStatusState `json:"states"`
}

// IRPConfig holds the Indian e-invoicing (IRP) credentials.
type IRPConfig struct {
	ID           string `json:"id,omitempty"`
	TenantID     string `json:"tenant_id,omitempty"`
	Environment  string `json:"environment"` // sandbox | production
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`
	GSTIN        string `json:"gstin"`
	IsEnabled    bool   `json:"is_enabled"`
}

// IRPTestResult is returned by TestIRP.
type IRPTestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// EUEInvoiceConfig is the tenant's EU e-invoicing (EN 16931 / UBL) identity.
type EUEInvoiceConfig struct {
	Enabled     bool   `json:"enabled"`
	LegalName   string `json:"legal_name"`
	VATNumber   string `json:"vat_number"`
	CountryCode string `json:"country_code"`
	Street      string `json:"street"`
	City        string `json:"city"`
	PostalZone  string `json:"postal_zone"`
}

// USTaxConfig is the tenant's US tax identity (W-9).
type USTaxConfig struct {
	LegalName string `json:"legal_name"`
	EIN       string `json:"ein"`
	Address   string `json:"address"`
}

// InvoiceBranding is the look of the tenant's printable documents.
type InvoiceBranding struct {
	CompanyName      string `json:"company_name"`
	LogoDataURL      string `json:"logo_data_url"`
	SignatureDataURL string `json:"signature_data_url"`
	SignatoryName    string `json:"signatory_name"`
	BankDetails      string `json:"bank_details"`
	Terms            string `json:"terms"`
}

// MCPSettings controls the tenant's MCP server exposure.
type MCPSettings struct {
	Tier3Enabled bool `json:"tier3_enabled"`
}

// SettingsService groups the tenant settings endpoints: GST/IRP (India), US
// sales tax, EU e-invoicing, invoice branding, and MCP.
type SettingsService struct{ client *Client }

// GST returns the GST configuration.
func (s *SettingsService) GST(ctx context.Context) (*GSTConfig, error) {
	return getData[*GSTConfig](ctx, s.client, http.MethodGet, "/settings/gst", nil)
}

// UpdateGST replaces the GST configuration.
func (s *SettingsService) UpdateGST(ctx context.Context, params *GSTConfig) (*GSTConfig, error) {
	return getData[*GSTConfig](ctx, s.client, http.MethodPut, "/settings/gst", params)
}

// ValidateGSTIN checks a GSTIN's format and derives its state and PAN.
func (s *SettingsService) ValidateGSTIN(ctx context.Context, gstin string) (*GSTINValidation, error) {
	body := map[string]string{"gstin": gstin}
	var out GSTINValidation
	if err := s.client.do(ctx, http.MethodPost, "/settings/gst/validate", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TaxRegistrations lists the US sales-tax registrations.
func (s *SettingsService) TaxRegistrations(ctx context.Context) ([]TaxRegistration, error) {
	return getData[[]TaxRegistration](ctx, s.client, http.MethodGet, "/settings/tax/registrations", nil)
}

// SetTaxRegistrations replaces the US sales-tax registrations.
func (s *SettingsService) SetTaxRegistrations(ctx context.Context, params *TaxRegistrationsParams) ([]TaxRegistration, error) {
	return getData[[]TaxRegistration](ctx, s.client, http.MethodPut, "/settings/tax/registrations", params)
}

// TaxNexus lists the US states where the tenant has nexus. entityID scopes
// to one legal entity; empty = the whole workspace.
func (s *SettingsService) TaxNexus(ctx context.Context, entityID string) ([]TaxNexusState, error) {
	path := newQuery().str("entity_id", entityID).apply("/settings/tax/nexus")
	return getData[[]TaxNexusState](ctx, s.client, http.MethodGet, path, nil)
}

// SetTaxNexus replaces the nexus states.
func (s *SettingsService) SetTaxNexus(ctx context.Context, entityID string, params *TaxNexusParams) ([]TaxNexusState, error) {
	path := newQuery().str("entity_id", entityID).apply("/settings/tax/nexus")
	return getData[[]TaxNexusState](ctx, s.client, http.MethodPut, path, params)
}

// TaxLiability returns the per-state US sales-tax liability report.
func (s *SettingsService) TaxLiability(ctx context.Context, params *TaxLiabilityParams) (*TaxLiabilityReport, error) {
	path := "/settings/tax/liability"
	if params != nil {
		path = newQuery().int("year", params.Year).str("from", params.From).str("to", params.To).apply(path)
	}
	return getData[*TaxLiabilityReport](ctx, s.client, http.MethodGet, path, nil)
}

// TaxNexusStatus returns per-state economic-nexus progress for a year (0 =
// current year).
func (s *SettingsService) TaxNexusStatus(ctx context.Context, year int) (*NexusStatusReport, error) {
	path := newQuery().int("year", year).apply("/settings/tax/nexus/status")
	return getData[*NexusStatusReport](ctx, s.client, http.MethodGet, path, nil)
}

// IRP returns the IRP (e-invoicing) configuration.
func (s *SettingsService) IRP(ctx context.Context) (*IRPConfig, error) {
	return getData[*IRPConfig](ctx, s.client, http.MethodGet, "/settings/irp", nil)
}

// UpdateIRP creates or replaces the IRP credentials.
func (s *SettingsService) UpdateIRP(ctx context.Context, params *IRPConfig) (*IRPConfig, error) {
	return getData[*IRPConfig](ctx, s.client, http.MethodPut, "/settings/irp", params)
}

// TestIRP checks that the stored IRP credentials authenticate.
func (s *SettingsService) TestIRP(ctx context.Context) (*IRPTestResult, error) {
	var out IRPTestResult
	if err := s.client.do(ctx, http.MethodPost, "/settings/irp/test", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EUEInvoice returns the EU e-invoicing configuration.
func (s *SettingsService) EUEInvoice(ctx context.Context) (*EUEInvoiceConfig, error) {
	return getData[*EUEInvoiceConfig](ctx, s.client, http.MethodGet, "/settings/eu-einvoice", nil)
}

// UpdateEUEInvoice creates or replaces the EU e-invoicing configuration.
func (s *SettingsService) UpdateEUEInvoice(ctx context.Context, params *EUEInvoiceConfig) (*EUEInvoiceConfig, error) {
	return getData[*EUEInvoiceConfig](ctx, s.client, http.MethodPut, "/settings/eu-einvoice", params)
}

// USTax returns the US tax identity (W-9).
func (s *SettingsService) USTax(ctx context.Context) (*USTaxConfig, error) {
	return getData[*USTaxConfig](ctx, s.client, http.MethodGet, "/settings/tax/us", nil)
}

// UpdateUSTax creates or replaces the US tax identity.
func (s *SettingsService) UpdateUSTax(ctx context.Context, params *USTaxConfig) (*USTaxConfig, error) {
	return getData[*USTaxConfig](ctx, s.client, http.MethodPut, "/settings/tax/us", params)
}

// InvoiceBranding returns the invoice branding.
func (s *SettingsService) InvoiceBranding(ctx context.Context) (*InvoiceBranding, error) {
	return getData[*InvoiceBranding](ctx, s.client, http.MethodGet, "/settings/invoice-branding", nil)
}

// UpdateInvoiceBranding creates or replaces the invoice branding.
func (s *SettingsService) UpdateInvoiceBranding(ctx context.Context, params *InvoiceBranding) (*InvoiceBranding, error) {
	return getData[*InvoiceBranding](ctx, s.client, http.MethodPut, "/settings/invoice-branding", params)
}

// MCP returns the MCP server settings.
func (s *SettingsService) MCP(ctx context.Context) (*MCPSettings, error) {
	return getData[*MCPSettings](ctx, s.client, http.MethodGet, "/settings/mcp", nil)
}

// UpdateMCP updates the MCP server settings.
func (s *SettingsService) UpdateMCP(ctx context.Context, params *MCPSettings) (*MCPSettings, error) {
	return getData[*MCPSettings](ctx, s.client, http.MethodPut, "/settings/mcp", params)
}
