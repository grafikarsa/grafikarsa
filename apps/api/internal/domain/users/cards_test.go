package users

import (
	"bytes"
	"testing"
	"time"
)

func sampleAccountCard() AccountCard {
	return AccountCard{
		DisplayName:       "Budi Santoso",
		NISN:              "1000000001",
		InitialPassword:   "k7m2-pq9x-4hst",
		ClassName:         "X RPL A",
		MajorName:         "Rekayasa Perangkat Lunak",
		AcademicYearLabel: "2026/2027",
		ExpiresAt:         time.Date(2026, time.December, 8, 15, 4, 5, 0, time.UTC),
	}
}

func TestCardLines_ContainsCardData(t *testing.T) {
	t.Parallel()

	lines := cardLines(sampleAccountCard())
	if len(lines) != 6 {
		t.Fatalf("cardLines returned %d lines, want 6", len(lines))
	}

	want := []string{
		"Grafikarsa - SMKN 4 Malang",
		"Budi Santoso",
		"X RPL A - Rekayasa Perangkat Lunak (2026/2027)",
		"NISN: 1000000001",
		"Sandi awal: k7m2-pq9x-4hst",
		"Berlaku sampai: 8 Desember 2026",
	}
	for i, expected := range want {
		if lines[i] != expected {
			t.Errorf("line %d = %q, want %q", i, lines[i], expected)
		}
	}
}

func TestFormatCardDate_IndonesianLongForm(t *testing.T) {
	t.Parallel()

	cases := []struct {
		date time.Time
		want string
	}{
		{time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), "1 Januari 2026"},
		{time.Date(2026, time.August, 17, 23, 59, 0, 0, time.UTC), "17 Agustus 2026"},
		{time.Date(2027, time.December, 31, 0, 0, 0, 0, time.UTC), "31 Desember 2027"},
	}
	for _, tc := range cases {
		if got := formatCardDate(tc.date); got != tc.want {
			t.Errorf("formatCardDate(%s) = %q, want %q", tc.date, got, tc.want)
		}
	}
}

func TestBuildAccountCardsPDF_MultiplePages(t *testing.T) {
	t.Parallel()

	cards := make([]AccountCard, 12)
	for i := range cards {
		cards[i] = sampleAccountCard()
	}

	pdf, err := buildAccountCardsPDF(cards)
	if err != nil {
		t.Fatalf("buildAccountCardsPDF: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("output does not start with %PDF-")
	}
	// Twelve cards need two A4 pages (plus the page tree object).
	if got := bytes.Count(pdf, []byte("/Type /Page\n")); got != 2 {
		t.Fatalf("page objects = %d, want 2", got)
	}
}
