-- +goose Up
-- =============================================================================
-- 10. MODERASI & LAPORAN
-- =============================================================================
CREATE TABLE moderation_reasons (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  slug        citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,60}$'),
  label       text NOT NULL CHECK (char_length(label) BETWEEN 2 AND 80),
  description text CHECK (char_length(description) <= 500),
  applies_to  text[] NOT NULL CHECK (applies_to <@ ARRAY['content', 'comment', 'profile', 'group', 'report']::text[]),
  sort_order  smallint NOT NULL DEFAULT 0,
  is_active   boolean NOT NULL DEFAULT true
);

CREATE TABLE moderation_cases (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  content_id        uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  version_id        uuid NOT NULL UNIQUE REFERENCES content_versions(id) ON DELETE CASCADE,
  kind              text NOT NULL CHECK (kind IN ('initial', 'resubmission', 'revision')),
  queue_major_id    uuid REFERENCES majors(id),        -- NULL = antrian global
  status            text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'claimed', 'decided', 'withdrawn')),
  claimed_by        uuid REFERENCES users(id),
  claimed_at        timestamptz,
  claim_expires_at  timestamptz,
  decision          text CHECK (decision IN ('approved', 'changes_requested', 'rejected')),
  reason_id         uuid REFERENCES moderation_reasons(id),
  decision_note     text CHECK (char_length(decision_note) <= 2000),
  decided_by        uuid REFERENCES users(id),
  decided_at        timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now(),
  CHECK ((status = 'claimed') = (claimed_by IS NOT NULL AND claim_expires_at IS NOT NULL) OR status = 'decided'),
  CHECK ((status = 'decided') = (decision IS NOT NULL)),
  CHECK (decision IS NULL OR decided_by IS NOT NULL),
  CHECK (decision NOT IN ('changes_requested', 'rejected') OR decision_note IS NOT NULL),
  CHECK (decision <> 'rejected' OR reason_id IS NOT NULL)
);
CREATE INDEX ix_mod_cases_queue ON moderation_cases (queue_major_id, created_at) WHERE status IN ('open', 'claimed');
CREATE INDEX ix_mod_cases_claim_expiry ON moderation_cases (claim_expires_at) WHERE status = 'claimed';
CREATE INDEX ix_mod_cases_decided ON moderation_cases (decided_by, decided_at DESC) WHERE status = 'decided';

-- Catatan per blok / node saat minta revisi.
CREATE TABLE moderation_annotations (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  case_id     uuid NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
  target_ref  text NOT NULL CHECK (char_length(target_ref) <= 100),   -- id blok / id node / "field:title"
  note        text NOT NULL CHECK (char_length(note) BETWEEN 1 AND 1000),
  created_by  uuid NOT NULL REFERENCES users(id),
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_mod_annotations_case ON moderation_annotations (case_id);

-- Banding: maksimal satu kali per penolakan.
CREATE TABLE appeals (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  case_id       uuid NOT NULL UNIQUE REFERENCES moderation_cases(id) ON DELETE CASCADE,
  content_id    uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  submitted_by  uuid NOT NULL REFERENCES users(id),
  message       text NOT NULL CHECK (char_length(message) BETWEEN 10 AND 2000),
  status        text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'upheld', 'overturned')),
  decided_by    uuid REFERENCES users(id),
  decided_at    timestamptz,
  decision_note text CHECK (char_length(decision_note) <= 2000),
  created_at    timestamptz NOT NULL DEFAULT now(),
  CHECK ((status = 'open') = (decided_at IS NULL))
);
CREATE INDEX ix_appeals_open ON appeals (created_at) WHERE status = 'open';

CREATE TABLE reports (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  reporter_id  uuid NOT NULL REFERENCES users(id),
  target_type  text NOT NULL CHECK (target_type IN ('content', 'comment', 'profile', 'group')),
  target_id    uuid NOT NULL,
  reason_id    uuid NOT NULL REFERENCES moderation_reasons(id),
  note         text CHECK (char_length(note) <= 500),
  status       text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'actioned', 'dismissed')),
  handled_by   uuid REFERENCES users(id),
  handled_at   timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now(),
  CHECK ((status = 'open') = (handled_at IS NULL))
);
CREATE UNIQUE INDEX uq_reports_open ON reports (reporter_id, target_type, target_id) WHERE status = 'open';
CREATE INDEX ix_reports_queue ON reports (created_at) WHERE status = 'open';
CREATE INDEX ix_reports_target ON reports (target_type, target_id);

-- Tindakan moderator di luar keputusan moderation_cases (takedown, sembunyikan komentar, reset profil).
CREATE TABLE moderation_actions (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  target_type  text NOT NULL CHECK (target_type IN ('content', 'comment', 'profile', 'experience', 'group')),
  target_id    uuid NOT NULL,
  action       text NOT NULL CHECK (action IN (
                 'take_down', 'hide_comment', 'restore_comment', 'reset_avatar', 'reset_banner',
                 'reset_bio', 'reset_headline', 'remove_experience', 'reset_group_avatar',
                 'reset_group_banner', 'reset_group_description', 'dismiss')),
  reason_id    uuid REFERENCES moderation_reasons(id),
  note         text CHECK (char_length(note) <= 2000),
  report_id    uuid REFERENCES reports(id),
  actor_id     uuid NOT NULL REFERENCES users(id),
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_mod_actions_target ON moderation_actions (target_type, target_id, created_at DESC);

CREATE TABLE word_filters (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  pattern     text NOT NULL UNIQUE CHECK (char_length(pattern) BETWEEN 2 AND 100),
  match_kind  text NOT NULL DEFAULT 'word' CHECK (match_kind IN ('word', 'substring')),
  action      text NOT NULL CHECK (action IN ('block', 'flag')),   -- block = tolak simpan, flag = laporan otomatis
  created_by  uuid REFERENCES users(id),
  created_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS word_filters;
DROP TABLE IF EXISTS moderation_actions;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS appeals;
DROP TABLE IF EXISTS moderation_annotations;
DROP TABLE IF EXISTS moderation_cases;
DROP TABLE IF EXISTS moderation_reasons;
