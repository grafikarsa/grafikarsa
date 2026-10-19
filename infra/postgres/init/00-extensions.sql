-- Ekstensi yang dibutuhkan API (idempoten, aman di-rerun).
-- Migrasi goose 00001 juga membuatnya; file ini untuk fresh volume
-- yang di-mount sebelum migrasi jalan.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
