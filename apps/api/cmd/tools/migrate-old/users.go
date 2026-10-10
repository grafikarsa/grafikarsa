package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// migrateUsers memindahkan users v1 -> users v2 + user_roles.
// Aturan PRD §16.2:
//   - role student + tahun_masuk+3 <= now -> siswa/lulus + graduated
//   - role admin -> admin (hash bcrypt dipertahankan, rehash saat login)
//   - role teacher -> guru
//   - special_roles text[] -> role custom (best-effort, hanya yang dikenal)
func migrateUsers(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	if !s.tables["users"] {
		st.Skipped["users"] = -1
		return fmt.Errorf("tabel users tidak ada di v1")
	}
	idCol := colOr(s, "users", "id")
	userCol := colOr(s, "users", "username")
	mailCol := colOr(s, "users", "email")
	nameCol := colOr(s, "users", "name", "nama", "full_name")
	roleCol := colOr(s, "users", "role")
	pwCol := colOr(s, "users", "password_hash", "password", "encrypted_password")
	masukCol := colOr(s, "users", "tahun_masuk", "angkatan", "entry_year")
	kelasCol := colOr(s, "users", "kelas_id", "class_id")
	jurusanCol := colOr(s, "users", "jurusan_id", "major_id")
	specialCol := colOr(s, "users", "special_roles")
	if idCol == "" || userCol == "" {
		return fmt.Errorf("kolom id/username users v1 tidak dikenali")
	}

	rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s, %s FROM users ORDER BY %s`,
		idCol, userCol, nullCol(mailCol), nullCol(nameCol), nullCol(roleCol),
		nullCol(pwCol), nullCol(masukCol), nullCol(kelasCol), nullCol(jurusanCol), nullCol(specialCol), idCol))
	if err != nil {
		return err
	}
	defer rows.Close()

	// Cache id jurusan/kelas v2 untuk mapping.
	majorByName, classByName := loadAcademicsV2(ctx, v2)

	for rows.Next() {
		var srcID, username, email, name, role, pw, masuk, kelas, jurusan, special any
		if err := rows.Scan(&srcID, &username, &email, &name, &role, &pw, &masuk, &kelas, &jurusan, &special); err != nil {
			return err
		}
		sid := fmt.Sprint(srcID)
		un := strings.ToLower(strings.TrimSpace(fmt.Sprint(username)))
		if un == "" {
			st.UsersSkipped++
			continue
		}
		if _, ok := mappedID(ctx, v2, "users", sid, "users"); ok {
			st.UsersSkipped++
			continue
		}

		rolePrimary, status := mapRole(fmt.Sprint(role), toInt(masuk))
		displayName := fmt.Sprint(name)
		if displayName == "" || displayName == "<nil>" {
			displayName = un
		}
		mail := strings.ToLower(strings.TrimSpace(fmt.Sprint(email)))
		if mail == "" || mail == "<nil>" {
			mail = un + "@migrated.local"
		}
		hash := fmt.Sprint(pw)
		if !isBcrypt(hash) {
			// Password tidak dikenali -> paksa ganti saat login pertama.
			hash = mustHash("Migrated-" + un)
		}
		majorID := resolveMajor(fmt.Sprint(jurusan), majorByName)
		classID := resolveClass(fmt.Sprint(kelas), classByName)

		var targetID string
		if !o.dryRun {
			err := v2.QueryRow(ctx, `INSERT INTO users
				(username, email, password_hash, name, role_primary, status, major_id, class_id, must_change_password)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,TRUE)
				ON CONFLICT (username) DO UPDATE SET email=EXCLUDED.email
				RETURNING id::text`,
				un, mail, hash, displayName, rolePrimary, status, nullUUID(majorID), nullUUID(classID)).Scan(&targetID)
			if err != nil {
				slog.Warn("user gagal dimigrasi", "username", un, "err", err)
				st.UsersSkipped++
				continue
			}
			if err := assignSystemRole(ctx, v2, targetID, rolePrimary, majorID); err != nil {
				slog.Warn("assign role gagal", "username", un, "err", err)
			}
			if spec := fmt.Sprint(special); spec != "" && spec != "<nil>" && spec != "{}" {
				assignSpecialRoles(ctx, v2, targetID, spec)
			}
			if err := saveMap(ctx, v2, o.dryRun, "users", sid, "users", targetID); err != nil {
				return err
			}
		}
		st.Users++
		_ = targetID
	}
	return rows.Err()
}

func nullCol(c string) string {
	if c == "" {
		return "NULL"
	}
	return c
}

func toInt(v any) int {
	var n int
	fmt.Sscanf(fmt.Sprint(v), "%d", &n)
	return n
}

// mapRole menerapkan aturan alumni PRD §16.2.
func mapRole(v1role string, tahunMasuk int) (rolePrimary, status string) {
	switch strings.ToLower(strings.TrimSpace(v1role)) {
	case "admin", "administrator", "superadmin":
		return "admin", "aktif"
	case "teacher", "guru", "pengajar":
		return "guru", "aktif"
	default:
		if tahunMasuk > 0 && tahunMasuk+3 <= nowYear() {
			return "siswa", "lulus"
		}
		return "siswa", "aktif"
	}
}

func isBcrypt(h string) bool {
	_, err := bcrypt.Cost([]byte(h))
	return err == nil
}

func mustHash(pw string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

func loadAcademicsV2(ctx context.Context, v2 *pgxpool.Pool) (map[string]string, map[string]string) {
	majors := map[string]string{}
	if rows, err := v2.Query(ctx, `SELECT id::text, lower(code) FROM majors`); err == nil {
		for rows.Next() {
			var id, code string
			if err := rows.Scan(&id, &code); err == nil {
				majors[code] = id
			}
		}
		rows.Close()
	}
	classes := map[string]string{}
	if rows, err := v2.Query(ctx, `SELECT id::text, lower(name) FROM classes`); err == nil {
		for rows.Next() {
			var id, name string
			if err := rows.Scan(&id, &name); err == nil {
				classes[name] = id
			}
		}
		rows.Close()
	}
	return majors, classes
}

func resolveMajor(v string, by map[string]string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	if id, ok := by[v]; ok {
		return id
	}
	return ""
}

func resolveClass(v string, by map[string]string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	if id, ok := by[v]; ok {
		return id
	}
	return ""
}

func nullUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func assignSystemRole(ctx context.Context, v2 *pgxpool.Pool, userID, rolePrimary, majorID string) error {
	var roleID string
	err := v2.QueryRow(ctx, `SELECT id::text FROM roles WHERE slug=$1`, rolePrimary).Scan(&roleID)
	if err != nil {
		return err
	}
	scope := nullUUID("")
	if rolePrimary == "admin_jurusan" && majorID != "" {
		scope = majorID
	}
	_, err = v2.Exec(ctx, `INSERT INTO user_roles (user_id, role_id, scope_major_id)
		VALUES ($1,$2,$3) ON CONFLICT (user_id, role_id) DO NOTHING`, userID, roleID, scope)
	return err
}

// assignSpecialRoles memetakan special_roles v1 -> permission v2 (best-effort).
func assignSpecialRoles(ctx context.Context, v2 *pgxpool.Pool, userID, spec string) {
	spec = strings.Trim(spec, "{}[]")
	for _, part := range strings.Split(spec, ",") {
		p := strings.ToLower(strings.TrimSpace(part))
		var perms []string
		switch {
		case strings.Contains(p, "moderation"), strings.Contains(p, "moderator"):
			perms = []string{"portfolios.moderate"}
		case strings.Contains(p, "major"):
			perms = []string{"academic.view"}
		case strings.Contains(p, "assess"):
			perms = []string{"assessments.score"}
		default:
			continue
		}
		for _, perm := range perms {
			var permID string
			if err := v2.QueryRow(ctx, `SELECT id::text FROM permissions WHERE key=$1`, perm).Scan(&permID); err != nil {
				continue
			}
			v2.Exec(ctx, `INSERT INTO user_roles (user_id, role_id, scope_major_id)
				SELECT $1, r.id, NULL FROM roles r WHERE r.slug='custom-import'
				ON CONFLICT DO NOTHING`, userID)
		}
	}
}

// migrateAcademics membangun enrollments dari users.kelas_id + riwayat kelas.
func migrateAcademics(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	if !s.tables["users"] {
		return nil
	}
	kelasCol := colOr(s, "users", "kelas_id", "class_id")
	if kelasCol == "" {
		return nil
	}
	// Enrollments dibuat untuk user yang punya kelas dan sudah termigrasi.
	rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT id, %s FROM users WHERE %s IS NOT NULL`, kelasCol, kelasCol))
	if err != nil {
		return err
	}
	defer rows.Close()
	var activeYear string
	_ = v2.QueryRow(ctx, `SELECT id::text FROM academic_years WHERE is_active ORDER BY start_year DESC LIMIT 1`).Scan(&activeYear)
	made := 0
	for rows.Next() {
		var srcID, kelas any
		if err := rows.Scan(&srcID, &kelas); err != nil {
			return err
		}
		targetID, ok := mappedID(ctx, v2, "users", fmt.Sprint(srcID), "users")
		if !ok || o.dryRun || activeYear == "" {
			continue
		}
		classV2, ok := mappedID(ctx, v2, "classes", fmt.Sprint(kelas), "classes")
		if !ok {
			continue
		}
		_, _ = v2.Exec(ctx, `INSERT INTO enrollments (user_id, class_id, academic_year_id, status, is_current)
			VALUES ($1,$2,$3,'active',TRUE) ON CONFLICT DO NOTHING`, targetID, classV2, activeYear)
		made++
	}
	slog.Info("enrollments dibuat dari users.kelas_id", "count", made)
	return rows.Err()
}
