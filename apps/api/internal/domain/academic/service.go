// Package academic implements academic master data: majors, academic years,
// classes, and the groups that mirror them. HTTP handlers live elsewhere; this
// package only exposes the domain service.
package academic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	permissionManageMajors  = "academic.manage_majors"
	permissionManageYears   = "academic.manage_years"
	permissionManageClasses = "academic.manage_classes"

	groupStatusActive  = "active"
	groupStatusHistory = "history"
)

// Actor is the authenticated principal performing an action, plus the request
// IP recorded in the audit log.
type Actor struct {
	Principal *auth.Principal
	IP        string
}

// Service implements academic master data operations.
type Service struct {
	st  *store.Store
	now func() time.Time
}

// NewService builds an academic Service.
func NewService(st *store.Store) *Service {
	return &Service{st: st, now: time.Now}
}

func (a Actor) canGlobally(permission string) bool {
	return a.Principal != nil && a.Principal.CanGlobally(permission)
}

func (a Actor) canForMajor(permission string, majorID uuid.UUID) bool {
	return a.Principal != nil && a.Principal.CanForMajor(permission, majorID)
}

// recordAudit writes one audit entry through the transaction-bound queries.
func (s *Service) recordAudit(
	ctx context.Context,
	q *store.Queries,
	a Actor,
	action, targetType string,
	targetID uuid.UUID,
	scopeMajorID *uuid.UUID,
	data map[string]any,
) error {
	if a.Principal == nil {
		return apperr.NotFound()
	}
	actorID := a.Principal.UserID
	err := audit.Record(ctx, q, audit.Entry{
		ActorID:      &actorID,
		ActorIP:      a.IP,
		Action:       action,
		TargetType:   targetType,
		TargetID:     &targetID,
		ScopeMajorID: scopeMajorID,
		Data:         data,
	})
	if err != nil {
		return fmt.Errorf("academic: record audit %s: %w", action, err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func dateFromTime(t time.Time) pgtype.Date {
	y, m, d := dateOnly(t).Date()
	return pgtype.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC), Valid: true}
}

func timeFromDate(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	y, m, day := d.Time.Date()
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func normalizeCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeShortName(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeSection(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeName(value string) string {
	return strings.TrimSpace(value)
}
