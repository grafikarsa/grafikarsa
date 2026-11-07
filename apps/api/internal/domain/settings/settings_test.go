package settings_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/settings"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

func TestInt(t *testing.T) {
	pool := testutil.NewSeededDB(t)
	ctx := context.Background()
	queries := store.NewStore(pool).Queries

	t.Run("seeded value", func(t *testing.T) {
		if got := settings.Int(ctx, queries, "auth.initial_password_ttl_days", 99); got != 30 {
			t.Fatalf("Int() = %d, mau 30", got)
		}
	})

	t.Run("missing key uses default", func(t *testing.T) {
		if got := settings.Int(ctx, queries, "test.missing_key", 17); got != 17 {
			t.Fatalf("Int() = %d, mau 17", got)
		}
	})

	t.Run("non-numeric value uses default", func(t *testing.T) {
		_, err := queries.UpsertSetting(ctx, store.UpsertSettingParams{
			Key:   "test.non_numeric",
			Value: json.RawMessage(`"tiga puluh"`),
		})
		if err != nil {
			t.Fatalf("UpsertSetting: %v", err)
		}

		if got := settings.Int(ctx, queries, "test.non_numeric", 42); got != 42 {
			t.Fatalf("Int() = %d, mau 42", got)
		}
	})
}
