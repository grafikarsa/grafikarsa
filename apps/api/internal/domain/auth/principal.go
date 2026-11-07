// Package auth implements authentication, sessions, and effective
// permissions (the Principal) for the API.
package auth

import (
	"bytes"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NextStep tells the client which onboarding flow to show after login.
type NextStep string

const (
	// NextStepActivate means the account still needs a username/password setup.
	NextStepActivate NextStep = "activate"
	// NextStepChangePassword means the user is logged in with an initial or
	// temporary password and must choose a new one.
	NextStepChangePassword NextStep = "change_password"
	// NextStepNone means the user can go straight to the app.
	NextStepNone NextStep = "none"
)

// Grant is one effective permission. A global grant applies everywhere;
// otherwise MajorIDs lists the majors the permission is scoped to.
type Grant struct {
	Permission string
	Global     bool
	MajorIDs   []uuid.UUID
}

// Principal is the authenticated user attached to a session.
type Principal struct {
	SessionID   uuid.UUID
	UserID      uuid.UUID
	Username    *string
	DisplayName string
	AccountType string
	Status      string
	NextStep    NextStep
	IsMigrated  bool
	Grants      []Grant
	ExpiresAt   time.Time
}

// grantRow is one raw explicit grant before merging (a nil MajorID means the
// role assignment is global).
type grantRow struct {
	Permission string
	MajorID    *uuid.UUID
}

// Can reports whether the principal holds permission, globally or scoped.
func (p *Principal) Can(permission string) bool {
	return p.grantFor(permission) != nil
}

// CanGlobally reports whether the principal holds permission without a major
// scope.
func (p *Principal) CanGlobally(permission string) bool {
	grant := p.grantFor(permission)
	return grant != nil && grant.Global
}

// CanForMajor reports whether the principal holds permission globally or
// scoped to majorID.
func (p *Principal) CanForMajor(permission string, majorID uuid.UUID) bool {
	grant := p.grantFor(permission)
	if grant == nil {
		return false
	}
	if grant.Global {
		return true
	}
	return slices.Contains(grant.MajorIDs, majorID)
}

// grantFor returns the grant for permission, assuming Grants is sorted by
// Permission.
func (p *Principal) grantFor(permission string) *Grant {
	i := sort.Search(len(p.Grants), func(i int) bool {
		return p.Grants[i].Permission >= permission
	})
	if i < len(p.Grants) && p.Grants[i].Permission == permission {
		return &p.Grants[i]
	}
	return nil
}

// ComputeNextStep decides the onboarding flow from the account state.
func ComputeNextStep(activatedAt *time.Time, passwordKind string) NextStep {
	switch {
	case activatedAt == nil:
		return NextStepActivate
	case passwordKind == "temporary":
		return NextStepChangePassword
	default:
		return NextStepNone
	}
}

// mergeGrants combines explicit role grants with implicit role permissions.
// Implicit permissions are always global; a permission granted both globally
// and with major scopes collapses into one global grant.
func mergeGrants(explicit []grantRow, implicit []string) []Grant {
	type merged struct {
		global   bool
		majorIDs map[uuid.UUID]struct{}
	}

	byPermission := make(map[string]*merged, len(explicit)+len(implicit))

	entry := func(permission string) *merged {
		m, ok := byPermission[permission]
		if !ok {
			m = &merged{majorIDs: map[uuid.UUID]struct{}{}}
			byPermission[permission] = m
		}
		return m
	}

	for _, permission := range implicit {
		entry(permission).global = true
	}

	for _, row := range explicit {
		m := entry(row.Permission)
		if row.MajorID == nil {
			m.global = true
			continue
		}
		m.majorIDs[*row.MajorID] = struct{}{}
	}

	grants := make([]Grant, 0, len(byPermission))
	for permission, m := range byPermission {
		grant := Grant{Permission: permission, Global: m.global}
		if !m.global {
			grant.MajorIDs = make([]uuid.UUID, 0, len(m.majorIDs))
			for majorID := range m.majorIDs {
				grant.MajorIDs = append(grant.MajorIDs, majorID)
			}
			slices.SortFunc(grant.MajorIDs, func(a, b uuid.UUID) int {
				return bytes.Compare(a[:], b[:])
			})
		}
		grants = append(grants, grant)
	}

	slices.SortFunc(grants, func(a, b Grant) int {
		return strings.Compare(a.Permission, b.Permission)
	})

	return grants
}
