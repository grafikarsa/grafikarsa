package middleware

import (
	"log/slog"
	"net/http"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
)

// Paths every authenticated session may use, whatever its onboarding step.
var onboardingCommonPaths = map[string]struct{}{
	"/api/v1/ping":         {},
	"/api/v1/auth/csrf":    {},
	"/api/v1/auth/session": {},
	"/api/v1/auth/logout":  {},
}

// Extra paths available while the account still needs activation.
var onboardingActivationPaths = map[string]struct{}{
	"/api/v1/auth/username-availability":  {},
	"/api/v1/auth/activate":               {},
	"/api/v1/auth/recovery-codes/confirm": {},
}

// Extra paths available while a temporary password must be replaced.
var onboardingPasswordChangePaths = map[string]struct{}{
	"/api/v1/auth/password/change": {},
}

// Onboarding blocks every endpoint outside the current onboarding step, so a
// half-activated account cannot use the rest of the API.
func Onboarding(_ *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := authctx.Principal(r.Context())
			if principal == nil || principal.NextStep == auth.NextStepNone {
				next.ServeHTTP(w, r)
				return
			}

			if allowedOnboardingPath(principal.NextStep, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			switch principal.NextStep {
			case auth.NextStepChangePassword:
				writeOnboardingProblem(w, r, "PASSWORD_CHANGE_REQUIRED", "Sandi sementara harus diganti")
			default:
				writeOnboardingProblem(w, r, "ACTIVATION_REQUIRED", "Akun perlu diaktivasi")
			}
		})
	}
}

func allowedOnboardingPath(step auth.NextStep, path string) bool {
	if _, ok := onboardingCommonPaths[path]; ok {
		return true
	}

	switch step {
	case auth.NextStepActivate:
		_, ok := onboardingActivationPaths[path]
		return ok
	case auth.NextStepChangePassword:
		_, ok := onboardingPasswordChangePaths[path]
		return ok
	default:
		return false
	}
}

func writeOnboardingProblem(w http.ResponseWriter, r *http.Request, code, title string) {
	problem.Write(w, r, problem.Problem{
		Status: http.StatusForbidden,
		Code:   code,
		Title:  title,
	})
}
