// Package dbmigrations membungkus file migrasi goose agar bisa di-embed.
// File SQL tinggal di sini (di-track git) supaya build bersih tidak butuh
// langkah salin manual.
package dbmigrations

import "embed"

//go:embed *.sql
var FS embed.FS
