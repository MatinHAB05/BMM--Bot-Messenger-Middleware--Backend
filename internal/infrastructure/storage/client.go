package storage

import (
	"context"
	"fmt"
	repository_contract "messenger-backend/internal/domain/repository"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds the MinIO connection settings, loaded from environment
// variables by bootstrap (see .env.example / INTEGRATION_NOTES.md for the
// exact var names this maps to).
type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	Region          string
	// PublicEndpoint is the host:port used ONLY when signing presigned
	// URLs (GetPresignedURL / GetPresignedURLsBatch) -- it must be an
	// address the final consumer of the URL (a browser outside the
	// Docker network, say) can actually reach, as opposed to Endpoint,
	// which is used for every other, server-to-server call and is
	// typically an internal Docker service name (e.g. "rustfs:9000")
	// that only resolves inside the compose network. If left empty,
	// Endpoint is reused for presigning too, preserving today's
	// behavior. See PublicUseSSL below for the matching scheme.
	PublicEndpoint string
	// PublicUseSSL is Secure for the presign client. Defaults to UseSSL
	// when PublicEndpoint is empty; must be set explicitly alongside a
	// non-empty PublicEndpoint (e.g. true, once it's served over https
	// via a reverse proxy).
	PublicUseSSL bool
	// Bucket is checked (and created if missing) once, at startup, by
	// NewMinIORepository. Individual StorageRepository methods still take
	// their own bucketName parameter -- pass this same value at call
	// sites unless you deliberately use more than one bucket.
	Bucket string
	// MaxConcurrency bounds how many goroutines a batch operation may run
	// at once (via errgroup.SetLimit), so a large batch can't exhaust
	// file descriptors or connections to MinIO. Defaults to 8 when <= 0.
	MaxConcurrency int
}

type minioRepository struct {
	client         *minio.Client
	presignClient  *minio.Client
	maxConcurrency int
}

// NewMinIORepository connects to MinIO and ensures cfg.Bucket exists,
// creating it if not. Intended to be called once, at startup (e.g. from
// bootstrap.Init) -- returns the domain interface, never the concrete
// type, so nothing outside this package can reach for minio-go-specific
// behavior.
func NewMinIORepository(cfg Config) (repository_contract.StorageRepository, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: init client: %w", err)
	}

	// presignClient only ever builds signed URLs -- it makes no network
	// calls of its own -- so it's cheap to construct even when
	// PublicEndpoint is unset and this just duplicates client's settings.
	publicEndpoint := cfg.PublicEndpoint
	publicUseSSL := cfg.PublicUseSSL
	if publicEndpoint == "" {
		publicEndpoint = cfg.Endpoint
		publicUseSSL = cfg.UseSSL
	}
	presignClient, err := minio.New(publicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: publicUseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: init presign client: %w", err)
	}

	setupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exists, err := client.BucketExists(setupCtx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio: check bucket %q: %w", cfg.Bucket, err)
	}
	if !exists {
		if err := client.MakeBucket(setupCtx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("minio: create bucket %q: %w", cfg.Bucket, err)
		}
	}

	concurrency := cfg.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 8
	}

	return &minioRepository{client: client, presignClient: presignClient, maxConcurrency: concurrency}, nil
}
