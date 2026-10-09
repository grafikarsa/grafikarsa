package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
	"grafikarsa/api/internal/config"
)

// Verifikasi seed: counts + password admin valid. Exit 1 + pesan jelas bila gagal.
func main() {
	cfg, err := config.Load()
	if err != nil {
		fail(err.Error())
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL())
	if err != nil {
		fail(err.Error())
	}
	defer conn.Close(ctx)

	checks := []struct {
		name     string
		query    string
		minCount int64
	}{
		{"majors", "SELECT COUNT(*) FROM majors", 5},
		{"academic_years", "SELECT COUNT(*) FROM academic_years", 1},
		{"roles", "SELECT COUNT(*) FROM roles WHERE is_system", 4},
		{"permissions", "SELECT COUNT(*) FROM permissions", 53},
		{"admin_users", "SELECT COUNT(*) FROM users WHERE username='admin' AND deleted_at IS NULL", 1},
	}
	for _, c := range checks {
		var n int64
		if err := conn.QueryRow(ctx, c.query).Scan(&n); err != nil {
			fail(c.name + ": " + err.Error())
		}
		if n < c.minCount {
			fail(fmt.Sprintf("%s: got %d, want >= %d", c.name, n, c.minCount))
		}
		fmt.Printf("verify OK: %s = %d\n", c.name, n)
	}

	var hash string
	if err := conn.QueryRow(ctx, "SELECT password_hash FROM users WHERE username='admin'").Scan(&hash); err != nil {
		fail("admin hash: " + err.Error())
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		fail("admin hash: bukan argon2id PHC")
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		fail("admin hash: format PHC tidak valid")
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[4])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[5])
	if err1 != nil || err2 != nil {
		fail("admin hash: decode gagal")
	}
	got := argon2.IDKey([]byte(cfg.Admin.Password), salt, 3, 64*1024, 2, 32)
	if subtleNe(len(got), len(want)) || !equalFold(got, want) {
		fail("admin password: ADMIN_SEED_PASSWORD tidak cocok dengan hash di DB")
	}
	fmt.Println("verify OK: admin password cocok")
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "verify-seed GAGAL:", msg)
	os.Exit(1)
}

func subtleNe(a, b int) bool { return a != b }

func equalFold(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
