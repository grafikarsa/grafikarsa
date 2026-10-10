-- +goose Up
-- Status 'processing' dipakai saat event di-claim: mencegah event hilang bila
-- worker crash di tengah proses (sebelumnya langsung ditandai 'sent').
ALTER TABLE outbox DROP CONSTRAINT IF EXISTS outbox_status_valid;
ALTER TABLE outbox ADD CONSTRAINT outbox_status_valid
  CHECK (status IN ('pending', 'processing', 'sent', 'failed'));

-- +goose Down
ALTER TABLE outbox DROP CONSTRAINT IF EXISTS outbox_status_valid;
ALTER TABLE outbox ADD CONSTRAINT outbox_status_valid
  CHECK (status IN ('pending', 'sent', 'failed'));
