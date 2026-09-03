package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// MFASetup is the TOTP secret to enroll in an authenticator app.
type MFASetup struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

// MFAStatus is returned after enabling or disabling MFA. BackupCodes are
// only issued on enable and are shown once.
type MFAStatus struct {
	MFAEnabled  bool     `json:"mfa_enabled"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

// Session is one active dashboard login session.
type Session struct {
	ID        string    `json:"id"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Current   bool      `json:"current"`
}

// AuthService groups the per-user security endpoints under /v1/auth: TOTP
// MFA enrollment and login-session management. These act on the calling
// user's dashboard session rather than a tenant API key, so the API accepts
// them with session-cookie authentication only.
type AuthService struct{ client *Client }

// MFASetup begins TOTP enrollment and returns the secret to scan.
func (s *AuthService) MFASetup(ctx context.Context) (*MFASetup, error) {
	var out MFASetup
	if err := s.client.do(ctx, http.MethodPost, "/auth/mfa/setup", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MFAVerify confirms the first TOTP code and enables MFA, returning the
// one-time backup codes.
func (s *AuthService) MFAVerify(ctx context.Context, code string) (*MFAStatus, error) {
	body := map[string]string{"code": code}
	var out MFAStatus
	if err := s.client.do(ctx, http.MethodPost, "/auth/mfa/verify", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MFADisable turns MFA off after verifying a current TOTP code.
func (s *AuthService) MFADisable(ctx context.Context, code string) (*MFAStatus, error) {
	body := map[string]string{"code": code}
	var out MFAStatus
	if err := s.client.do(ctx, http.MethodPost, "/auth/mfa/disable", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Sessions lists the caller's active login sessions.
func (s *AuthService) Sessions(ctx context.Context) ([]Session, error) {
	return getData[[]Session](ctx, s.client, http.MethodGet, "/auth/sessions", nil)
}

// RevokeOtherSessions signs out every session except the current one.
func (s *AuthService) RevokeOtherSessions(ctx context.Context) (*MessageResponse, error) {
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodDelete, "/auth/sessions", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RevokeSession signs out one session by id.
func (s *AuthService) RevokeSession(ctx context.Context, id string) (*MessageResponse, error) {
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/auth/sessions/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
