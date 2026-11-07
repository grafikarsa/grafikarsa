// Package migrations embeds the goose SQL migrations.
package migrations

import "embed"

// FS contains every goose SQL migration file.
//
//go:embed *.sql
var FS embed.FS
