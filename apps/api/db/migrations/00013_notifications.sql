-- +goose Up
-- =============================================================================
-- 12. NOTIFIKASI
-- =============================================================================
CREATE TABLE notifications (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  recipient_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type          text NOT NULL CHECK (type IN (
                  'collab_invited', 'collab_accepted', 'collab_declined', 'collab_removed',
                  'ownership_transferred', 'moderation_approved', 'moderation_changes_requested',
                  'moderation_rejected', 'revision_approved', 'revision_changes_requested',
                  'revision_rejected', 'content_taken_down', 'appeal_decided', 'comment_on_content',
                  'comment_reply', 'comment_mention', 'appreciation', 'new_follower', 'badge_awarded',
                  'report_resolved', 'profile_reset', 'group_manager_assigned', 'class_changed',
                  'password_changed', 'recovery_codes_used', 'password_reset_by_admin')),
  -- Kunci agregasi: notifikasi belum dibaca dengan kunci sama digabung
  -- (mis. "appreciation:<content_id>" → "Rafa dan 12 lainnya mengapresiasi karyamu").
  group_key     text NOT NULL,
  actor_id      uuid REFERENCES users(id) ON DELETE SET NULL,   -- aktor terakhir
  actor_count   integer NOT NULL DEFAULT 1 CHECK (actor_count >= 1),
  subject_type  text CHECK (subject_type IN ('content', 'comment', 'user', 'group', 'badge', 'appeal', 'report')),
  subject_id    uuid,
  data          jsonb NOT NULL DEFAULT '{}'::jsonb,
  read_at       timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_notifications_unread_group ON notifications (recipient_id, group_key) WHERE read_at IS NULL;
CREATE INDEX ix_notifications_recipient ON notifications (recipient_id, updated_at DESC, id DESC);
CREATE INDEX ix_notifications_unread ON notifications (recipient_id) WHERE read_at IS NULL;
CREATE INDEX ix_notifications_retention ON notifications (created_at);

CREATE TABLE notification_preferences (
  user_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type     text NOT NULL,
  enabled  boolean NOT NULL,
  PRIMARY KEY (user_id, type)
);

-- +goose Down
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
