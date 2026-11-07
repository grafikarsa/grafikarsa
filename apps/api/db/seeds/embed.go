// Package seeds embeds the idempotent seed SQL files.
package seeds

import "embed"

// FS contains every seed SQL file.
//
//go:embed *.sql
var FS embed.FS
