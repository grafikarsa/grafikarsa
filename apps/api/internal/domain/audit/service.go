package audit

import (
	"context"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// Service reads the audit log through a store, applying the caller's
// audit.view scope. Recording entries stays a package-level function so it can
// run inside the transaction that performs the audited change.
type Service struct {
	st *store.Store
}

// NewService builds an audit Service.
func NewService(st *store.Store) *Service {
	return &Service{st: st}
}

// ListForActor returns the audit entries visible through scope.
func (s *Service) ListForActor(ctx context.Context, scope Scope, f Filter) (Page, error) {
	return ListForActor(ctx, s.st.Queries, scope, f)
}
