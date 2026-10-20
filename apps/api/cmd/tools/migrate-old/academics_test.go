package main

import "testing"

// TestValidClassGradeGrade v2 hanya menerima 10/11/12 (CHECK constraint).
// Baris lain harus DILEWATI dan dilaporkan, bukan dipaksa masuk — kalau
// dipaksa, satu kelas invalid menggagalkan seluruh migrasi.
func TestValidClassGrade(t *testing.T) {
	for _, g := range []int{10, 11, 12} {
		if !validClassGrade(g) {
			t.Fatalf("grade %d harus valid", g)
		}
	}
	for _, g := range []int{0, 9, 13, -1, 99} {
		if validClassGrade(g) {
			t.Fatalf("grade %d harus tidak valid", g)
		}
	}
}

// TestValidClassGroupClass group v2 harus satu huruf kapital.
func TestValidClassGroup(t *testing.T) {
	for _, g := range []string{"A", "B", "Z"} {
		if !validClassGroup(g) {
			t.Fatalf("group %q harus valid", g)
		}
	}
	for _, g := range []string{"", "a", "AB", "1", "A1", " "} {
		if validClassGroup(g) {
			t.Fatalf("group %q harus tidak valid", g)
		}
	}
}

// TestSlugifyMajor v2 punya majors.code NOT NULL. Kalau satu jurusan v1 punya
// kode kosong, slug dari nama dipakai supaya majors itu tetap tersimpan.
func TestSlugifyMajor(t *testing.T) {
	cases := map[string]string{
		"Animasi":                       "animasi",
		"Teknik Grafika":                "teknik-grafika",
		"Teknik Komputer dan Jaringan":  "teknik-komputer-dan-jaringan",
		"  Rekaayasa Perangkat Lunak  ": "rekaayasa-perangkat-lunak",
		"":                              "",
		"!!!":                           "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Fatalf("slugify(%q) = %q, mau %q", in, got, want)
		}
	}
}
