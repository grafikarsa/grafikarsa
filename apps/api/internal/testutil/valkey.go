package testutil

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const valkeyImage = "valkey/valkey:9.1-alpine"

// NewValkey starts a Valkey container and returns a connected client. The
// container and client are cleaned up when the test finishes. The test is
// skipped (t.Skip) when Docker is not available.
func NewValkey(t testing.TB) *redis.Client {
	t.Helper()

	if tt, ok := t.(*testing.T); ok {
		testcontainers.SkipIfProviderIsNotHealthy(tt)
	}

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
		t.Fatalf("jalankan container valkey: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("ambil endpoint valkey: %v", err)
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
