package body

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCanonicalHash_ObjectKeyOrderDoesNotMatter(t *testing.T) {
	t.Parallel()

	first, err := CanonicalHash(map[string]any{"b": 2, "a": 1, "nested": map[string]any{"y": true, "x": "v"}})
	if err != nil {
		t.Fatalf("CanonicalHash: %v", err)
	}
	second, err := CanonicalHash(map[string]any{"nested": map[string]any{"x": "v", "y": true}, "a": 1, "b": 2})
	if err != nil {
		t.Fatalf("CanonicalHash: %v", err)
	}
	if len(first) != 32 {
		t.Fatalf("hash length = %d, want 32", len(first))
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("hashes differ for differently ordered keys")
	}
}

func TestCanonicalHash_RawMessageIgnoresWhitespaceAndKeyOrder(t *testing.T) {
	t.Parallel()

	first, err := CanonicalHash(json.RawMessage(`{"b":2, "a":1}`))
	if err != nil {
		t.Fatalf("CanonicalHash: %v", err)
	}
	second, err := CanonicalHash(json.RawMessage(`{"a":1,"b":2}`))
	if err != nil {
		t.Fatalf("CanonicalHash: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("raw message hashes differ")
	}
}

func TestCanonicalHash_StructKeysSorted(t *testing.T) {
	t.Parallel()

	type fixture struct {
		B int `json:"b"`
		A int `json:"a"`
	}
	fromStruct, err := CanonicalHash(fixture{B: 2, A: 1})
	if err != nil {
		t.Fatalf("CanonicalHash: %v", err)
	}
	fromMap, err := CanonicalHash(map[string]any{"a": 1, "b": 2})
	if err != nil {
		t.Fatalf("CanonicalHash: %v", err)
	}
	if !bytes.Equal(fromStruct, fromMap) {
		t.Fatalf("struct and map hashes differ")
	}
}

func TestCanonicalHash_DifferentValuesDiffer(t *testing.T) {
	t.Parallel()

	first, err := CanonicalHash(map[string]any{"a": 1})
	if err != nil {
		t.Fatalf("CanonicalHash: %v", err)
	}
	second, err := CanonicalHash(map[string]any{"a": 2})
	if err != nil {
		t.Fatalf("CanonicalHash: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatalf("hashes should differ")
	}
}

func TestCanonicalHash_ErrorsOnUnsupportedValue(t *testing.T) {
	t.Parallel()
	if _, err := CanonicalHash(make(chan int)); err == nil {
		t.Fatalf("expected error for channel value")
	}
	if _, err := CanonicalHash(json.RawMessage(`{"a":`)); err == nil {
		t.Fatalf("expected error for invalid raw message")
	}
}

func TestAutoExcerpt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"short", "Halo dunia", "Halo dunia"},
		{"exactly 300", strings.Repeat("a", 300), strings.Repeat("a", 300)},
		{"no word boundary", strings.Repeat("a", 301), strings.Repeat("a", 300) + "…"},
		{
			"cuts at word boundary",
			strings.Repeat("kata ", 60) + "terakhir",
			strings.TrimSpace(strings.Repeat("kata ", 60))[:300-5] + "…",
		},
	}
	// Build the expected word-boundary case explicitly: the first 300 runes end
	// inside "kata"; the excerpt stops before the partial word.
	tests[4].want = strings.TrimRight(strings.Repeat("kata ", 60), " ") + "…"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := autoExcerpt(tc.in)
			if got != tc.want {
				t.Fatalf("autoExcerpt(%d runes) = %q, want %q", len([]rune(tc.in)), got, tc.want)
			}
		})
	}

	t.Run("space at boundary", func(t *testing.T) {
		t.Parallel()
		in := strings.Repeat("a", 300) + " " + strings.Repeat("b", 50)
		want := strings.Repeat("a", 300) + "…"
		if got := autoExcerpt(in); got != want {
			t.Fatalf("autoExcerpt = %q, want %q", got, want)
		}
	})
}

func TestWordCountAndReadingMinutes(t *testing.T) {
	t.Parallel()

	if got := wordCount("  satu\ndua\t tiga  "); got != 3 {
		t.Fatalf("wordCount = %d, want 3", got)
	}
	if got := wordCount("satu\u00a0dua"); got != 2 {
		t.Fatalf("wordCount with nbsp = %d, want 2", got)
	}
	if got := readingMinutes(Blog, 0); got != 1 {
		t.Fatalf("blog readingMinutes(0) = %d, want 1", got)
	}
	if got := readingMinutes(Blog, 200); got != 1 {
		t.Fatalf("blog readingMinutes(200) = %d, want 1", got)
	}
	if got := readingMinutes(Blog, 201); got != 2 {
		t.Fatalf("blog readingMinutes(201) = %d, want 2", got)
	}
	if got := readingMinutes(Portfolio, 500); got != 0 {
		t.Fatalf("portfolio readingMinutes = %d, want 0", got)
	}
}

func TestSanitizeText(t *testing.T) {
	t.Parallel()
	if got := sanitizeText("a\x00b\x07c\x7f\nd\te"); got != "abc\nd\te" {
		t.Fatalf("sanitizeText = %q", got)
	}
	if got := sanitizeText("clean"); got != "clean" {
		t.Fatalf("sanitizeText clean = %q", got)
	}
}

func TestCollapseSpace(t *testing.T) {
	t.Parallel()
	if got := collapseSpace("  a \n b\t\tc  "); got != "a b c" {
		t.Fatalf("collapseSpace = %q", got)
	}
	if got := collapseSpace("   "); got != "" {
		t.Fatalf("collapseSpace whitespace = %q", got)
	}
}
