package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/grafikarsa/backend/internal/config"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		printMenu()
		fmt.Print("Pilih menu: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			createDatabase(cfg)
		case "2":
			migrateSchema(cfg)
		case "3":
			migrateFresh(cfg)
		case "4":
			truncateTables(cfg)
		case "5":
			seedData(cfg)
		case "6":
			deleteDatabase(cfg)
		case "0":
			fmt.Println("Keluar...")
			os.Exit(0)
		default:
			fmt.Println("Pilihan tidak valid")
		}

		fmt.Println()
		fmt.Print("Tekan Enter untuk melanjutkan...")
		reader.ReadString('\n')
	}
}

func printMenu() {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("    GRAFIKARSA DATABASE CLI MANAGER")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("1. Buat Database (jika belum ada) + Migrasi Schema")
	fmt.Println("2. Migrasi Schema (tanpa buat database)")
	fmt.Println("3. Migrate Fresh (drop semua + migrasi ulang)")
	fmt.Println("4. Truncate Tables (kecuali reference data)")
	fmt.Println("5. Seed Data (generate dummy data)")
	fmt.Println("6. Hapus Database")
	fmt.Println("0. Keluar")
	fmt.Println()
	fmt.Println("----------------------------------------")
}

func getPostgresConn(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode,
	)
	return sql.Open("postgres", connStr)
}

func getDBConn(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode,
	)
	return sql.Open("postgres", connStr)
}

func databaseExists(cfg *config.Config) (bool, error) {
	db, err := getPostgresConn(cfg)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.Database.Name).Scan(&exists)
	return exists, err
}

func createDatabase(cfg *config.Config) {
	fmt.Println()
	fmt.Println("--- Buat Database + Migrasi Schema ---")

	exists, err := databaseExists(cfg)
	if err != nil {
		fmt.Printf("Error cek database: %v\n", err)
		return
	}

	if exists {
		fmt.Printf("Database '%s' sudah ada.\n", cfg.Database.Name)
		fmt.Print("Lanjutkan migrasi schema? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(input)) != "y" {
			fmt.Println("Dibatalkan.")
			return
		}
	} else {
		db, err := getPostgresConn(cfg)
		if err != nil {
			fmt.Printf("Error koneksi: %v\n", err)
			return
		}
		defer db.Close()

		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.Database.Name))
		if err != nil {
			fmt.Printf("Error buat database: %v\n", err)
			return
		}
		fmt.Printf("Database '%s' berhasil dibuat.\n", cfg.Database.Name)
	}

	migrateSchema(cfg)
}

func migrateSchema(cfg *config.Config) {
	fmt.Println()
	fmt.Println("--- Migrasi Schema ---")

	db, err := getDBConn(cfg)
	if err != nil {
		fmt.Printf("Error koneksi: %v\n", err)
		return
	}
	defer db.Close()

	fmt.Println("Membuat extensions...")
	if err := createExtensions(db); err != nil {
		fmt.Printf("Error buat extensions: %v\n", err)
		return
	}

	fmt.Println("Membuat enum types...")
	if err := createEnumTypes(db); err != nil {
		fmt.Printf("Error buat enum types: %v\n", err)
		return
	}

	fmt.Println("Membuat tables...")
	if err := createTables(db); err != nil {
		fmt.Printf("Error buat tables: %v\n", err)
		return
	}

	fmt.Println("Membuat indexes...")
	if err := createIndexes(db); err != nil {
		fmt.Printf("Error buat indexes: %v\n", err)
		return
	}

	fmt.Println("Membuat functions dan triggers...")
	if err := createFunctionsAndTriggers(db); err != nil {
		fmt.Printf("Error buat functions/triggers: %v\n", err)
		return
	}

	fmt.Println("Membuat views...")
	if err := createViews(db); err != nil {
		fmt.Printf("Error buat views: %v\n", err)
		return
	}

	fmt.Println("Memasukkan seed data...")
	if err := seedReferenceData(db); err != nil {
		fmt.Printf("Error seed data: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Migrasi schema selesai!")
}

func migrateFresh(cfg *config.Config) {
	fmt.Println()
	fmt.Println("--- Migrate Fresh ---")
	fmt.Println("PERINGATAN: Semua data akan dihapus!")
	fmt.Print("Ketik 'FRESH' untuk konfirmasi: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(input) != "FRESH" {
		fmt.Println("Dibatalkan.")
		return
	}

	db, err := getDBConn(cfg)
	if err != nil {
		fmt.Printf("Error koneksi: %v\n", err)
		return
	}
	defer db.Close()

	fmt.Println("Menghapus semua objects...")
	if err := dropAllObjects(db); err != nil {
		fmt.Printf("Error drop objects: %v\n", err)
		return
	}

	fmt.Println("Memulai migrasi ulang...")
	migrateSchema(cfg)
}

func truncateTables(cfg *config.Config) {
	fmt.Println()
	fmt.Println("--- Truncate Tables ---")
	fmt.Println("Data berikut akan DIHAPUS:")
	fmt.Println("- users, portfolios, content_blocks")
	fmt.Println("- follows, portfolio_likes, portfolio_tags")
	fmt.Println("- refresh_tokens, token_blacklist")
	fmt.Println("- student_class_history, user_social_links")
	fmt.Println()
	fmt.Println("Data berikut akan DIPERTAHANKAN:")
	fmt.Println("- jurusan, tahun_ajaran, kelas, tags, app_settings")
	fmt.Println()
	fmt.Print("Ketik 'TRUNCATE' untuk konfirmasi: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(input) != "TRUNCATE" {
		fmt.Println("Dibatalkan.")
		return
	}

	db, err := getDBConn(cfg)
	if err != nil {
		fmt.Printf("Error koneksi: %v\n", err)
		return
	}
	defer db.Close()

	tablesToTruncate := []string{
		"token_blacklist",
		"refresh_tokens",
		"portfolio_likes",
		"content_blocks",
		"portfolio_tags",
		"portfolios",
		"follows",
		"student_class_history",
		"user_social_links",
		"users",
	}

	for _, table := range tablesToTruncate {
		fmt.Printf("Truncating %s...\n", table)
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			fmt.Printf("Error truncate %s: %v\n", table, err)
		}
	}

	fmt.Println()
	fmt.Println("Truncate selesai!")
}

func deleteDatabase(cfg *config.Config) {
	fmt.Println()
	fmt.Println("--- Hapus Database ---")
	fmt.Printf("PERINGATAN: Database '%s' akan dihapus permanen!\n", cfg.Database.Name)
	fmt.Print("Ketik nama database untuk konfirmasi: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(input) != cfg.Database.Name {
		fmt.Println("Nama database tidak cocok. Dibatalkan.")
		return
	}

	db, err := getPostgresConn(cfg)
	if err != nil {
		fmt.Printf("Error koneksi: %v\n", err)
		return
	}
	defer db.Close()

	// Terminate existing connections
	_, _ = db.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()
	`, cfg.Database.Name))

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.Database.Name))
	if err != nil {
		fmt.Printf("Error hapus database: %v\n", err)
		return
	}

	fmt.Printf("Database '%s' berhasil dihapus.\n", cfg.Database.Name)
}

func dropAllObjects(db *sql.DB) error {
	// Drop views
	_, _ = db.Exec("DROP VIEW IF EXISTS v_published_portfolios CASCADE")
	_, _ = db.Exec("DROP VIEW IF EXISTS v_user_profiles CASCADE")

	// Drop tables in order (respecting foreign keys)
	tables := []string{
		"token_blacklist",
		"refresh_tokens",
		"portfolio_likes",
		"content_blocks",
		"portfolio_tags",
		"portfolios",
		"follows",
		"student_class_history",
		"user_social_links",
		"users",
		"kelas",
		"tahun_ajaran",
		"jurusan",
		"tags",
		"app_settings",
	}

	for _, table := range tables {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
	}

	// Drop enum types
	enums := []string{"user_role", "portfolio_status", "content_block_type", "social_platform"}
	for _, enum := range enums {
		_, _ = db.Exec(fmt.Sprintf("DROP TYPE IF EXISTS %s CASCADE", enum))
	}

	return nil
}

func createExtensions(db *sql.DB) error {
	extensions := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		`CREATE EXTENSION IF NOT EXISTS "pg_trgm"`,
	}

	for _, ext := range extensions {
		if _, err := db.Exec(ext); err != nil {
			return fmt.Errorf("extension error: %v", err)
		}
	}
	return nil
}

func createEnumTypes(db *sql.DB) error {
	enums := []string{
		`DO $$ BEGIN
			CREATE TYPE user_role AS ENUM ('student', 'alumni', 'admin');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
			CREATE TYPE portfolio_status AS ENUM ('draft', 'pending_review', 'rejected', 'published', 'archived');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
			CREATE TYPE content_block_type AS ENUM ('text', 'image', 'table', 'youtube', 'button', 'embed', 'figma', 'canva', 'ppt', 'pdf', 'doc');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
			CREATE TYPE social_platform AS ENUM (
				'facebook', 'instagram', 'github', 'linkedin', 'twitter',
				'personal_website', 'tiktok', 'youtube', 'behance', 'dribbble',
				'threads', 'bluesky', 'medium', 'gitlab'
			);
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	}

	for _, enum := range enums {
		if _, err := db.Exec(enum); err != nil {
			return fmt.Errorf("enum error: %v", err)
		}
	}
	return nil
}

func createTables(db *sql.DB) error {
	tables := []string{
		// Jurusan
		`CREATE TABLE IF NOT EXISTS jurusan (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			nama VARCHAR(100) NOT NULL,
			kode VARCHAR(10) NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ,
			CONSTRAINT jurusan_kode_lowercase CHECK (kode = LOWER(kode)),
			CONSTRAINT jurusan_kode_alpha CHECK (kode ~ '^[a-z]+$')
		)`,

		// Tahun Ajaran
		`CREATE TABLE IF NOT EXISTS tahun_ajaran (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tahun_mulai INTEGER NOT NULL UNIQUE,
			is_active BOOLEAN NOT NULL DEFAULT FALSE,
			promotion_month SMALLINT NOT NULL DEFAULT 7,
			promotion_day SMALLINT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ,
			CONSTRAINT tahun_ajaran_tahun_valid CHECK (tahun_mulai >= 2000 AND tahun_mulai <= 2100),
			CONSTRAINT tahun_ajaran_promotion_month_valid CHECK (promotion_month BETWEEN 1 AND 12),
			CONSTRAINT tahun_ajaran_promotion_day_valid CHECK (promotion_day BETWEEN 1 AND 31)
		)`,

		// Kelas
		`CREATE TABLE IF NOT EXISTS kelas (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tahun_ajaran_id UUID NOT NULL REFERENCES tahun_ajaran(id) ON DELETE RESTRICT,
			jurusan_id UUID NOT NULL REFERENCES jurusan(id) ON DELETE RESTRICT,
			tingkat SMALLINT NOT NULL,
			rombel CHAR(1) NOT NULL,
			nama VARCHAR(20) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ,
			CONSTRAINT kelas_tingkat_valid CHECK (tingkat IN (10, 11, 12)),
			CONSTRAINT kelas_rombel_valid CHECK (rombel ~ '^[A-Z]$'),
			CONSTRAINT kelas_unique_per_tahun UNIQUE (tahun_ajaran_id, jurusan_id, tingkat, rombel)
		)`,

		// Tags
		`CREATE TABLE IF NOT EXISTS tags (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			nama VARCHAR(50) NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		)`,

		// Users
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			username VARCHAR(30) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			nama VARCHAR(100) NOT NULL,
			bio TEXT,
			avatar_url TEXT,
			banner_url TEXT,
			role user_role NOT NULL DEFAULT 'student',
			nisn VARCHAR(20),
			nis VARCHAR(30),
			kelas_id UUID REFERENCES kelas(id) ON DELETE SET NULL,
			tahun_masuk INTEGER,
			tahun_lulus INTEGER,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			last_login_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ,
			CONSTRAINT users_nisn_numeric CHECK (nisn IS NULL OR nisn ~ '^\d+$'),
			CONSTRAINT users_tahun_masuk_valid CHECK (tahun_masuk IS NULL OR (tahun_masuk >= 2000 AND tahun_masuk <= 2100)),
			CONSTRAINT users_tahun_lulus_valid CHECK (tahun_lulus IS NULL OR tahun_lulus >= tahun_masuk)
		)`,

		// User Social Links
		`CREATE TABLE IF NOT EXISTS user_social_links (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			platform social_platform NOT NULL,
			url TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT user_social_links_unique UNIQUE (user_id, platform)
		)`,

		// Student Class History
		`CREATE TABLE IF NOT EXISTS student_class_history (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			kelas_id UUID NOT NULL REFERENCES kelas(id) ON DELETE RESTRICT,
			tahun_ajaran_id UUID NOT NULL REFERENCES tahun_ajaran(id) ON DELETE RESTRICT,
			is_current BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT student_class_history_unique UNIQUE (user_id, kelas_id, tahun_ajaran_id)
		)`,

		// Refresh Tokens
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(64) NOT NULL UNIQUE,
			family_id UUID NOT NULL,
			device_info JSONB,
			ip_address INET,
			is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
			revoked_at TIMESTAMPTZ,
			revoked_reason VARCHAR(100),
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_used_at TIMESTAMPTZ
		)`,

		// Token Blacklist
		`CREATE TABLE IF NOT EXISTS token_blacklist (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			jti VARCHAR(64) NOT NULL UNIQUE,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL,
			blacklisted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			reason VARCHAR(100)
		)`,

		// Follows
		`CREATE TABLE IF NOT EXISTS follows (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT follows_no_self_follow CHECK (follower_id != following_id),
			CONSTRAINT follows_unique UNIQUE (follower_id, following_id)
		)`,

		// Portfolios
		`CREATE TABLE IF NOT EXISTS portfolios (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			judul VARCHAR(200) NOT NULL,
			slug VARCHAR(250) NOT NULL,
			thumbnail_url TEXT,
			status portfolio_status NOT NULL DEFAULT 'draft',
			admin_review_note TEXT,
			reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
			reviewed_at TIMESTAMPTZ,
			published_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ,
			CONSTRAINT portfolios_slug_unique UNIQUE (user_id, slug)
		)`,

		// Portfolio Tags
		`CREATE TABLE IF NOT EXISTS portfolio_tags (
			portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
			tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (portfolio_id, tag_id)
		)`,

		// Content Blocks
		`CREATE TABLE IF NOT EXISTS content_blocks (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
			block_type content_block_type NOT NULL,
			block_order INTEGER NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT content_blocks_order_positive CHECK (block_order >= 0)
		)`,

		// Portfolio Likes
		`CREATE TABLE IF NOT EXISTS portfolio_likes (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, portfolio_id)
		)`,

		// App Settings
		`CREATE TABLE IF NOT EXISTS app_settings (
			key VARCHAR(100) PRIMARY KEY,
			value JSONB NOT NULL,
			description TEXT,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_by UUID REFERENCES users(id) ON DELETE SET NULL
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("table error: %v", err)
		}
	}
	return nil
}

func createIndexes(db *sql.DB) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_jurusan_kode ON jurusan(kode) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tahun_ajaran_active ON tahun_ajaran(is_active) WHERE is_active = TRUE AND deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_kelas_tahun_ajaran ON kelas(tahun_ajaran_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_kelas_jurusan ON kelas(jurusan_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_kelas_nama ON kelas(nama) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_tags_nama ON tags(nama) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_users_kelas ON users(kelas_id) WHERE deleted_at IS NULL AND kelas_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_users_nama_trgm ON users USING gin(nama gin_trgm_ops) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_user_social_links_user ON user_social_links(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_student_class_history_user ON student_class_history(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_student_class_history_current ON student_class_history(user_id, is_current) WHERE is_current = TRUE`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id) WHERE is_revoked = FALSE`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash) WHERE is_revoked = FALSE`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family ON refresh_tokens(family_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE is_revoked = FALSE`,
		`CREATE INDEX IF NOT EXISTS idx_token_blacklist_jti ON token_blacklist(jti)`,
		`CREATE INDEX IF NOT EXISTS idx_token_blacklist_expires ON token_blacklist(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_follows_follower ON follows(follower_id)`,
		`CREATE INDEX IF NOT EXISTS idx_follows_following ON follows(following_id)`,
		`CREATE INDEX IF NOT EXISTS idx_portfolios_user ON portfolios(user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_portfolios_status ON portfolios(status) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_portfolios_published ON portfolios(published_at DESC) WHERE status = 'published' AND deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_portfolios_pending ON portfolios(created_at) WHERE status = 'pending_review' AND deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_portfolios_slug ON portfolios(slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_portfolio_tags_tag ON portfolio_tags(tag_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_blocks_portfolio ON content_blocks(portfolio_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_blocks_order ON content_blocks(portfolio_id, block_order)`,
		`CREATE INDEX IF NOT EXISTS idx_portfolio_likes_portfolio ON portfolio_likes(portfolio_id)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			// Ignore duplicate index errors
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("index error: %v", err)
			}
		}
	}
	return nil
}

func createFunctionsAndTriggers(db *sql.DB) error {
	functions := []string{
		// Generate nama kelas
		`CREATE OR REPLACE FUNCTION generate_nama_kelas()
		RETURNS TRIGGER AS $$
		DECLARE
			v_kode_jurusan VARCHAR(10);
			v_tingkat_romawi VARCHAR(4);
		BEGIN
			SELECT UPPER(kode) INTO v_kode_jurusan FROM jurusan WHERE id = NEW.jurusan_id;
			v_tingkat_romawi := CASE NEW.tingkat
				WHEN 10 THEN 'X'
				WHEN 11 THEN 'XI'
				WHEN 12 THEN 'XII'
			END;
			NEW.nama := v_tingkat_romawi || '-' || v_kode_jurusan || '-' || NEW.rombel;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Generate portfolio slug
		`CREATE OR REPLACE FUNCTION generate_portfolio_slug()
		RETURNS TRIGGER AS $$
		DECLARE
			v_base_slug VARCHAR(250);
			v_slug VARCHAR(250);
			v_counter INTEGER := 0;
		BEGIN
			v_base_slug := LOWER(TRIM(NEW.judul));
			v_base_slug := REGEXP_REPLACE(v_base_slug, '[^a-z0-9\s-]', '', 'g');
			v_base_slug := REGEXP_REPLACE(v_base_slug, '\s+', '-', 'g');
			v_base_slug := REGEXP_REPLACE(v_base_slug, '-+', '-', 'g');
			v_base_slug := TRIM(BOTH '-' FROM v_base_slug);
			v_base_slug := LEFT(v_base_slug, 200);
			v_slug := v_base_slug;
			WHILE EXISTS (
				SELECT 1 FROM portfolios 
				WHERE user_id = NEW.user_id 
				AND slug = v_slug 
				AND id != COALESCE(NEW.id, uuid_generate_v4())
				AND deleted_at IS NULL
			) LOOP
				v_counter := v_counter + 1;
				v_slug := v_base_slug || '-' || v_counter;
			END LOOP;
			NEW.slug := v_slug;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Set published_at
		`CREATE OR REPLACE FUNCTION set_portfolio_published_at()
		RETURNS TRIGGER AS $$
		BEGIN
			IF NEW.status = 'published' AND (OLD.status IS NULL OR OLD.status != 'published') THEN
				NEW.published_at := NOW();
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Update updated_at
		`CREATE OR REPLACE FUNCTION update_updated_at()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at := NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Ensure single active tahun_ajaran
		`CREATE OR REPLACE FUNCTION ensure_single_active_tahun_ajaran()
		RETURNS TRIGGER AS $$
		BEGIN
			IF NEW.is_active = TRUE THEN
				UPDATE tahun_ajaran SET is_active = FALSE 
				WHERE id != NEW.id AND is_active = TRUE AND deleted_at IS NULL;
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Sync student class history
		`CREATE OR REPLACE FUNCTION sync_student_class_history()
		RETURNS TRIGGER AS $$
		DECLARE
			v_tahun_ajaran_id UUID;
		BEGIN
			IF NEW.kelas_id IS NOT NULL AND NEW.role IN ('student', 'alumni') THEN
				SELECT tahun_ajaran_id INTO v_tahun_ajaran_id FROM kelas WHERE id = NEW.kelas_id;
				UPDATE student_class_history SET is_current = FALSE WHERE user_id = NEW.id;
				INSERT INTO student_class_history (user_id, kelas_id, tahun_ajaran_id, is_current)
				VALUES (NEW.id, NEW.kelas_id, v_tahun_ajaran_id, TRUE)
				ON CONFLICT (user_id, kelas_id, tahun_ajaran_id) 
				DO UPDATE SET is_current = TRUE;
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Cleanup expired tokens
		`CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
		RETURNS INTEGER AS $$
		DECLARE
			deleted_count INTEGER;
		BEGIN
			DELETE FROM refresh_tokens WHERE expires_at < NOW();
			GET DIAGNOSTICS deleted_count = ROW_COUNT;
			DELETE FROM token_blacklist WHERE expires_at < NOW();
			RETURN deleted_count;
		END;
		$$ LANGUAGE plpgsql`,
	}

	for _, fn := range functions {
		if _, err := db.Exec(fn); err != nil {
			return fmt.Errorf("function error: %v", err)
		}
	}

	// Create triggers
	triggers := []string{
		`DROP TRIGGER IF EXISTS trg_kelas_generate_nama ON kelas`,
		`CREATE TRIGGER trg_kelas_generate_nama BEFORE INSERT OR UPDATE ON kelas FOR EACH ROW EXECUTE FUNCTION generate_nama_kelas()`,

		`DROP TRIGGER IF EXISTS trg_portfolio_generate_slug ON portfolios`,
		`CREATE TRIGGER trg_portfolio_generate_slug BEFORE INSERT OR UPDATE OF judul ON portfolios FOR EACH ROW EXECUTE FUNCTION generate_portfolio_slug()`,

		`DROP TRIGGER IF EXISTS trg_portfolio_set_published_at ON portfolios`,
		`CREATE TRIGGER trg_portfolio_set_published_at BEFORE UPDATE ON portfolios FOR EACH ROW EXECUTE FUNCTION set_portfolio_published_at()`,

		`DROP TRIGGER IF EXISTS trg_jurusan_updated_at ON jurusan`,
		`CREATE TRIGGER trg_jurusan_updated_at BEFORE UPDATE ON jurusan FOR EACH ROW EXECUTE FUNCTION update_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_tahun_ajaran_updated_at ON tahun_ajaran`,
		`CREATE TRIGGER trg_tahun_ajaran_updated_at BEFORE UPDATE ON tahun_ajaran FOR EACH ROW EXECUTE FUNCTION update_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_kelas_updated_at ON kelas`,
		`CREATE TRIGGER trg_kelas_updated_at BEFORE UPDATE ON kelas FOR EACH ROW EXECUTE FUNCTION update_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_tags_updated_at ON tags`,
		`CREATE TRIGGER trg_tags_updated_at BEFORE UPDATE ON tags FOR EACH ROW EXECUTE FUNCTION update_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_users_updated_at ON users`,
		`CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_user_social_links_updated_at ON user_social_links`,
		`CREATE TRIGGER trg_user_social_links_updated_at BEFORE UPDATE ON user_social_links FOR EACH ROW EXECUTE FUNCTION update_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_portfolios_updated_at ON portfolios`,
		`CREATE TRIGGER trg_portfolios_updated_at BEFORE UPDATE ON portfolios FOR EACH ROW EXECUTE FUNCTION update_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_content_blocks_updated_at ON content_blocks`,
		`CREATE TRIGGER trg_content_blocks_updated_at BEFORE UPDATE ON content_blocks FOR EACH ROW EXECUTE FUNCTION update_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_tahun_ajaran_single_active ON tahun_ajaran`,
		`CREATE TRIGGER trg_tahun_ajaran_single_active AFTER INSERT OR UPDATE OF is_active ON tahun_ajaran FOR EACH ROW WHEN (NEW.is_active = TRUE) EXECUTE FUNCTION ensure_single_active_tahun_ajaran()`,

		`DROP TRIGGER IF EXISTS trg_users_sync_class_history ON users`,
		`CREATE TRIGGER trg_users_sync_class_history AFTER INSERT OR UPDATE OF kelas_id ON users FOR EACH ROW EXECUTE FUNCTION sync_student_class_history()`,
	}

	for _, trg := range triggers {
		if _, err := db.Exec(trg); err != nil {
			return fmt.Errorf("trigger error: %v", err)
		}
	}

	return nil
}

func createViews(db *sql.DB) error {
	views := []string{
		// User profiles view
		`CREATE OR REPLACE VIEW v_user_profiles AS
		SELECT 
			u.id,
			u.username,
			u.email,
			u.nama,
			u.bio,
			u.avatar_url,
			u.banner_url,
			u.role,
			u.nisn,
			u.nis,
			u.tahun_masuk,
			u.tahun_lulus,
			u.is_active,
			u.created_at,
			k.id AS kelas_id,
			k.nama AS kelas_nama,
			k.tingkat AS kelas_tingkat,
			j.id AS jurusan_id,
			j.nama AS jurusan_nama,
			j.kode AS jurusan_kode,
			(SELECT COUNT(*) FROM follows WHERE following_id = u.id) AS follower_count,
			(SELECT COUNT(*) FROM follows WHERE follower_id = u.id) AS following_count,
			(SELECT COUNT(*) FROM portfolios WHERE user_id = u.id AND status = 'published' AND deleted_at IS NULL) AS portfolio_count
		FROM users u
		LEFT JOIN kelas k ON u.kelas_id = k.id AND k.deleted_at IS NULL
		LEFT JOIN jurusan j ON k.jurusan_id = j.id AND j.deleted_at IS NULL
		WHERE u.deleted_at IS NULL`,

		// Published portfolios view
		`CREATE OR REPLACE VIEW v_published_portfolios AS
		SELECT 
			p.id,
			p.user_id,
			p.judul,
			p.slug,
			p.thumbnail_url,
			p.published_at,
			p.created_at,
			p.updated_at,
			u.username AS user_username,
			u.nama AS user_nama,
			u.avatar_url AS user_avatar,
			u.role AS user_role,
			k.nama AS user_kelas,
			j.nama AS user_jurusan,
			(SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = p.id) AS like_count,
			(SELECT ARRAY_AGG(t.nama) FROM portfolio_tags pt JOIN tags t ON pt.tag_id = t.id WHERE pt.portfolio_id = p.id) AS tags
		FROM portfolios p
		JOIN users u ON p.user_id = u.id AND u.deleted_at IS NULL
		LEFT JOIN kelas k ON u.kelas_id = k.id
		LEFT JOIN jurusan j ON k.jurusan_id = j.id
		WHERE p.status = 'published' AND p.deleted_at IS NULL
		ORDER BY p.published_at DESC`,
	}

	for _, view := range views {
		if _, err := db.Exec(view); err != nil {
			return fmt.Errorf("view error: %v", err)
		}
	}
	return nil
}

func seedReferenceData(db *sql.DB) error {
	// Check if data already exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM jurusan").Scan(&count)
	if count > 0 {
		fmt.Println("  Reference data sudah ada, skip seeding.")
		return nil
	}

	seeds := []string{
		// App settings
		`INSERT INTO app_settings (key, value, description) VALUES
		('admin_login_path', '"loginadmin"', 'Path untuk halaman login admin (tanpa slash)')
		ON CONFLICT (key) DO NOTHING`,

		// Tahun ajaran
		`INSERT INTO tahun_ajaran (tahun_mulai, is_active, promotion_month, promotion_day) VALUES
		(2024, FALSE, 7, 1),
		(2025, TRUE, 7, 1)
		ON CONFLICT (tahun_mulai) DO NOTHING`,

		// Jurusan
		`INSERT INTO jurusan (nama, kode) VALUES
		('Rekayasa Perangkat Lunak', 'rpl'),
		('Teknik Komputer dan Jaringan', 'tkj'),
		('Multimedia', 'mm'),
		('Desain Komunikasi Visual', 'dkv'),
		('Animasi', 'ani')
		ON CONFLICT (kode) DO NOTHING`,

		// Tags
		`INSERT INTO tags (nama) VALUES
		('Web Development'),
		('Mobile App'),
		('UI/UX Design'),
		('Graphic Design'),
		('3D Modeling'),
		('Animation'),
		('Video Editing'),
		('Photography'),
		('Illustration'),
		('Game Development')
		ON CONFLICT (nama) DO NOTHING`,
	}

	for _, seed := range seeds {
		if _, err := db.Exec(seed); err != nil {
			return fmt.Errorf("seed error: %v", err)
		}
	}

	fmt.Println("  Reference data berhasil di-seed.")
	return nil
}

// Seeder configuration
type SeederConfig struct {
	Users      int
	Kelas      int
	Portfolios int
	Follows    int
	Likes      int
}

var seederPresets = map[string]SeederConfig{
	"1": {Users: 10, Kelas: 5, Portfolios: 20, Follows: 30, Likes: 50},
	"2": {Users: 50, Kelas: 15, Portfolios: 100, Follows: 200, Likes: 500},
	"3": {Users: 200, Kelas: 30, Portfolios: 500, Follows: 1000, Likes: 3000},
	"4": {Users: 500, Kelas: 60, Portfolios: 2000, Follows: 5000, Likes: 15000},
}

func seedData(cfg *config.Config) {
	fmt.Println()
	fmt.Println("--- Seed Data ---")
	fmt.Println()
	fmt.Println("Pilih jumlah data:")
	fmt.Println("1. Sedikit   - 10 users, 5 kelas, 20 portfolios, 30 follows, 50 likes")
	fmt.Println("2. Sedang    - 50 users, 15 kelas, 100 portfolios, 200 follows, 500 likes")
	fmt.Println("3. Banyak    - 200 users, 30 kelas, 500 portfolios, 1000 follows, 3000 likes")
	fmt.Println("4. Banyak Banget - 500 users, 60 kelas, 2000 portfolios, 5000 follows, 15000 likes")
	fmt.Println("0. Batal")
	fmt.Println()
	fmt.Print("Pilih (1-4): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "0" {
		fmt.Println("Dibatalkan.")
		return
	}

	preset, ok := seederPresets[input]
	if !ok {
		fmt.Println("Pilihan tidak valid.")
		return
	}

	db, err := getDBConn(cfg)
	if err != nil {
		fmt.Printf("Error koneksi: %v\n", err)
		return
	}
	defer db.Close()

	rand.Seed(time.Now().UnixNano())

	fmt.Println()
	fmt.Println("Memulai seeding...")

	// Get reference data IDs
	jurusanIDs, err := getJurusanIDs(db)
	if err != nil || len(jurusanIDs) == 0 {
		fmt.Println("Error: Jalankan migrasi schema terlebih dahulu untuk seed reference data.")
		return
	}

	tahunAjaranIDs, err := getTahunAjaranIDs(db)
	if err != nil || len(tahunAjaranIDs) == 0 {
		fmt.Println("Error: Jalankan migrasi schema terlebih dahulu untuk seed reference data.")
		return
	}

	tagIDs, err := getTagIDs(db)
	if err != nil || len(tagIDs) == 0 {
		fmt.Println("Error: Jalankan migrasi schema terlebih dahulu untuk seed reference data.")
		return
	}

	// Seed kelas
	fmt.Printf("Seeding %d kelas...\n", preset.Kelas)
	kelasIDs, err := seedKelas(db, preset.Kelas, jurusanIDs, tahunAjaranIDs)
	if err != nil {
		fmt.Printf("Error seed kelas: %v\n", err)
		return
	}

	// Seed users
	fmt.Printf("Seeding %d users...\n", preset.Users)
	userIDs, err := seedUsers(db, preset.Users, kelasIDs)
	if err != nil {
		fmt.Printf("Error seed users: %v\n", err)
		return
	}

	// Seed portfolios
	fmt.Printf("Seeding %d portfolios...\n", preset.Portfolios)
	portfolioIDs, err := seedPortfolios(db, preset.Portfolios, userIDs, tagIDs)
	if err != nil {
		fmt.Printf("Error seed portfolios: %v\n", err)
		return
	}

	// Seed follows
	fmt.Printf("Seeding %d follows...\n", preset.Follows)
	if err := seedFollows(db, preset.Follows, userIDs); err != nil {
		fmt.Printf("Error seed follows: %v\n", err)
		return
	}

	// Seed likes
	fmt.Printf("Seeding %d likes...\n", preset.Likes)
	if err := seedLikes(db, preset.Likes, userIDs, portfolioIDs); err != nil {
		fmt.Printf("Error seed likes: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Seeding selesai!")
}

func getJurusanIDs(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT id FROM jurusan WHERE deleted_at IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func getTahunAjaranIDs(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT id FROM tahun_ajaran WHERE deleted_at IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func getTagIDs(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT id FROM tags WHERE deleted_at IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func seedKelas(db *sql.DB, count int, jurusanIDs, tahunAjaranIDs []string) ([]string, error) {
	var kelasIDs []string
	tingkatList := []int{10, 11, 12}
	rombelList := []string{"A", "B", "C", "D", "E"}

	for i := 0; i < count; i++ {
		jurusanID := jurusanIDs[rand.Intn(len(jurusanIDs))]
		tahunAjaranID := tahunAjaranIDs[rand.Intn(len(tahunAjaranIDs))]
		tingkat := tingkatList[rand.Intn(len(tingkatList))]
		rombel := rombelList[rand.Intn(len(rombelList))]

		var id string
		err := db.QueryRow(`
			INSERT INTO kelas (tahun_ajaran_id, jurusan_id, tingkat, rombel, nama)
			VALUES ($1, $2, $3, $4, 'temp')
			ON CONFLICT (tahun_ajaran_id, jurusan_id, tingkat, rombel) DO UPDATE SET updated_at = NOW()
			RETURNING id
		`, tahunAjaranID, jurusanID, tingkat, rombel).Scan(&id)

		if err != nil {
			continue
		}
		kelasIDs = append(kelasIDs, id)
	}

	// Get all kelas IDs if we need more
	if len(kelasIDs) < 3 {
		rows, _ := db.Query("SELECT id FROM kelas WHERE deleted_at IS NULL LIMIT 10")
		defer rows.Close()
		for rows.Next() {
			var id string
			rows.Scan(&id)
			kelasIDs = append(kelasIDs, id)
		}
	}

	return kelasIDs, nil
}

var firstNames = []string{
	"Ahmad", "Budi", "Citra", "Dewi", "Eka", "Fajar", "Gita", "Hadi", "Indah", "Joko",
	"Kartika", "Lina", "Maya", "Nadia", "Omar", "Putri", "Qori", "Rina", "Sari", "Tono",
	"Umar", "Vina", "Wati", "Xena", "Yudi", "Zahra", "Andi", "Bella", "Cahya", "Dian",
	"Eko", "Fitri", "Galih", "Hana", "Irfan", "Jasmine", "Kevin", "Laras", "Mira", "Niko",
}

var lastNames = []string{
	"Pratama", "Wijaya", "Kusuma", "Santoso", "Hidayat", "Putra", "Sari", "Wibowo", "Nugroho", "Setiawan",
	"Rahayu", "Permana", "Saputra", "Lestari", "Kurniawan", "Utami", "Firmansyah", "Anggraini", "Ramadhan", "Puspita",
}

var portfolioTitles = []string{
	"Website Portfolio Pribadi", "Aplikasi Mobile E-Commerce", "Desain UI Dashboard Admin",
	"Logo Brand Fashion", "Animasi 3D Karakter", "Video Company Profile", "Ilustrasi Digital Art",
	"Game Mobile Puzzle", "Website Toko Online", "Aplikasi Manajemen Inventori",
	"Desain Poster Event", "Motion Graphics Intro", "Fotografi Produk", "Website Blog Personal",
	"Aplikasi Absensi Karyawan", "Desain Kemasan Produk", "Animasi Explainer Video",
	"Website Landing Page", "Aplikasi Kasir POS", "Desain Social Media Kit",
	"Branding Identity", "Website Company Profile", "Aplikasi Booking Online",
	"Desain Banner Iklan", "Video Tutorial", "Ilustrasi Buku Anak",
	"Game Edukasi", "Website E-Learning", "Aplikasi Delivery",
	"Desain Infografis", "Animasi Logo", "Fotografi Portrait",
}

var bioTemplates = []string{
	"Siswa yang passionate di bidang teknologi dan desain.",
	"Suka coding dan membuat aplikasi yang bermanfaat.",
	"Tertarik dengan UI/UX design dan web development.",
	"Hobi membuat konten kreatif dan video editing.",
	"Belajar programming sejak SMP.",
	"Dream big, start small, act now.",
	"Kreator konten digital dan graphic designer.",
	"Full-stack developer in training.",
	"Passionate about creating beautiful and functional designs.",
	"Always learning, always growing.",
}

var contentBlockTexts = []string{
	"<p>Proyek ini dibuat sebagai bagian dari pembelajaran di sekolah. Tujuannya adalah untuk mengaplikasikan ilmu yang sudah dipelajari ke dalam proyek nyata.</p>",
	"<p>Dalam proyek ini, saya menggunakan berbagai teknologi modern untuk menciptakan solusi yang efektif dan efisien.</p>",
	"<p>Tantangan utama dalam proyek ini adalah mengintegrasikan berbagai komponen agar bekerja dengan harmonis.</p>",
	"<p>Hasil akhir dari proyek ini melebihi ekspektasi awal dan mendapat apresiasi dari guru pembimbing.</p>",
	"<p>Proses pengerjaan proyek ini mengajarkan banyak hal tentang manajemen waktu dan kerja tim.</p>",
}

var socialPlatforms = []string{
	"github", "instagram", "linkedin", "twitter", "behance", "dribbble", "youtube",
}

func seedUsers(db *sql.DB, count int, kelasIDs []string) ([]string, error) {
	var userIDs []string
	roles := []string{"student", "student", "student", "student", "alumni"}

	// Generate bcrypt hash for "password"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	passwordHash := string(hashedPassword)

	// Create admin user first
	var adminID string
	err := db.QueryRow(`
		INSERT INTO users (username, email, password_hash, nama, role, avatar_url, is_active)
		VALUES ('admin', 'admin@grafikarsa.com', $1, 'Administrator', 'admin', 'https://i.pravatar.cc/300?u=admin', true)
		ON CONFLICT (username) DO UPDATE SET password_hash = $1, updated_at = NOW()
		RETURNING id
	`, passwordHash).Scan(&adminID)
	if err == nil {
		userIDs = append(userIDs, adminID)
	}

	for i := 0; i < count; i++ {
		firstName := firstNames[rand.Intn(len(firstNames))]
		lastName := lastNames[rand.Intn(len(lastNames))]
		nama := firstName + " " + lastName
		username := strings.ToLower(firstName) + strings.ToLower(lastName[:3]) + fmt.Sprintf("%d", rand.Intn(999))
		email := username + "@example.com"
		role := roles[rand.Intn(len(roles))]
		bio := bioTemplates[rand.Intn(len(bioTemplates))]
		tahunMasuk := 2020 + rand.Intn(5)
		avatarURL := fmt.Sprintf("https://i.pravatar.cc/300?u=%s", username)
		bannerURL := fmt.Sprintf("https://picsum.photos/seed/%s/1200/400", username)
		nisn := fmt.Sprintf("%010d", 3000000000+rand.Intn(999999999))
		nis := fmt.Sprintf("%d%04d", tahunMasuk, rand.Intn(9999))

		var kelasID *string
		if len(kelasIDs) > 0 && role == "student" {
			k := kelasIDs[rand.Intn(len(kelasIDs))]
			kelasID = &k
		}

		var id string
		var err error
		if kelasID != nil {
			err = db.QueryRow(`
				INSERT INTO users (username, email, password_hash, nama, bio, avatar_url, banner_url, role, nisn, nis, kelas_id, tahun_masuk, is_active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, true)
				ON CONFLICT (username) DO NOTHING
				RETURNING id
			`, username, email, passwordHash, nama, bio, avatarURL, bannerURL, role, nisn, nis, *kelasID, tahunMasuk).Scan(&id)
		} else {
			err = db.QueryRow(`
				INSERT INTO users (username, email, password_hash, nama, bio, avatar_url, banner_url, role, nisn, nis, tahun_masuk, is_active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, true)
				ON CONFLICT (username) DO NOTHING
				RETURNING id
			`, username, email, passwordHash, nama, bio, avatarURL, bannerURL, role, nisn, nis, tahunMasuk).Scan(&id)
		}

		if err != nil {
			continue
		}
		userIDs = append(userIDs, id)

		// Add social links for some users
		if rand.Float32() < 0.5 {
			platform := socialPlatforms[rand.Intn(len(socialPlatforms))]
			url := fmt.Sprintf("https://%s.com/%s", platform, username)
			db.Exec(`
				INSERT INTO user_social_links (user_id, platform, url)
				VALUES ($1, $2, $3)
				ON CONFLICT (user_id, platform) DO NOTHING
			`, id, platform, url)
		}
	}

	return userIDs, nil
}

func seedPortfolios(db *sql.DB, count int, userIDs, tagIDs []string) ([]string, error) {
	var portfolioIDs []string
	statuses := []string{"draft", "pending_review", "published", "published", "published", "published"}

	for i := 0; i < count; i++ {
		if len(userIDs) == 0 {
			break
		}

		userID := userIDs[rand.Intn(len(userIDs))]
		judul := portfolioTitles[rand.Intn(len(portfolioTitles))] + fmt.Sprintf(" %d", rand.Intn(1000))
		status := statuses[rand.Intn(len(statuses))]
		thumbnailURL := fmt.Sprintf("https://picsum.photos/seed/portfolio%d/800/600", rand.Intn(10000))

		var id string
		err := db.QueryRow(`
			INSERT INTO portfolios (user_id, judul, slug, thumbnail_url, status)
			VALUES ($1, $2, 'temp', $3, $4)
			RETURNING id
		`, userID, judul, thumbnailURL, status).Scan(&id)

		if err != nil {
			continue
		}
		portfolioIDs = append(portfolioIDs, id)

		// Add content blocks (2-5 per portfolio)
		blockCount := 2 + rand.Intn(4)
		for j := 0; j < blockCount; j++ {
			blockType := "text"
			if rand.Float32() < 0.3 {
				blockType = "image"
			}

			var payload string
			if blockType == "text" {
				text := contentBlockTexts[rand.Intn(len(contentBlockTexts))]
				payload = fmt.Sprintf(`{"content": %q}`, text)
			} else {
				payload = fmt.Sprintf(`{"url": "https://picsum.photos/800/600?random=%d", "alt": "Project image"}`, rand.Intn(1000))
			}

			db.Exec(`
				INSERT INTO content_blocks (portfolio_id, block_type, block_order, payload)
				VALUES ($1, $2, $3, $4)
			`, id, blockType, j, payload)
		}

		// Add tags (1-3 per portfolio)
		tagCount := 1 + rand.Intn(3)
		usedTags := make(map[int]bool)
		for j := 0; j < tagCount && j < len(tagIDs); j++ {
			tagIdx := rand.Intn(len(tagIDs))
			if usedTags[tagIdx] {
				continue
			}
			usedTags[tagIdx] = true

			db.Exec(`
				INSERT INTO portfolio_tags (portfolio_id, tag_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, id, tagIDs[tagIdx])
		}
	}

	return portfolioIDs, nil
}

func seedFollows(db *sql.DB, count int, userIDs []string) error {
	if len(userIDs) < 2 {
		return nil
	}

	created := 0
	attempts := 0
	maxAttempts := count * 3

	for created < count && attempts < maxAttempts {
		attempts++

		followerIdx := rand.Intn(len(userIDs))
		followingIdx := rand.Intn(len(userIDs))

		if followerIdx == followingIdx {
			continue
		}

		_, err := db.Exec(`
			INSERT INTO follows (follower_id, following_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, userIDs[followerIdx], userIDs[followingIdx])

		if err == nil {
			created++
		}
	}

	return nil
}

func seedLikes(db *sql.DB, count int, userIDs, portfolioIDs []string) error {
	if len(userIDs) == 0 || len(portfolioIDs) == 0 {
		return nil
	}

	created := 0
	attempts := 0
	maxAttempts := count * 3

	for created < count && attempts < maxAttempts {
		attempts++

		userID := userIDs[rand.Intn(len(userIDs))]
		portfolioID := portfolioIDs[rand.Intn(len(portfolioIDs))]

		_, err := db.Exec(`
			INSERT INTO portfolio_likes (user_id, portfolio_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, userID, portfolioID)

		if err == nil {
			created++
		}
	}

	return nil
}
