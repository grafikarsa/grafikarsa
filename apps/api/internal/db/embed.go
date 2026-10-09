package db

import "embed"

// MigrationsFS berisi file goose. Di-sync dari db/migrations saat build
// via Makefile (target migrate-up menyalin sebelum go run).
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
