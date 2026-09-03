package recurso

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// GSTR1B2BInvoice is one invoice to a GST-registered buyer. Amounts are in
// paise; Rate is the combined GST rate in percent.
type GSTR1B2BInvoice struct {
	InvoiceNumber string    `json:"invoice_number"`
	Date          time.Time `json:"date"`
	PlaceOfSupply string    `json:"place_of_supply"`
	TaxableValue  int64     `json:"taxable_value"`
	IGST          int64     `json:"igst"`
	CGST          int64     `json:"cgst"`
	SGST          int64     `json:"sgst"`
	Rate          float64   `json:"rate"`
}

// GSTR1B2B groups a registered buyer's invoices under its GSTIN.
type GSTR1B2B struct {
	GSTIN    string            `json:"gstin"`
	Invoices []GSTR1B2BInvoice `json:"invoices"`
}

// GSTR1B2CS is a small-value unregistered-buyer summary line per place of
// supply and rate.
type GSTR1B2CS struct {
	PlaceOfSupply string  `json:"place_of_supply"`
	Rate          float64 `json:"rate"`
	TaxableValue  int64   `json:"taxable_value"`
	IGST          int64   `json:"igst"`
	CGST          int64   `json:"cgst"`
	SGST          int64   `json:"sgst"`
}

// GSTR1CDNRNote is one credit note issued to a registered buyer.
type GSTR1CDNRNote struct {
	NoteNumber            string    `json:"note_number"`
	Date                  time.Time `json:"date"`
	OriginalInvoiceNumber string    `json:"original_invoice_number"`
	PlaceOfSupply         string    `json:"place_of_supply"`
	TaxableValue          int64     `json:"taxable_value"`
	IGST                  int64     `json:"igst"`
	CGST                  int64     `json:"cgst"`
	SGST                  int64     `json:"sgst"`
	Rate                  float64   `json:"rate"`
}

// GSTR1CDNR groups a registered buyer's credit notes under its GSTIN.
type GSTR1CDNR struct {
	GSTIN string          `json:"gstin"`
	Notes []GSTR1CDNRNote `json:"notes"`
}

// GSTR1HSNSummary is the HSN/SAC-wise summary of outward supplies.
type GSTR1HSNSummary struct {
	HSNCode      string `json:"hsn_code"`
	TaxableValue int64  `json:"taxable_value"`
	IGST         int64  `json:"igst"`
	CGST         int64  `json:"cgst"`
	SGST         int64  `json:"sgst"`
	InvoiceCount int    `json:"invoice_count"`
}

// GSTR1Return is the GSTR-1 outward-supply return for a tax period.
type GSTR1Return struct {
	TenantID                string            `json:"tenant_id"`
	Month                   int               `json:"month"`
	Year                    int               `json:"year"`
	B2B                     []GSTR1B2B        `json:"b2b"`
	B2CS                    []GSTR1B2CS       `json:"b2cs"`
	CDNR                    []GSTR1CDNR       `json:"cdnr"`
	HSNSummary              []GSTR1HSNSummary `json:"hsn_summary"`
	TotalTaxableValue       int64             `json:"total_taxable_value"`
	TotalIGST               int64             `json:"total_igst"`
	TotalCGST               int64             `json:"total_cgst"`
	TotalSGST               int64             `json:"total_sgst"`
	InvoiceCount            int               `json:"invoice_count"`
	TotalCreditTaxableValue int64             `json:"total_credit_taxable_value"`
	TotalCreditIGST         int64             `json:"total_credit_igst"`
	TotalCreditCGST         int64             `json:"total_credit_cgst"`
	TotalCreditSGST         int64             `json:"total_credit_sgst"`
	CreditNoteCount         int               `json:"credit_note_count"`
}

// GSTR3BValues is one GSTR-3B table 3.1 row.
type GSTR3BValues struct {
	TaxableValue int64 `json:"taxable_value"`
	IGST         int64 `json:"igst"`
	CGST         int64 `json:"cgst"`
	SGST         int64 `json:"sgst"`
}

// GSTR3BInterStateUnreg is one GSTR-3B table 3.2 row (inter-state supplies
// to unregistered persons).
type GSTR3BInterStateUnreg struct {
	PlaceOfSupply string `json:"place_of_supply"`
	TaxableValue  int64  `json:"taxable_value"`
	IGST          int64  `json:"igst"`
}

// GSTR3BReturn is the GSTR-3B summary return for a tax period.
type GSTR3BReturn struct {
	TenantID               string                  `json:"tenant_id"`
	Month                  int                     `json:"month"`
	Year                   int                     `json:"year"`
	OutwardTaxable         GSTR3BValues            `json:"outward_taxable"`
	ZeroRated              GSTR3BValues            `json:"zero_rated"`
	NilExempt              GSTR3BValues            `json:"nil_exempt"`
	InwardReverseCharge    GSTR3BValues            `json:"inward_reverse_charge"`
	NonGST                 GSTR3BValues            `json:"non_gst"`
	InterStateUnregistered []GSTR3BInterStateUnreg `json:"inter_state_unregistered"`
	InvoiceCount           int                     `json:"invoice_count"`
	CreditNoteCount        int                     `json:"credit_note_count"`
}

// GSTR1Result pairs the readable GSTR-1 return with the GSTN upload JSON.
type GSTR1Result struct {
	Data      GSTR1Return     `json:"data"`
	GovSchema json.RawMessage `json:"gov_schema"`
}

// GSTR3BResult pairs the readable GSTR-3B return with the GSTN upload JSON.
type GSTR3BResult struct {
	Data      GSTR3BReturn    `json:"data"`
	GovSchema json.RawMessage `json:"gov_schema"`
}

// GSTReturnParams selects the tax period. EntityID scopes to one legal
// entity; empty = the whole workspace.
type GSTReturnParams struct {
	Month    int
	Year     int
	EntityID string
}

// IndiaService groups the Indian GST return endpoints.
type IndiaService struct{ client *Client }

func gstReturnPath(base string, params *GSTReturnParams) string {
	if params == nil {
		return base
	}
	return newQuery().int("month", params.Month).int("year", params.Year).str("entity_id", params.EntityID).apply(base)
}

// GSTR1 returns the GSTR-1 outward-supply return for the period.
func (s *IndiaService) GSTR1(ctx context.Context, params *GSTReturnParams) (*GSTR1Result, error) {
	var out GSTR1Result
	if err := s.client.do(ctx, http.MethodGet, gstReturnPath("/india/gstr1", params), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GSTR3B returns the GSTR-3B summary return for the period.
func (s *IndiaService) GSTR3B(ctx context.Context, params *GSTReturnParams) (*GSTR3BResult, error) {
	var out GSTR3BResult
	if err := s.client.do(ctx, http.MethodGet, gstReturnPath("/india/gstr3b", params), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
