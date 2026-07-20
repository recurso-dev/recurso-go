package recurso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CancelFlowStep is one step in a cancel flow: "survey", "offer", or
// "confirmation". Config is step-type-specific configuration (survey options,
// offer details, ...).
type CancelFlowStep struct {
	ID        string          `json:"id"`
	FlowID    string          `json:"flow_id"`
	StepOrder int             `json:"step_order"`
	StepType  string          `json:"step_type"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
}

// CancelFlow is a configurable subscription-cancellation retention flow.
type CancelFlow struct {
	ID           string           `json:"id"`
	TenantID     string           `json:"tenant_id"`
	Name         string           `json:"name"`
	IsActive     bool             `json:"is_active"`
	IsDefault    bool             `json:"is_default"`
	CooldownDays int              `json:"cooldown_days"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	Steps        []CancelFlowStep `json:"steps"`
}

// CancelFlowSession is one customer's pass through a cancel flow.
type CancelFlowSession struct {
	ID                 string          `json:"id"`
	TenantID           string          `json:"tenant_id"`
	CustomerID         string          `json:"customer_id"`
	SubscriptionID     string          `json:"subscription_id"`
	FlowID             string          `json:"flow_id"`
	Status             string          `json:"status"`
	CurrentStepIndex   int             `json:"current_step_index"`
	CancellationReason string          `json:"cancellation_reason"`
	Feedback           string          `json:"feedback"`
	OfferPresented     json.RawMessage `json:"offer_presented"`
	OfferAccepted      bool            `json:"offer_accepted"`
	SavedByOffer       bool            `json:"saved_by_offer"`
	StartedAt          time.Time       `json:"started_at"`
	CompletedAt        *time.Time      `json:"completed_at"`
}

// CancelFlowStartResult is returned when a session starts: the session, its
// flow's steps, and the first step to present (nil when the flow has none).
type CancelFlowStartResult struct {
	SessionID string           `json:"session_id"`
	FlowID    string           `json:"flow_id"`
	Steps     []CancelFlowStep `json:"steps"`
	FirstStep *CancelFlowStep  `json:"first_step"`
}

// CancelFlowSubmitResult is returned after submitting a step response. It
// indicates the next step or completion.
type CancelFlowSubmitResult struct {
	SessionID    string          `json:"session_id"`
	Status       string          `json:"status"`
	NextStep     *CancelFlowStep `json:"next_step"`
	SavedByOffer bool            `json:"saved_by_offer"`
	Completed    bool            `json:"completed"`
}

// CancelFlowStats aggregates a flow's session outcomes.
type CancelFlowStats struct {
	TotalSessions   int            `json:"total_sessions"`
	CompletedCount  int            `json:"completed_count"`
	SavedCount      int            `json:"saved_count"`
	SaveRate        float64        `json:"save_rate"`
	ReasonBreakdown map[string]int `json:"reason_breakdown"`
	OfferAcceptRate float64        `json:"offer_accept_rate"`
}

// CancelFlowCreateParams is the body for creating a cancel flow. CooldownDays
// is how long a customer must wait after accepting an offer before re-entering
// the flow (server default 30).
type CancelFlowCreateParams struct {
	Name         string `json:"name"`
	IsDefault    bool   `json:"is_default,omitempty"`
	CooldownDays int    `json:"cooldown_days,omitempty"`
}

// CancelFlowUpdateParams is the body for updating a cancel flow. Omitted
// fields are left unchanged.
type CancelFlowUpdateParams struct {
	Name         string `json:"name,omitempty"`
	IsActive     *bool  `json:"is_active,omitempty"`
	IsDefault    *bool  `json:"is_default,omitempty"`
	CooldownDays int    `json:"cooldown_days,omitempty"`
}

// CancelFlowStepCreateParams adds a step to a cancel flow. StepType is
// "survey", "offer", or "confirmation"; Config defaults to {}.
type CancelFlowStepCreateParams struct {
	StepOrder int             `json:"step_order"`
	StepType  string          `json:"step_type"`
	Config    json.RawMessage `json:"config,omitempty"`
}

// CancelFlowStepUpdateParams is the body for updating a cancel flow step.
// Omitted fields are left unchanged.
type CancelFlowStepUpdateParams struct {
	StepOrder int             `json:"step_order,omitempty"`
	StepType  string          `json:"step_type,omitempty"`
	Config    json.RawMessage `json:"config,omitempty"`
}

// CancelFlowSessionStartParams begins the tenant's default flow for a
// customer/subscription.
type CancelFlowSessionStartParams struct {
	CustomerID     string `json:"customer_id"`
	SubscriptionID string `json:"subscription_id"`
}

// CancelFlowSubmitParams submits a step response in a session. Response is the
// step-type-specific payload (e.g. selected survey reason, offer accepted
// flag).
type CancelFlowSubmitParams struct {
	StepIndex int `json:"step_index,omitempty"`
	Response  any `json:"response"`
}

// CancelFlowsService groups the cancel-flow endpoints.
type CancelFlowsService struct{ client *Client }

// Create creates a cancel flow.
func (s *CancelFlowsService) Create(ctx context.Context, params *CancelFlowCreateParams) (*CancelFlow, error) {
	var out CancelFlow
	if err := s.client.do(ctx, http.MethodPost, "/cancel-flows", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's cancel flows.
func (s *CancelFlowsService) List(ctx context.Context) ([]CancelFlow, error) {
	var out []CancelFlow
	if err := s.client.do(ctx, http.MethodGet, "/cancel-flows", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get retrieves a cancel flow with its steps.
func (s *CancelFlowsService) Get(ctx context.Context, id string) (*CancelFlow, error) {
	var out CancelFlow
	if err := s.client.do(ctx, http.MethodGet, "/cancel-flows/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a cancel flow.
func (s *CancelFlowsService) Update(ctx context.Context, id string, params *CancelFlowUpdateParams) (*CancelFlow, error) {
	var out CancelFlow
	if err := s.client.do(ctx, http.MethodPut, "/cancel-flows/"+id, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddStep adds a step to a cancel flow.
func (s *CancelFlowsService) AddStep(ctx context.Context, id string, params *CancelFlowStepCreateParams) (*CancelFlowStep, error) {
	var out CancelFlowStep
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/cancel-flows/%s/steps", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateStep updates a cancel flow step.
func (s *CancelFlowsService) UpdateStep(ctx context.Context, stepID string, params *CancelFlowStepUpdateParams) (*CancelFlowStep, error) {
	var out CancelFlowStep
	if err := s.client.do(ctx, http.MethodPut, "/cancel-flows/steps/"+stepID, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteStep deletes a cancel flow step.
func (s *CancelFlowsService) DeleteStep(ctx context.Context, stepID string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodDelete, "/cancel-flows/steps/"+stepID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StartSession begins the tenant's default cancel flow for a customer and
// subscription. Rejected while the customer's offer cooldown is active.
func (s *CancelFlowsService) StartSession(ctx context.Context, params *CancelFlowSessionStartParams) (*CancelFlowStartResult, error) {
	var out CancelFlowStartResult
	if err := s.client.do(ctx, http.MethodPost, "/cancel-flows/sessions/start", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSession retrieves a cancel flow session.
func (s *CancelFlowsService) GetSession(ctx context.Context, id string) (*CancelFlowSession, error) {
	var out CancelFlowSession
	if err := s.client.do(ctx, http.MethodGet, "/cancel-flows/sessions/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SubmitStep submits a step response in a session and returns the next step
// or completion.
func (s *CancelFlowsService) SubmitStep(ctx context.Context, sessionID string, params *CancelFlowSubmitParams) (*CancelFlowSubmitResult, error) {
	var out CancelFlowSubmitResult
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/cancel-flows/sessions/%s/submit", sessionID), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Stats returns aggregated statistics for a flow.
func (s *CancelFlowsService) Stats(ctx context.Context, flowID string) (*CancelFlowStats, error) {
	path := newQuery().str("flow_id", flowID).apply("/cancel-flows/stats")
	var out CancelFlowStats
	if err := s.client.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
