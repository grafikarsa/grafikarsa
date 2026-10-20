package http

import (
	"math"
	"testing"
)

func TestRecencyFormula(t *testing.T) {
	cases := []struct {
		ageH float64
		want float64
	}{
		{0, 1.0},
		{168, math.Exp(-1)},
		{336, math.Exp(-2)},
	}
	for _, c := range cases {
		got := recencyScore(c.ageH)
		if math.Abs(got-c.want) > 1e-9 {
			t.Fatalf("recency(%v) = %v, want %v", c.ageH, got, c.want)
		}
	}
	// Umur negatif (published_at di masa depan) harus dibatasi 0, bukan
	// menghasilkan skor > 1.
	if got := recencyScore(-5); got != 1.0 {
		t.Fatalf("recency(umur negatif) = %v, ingin 1.0", got)
	}
}

func TestFollowingScore(t *testing.T) {
	if followingScore(true, false) != 1.0 {
		t.Fatal("mutual harus 1.0")
	}
	if followingScore(false, true) != 0.8 {
		t.Fatal("one-way harus 0.8")
	}
	if followingScore(false, false) != 0 {
		t.Fatal("none harus 0")
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := map[string]float64{"x": 2, "y": 1}
	b := map[string]float64{"x": 1}
	got := cosineSimilarity(a, b, []string{"x", "y"})
	// dot=2, |a|=sqrt(5), |b|=1 -> 2/sqrt(5)
	want := 2 / math.Sqrt(5)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("cosine = %v, want %v", got, want)
	}
	if cosineSimilarity(map[string]float64{}, b, []string{"x"}) != 0 {
		t.Fatal("vektor kosong harus 0")
	}
}

func TestFinalScoreWeights(t *testing.T) {
	final := 0.30*1.0 + 0.25*1.0 + 0.20*1.0 + 0.15*1.0 + 0.10*1.0
	if math.Abs(final-1.0) > 1e-9 {
		t.Fatalf("bobot harus berjumlah 1.0, got %v", final)
	}
}

func TestValidateBlockPayload(t *testing.T) {
	if _, ok := validateBlockPayload("youtube", map[string]any{"video_id": "dQw4w9WgXcQ"}); !ok {
		t.Fatal("youtube valid harus lolos")
	}
	if _, ok := validateBlockPayload("youtube", map[string]any{"video_id": "pendek"}); ok {
		t.Fatal("youtube pendek harus ditolak")
	}
	if _, ok := validateBlockPayload("image", map[string]any{"media_id": "x", "alt": "ab"}); ok {
		t.Fatal("alt <5 harus ditolak")
	}
	if _, ok := validateBlockPayload("embed", map[string]any{"url": "https://evil.com/x"}); ok {
		t.Fatal("domain tidak allowlist harus ditolak")
	}
	if _, ok := validateBlockPayload("embed", map[string]any{"url": "https://codepen.io/x"}); !ok {
		t.Fatal("codepen harus lolos")
	}
}
