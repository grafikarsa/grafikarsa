package storage

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"grafikarsa/api/internal/config"
)

// Storage adalah satu-satunya pintu ke object storage.
// Implementasi: AWS SDK S3 standar — agnostik vendor (RustFS/MinIO/R2/S3).
//
//nolint:misspell // "Implementasi" adalah kata Indonesia, bukan salah ketik.
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

// EnsureBucket membuat bucket bila belum ada dan mengaktifkan versioning.
//
// Versioning penting untuk object storage: tanpa itu, satu file yang salah
// hapus (atau tertimpa) hilang permanen. Media di bucket ini bisa diganti
// user kapan saja (avatar, banner, thumbnail), jadi versioning adalah jaring
// pengaman terakhir, bukan sekadar nicety.
//
// Idempoten: aman dipanggil setiap boot.
func (s *Storage) EnsureBucket(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)}); err != nil {
		if _, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}); err != nil {
			return err
		}
	}
	_, err := s.client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(s.bucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	return err
}

// Size mengembalikan ukuran objek (dipakai untuk streaming + ServeContent
// agar tidak perlu membaca seluruh file ke memori).
func (s *Storage) Size(ctx context.Context, key string) (int64, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return 0, err
	}
	if out.ContentLength == nil {
		return 0, nil
	}
	return *out.ContentLength, nil
}

func (s *Storage) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	return err
}

func (s *Storage) Put(ctx context.Context, key, contentType string, body io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	return err
}

func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func Noop() error { return nil }
