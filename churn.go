package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ChurnAlert is raised when a customer's churn score crosses the alert
// threshold.
type ChurnAlert struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	CustomerID    string    `json:"customer_id"`
	PreviousScore int       `json:"previous_score"`
	NewScore      int       `json:"new_score"`
	Threshold     int       `json:"threshold"`
	AlertType     string    `json:"alert_type"`
	Acknowledged  bool      `json:"acknowledged"`
	CreatedAt     time.Time `json:"created_at"`
}

// ChurnService groups the churn-risk endpoints. Per-customer scores are also
// available via CustomersService.Churn.
type ChurnService struct{ client *Client }

// HighRisk lists customers whose churn score is at or above threshold (0-100).
// Pass 0 to use the server default of 70.
func (s *ChurnService) HighRisk(ctx context.Context, threshold int) ([]ChurnScore, error) {
	path := newQuery().int("threshold", threshold).apply("/churn/high-risk")
	return getData[[]ChurnScore](ctx, s.client, http.MethodGet, path, nil)
}

// Alerts lists up to 100 unacknowledged churn alerts, newest first.
func (s *ChurnService) Alerts(ctx context.Context) ([]ChurnAlert, error) {
	return getData[[]ChurnAlert](ctx, s.client, http.MethodGet, "/churn/alerts", nil)
}

// AcknowledgeAlert acknowledges a churn alert.
func (s *ChurnService) AcknowledgeAlert(ctx context.Context, id string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/churn/alerts/%s/ack", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
