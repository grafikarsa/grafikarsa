// Command migrate applies the goose and River database migrations.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/grafikarsa/grafikarsa/apps/api/db/migrations"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/config"
)

const migrateTimeout = 5 * time.Minute

const usage = `usage: migrate <command> [arg]

commands:
  up              apply all goose migrations, then all River migrations
  down            roll back one goose migration
  down-to <ver>   roll back goose migrations to <ver>
  status          show goose status and the current River version
  version         show the current goose version`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}

	command := args[0]
	switch command {
	case "up", "down", "status", "version":
	case "down-to":
		if len(args) < 2 {
			printUsage()
			return 2
		}
		if _, err := strconv.ParseInt(args[1], 10, 64); err != nil {
			fmt.Fprintf(os.Stderr, "versi tidak valid: %s\n", args[1])
			printUsage()
			return 2
		}
	default:
		printUsage()
		return 2
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

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()

	if err := runCommand(ctx, command, args[1:], databaseURL); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

func runCommand(ctx context.Context, command string, args []string, databaseURL string) error {
	switch command {
	case "up":
		return withGoose(ctx, databaseURL, func(db *sql.DB) error {
			if err := goose.UpContext(ctx, db, "."); err != nil {
				return fmt.Errorf("goose up: %w", err)
			}
			return migrateRiverUp(ctx, databaseURL)
		})
	case "down":
		return withGoose(ctx, databaseURL, func(db *sql.DB) error {
			if err := goose.DownContext(ctx, db, "."); err != nil {
				return fmt.Errorf("goose down: %w", err)
			}
			return nil
		})
	case "down-to":
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("versi tidak valid: %s", args[0])
		}
		return withGoose(ctx, databaseURL, func(db *sql.DB) error {
			if err := goose.DownToContext(ctx, db, ".", version); err != nil {
				return fmt.Errorf("goose down-to %d: %w", version, err)
			}
			return nil
		})
	case "status":
		return withGoose(ctx, databaseURL, func(db *sql.DB) error {
			if err := goose.StatusContext(ctx, db, "."); err != nil {
				return fmt.Errorf("goose status: %w", err)
			}
			return riverStatus(ctx, databaseURL)
		})
	case "version":
		return withGoose(ctx, databaseURL, func(db *sql.DB) error {
			version, err := goose.GetDBVersionContext(ctx, db)
			if err != nil {
				return fmt.Errorf("goose version: %w", err)
			}
			fmt.Printf("goose version: %d\n", version)
			return nil
		})
	default:
		return fmt.Errorf("perintah tidak dikenal: %s", command)
	}
}

func withGoose(ctx context.Context, databaseURL string, fn func(*sql.DB) error) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("buka database: %w", err)
	}
	defer db.Close()

	return fn(db)
}

func migrateRiverUp(ctx context.Context, databaseURL string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("river buka database: %w", err)
	}
	defer pool.Close()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("river: %w", err)
	}

	result, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("river migrate up: %w", err)
	}

	if len(result.Versions) == 0 {
		fmt.Println("River migrations already up to date")
		return nil
	}
	for _, version := range result.Versions {
		fmt.Printf("River migration %d applied: %s\n", version.Version, version.Name)
	}

	return nil
}

func riverStatus(ctx context.Context, databaseURL string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("river buka database: %w", err)
	}
	defer pool.Close()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("river: %w", err)
	}

	existing, err := migrator.ExistingVersions(ctx)
	if err != nil {
		return fmt.Errorf("river status: %w", err)
	}

	var current int
	for _, migration := range existing {
		if migration.Version > current {
			current = migration.Version
		}
	}
	fmt.Printf("River current version: %d\n", current)

	return nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, usage)
}
