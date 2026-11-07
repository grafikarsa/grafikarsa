package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

const accountCardsPath = "/api/v1/admin/account-cards"

// createStudentViaAPI creates a not-yet-activated student through the admin
// endpoint and returns its id and initial password.
func createStudentViaAPI(t *testing.T, env *usersEnv, client *http.Client, csrf, nisn, displayName string, classID uuid.UUID) (uuid.UUID, string) {
	t.Helper()

	res, body := env.do(client, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"accountType": "student",
		"displayName": displayName,
		"nisn":        nisn,
		"classId":     classID.String(),
	}, csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create student %q status = %d, body %s", displayName, res.StatusCode, body)
	}
	created := decodeUsersBody[apigen.AdminUserCreated](t, body)
	return created.User.Id, created.InitialPassword
}

// exportAuditData returns the decoded data of the newest account-card export
// audit entry.
func exportAuditData(t *testing.T, env *usersEnv) (map[string]any, []byte) {
	t.Helper()

	var raw []byte
	err := env.pool.QueryRow(context.Background(), `
		SELECT data FROM audit_logs
		WHERE action = 'user.export_account_cards'
		ORDER BY id DESC
		LIMIT 1`).Scan(&raw)
	if err != nil {
		t.Fatalf("load export audit row: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("decode export audit data: %v (data %s)", err, raw)
	}
	return data, raw
}

func TestAccountCardsHTTP_AdminExportsClassCards(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.kartu", "Admin Kartu", "admin")
	adminClient, adminCSRF := env.loginAs("admin.kartu", usersTestPassword)

	passwords := make([]string, 0, 3)
	for i, name := range []string{"Budi Santoso", "Siti Aminah", "Ahmad Fauzi"} {
		_, initialPassword := createStudentViaAPI(t, env, adminClient, adminCSRF,
			fmt.Sprintf("10000001%02d", i), name, env.rplClassID)
		requireInitialPassword(t, initialPassword)
		passwords = append(passwords, initialPassword)
	}

	res, body := env.do(adminClient, http.MethodGet,
		accountCardsPath+"?classId="+env.rplClassID.String(), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET account cards status = %d, body %s", res.StatusCode, body)
	}
	if got := res.Header.Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", got)
	}
	if !bytes.HasPrefix(body, []byte("%PDF")) {
		t.Fatalf("response body does not start with %%PDF (first bytes %q)", body[:min(len(body), 16)])
	}
	if got := res.Header.Get("Content-Disposition"); !strings.Contains(got, "attachment") ||
		!strings.Contains(got, "kartu-akun-x-rpl-a.pdf") {
		t.Errorf("Content-Disposition = %q, want an attachment named kartu-akun-x-rpl-a.pdf", got)
	}
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}

	data, raw := exportAuditData(t, env)
	if got, _ := data["class_id"].(string); got != env.rplClassID.String() {
		t.Errorf("audit class_id = %q, want %s", got, env.rplClassID)
	}
	if got, _ := data["count"].(float64); got != 3 {
		t.Errorf("audit count = %v, want 3", data["count"])
	}
	for _, initialPassword := range passwords {
		if bytes.Contains(raw, []byte(initialPassword)) {
			t.Fatalf("audit data contains an initial password: %s", raw)
		}
	}
}

func TestAccountCardsHTTP_MajorAdminCannotExportOtherMajor(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.kartu.global", "Admin Kartu Global", "admin")
	adminClient, adminCSRF := env.loginAs("admin.kartu.global", usersTestPassword)

	createStudentViaAPI(t, env, adminClient, adminCSRF, "1000000201", "Siswa TKJ", env.tkjClassID)

	env.createMajorAdmin("admin.kartu.rpl", "Kaprog RPL Kartu", "rpl")
	majorClient, _ := env.loginAs("admin.kartu.rpl", usersTestPassword)

	res, body := env.do(majorClient, http.MethodGet,
		accountCardsPath+"?classId="+env.tkjClassID.String(), nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	// A plain student must not reach the export endpoint either.
	env.createActivatedStudent("1000000202", "siswa.kartu.biasa", usersTestPassword)
	studentClient, _ := env.loginAs("siswa.kartu.biasa", usersTestPassword)

	res, body = env.do(studentClient, http.MethodGet,
		accountCardsPath+"?classId="+env.rplClassID.String(), nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
}

func TestAccountCardsHTTP_ActivatedClassReturnsValidationError(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.kartu.aktif", "Admin Kartu Aktif", "admin")
	adminClient, adminCSRF := env.loginAs("admin.kartu.aktif", usersTestPassword)

	for i, name := range []string{"Siswa Aktif Satu", "Siswa Aktif Dua"} {
		studentID, _ := createStudentViaAPI(t, env, adminClient, adminCSRF,
			fmt.Sprintf("10000003%02d", i), name, env.rplClassID)
		if _, err := env.pool.Exec(context.Background(), `
			UPDATE users SET activated_at = now(), username = $2 WHERE id = $1`,
			studentID, fmt.Sprintf("siswa.aktif.%d", i),
		); err != nil {
			t.Fatalf("activate student: %v", err)
		}
	}

	res, body := env.do(adminClient, http.MethodGet,
		accountCardsPath+"?classId="+env.rplClassID.String(), nil, "", nil)
	requireProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !bytes.Contains(body, []byte("classId")) {
		t.Errorf("problem body does not mention classId: %s", body)
	}
}
