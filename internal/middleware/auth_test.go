package middleware

import (
	"net/http"
	"testing"

	"opentab-server/internal/repositories"
)

func TestAuthErrorResponseMapsTokenStates(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "expired", err: repositories.ErrTokenExpired, wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_EXPIRED"},
		{name: "revoked", err: repositories.ErrTokenRevoked, wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_REVOKED"},
		{name: "disabled", err: repositories.ErrUserDisabled, wantStatus: http.StatusForbidden, wantCode: "USER_DISABLED"},
		{name: "unknown", err: repositories.ErrNotFound, wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHORIZED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, code, _ := authErrorResponse(tc.err)
			if status != tc.wantStatus || code != tc.wantCode {
				t.Fatalf("expected %d/%s, got %d/%s", tc.wantStatus, tc.wantCode, status, code)
			}
		})
	}
}
