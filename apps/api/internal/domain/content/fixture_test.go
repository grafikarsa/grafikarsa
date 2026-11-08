package content

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

type fixture struct {
	t       *testing.T
	ctx     context.Context
	pool    *pgxpool.Pool
	store   *store.Store
	service *Service

	ownerID       uuid.UUID
	editorID      uuid.UUID
	contributorID uuid.UUID
	outsiderID    uuid.UUID

	portfolioCategoryID uuid.UUID
	blogCategoryID      uuid.UUID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	st := store.NewStore(pool)
	f := &fixture{
		t:       t,
		ctx:     context.Background(),
		pool:    pool,
		store:   st,
		service: NewService(st),
	}

	f.ownerID = f.insertUser("pemilik", "Pemilik Karya")
	f.editorID = f.insertUser("editor", "Editor Karya")
	f.contributorID = f.insertUser("kontributor", "Kontributor Karya")
	f.outsiderID = f.insertUser("oranglain", "Orang Lain")

	f.portfolioCategoryID = f.categoryID(TypePortfolio, "web-app")
	f.blogCategoryID = f.categoryID(TypeBlog, "tutorial")
	return f
}

func (f *fixture) insertUser(username, displayName string) uuid.UUID {
	f.t.Helper()

	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO users (account_type, display_name, username, password_hash, password_kind, activated_at)
		VALUES ('staff', $1, $2, 'x', 'user', now())
		RETURNING id`,
		displayName, username,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("insert user %s: %v", username, err)
	}
	return id
}

func (f *fixture) categoryID(contentType, slug string) uuid.UUID {
	f.t.Helper()

	var id uuid.UUID
	if err := f.pool.QueryRow(f.ctx, `
		SELECT id FROM content_categories WHERE content_type = $1 AND slug = $2`,
		contentType, slug,
	).Scan(&id); err != nil {
		f.t.Fatalf("load category %s/%s: %v", contentType, slug, err)
	}
	return id
}

func (f *fixture) principal(userID uuid.UUID, grants ...auth.Grant) *auth.Principal {
	return &auth.Principal{
		UserID:      userID,
		DisplayName: "actor",
		AccountType: "staff",
		Status:      "active",
		Grants:      grants,
	}
}

func contentGrants() []auth.Grant {
	return []auth.Grant{{Permission: "content.create", Global: true}}
}

func (f *fixture) actor(userID uuid.UUID) Actor {
	return Actor{Principal: f.principal(userID, contentGrants()...)}
}

func (f *fixture) create(contentType string) StudioContent {
	f.t.Helper()

	created, err := f.service.Create(f.ctx, f.actor(f.ownerID), CreateInput{Type: contentType})
	if err != nil {
		f.t.Fatalf("Create(%s): %v", contentType, err)
	}
	return created
}

func (f *fixture) insertImage(ownerID uuid.UUID) uuid.UUID {
	f.t.Helper()

	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO media_assets (
			owner_id, kind, purpose, original_key, original_filename, mime_type,
			size_bytes, width, height, blurhash, dominant_color
		) VALUES ($1, 'image', 'content', $2, 'gambar.png', 'image/png', 1024, 1200, 675,
			'LEHV6nWB2yk8pyo0adR*.7kCMdnj', '#1e293b')
		RETURNING id`,
		ownerID, "test/"+uuid.NewString()+".png",
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("insert image: %v", err)
	}
	return id
}

func (f *fixture) addAuthor(contentID, userID uuid.UUID, level string) {
	f.t.Helper()

	_, err := f.pool.Exec(f.ctx, `
		INSERT INTO content_authors (content_id, user_id, level, invite_status, position, responded_at)
		VALUES ($1, $2, $3, 'accepted', 1, now())`,
		contentID, userID, level,
	)
	if err != nil {
		f.t.Fatalf("add author %s: %v", level, err)
	}
}

func (f *fixture) addPendingInvite(contentID, userID uuid.UUID, expiresAt time.Time) {
	f.t.Helper()

	_, err := f.pool.Exec(f.ctx, `
		INSERT INTO content_authors (
			content_id, user_id, level, invite_status, invited_at, invite_expires_at
		) VALUES ($1, $2, 'editor', 'pending', now(), $3)`,
		contentID, userID, expiresAt,
	)
	if err != nil {
		f.t.Fatalf("add pending invite: %v", err)
	}
}

func (f *fixture) setStatus(contentID uuid.UUID, status string) {
	f.t.Helper()

	if _, err := f.pool.Exec(f.ctx, `UPDATE contents SET status = $2 WHERE id = $1`, contentID, status); err != nil {
		f.t.Fatalf("set status %s: %v", status, err)
	}
}

// publish inserts an approved version and marks the content as published.
// revision is nil for a normal live content or a revision status when the
// content has an open revision draft.
func (f *fixture) publish(contentID uuid.UUID, revision *string) {
	f.t.Helper()

	var versionID uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO content_versions (
			content_id, number, kind, state, title, slug, body, content_hash,
			submitted_by, decided_at
		) VALUES ($1, 1, 'initial', 'approved', 'Versi tayang', 'versi-tayang',
			'{"blocks":[]}'::jsonb, '\x00'::bytea, $2, now())
		RETURNING id`,
		contentID, f.ownerID,
	).Scan(&versionID)
	if err != nil {
		f.t.Fatalf("insert version: %v", err)
	}

	if _, err := f.pool.Exec(f.ctx, `
		UPDATE contents SET status = 'published', live_version_id = $2, revision_status = $3
		WHERE id = $1`,
		contentID, versionID, revision,
	); err != nil {
		f.t.Fatalf("publish content: %v", err)
	}
}

func (f *fixture) setActiveDraftLimit(limit int) {
	f.t.Helper()

	raw, err := json.Marshal(limit)
	if err != nil {
		f.t.Fatalf("marshal setting: %v", err)
	}
	if _, err := f.store.UpsertSetting(f.ctx, store.UpsertSettingParams{
		Key:   maxActiveDraftsSetting,
		Value: raw,
	}); err != nil {
		f.t.Fatalf("set draft limit: %v", err)
	}
}

func (f *fixture) blockedTag(slug, name string) uuid.UUID {
	f.t.Helper()

	var id uuid.UUID
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO tags (slug, name, is_blocked) VALUES ($1, $2, true) RETURNING id`,
		slug, name,
	).Scan(&id); err != nil {
		f.t.Fatalf("insert blocked tag: %v", err)
	}
	return id
}

func (f *fixture) draftInput(title string) DraftInput {
	return DraftInput{
		LockVersion: 1,
		Title:       title,
		Body:        map[string]any{"blocks": []any{}},
		Meta:        map[string]any{"context": "personal"},
	}
}

func requireAppError(t *testing.T, err error, code string) *apperr.Error {
	t.Helper()

	appErr, ok := apperr.As(err)
	if !ok {
		t.Fatalf("error = %v, mau *apperr.Error dengan kode %s", err, code)
	}
	if appErr.Code != code {
		t.Fatalf("kode error = %s (%s), mau %s", appErr.Code, appErr.Title, code)
	}
	return appErr
}

func hasFieldError(err *apperr.Error, field string) bool {
	for _, fieldErr := range err.Fields {
		if fieldErr.Field == field {
			return true
		}
	}
	return false
}

func fieldErrorMessage(err *apperr.Error, field string) string {
	for _, fieldErr := range err.Fields {
		if fieldErr.Field == field {
			return fieldErr.Message
		}
	}
	return ""
}

func requireFields(t *testing.T, err *apperr.Error, fields ...string) {
	t.Helper()

	for _, field := range fields {
		if !hasFieldError(err, field) {
			t.Fatalf("field %s tidak ada pada error %+v", field, err.Fields)
		}
	}
}

func (f *fixture) contentRow(contentID uuid.UUID) store.Content {
	f.t.Helper()

	row, err := f.store.GetContent(f.ctx, contentID)
	if err != nil {
		f.t.Fatalf("load content: %v", err)
	}
	return row
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	return count
}
