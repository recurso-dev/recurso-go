package recurso

import (
	"context"
	"net/http"
	"time"
)

// InvoiceDispute is a customer-raised dispute against an invoice.
type InvoiceDispute struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	InvoiceID  string     `json:"invoice_id"`
	CustomerID string     `json:"customer_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	Note       string     `json:"note,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// DisputeListParams filters List.
type DisputeListParams struct {
	Status string
	// Limit/Offset page the list; the API clamps at 1000 rows per call.
	Limit  int
	Offset int
}

// DisputeResolveParams carries the optional resolution note.
type DisputeResolveParams struct {
	Note string `json:"note,omitempty"`
}

// DisputesService groups the invoice-dispute endpoints.
type DisputesService struct{ client *Client }

// List returns the tenant's invoice disputes.
func (s *DisputesService) List(ctx context.Context, params *DisputeListParams) ([]InvoiceDispute, error) {
	path := "/disputes"
	if params != nil {
		path = newQuery().
			str("status", params.Status).
			int("limit", params.Limit).
			int("offset", params.Offset).
			apply(path)
	}
	return getData[[]InvoiceDispute](ctx, s.client, http.MethodGet, path, nil)
}

// Resolve marks an open dispute resolved, with an optional note. The API
// returns a bare {"status":"resolved"} ack, so success is just a nil error.
func (s *DisputesService) Resolve(ctx context.Context, id string, params *DisputeResolveParams) error {
	var out struct {
		Status string `json:"status"`
	}
	return s.client.do(ctx, http.MethodPost, "/disputes/"+id+"/resolve", params, &out)
}
