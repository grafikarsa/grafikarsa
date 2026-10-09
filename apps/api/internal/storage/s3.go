package storage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"grafikarsa/api/internal/config"
)

// Storage adalah satu-satunya pintu ke object storage.
// Implementasi: AWS SDK S3 standar — agnostik vendor (RustFS/MinIO/R2/S3).
type Storage struct {
	client *s3.Client
	bucket string
}

func New(cfg *config.Config) *Storage {
	client := s3.New(s3.Options{
		Region:       cfg.S3.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.S3.AccessKey, cfg.S3.SecretKey, ""),
		BaseEndpoint: aws.String(cfg.S3.Endpoint),
		UsePathStyle: cfg.S3.UsePathStyle,
	})
	return &Storage{client: client, bucket: cfg.S3.Bucket}
}

func (s *Storage) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	return err
}
