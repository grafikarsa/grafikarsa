-- ============================================================================
-- GRAFIKARSA DATABASE SCHEMA
-- Platform Katalog Portofolio & Social Network SMKN 4 Malang
-- PostgreSQL 16+ Compatible
-- Normalization: BCNF (Boyce-Codd Normal Form)
-- ============================================================================

-- ============================================================================
-- A. DATABASE DESIGN SUMMARY
-- ============================================================================
/*
TABLES AND PURPOSES:
1.  users                   - All users (admin, student, alumni)
2.  majors                  - Academic majors/departments (Jurusan)
3.  academic_years          - Academic year periods with promotion config
4.  classes                 - Class groups per academic year
5.  student_class_history   - Historical record of student class assignments
6.  portfolios              - Portfolio headers (metadata)
7.  portfolio_versions      - Versioned content for portfolios (max 2 per portfolio)
8.  content_blocks          - Modular content blocks within portfolio versions
9.  tags                    - Portfolio tags/categories
10. portfolio_tags          - Many-to-many: portfolio_versions <-> tags
11. follows                 - User follow relationships
12. likes                   - Portfolio likes

RELATIONSHIP OVERVIEW:
- users 1:N portfolios (user owns portfolios)
- portfolios 1:2 portfolio_versions (max 2 versions: live + draft)
- portfolio_versions 1:N content_blocks
- portfolio_versions N:M tags (via portfolio_tags)
- users N:M users (via follows)
- users N:M portfolio_versions (via likes)
- majors 1:N classes
- academic_years 1:N classes
- classes N:M users (via student_class_history)
*/

-- ============================================================================
-- B. ENTITY RELATIONSHIP DIAGRAM (TEXT-BASED)
-- ============================================================================
/*
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│  academic_years │       │     majors      │       │      tags       │
│─────────────────│       │─────────────────│       │─────────────────│
│ id (PK)         │       │ id (PK)         │       │ id (PK)         │
│ year_start      │       │ name            │       │ name            │
│ is_active       │       │ code            │       │ created_at      │
│ promotion_month │       │ created_at      │       │ updated_at      │
│ promotion_day   │       │ updated_at      │       └─────────────────┘
│ created_at      │       │ deleted_at      │               │
│ updated_at      │       └─────────────────┘               │
└─────────────────┘               │                         │
        │                         │                         │
        │         ┌───────────────┴───────────────┐         │
        │         │                               │         │
        ▼         ▼                               │         │
┌─────────────────────────────┐                   │         │
│          classes            │                   │         │
│─────────────────────────────│                   │         │
│ id (PK)                     │                   │         │
│ academic_year_id (FK)       │                   │         │
│ major_id (FK)               │                   │         │
│ grade_level (10/11/12)      │                   │         │
│ group_letter (A-Z)          │                   │         │
│ name (auto-generated)       │                   │         │
│ created_at, updated_at      │                   │         │
└─────────────────────────────┘                   │         │
        │                                         │         │
        │                                         │         │
        ▼                                         │         │
┌─────────────────────────────┐                   │         │
│   student_class_history     │                   │         │
│─────────────────────────────│                   │         │
│ id (PK)                     │                   │         │
│ user_id (FK)                │◄──────────────────┼─────────┤
│ class_id (FK)               │                   │         │
│ assigned_at                 │                   │         │
│ created_at                  │                   │         │
└─────────────────────────────┘                   │         │
        │                                         │         │
        │                                         │         │
        ▼                                         │         │
┌─────────────────────────────────────────────────┴─────────┴───┐
│                            users                              │
│───────────────────────────────────────────────────────────────│
│ id (PK)                                                       │
│ username, email, password_hash, name                          │
│ avatar_url, banner_url, bio                                   │
│ role (admin/student/alumni)                                   │
│ status (active/graduated/dropped_out/inactive)                │
│ nisn, nis, current_class_id (FK), entry_year, graduation_year │
│ social_links (JSONB)                                          │
│ created_at, updated_at, deleted_at                            │
└───────────────────────────────────────────────────────────────┘
        │                      │                      │
        │ (owns)               │ (follows)            │ (likes)
        ▼                      ▼                      ▼
┌───────────────────┐   ┌───────────────┐    ┌───────────────────┐
│    portfolios     │   │    follows    │    │      likes        │
│───────────────────│   │───────────────│    │───────────────────│
│ id (PK)           │   │ follower_id   │    │ user_id (FK)      │
│ user_id (FK)      │   │ following_id  │    │ portfolio_version │
│ slug              │   │ created_at    │    │ created_at        │
│ created_at        │   └───────────────┘    └───────────────────┘
│ updated_at        │
│ deleted_at        │
└───────────────────┘
        │
        │ (has versions, max 2)
        ▼
┌───────────────────────────────────────────┐
│           portfolio_versions              │
│───────────────────────────────────────────│
│ id (PK)                                   │
│ portfolio_id (FK)                         │
│ version_number (1=live, 2=draft)          │
│ title, thumbnail_url                      │
│ status (draft/pending/rejected/published) │
│ admin_review_note                         │
│ published_at, created_at, updated_at      │
└───────────────────────────────────────────┘
        │                      │
        │ (has blocks)         │ (has tags)
        ▼                      ▼
┌───────────────────┐   ┌───────────────────┐
│  content_blocks   │   │  portfolio_tags   │
│───────────────────│   │───────────────────│
│ id (PK)           │   │ portfolio_version │
│ portfolio_version │   │ tag_id (FK)       │
│ block_type        │   └───────────────────┘
│ block_order       │
│ payload (JSONB)   │
│ created_at        │
│ updated_at        │
└───────────────────┘
*/

-- ============================================================================
-- C. POSTGRESQL SQL (PRODUCTION-READY)
-- ============================================================================

-- --------------------------------------------------------------------------
-- 1. REQUIRED EXTENSIONS
-- --------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search optimization

-- --------------------------------------------------------------------------
-- 2. ENUM TYPES (Domain-specific constrained types)
-- --------------------------------------------------------------------------

-- User roles
CREATE TYPE user_role AS ENUM ('admin', 'student', 'alumni');

-- User account status
CREATE TYPE user_status AS ENUM ('active', 'graduated', 'dropped_out', 'inactive');

-- Portfolio version status
CREATE TYPE portfolio_status AS ENUM ('draft', 'pending_review', 'rejected', 'published', 'archived');

-- Content block types
CREATE TYPE block_type AS ENUM ('text', 'image', 'table', 'youtube', 'button');

-- Grade levels for classes
CREATE TYPE grade_level AS ENUM ('10', '11', '12');

-- --------------------------------------------------------------------------
-- 3. CREATE TABLE STATEMENTS
-- --------------------------------------------------------------------------

-- ==========================================================================
-- TABLE: majors (Jurusan)
-- ==========================================================================
CREATE TABLE majors (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(100) NOT NULL,
    code            VARCHAR(10) NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT majors_code_unique UNIQUE (code),
    CONSTRAINT majors_code_lowercase CHECK (code = LOWER(code)),
    CONSTRAINT majors_code_alpha CHECK (code ~ '^[a-z]+$')
);

COMMENT ON TABLE majors IS 'Academic majors/departments (Jurusan) at SMKN 4 Malang';
COMMENT ON COLUMN majors.code IS 'Lowercase alphabetic code (e.g., rpl, tkj, dkv)';

-- ==========================================================================
-- TABLE: academic_years (Tahun Ajaran)
-- ==========================================================================
CREATE TABLE academic_years (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    year_start      INTEGER NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT FALSE,
    promotion_month INTEGER NOT NULL DEFAULT 7,
    promotion_day   INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT academic_years_year_start_unique UNIQUE (year_start),
    CONSTRAINT academic_years_year_start_valid CHECK (year_start >= 2000),
    CONSTRAINT academic_years_promotion_month_valid CHECK (promotion_month >= 1 AND promotion_month <= 12),
    CONSTRAINT academic_years_promotion_day_valid CHECK (promotion_day >= 1 AND promotion_day <= 31)
);

COMMENT ON TABLE academic_years IS 'Academic year periods with automatic class promotion configuration';
COMMENT ON COLUMN academic_years.year_start IS 'Starting year of academic period (e.g., 2024 for 2024/2025)';
COMMENT ON COLUMN academic_years.is_active IS 'Only one academic year can be active at a time';
COMMENT ON COLUMN academic_years.promotion_month IS 'Month when automatic class promotion occurs (1-12)';
COMMENT ON COLUMN academic_years.promotion_day IS 'Day when automatic class promotion occurs (1-31)';

-- ==========================================================================
-- TABLE: classes (Kelas)
-- ==========================================================================
CREATE TABLE classes (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    academic_year_id    UUID NOT NULL,
    major_id            UUID NOT NULL,
    grade_level         grade_level NOT NULL,
    group_letter        CHAR(1) NOT NULL,
    name                VARCHAR(20) NOT NULL,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMP WITH TIME ZONE,

    -- Foreign Keys
    CONSTRAINT classes_academic_year_fk FOREIGN KEY (academic_year_id)
        REFERENCES academic_years(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT classes_major_fk FOREIGN KEY (major_id)
        REFERENCES majors(id) ON DELETE RESTRICT ON UPDATE CASCADE,

    -- Constraints
    CONSTRAINT classes_group_letter_uppercase CHECK (group_letter = UPPER(group_letter)),
    CONSTRAINT classes_group_letter_alpha CHECK (group_letter ~ '^[A-Z]$'),
    CONSTRAINT classes_unique_per_year UNIQUE (academic_year_id, major_id, grade_level, group_letter)
);

COMMENT ON TABLE classes IS 'Class groups per academic year (e.g., X-RPL-A, XI-TKJ-B)';
COMMENT ON COLUMN classes.grade_level IS 'Grade level: 10, 11, or 12';
COMMENT ON COLUMN classes.group_letter IS 'Class group letter (A-Z)';
COMMENT ON COLUMN classes.name IS 'Auto-generated class name (e.g., X-RPL-A, XII-DKV-B)';

-- ==========================================================================
-- TABLE: users
-- ==========================================================================
CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username            VARCHAR(50) NOT NULL,
    email               VARCHAR(255) NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    name                VARCHAR(100) NOT NULL,
    avatar_url          VARCHAR(500),
    banner_url          VARCHAR(500),
    bio                 TEXT,
    role                user_role NOT NULL DEFAULT 'student',
    status              user_status NOT NULL DEFAULT 'active',
    
    -- Student-specific fields (nullable for admin)
    nisn                VARCHAR(20),
    nis                 VARCHAR(30),
    current_class_id    UUID,
    entry_year          INTEGER,
    graduation_year     INTEGER,
    
    -- Social links stored as JSONB for flexibility
    social_links        JSONB DEFAULT '{}'::jsonb,
    
    -- Timestamps
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMP WITH TIME ZONE,

    -- Foreign Keys
    CONSTRAINT users_current_class_fk FOREIGN KEY (current_class_id)
        REFERENCES classes(id) ON DELETE SET NULL ON UPDATE CASCADE,

    -- Constraints
    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_username_lowercase CHECK (username = LOWER(username)),
    CONSTRAINT users_username_format CHECK (username ~ '^[a-z0-9_]+$'),
    CONSTRAINT users_username_length CHECK (LENGTH(username) >= 3 AND LENGTH(username) <= 50),
    CONSTRAINT users_email_format CHECK (email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'),
    CONSTRAINT users_entry_year_valid CHECK (entry_year IS NULL OR (entry_year >= 2000 AND entry_year <= 2100)),
    CONSTRAINT users_graduation_year_valid CHECK (graduation_year IS NULL OR (graduation_year >= 2000 AND graduation_year <= 2100)),
    
    -- Reserved usernames check (enforced at application level, listed here for documentation)
    CONSTRAINT users_username_not_reserved CHECK (
        username NOT IN (
            'admin', 'dashboard', 'login', 'register', 'api', 'feed', 
            'explore', 'search', 'settings', 'profile', 'logout', 'grafikarsa'
        )
    )
);

COMMENT ON TABLE users IS 'All platform users: admins, students, and alumni';
COMMENT ON COLUMN users.social_links IS 'JSONB containing social profile URLs (facebook, instagram, github, etc.)';
COMMENT ON COLUMN users.current_class_id IS 'Current class assignment; NULL for admin or unassigned students';

-- ==========================================================================
-- TABLE: student_class_history (Riwayat Kelas Siswa)
-- ==========================================================================
CREATE TABLE student_class_history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL,
    class_id        UUID NOT NULL,
    assigned_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT student_class_history_user_fk FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT student_class_history_class_fk FOREIGN KEY (class_id)
        REFERENCES classes(id) ON DELETE RESTRICT ON UPDATE CASCADE,

    -- A student can only be assigned to a specific class once
    CONSTRAINT student_class_history_unique UNIQUE (user_id, class_id)
);

COMMENT ON TABLE student_class_history IS 'Historical record of student class assignments across academic years';

-- ==========================================================================
-- TABLE: tags
-- ==========================================================================
CREATE TABLE tags (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(50) NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT tags_name_unique UNIQUE (name)
);

COMMENT ON TABLE tags IS 'Portfolio categorization tags';

-- ==========================================================================
-- TABLE: portfolios (Portfolio Headers)
-- ==========================================================================
CREATE TABLE portfolios (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL,
    slug            VARCHAR(200) NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE,

    -- Foreign Keys
    CONSTRAINT portfolios_user_fk FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Constraints: slug unique per user
    CONSTRAINT portfolios_slug_unique_per_user UNIQUE (user_id, slug),
    CONSTRAINT portfolios_slug_format CHECK (slug ~ '^[a-z0-9-]+$')
);

COMMENT ON TABLE portfolios IS 'Portfolio header/metadata; actual content is in portfolio_versions';
COMMENT ON COLUMN portfolios.slug IS 'URL-friendly identifier, unique per user';

-- ==========================================================================
-- TABLE: portfolio_versions (Versioned Portfolio Content)
-- ==========================================================================
CREATE TABLE portfolio_versions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id        UUID NOT NULL,
    version_number      INTEGER NOT NULL DEFAULT 1,
    title               VARCHAR(200) NOT NULL,
    thumbnail_url       VARCHAR(500),
    status              portfolio_status NOT NULL DEFAULT 'draft',
    admin_review_note   TEXT,
    published_at        TIMESTAMP WITH TIME ZONE,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT portfolio_versions_portfolio_fk FOREIGN KEY (portfolio_id)
        REFERENCES portfolios(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Constraints: max 2 versions per portfolio (1=live, 2=draft)
    CONSTRAINT portfolio_versions_number_valid CHECK (version_number IN (1, 2)),
    CONSTRAINT portfolio_versions_unique_per_portfolio UNIQUE (portfolio_id, version_number)
);

COMMENT ON TABLE portfolio_versions IS 'Versioned portfolio content; max 2 versions per portfolio';
COMMENT ON COLUMN portfolio_versions.version_number IS '1 = live/published version, 2 = draft/pending version';
COMMENT ON COLUMN portfolio_versions.admin_review_note IS 'Admin feedback or rejection reason';

-- ==========================================================================
-- TABLE: content_blocks (Modular Content Blocks)
-- ==========================================================================
CREATE TABLE content_blocks (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_version_id    UUID NOT NULL,
    block_type              block_type NOT NULL,
    block_order             INTEGER NOT NULL DEFAULT 0,
    payload                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT content_blocks_portfolio_version_fk FOREIGN KEY (portfolio_version_id)
        REFERENCES portfolio_versions(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Constraints
    CONSTRAINT content_blocks_order_positive CHECK (block_order >= 0)
);

COMMENT ON TABLE content_blocks IS 'Modular content blocks within portfolio versions';
COMMENT ON COLUMN content_blocks.block_type IS 'Type: text, image, table, youtube, button';
COMMENT ON COLUMN content_blocks.block_order IS 'Display order (0-indexed)';
COMMENT ON COLUMN content_blocks.payload IS 'Block-specific content as JSONB';

-- ==========================================================================
-- TABLE: portfolio_tags (Many-to-Many: portfolio_versions <-> tags)
-- ==========================================================================
CREATE TABLE portfolio_tags (
    portfolio_version_id    UUID NOT NULL,
    tag_id                  UUID NOT NULL,
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Composite Primary Key
    PRIMARY KEY (portfolio_version_id, tag_id),

    -- Foreign Keys
    CONSTRAINT portfolio_tags_version_fk FOREIGN KEY (portfolio_version_id)
        REFERENCES portfolio_versions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT portfolio_tags_tag_fk FOREIGN KEY (tag_id)
        REFERENCES tags(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE portfolio_tags IS 'Many-to-many relationship between portfolio versions and tags';

-- ==========================================================================
-- TABLE: follows (User Follow Relationships)
-- ==========================================================================
CREATE TABLE follows (
    follower_id     UUID NOT NULL,
    following_id    UUID NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Composite Primary Key
    PRIMARY KEY (follower_id, following_id),

    -- Foreign Keys
    CONSTRAINT follows_follower_fk FOREIGN KEY (follower_id)
        REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT follows_following_fk FOREIGN KEY (following_id)
        REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Cannot follow yourself
    CONSTRAINT follows_no_self_follow CHECK (follower_id != following_id)
);

COMMENT ON TABLE follows IS 'User follow relationships (follower follows following)';

-- ==========================================================================
-- TABLE: likes (Portfolio Likes)
-- ==========================================================================
CREATE TABLE likes (
    user_id                 UUID NOT NULL,
    portfolio_version_id    UUID NOT NULL,
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Composite Primary Key
    PRIMARY KEY (user_id, portfolio_version_id),

    -- Foreign Keys
    CONSTRAINT likes_user_fk FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT likes_portfolio_version_fk FOREIGN KEY (portfolio_version_id)
        REFERENCES portfolio_versions(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE likes IS 'User likes on published portfolio versions';

-- --------------------------------------------------------------------------
-- 4. INDEX DEFINITIONS
-- --------------------------------------------------------------------------

-- Majors
CREATE INDEX idx_majors_code ON majors(code) WHERE deleted_at IS NULL;
CREATE INDEX idx_majors_deleted_at ON majors(deleted_at) WHERE deleted_at IS NOT NULL;

-- Academic Years
CREATE INDEX idx_academic_years_is_active ON academic_years(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_academic_years_year_start ON academic_years(year_start);

-- Classes
CREATE INDEX idx_classes_academic_year ON classes(academic_year_id);
CREATE INDEX idx_classes_major ON classes(major_id);
CREATE INDEX idx_classes_grade_level ON classes(grade_level);
CREATE INDEX idx_classes_deleted_at ON classes(deleted_at) WHERE deleted_at IS NOT NULL;

-- Users
CREATE INDEX idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_current_class ON users(current_class_id) WHERE current_class_id IS NOT NULL;
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;

-- Full-text search indexes for users (using pg_trgm)
CREATE INDEX idx_users_name_trgm ON users USING gin(name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_username_trgm ON users USING gin(username gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_bio_trgm ON users USING gin(bio gin_trgm_ops) WHERE deleted_at IS NULL AND bio IS NOT NULL;

-- Student Class History
CREATE INDEX idx_student_class_history_user ON student_class_history(user_id);
CREATE INDEX idx_student_class_history_class ON student_class_history(class_id);
CREATE INDEX idx_student_class_history_assigned_at ON student_class_history(assigned_at);

-- Tags
CREATE INDEX idx_tags_name ON tags(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_tags_deleted_at ON tags(deleted_at) WHERE deleted_at IS NOT NULL;

-- Portfolios
CREATE INDEX idx_portfolios_user ON portfolios(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_portfolios_slug ON portfolios(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_portfolios_deleted_at ON portfolios(deleted_at) WHERE deleted_at IS NOT NULL;

-- Portfolio Versions
CREATE INDEX idx_portfolio_versions_portfolio ON portfolio_versions(portfolio_id);
CREATE INDEX idx_portfolio_versions_status ON portfolio_versions(status);
CREATE INDEX idx_portfolio_versions_published_at ON portfolio_versions(published_at) WHERE published_at IS NOT NULL;

-- Full-text search for portfolio titles
CREATE INDEX idx_portfolio_versions_title_trgm ON portfolio_versions USING gin(title gin_trgm_ops);

-- Content Blocks
CREATE INDEX idx_content_blocks_portfolio_version ON content_blocks(portfolio_version_id);
CREATE INDEX idx_content_blocks_order ON content_blocks(portfolio_version_id, block_order);

-- Portfolio Tags
CREATE INDEX idx_portfolio_tags_tag ON portfolio_tags(tag_id);

-- Follows
CREATE INDEX idx_follows_follower ON follows(follower_id);
CREATE INDEX idx_follows_following ON follows(following_id);

-- Likes
CREATE INDEX idx_likes_user ON likes(user_id);
CREATE INDEX idx_likes_portfolio_version ON likes(portfolio_version_id);

-- --------------------------------------------------------------------------
-- 5. TRIGGERS AND FUNCTIONS
-- --------------------------------------------------------------------------

-- ==========================================================================
-- Function: Auto-update updated_at timestamp
-- ==========================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to all relevant tables
CREATE TRIGGER trigger_majors_updated_at
    BEFORE UPDATE ON majors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_academic_years_updated_at
    BEFORE UPDATE ON academic_years
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_classes_updated_at
    BEFORE UPDATE ON classes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_tags_updated_at
    BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_portfolios_updated_at
    BEFORE UPDATE ON portfolios
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_portfolio_versions_updated_at
    BEFORE UPDATE ON portfolio_versions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_content_blocks_updated_at
    BEFORE UPDATE ON content_blocks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==========================================================================
-- Function: Ensure only one active academic year
-- ==========================================================================
CREATE OR REPLACE FUNCTION ensure_single_active_academic_year()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_active = TRUE THEN
        UPDATE academic_years SET is_active = FALSE WHERE id != NEW.id AND is_active = TRUE;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_single_active_academic_year
    BEFORE INSERT OR UPDATE ON academic_years
    FOR EACH ROW
    WHEN (NEW.is_active = TRUE)
    EXECUTE FUNCTION ensure_single_active_academic_year();

-- ==========================================================================
-- Function: Auto-generate class name
-- ==========================================================================
CREATE OR REPLACE FUNCTION generate_class_name()
RETURNS TRIGGER AS $$
DECLARE
    grade_roman VARCHAR(4);
    major_code_upper VARCHAR(10);
BEGIN
    -- Convert grade level to Roman numerals
    CASE NEW.grade_level
        WHEN '10' THEN grade_roman := 'X';
        WHEN '11' THEN grade_roman := 'XI';
        WHEN '12' THEN grade_roman := 'XII';
    END CASE;
    
    -- Get major code and uppercase it
    SELECT UPPER(code) INTO major_code_upper FROM majors WHERE id = NEW.major_id;
    
    -- Generate name: X-RPL-A, XI-TKJ-B, XII-DKV-C
    NEW.name := grade_roman || '-' || major_code_upper || '-' || NEW.group_letter;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_generate_class_name
    BEFORE INSERT OR UPDATE ON classes
    FOR EACH ROW EXECUTE FUNCTION generate_class_name();

-- ==========================================================================
-- Function: Set published_at when status changes to 'published'
-- ==========================================================================
CREATE OR REPLACE FUNCTION set_published_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'published' AND (OLD.status IS NULL OR OLD.status != 'published') THEN
        NEW.published_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_published_at
    BEFORE INSERT OR UPDATE ON portfolio_versions
    FOR EACH ROW EXECUTE FUNCTION set_published_at();

-- ==========================================================================
-- Function: Enforce max 2 versions per portfolio
-- ==========================================================================
CREATE OR REPLACE FUNCTION enforce_max_portfolio_versions()
RETURNS TRIGGER AS $$
DECLARE
    version_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO version_count 
    FROM portfolio_versions 
    WHERE portfolio_id = NEW.portfolio_id;
    
    IF version_count >= 2 THEN
        RAISE EXCEPTION 'A portfolio cannot have more than 2 versions';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_enforce_max_portfolio_versions
    BEFORE INSERT ON portfolio_versions
    FOR EACH ROW EXECUTE FUNCTION enforce_max_portfolio_versions();

-- ==========================================================================
-- Function: Daily portfolio creation rate limit check (helper for application)
-- This is meant to be called by the application, not as a trigger
-- ==========================================================================
CREATE OR REPLACE FUNCTION check_daily_portfolio_limit(p_user_id UUID, p_limit INTEGER DEFAULT 10)
RETURNS BOOLEAN AS $$
DECLARE
    today_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO today_count
    FROM portfolios
    WHERE user_id = p_user_id
      AND created_at >= CURRENT_DATE
      AND created_at < CURRENT_DATE + INTERVAL '1 day';
    
    RETURN today_count < p_limit;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION check_daily_portfolio_limit IS 'Returns TRUE if user can create more portfolios today';

-- ============================================================================
-- D. DESIGN ASSUMPTIONS & IMPORTANT NOTES
-- ============================================================================
/*
ASSUMPTIONS MADE:

1. PORTFOLIO VERSIONING:
   - Max 2 versions per portfolio (version_number 1 or 2)
   - Version 1 is typically the live/published version
   - Version 2 is the draft/pending version when editing a published portfolio
   - When draft is approved, it overwrites version 1 (handled by application logic)

2. SOCIAL LINKS JSONB STRUCTURE:
   Expected format:
   {
     "facebook": "https://facebook.com/username",
     "instagram": "https://instagram.com/username",
     "github": "https://github.com/username",
     "linkedin": "https://linkedin.com/in/username",
     "twitter": "https://twitter.com/username",
     "website": "https://example.com",
     "tiktok": "https://tiktok.com/@username",
     "youtube": "https://youtube.com/@username",
     "behance": "https://behance.net/username",
     "dribbble": "https://dribbble.com/username",
     "threads": "https://threads.net/@username",
     "bluesky": "https://bsky.app/profile/username",
     "medium": "https://medium.com/@username",
     "gitlab": "https://gitlab.com/username"
   }

3. CONTENT BLOCK PAYLOAD STRUCTURES:
   - text: { "content": "<p>Rich HTML content</p>" }
   - image: { "url": "https://...", "caption": "Optional caption" }
   - table: { "headers": ["Col1", "Col2"], "rows": [["A", "B"], ["C", "D"]] }
   - youtube: { "video_id": "dQw4w9WgXcQ" }
   - button: { "label": "Click Me", "url": "https://..." }

4. SOFT DELETE:
   - Tables with deleted_at support soft delete
   - Application should filter WHERE deleted_at IS NULL for normal queries
   - Unique constraints consider deleted_at where applicable

5. CLASS PROMOTION:
   - Handled by external cronjob, not database triggers
   - Cronjob reads promotion_month/day from active academic_year
   - Creates new student_class_history entries and updates current_class_id

6. RATE LIMITING:
   - Daily portfolio limit (10/day) checked via check_daily_portfolio_limit() function
   - Application must call this before allowing portfolio creation

7. LIKES:
   - Likes are on portfolio_versions, not portfolios
   - Only published versions should be likeable (enforced by application)

TRADE-OFFS:

1. JSONB for social_links:
   - Pros: Flexible, easy to add new platforms without migration
   - Cons: No strict schema validation at DB level
   - Mitigation: Application-level validation

2. Separate portfolio_versions table:
   - Pros: Clean versioning, easy to query latest version
   - Cons: Slightly more complex queries
   - Mitigation: Views can simplify common queries if needed

3. Composite primary keys on junction tables:
   - Pros: Prevents duplicate relationships, no extra surrogate key
   - Cons: Slightly longer foreign key references
   - Decision: Worth it for data integrity

IMPORTANT INDEXES JUSTIFICATION:

- GIN trigram indexes on users.name, users.username, users.bio:
  For fuzzy text search across user profiles

- Partial indexes with WHERE deleted_at IS NULL:
  Optimize queries that filter out soft-deleted records

- Composite index on content_blocks(portfolio_version_id, block_order):
  Optimize ordering of blocks within a portfolio version
*/

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================