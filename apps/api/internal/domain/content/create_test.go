package content

import (
	"regexp"
	"strings"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
)

var publicIDPattern = regexp.MustCompile(`^[0-9a-z]{8}$`)

func TestCreate_PortfolioStartsAsEmptyDraftOwnedByActor(t *testing.T) {
	f := newFixture(t)

	created := f.create(TypePortfolio)

	if !publicIDPattern.MatchString(created.PublicID) {
		t.Fatalf("publicId = %q, mau 8 karakter [0-9a-z]", created.PublicID)
	}
	if created.Type != TypePortfolio || created.Status != StatusDraft {
		t.Errorf("type/status = %s/%s, mau portfolio/draft", created.Type, created.Status)
	}
	if created.Slug != "tanpa-judul" || created.Title != "Tanpa judul" {
		t.Errorf("slug/title = %q/%q, mau tanpa-judul/Tanpa judul", created.Slug, created.Title)
	}
	if created.MyLevel != LevelOwner {
		t.Errorf("myLevel = %s, mau owner", created.MyLevel)
	}
	if created.Draft.LockVersion != 1 || created.Draft.Title != "" {
		t.Errorf("draft = lock %d title %q, mau lock 1 title kosong", created.Draft.LockVersion, created.Draft.Title)
	}
	blocks, ok := created.Draft.Body["blocks"].([]any)
	if !ok || len(blocks) != 0 {
		t.Errorf("body = %#v, mau blocks kosong", created.Draft.Body)
	}
	if created.Draft.Meta["context"] != "personal" {
		t.Errorf("meta.context = %v, mau personal", created.Draft.Meta["context"])
	}
	if links, ok := created.Draft.Meta["links"].([]any); !ok || len(links) != 0 {
		t.Errorf("meta.links = %#v, mau []", created.Draft.Meta["links"])
	}
	if created.HasLiveVersion || created.InReviewClaimed {
		t.Errorf("hasLiveVersion/claimed = %v/%v, mau false/false", created.HasLiveVersion, created.InReviewClaimed)
	}
	if created.AuthorsCount != 1 {
		t.Errorf("authorsCount = %d, mau 1", created.AuthorsCount)
	}

	var level, inviteStatus string
	var position int
	var responded bool
	err := f.pool.QueryRow(f.ctx, `
		SELECT level, invite_status, position, responded_at IS NOT NULL
		FROM content_authors WHERE content_id = $1`, created.ID,
	).Scan(&level, &inviteStatus, &position, &responded)
	if err != nil {
		t.Fatalf("load owner row: %v", err)
	}
	if level != LevelOwner || inviteStatus != "accepted" || position != 0 || !responded {
		t.Errorf("owner row = %s/%s/%d/%v", level, inviteStatus, position, responded)
	}
	if got := countRows(t, f.pool, `SELECT count(*) FROM content_stats WHERE content_id = $1`, created.ID); got != 1 {
		t.Errorf("content_stats rows = %d, mau 1", got)
	}
}

func TestCreate_BlogStartsEmptyWithEmptyMeta(t *testing.T) {
	f := newFixture(t)

	created := f.create(TypeBlog)

	if created.Type != TypeBlog || created.Draft.LockVersion != 1 {
		t.Fatalf("created = %s lock %d, mau blog lock 1", created.Type, created.Draft.LockVersion)
	}
	if nodes, ok := created.Draft.Body["content"].([]any); !ok || len(nodes) != 0 {
		t.Errorf("body = %#v, mau dokumen kosong", created.Draft.Body)
	}
	if len(created.Draft.Meta) != 0 {
		t.Errorf("meta = %#v, mau {}", created.Draft.Meta)
	}
}

func TestCreate_WithoutContentPermissionReturnsContentBlocked(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Create(f.ctx, Actor{
		Principal: &auth.Principal{UserID: f.outsiderID},
	}, CreateInput{Type: TypePortfolio})

	requireAppError(t, err, "CONTENT_BLOCKED")
}

func TestCreate_InvalidTypeReturnsValidation(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Create(f.ctx, f.actor(f.ownerID), CreateInput{Type: "galeri"})

	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	requireFields(t, appErr, "type")
}

func TestCreate_ActiveDraftLimitUsesSettingValue(t *testing.T) {
	f := newFixture(t)
	f.setActiveDraftLimit(2)

	f.create(TypePortfolio)
	f.create(TypeBlog)

	_, err := f.service.Create(f.ctx, f.actor(f.ownerID), CreateInput{Type: TypePortfolio})

	appErr := requireAppError(t, err, "LIMIT_REACHED")
	if !strings.Contains(appErr.Detail, "(2)") {
		t.Errorf("detail = %q, mau menyebut batas 2", appErr.Detail)
	}
}
