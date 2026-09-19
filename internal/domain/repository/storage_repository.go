package repository_contract

import (
	"context"
	"io"
	"time"
)

// FileUploadPayload is one item in a batch upload: the object key it
// should be stored under, its byte stream, and the metadata MinIO needs
// up front (size, content type). Reader is consumed exactly once by
// UploadFilesBatch/UploadFile.
type FileUploadPayload struct {
	ObjectKey   string
	Reader      io.Reader
	Size        int64
	ContentType string
}

// StorageRepository abstracts object storage (MinIO/S3) for attachment
// bytes and their thumbnails. This interface -- io.Reader/io.ReadCloser,
// string keys, string URLs -- is the ONLY thing the service and handler
// layers may depend on; no minio-go type (minio.Client, minio.Object, ...)
// may appear outside internal/infrastructure/storage.
//
// bucketName is an explicit parameter on every method rather than baked
// into the implementation, so a single StorageRepository can serve
// multiple buckets (e.g. separating attachments from thumbnails, or
// per-environment buckets) if that's ever needed; callers that only use
// one bucket just pass the same configured name every time.
type StorageRepository interface {
	UploadFile(ctx context.Context, bucketName string, objectKey string, reader io.Reader, objectSize int64, contentType string) (string, error)
	DownloadFile(ctx context.Context, bucketName string, objectKey string) (io.ReadCloser, error)
	GetPresignedURL(ctx context.Context, bucketName string, objectKey string, expiry time.Duration) (string, error)
	DeleteFile(ctx context.Context, bucketName string, objectKey string) error

	// UploadFilesBatch uploads every payload concurrently (bounded
	// concurrency -- see the MinIO implementation's doc comment) and
	// returns a map of ObjectKey -> the same key on success. A given
	// key's absence from the returned map (with a non-nil error) means
	// that one upload failed; callers that need per-file success/failure
	// detail should upload individually instead.
	UploadFilesBatch(ctx context.Context, bucketName string, files []FileUploadPayload) (map[string]string, error)
	// DownloadFilesBatch fetches every key concurrently. Callers MUST
	// Close() every returned io.ReadCloser, including when only some
	// keys in the batch succeeded (the ones that did are still open and
	// must be closed to avoid leaking connections).
	DownloadFilesBatch(ctx context.Context, bucketName string, objectKeys []string) (map[string]io.ReadCloser, error)
	GetPresignedURLsBatch(ctx context.Context, bucketName string, objectKeys []string, expiry time.Duration) (map[string]string, error)
	// DeleteFilesBatch uses MinIO's native bulk-delete API (a single
	// request for the whole batch) rather than one DeleteFile call per
	// key.
	DeleteFilesBatch(ctx context.Context, bucketName string, objectKeys []string) error
}
