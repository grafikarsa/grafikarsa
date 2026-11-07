-- Seed 0003: setting default aplikasi (settings).
-- Idempoten: value di-upsert berdasarkan key.
--
-- Catatan: docs/revamp/seed.md menulis `media.max_model3d_mb`, tetapi CHECK
-- pada kolom key hanya mengizinkan huruf dan underscore
-- (`^[a-z_]+(\.[a-z_]+)*$`), sehingga digit "3" tidak valid. Key yang dipakai
-- di sini adalah `media.max_model_mb` dengan nilai yang sama (30 MB).
BEGIN;

INSERT INTO settings (key, value) VALUES
  ('collab.max_authors_portfolio',      '50'::jsonb),
  ('collab.max_authors_blog',           '10'::jsonb),
  ('collab.invite_ttl_days',            '14'::jsonb),
  ('content.max_active_drafts',         '20'::jsonb),
  ('content.max_in_review',             '5'::jsonb),
  ('content.trash_retention_days',      '30'::jsonb),
  ('profile.max_pinned',                '3'::jsonb),
  ('profile.max_links',                 '10'::jsonb),
  ('profile.max_skills',                '20'::jsonb),
  ('profile.username_change_days',      '30'::jsonb),
  ('profile.username_redirect_days',    '90'::jsonb),
  ('moderation.claim_ttl_minutes',      '30'::jsonb),
  ('moderation.sla_hours',              '48'::jsonb),
  ('auth.initial_password_ttl_days',    '30'::jsonb),
  ('auth.temporary_password_ttl_hours', '72'::jsonb),
  ('auth.recovery_code_count',          '8'::jsonb),
  ('auth.session_idle_days',            '7'::jsonb),
  ('auth.session_absolute_days',        '30'::jsonb),
  ('promotion.reminder_days',           '[30, 7]'::jsonb),
  ('promotion.rollback_days',           '14'::jsonb),
  ('media.max_image_mb',                '10'::jsonb),
  ('media.max_pdf_mb',                  '25'::jsonb),
  ('media.max_model_mb',                '30'::jsonb),
  ('media.daily_upload_mb',             '1024'::jsonb),
  ('media.orphan_days',                 '7'::jsonb),
  ('comment.edit_window_minutes',       '15'::jsonb),
  ('algo.weights',                      '{"view":1,"appreciation":4,"comment":6,"bookmark":5,"share":3}'::jsonb),
  ('algo.trending',                     '{"half_life_hours":24,"signal_window_days":7,"max_content_age_days":14,"audience_factor":0.25,"min_unique_actors_72h":3,"interval_minutes":15}'::jsonb),
  ('algo.popular',                      '{"periods":["7d","30d","all"]}'::jsonb),
  ('ratelimit.login',                   '{"per_identity":{"limit":5,"window_seconds":900},"per_ip":{"limit":20,"window_seconds":900}}'::jsonb),
  ('ratelimit.password_recover',        '{"limit":5,"window_seconds":3600}'::jsonb),
  ('ratelimit.password_change',         '{"limit":10,"window_seconds":3600}'::jsonb),
  ('ratelimit.comment',                 '{"per_minute":10,"per_day":200}'::jsonb),
  ('ratelimit.interaction',             '{"limit":60,"window_seconds":60}'::jsonb),
  ('ratelimit.collab_invite',           '{"limit":50,"window_seconds":3600}'::jsonb),
  ('ratelimit.report',                  '{"limit":20,"window_seconds":3600}'::jsonb),
  ('ratelimit.upload',                  '{"limit":60,"window_seconds":3600}'::jsonb),
  ('ratelimit.search',                  '{"limit":30,"window_seconds":60}'::jsonb),
  ('ratelimit.beacon',                  '{"limit":120,"window_seconds":60}'::jsonb),
  ('ratelimit.read',                    '{"limit":600,"window_seconds":60}'::jsonb)
ON CONFLICT (key) DO UPDATE
SET value      = EXCLUDED.value,
    updated_at = now();

COMMIT;
