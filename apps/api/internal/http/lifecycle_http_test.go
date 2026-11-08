package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

func TestStudioContentLifecycle_SubmitWithdrawDeleteRestore(t *testing.T) {
	env := newStudioEnv(t)
	ownerID := env.createStudent("1000000101", "siswalifecycle", "Siswa Lifecycle")
	client, csrf := env.loginAs("1000000101")

	res, body := env.do(client, http.MethodPost, "/api/v1/studio/contents", map[string]any{
		"type": "portfolio",
	}, csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /studio/contents status = %d, body %s", res.StatusCode, body)
	}
	var created apigen.StudioContent
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}

	var categoryID uuid.UUID
	if err := env.pool.QueryRow(env.t.Context(), `
		SELECT id FROM content_categories WHERE content_type = 'portfolio' AND slug = 'web-app'`,
	).Scan(&categoryID); err != nil {
		t.Fatalf("load category: %v", err)
	}

	var mediaID uuid.UUID
	if err := env.pool.QueryRow(env.t.Context(), `
		INSERT INTO media_assets (
			owner_id, kind, purpose, original_key, original_filename, mime_type,
			size_bytes, width, height, status
		) VALUES ($1, 'image', 'content', $2, 'poster.png', 'image/png', 1024, 1200, 675, 'ready')
		RETURNING id`,
		ownerID, "test/"+uuid.NewString()+".png",
	).Scan(&mediaID); err != nil {
		t.Fatalf("insert ready media: %v", err)
	}

	draftPath := "/api/v1/studio/contents/" + created.Id.String() + "/draft"
	draftBody := map[string]any{
		"lockVersion":  created.Draft.LockVersion,
		"title":        "Poster Lifecycle",
		"excerpt":      "Poster untuk uji alur lifecycle konten.",
		"categoryId":   categoryID.String(),
		"coverMediaId": mediaID.String(),
		"body": map[string]any{"blocks": []any{
			map[string]any{
				"id":   uuid.NewString(),
				"type": "image",
				"data": map[string]any{"media_id": mediaID.String(), "fit": "contain"},
			},
		}},
		"meta": map[string]any{"context": "personal"},
		"tags": []string{"poster"},
	}
	res, body = env.do(client, http.MethodPut, draftPath, draftBody, csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT draft status = %d, body %s", res.StatusCode, body)
	}

	contentPath := "/api/v1/studio/contents/" + created.Id.String()
	res, body = env.do(client, http.MethodPost, contentPath+"/submit", nil, csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST submit status = %d, body %s", res.StatusCode, body)
	}
	var submitted apigen.StudioContent
	if err := json.Unmarshal(body, &submitted); err != nil {
		t.Fatalf("decode submitted: %v", err)
	}
	if submitted.Status != apigen.ContentStatusInReview {
		t.Fatalf("status submit = %s, mau in_review", submitted.Status)
	}
	if submitted.RevisionStatus != nil {
		t.Errorf("revisionStatus = %v, mau nil", submitted.RevisionStatus)
	}

	res, body = env.do(client, http.MethodGet, contentPath+"/versions", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET versions status = %d, body %s", res.StatusCode, body)
	}
	var versions apigen.ContentVersionList
	if err := json.Unmarshal(body, &versions); err != nil {
		t.Fatalf("decode versions: %v", err)
	}
	if len(versions.Items) != 1 {
		t.Fatalf("jumlah versi = %d, mau 1", len(versions.Items))
	}
	version := versions.Items[0]
	if version.Number != 1 || version.Kind != apigen.ContentVersionKindInitial {
		t.Errorf("versi = %+v, mau number 1 kind initial", version)
	}
	if version.SubmittedBy.DisplayName != "Siswa Lifecycle" {
		t.Errorf("submittedBy = %+v", version.SubmittedBy)
	}

	res, body = env.do(client, http.MethodGet,
		contentPath+"/versions/"+version.Id.String(), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET version detail status = %d, body %s", res.StatusCode, body)
	}
	var detail apigen.ContentVersionDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode version detail: %v", err)
	}
	if detail.Title != "Poster Lifecycle" {
		t.Errorf("detail title = %q", detail.Title)
	}
	if detail.Category == nil || detail.Category.Slug != "web-app" {
		t.Errorf("detail category = %+v", detail.Category)
	}
	if detail.Cover == nil || detail.Cover.MediaId != mediaID {
		t.Errorf("detail cover = %+v", detail.Cover)
	}
	if len(detail.Tags) != 1 || detail.Tags[0].Slug != "poster" {
		t.Errorf("detail tags = %+v", detail.Tags)
	}

	env.createStudent("1000000102", "siswalainlifecycle", "Siswa Lain Lifecycle")
	otherClient, otherCSRF := env.loginAs("1000000102")
	res, body = env.do(otherClient, http.MethodPost, contentPath+"/submit", nil, otherCSRF, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("submit oleh pengguna lain status = %d, body %s", res.StatusCode, body)
	}
	res, body = env.do(otherClient, http.MethodGet, contentPath+"/versions", nil, "", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("versions oleh pengguna lain status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(client, http.MethodPost, contentPath+"/withdraw", nil, csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST withdraw status = %d, body %s", res.StatusCode, body)
	}
	var withdrawn apigen.StudioContent
	if err := json.Unmarshal(body, &withdrawn); err != nil {
		t.Fatalf("decode withdrawn: %v", err)
	}
	if withdrawn.Status != apigen.ContentStatusDraft {
		t.Fatalf("status withdraw = %s, mau draft", withdrawn.Status)
	}

	res, body = env.do(client, http.MethodDelete, contentPath, nil, csrf, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE content status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(client, http.MethodPost, contentPath+"/restore", nil, csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST restore status = %d, body %s", res.StatusCode, body)
	}
	var restored apigen.StudioContent
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("decode restored: %v", err)
	}
	if restored.Status != apigen.ContentStatusDraft {
		t.Fatalf("status restore = %s, mau draft", restored.Status)
	}
	if restored.DeletedAt != nil || restored.PurgeAfter != nil {
		t.Errorf("deletedAt/purgeAfter = %v/%v, mau nil", restored.DeletedAt, restored.PurgeAfter)
	}
}
