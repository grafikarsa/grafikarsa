package storage

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/grafikarsa/backend/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client        *minio.Client
	presignClient *minio.Client // Separate client for presigning with browser-accessible endpoint
	bucket        string
	publicURL     string
	// Path prefix for presigned URLs when MinIO sits behind a path-based
	// reverse proxy (e.g. Caddy `/storage/*` -> MinIO). See config.
	presignPathPrefix string
}

func NewMinIOClient(cfg *config.Config) (*MinIOClient, error) {
	minioCfg := cfg.MinIO

	// Main client for internal operations (uses Docker internal hostname)
	client, err := minio.New(minioCfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.AccessKey, minioCfg.SecretKey, ""),
		Secure: minioCfg.UseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Presign client for generating browser-accessible URLs
	// Uses public-facing endpoint so presigned URLs work from browsers
	// Note: minio.New() doesn't actually connect - it just stores config
	// PresignedPutObject generates signature locally without server connection
	presignEndpoint := minioCfg.PresignHost
	if presignEndpoint == "" {
		presignEndpoint = minioCfg.Endpoint
	}

	presignClient, err := minio.New(presignEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.AccessKey, minioCfg.SecretKey, ""),
		Secure: minioCfg.PresignUseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO presign client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, minioCfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, minioCfg.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Bucket %s created successfully", minioCfg.Bucket)
	}

	// Set bucket policy to public read
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"AWS": ["*"]
				},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, minioCfg.Bucket)

	if err := client.SetBucketPolicy(ctx, minioCfg.Bucket, policy); err != nil {
		log.Printf("Failed to set bucket policy: %v", err)
		// Don't fail startup for this, but log it
	} else {
		log.Printf("Bucket policy set to public read for %s", minioCfg.Bucket)
	}

	return &MinIOClient{
		client:            client,
		presignClient:     presignClient,
		bucket:            minioCfg.Bucket,
		publicURL:         minioCfg.PublicURL,
		presignPathPrefix: minioCfg.PresignPathPrefix,
	}, nil
}

// withPathPrefix inserts the reverse-proxy path prefix (e.g. "/storage") in
// front of a presigned URL path. The signature stays valid because the proxy
// strips the prefix before forwarding, so MinIO sees the exact path + query
// that was signed, with the same host.
func (m *MinIOClient) withPathPrefix(raw string) string {
	if m.presignPathPrefix == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	prefix := m.presignPathPrefix
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimSuffix(prefix, "/")
	if !strings.HasPrefix(u.Path, prefix+"/") {
		u.Path = prefix + u.Path
	}
	return u.String()
}

func (m *MinIOClient) GetPresignedPutURL(objectKey, contentType string, expiry time.Duration) (string, error) {
	// Use presignClient which has browser-accessible endpoint
	presignedURL, err := m.presignClient.PresignedPutObject(
		context.Background(),
		m.bucket,
		objectKey,
		expiry,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return m.withPathPrefix(presignedURL.String()), nil
}

func (m *MinIOClient) GetPresignedGetURL(objectKey string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	// NOTE: presignClient (public host), not internal client — the old code
	// returned internal-endpoint URLs unreachable from browsers.
	presignedURL, err := m.presignClient.PresignedGetObject(
		context.Background(),
		m.bucket,
		objectKey,
		expiry,
		reqParams,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return m.withPathPrefix(presignedURL.String()), nil
}

func (m *MinIOClient) ObjectExists(objectKey string) (bool, error) {
	_, err := m.client.StatObject(context.Background(), m.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *MinIOClient) DeleteObject(objectKey string) error {
	return m.client.RemoveObject(context.Background(), m.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (m *MinIOClient) GetPublicURL(objectKey string) string {
	return fmt.Sprintf("%s/%s", m.publicURL, objectKey)
}
