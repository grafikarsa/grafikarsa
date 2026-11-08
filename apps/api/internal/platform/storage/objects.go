package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// ErrNotFound reports that an object does not exist in the bucket.
var ErrNotFound = errors.New("storage: object not found")

// ObjectInfo describes one stored object without downloading it.
type ObjectInfo struct {
	Size        int64
	ContentType string
	ETag        string
}

// PutObject writes size bytes from body to bucket/key. body must yield exactly
// size bytes; the length is sent explicitly so non-seekable streams work.
func (c *Client) PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	if _, err := c.S3.PutObject(ctx, input); err != nil {
		return fmt.Errorf("s3: put object %s/%s: %w", bucket, key, err)
	}
	return nil
}

// GetObject opens bucket/key for reading together with its metadata.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, ObjectInfo, error) {
	out, err := c.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ObjectInfo{}, ErrNotFound
		}
		return nil, ObjectInfo{}, fmt.Errorf("s3: get object %s/%s: %w", bucket, key, err)
	}

	return out.Body, ObjectInfo{
		Size:        aws.ToInt64(out.ContentLength),
		ContentType: aws.ToString(out.ContentType),
		ETag:        aws.ToString(out.ETag),
	}, nil
}

// DeleteObject removes bucket/key. Removing a missing object is not an error.
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	if _, err := c.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("s3: delete object %s/%s: %w", bucket, key, err)
	}
	return nil
}

// CopyObject copies one object between buckets or keys on the same endpoint.
func (c *Client) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	source := url.PathEscape(srcBucket + "/" + srcKey)
	if _, err := c.S3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(source),
	}); err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("s3: copy object %s/%s to %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}
	return nil
}

// StatObject reads the metadata of bucket/key without downloading it.
func (c *Client) StatObject(ctx context.Context, bucket, key string) (ObjectInfo, error) {
	out, err := c.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return ObjectInfo{}, ErrNotFound
		}
		return ObjectInfo{}, fmt.Errorf("s3: stat object %s/%s: %w", bucket, key, err)
	}

	return ObjectInfo{
		Size:        aws.ToInt64(out.ContentLength),
		ContentType: aws.ToString(out.ContentType),
		ETag:        aws.ToString(out.ETag),
	}, nil
}

// isNotFound reports whether an S3 error means the object or bucket is absent.
func isNotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound", "NoSuchBucket", "404":
			return true
		}
	}
	return strings.Contains(err.Error(), "StatusCode: 404")
}
