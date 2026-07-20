package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// DunningCampaignStep is one step of a dunning campaign, delivered over
// "email", "sms", or "in_app" after DelayHours. When IsPaymentWall is true the
// step activates the payment wall on the overdue invoice.
type DunningCampaignStep struct {
	ID            string    `json:"id"`
	CampaignID    string    `json:"campaign_id"`
	StepOrder     int       `json:"step_order"`
	Channel       string    `json:"channel"`
	DelayHours    int       `json:"delay_hours"`
	TemplateName  string    `json:"template_name"`
	Subject       string    `json:"subject"`
	Body          string    `json:"body"`
	IsPaymentWall bool      `json:"is_payment_wall"`
	CreatedAt     time.Time `json:"created_at"`
}

// DunningCampaign is a failed-payment recovery campaign triggered by an event
// such as "payment_failed" or "invoice_overdue".
type DunningCampaign struct {
	ID           string                `json:"id"`
	TenantID     string                `json:"tenant_id"`
	Name         string                `json:"name"`
	IsActive     bool                  `json:"is_active"`
	TriggerEvent string                `json:"trigger_event"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	Steps        []DunningCampaignStep `json:"steps"`
}

// DunningCampaignCreateParams is the body for creating a dunning campaign.
type DunningCampaignCreateParams struct {
	Name         string `json:"name"`
	TriggerEvent string `json:"trigger_event"`
}

// DunningCampaignUpdateParams is the body for updating a dunning campaign.
// Omitted fields are left unchanged.
type DunningCampaignUpdateParams struct {
	Name         string `json:"name,omitempty"`
	IsActive     *bool  `json:"is_active,omitempty"`
	TriggerEvent string `json:"trigger_event,omitempty"`
}

// DunningStepCreateParams adds a step to a dunning campaign. Channel is
// "email", "sms", or "in_app".
type DunningStepCreateParams struct {
	StepOrder     int    `json:"step_order"`
	Channel       string `json:"channel"`
	DelayHours    int    `json:"delay_hours,omitempty"`
	TemplateName  string `json:"template_name,omitempty"`
	Subject       string `json:"subject,omitempty"`
	Body          string `json:"body,omitempty"`
	IsPaymentWall bool   `json:"is_payment_wall,omitempty"`
}

// DunningStepUpdateParams is the body for updating a dunning campaign step.
// Omitted fields are left unchanged.
type DunningStepUpdateParams struct {
	StepOrder     int    `json:"step_order,omitempty"`
	Channel       string `json:"channel,omitempty"`
	DelayHours    int    `json:"delay_hours,omitempty"`
	TemplateName  string `json:"template_name,omitempty"`
	Subject       string `json:"subject,omitempty"`
	Body          string `json:"body,omitempty"`
	IsPaymentWall *bool  `json:"is_payment_wall,omitempty"`
}

// DunningCampaignsService groups the dunning-campaign endpoints.
type DunningCampaignsService struct{ client *Client }

// Create creates a dunning campaign.
func (s *DunningCampaignsService) Create(ctx context.Context, params *DunningCampaignCreateParams) (*DunningCampaign, error) {
	var out DunningCampaign
	if err := s.client.do(ctx, http.MethodPost, "/dunning-campaigns", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's dunning campaigns.
func (s *DunningCampaignsService) List(ctx context.Context) ([]DunningCampaign, error) {
	var out []DunningCampaign
	if err := s.client.do(ctx, http.MethodGet, "/dunning-campaigns", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get retrieves a dunning campaign with its steps.
func (s *DunningCampaignsService) Get(ctx context.Context, id string) (*DunningCampaign, error) {
	var out DunningCampaign
	if err := s.client.do(ctx, http.MethodGet, "/dunning-campaigns/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a dunning campaign.
func (s *DunningCampaignsService) Update(ctx context.Context, id string, params *DunningCampaignUpdateParams) (*DunningCampaign, error) {
	var out DunningCampaign
	if err := s.client.do(ctx, http.MethodPut, "/dunning-campaigns/"+id, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddStep adds a step to a dunning campaign.
func (s *DunningCampaignsService) AddStep(ctx context.Context, id string, params *DunningStepCreateParams) (*DunningCampaignStep, error) {
	var out DunningCampaignStep
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/dunning-campaigns/%s/steps", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateStep updates a dunning campaign step.
func (s *DunningCampaignsService) UpdateStep(ctx context.Context, stepID string, params *DunningStepUpdateParams) (*DunningCampaignStep, error) {
	var out DunningCampaignStep
	if err := s.client.do(ctx, http.MethodPut, "/dunning-campaigns/steps/"+stepID, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteStep deletes a dunning campaign step.
func (s *DunningCampaignsService) DeleteStep(ctx context.Context, stepID string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodDelete, "/dunning-campaigns/steps/"+stepID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
