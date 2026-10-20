package http

import (
	"testing"
	"time"

	"grafikarsa/api/internal/clock"
)

// TestAppClockBisaDisisipkan memastikan sumber waktu handler bisa diganti.
//
// PRD §13.2: `time.Now()` langsung di handler membuat kasus "waktu sudah
// lewat" mustahil diuji. Dengan jam yang bisa disisipkan, test bisa
// menentukan waktunya sendiri.
func TestAppClockBisaDisisipkan(t *testing.T) {
	kfixed := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)
	a := &App{Clock: clock.Fixed{T: kfixed}}

	if got := a.now(); !got.Equal(kfixed) {
		t.Errorf("a.now() = %v, ingin %v", got, kfixed)
	}
}

// TestAppClockDefaultSistem memastikan App tanpa Clock eksplisit tetap
// memakai jam sistem (bukan panic).
func TestAppClockDefaultSistem(t *testing.T) {
	a := &App{}
	sebelum := time.Now()
	hasil := a.now()
	if hasil.Before(sebelum.Add(-time.Minute)) {
		t.Errorf("a.now() = %v, tampaknya bukan jam sistem", hasil)
	}
}

// TestCSRFTokenTerikatPadaJamYangDisisipkan menguji bahwa token CSRF
// benar-benar mengikuti sumber waktu handler.
//
// Token CSRF = HMAC(secret, tanggal + user_id) dan sengaja menerima token
// H-1 agar sesi lintas tengah malam tidak ikut putus. Kalau jamnya tidak
// bisa disisipkan, aturan H-1 ini mustahil diuji.
func TestCSRFTokenTerikatPadaJamYangDisisipkan(t *testing.T) {
	const secret = "rahasia-uji-csrf"
	const userID = "11111111-1111-4111-8111-111111111111"

	hariIni := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	kemarin := hariIni.Add(-24 * time.Hour)

	tokenHariIni := newCSRFToken(secret, userID, hariIni)
	tokenKemarin := newCSRFToken(secret, userID, kemarin)

	// Token hari ini valid di hari ini.
	if !verifyCSRFToken(secret, userID, tokenHariIni, hariIni) {
		t.Error("token hari ini harus valid di hari ini")
	}
	// Token H-1 juga valid (aturan H-1).
	if !verifyCSRFToken(secret, userID, tokenKemarin, hariIni) {
		t.Error("token H-1 harus tetap valid (sesi lintas tengah malam)")
	}
	// Token H-2 sudah tidak berlaku.
	if verifyCSRFToken(secret, userID, tokenKemarin, hariIni.Add(-48*time.Hour)) {
		t.Error("token H-2 tidak boleh berlaku")
	}
	// Token milik user lain tidak berlaku.
	if verifyCSRFToken(secret, "user-lain", tokenHariIni, hariIni) {
		t.Error("token milik user lain tidak boleh berlaku")
	}
	// Token yang rusak tidak berlaku.
	if verifyCSRFToken(secret, userID, "sampah", hariIni) {
		t.Error("token rusak tidak boleh berlaku")
	}
}
