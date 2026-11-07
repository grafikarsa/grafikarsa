package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

func TestSettingsIntegration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewDB(t)

	s := store.NewStore(pool)
	const key = "test.sqlc_check"

	cleanup := func() {
		if _, err := pool.Exec(context.Background(), "DELETE FROM settings WHERE key = $1", key); err != nil {
			t.Fatalf("hapus setting uji: %v", err)
		}
	}
	cleanup()
	defer cleanup()

	updated, err := s.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   key,
		Value: json.RawMessage(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("UpsertSetting: %v", err)
	}
	if updated.Key != key {
		t.Fatalf("UpsertSetting key = %q, mau %q", updated.Key, key)
	}

	got, err := s.GetSetting(ctx, key)
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if ok := settingOK(t, got.Value); !ok {
		t.Fatalf("GetSetting value = %s, mau {\"ok\":true}", got.Value)
	}

	errRollback := errors.New("uji rollback")
	err = s.WithTx(ctx, func(q *store.Queries) error {
		if _, err := q.UpsertSetting(ctx, store.UpsertSettingParams{
			Key:   key,
			Value: json.RawMessage(`{"ok":false}`),
		}); err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("WithTx error = %v, mau %v", err, errRollback)
	}

	after, err := s.GetSetting(ctx, key)
	if err != nil {
		t.Fatalf("GetSetting setelah rollback: %v", err)
	}
	if ok := settingOK(t, after.Value); !ok {
		t.Fatalf("nilai berubah setelah rollback: %s, mau {\"ok\":true}", after.Value)
	}
}

func settingOK(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var v struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal value %s: %v", raw, err)
	}
	return v.OK
}
