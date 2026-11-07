// Package settings reads application settings stored in the settings table.
package settings

import (
	"context"
	"encoding/json"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// Int returns the integer value of setting key. A missing row, a value that is
// not a JSON number, or any other read error yields def.
func Int(ctx context.Context, q *store.Queries, key string, def int) int {
	row, err := q.GetSetting(ctx, key)
	if err != nil {
		return def
	}

	var value int
	if err := json.Unmarshal(row.Value, &value); err != nil {
		return def
	}
	return value
}
