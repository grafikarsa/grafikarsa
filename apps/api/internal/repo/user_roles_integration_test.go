package repo_test

import (
	"context"
	"testing"

	"grafikarsa/api/internal/dbtest"
	"grafikarsa/api/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedRole membuat role khusus test.
//
// Slug-nya unik per eksekusi dan role-nya dihapus setelah test selesai.
//
// Penting: test TIDAK boleh meng-truncate tabel `roles` atau `permissions`
// — keduanya berisi role & permission SISTEM yang dibuat migrasi. Menghapus
// membuat seluruh aplikasi kehilangan hak akses: pada database dev lokal
// efeknya langsung terasa (siswa tidak lagi bisa membuat karya), dan di CI
// kebetulan tidak ketahuan karena database selalu dibuat ulang.
//
// Test ini hanya butuh role milik sendiri, jadi cukup membuat & membersihkannya.
func seedRole(t *testing.T, pool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	unique := slug + "_" + id.String()[:8]
	_, err := pool.Exec(ctx,
		`INSERT INTO roles (id, slug, name, scope_type, is_system) VALUES ($1, $2, $3, 'jurusan', false)`,
		id, unique, unique)
	if err != nil {
		t.Fatalf("seed role %s: %v", unique, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM roles WHERE id = $1`, id)
	})
	return id
}

// seedMajor membuat jurusan khusus test, lalu membersihkannya.
func seedMajor(t *testing.T, pool *pgxpool.Pool, code string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	unique := code + "-" + id.String()[:8]
	_, err := pool.Exec(ctx, `INSERT INTO majors (id, code, name) VALUES ($1, $2, $3)`, id, unique, unique)
	if err != nil {
		t.Fatalf("seed major %s: %v", unique, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM majors WHERE id = $1`, id)
	})
	return id
}

func countUserRoles(t *testing.T, pool *pgxpool.Pool, userID, roleID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM user_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID).Scan(&n)
	if err != nil {
		t.Fatalf("count user_roles: %v", err)
	}
	return n
}

// scopeUUID membungkus uuid.UUID menjadi pgtype.UUID yang valid.
func scopeUUID(id uuid.UUID) pgtype.UUID { return mustUUID(id) }

// scopeNull adalah scope kosong (role global).
func scopeNull() pgtype.UUID { return pgtype.UUID{} }

// TestAssignUserRoleScopeSemantics menguji semantik assignment role setelah
// primary key user_roles diubah.
//
// Tiga hal yang dijaga:
//
//  1. Assignment dengan scope yang SAMA diperpanjang (bukan diduplikasi) —
//     inilah makna "renew".
//  2. Assignment dengan scope BERBEDA menjadi baris baru; skema tidak lagi
//     melarangnya. Di sisi aplikasi scope lama tetap dibuang (satu user satu
//     scope per role), jadi tidak ada dua baris yang saling bertentangan.
//  3. Scope NULL (role global) diperlakukan sebagai satu nilai oleh unique
//     index, jadi assignment global tidak bisa terduplikasi.
func TestAssignUserRoleScopeSemantics(t *testing.T) {
	pool := dbtest.Pool(t)
	// Hanya tabel yang dipakai test yang dikosongkan. `roles` dan
	// `permissions` TIDAK boleh disentuh: isinya role & permission sistem.
	dbtest.Truncate(t, pool, "user_roles", "users", "majors")
	q := repo.New(pool)
	ctx := context.Background()

	uid := seedUser(t, pool, "roleuser01")
	roleID := seedRole(t, pool, "test_role_a")
	majorA := seedMajor(t, pool, "MJA")
	majorB := seedMajor(t, pool, "MJB")

	assign := func(scope pgtype.UUID) {
		t.Helper()
		if err := q.AssignUserRole(ctx, repo.AssignUserRoleParams{
			UserID:       mustUUID(uid),
			RoleID:       mustUUID(roleID),
			ScopeMajorID: scope,
			AssignedBy:   mustUUID(uid),
		}); err != nil {
			t.Fatalf("AssignUserRole: %v", err)
		}
	}

	// 1. Scope yang sama dua kali -> diperpanjang, tetap satu baris.
	assign(scopeUUID(majorA))
	assign(scopeUUID(majorA))
	if n := countUserRoles(t, pool, uid, roleID); n != 1 {
		t.Errorf("assignment dengan scope sama harus tetap 1 baris, dapat %d", n)
	}

	// 2. Scope berbeda -> baris baru (skema mengizinkan).
	assign(scopeUUID(majorB))
	if n := countUserRoles(t, pool, uid, roleID); n != 2 {
		t.Errorf("assignment dengan scope berbeda harus menambah baris, dapat %d", n)
	}

	// 3. DeleteUserRoleOtherScope(X) membuang semua assignment dengan scope
	//    selain X. Handler memanggilnya dengan scope BARU sebelum assign,
	//    sehingga hasil akhirnya tepat satu baris dengan scope baru.
	if err := q.DeleteUserRoleOtherScope(ctx, repo.DeleteUserRoleOtherScopeParams{
		UserID: mustUUID(uid), RoleID: mustUUID(roleID), ScopeMajorID: scopeUUID(majorB),
	}); err != nil {
		t.Fatalf("DeleteUserRoleOtherScope: %v", err)
	}
	if n := countUserRoles(t, pool, uid, roleID); n != 1 {
		t.Errorf("setelah DeleteUserRoleOtherScope harusnya 1 baris, dapat %d", n)
	}
	var left uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT scope_major_id FROM user_roles WHERE user_id = $1 AND role_id = $2`, uid, roleID).Scan(&left); err != nil {
		t.Fatalf("baca scope yang tersisa: %v", err)
	}
	if left != majorB {
		t.Errorf("scope yang tersisa = %s, ingin %s", left, majorB)
	}
}

// TestAssignUserRoleGlobalScopeUnik memastikan scope NULL tidak bisa
// terduplikasi (unique index memakai NULLS NOT DISTINCT).
func TestAssignUserRoleGlobalScopeUnik(t *testing.T) {
	pool := dbtest.Pool(t)
	// Hanya tabel yang dipakai test yang dikosongkan. `roles` dan
	// `permissions` TIDAK boleh disentuh: isinya role & permission sistem.
	dbtest.Truncate(t, pool, "user_roles", "users", "majors")
	q := repo.New(pool)
	ctx := context.Background()

	uid := seedUser(t, pool, "globaluser01")
	roleID := seedRole(t, pool, "test_role_global")

	for i := 0; i < 3; i++ {
		if err := q.AssignUserRole(ctx, repo.AssignUserRoleParams{
			UserID:       mustUUID(uid),
			RoleID:       mustUUID(roleID),
			ScopeMajorID: scopeNull(),
			AssignedBy:   mustUUID(uid),
		}); err != nil {
			t.Fatalf("AssignUserRole (global): %v", err)
		}
	}
	if n := countUserRoles(t, pool, uid, roleID); n != 1 {
		t.Errorf("assignment global (scope NULL) harus tetap 1 baris, dapat %d", n)
	}
}
