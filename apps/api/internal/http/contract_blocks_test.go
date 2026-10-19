package http

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// TestBlockSchemasSinkron menjaga agar JSON Schema di
// packages/contracts/blocks/*.json tidak pernah basi dibanding validator Go:
// setiap tipe block yang diizinkan harus punya file schema dengan field wajib
// yang sama. Kalau validator berubah tanpa memperbarui kontrak (atau sebaliknya)
// test ini gagal.
func TestBlockSchemasSinkron(t *testing.T) {
	dir := contractsBlocksDir(t)
	if dir == "" {
		t.Skip("direktori packages/contracts/blocks tidak ditemukan")
	}

	// 1. Setiap tipe block yang dikenal harus punya schema.
	for blockType := range allowedBlockTypes {
		p := filepath.Join(dir, blockType+".json")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("block %q tidak punya schema %s", blockType, p)
		}
	}

	// 2. Tidak ada schema yatim (tipe yang tidak dikenal kode).
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("baca %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		bt := e.Name()[:len(e.Name())-len(".json")]
		if !allowedBlockTypes[bt] {
			t.Errorf("schema %s tidak punya tipe block di kode", e.Name())
		}
	}

	// 3. Field wajib pada schema harus benar-benar diminta validator Go, dan
	// payload minimal valid menurut schema harus diterima validator.
	for blockType := range allowedBlockTypes {
		raw, err := os.ReadFile(filepath.Join(dir, blockType+".json"))
		if err != nil {
			continue
		}
		var schema struct {
			Required []string `json:"required"`
			AnyOf    []struct {
				Required []string `json:"required"`
			} `json:"anyOf"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Errorf("schema %s tidak valid JSON: %v", blockType, err)
			continue
		}
		sample := samplePayload(blockType)
		if sample == nil {
			continue
		}
		if msg, ok := validateBlockPayload(blockType, sample); !ok {
			t.Errorf("payload contoh untuk %s ditolak validator: %s", blockType, msg)
		}
		// Field wajib schema (bila ada) harus muncul di contoh.
		for _, req := range schema.Required {
			if _, ok := sample[req]; !ok {
				t.Errorf("schema %s mewajibkan %q tapi contoh kode tidak punya", blockType, req)
			}
		}
		// Tanpa field wajib, validator harus menolak.
		if len(schema.Required) > 0 {
			if _, ok := validateBlockPayload(blockType, map[string]any{}); ok {
				t.Errorf("validator menerima %s tanpa field wajib %v", blockType, schema.Required)
			}
		}
	}
}

// samplePayload menghasilkan payload minimal valid per tipe block.
func samplePayload(blockType string) map[string]any {
	switch blockType {
	case "text", "heading":
		return map[string]any{"markdown": "contoh"}
	case "image", "gallery":
		return map[string]any{"url": "https://example.com/a.png", "alt": "contoh gambar"}
	case "table":
		return map[string]any{"headers": []any{"H1"}, "rows": []any{}}
	case "youtube":
		return map[string]any{"video_id": "dQw4w9WgXcQ"}
	case "button":
		return map[string]any{"label": "Buka", "url": "https://example.com"}
	case "link":
		return map[string]any{"url": "https://example.com", "title": "contoh"}
	case "code":
		return map[string]any{"code": "print(1)"}
	default:
		return map[string]any{"url": "https://codepen.io/pen/abc"}
	}
}

func contractsBlocksDir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "..", "packages", "contracts", "blocks"),
		filepath.Join("..", "..", "..", "..", "packages", "contracts", "blocks"),
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Dir(file)
		candidates = append(candidates, filepath.Join(dir, "..", "..", "..", "packages", "contracts", "blocks"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

// TestAllowedBlockTypesTerurut memastikan daftar tipe block deterministik
// (dipakai juga oleh dokumen kontrak).
func TestAllowedBlockTypesTerurut(t *testing.T) {
	types := make([]string, 0, len(allowedBlockTypes))
	for k := range allowedBlockTypes {
		types = append(types, k)
	}
	sort.Strings(types)
	if len(types) == 0 {
		t.Fatal("allowedBlockTypes kosong")
	}
	for _, ty := range types {
		if blockTypeAllowed(ty) == "" {
			t.Errorf("tipe %q ada di allowedBlockTypes tapi ditolak blockTypeAllowed", ty)
		}
	}
}
