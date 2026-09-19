package storage

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"io"
// 	"sync"
// 	"time"

// 	repository_contract "messenger-backend/internal/domain/repository"
// )

// type MockStorageRepository struct {
// 	mu    sync.RWMutex
// 	store map[string][]byte
// }

// func NewMockStorageRepository() repository_contract.StorageRepository {
// 	return &MockStorageRepository{store: make(map[string][]byte)}
// }

// func (m *MockStorageRepository) UploadFile(_ context.Context, _ string, objectKey string, reader io.Reader, _ int64, _ string) (string, error) {
// 	data, err := io.ReadAll(reader)
// 	if err != nil {
// 		return "", fmt.Errorf("mock storage: read upload body for %q: %w", objectKey, err)
// 	}

// 	m.mu.Lock()
// 	m.store[objectKey] = data
// 	m.mu.Unlock()

// 	return objectKey, nil
// }

// func (m *MockStorageRepository) DownloadFile(_ context.Context, _ string, objectKey string) (io.ReadCloser, error) {
// 	m.mu.RLock()
// 	data, ok := m.store[objectKey]
// 	m.mu.RUnlock()
// 	if !ok {
// 		return nil, fmt.Errorf("mock storage: key %q not found", objectKey)
// 	}
// 	return io.NopCloser(bytes.NewReader(data)), nil
// }

// func (m *MockStorageRepository) GetPresignedURL(_ context.Context, _ string, objectKey string, expiry time.Duration) (string, error) {
// 	m.mu.RLock()
// 	_, ok := m.store[objectKey]
// 	m.mu.RUnlock()
// 	if !ok {
// 		return "", fmt.Errorf("mock storage: key %q not found", objectKey)
// 	}
// 	return fmt.Sprintf("mock://storage/%s?expires_in=%s", objectKey, expiry), nil
// }

// func (m *MockStorageRepository) DeleteFile(_ context.Context, _ string, objectKey string) error {
// 	m.mu.Lock()
// 	delete(m.store, objectKey)
// 	m.mu.Unlock()
// 	return nil
// }

// func (m *MockStorageRepository) UploadFilesBatch(ctx context.Context, bucketName string, files []repository_contract.FileUploadPayload) (map[string]string, error) {
// 	results := make(map[string]string, len(files))
// 	var firstErr error

// 	for _, f := range files {
// 		key, err := m.UploadFile(ctx, bucketName, f.ObjectKey, f.Reader, f.Size, f.ContentType)
// 		if err != nil {
// 			if firstErr == nil {
// 				firstErr = err
// 			}
// 			continue
// 		}
// 		results[f.ObjectKey] = key
// 	}

// 	return results, firstErr
// }

// func (m *MockStorageRepository) DownloadFilesBatch(ctx context.Context, bucketName string, objectKeys []string) (map[string]io.ReadCloser, error) {
// 	results := make(map[string]io.ReadCloser, len(objectKeys))
// 	var firstErr error

// 	for _, key := range objectKeys {
// 		rc, err := m.DownloadFile(ctx, bucketName, key)
// 		if err != nil {
// 			if firstErr == nil {
// 				firstErr = err
// 			}
// 			continue
// 		}
// 		results[key] = rc
// 	}

// 	return results, firstErr
// }

// func (m *MockStorageRepository) GetPresignedURLsBatch(ctx context.Context, bucketName string, objectKeys []string, expiry time.Duration) (map[string]string, error) {
// 	results := make(map[string]string, len(objectKeys))
// 	var firstErr error

// 	for _, key := range objectKeys {
// 		url, err := m.GetPresignedURL(ctx, bucketName, key, expiry)
// 		if err != nil {
// 			if firstErr == nil {
// 				firstErr = err
// 			}
// 			continue
// 		}
// 		results[key] = url
// 	}

// 	return results, firstErr
// }

// func (m *MockStorageRepository) DeleteFilesBatch(_ context.Context, _ string, objectKeys []string) error {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
// 	for _, key := range objectKeys {
// 		delete(m.store, key)
// 	}
// 	return nil
// }
