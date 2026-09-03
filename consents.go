package recurso

import (
	"context"
	"net/http"
)

// ConsentRecordParams records a customer's consent. ConsentType is one of
// recurring_billing, email_marketing, data_processing, terms_of_service, or
// privacy_policy.
type ConsentRecordParams struct {
	CustomerID     string `json:"customer_id"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	ConsentType    string `json:"consent_type"`
	Granted        bool   `json:"granted"`
	ConsentText    string `json:"consent_text,omitempty"`
}

// ConsentsService groups the consent-record endpoints (RBI compliance).
// Per-customer listing is CustomersService.Consents; the consent tied to a
// subscription is SubscriptionsService.Consent.
type ConsentsService struct{ client *Client }

// Record stores a consent record.
func (s *ConsentsService) Record(ctx context.Context, params *ConsentRecordParams) (*Consent, error) {
	var out Consent
	if err := s.client.do(ctx, http.MethodPost, "/consents", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Revoke revokes a consent record by id.
func (s *ConsentsService) Revoke(ctx context.Context, consentID string) (*MessageResponse, error) {
	body := map[string]string{"consent_id": consentID}
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodPost, "/consents/revoke", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
