package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const valkeyImage = "valkey/valkey:9.1-alpine"

func newTestClient(t *testing.T) *redis.Client {
	t.Helper()

	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        valkeyImage,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start valkey container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("valkey endpoint: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: endpoint})
	t.Cleanup(func() {
		_ = client.Close()
	})

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping valkey: %v", err)
	}

	return client
}

func TestAllowConsumesTokens(t *testing.T) {
	ctx := context.Background()
	limiter := New(newTestClient(t))
	rule := Rule{Limit: 3, Window: time.Minute}
	key := "consume:" + t.Name()

	for i, wantRemaining := range []int{2, 1, 0} {
		res, err := limiter.Allow(ctx, key, rule)
		if err != nil {
			t.Fatalf("Allow call %d: %v", i+1, err)
		}
		if !res.Allowed {
			t.Fatalf("Allow call %d: not allowed, want allowed", i+1)
		}
		if res.Limit != rule.Limit {
			t.Errorf("Allow call %d: Limit = %d, want %d", i+1, res.Limit, rule.Limit)
		}
		if res.Remaining != wantRemaining {
			t.Errorf("Allow call %d: Remaining = %d, want %d", i+1, res.Remaining, wantRemaining)
		}
	}

	res, err := limiter.Allow(ctx, key, rule)
	if err != nil {
		t.Fatalf("Allow over limit: %v", err)
	}
	if res.Allowed {
		t.Error("Allow over limit: allowed = true, want false")
	}
	if res.RetryAfter <= 0 {
		t.Errorf("Allow over limit: RetryAfter = %v, want > 0", res.RetryAfter)
	}
	if res.Remaining != 0 {
		t.Errorf("Allow over limit: Remaining = %d, want 0", res.Remaining)
	}
}

func TestAllowKeysAreIndependent(t *testing.T) {
	ctx := context.Background()
	limiter := New(newTestClient(t))
	rule := Rule{Limit: 1, Window: time.Minute}
	keyA := "independent:a:" + t.Name()
	keyB := "independent:b:" + t.Name()

	first, err := limiter.Allow(ctx, keyA, rule)
	if err != nil {
		t.Fatalf("Allow key A first: %v", err)
	}
	if !first.Allowed {
		t.Fatal("Allow key A first: not allowed, want allowed")
	}

	second, err := limiter.Allow(ctx, keyA, rule)
	if err != nil {
		t.Fatalf("Allow key A second: %v", err)
	}
	if second.Allowed {
		t.Error("Allow key A second: allowed = true, want false")
	}

	other, err := limiter.Allow(ctx, keyB, rule)
	if err != nil {
		t.Fatalf("Allow key B: %v", err)
	}
	if !other.Allowed {
		t.Error("Allow key B: not allowed, want allowed")
	}
}

func TestResetClearsCounter(t *testing.T) {
	ctx := context.Background()
	limiter := New(newTestClient(t))
	rule := Rule{Limit: 1, Window: time.Minute}
	key := "reset:" + t.Name()

	first, err := limiter.Allow(ctx, key, rule)
	if err != nil {
		t.Fatalf("Allow first: %v", err)
	}
	if !first.Allowed {
		t.Fatal("Allow first: not allowed, want allowed")
	}

	second, err := limiter.Allow(ctx, key, rule)
	if err != nil {
		t.Fatalf("Allow second: %v", err)
	}
	if second.Allowed {
		t.Fatal("Allow second: allowed = true, want false")
	}

	if err := limiter.Reset(ctx, key); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	afterReset, err := limiter.Allow(ctx, key, rule)
	if err != nil {
		t.Fatalf("Allow after reset: %v", err)
	}
	if !afterReset.Allowed {
		t.Error("Allow after reset: not allowed, want allowed")
	}
	if afterReset.Remaining != 0 {
		t.Errorf("Allow after reset: Remaining = %d, want 0", afterReset.Remaining)
	}
}
