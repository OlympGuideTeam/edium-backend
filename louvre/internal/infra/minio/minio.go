package minio

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"louvre/internal/config"
)

type Client struct {
	client *minio.Client
	bucket string
}

func NewClient(cfg config.MinIOConfig) (*Client, error) {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось создать MinIO клиент: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bucket := cfg.Bucket
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("не удалось проверить существование bucket: %w", err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("не удалось создать bucket %s: %w", bucket, err)
		}
		slog.Info("MinIO: создан bucket", "bucket", bucket)
	} else {
		slog.Info("MinIO: bucket существует", "bucket", bucket)
	}

	return &Client{
		client: minioClient,
		bucket: bucket,
	}, nil
}

func (c *Client) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := c.client.PutObject(ctx, c.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("не удалось загрузить в MinIO: %w", err)
	}
	return objectName, nil
}

func (c *Client) Download(ctx context.Context, objectName string) (*minio.Object, error) {
	object, err := c.client.GetObject(ctx, c.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("не удалось скачать из MinIO: %w", err)
	}
	return object, nil
}

func (c *Client) Delete(ctx context.Context, objectName string) error {
	err := c.client.RemoveObject(ctx, c.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("не удалось удалить из MinIO: %w", err)
	}
	return nil
}
