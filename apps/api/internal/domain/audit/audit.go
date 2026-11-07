// Package audit records administrative audit log entries and reads them back
// with filtering and keyset pagination. Callers are expected to write entries
// through the transaction-bound *store.Queries so the log and the audited
// change commit atomically.
package audit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	// DefaultLimit is used when Filter.Limit is not set.
	DefaultLimit = 50
	// MaxLimit is the largest page List will return.
	MaxLimit = 100
)

// actionPattern mirrors the audit_logs.action CHECK constraint.
var actionPattern = regexp.MustCompile(`^[a-z_]+\.[a-z_.]+$`)

// sensitiveKeys are payload keys that must never reach the audit log. Keys are
// compared case-insensitively at every nesting level.
var sensitiveKeys = map[string]struct{}{
	"password":      {},
	"new_password":  {},
	"token":         {},
	"secret":        {},
	"recovery_code": {},
	"code_hash":     {},
	"password_hash": {},
}

// Entry describes one auditable action.
type Entry struct {
	ActorID      *uuid.UUID // nil for system actions
	ActorIP      string     // empty = unknown
	Action       string     // "<domain>.<verb>", e.g. "user.reset_password"; must match ^[a-z_]+\.[a-z_.]+$
	TargetType   string     // e.g. "user", "content"; optional
	TargetID     *uuid.UUID
	ScopeMajorID *uuid.UUID
	Data         map[string]any // small JSON payload; never put secrets (passwords, tokens, codes) here
}

// Record inserts an audit entry using q, so callers can write it inside the same
// transaction as the action itself (pass the tx-bound *store.Queries).
func Record(ctx context.Context, q *store.Queries, e Entry) error {
	if !actionPattern.MatchString(e.Action) {
		return fmt.Errorf("audit: action %q must match %s", e.Action, actionPattern)
	}
	if err := validateData(e.Data); err != nil {
		return err
	}

	data := e.Data
	if data == nil {
		data = map[string]any{}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("audit: encode data: %w", err)
	}

	var actorIP *netip.Addr
	if e.ActorIP != "" {
		if addr, parseErr := netip.ParseAddr(e.ActorIP); parseErr == nil {
			actorIP = &addr
		}
	}

	var targetType *string
	if e.TargetType != "" {
		targetType = &e.TargetType
	}

	_, err = q.InsertAuditLog(ctx, store.InsertAuditLogParams{
		ActorID:      e.ActorID,
		ActorIp:      actorIP,
		Action:       e.Action,
		TargetType:   targetType,
		TargetID:     e.TargetID,
		ScopeMajorID: e.ScopeMajorID,
		Data:         raw,
	})
	if err != nil {
		return fmt.Errorf("audit: insert entry: %w", err)
	}
	return nil
}

// Filter narrows a List call. Zero values mean "no filter".
type Filter struct {
	ActorID      *uuid.UUID
	Action       string
	ActionPrefix string
	TargetType   string
	TargetID     *uuid.UUID
	ScopeMajorID *uuid.UUID
	From, To     *time.Time
	Cursor       string // opaque, from previous page
	Limit        int    // default 50, max 100
}

// Item is one audit log row as returned by List.
type Item struct {
	ID           int64
	OccurredAt   time.Time
	ActorID      *uuid.UUID
	ActorIP      *string
	Action       string
	TargetType   *string
	TargetID     *uuid.UUID
	ScopeMajorID *uuid.UUID
	Data         map[string]any
}

// Page is one page of audit log entries.
type Page struct {
	Items      []Item
	NextCursor string
}

// List returns audit entries matching f, newest first.
func List(ctx context.Context, q *store.Queries, f Filter) (Page, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	params := store.ListAuditLogsParams{
		ActorID:      f.ActorID,
		Action:       optionalString(f.Action),
		ActionPrefix: optionalString(f.ActionPrefix),
		TargetType:   optionalString(f.TargetType),
		TargetID:     f.TargetID,
		ScopeMajorID: f.ScopeMajorID,
		From:         f.From,
		To:           f.To,
		// Fetch one extra row to detect whether another page exists.
		LimitCount: int32(limit + 1),
	}

	if f.Cursor != "" {
		cursorTime, cursorID, err := decodeCursor(f.Cursor)
		if err != nil {
			return Page{}, apperr.BadRequest("cursor tidak valid")
		}
		params.CursorTime = &cursorTime
		params.CursorID = &cursorID
	}

	rows, err := q.ListAuditLogs(ctx, params)
	if err != nil {
		return Page{}, fmt.Errorf("audit: list entries: %w", err)
	}

	page := Page{Items: make([]Item, 0, min(len(rows), limit))}
	if len(rows) > limit {
		rows = rows[:limit]
		page.NextCursor = encodeCursor(rows[len(rows)-1].OccurredAt, rows[len(rows)-1].ID)
	}
	for _, row := range rows {
		item, err := toItem(row)
		if err != nil {
			return Page{}, err
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func toItem(row store.AuditLog) (Item, error) {
	data := map[string]any{}
	if len(row.Data) > 0 {
		if err := json.Unmarshal(row.Data, &data); err != nil {
			return Item{}, fmt.Errorf("audit: decode data: %w", err)
		}
		if data == nil {
			data = map[string]any{}
		}
	}

	var actorIP *string
	if row.ActorIp != nil {
		value := row.ActorIp.String()
		actorIP = &value
	}

	return Item{
		ID:           row.ID,
		OccurredAt:   row.OccurredAt.UTC(),
		ActorID:      row.ActorID,
		ActorIP:      actorIP,
		Action:       row.Action,
		TargetType:   row.TargetType,
		TargetID:     row.TargetID,
		ScopeMajorID: row.ScopeMajorID,
		Data:         data,
	}, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// validateData rejects payload keys that would leak credentials.
func validateData(data map[string]any) error {
	return walkData(data)
}

func walkData(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if _, blocked := sensitiveKeys[strings.ToLower(key)]; blocked {
				return fmt.Errorf("audit: data key %q is not allowed", key)
			}
			if err := walkData(nested); err != nil {
				return err
			}
		}
	case []any:
		for _, nested := range typed {
			if err := walkData(nested); err != nil {
				return err
			}
		}
	}
	return nil
}

type cursorPayload struct {
	Time string `json:"t"`
	ID   int64  `json:"i"`
}

func encodeCursor(occurredAt time.Time, id int64) string {
	raw, err := json.Marshal(cursorPayload{
		Time: occurredAt.UTC().Format(time.RFC3339Nano),
		ID:   id,
	})
	if err != nil {
		// Marshalling a struct of string and int64 cannot fail.
		panic("audit: encode cursor: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(value string) (time.Time, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, 0, err
	}
	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return time.Time{}, 0, err
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, payload.Time)
	if err != nil {
		return time.Time{}, 0, err
	}
	if payload.ID <= 0 {
		return time.Time{}, 0, fmt.Errorf("audit: cursor id must be positive")
	}
	return occurredAt, payload.ID, nil
}
