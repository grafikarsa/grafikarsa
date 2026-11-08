-- +goose Up
-- =============================================================================
-- 3b. BRANKAS SANDI AWAL (untuk cetak kartu akun)
-- =============================================================================
-- Sandi awal disimpan terenkripsi (AES-256-GCM) hanya agar admin dapat
-- mencetak kartu akun. Baris dihapus begitu akun diaktivasi atau sandi awal
-- diterbitkan ulang/diganti.
CREATE TABLE initial_password_vault (
  user_id     uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  ciphertext  bytea NOT NULL,           -- AES-256-GCM: nonce || sealed
  expires_at  timestamptz NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_vault_expiry ON initial_password_vault (expires_at);

-- +goose Down
DROP TABLE IF EXISTS initial_password_vault;
