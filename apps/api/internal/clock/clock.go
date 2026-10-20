// Package clock menyediakan sumber waktu yang bisa disisipkan.
//
// Kenapa perlu: kode yang memanggil time.Now() langsung tidak bisa diuji
// untuk kasus "waktu sudah lewat" — expiry token, cooldown rate limit,
// scheduled job, dan rotasi refresh token semuanya bergantung pada waktu.
// Test kemudian hanya bisa menunggu atau memanipulasi state, dan keduanya lambat
// atau rapuh.
//
// Dengan Clock, test bisa memakai jam palsu dan majukan waktunya secara
// deterministik.
//
// Pakai lewat App.Clock, bukan time.Now() langsung.
package clock

import "time"

// Clock adalah sumber waktu.
type Clock interface {
	// Now mengembalikan waktu saat ini.
	Now() time.Time
}

// System adalah jam sungguhan.
type System struct{}

// Now mengembalikan waktu sistem.
func (System) Now() time.Time { return time.Now() }

// Fixed adalah jam yang diam — selalu mengembalikan waktu yang sama.
// Dipakai test untuk kasus "waktu tidak bergeser".
type Fixed struct {
	T time.Time
}

// Now mengembalikan waktu tetap.
func (f Fixed) Now() time.Time { return f.T }

// Func adalah jam yang memanggil fungsi yang diberikan.
type Func func() time.Time

// Now memanggil fungsi yang diberikan.
func (f Func) Now() time.Time { return f() }

// Pastikan semua tipe di bawah memenuhi interface.
var (
	_ Clock = System{}
	_ Clock = Fixed{}
	_ Clock = Func(nil)
)
