package http

import (
	"strings"
	"testing"
)

// TestPasswordStrong menguji kebijakan password (PRD §6.2): minimal 10
// karakter dan zxcvbn score >= 3.
//
// Aturan lama ("harus ada huruf besar, kecil, dan angka") terlalu lemah:
// "Password1" dan "password123" lolos padahal paling mudah ditebak. Semua
// password di bawah harus DITOLAK.
func TestPasswordStrong(t *testing.T) {
	lemah := []struct {
		pw     string
		alasan string
	}{
		{"Password1", "klasik: huruf besar+kecil+angka"},
		{"password123", "klasik: huruf kecil+angka"},
		{"qwerty12345", "pola keyboard"},
		{"asdfghjkl12", "pola keyboard"},
		{"12345678901", "deret angka"},
		{"aaaaaaaaaaaa", "pengulangan karakter"},
		{"iloveyou1234", "frasa umum"},
	}
	for _, c := range lemah {
		if ok, _ := passwordStrong(c.pw); ok {
			t.Errorf("password %q (%s) seharusnya ditolak", c.pw, c.alasan)
		}
	}

	// Terlalu pendek.
	if ok, _ := passwordStrong("Ab1!xyz"); ok {
		t.Error("password < 10 karakter harus ditolak")
	}
	// Terlalu panjang.
	if ok, _ := passwordStrong(strings.Repeat("aB3", 50)); ok {
		t.Error("password > 128 karakter harus ditolak")
	}

	// Yang cukup kuat harus diterima.
	kuat := []string{
		"Mon_Kitten_9",
		"Smkn4Malang2026!",
		"kucinglucu123",
		"correcthorsebatterystaple",
	}
	for _, pw := range kuat {
		if ok, msg := passwordStrong(pw); !ok {
			t.Errorf("password %q seharusnya diterima, tapi ditolak: %s", pw, msg)
		}
	}
}

// TestPasswordStrongRejectsUserInputs memastikan password yang merupakan
// variasi dari username/email user sendiri ditolak. Pola "budi123" untuk
// user "budi" adalah salah satu yang paling sering dipakai.
//
// Catatan: zxcvbn menilai KECOCOKAN dengan userInput, bukan substring.
// Jadi "budi123" (turunan langsung) ditolak, sedangkan
// "budi2024sayafriend" tetap lolos — dan memang itu password yang kuat.
func TestPasswordStrongRejectsUserInputs(t *testing.T) {
	// Tanpa userInput, password yang kuat ini lolos.
	if ok, _ := passwordStrong("Mon_Kitten_9"); !ok {
		t.Fatal("prekondisi: password harusnya lolos tanpa userInput")
	}
	// Dengan username sebagai userInput, variannya harus ditolak.
	for _, pw := range []string{"budi", "budi123", "Budi!", "BUDI2026"} {
		if ok, _ := passwordStrong(pw, "budi"); ok {
			t.Errorf("password %q (turunan dari username) harus ditolak", pw)
		}
	}
	if ok, _ := passwordStrong("budi.santoso@smkn4.sch.id", "budi.santoso@smkn4.sch.id"); ok {
		t.Error("password yang sama dengan email sendiri harus ditolak")
	}
}

// TestRandomPasswordAlwaysStrong memastikan password acak yang dibuat untuk
// import siswa SELALU lolos kebijakan — kalau tidak, alur import akan
// menghasilkan password yang tidak bisa dipakai.
func TestRandomPasswordAlwaysStrong(t *testing.T) {
	for i := 0; i < 200; i++ {
		pw := randomPassword(12)
		if ok, msg := passwordStrong(pw); !ok {
			t.Fatalf("password acak %q ditolak: %s", pw, msg)
		}
	}
}
