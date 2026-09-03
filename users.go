package recurso

import (
	"context"
	"fmt"
	"net/http"
)

// User is a team member of the tenant. Role is "owner", "admin", or
// "member".
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// UserCreateParams adds a team member with a password.
type UserCreateParams struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

// UserInviteParams invites a team member by email; they set their own
// password from the invitation.
type UserInviteParams struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Role  string `json:"role"`
}

// UsersService groups the team-member endpoints.
type UsersService struct{ client *Client }

// List returns the tenant's team members.
func (s *UsersService) List(ctx context.Context) ([]User, error) {
	return getData[[]User](ctx, s.client, http.MethodGet, "/users", nil)
}

// Create adds a team member.
func (s *UsersService) Create(ctx context.Context, params *UserCreateParams) (*User, error) {
	return getData[*User](ctx, s.client, http.MethodPost, "/users", params)
}

// Invite emails an invitation to join the tenant.
func (s *UsersService) Invite(ctx context.Context, params *UserInviteParams) (*User, error) {
	return getData[*User](ctx, s.client, http.MethodPost, "/users/invite", params)
}

// UpdateRole changes a team member's role.
func (s *UsersService) UpdateRole(ctx context.Context, id, role string) (*User, error) {
	body := map[string]string{"role": role}
	return getData[*User](ctx, s.client, http.MethodPatch, fmt.Sprintf("/users/%s", id), body)
}

// Delete removes a team member.
func (s *UsersService) Delete(ctx context.Context, id string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/users/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
