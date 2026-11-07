package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
)

func onboardingTestHandler() http.Handler {
	return Onboarding(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

func onboardingRequest(method, path string, principal *auth.Principal) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if principal != nil {
		req = req.WithContext(authctx.WithPrincipal(req.Context(), principal))
	}
	return req
}

func TestOnboardingAnonymousPasses(t *testing.T) {
	rec := httptest.NewRecorder()
	onboardingTestHandler().ServeHTTP(rec, onboardingRequest(http.MethodGet, "/api/v1/feed", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestOnboardingAllowsPathsByStep(t *testing.T) {
	common := []string{
		"/api/v1/ping",
		"/api/v1/auth/csrf",
		"/api/v1/auth/session",
		"/api/v1/auth/logout",
	}

	tests := []struct {
		name  string
		step  auth.NextStep
		paths []string
	}{
		{
			name: "activate",
			step: auth.NextStepActivate,
			paths: append(append([]string{}, common...),
				"/api/v1/auth/username-availability",
				"/api/v1/auth/activate",
				"/api/v1/auth/recovery-codes/confirm",
			),
		},
		{
			name: "change password",
			step: auth.NextStepChangePassword,
			paths: append(append([]string{}, common...),
				"/api/v1/auth/password/change",
			),
		},
		{
			name:  "none",
			step:  auth.NextStepNone,
			paths: []string{"/api/v1/feed", "/api/v1/auth/password/change", "/api/v1/auth/activate"},
		},
	}

	for _, tt := range tests {
		for _, path := range tt.paths {
			t.Run(tt.name+" "+path, func(t *testing.T) {
				rec := httptest.NewRecorder()
				onboardingTestHandler().
					ServeHTTP(rec, onboardingRequest(http.MethodGet, path, &auth.Principal{NextStep: tt.step}))

				if rec.Code != http.StatusNoContent {
					t.Fatalf("status = %d, want 204", rec.Code)
				}
			})
		}
	}
}

func TestOnboardingBlocksPathsByStep(t *testing.T) {
	tests := []struct {
		name     string
		step     auth.NextStep
		path     string
		wantCode string
		wantText string
	}{
		{
			name:     "activate blocks feed",
			step:     auth.NextStepActivate,
			path:     "/api/v1/feed",
			wantCode: "ACTIVATION_REQUIRED",
			wantText: "Akun perlu diaktivasi",
		},
		{
			name:     "activate blocks password change",
			step:     auth.NextStepActivate,
			path:     "/api/v1/auth/password/change",
			wantCode: "ACTIVATION_REQUIRED",
			wantText: "Akun perlu diaktivasi",
		},
		{
			name:     "change password blocks feed",
			step:     auth.NextStepChangePassword,
			path:     "/api/v1/feed",
			wantCode: "PASSWORD_CHANGE_REQUIRED",
			wantText: "Sandi sementara harus diganti",
		},
		{
			name:     "change password blocks activation",
			step:     auth.NextStepChangePassword,
			path:     "/api/v1/auth/activate",
			wantCode: "PASSWORD_CHANGE_REQUIRED",
			wantText: "Sandi sementara harus diganti",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			onboardingTestHandler().
				ServeHTTP(rec, onboardingRequest(http.MethodGet, tt.path, &auth.Principal{NextStep: tt.step}))

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", res.StatusCode)
			}
			if got := res.Header.Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", got)
			}

			var body problem.Problem
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Code, tt.wantCode)
			}
			if body.Title != tt.wantText {
				t.Errorf("title = %q, want %q", body.Title, tt.wantText)
			}
		})
	}
}
