package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
	"grafikarsa/api/internal/config"
)

// Seed idempoten: 001 majors/years, 002 role_permissions, admin user.
// Admin password di-hash Argon2id (PHC string di password_hash).
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if cfg.Admin.Password == "" || len(cfg.Admin.Password) < 10 {
		fmt.Fprintln(os.Stderr, "seed: ADMIN_SEED_PASSWORD wajib >= 10 char")
		os.Exit(1)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	files, _ := filepath.Glob("db/seeds/*.sql")
	cwdFiles, _ := filepath.Glob("apps/api/db/seeds/*.sql")
	files = append(files, cwdFiles...)
	sort.Strings(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			fmt.Fprintln(os.Stderr, "seed "+f+":", err)
			os.Exit(1)
		}
		fmt.Println("seed OK:", f)
	}

	hash := hashPassword(cfg.Admin.Password)
	_, err = conn.Exec(ctx, `
INSERT INTO users (username, email, password_hash, name, role_primary, status)
VALUES ('admin', $1, $2, 'Administrator', 'admin', 'aktif')
ON CONFLICT (username) DO UPDATE SET email = EXCLUDED.email;
`, cfg.Admin.Email, hash)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed admin:", err)
		os.Exit(1)
	}
	_, err = conn.Exec(ctx, `
INSERT INTO user_roles (user_id, role_id, scope_major_id)
SELECT u.id, r.id, NULL FROM users u, roles r
WHERE u.username = 'admin' AND r.slug = 'admin'
ON CONFLICT DO NOTHING;
`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed admin role:", err)
		os.Exit(1)
	}
	fmt.Println("seed OK: admin", cfg.Admin.Email)
}

func hashPassword(pw string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	hash := argon2.IDKey([]byte(pw), salt, 3, 64*1024, 2, 32)
	return "$argon2id$v=19$m=65536,t=3,p=2$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(hash)
}
