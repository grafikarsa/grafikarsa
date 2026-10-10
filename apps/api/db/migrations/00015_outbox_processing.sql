-- +goose Up
-- Status 'processing' dipakai saat event di-claim: mencegah event hilang bila
-- worker crash di tengah proses (sebelumnya langsung ditandai 'sent').
ALTER TABLE outbox DROP CONSTRAINT IF EXISTS outbox_status_valid;
ALTER TABLE outbox ADD CONSTRAINT outbox_status_valid
  CHECK (status IN ('pending', 'processing', 'sent', 'failed'));

-- +goose Down
-- Kembalikan baris 'processing' ke 'pending' dulu, kalau tidak constraint baru
-- ditolak saat rollback (keadaan normal setelah dispatcher jalan).
UPDATE outbox SET status = 'pending' WHERE status = 'processing';
ALTER TABLE outbox DROP CONSTRAINT IF EXISTS outbox_status_valid;
ALTER TABLE outbox ADD CONSTRAINT outbox_status_valid
  CHECK (status IN ('pending', 'sent', 'failed'));
