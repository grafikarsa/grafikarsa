// Package storage manages the S3-compatible object storage client.
package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps the S3 client together with the bucket names used by the API.
type Client struct {
	S3            *s3.Client
	PrivateBucket string
	PublicBucket  string
}

// New builds an S3 client for a path-style, S3-compatible endpoint.
func New(ctx context.Context, endpoint, region, accessKey, secretKey, privateBucket, publicBucket string) (*Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &Client{
		S3:            client,
		PrivateBucket: privateBucket,
		PublicBucket:  publicBucket,
	}, nil
}

// Ping verifies that the private bucket is reachable.
func (c *Client) Ping(ctx context.Context) error {
	if _, err := c.S3.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.PrivateBucket),
	}); err != nil {
		return fmt.Errorf("s3: head bucket %s: %w", c.PrivateBucket, err)
	}

	return nil
}

// EnsureBuckets creates the private and public buckets when they are absent.
func (c *Client) EnsureBuckets(ctx context.Context) error {
	for _, bucket := range []string{c.PrivateBucket, c.PublicBucket} {
		if err := c.ensureBucket(ctx, bucket); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) ensureBucket(ctx context.Context, bucket string) error {
	if _, err := c.S3.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	}); err == nil {
		return nil
	}

	if _, err := c.S3.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}); err == nil {
		return nil
	} else if isBucketAlreadyOwned(err) {
		return nil
	} else {
		return fmt.Errorf("s3: create bucket %s: %w", bucket, err)
	}
}

func isBucketAlreadyOwned(err error) bool {
	message := err.Error()
	return strings.Contains(message, "BucketAlreadyOwnedByYou") ||
		strings.Contains(message, "BucketAlreadyExists")
}
