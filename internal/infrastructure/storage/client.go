package storage

// todo
// import (
// 	"context"
// 	"fmt"
// 	"sync"
// 	"time"

// 	"github.com/minio/minio-go/v7"
// 	"github.com/minio/minio-go/v7/pkg/credentials"
// )

// type Config struct {
// 	Endpoint        string
// 	AccessKeyID     string
// 	SecretAccessKey string
// 	UseSSL          bool
// 	Region          string
// 	Bucket          string
// 	MaxConcurrency  int
// }

// type StorageClient interface {
// 	GetMinIOClient() *minio.Client
// 	GetConfig() Config
// }

// type MinIODatabase struct {
// 	client *minio.Client
// 	cfg    Config
// }

// var (
// 	minioOnce     sync.Once
// 	minioInstance *MinIODatabase
// 	minioErr      error
// )

// func NewMinIOClient(ctx context.Context, cfg Config) (StorageClient, error) {
// 	minioOnce.Do(func() {
// 		client, err := minio.New(cfg.Endpoint, &minio.Options{
// 			Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
// 			Secure: cfg.UseSSL,
// 			Region: cfg.Region,
// 		})
// 		if err != nil {
// 			minioErr = fmt.Errorf("failed to init minio client: %w", err)
// 			return
// 		}

// 		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
// 		defer cancel()

// 		_, err = client.BucketExists(pingCtx, cfg.Bucket)
// 		if err != nil {
// 			minioErr = fmt.Errorf("failed to connect to MinIO: %w", err)
// 			return
// 		}

// 		minioInstance = &MinIODatabase{
// 			client: client,
// 			cfg:    cfg,
// 		}
// 	})

// 	if minioErr != nil {
// 		return nil, minioErr
// 	}

// 	return minioInstance, nil
// }

// func (m *MinIODatabase) GetMinIOClient() *minio.Client {
// 	return m.client
// }

// func (m *MinIODatabase) GetConfig() Config {
// 	return m.cfg
// }
