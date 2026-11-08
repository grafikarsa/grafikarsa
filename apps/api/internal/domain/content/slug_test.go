package content

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"indonesian title", "Poster HUT ke-80 SMKN 4 Malang", 120, "poster-hut-ke-80-smkn-4-malang"},
		{"diacritics and ampersand", "Désain & Ilustrasi", 120, "desain-dan-ilustrasi"},
		{"punctuation and spaces", "  --Hello!! World--  ", 120, "hello-world"},
		{"already a slug", "web-app-2026", 120, "web-app-2026"},
		{"empty", "   ", 120, ""},
		{"cut without trailing dash", "poster grafis sekolah", 10, "poster-gra"},
		{"cut exactly after dash", "abc def", 4, "abc"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Slugify(test.input, test.maxLen); got != test.want {
				t.Fatalf("Slugify(%q, %d) = %q, mau %q", test.input, test.maxLen, got, test.want)
			}
		})
	}
}
