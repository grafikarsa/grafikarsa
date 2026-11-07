// Command admin manages accounts that cannot be created through the API yet.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/config"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/db"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	adminTimeout       = 2 * time.Minute
	initialPasswordTTL = 7 * 24 * time.Hour
	maxDisplayName     = 100
)

const usage = `usage: admin create --name "Nama Admin" --username admin.sekolah [--staff-number 123]

Membuat akun admin dengan sandi awal yang berlaku 7 hari. Akun wajib
menyelesaikan aktivasi saat pertama kali masuk.`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}

	switch args[0] {
	case "create":
		return runCreate(args[1:])
	default:
		printUsage()
		return 2
	}
}

func runCreate(args []string) int {
	flags := flag.NewFlagSet("admin create", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	name := flags.String("name", "", "nama tampilan admin")
	username := flags.String("username", "", "username admin")
	staffNumber := flags.String("staff-number", "", "NIP atau ID staf (opsional)")

	if err := flags.Parse(args); err != nil {
		printUsage()
		return 2
	}
	if flags.NArg() > 0 {
		printUsage()
		return 2
	}

	displayName := strings.TrimSpace(*name)
	if displayName == "" || utf8.RuneCountInString(displayName) > maxDisplayName {
		fmt.Fprintf(os.Stderr, "nama wajib diisi (1–%d karakter)\n", maxDisplayName)
		printUsage()
		return 2
	}

	normalized := auth.NormalizeUsername(*username)
	if reason := auth.ValidateUsernameFormat(normalized); reason != "" {
		fmt.Fprintf(os.Stderr,
			"username %q tidak valid: 3–30 karakter, huruf kecil, angka, titik, atau garis bawah\n",
			normalized)
		printUsage()
		return 2
	}

	staff := strings.TrimSpace(*staffNumber)
	var staffValue *string
	if staff != "" {
		staffValue = &staff
	}

	if err := config.LoadDotEnv(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "config:", err)
		return 1
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL wajib diisi")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminTimeout)
	defer cancel()

	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		return 1
	}
	defer pool.Close()

	st := store.NewStore(pool)

	if reserved, err := st.IsUsernameReserved(ctx, normalized); err != nil {
		fmt.Fprintln(os.Stderr, "cek username:", err)
		return 1
	} else if reserved {
		fmt.Fprintf(os.Stderr, "username %q dicadangkan dan tidak boleh dipakai\n", normalized)
		return 1
	}
	if taken, err := st.IsUsernameTaken(ctx, store.IsUsernameTakenParams{
		Username: normalized,
		UserID:   uuid.Nil,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "cek username:", err)
		return 1
	} else if taken {
		fmt.Fprintf(os.Stderr, "username %q sudah dipakai\n", normalized)
		return 1
	}

	plain, err := securetoken.ReadablePassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, "buat sandi awal:", err)
		return 1
	}

	hasher := password.NewHasher(password.DefaultParams, 0)
	hash, err := hasher.Hash(ctx, plain)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash sandi awal:", err)
		return 1
	}

	err = st.WithTx(ctx, func(q *store.Queries) error {
		id, err := q.CreateStaffUser(ctx, store.CreateStaffUserParams{
			DisplayName:       displayName,
			Username:          normalized,
			StaffNumber:       staffValue,
			PasswordHash:      hash,
			PasswordExpiresAt: time.Now().UTC().Add(initialPasswordTTL),
		})
		if err != nil {
			return err
		}
		roleID, err := q.GetRoleIDBySlug(ctx, "admin")
		if err != nil {
			return fmt.Errorf("role admin tidak ditemukan (jalankan seed dulu): %w", err)
		}
		if err := q.InsertSystemUserRole(ctx, store.InsertSystemUserRoleParams{
			UserID: id,
			RoleID: roleID,
		}); err != nil {
			return err
		}

		return audit.Record(ctx, q, audit.Entry{
			Action:     "user.create_admin",
			TargetType: "user",
			TargetID:   &id,
		})
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fmt.Fprintf(os.Stderr, "username %q sudah dipakai\n", normalized)
			return 1
		}
		fmt.Fprintln(os.Stderr, "buat admin:", err)
		return 1
	}

	fmt.Printf("Admin dibuat.\n")
	fmt.Printf("  Username : %s\n", normalized)
	fmt.Printf("  Sandi awal: %s   (berlaku 7 hari, wajib diganti saat aktivasi)\n", plain)

	return 0
}

func printUsage() {
	fmt.Fprintln(os.Stderr, usage)
}
