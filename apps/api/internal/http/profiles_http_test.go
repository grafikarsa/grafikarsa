package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/profiles"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

const profilesTestPassword = "sandi-profil-123"

type profilesEnv struct {
	*flowEnv
}

// newProfilesEnv builds the real router with auth and profile services on a
// seeded database, served over TLS so __Host- cookies are accepted.
func newProfilesEnv(t *testing.T) *profilesEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	rdb := testutil.NewValkey(t)
	hasher := password.NewHasher(cheapPasswordParams, 0)
	logger := testLogger()

	st := store.NewStore(pool)
	authService, err := auth.NewService(
		st,
		rdb,
		hasher,
		ratelimit.New(rdb),
		auth.Config{RecoverySecret: testRecoverySecret},
		logger,
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	handler := NewRouter(RouterDeps{
		Logger:         logger,
		Auth:           authService,
		Profiles:       profiles.NewService(st, authService),
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	return &profilesEnv{flowEnv: &flowEnv{t: t, pool: pool, server: server, hasher: hasher}}
}

// signIn logs in and returns a client plus its current CSRF token.
func (e *profilesEnv) signIn(identifier, plain string) (*http.Client, string) {
	e.t.Helper()

	client := e.client()
	csrf := e.fetchCSRF(client)
	res, body := e.login(client, identifier, plain, csrf)
	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("login %s status = %d, body %s", identifier, res.StatusCode, body)
	}
	return client, e.csrfFromJar(client)
}

// anonymousClient does not follow redirects so 301 responses stay observable.
func (e *profilesEnv) anonymousClient() *http.Client {
	e.t.Helper()

	return &http.Client{
		Transport: e.server.Client().Transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// createProfileUser inserts an activated user with a known password.
func (e *profilesEnv) createProfileUser(username, displayName string) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	return e.createUser(flowUserSpec{
		accountType: "staff",
		username:    &username,
		displayName: displayName,
		password:    profilesTestPassword,
		activatedAt: &now,
	})
}

func (e *profilesEnv) requirePublicProfile(res *http.Response, body []byte) apigen.PublicProfile {
	e.t.Helper()

	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("status = %d, want 200 (body %s)", res.StatusCode, body)
	}
	var profile apigen.PublicProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		e.t.Fatalf("decode profile: %v (body %s)", err, body)
	}
	return profile
}

func (e *profilesEnv) requireMyProfile(res *http.Response, body []byte) apigen.MyProfile {
	e.t.Helper()

	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("status = %d, want 200 (body %s)", res.StatusCode, body)
	}
	var profile apigen.MyProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		e.t.Fatalf("decode profile: %v (body %s)", err, body)
	}
	return profile
}

func (e *profilesEnv) createMedia(ownerID uuid.UUID, kind, purpose string) uuid.UUID {
	e.t.Helper()

	var id uuid.UUID
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO media_assets (owner_id, kind, purpose, original_key, mime_type, size_bytes)
		VALUES ($1, $2, $3, $4, 'image/png', 1024)
		RETURNING id`,
		ownerID, kind, purpose, fmt.Sprintf("test/%s.png", uuid.New()),
	).Scan(&id)
	if err != nil {
		e.t.Fatalf("create media: %v", err)
	}
	return id
}

func TestGetUser_PublicProfileHidesNisn(t *testing.T) {
	env := newProfilesEnv(t)
	nisn := "1234567890"
	now := time.Now().UTC()
	env.createUser(flowUserSpec{
		nisn:        &nisn,
		username:    ptrString("siswa.rahasia"),
		displayName: "Siswa Rahasia",
		password:    profilesTestPassword,
		activatedAt: &now,
	})

	res, body := env.do(env.anonymousClient(), http.MethodGet, "/api/v1/users/siswa.rahasia", nil, "", nil)
	profile := env.requirePublicProfile(res, body)

	if profile.Username != "siswa.rahasia" || profile.DisplayName != "Siswa Rahasia" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
	if profile.AccountType != apigen.AccountTypeStudent || profile.Status != apigen.ProfileStatusActive {
		t.Fatalf("unexpected identity: %+v", profile)
	}
	if strings.Contains(string(body), "nisn") || strings.Contains(string(body), nisn) {
		t.Fatalf("public profile leaks NISN: %s", body)
	}
	if len(profile.Links) != 0 || len(profile.Skills) != 0 || len(profile.Badges) != 0 {
		t.Fatalf("unexpected related data: %+v", profile)
	}
}

func TestGetUser_RenamedUsernameRedirects(t *testing.T) {
	env := newProfilesEnv(t)
	env.createProfileUser("nama.lama", "Nama Lama")
	client, csrf := env.signIn("nama.lama", profilesTestPassword)

	res, body := env.do(client, http.MethodPatch, "/api/v1/me/username",
		map[string]string{"username": "nama.baru"}, csrf, nil)
	env.requireMyProfile(res, body)

	res, body = env.do(env.anonymousClient(), http.MethodGet, "/api/v1/users/nama.lama", nil, "", nil)
	if res.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301 (body %s)", res.StatusCode, body)
	}
	if location := res.Header.Get("Location"); location != "/@nama.baru" {
		t.Fatalf("Location = %q, want /@nama.baru", location)
	}

	res, body = env.do(env.anonymousClient(), http.MethodGet, "/api/v1/users/nama.baru", nil, "", nil)
	profile := env.requirePublicProfile(res, body)
	if profile.Username != "nama.baru" {
		t.Fatalf("username = %q, want nama.baru", profile.Username)
	}
}

func TestChangeUsername_SecondChangeWithinThirtyDaysIsLimited(t *testing.T) {
	env := newProfilesEnv(t)
	env.createProfileUser("ganti.sekali", "Ganti Sekali")
	client, csrf := env.signIn("ganti.sekali", profilesTestPassword)

	res, body := env.do(client, http.MethodPatch, "/api/v1/me/username",
		map[string]string{"username": "ganti.dua"}, csrf, nil)
	env.requireMyProfile(res, body)

	res, body = env.do(client, http.MethodPatch, "/api/v1/me/username",
		map[string]string{"username": "ganti.tiga"}, csrf, nil)
	requireProblem(t, res, body, http.StatusConflict, "LIMIT_REACHED")
}

func TestGetUser_InactiveAndUnactivatedAreNotFound(t *testing.T) {
	env := newProfilesEnv(t)
	inactiveID := env.createProfileUser("sudah.nonaktif", "Nonaktif")
	if _, err := env.pool.Exec(context.Background(),
		`UPDATE users SET status = 'inactive' WHERE id = $1`, inactiveID); err != nil {
		t.Fatalf("deactivate user: %v", err)
	}
	env.createUser(flowUserSpec{
		accountType: "staff",
		username:    ptrString("belum.aktivasi"),
		displayName: "Belum Aktivasi",
		password:    profilesTestPassword,
	})

	for _, username := range []string{"sudah.nonaktif", "belum.aktivasi"} {
		res, body := env.do(env.anonymousClient(), http.MethodGet, "/api/v1/users/"+username, nil, "", nil)
		requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
	}
}

func TestFollow_IdempotentAndUpdatesCounts(t *testing.T) {
	env := newProfilesEnv(t)
	aID := env.createProfileUser("pengikut.a", "Pengikut A")
	env.createProfileUser("diikuti.b", "Diikuti B")
	client, csrf := env.signIn("pengikut.a", profilesTestPassword)

	for i := 0; i < 2; i++ {
		res, body := env.do(client, http.MethodPut, "/api/v1/users/diikuti.b/follow", nil, csrf, nil)
		if res.StatusCode != http.StatusNoContent {
			t.Fatalf("follow #%d status = %d, body %s", i+1, res.StatusCode, body)
		}
	}

	res, body := env.do(env.anonymousClient(), http.MethodGet, "/api/v1/users/diikuti.b", nil, "", nil)
	profile := env.requirePublicProfile(res, body)
	if profile.Stats.Followers != 1 || profile.Viewer.Following {
		t.Fatalf("anonymous view = %+v", profile)
	}

	res, body = env.do(client, http.MethodGet, "/api/v1/users/diikuti.b", nil, "", nil)
	profile = env.requirePublicProfile(res, body)
	if profile.Stats.Followers != 1 || !profile.Viewer.Following || profile.Viewer.IsSelf {
		t.Fatalf("follower view = %+v", profile)
	}

	res, body = env.do(env.anonymousClient(), http.MethodGet, "/api/v1/users/diikuti.b/followers", nil, "", nil)
	var page apigen.UserMiniPage
	if res.StatusCode != http.StatusOK {
		t.Fatalf("followers status = %d, body %s", res.StatusCode, body)
	}
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("decode followers: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Username != "pengikut.a" {
		t.Fatalf("followers = %+v", page.Items)
	}

	for i := 0; i < 2; i++ {
		res, body = env.do(client, http.MethodDelete, "/api/v1/users/diikuti.b/follow", nil, csrf, nil)
		if res.StatusCode != http.StatusNoContent {
			t.Fatalf("unfollow #%d status = %d, body %s", i+1, res.StatusCode, body)
		}
	}

	res, body = env.do(env.anonymousClient(), http.MethodGet, "/api/v1/users/diikuti.b", nil, "", nil)
	profile = env.requirePublicProfile(res, body)
	if profile.Stats.Followers != 0 {
		t.Fatalf("followers after unfollow = %d, want 0", profile.Stats.Followers)
	}
	_ = aID
}

func TestFollow_SelfIsValidationError(t *testing.T) {
	env := newProfilesEnv(t)
	env.createProfileUser("diri.sendiri", "Diri Sendiri")
	client, csrf := env.signIn("diri.sendiri", profilesTestPassword)

	res, body := env.do(client, http.MethodPut, "/api/v1/users/diri.sendiri/follow", nil, csrf, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
}

func TestUpdateProfile_ValidatesAndClearsFields(t *testing.T) {
	env := newProfilesEnv(t)
	env.createProfileUser("ubah.profil", "Ubah Profil")
	client, csrf := env.signIn("ubah.profil", profilesTestPassword)

	res, body := env.do(client, http.MethodPatch, "/api/v1/me/profile",
		map[string]string{"bio": strings.Repeat("a", 1001)}, csrf, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	res, body = env.do(client, http.MethodPatch, "/api/v1/me/profile", map[string]any{
		"displayName": "  Nama Baru  ",
		"headline":    "  ",
		"city":        "  Malang  ",
	}, csrf, nil)
	profile := env.requireMyProfile(res, body)
	if profile.DisplayName != "Nama Baru" {
		t.Fatalf("displayName = %q, want trimmed value", profile.DisplayName)
	}
	if profile.Headline != nil {
		t.Fatalf("headline = %v, want nil", *profile.Headline)
	}
	if profile.City == nil || *profile.City != "Malang" {
		t.Fatalf("city = %v, want Malang", profile.City)
	}
}

func TestReplaceLinks_ValidatesLimitAndHTTPS(t *testing.T) {
	env := newProfilesEnv(t)
	env.createProfileUser("tautan.profil", "Tautan Profil")
	client, csrf := env.signIn("tautan.profil", profilesTestPassword)

	items := make([]map[string]string, 0, 11)
	for i := 0; i < 11; i++ {
		items = append(items, map[string]string{
			"platform": "website",
			"url":      fmt.Sprintf("https://example.com/%d", i),
		})
	}
	res, body := env.do(client, http.MethodPut, "/api/v1/me/links", map[string]any{"items": items}, csrf, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	res, body = env.do(client, http.MethodPut, "/api/v1/me/links", map[string]any{
		"items": []map[string]string{{"platform": "github", "url": "http://example.com"}},
	}, csrf, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	res, body = env.do(client, http.MethodPut, "/api/v1/me/links", map[string]any{
		"items": []map[string]string{
			{"platform": "github", "url": "https://github.com/grafikarsa"},
			{"platform": "website", "url": "https://grafikarsa.com"},
		},
	}, csrf, nil)
	profile := env.requireMyProfile(res, body)
	if len(profile.Links) != 2 || profile.Links[0].Platform != "github" || profile.Links[1].Platform != "website" {
		t.Fatalf("links = %+v", profile.Links)
	}
}

func TestSkills_ReplaceAndEndorse(t *testing.T) {
	env := newProfilesEnv(t)
	env.createProfileUser("punya.skill", "Punya Skill")
	env.createProfileUser("pengendorse", "Pengendorse")

	ownerClient, ownerCSRF := env.signIn("punya.skill", profilesTestPassword)
	res, body := env.do(ownerClient, http.MethodPut, "/api/v1/me/skills", map[string]any{
		"skills": []string{"desain-grafis", "tipografi"},
	}, ownerCSRF, nil)
	profile := env.requireMyProfile(res, body)
	if len(profile.Skills) != 2 || profile.Skills[0].Slug != "desain-grafis" || profile.Skills[0].EndorsementCount != 0 {
		t.Fatalf("skills = %+v", profile.Skills)
	}

	endorserClient, endorserCSRF := env.signIn("pengendorse", profilesTestPassword)
	for i := 0; i < 2; i++ {
		res, body = env.do(endorserClient, http.MethodPut,
			"/api/v1/users/punya.skill/skills/desain-grafis/endorsement", nil, endorserCSRF, nil)
		if res.StatusCode != http.StatusNoContent {
			t.Fatalf("endorse #%d status = %d, body %s", i+1, res.StatusCode, body)
		}
	}

	res, body = env.do(endorserClient, http.MethodGet, "/api/v1/users/punya.skill", nil, "", nil)
	public := env.requirePublicProfile(res, body)
	if len(public.Skills) != 2 {
		t.Fatalf("public skills = %+v", public.Skills)
	}
	if public.Skills[0].EndorsementCount != 1 || !public.Skills[0].EndorsedByViewer {
		t.Fatalf("endorsement state = %+v", public.Skills[0])
	}

	res, body = env.do(env.anonymousClient(), http.MethodGet, "/api/v1/users/punya.skill", nil, "", nil)
	public = env.requirePublicProfile(res, body)
	if public.Skills[0].EndorsementCount != 1 || public.Skills[0].EndorsedByViewer {
		t.Fatalf("anonymous endorsement state = %+v", public.Skills[0])
	}

	res, body = env.do(ownerClient, http.MethodPut,
		"/api/v1/users/punya.skill/skills/desain-grafis/endorsement", nil, ownerCSRF, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	res, body = env.do(endorserClient, http.MethodDelete,
		"/api/v1/users/punya.skill/skills/desain-grafis/endorsement", nil, endorserCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("unendorse status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(endorserClient, http.MethodGet, "/api/v1/users/punya.skill", nil, "", nil)
	public = env.requirePublicProfile(res, body)
	if public.Skills[0].EndorsementCount != 0 || public.Skills[0].EndorsedByViewer {
		t.Fatalf("endorsement after removal = %+v", public.Skills[0])
	}
}

func TestExperiences_CRUDAndSystemProtection(t *testing.T) {
	env := newProfilesEnv(t)
	ownerID := env.createProfileUser("punya.karier", "Punya Karier")
	env.createProfileUser("pengintip.karier", "Pengintip Karier")

	var systemID uuid.UUID
	if err := env.pool.QueryRow(context.Background(), `
		INSERT INTO experiences (user_id, kind, title, organization, start_month, is_current, is_system)
		VALUES ($1, 'education', 'Siswa RPL', 'SMK Negeri 4 Malang', '2024-07-01', true, true)
		RETURNING id`, ownerID).Scan(&systemID); err != nil {
		t.Fatalf("insert system experience: %v", err)
	}

	client, csrf := env.signIn("punya.karier", profilesTestPassword)

	res, body := env.do(client, http.MethodPatch, "/api/v1/me/experiences/"+systemID.String(),
		map[string]any{
			"kind": "education", "title": "Diubah", "organization": "Sekolah",
			"startMonth": "2024-07", "isCurrent": true,
		}, csrf, nil)
	requireProblem(t, res, body, http.StatusConflict, "CONFLICT")

	res, body = env.do(client, http.MethodDelete, "/api/v1/me/experiences/"+systemID.String(), nil, csrf, nil)
	requireProblem(t, res, body, http.StatusConflict, "CONFLICT")

	res, body = env.do(client, http.MethodPost, "/api/v1/me/experiences", map[string]any{
		"kind": "job", "title": "Desainer", "organization": "Studio Kreatif",
		"startMonth": "2024-01", "isCurrent": true, "skills": []string{"desain-grafis"},
	}, csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body %s", res.StatusCode, body)
	}
	var created apigen.Experience
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode experience: %v", err)
	}
	if !created.IsCurrent || created.EndMonth != nil || len(created.Skills) != 1 {
		t.Fatalf("created experience = %+v", created)
	}

	res, body = env.do(client, http.MethodPatch, "/api/v1/me/experiences/"+created.Id.String(), map[string]any{
		"kind": "job", "title": "Desainer", "organization": "Studio Kreatif",
		"startMonth": "2024-01", "isCurrent": false,
	}, csrf, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	res, body = env.do(client, http.MethodPatch, "/api/v1/me/experiences/"+created.Id.String(), map[string]any{
		"kind": "job", "title": "Desainer", "organization": "Studio Kreatif",
		"startMonth": "2024-06", "endMonth": "2024-01", "isCurrent": false,
	}, csrf, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	res, body = env.do(client, http.MethodPatch, "/api/v1/me/experiences/"+created.Id.String(), map[string]any{
		"kind": "job", "title": "Desainer", "organization": "Studio Kreatif",
		"startMonth": "2024-01", "endMonth": "2024-12", "isCurrent": false,
	}, csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, body %s", res.StatusCode, body)
	}
	var updated apigen.Experience
	if err := json.Unmarshal(body, &updated); err != nil {
		t.Fatalf("decode experience: %v", err)
	}
	if updated.IsCurrent || updated.EndMonth == nil || *updated.EndMonth != "2024-12" {
		t.Fatalf("updated experience = %+v", updated)
	}

	res, body = env.do(env.anonymousClient(), http.MethodGet,
		"/api/v1/users/punya.karier/experiences", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("public experiences status = %d, body %s", res.StatusCode, body)
	}
	var list apigen.ExperienceList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode experiences: %v", err)
	}
	if len(list.Items) != 2 || !list.Items[0].IsSystem {
		t.Fatalf("public experiences = %+v", list.Items)
	}

	intruderClient, intruderCSRF := env.signIn("pengintip.karier", profilesTestPassword)
	res, body = env.do(intruderClient, http.MethodPatch, "/api/v1/me/experiences/"+created.Id.String(),
		map[string]any{
			"kind": "job", "title": "Diambil Alih", "organization": "Lain",
			"startMonth": "2024-01", "isCurrent": true,
		}, intruderCSRF, nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(client, http.MethodDelete, "/api/v1/me/experiences/"+created.Id.String(), nil, csrf, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, body %s", res.StatusCode, body)
	}
}

func TestSetAvatar_RejectsForeignMedia(t *testing.T) {
	env := newProfilesEnv(t)
	ownerID := env.createProfileUser("punya.avatar", "Punya Avatar")
	otherID := env.createProfileUser("punya.media.lain", "Punya Media Lain")
	env.createProfileUser("pengunggah", "Pengunggah")

	foreignAvatar := env.createMedia(otherID, "image", "avatar")
	ownBanner := env.createMedia(ownerID, "image", "banner")
	ownAvatar := env.createMedia(ownerID, "image", "avatar")

	client, csrf := env.signIn("punya.avatar", profilesTestPassword)

	res, body := env.do(client, http.MethodPut, "/api/v1/me/avatar",
		map[string]any{"mediaId": foreignAvatar.String()}, csrf, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	res, body = env.do(client, http.MethodPut, "/api/v1/me/avatar",
		map[string]any{"mediaId": ownBanner.String()}, csrf, nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	res, body = env.do(client, http.MethodPut, "/api/v1/me/avatar",
		map[string]any{"mediaId": ownAvatar.String()}, csrf, nil)
	profile := env.requireMyProfile(res, body)
	if profile.AvatarMediaId == nil || uuid.UUID(*profile.AvatarMediaId) != ownAvatar {
		t.Fatalf("avatarMediaId = %v, want %s", profile.AvatarMediaId, ownAvatar)
	}

	res, body = env.do(client, http.MethodPut, "/api/v1/me/avatar",
		map[string]any{"mediaId": nil}, csrf, nil)
	profile = env.requireMyProfile(res, body)
	if profile.AvatarMediaId != nil {
		t.Fatalf("avatarMediaId = %v, want nil", *profile.AvatarMediaId)
	}
}

func TestProfileEndpoints_RequireAuthentication(t *testing.T) {
	env := newProfilesEnv(t)
	client := env.client()
	csrf := env.fetchCSRF(client)

	res, body := env.do(client, http.MethodGet, "/api/v1/me", nil, "", nil)
	requireProblem(t, res, body, http.StatusUnauthorized, "UNAUTHENTICATED")

	res, body = env.do(client, http.MethodPatch, "/api/v1/me/profile",
		map[string]string{"displayName": "Tanpa Login"}, csrf, nil)
	requireProblem(t, res, body, http.StatusUnauthorized, "UNAUTHENTICATED")
}

func ptrString(value string) *string { return &value }
