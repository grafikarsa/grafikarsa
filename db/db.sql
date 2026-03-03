-- ============================================================================
-- GRAFIKARSA DATABASE SCHEMA
-- Platform Katalog Portofolio & Social Network Warga SMKN 4 Malang
-- PostgreSQL 15+
-- ============================================================================
-- 
-- KEPUTUSAN DESAIN UTAMA:
-- 
-- 1. NORMALISASI (BCNF):
--    - Setiap tabel memiliki single-purpose dan tidak ada transitive dependency.
--    - Social links dipisah ke tabel tersendiri (user_social_links) untuk fleksibilitas
--      penambahan platform baru tanpa alter schema.
--    - Histori kelas siswa disimpan di tabel terpisah (student_class_history) untuk
--      menjaga riwayat lengkap tanpa redundansi.
--
-- 2. SCALABILITY:
--    - UUID sebagai primary key untuk distributed system compatibility.
--    - Proper indexing pada kolom yang sering di-query (username, email, slug, status).
--    - Partisi-ready design untuk tabel besar (portfolios, content_blocks).
--    - Soft delete pattern dengan deleted_at untuk data recovery.
--
-- 3. KEAMANAN JWT:
--    - Tabel refresh_tokens terpisah dengan device tracking dan revocation support.
--    - Token family tracking untuk mendeteksi token reuse attack.
--    - Automatic cleanup via expires_at untuk token hygiene.
--    - Password hash disimpan terpisah dari data user untuk query optimization.
--
-- 4. FLEKSIBILITAS:
--    - JSONB untuk content block payload (schema-less untuk berbagai tipe konten).
--    - Enum types untuk status dan role agar type-safe namun extensible.
--    - Trigger untuk auto-generate nama kelas dan slug.
--
-- ============================================================================

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- untuk fuzzy search

-- ============================================================================
-- ENUM TYPES
-- ============================================================================

CREATE TYPE user_role AS ENUM ('student', 'alumni', 'admin');
CREATE TYPE portfolio_status AS ENUM ('draft', 'pending_review', 'rejected', 'published', 'archived');
CREATE TYPE content_block_type AS ENUM ('text', 'image', 'table', 'youtube', 'button', 'embed', 'figma', 'canva', 'ppt', 'pdf', 'doc');
CREATE TYPE social_platform AS ENUM (
    'facebook', 'instagram', 'github', 'linkedin', 'twitter',
    'personal_website', 'tiktok', 'youtube', 'behance', 'dribbble',
    'threads', 'bluesky', 'medium', 'gitlab'
);
CREATE TYPE feedback_kategori AS ENUM ('bug', 'saran', 'lainnya');
CREATE TYPE feedback_status AS ENUM ('pending', 'read', 'resolved');

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Jurusan (Department/Major)
CREATE TABLE jurusan (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama VARCHAR(100) NOT NULL,
    kode VARCHAR(10) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT jurusan_kode_lowercase CHECK (kode = LOWER(kode)),
    CONSTRAINT jurusan_kode_alpha CHECK (kode ~ '^[a-z]+$')
);

CREATE INDEX idx_jurusan_kode ON jurusan(kode) WHERE deleted_at IS NULL;

COMMENT ON TABLE jurusan IS 'Master data jurusan/program keahlian';
COMMENT ON COLUMN jurusan.kode IS 'Kode jurusan lowercase, hanya huruf (contoh: rpl, tkj, mm)';

-- Tahun Ajaran (Academic Year)
CREATE TABLE tahun_ajaran (
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
);

CREATE UNIQUE INDEX idx_tahun_ajaran_active ON tahun_ajaran(is_active) 
    WHERE is_active = TRUE AND deleted_at IS NULL;

COMMENT ON TABLE tahun_ajaran IS 'Master data tahun ajaran untuk tracking kelas per periode';
COMMENT ON COLUMN tahun_ajaran.is_active IS 'Hanya satu tahun ajaran yang boleh aktif';
COMMENT ON COLUMN tahun_ajaran.promotion_month IS 'Bulan kenaikan kelas otomatis (1-12)';
COMMENT ON COLUMN tahun_ajaran.promotion_day IS 'Tanggal kenaikan kelas otomatis (1-31)';

-- Kelas (Class)
CREATE TABLE kelas (
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
);

CREATE INDEX idx_kelas_tahun_ajaran ON kelas(tahun_ajaran_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_kelas_jurusan ON kelas(jurusan_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_kelas_nama ON kelas(nama) WHERE deleted_at IS NULL;

COMMENT ON TABLE kelas IS 'Data kelas per tahun ajaran';
COMMENT ON COLUMN kelas.tingkat IS 'Tingkat kelas: 10, 11, atau 12';
COMMENT ON COLUMN kelas.rombel IS 'Rombongan belajar: A-Z';
COMMENT ON COLUMN kelas.nama IS 'Nama kelas auto-generated: X-RPL-A, XI-MM-B, dst';

-- Tags
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tags_nama ON tags(nama) WHERE deleted_at IS NULL;

COMMENT ON TABLE tags IS 'Master data tags untuk kategorisasi portofolio';

-- Series (template portofolio dengan block konten)
CREATE TABLE series (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama VARCHAR(100) NOT NULL UNIQUE,
    deskripsi TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_series_nama ON series(nama) WHERE deleted_at IS NULL;
CREATE INDEX idx_series_active ON series(is_active) WHERE deleted_at IS NULL;

COMMENT ON TABLE series IS 'Template series untuk struktur portofolio dengan block konten yang sudah ditentukan';
COMMENT ON COLUMN series.deskripsi IS 'Deskripsi/penjelasan tentang series template ini';
COMMENT ON COLUMN series.is_active IS 'Status aktif series, hanya series aktif yang ditampilkan ke user';

-- Series Blocks (template block konten untuk series)
CREATE TABLE series_blocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    series_id UUID NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    block_type content_block_type NOT NULL,
    block_order INTEGER NOT NULL,
    instruksi TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT series_blocks_order_positive CHECK (block_order >= 0),
    CONSTRAINT series_blocks_unique_order UNIQUE (series_id, block_order) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_series_blocks_series ON series_blocks(series_id);
CREATE INDEX idx_series_blocks_order ON series_blocks(series_id, block_order);

COMMENT ON TABLE series_blocks IS 'Template block konten untuk series';
COMMENT ON COLUMN series_blocks.block_type IS 'Tipe block: text, image, youtube, dll';
COMMENT ON COLUMN series_blocks.instruksi IS 'Instruksi/panduan untuk mengisi block ini';
COMMENT ON COLUMN series_blocks.block_order IS 'Urutan block dalam template (dimulai dari 0)';

-- ============================================================================
-- USER & AUTHENTICATION
-- ============================================================================

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(30) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nama VARCHAR(100) NOT NULL,
    bio TEXT,
    avatar_url TEXT,
    banner_url TEXT,
    role user_role NOT NULL DEFAULT 'student',
    
    -- Student-specific fields (nullable for non-students)
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
);

CREATE INDEX idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_kelas ON users(kelas_id) WHERE deleted_at IS NULL AND kelas_id IS NOT NULL;
CREATE INDEX idx_users_nama_trgm ON users USING gin(nama gin_trgm_ops) WHERE deleted_at IS NULL;

COMMENT ON TABLE users IS 'Data user (student, alumni, admin)';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN users.kelas_id IS 'Kelas saat ini (untuk student aktif)';

-- User Social Links (normalized)
CREATE TABLE user_social_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform social_platform NOT NULL,
    url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT user_social_links_unique UNIQUE (user_id, platform)
);

CREATE INDEX idx_user_social_links_user ON user_social_links(user_id);

COMMENT ON TABLE user_social_links IS 'Social media links user (dinormalisasi untuk fleksibilitas)';

-- Student Class History (untuk tracking riwayat kelas)
CREATE TABLE student_class_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kelas_id UUID NOT NULL REFERENCES kelas(id) ON DELETE RESTRICT,
    tahun_ajaran_id UUID NOT NULL REFERENCES tahun_ajaran(id) ON DELETE RESTRICT,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT student_class_history_unique UNIQUE (user_id, kelas_id, tahun_ajaran_id)
);

CREATE INDEX idx_student_class_history_user ON student_class_history(user_id);
CREATE INDEX idx_student_class_history_current ON student_class_history(user_id, is_current) WHERE is_current = TRUE;

COMMENT ON TABLE student_class_history IS 'Riwayat kelas siswa per tahun ajaran';

-- ============================================================================
-- JWT REFRESH TOKEN MANAGEMENT
-- ============================================================================

CREATE TABLE refresh_tokens (
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
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id) WHERE is_revoked = FALSE;
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash) WHERE is_revoked = FALSE;
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE is_revoked = FALSE;

COMMENT ON TABLE refresh_tokens IS 'JWT refresh tokens dengan rotation dan revocation support';
COMMENT ON COLUMN refresh_tokens.token_hash IS 'SHA-256 hash dari refresh token (token asli tidak disimpan)';
COMMENT ON COLUMN refresh_tokens.family_id IS 'Token family untuk deteksi reuse attack';
COMMENT ON COLUMN refresh_tokens.device_info IS 'Info device: user_agent, device_type, dll';

-- Token Blacklist (untuk access token yang perlu di-revoke sebelum expire)
CREATE TABLE token_blacklist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    jti VARCHAR(64) NOT NULL UNIQUE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    blacklisted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason VARCHAR(100)
);

CREATE INDEX idx_token_blacklist_jti ON token_blacklist(jti);
CREATE INDEX idx_token_blacklist_expires ON token_blacklist(expires_at);

COMMENT ON TABLE token_blacklist IS 'Blacklist untuk access token yang di-revoke sebelum expire';
COMMENT ON COLUMN token_blacklist.jti IS 'JWT ID (unique identifier per token)';

-- ============================================================================
-- SOCIAL FEATURES
-- ============================================================================

-- Follows (user following system)
CREATE TABLE follows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT follows_no_self_follow CHECK (follower_id != following_id),
    CONSTRAINT follows_unique UNIQUE (follower_id, following_id)
);

CREATE INDEX idx_follows_follower ON follows(follower_id);
CREATE INDEX idx_follows_following ON follows(following_id);

COMMENT ON TABLE follows IS 'Relasi follow antar user';

-- ============================================================================
-- PORTFOLIO
-- ============================================================================

-- Portfolios
CREATE TABLE portfolios (
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
    series_id UUID REFERENCES series(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT portfolios_slug_unique UNIQUE (user_id, slug)
);

CREATE INDEX idx_portfolios_user ON portfolios(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_portfolios_status ON portfolios(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_portfolios_published ON portfolios(published_at DESC) WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX idx_portfolios_pending ON portfolios(created_at) WHERE status = 'pending_review' AND deleted_at IS NULL;
CREATE INDEX idx_portfolios_slug ON portfolios(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_portfolios_series ON portfolios(series_id) WHERE deleted_at IS NULL AND series_id IS NOT NULL;

COMMENT ON TABLE portfolios IS 'Portofolio karya user';
COMMENT ON COLUMN portfolios.slug IS 'URL-friendly identifier, auto-generated dari judul';
COMMENT ON COLUMN portfolios.admin_review_note IS 'Catatan review dari admin (alasan reject, feedback, dll)';
COMMENT ON COLUMN portfolios.series_id IS 'Series template yang digunakan (NULL jika portofolio bebas)';

-- Portfolio Tags (many-to-many)
CREATE TABLE portfolio_tags (
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (portfolio_id, tag_id)
);

CREATE INDEX idx_portfolio_tags_tag ON portfolio_tags(tag_id);

COMMENT ON TABLE portfolio_tags IS 'Relasi many-to-many portfolio dan tags';

-- Content Blocks
CREATE TABLE content_blocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    block_type content_block_type NOT NULL,
    block_order INTEGER NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT content_blocks_order_positive CHECK (block_order >= 0),
    CONSTRAINT content_blocks_unique_order UNIQUE (portfolio_id, block_order) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_content_blocks_portfolio ON content_blocks(portfolio_id);
CREATE INDEX idx_content_blocks_order ON content_blocks(portfolio_id, block_order);

COMMENT ON TABLE content_blocks IS 'Modular content blocks untuk portofolio';
COMMENT ON COLUMN content_blocks.block_type IS 'Tipe block: text, image, table, youtube, button, embed';
COMMENT ON COLUMN content_blocks.payload IS 'Konten block dalam format JSON sesuai tipe';

-- Portfolio Likes
CREATE TABLE portfolio_likes (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (user_id, portfolio_id)
);

CREATE INDEX idx_portfolio_likes_portfolio ON portfolio_likes(portfolio_id);

COMMENT ON TABLE portfolio_likes IS 'Like/favorit portofolio oleh user';

-- ============================================================================
-- ADMIN CONFIGURATION
-- ============================================================================

-- App Settings (untuk konfigurasi seperti admin login path)
CREATE TABLE app_settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

COMMENT ON TABLE app_settings IS 'Konfigurasi aplikasi (termasuk admin login path)';

-- ============================================================================
-- FEEDBACK
-- ============================================================================

-- Feedback (saran, bug report, dll dari user)
CREATE TABLE feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    kategori feedback_kategori NOT NULL,
    pesan TEXT NOT NULL,
    status feedback_status NOT NULL DEFAULT 'pending',
    admin_notes TEXT,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_user ON feedback(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_feedback_status ON feedback(status);
CREATE INDEX idx_feedback_kategori ON feedback(kategori);
CREATE INDEX idx_feedback_created ON feedback(created_at DESC);

COMMENT ON TABLE feedback IS 'Feedback dari user (bug report, saran, dll)';
COMMENT ON COLUMN feedback.user_id IS 'NULL jika feedback dari guest (tidak login)';
COMMENT ON COLUMN feedback.admin_notes IS 'Catatan internal dari admin';

-- NOTE: Trigger trg_feedback_updated_at dibuat di bagian TRIGGERS setelah function update_updated_at() didefinisikan

-- ============================================================================
-- ASSESSMENT (Penilaian Portfolio)
-- ============================================================================

-- Assessment Metrics (Master data metrik penilaian)
CREATE TABLE assessment_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama VARCHAR(100) NOT NULL,
    deskripsi TEXT,
    urutan INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_assessment_metrics_urutan ON assessment_metrics(urutan) WHERE deleted_at IS NULL;
CREATE INDEX idx_assessment_metrics_active ON assessment_metrics(is_active) WHERE deleted_at IS NULL;

COMMENT ON TABLE assessment_metrics IS 'Master data metrik penilaian portfolio';
COMMENT ON COLUMN assessment_metrics.nama IS 'Nama metrik penilaian (contoh: Kreativitas, Teknis, dll)';
COMMENT ON COLUMN assessment_metrics.deskripsi IS 'Deskripsi/penjelasan metrik untuk panduan penilaian';
COMMENT ON COLUMN assessment_metrics.urutan IS 'Urutan tampilan metrik';
COMMENT ON COLUMN assessment_metrics.is_active IS 'Status aktif metrik (soft disable tanpa hapus)';

-- Portfolio Assessments (Header penilaian portfolio)
CREATE TABLE portfolio_assessments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    assessed_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    final_comment TEXT,
    total_score DECIMAL(4,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT portfolio_assessments_unique UNIQUE (portfolio_id),
    CONSTRAINT portfolio_assessments_score_range CHECK (total_score IS NULL OR (total_score >= 1 AND total_score <= 10))
);

CREATE INDEX idx_portfolio_assessments_portfolio ON portfolio_assessments(portfolio_id);
CREATE INDEX idx_portfolio_assessments_assessed_by ON portfolio_assessments(assessed_by);

COMMENT ON TABLE portfolio_assessments IS 'Header penilaian portfolio oleh admin';
COMMENT ON COLUMN portfolio_assessments.portfolio_id IS 'Portfolio yang dinilai (hanya published)';
COMMENT ON COLUMN portfolio_assessments.assessed_by IS 'Admin yang melakukan penilaian';
COMMENT ON COLUMN portfolio_assessments.final_comment IS 'Komentar/pesan akhir dari admin';
COMMENT ON COLUMN portfolio_assessments.total_score IS 'Rata-rata nilai dari semua metrik (auto-calculated)';

-- Portfolio Assessment Scores (Detail nilai per metrik)
CREATE TABLE portfolio_assessment_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assessment_id UUID NOT NULL REFERENCES portfolio_assessments(id) ON DELETE CASCADE,
    metric_id UUID NOT NULL REFERENCES assessment_metrics(id) ON DELETE RESTRICT,
    score SMALLINT NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT portfolio_assessment_scores_unique UNIQUE (assessment_id, metric_id),
    CONSTRAINT portfolio_assessment_scores_range CHECK (score >= 1 AND score <= 10)
);

CREATE INDEX idx_portfolio_assessment_scores_assessment ON portfolio_assessment_scores(assessment_id);
CREATE INDEX idx_portfolio_assessment_scores_metric ON portfolio_assessment_scores(metric_id);

COMMENT ON TABLE portfolio_assessment_scores IS 'Detail nilai per metrik untuk setiap penilaian';
COMMENT ON COLUMN portfolio_assessment_scores.score IS 'Nilai 1-10 untuk metrik ini';
COMMENT ON COLUMN portfolio_assessment_scores.comment IS 'Komentar opsional untuk metrik ini';

-- ============================================================================
-- FUNCTIONS & TRIGGERS
-- ============================================================================

-- Function: Generate nama kelas
CREATE OR REPLACE FUNCTION generate_nama_kelas()
RETURNS TRIGGER AS $$
DECLARE
    v_kode_jurusan VARCHAR(10);
    v_tingkat_romawi VARCHAR(4);
BEGIN
    -- Get kode jurusan
    SELECT UPPER(kode) INTO v_kode_jurusan FROM jurusan WHERE id = NEW.jurusan_id;
    
    -- Convert tingkat to Roman numeral
    v_tingkat_romawi := CASE NEW.tingkat
        WHEN 10 THEN 'X'
        WHEN 11 THEN 'XI'
        WHEN 12 THEN 'XII'
    END;
    
    -- Generate nama: XII-RPL-A
    NEW.nama := v_tingkat_romawi || '-' || v_kode_jurusan || '-' || NEW.rombel;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_kelas_generate_nama
    BEFORE INSERT OR UPDATE ON kelas
    FOR EACH ROW
    EXECUTE FUNCTION generate_nama_kelas();

-- Function: Generate slug dari judul
CREATE OR REPLACE FUNCTION generate_portfolio_slug()
RETURNS TRIGGER AS $$
DECLARE
    v_base_slug VARCHAR(250);
    v_slug VARCHAR(250);
    v_counter INTEGER := 0;
BEGIN
    -- Generate base slug: lowercase, replace spaces with dash, remove special chars
    v_base_slug := LOWER(TRIM(NEW.judul));
    v_base_slug := REGEXP_REPLACE(v_base_slug, '[^a-z0-9\s-]', '', 'g');
    v_base_slug := REGEXP_REPLACE(v_base_slug, '\s+', '-', 'g');
    v_base_slug := REGEXP_REPLACE(v_base_slug, '-+', '-', 'g');
    v_base_slug := TRIM(BOTH '-' FROM v_base_slug);
    v_base_slug := LEFT(v_base_slug, 200);
    
    v_slug := v_base_slug;
    
    -- Check uniqueness and append counter if needed
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
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_portfolio_generate_slug
    BEFORE INSERT OR UPDATE OF judul ON portfolios
    FOR EACH ROW
    EXECUTE FUNCTION generate_portfolio_slug();

-- Function: Set published_at when status changes to published
CREATE OR REPLACE FUNCTION set_portfolio_published_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'published' AND (OLD.status IS NULL OR OLD.status != 'published') THEN
        NEW.published_at := NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_portfolio_set_published_at
    BEFORE UPDATE ON portfolios
    FOR EACH ROW
    EXECUTE FUNCTION set_portfolio_published_at();

-- Function: Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to relevant tables
CREATE TRIGGER trg_jurusan_updated_at BEFORE UPDATE ON jurusan FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_tahun_ajaran_updated_at BEFORE UPDATE ON tahun_ajaran FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_kelas_updated_at BEFORE UPDATE ON kelas FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_tags_updated_at BEFORE UPDATE ON tags FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_series_updated_at BEFORE UPDATE ON series FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_series_blocks_updated_at BEFORE UPDATE ON series_blocks FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_user_social_links_updated_at BEFORE UPDATE ON user_social_links FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_portfolios_updated_at BEFORE UPDATE ON portfolios FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_content_blocks_updated_at BEFORE UPDATE ON content_blocks FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_feedback_updated_at BEFORE UPDATE ON feedback FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_assessment_metrics_updated_at BEFORE UPDATE ON assessment_metrics FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_portfolio_assessments_updated_at BEFORE UPDATE ON portfolio_assessments FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_portfolio_assessment_scores_updated_at BEFORE UPDATE ON portfolio_assessment_scores FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Function: Auto-calculate assessment total_score
CREATE OR REPLACE FUNCTION calculate_assessment_total_score()
RETURNS TRIGGER AS $$
DECLARE
    v_avg_score DECIMAL(4,2);
BEGIN
    -- Calculate average score for the assessment
    SELECT AVG(score)::DECIMAL(4,2) INTO v_avg_score
    FROM portfolio_assessment_scores
    WHERE assessment_id = COALESCE(NEW.assessment_id, OLD.assessment_id);
    
    -- Update total_score in portfolio_assessments
    UPDATE portfolio_assessments
    SET total_score = v_avg_score
    WHERE id = COALESCE(NEW.assessment_id, OLD.assessment_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_assessment_scores_calc_total
    AFTER INSERT OR UPDATE OR DELETE ON portfolio_assessment_scores
    FOR EACH ROW
    EXECUTE FUNCTION calculate_assessment_total_score();

-- Function: Ensure only one active tahun_ajaran
CREATE OR REPLACE FUNCTION ensure_single_active_tahun_ajaran()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_active = TRUE THEN
        UPDATE tahun_ajaran SET is_active = FALSE 
        WHERE id != NEW.id AND is_active = TRUE AND deleted_at IS NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tahun_ajaran_single_active
    AFTER INSERT OR UPDATE OF is_active ON tahun_ajaran
    FOR EACH ROW
    WHEN (NEW.is_active = TRUE)
    EXECUTE FUNCTION ensure_single_active_tahun_ajaran();

-- Function: Sync kelas_id to student_class_history
CREATE OR REPLACE FUNCTION sync_student_class_history()
RETURNS TRIGGER AS $$
DECLARE
    v_tahun_ajaran_id UUID;
BEGIN
    IF NEW.kelas_id IS NOT NULL AND NEW.role IN ('student', 'alumni') THEN
        -- Get tahun_ajaran_id from kelas
        SELECT tahun_ajaran_id INTO v_tahun_ajaran_id FROM kelas WHERE id = NEW.kelas_id;
        
        -- Set all previous history to not current
        UPDATE student_class_history SET is_current = FALSE WHERE user_id = NEW.id;
        
        -- Insert or update history
        INSERT INTO student_class_history (user_id, kelas_id, tahun_ajaran_id, is_current)
        VALUES (NEW.id, NEW.kelas_id, v_tahun_ajaran_id, TRUE)
        ON CONFLICT (user_id, kelas_id, tahun_ajaran_id) 
        DO UPDATE SET is_current = TRUE;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_sync_class_history
    AFTER INSERT OR UPDATE OF kelas_id ON users
    FOR EACH ROW
    EXECUTE FUNCTION sync_student_class_history();

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Insert default admin login path setting
INSERT INTO app_settings (key, value, description) VALUES
('admin_login_path', '"loginadmin"', 'Path untuk halaman login admin (tanpa slash)');

-- Insert sample tahun ajaran
INSERT INTO tahun_ajaran (tahun_mulai, is_active, promotion_month, promotion_day) VALUES
(2024, FALSE, 7, 1),
(2025, TRUE, 7, 1);

-- Insert sample jurusan
INSERT INTO jurusan (nama, kode) VALUES
('Rekayasa Perangkat Lunak', 'rpl'),
('Teknik Komputer dan Jaringan', 'tkj'),
('Multimedia', 'mm'),
('Desain Komunikasi Visual', 'dkv'),
('Animasi', 'ani');

-- Insert sample tags
INSERT INTO tags (nama) VALUES
('Web Development'),
('Mobile App'),
('UI/UX Design'),
('Graphic Design'),
('3D Modeling'),
('Animation'),
('Video Editing'),
('Photography'),
('Illustration'),
('Game Development');

-- Insert sample series with blocks
DO $$
DECLARE
    v_series_id_1 UUID;
    v_series_id_2 UUID;
    v_series_id_3 UUID;
BEGIN
    -- Series 1: PJBL Semester 1
    INSERT INTO series (nama, deskripsi, is_active) VALUES
    ('PJBL Semester 1', 'Template untuk proyek PJBL semester 1. Siswa wajib mengisi semua block sesuai instruksi.', TRUE)
    RETURNING id INTO v_series_id_1;
    
    INSERT INTO series_blocks (series_id, block_type, block_order, instruksi) VALUES
    (v_series_id_1, 'text', 0, 'Judul dan deskripsi singkat proyek PJBL'),
    (v_series_id_1, 'image', 1, 'Thumbnail/cover proyek (rasio 16:9 recommended)'),
    (v_series_id_1, 'youtube', 2, 'Video dokumentasi presentasi PJBL'),
    (v_series_id_1, 'text', 3, 'Tujuan dan latar belakang pembuatan proyek'),
    (v_series_id_1, 'text', 4, 'Proses pengerjaan dan tantangan yang dihadapi'),
    (v_series_id_1, 'image', 5, 'Screenshot atau foto hasil akhir proyek'),
    (v_series_id_1, 'text', 6, 'Kesimpulan dan pembelajaran yang didapat');

    -- Series 2: Ujian Praktik
    INSERT INTO series (nama, deskripsi, is_active) VALUES
    ('Ujian Praktik', 'Template untuk dokumentasi ujian praktik kejuruan.', TRUE)
    RETURNING id INTO v_series_id_2;
    
    INSERT INTO series_blocks (series_id, block_type, block_order, instruksi) VALUES
    (v_series_id_2, 'text', 0, 'Judul proyek ujian praktik'),
    (v_series_id_2, 'text', 1, 'Deskripsi singkat dan tujuan proyek'),
    (v_series_id_2, 'image', 2, 'Screenshot tampilan utama aplikasi/karya'),
    (v_series_id_2, 'text', 3, 'Fitur-fitur utama yang dikembangkan'),
    (v_series_id_2, 'youtube', 4, 'Video demo aplikasi/karya (opsional)');

    -- Series 3: Lomba Internal
    INSERT INTO series (nama, deskripsi, is_active) VALUES
    ('Lomba Internal', 'Template untuk karya lomba internal sekolah.', TRUE)
    RETURNING id INTO v_series_id_3;
    
    INSERT INTO series_blocks (series_id, block_type, block_order, instruksi) VALUES
    (v_series_id_3, 'text', 0, 'Judul karya lomba'),
    (v_series_id_3, 'image', 1, 'Gambar utama karya'),
    (v_series_id_3, 'text', 2, 'Konsep dan ide di balik karya'),
    (v_series_id_3, 'text', 3, 'Proses kreatif pembuatan karya');
END $$;

-- Insert sample assessment metrics
INSERT INTO assessment_metrics (nama, deskripsi, urutan) VALUES
('Kreativitas', 'Tingkat orisinalitas dan inovasi dalam karya', 1),
('Kualitas Teknis', 'Kualitas eksekusi teknis dan penguasaan tools', 2),
('Estetika Visual', 'Keindahan visual dan komposisi desain', 3),
('Kelengkapan', 'Kelengkapan dokumentasi dan penjelasan karya', 4),
('Relevansi', 'Kesesuaian dengan tujuan dan target audience', 5);

-- Insert default admin user
-- Username: admin, Password: password (bcrypt hash with cost 10)
INSERT INTO users (username, email, password_hash, nama, role, is_active) VALUES
('admin', 'admin@grafikarsa.com', '$2a$10$awvzkFPY1N91aqpBAunz3evxSxfx/841EFqTwdnw2SKYxYBQ2nneG', 'Administrator', 'admin', TRUE);

-- ============================================================================
-- VIEWS (untuk kemudahan query)
-- ============================================================================

-- View: User profile dengan info kelas
CREATE OR REPLACE VIEW v_user_profiles AS
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
WHERE u.deleted_at IS NULL;

-- View: Published portfolios dengan info user
CREATE OR REPLACE VIEW v_published_portfolios AS
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
    (SELECT ARRAY_AGG(t.nama) FROM portfolio_tags pt JOIN tags t ON pt.tag_id = t.id WHERE pt.portfolio_id = p.id) AS tags,
    s.nama AS series_nama
FROM portfolios p
JOIN users u ON p.user_id = u.id AND u.deleted_at IS NULL
LEFT JOIN kelas k ON u.kelas_id = k.id
LEFT JOIN jurusan j ON k.jurusan_id = j.id
LEFT JOIN series s ON p.series_id = s.id AND s.deleted_at IS NULL
WHERE p.status = 'published' AND p.deleted_at IS NULL
ORDER BY p.published_at DESC;

-- View: Portfolio dengan assessment info
CREATE OR REPLACE VIEW v_portfolio_assessments AS
SELECT 
    p.id AS portfolio_id,
    p.judul,
    p.slug,
    p.thumbnail_url,
    p.published_at,
    p.user_id,
    u.username AS user_username,
    u.nama AS user_nama,
    u.avatar_url AS user_avatar,
    pa.id AS assessment_id,
    pa.total_score,
    pa.final_comment,
    pa.assessed_by,
    assessor.nama AS assessor_nama,
    pa.created_at AS assessed_at
FROM portfolios p
JOIN users u ON p.user_id = u.id AND u.deleted_at IS NULL
LEFT JOIN portfolio_assessments pa ON p.id = pa.portfolio_id
LEFT JOIN users assessor ON pa.assessed_by = assessor.id
WHERE p.status = 'published' AND p.deleted_at IS NULL;

-- ============================================================================
-- CLEANUP FUNCTIONS (untuk maintenance)
-- ============================================================================

-- Function: Cleanup expired refresh tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM refresh_tokens WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    DELETE FROM token_blacklist WHERE expires_at < NOW();
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_expired_tokens() IS 'Hapus refresh tokens dan blacklist yang sudah expired. Jalankan via cron job.';

-- ============================================================================
-- PERMISSIONS (contoh untuk role-based access)
-- ============================================================================

-- Create application roles (uncomment jika diperlukan)
-- CREATE ROLE grafikarsa_app LOGIN PASSWORD 'secure_password';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO grafikarsa_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO grafikarsa_app;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO grafikarsa_app;

-- ============================================================================
-- NOTIFICATIONS
-- ============================================================================

-- Notification type enum
-- Notification type enum
CREATE TYPE notification_type AS ENUM ('new_follower', 'portfolio_liked', 'portfolio_approved', 'portfolio_rejected', 'feedback_updated', 'new_comment', 'reply_comment');

-- Notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type notification_type NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    data JSONB DEFAULT '{}',
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for fast queries
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);
CREATE INDEX idx_notifications_type ON notifications(type);

COMMENT ON TABLE notifications IS 'Notifikasi untuk user';
COMMENT ON COLUMN notifications.type IS 'Tipe notifikasi: new_follower, portfolio_liked, portfolio_approved, portfolio_rejected';
COMMENT ON COLUMN notifications.data IS 'Data tambahan dalam format JSON (actor info, portfolio info, dll)';

-- ============================================================================
-- SPECIAL ROLES (Akses Admin Terbatas)
-- ============================================================================

-- Special Roles table
CREATE TABLE special_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    color VARCHAR(7) NOT NULL DEFAULT '#6366f1',
    capabilities TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_special_roles_nama ON special_roles(nama) WHERE deleted_at IS NULL;
CREATE INDEX idx_special_roles_active ON special_roles(is_active) WHERE deleted_at IS NULL;

COMMENT ON TABLE special_roles IS 'Master data special roles untuk akses admin terbatas';
COMMENT ON COLUMN special_roles.color IS 'Warna hex untuk badge/chip (base color untuk text)';
COMMENT ON COLUMN special_roles.capabilities IS 'Array capability keys yang dimiliki role ini';

-- User Special Roles junction table
CREATE TABLE user_special_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    special_role_id UUID NOT NULL REFERENCES special_roles(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (user_id, special_role_id)
);

CREATE INDEX idx_user_special_roles_role ON user_special_roles(special_role_id);
CREATE INDEX idx_user_special_roles_user ON user_special_roles(user_id);

COMMENT ON TABLE user_special_roles IS 'Relasi many-to-many user dan special roles';
COMMENT ON COLUMN user_special_roles.assigned_by IS 'Admin yang meng-assign role ini ke user';

-- Trigger for updated_at
CREATE TRIGGER trg_special_roles_updated_at 
    BEFORE UPDATE ON special_roles 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at();

-- Insert sample special roles
INSERT INTO special_roles (nama, description, color, capabilities) VALUES
('Moderator Konten', 'Dapat memoderasi dan mengelola portfolio', '#f97316', ARRAY['portfolios', 'moderation']),
('Pengelola Akademik', 'Mengelola data jurusan, kelas, dan tahun ajaran', '#06b6d4', ARRAY['majors', 'classes', 'academic_years']),
('Penilai Portfolio', 'Menilai portfolio siswa', '#eab308', ARRAY['assessments', 'assessment_metrics']),
('Super Moderator', 'Akses hampir semua fitur admin', '#8b5cf6', ARRAY['dashboard', 'portfolios', 'moderation', 'assessments', 'assessment_metrics', 'tags', 'series', 'feedback'])
ON CONFLICT (nama) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================

CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_edited BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_comments_portfolio_id ON comments(portfolio_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_parent_id ON comments(parent_id);
CREATE INDEX idx_comments_created_at ON comments(created_at);

COMMENT ON TABLE comments IS 'Komentar pada portfolio, mendukung threading (replies)';
COMMENT ON COLUMN comments.parent_id IS 'ID komentar induk jika ini adalah balasan (NULL untuk top-level comment)';

-- Trigger updated_at for comments
CREATE TRIGGER trg_comments_updated_at 
    BEFORE UPDATE ON comments 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at();


-- ============================================================================
-- SMART FEED ALGORITHM
-- ============================================================================

-- Portfolio Views Tracking
CREATE TABLE portfolio_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id VARCHAR(64),
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT portfolio_views_unique_user UNIQUE (portfolio_id, user_id),
    CONSTRAINT portfolio_views_unique_session UNIQUE (portfolio_id, session_id),
    CONSTRAINT portfolio_views_has_identifier CHECK (user_id IS NOT NULL OR session_id IS NOT NULL)
);

CREATE INDEX idx_portfolio_views_portfolio ON portfolio_views(portfolio_id);
CREATE INDEX idx_portfolio_views_user ON portfolio_views(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_portfolio_views_session ON portfolio_views(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_portfolio_views_viewed_at ON portfolio_views(viewed_at DESC);

COMMENT ON TABLE portfolio_views IS 'Tracking view portfolio untuk feed algorithm dan analytics';

-- User Interests (auto-generated dari aktivitas)
CREATE TABLE user_interests (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    liked_tags JSONB NOT NULL DEFAULT '{}',
    liked_jurusan JSONB NOT NULL DEFAULT '{}',
    total_likes INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_interests_updated ON user_interests(updated_at DESC);

COMMENT ON TABLE user_interests IS 'Profil interest user dari aktivitas like';

-- User Feed Preferences
CREATE TABLE user_feed_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    algorithm VARCHAR(20) NOT NULL DEFAULT 'smart',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT user_feed_preferences_algorithm_valid CHECK (algorithm IN ('smart', 'recent', 'following'))
);

COMMENT ON TABLE user_feed_preferences IS 'Preferensi algoritma feed per user';

-- Triggers for updated_at
CREATE TRIGGER trg_user_interests_updated_at 
    BEFORE UPDATE ON user_interests 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_user_feed_preferences_updated_at 
    BEFORE UPDATE ON user_feed_preferences 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at();

-- Helper function: Get unique view count
CREATE OR REPLACE FUNCTION get_portfolio_view_count(p_portfolio_id UUID)
RETURNS BIGINT AS $$
BEGIN
    RETURN (
        SELECT COUNT(DISTINCT COALESCE(user_id::text, session_id))
        FROM portfolio_views
        WHERE portfolio_id = p_portfolio_id
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Helper function: Get max engagement stats for normalization
CREATE OR REPLACE FUNCTION get_max_engagement_stats()
RETURNS TABLE (max_likes BIGINT, max_views BIGINT) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COALESCE(MAX(like_count), 1)::BIGINT as max_likes,
        COALESCE(MAX(view_count), 1)::BIGINT as max_views
    FROM (
        SELECT 
            p.id,
            (SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = p.id) as like_count,
            get_portfolio_view_count(p.id) as view_count
        FROM portfolios p
        WHERE p.status = 'published' AND p.deleted_at IS NULL
    ) stats;
END;
$$ LANGUAGE plpgsql STABLE;


-- ============================================================================
-- CHANGELOG
-- ============================================================================

-- Main changelog table
CREATE TABLE changelogs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    release_date DATE NOT NULL DEFAULT CURRENT_DATE,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_changelogs_release_date ON changelogs(release_date DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_changelogs_is_published ON changelogs(is_published) WHERE deleted_at IS NULL;
CREATE INDEX idx_changelogs_deleted_at ON changelogs(deleted_at);

COMMENT ON TABLE changelogs IS 'Changelog/release notes untuk update aplikasi';
COMMENT ON COLUMN changelogs.version IS 'Versi release (contoh: 1.0.0, 1.2.0)';
COMMENT ON COLUMN changelogs.is_published IS 'Status publish, hanya yang published yang tampil ke user';

-- Changelog sections (added, updated, removed, fixed)
CREATE TABLE changelog_sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    changelog_id UUID NOT NULL REFERENCES changelogs(id) ON DELETE CASCADE,
    category VARCHAR(20) NOT NULL,
    section_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT changelog_sections_category_valid CHECK (category IN ('added', 'updated', 'removed', 'fixed')),
    CONSTRAINT changelog_sections_order_positive CHECK (section_order >= 0)
);

CREATE INDEX idx_changelog_sections_changelog ON changelog_sections(changelog_id);
CREATE INDEX idx_changelog_sections_order ON changelog_sections(changelog_id, section_order);

COMMENT ON TABLE changelog_sections IS 'Section dalam changelog berdasarkan kategori perubahan';
COMMENT ON COLUMN changelog_sections.category IS 'Kategori: added, updated, removed, fixed';

-- Content blocks for each section
CREATE TABLE changelog_section_blocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    section_id UUID NOT NULL REFERENCES changelog_sections(id) ON DELETE CASCADE,
    block_type content_block_type NOT NULL,
    block_order INTEGER NOT NULL DEFAULT 0,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT changelog_section_blocks_order_positive CHECK (block_order >= 0)
);

CREATE INDEX idx_changelog_section_blocks_section ON changelog_section_blocks(section_id);
CREATE INDEX idx_changelog_section_blocks_order ON changelog_section_blocks(section_id, block_order);

COMMENT ON TABLE changelog_section_blocks IS 'Content blocks untuk setiap section changelog';
COMMENT ON COLUMN changelog_section_blocks.block_type IS 'Tipe block: text, image, dll';
COMMENT ON COLUMN changelog_section_blocks.payload IS 'Konten block dalam format JSON';

-- Contributors for each changelog
CREATE TABLE changelog_contributors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    changelog_id UUID NOT NULL REFERENCES changelogs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    contribution VARCHAR(255) NOT NULL,
    contributor_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT changelog_contributors_order_positive CHECK (contributor_order >= 0)
);

CREATE INDEX idx_changelog_contributors_changelog ON changelog_contributors(changelog_id);
CREATE INDEX idx_changelog_contributors_user ON changelog_contributors(user_id);
CREATE INDEX idx_changelog_contributors_order ON changelog_contributors(changelog_id, contributor_order);

COMMENT ON TABLE changelog_contributors IS 'Kontributor untuk setiap changelog';
COMMENT ON COLUMN changelog_contributors.contribution IS 'Deskripsi kontribusi (contoh: Backend Development)';

-- Track which changelogs have been read by users (for "New" badge)
CREATE TABLE changelog_reads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    changelog_id UUID NOT NULL REFERENCES changelogs(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT changelog_reads_unique UNIQUE (user_id, changelog_id)
);

CREATE INDEX idx_changelog_reads_user ON changelog_reads(user_id);
CREATE INDEX idx_changelog_reads_changelog ON changelog_reads(changelog_id);

COMMENT ON TABLE changelog_reads IS 'Tracking changelog yang sudah dibaca user untuk badge "New"';

-- Triggers for updated_at
CREATE TRIGGER trg_changelogs_updated_at 
    BEFORE UPDATE ON changelogs 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_changelog_section_blocks_updated_at 
    BEFORE UPDATE ON changelog_section_blocks 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at();
