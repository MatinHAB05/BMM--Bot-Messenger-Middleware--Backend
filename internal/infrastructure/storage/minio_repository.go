// Package storage holds the real, MinIO-backed StorageRepository
// implementation. All github.com/minio/minio-go/v7 usage is confined to
// this file (and mock_storage_repository.go's in-memory stand-in) --
// nothing outside internal/infrastructure/storage ever imports minio-go
// or sees a minio.Client/minio.Object; every method here returns only
// io.Reader/io.ReadCloser, strings, and errors, per
// repository_contract.StorageRepository.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"golang.org/x/sync/errgroup"

	repository_contract "messenger-backend/internal/domain/repository"
)

// --- Single-file operations ---

func (r *minioRepository) UploadFile(ctx context.Context, bucketName, objectKey string, reader io.Reader, objectSize int64, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, bucketName, objectKey, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return "", fmt.Errorf("minio: upload %q: %w", objectKey, err)
	}
	return objectKey, nil
}

func (r *minioRepository) DownloadFile(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error) {
	obj, err := r.client.GetObject(ctx, bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio: download %q: %w", objectKey, err)
	}

	// GetObject itself never errors on a missing key -- minio-go defers
	// the actual request until the first Read/Stat -- so confirm the
	// object is really there now, rather than handing back a stream whose
	// first Read fails confusingly far from this call site.
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("minio: stat %q: %w", objectKey, err)
	}

	return obj, nil
}

func (r *minioRepository) GetPresignedURL(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error) {
	// Deliberately uses presignClient, not client: the resulting URL is
	// handed to an external caller (a browser outside the Docker
	// network), so it must be signed against the publicly-reachable
	// endpoint, not the internal one used for server-to-server calls.
	presigned, err := r.presignClient.PresignedGetObject(ctx, bucketName, objectKey, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("minio: presign %q: %w", objectKey, err)
	}
	return presigned.String(), nil
}

func (r *minioRepository) DeleteFile(ctx context.Context, bucketName, objectKey string) error {
	if err := r.client.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("minio: delete %q: %w", objectKey, err)
	}
	return nil
}

// --- Batch operations ---
//
// Every batch method below uses errgroup.SetLimit to bound concurrency,
// but deliberately does NOT let one item's failure cancel its siblings
// (each goroutine always returns nil to the group; the first real error
// is captured separately and returned alongside whatever DID succeed).
// A batch upload/download/presign is much more useful in production if
// one bad key doesn't waste 49 otherwise-successful ones -- callers that
// need strict all-or-nothing semantics should loop the single-item method
// themselves instead.

func (r *minioRepository) UploadFilesBatch(ctx context.Context, bucketName string, files []repository_contract.FileUploadPayload) (map[string]string, error) {
	results := make(map[string]string, len(files))
	var (
		mu       sync.Mutex
		firstErr error
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(r.maxConcurrency)

	for _, f := range files {
		f := f
		g.Go(func() error {
			key, err := r.UploadFile(gCtx, bucketName, f.ObjectKey, f.Reader, f.Size, f.ContentType)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return nil
			}
			results[f.ObjectKey] = key
			return nil
		})
	}
	_ = g.Wait()

	return results, firstErr
}

func (r *minioRepository) DownloadFilesBatch(ctx context.Context, bucketName string, objectKeys []string) (map[string]io.ReadCloser, error) {
	results := make(map[string]io.ReadCloser, len(objectKeys))
	var (
		mu       sync.Mutex
		firstErr error
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(r.maxConcurrency)

	for _, key := range objectKeys {
		key := key
		g.Go(func() error {
			rc, err := r.DownloadFile(gCtx, bucketName, key)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return nil
			}
			results[key] = rc
			return nil
		})
	}
	_ = g.Wait()

	return results, firstErr
}

func (r *minioRepository) GetPresignedURLsBatch(ctx context.Context, bucketName string, objectKeys []string, expiry time.Duration) (map[string]string, error) {
	results := make(map[string]string, len(objectKeys))
	var (
		mu       sync.Mutex
		firstErr error
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(r.maxConcurrency)

	for _, key := range objectKeys {
		key := key
		g.Go(func() error {
			presignedURL, err := r.GetPresignedURL(gCtx, bucketName, key, expiry)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return nil
			}
			results[key] = presignedURL
			return nil
		})
	}
	_ = g.Wait()

	return results, firstErr
}

// DeleteFilesBatch uses MinIO's native bulk-delete API (RemoveObjects) --
// a single request for the whole batch, server-side, rather than one
// DeleteFile call per key.
func (r *minioRepository) DeleteFilesBatch(ctx context.Context, bucketName string, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	objectsCh := make(chan minio.ObjectInfo, len(objectKeys))
	go func() {
		defer close(objectsCh)
		for _, key := range objectKeys {
			select {
			case <-ctx.Done():
				return
			case objectsCh <- minio.ObjectInfo{Key: key}:
			}
		}
	}()

	var failures []string
	for removeErr := range r.client.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{}) {
		if removeErr.Err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", removeErr.ObjectName, removeErr.Err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("minio: batch delete had %d failure(s): %s", len(failures), strings.Join(failures, "; "))
	}
	return nil
}
