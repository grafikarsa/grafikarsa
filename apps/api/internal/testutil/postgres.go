// Package testutil provides integration-test helpers backed by real
// PostgreSQL databases started with testcontainers.
package testutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/grafikarsa/grafikarsa/apps/api/db/migrations"
	"github.com/grafikarsa/grafikarsa/apps/api/db/seeds"
)

const (
	postgresImage = "postgres:18.4-alpine"
	adminDatabase = "postgres"
	dbUser        = "postgres"
	dbPassword    = "postgres"

	templateDatabase       = "gk_template"
	templateSeededDatabase = "gk_template_seeded"

	setupTimeout = 2 * time.Minute
)

var (
	setupOnce sync.Once
	// setupErr holds a non-Docker setup failure; dockerErr holds the original
	// error when the container could not be started at all.
	setupErr  error
	dockerErr error

	adminConn *pgx.Conn
	adminURL  string

	// dbMu serializes CREATE DATABASE clones and DROP DATABASE calls, because
	// PostgreSQL refuses concurrent clones from the same template.
	dbMu sync.Mutex

	started atomic.Bool
)

// NewDB returns a pgx pool connected to a fresh, fully migrated database
// (goose + River) dedicated to this test. Databases are cloned from a
// template, so each call takes milliseconds after the first one.
// The test is skipped (t.Skip) when Docker is not available.
func NewDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return newDatabase(t, templateDatabase)
}

// NewSeededDB is NewDB plus all seed files from db/seeds applied in order.
func NewSeededDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return newDatabase(t, templateSeededDatabase)
}

// DatabaseURL returns the connection string of the database behind pool (handy for code that needs a URL).
func DatabaseURL(t testing.TB, pool *pgxpool.Pool) string {
	t.Helper()
	if pool == nil {
		t.Fatal("pool database bernilai nil")
	}
	return pool.Config().ConnString()
}

func newDatabase(t testing.TB, template string) *pgxpool.Pool {
	t.Helper()
	ensureSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
	defer cancel()

	name := newDatabaseName(t)
	if err := createDatabase(ctx, name, template); err != nil {
		t.Fatalf("siapkan database uji: %v", err)
	}

	url := databaseURL(name)
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		dropDatabase(t, name)
		t.Fatalf("urai URL database uji: %v", err)
	}
	cfg.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		dropDatabase(t, name)
		t.Fatalf("buka pool database uji: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		dropDatabase(t, name)
		t.Fatalf("ping database uji: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		dropDatabase(t, name)
	})

	return pool
}

func ensureSetup(t testing.TB) {
	t.Helper()

	// Skip early when Docker is not running instead of waiting for the
	// container start to time out. The check only matters until the shared
	// container is up.
	if !started.Load() {
		if tt, ok := t.(*testing.T); ok {
			testcontainers.SkipIfProviderIsNotHealthy(tt)
		}
	}

	setupOnce.Do(provision)

	if dockerErr != nil {
		t.Skipf("docker tidak tersedia: %v", dockerErr)
	}
	if setupErr != nil {
		t.Fatalf("siapkan database uji: %v", setupErr)
	}
}

// provision starts the shared container and builds both template databases.
// It runs once per test binary.
func provision() {
	ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
	defer cancel()

	container, err := tcpostgres.Run(ctx, postgresImage,
		tcpostgres.WithDatabase(adminDatabase),
		tcpostgres.WithUsername(dbUser),
		tcpostgres.WithPassword(dbPassword),
		// The standard Postgres wait strategy: the readiness log line is
		// expected twice because the entrypoint restarts the server once,
		// plus a check that the mapped port accepts connections. Both waits
		// share the default 60s startup timeout.
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		dockerErr = err
		setupErr = fmt.Errorf("jalankan container postgres: %w", err)
		return
	}
	// The container is intentionally left running for the whole test binary;
	// Ryuk removes it once the test process exits.

	adminURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		setupErr = fmt.Errorf("ambil URL container: %w", err)
		return
	}

	adminConn, err = pgx.Connect(ctx, adminURL)
	if err != nil {
		setupErr = fmt.Errorf("hubungkan admin database: %w", err)
		return
	}

	if err := createDatabase(ctx, templateDatabase, ""); err != nil {
		setupErr = err
		return
	}
	if err := migrateGoose(ctx, databaseURL(templateDatabase)); err != nil {
		setupErr = err
		return
	}
	if err := migrateRiver(ctx, databaseURL(templateDatabase)); err != nil {
		setupErr = err
		return
	}

	// gk_template must have no active connections, which migrateGoose and
	// migrateRiver guarantee by closing their connections before returning.
	if err := createDatabase(ctx, templateSeededDatabase, templateDatabase); err != nil {
		setupErr = err
		return
	}
	if err := seedDatabase(ctx, databaseURL(templateSeededDatabase)); err != nil {
		setupErr = err
		return
	}

	started.Store(true)
}

func createDatabase(ctx context.Context, name, template string) error {
	stmt := "CREATE DATABASE " + pgx.Identifier{name}.Sanitize()
	if template != "" {
		stmt += " TEMPLATE " + pgx.Identifier{template}.Sanitize()
	}

	dbMu.Lock()
	defer dbMu.Unlock()

	if _, err := adminConn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("buat database %s: %w", name, err)
	}
	return nil
}

func dropDatabase(t testing.TB, name string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
	defer cancel()

	dbMu.Lock()
	defer dbMu.Unlock()

	stmt := "DROP DATABASE IF EXISTS " + pgx.Identifier{name}.Sanitize() + " WITH (FORCE)"
	if _, err := adminConn.Exec(ctx, stmt); err != nil {
		t.Errorf("hapus database %s: %v", name, err)
	}
}

func migrateGoose(ctx context.Context, url string) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("dialek goose: %w", err)
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("buka database migrasi: %w", err)
	}
	defer db.Close()

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

func migrateRiver(ctx context.Context, url string) error {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return fmt.Errorf("river buka database: %w", err)
	}
	defer pool.Close()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("river migrate up: %w", err)
	}
	return nil
}

func seedDatabase(ctx context.Context, url string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("buka database seed: %w", err)
	}
	defer conn.Close(ctx)

	entries, err := fs.ReadDir(seeds.FS, ".")
	if err != nil {
		return fmt.Errorf("baca seed: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := seeds.FS.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("baca seed %s: %w", entry.Name(), err)
		}
		// The seed files contain multiple statements wrapped in BEGIN/COMMIT,
		// so they must be sent through the simple query protocol.
		if _, err := conn.PgConn().Exec(ctx, string(content)).ReadAll(); err != nil {
			return fmt.Errorf("jalankan seed %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func newDatabaseName(t testing.TB) string {
	t.Helper()

	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		t.Fatalf("buat nama database uji: %v", err)
	}
	return "t_" + hex.EncodeToString(buf[:])
}

func databaseURL(dbName string) string {
	u, err := url.Parse(adminURL)
	if err != nil {
		panic("testutil: invalid admin database URL: " + err.Error())
	}
	u.Path = "/" + dbName
	return u.String()
}
