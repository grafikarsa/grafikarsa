package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
)

const (
	rustfsImage        = "rustfs/rustfs:1.0.4"
	rustfsAccessKey    = "gk-test-access-key"
	rustfsSecretKey    = "gk-test-secret-key-0123456789"
	rustfsSetupTimeout = 2 * time.Minute
)

var (
	s3Once      sync.Once
	s3Endpoint  string
	s3SetupErr  error
	s3DockerErr error
)

// NewS3 starts one shared RustFS container per test binary and returns a
// storage client whose private and public buckets are unique to this test.
// The test is skipped (t.Skip) when Docker is not available.
func NewS3(t testing.TB) *storage.Client {
	t.Helper()

	if tt, ok := t.(*testing.T); ok {
		testcontainers.SkipIfProviderIsNotHealthy(tt)
	}

	s3Once.Do(provisionS3)

	if s3DockerErr != nil {
		t.Skipf("docker tidak tersedia: %v", s3DockerErr)
	}
	if s3SetupErr != nil {
		t.Fatalf("siapkan storage uji: %v", s3SetupErr)
	}

	suffix := randomBucketSuffix(t)
	client, err := storage.New(
		context.Background(),
		s3Endpoint,
		"us-east-1",
		rustfsAccessKey,
		rustfsSecretKey,
		"t-"+suffix+"-private",
		"t-"+suffix+"-public",
	)
	if err != nil {
		t.Fatalf("buat klien storage uji: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.EnsureBuckets(ctx); err != nil {
		t.Fatalf("siapkan bucket uji: %v", err)
	}

	return client
}

// provisionS3 starts the shared RustFS container. It runs once per test
// binary; Ryuk removes the container when the test process exits.
func provisionS3() {
	ctx, cancel := context.WithTimeout(context.Background(), rustfsSetupTimeout)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        rustfsImage,
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"RUSTFS_ACCESS_KEY": rustfsAccessKey,
				"RUSTFS_SECRET_KEY": rustfsSecretKey,
				"RUSTFS_VOLUMES":    "/data",
			},
			WaitingFor: wait.ForHTTP("/health").
				WithPort("9000/tcp").
				WithStartupTimeout(rustfsSetupTimeout),
		},
		Started: true,
	})
	if err != nil {
		s3DockerErr = err
		s3SetupErr = fmt.Errorf("jalankan container rustfs: %w", err)
		return
	}

	endpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		s3SetupErr = fmt.Errorf("ambil endpoint rustfs: %w", err)
		return
	}
	s3Endpoint = endpoint
}

// randomBucketSuffix returns a random 12-character hex string.
func randomBucketSuffix(t testing.TB) string {
	t.Helper()

	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("buat akhiran bucket acak: %v", err)
	}
	return hex.EncodeToString(buf)
}
