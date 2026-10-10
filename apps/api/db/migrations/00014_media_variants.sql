-- +goose Up
-- Varian lebar gambar (mis. {"320": "thumbnails/x_320.jpg"}) agar serve bisa
-- memilih varian tanpa resize on-the-fly.
ALTER TABLE media_assets ADD COLUMN variants JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE media_assets DROP COLUMN IF EXISTS variants;
