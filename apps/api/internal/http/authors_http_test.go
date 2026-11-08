package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

func TestAuthorsFlow_InviteAcceptAndTransferOwnership(t *testing.T) {
	env := newStudioEnv(t)

	env.createStudent("1000000101", "pemilikkolab", "Pemilik Kolab")
	inviteeID := env.createStudent("1000000102", "undangankolab", "Undangan Kolab")

	ownerClient, ownerCSRF := env.loginAs("1000000101")
	inviteeClient, inviteeCSRF := env.loginAs("1000000102")

	res, body := env.do(ownerClient, http.MethodPost, "/api/v1/studio/contents", map[string]any{
		"type": "portfolio",
	}, ownerCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /studio/contents status = %d, body %s", res.StatusCode, body)
	}
	var created apigen.StudioContent
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode content: %v", err)
	}
	authorsPath := "/api/v1/studio/contents/" + created.Id.String() + "/authors"

	res, body = env.do(ownerClient, http.MethodPost, authorsPath, map[string]any{
		"userId":    "00000000-0000-0000-0000-000000000000",
		"level":     "editor",
		"roleLabel": "Backend",
	}, ownerCSRF, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("undang user tidak ada status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(ownerClient, http.MethodPost, authorsPath, map[string]any{
		"userId":    inviteeID,
		"level":     "editor",
		"roleLabel": "Backend",
	}, ownerCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST author status = %d, body %s", res.StatusCode, body)
	}
	var invited apigen.ContentAuthor
	if err := json.Unmarshal(body, &invited); err != nil {
		t.Fatalf("decode author: %v", err)
	}
	if invited.InviteStatus != apigen.InviteStatusPending || invited.Level != apigen.AuthorLevelEditor {
		t.Fatalf("author = %+v, mau pending/editor", invited)
	}

	res, body = env.do(inviteeClient, http.MethodGet, "/api/v1/studio/invitations", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET invitations status = %d, body %s", res.StatusCode, body)
	}
	var invitations apigen.InvitationPage
	if err := json.Unmarshal(body, &invitations); err != nil {
		t.Fatalf("decode invitations: %v", err)
	}
	if len(invitations.Items) != 1 || invitations.Items[0].Content.Id != created.Id {
		t.Fatalf("invitations = %+v, mau satu untuk %s", invitations.Items, created.Id)
	}
	if invitations.Items[0].Content.Title != created.Title {
		t.Errorf("judul undangan = %q, mau %q", invitations.Items[0].Content.Title, created.Title)
	}

	res, body = env.do(inviteeClient, http.MethodPost,
		"/api/v1/studio/invitations/"+created.Id.String()+"/accept", nil, inviteeCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("accept invitation status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(inviteeClient, http.MethodGet, "/api/v1/studio/contents", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET studio contents invitee status = %d, body %s", res.StatusCode, body)
	}
	var inviteePage apigen.StudioContentPage
	if err := json.Unmarshal(body, &inviteePage); err != nil {
		t.Fatalf("decode invitee page: %v", err)
	}
	if len(inviteePage.Items) != 1 || inviteePage.Items[0].Id != created.Id {
		t.Fatalf("items invitee = %+v, mau %s", inviteePage.Items, created.Id)
	}
	if inviteePage.Items[0].MyLevel != apigen.AuthorLevelEditor {
		t.Errorf("myLevel invitee = %s, mau editor", inviteePage.Items[0].MyLevel)
	}

	res, body = env.do(ownerClient, http.MethodPost, "/api/v1/studio/contents/"+created.Id.String()+"/transfer-ownership", map[string]any{
		"userId": inviteeID,
	}, ownerCSRF, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("transfer ownership status = %d, body %s", res.StatusCode, body)
	}
	var authors apigen.ContentAuthorList
	if err := json.Unmarshal(body, &authors); err != nil {
		t.Fatalf("decode authors: %v", err)
	}
	if len(authors.Items) != 2 {
		t.Fatalf("authors = %+v, mau 2", authors.Items)
	}

	res, body = env.do(ownerClient, http.MethodGet, "/api/v1/studio/contents", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET studio contents mantan pemilik status = %d, body %s", res.StatusCode, body)
	}
	var formerOwnerPage apigen.StudioContentPage
	if err := json.Unmarshal(body, &formerOwnerPage); err != nil {
		t.Fatalf("decode former owner page: %v", err)
	}
	if len(formerOwnerPage.Items) != 1 {
		t.Fatalf("items mantan pemilik = %+v, mau 1", formerOwnerPage.Items)
	}
	if formerOwnerPage.Items[0].MyLevel != apigen.AuthorLevelEditor {
		t.Errorf("myLevel mantan pemilik = %s, mau editor", formerOwnerPage.Items[0].MyLevel)
	}
}

func TestAuthorsFlow_StrangerCannotListAuthors(t *testing.T) {
	env := newStudioEnv(t)

	env.createStudent("1000000111", "pemilikkedua", "Pemilik Kedua")
	env.createStudent("1000000112", "orangasing", "Orang Asing")

	ownerClient, ownerCSRF := env.loginAs("1000000111")
	strangerClient, _ := env.loginAs("1000000112")

	res, body := env.do(ownerClient, http.MethodPost, "/api/v1/studio/contents", map[string]any{
		"type": "portfolio",
	}, ownerCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /studio/contents status = %d, body %s", res.StatusCode, body)
	}
	var created apigen.StudioContent
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode content: %v", err)
	}

	res, body = env.do(strangerClient, http.MethodGet,
		"/api/v1/studio/contents/"+created.Id.String()+"/authors", nil, "", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("GET authors oleh orang asing status = %d, body %s", res.StatusCode, body)
	}
}
