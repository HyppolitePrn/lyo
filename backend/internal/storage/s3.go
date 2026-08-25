// Package storage provides a thin wrapper around S3 for track audio files.
package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/hyppoliteprn/lyo/pkg/config"
)

// presignExpiry is how long a presigned upload URL remains valid.
const presignExpiry = 15 * time.Minute

// Storage defines the S3 operations consumed by the track feature.
type Storage interface {
	// PresignUpload returns a presigned PUT URL for the given object key.
	PresignUpload(ctx context.Context, key string) (string, error)
	// Delete removes the object at key. Deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
	// PublicURL returns the public URL a track's audio_url should store for key.
	PublicURL(key string) string
	// KeyFromURL extracts the object key from a URL previously returned by PublicURL.
	KeyFromURL(url string) (string, bool)
}

type s3Storage struct {
	client        *s3.Client        // internal endpoint — server-side ops (Delete)
	presign       *s3.PresignClient // public endpoint — consumed by the mobile client
	bucket        string
	region        string
	publicBaseURL string // "" means real AWS virtual-hosted-style fallback
}

// New returns an S3-compatible Storage (real AWS or a self-hosted server
// such as MinIO) using static credentials from cfg.
//
// Two clients are built because a presigned URL is SigV4-signed over its
// Host header: a client pointed at a Docker-internal hostname would produce
// presigned URLs the mobile client can't reach, and rewriting the host
// after signing breaks the signature. So PresignUpload signs against
// cfg.PublicEndpoint while Delete talks to cfg.Endpoint directly.
func New(cfg config.S3Config) (Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	internalClient := s3.NewFromConfig(awsCfg, endpointOptions(cfg.Endpoint))

	publicEndpoint := cfg.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = cfg.Endpoint
	}
	publicClient := internalClient
	if publicEndpoint != cfg.Endpoint {
		publicClient = s3.NewFromConfig(awsCfg, endpointOptions(publicEndpoint))
	}

	return &s3Storage{
		client:        internalClient,
		presign:       s3.NewPresignClient(publicClient),
		bucket:        cfg.BucketName,
		region:        cfg.Region,
		publicBaseURL: publicEndpoint,
	}, nil
}

// endpointOptions configures a custom base endpoint and path-style
// addressing, required by MinIO and most self-hosted S3-compatible servers
// (they lack wildcard DNS for virtual-hosted buckets). An empty endpoint
// leaves the SDK's default (real AWS) resolution intact.
func endpointOptions(endpoint string) func(*s3.Options) {
	return func(o *s3.Options) {
		if endpoint == "" {
			return
		}
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	}
}

func (s *s3Storage) PresignUpload(ctx context.Context, key string) (string, error) {
	req, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(presignExpiry))
	if err != nil {
		return "", fmt.Errorf("presign upload: %w", err)
	}
	return req.URL, nil
}

func (s *s3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *s3Storage) PublicURL(key string) string {
	if s.publicBaseURL == "" {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
	}
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.publicBaseURL, "/"), s.bucket, key)
}

func (s *s3Storage) KeyFromURL(url string) (string, bool) {
	prefix := s.PublicURL("")
	if !strings.HasPrefix(url, prefix) {
		return "", false
	}
	return strings.TrimPrefix(url, prefix), true
}
