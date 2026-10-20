// Command migrate-old memindahkan data v1 ke v2 sesuai PRD §16.
//
// Prinsip: idempotent (tabel legacy_id_map), dry-run dulu, tidak ada overwrite
// buta URL lama (masuk legacy_urls untuk redirect 301). Tool ini membaca dump
// v1 yang sudah di-restore ke database staging terpisah, lalu menulis ke v2.
//
// Skema v1 DITEMUKAN saat runtime (introspeksi information_schema), jadi tool
// tidak pecah bila nama kolom v1 sedikit berbeda. Yang diasumsikan hanya
// tabel inti v1: users, portfolios, content_blocks (atau portfolio_blocks),
// portfolio_likes, follows, comments, notifications, assessment_metrics
// (atau metrics), assessments, feedback, changelogs.
//
// Contoh:
//
//	go run ./cmd/tools/migrate-old --v1 "postgres://u:p@h:5432/v1db?sslmode=disable" \
//	  --v2 "$DATABASE_URL" --dry-run
//	go run ./cmd/tools/migrate-old --v1 ... --v2 ... --execute
//
// Storage dipindahkan terpisah dengan mc/rclone mirror (PRD §16.3), lalu:
//
//	go run ./cmd/tools/migrate-old --v1 ... --v2 ... --execute --migrate-media
//
// yang mendaftarkan objek storage menjadi media_assets + legacy_urls.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"grafikarsa/api/internal/config"
)

type options struct {
	v1           string
	v2           string
	dryRun       bool
	migrateMedia bool
	storageHost  string
	batchSize    int
}

type stats struct {
	Users, UsersSkipped      int
	Portfolios, Blocks       int
	BlocksQuarantined        int
	Likes, Follows, Comments int
	Notifications            int
	Assessments              int
	Feedback, Changelogs     int
	MediaRegistered          int
	// Akademik v1 -> v2.
	Majors, MajorsCreated int
	Classes               int
	Enrollments           int
	// MediaFallback: URL yang TIDAK bisa didaftarkan sebagai media_assets
	// (hanya masuk legacy_urls tanpa media_id). Ini berarti ada objek yang
	// tidak terikat ke media — harus menggagalkan migrasi, bukan cuma
	// logged sebagai warning.
	MediaFallback int
	Skipped       map[string]int
}

func newStats() *stats { return &stats{Skipped: map[string]int{}} }

func main() {
	var o options
	var execute bool
	flag.StringVar(&o.v1, "v1", "", "DSN database v1 (staging/restore dump)")
	flag.StringVar(&o.v2, "v2", "", "DSN database v2 (default: DATABASE_URL dari env)")
	flag.BoolVar(&o.dryRun, "dry-run", true, "Hanya hitung + validasi, tanpa menulis")
	flag.BoolVar(&execute, "execute", false, "Tulis ke v2 (menonaktifkan dry-run)")
	flag.BoolVar(&o.migrateMedia, "migrate-media", false, "Daftarkan objek storage jadi media_assets + legacy_urls")
	flag.StringVar(&o.storageHost, "storage-host", "", "Host S3 baru (untuk verifikasi objek, opsional)")
	flag.IntVar(&o.batchSize, "batch", 500, "Ukuran batch insert")
	flag.Parse()

	if execute {
		o.dryRun = false
	}
	if o.v1 == "" {
		fatal("flag --v1 wajib diisi (DSN database v1 hasil restore dump)")
	}
	if o.v2 == "" {
		if cfg, err := config.Load(); err == nil {
			o.v2 = cfg.DatabaseURL()
		}
		if o.v2 == "" {
			fatal("flag --v2 wajib diisi atau DATABASE_URL dari env")
		}
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx := context.Background()

	v1, err := pgxpool.New(ctx, o.v1)
	if err != nil {
		fatal("konek v1: %v", err)
	}
	defer v1.Close()
	v2, err := pgxpool.New(ctx, o.v2)
	if err != nil {
		fatal("konek v2: %v", err)
	}
	defer v2.Close()

	introspect, err := inspectV1(ctx, v1)
	if err != nil {
		fatal("introspeksi v1: %v", err)
	}
	names := []string{}
	for t := range introspect.tables {
		names = append(names, t)
	}
	slog.Info("skema v1 terdeteksi", "tables", strings.Join(names, ","))

	st := newStats()
	// Akademik (jurusan -> majors, kelas -> classes) harus lebih dulu:
	// user menunjuk kelas lewat UUID v1, dan mapping-nya baru ada setelah
	// langkah ini. Kalau dibalik, semua user dapat class_id/major_id NULL.
	classMajor, err := migrateAcademics(ctx, v1, v2, introspect, o, st)
	if err != nil {
		fatal("migrate academics: %v", err)
	}
	if err := migrateUsers(ctx, v1, v2, introspect, o, st, classMajor); err != nil {
		fatal("migrate users: %v", err)
	}
	// Enrollments setelah users: enrollment meng_acuan user (foreign key),
	// jadi user-nya harus sudah ada di v2.
	if err := migrateEnrollments(ctx, v1, v2, introspect, o, st); err != nil {
		slog.Warn("migrate enrollments dilewati sebagian", "err", err)
	}
	if err := migratePortfolios(ctx, v1, v2, introspect, o, st); err != nil {
		fatal("migrate portfolios: %v", err)
	}
	if err := migrateSocial(ctx, v1, v2, introspect, o, st); err != nil {
		slog.Warn("migrate social dilewati sebagian", "err", err)
	}
	if err := migrateAssessments(ctx, v1, v2, introspect, o, st); err != nil {
		slog.Warn("migrate assessments dilewati sebagian", "err", err)
	}
	if err := migrateFeedbackChangelog(ctx, v1, v2, introspect, o, st); err != nil {
		slog.Warn("migrate feedback/changelog dilewati sebagian", "err", err)
	}
	if o.migrateMedia {
		if err := migrateMediaRefs(ctx, v1, v2, introspect, o, st); err != nil {
			// Fase media tidak boleh "dilewati sebagian": kalau gagal,
			// operator harus tahu dan memperbaiki sebelum go-live. Kalau
			// hanya di-warn, migrasi tetap exit 0 padahal sebagian objek
			// tidak terikat ke media — dan itu baru ketahuan setelah
			// pengguna membuka karya yang gambarnya hilang.
			fatal("migrate media gagal: %v", err)
		}
	}

	slog.Info("migrasi selesai",
		"dry_run", o.dryRun,
		"users", st.Users, "users_skipped", st.UsersSkipped,
		"majors", st.Majors, "majors_baru", st.MajorsCreated,
		"classes", st.Classes, "enrollments", st.Enrollments,
		"portfolios", st.Portfolios, "blocks", st.Blocks,
		"blocks_quarantined", st.BlocksQuarantined,
		"likes", st.Likes, "follows", st.Follows, "comments", st.Comments,
		"notifications", st.Notifications, "assessments", st.Assessments,
		"feedback", st.Feedback, "changelogs", st.Changelogs,
		"media_registered", st.MediaRegistered,
		"skipped", st.Skipped)

	if o.dryRun {
		fmt.Println("DRY-RUN OK: tidak ada baris yang ditulis. Jalankan ulang dengan --execute untuk menulis.")
		return
	}
	if err := verifyMigration(ctx, v1, v2, st); err != nil {
		fatal("verifikasi pasca-migrasi GAGAL: %v", err)
	}
	fmt.Println("MIGRATE OK: verifikasi pasca-migrasi lolos.")
}

func fatal(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// ---- introspeksi skema v1 ----

type v1schema struct {
	tables map[string]bool
	cols   map[string]map[string]bool
}

func inspectV1(ctx context.Context, db *pgxpool.Pool) (*v1schema, error) {
	s := &v1schema{tables: map[string]bool{}, cols: map[string]map[string]bool{}}
	// View ikut diintrospeksi: di v1 data assessment ada di view
	// v_portfolio_assessments (901 baris), bukan di tabel biasa.
	rows, err := db.Query(ctx, `SELECT table_name FROM information_schema.tables
		WHERE table_schema='public' AND table_type IN ('BASE TABLE','VIEW')`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, n)
		s.tables[n] = true
	}
	rows.Close()
	for _, t := range names {
		crows, err := db.Query(ctx, `SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=$1`, t)
		if err != nil {
			return nil, err
		}
		m := map[string]bool{}
		for crows.Next() {
			var c string
			if err := crows.Scan(&c); err != nil {
				crows.Close()
				return nil, err
			}
			m[c] = true
		}
		crows.Close()
		s.cols[t] = m
	}
	return s, nil
}

func (s *v1schema) has(table string, cols ...string) bool {
	if !s.tables[table] {
		return false
	}
	for _, c := range cols {
		if !s.cols[table][c] {
			return false
		}
	}
	return true
}

// firstTableWithRows memilih kandidat tabel pertama yang benar-benar punya
// baris.
//
// Ini perlu untuk assessment: di v1 prod ada tabel 'portfolio_assessments'
// yang KOSONG dan view 'v_portfolio_assessments' yang berisi 901 baris.
// firstTable akan memilih yang kosong lebih dulu, jadi assessment-nya hilang.
func (s *v1schema) firstTableWithRows(ctx context.Context, db *pgxpool.Pool, cands ...string) string {
	fallback := ""
	for _, t := range cands {
		if !s.tables[t] {
			continue
		}
		if fallback == "" {
			fallback = t
		}
		var n int64
		if err := db.QueryRow(ctx, "SELECT count(*) FROM "+t).Scan(&n); err == nil && n > 0 {
			return t
		}
	}
	return fallback
}

func (s *v1schema) firstTable(cands ...string) string {
	for _, t := range cands {
		if s.tables[t] {
			return t
		}
	}
	return ""
}

func colOr(s *v1schema, table string, cands ...string) string {
	for _, c := range cands {
		if s.cols[table][c] {
			return c
		}
	}
	return ""
}

// dryRunMap menyimpan pemetaan id selama dry-run.
//
// Peta ini TIDAK bisa ditulis ke legacy_id_map karena dry-run tidak boleh
// menulis apa pun ke database target. Tapi tanpa peta, semua referensi
// (like/follow/comment/notification) selalu gagal dipetakan, sehingga
// dry-run melaporkan 0 relasional dimigrasi — memberi keyakinan palsu
// padahal datanya sebenarnya bisa dimigrasi.
//
// Peta di memori membuat dry-run benar-benar memvalidasi data relasional.
var dryRunMap sync.Map

// jsonText mengubah nilai jsonb yang dibaca driver menjadi teks JSON.
//
// pgx membaca kolom jsonb ke `any` sebagai map[string]any. fmt.Sprint atas
// map itu menghasilkan "map[content:...]" — BUKAN JSON. Akibatnya
// json.Unmarshal gagal: isi text block jadi sampah ("map[content:]"), dan
// semua tipe block lain masuk karantina karena payload-nya dianggap rusak.
func jsonText(v any) string {
	switch t := v.(type) {
	case nil:
		return "{}"
	case string:
		return t
	case []byte:
		return string(t)
	case map[string]any:
		b, err := json.Marshal(t)
		if err != nil {
			return "{}"
		}
		return string(b)
	case map[string]string:
		b, err := json.Marshal(t)
		if err != nil {
			return "{}"
		}
		return string(b)
	default:
		return fmt.Sprint(v)
	}
}

// uuidString menormalkan nilai id yang dibaca dari driver menjadi string UUID.
//
// pgx membaca kolom uuid ke `any` sebagai array 16 byte, jadi fmt.Sprint
// menghasilkan "[0 43 33 205 ...]" — bukan UUID. Nilai itu tidak bisa
// dipakai di query (invalid input syntax for type uuid) dan juga membuat
// legacy_id_map storing tidak terbaca manusia.
func uuidString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case uuid.UUID:
		return t.String()
	case pgtype.UUID:
		if t.Valid {
			return uuid.UUID(t.Bytes).String()
		}
		return ""
	case [16]byte:
		return uuid.UUID(t).String()
	case []byte:
		if len(t) == 16 {
			return uuid.UUID(t).String()
		}
		return string(t)
	default:
		return fmt.Sprint(v)
	}
}

// mapID mencatat / membaca mapping idempotent.
func mappedID(ctx context.Context, v2 *pgxpool.Pool, srcTable, srcID, targetTable string) (string, bool) {
	// Dry-run: pakai peta di memori (tidak ada yang ditulis ke database).
	if v, ok := dryRunMap.Load(srcTable + "|" + srcID + "|" + targetTable); ok {
		id, _ := v.(string)
		return id, id != ""
	}
	var id string
	err := v2.QueryRow(ctx, `SELECT target_id::text FROM legacy_id_map WHERE source_table=$1 AND source_id=$2 AND target_table=$3`, srcTable, srcID, targetTable).Scan(&id)
	return id, err == nil
}

func saveMap(ctx context.Context, v2 *pgxpool.Pool, dry bool, srcTable, srcID, targetTable, targetID string) error {
	if dry {
		// Dry-run: simpan pemetaan di memori supaya referensi antar tabel
		// (like/follow/comment/notification) tetap bisa diverifikasi.
		dryRunMap.Store(srcTable+"|"+srcID+"|"+targetTable, targetID)
		return nil
	}
	_, err := v2.Exec(ctx, `INSERT INTO legacy_id_map (source_table, source_id, target_table, target_id) VALUES ($1,$2,$3,$4)
		ON CONFLICT (source_table, source_id) DO NOTHING`, srcTable, srcID, targetTable, targetID)
	return err
}

func sha256str(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// nowYear dipakai untuk aturan alumni (tahun_masuk+3 <= now).
func nowYear() int { return time.Now().Year() }

var _ = pgx.ErrNoRows
