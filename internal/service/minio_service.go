package service

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioService struct {
	client   *minio.Client
	bucket   string
	endpoint string
	useSSL   bool
}

func NewMinioService(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioService, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("minio endpoint is required")
	}
	if bucket == "" {
		return nil, fmt.Errorf("minio bucket is required")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinioService{
		client:   client,
		bucket:   bucket,
		endpoint: endpoint,
		useSSL:   useSSL,
	}, nil
}

func (s *MinioService) Upload(ctx context.Context, filename, size string, data []byte) (string, error) {
	log.Printf("[service] MinIO Upload started: filename=%s size=%s", filename, size)
	key := fmt.Sprintf("thumbnails/%s/%s-%s.jpg", time.Now().Format("20060102"), filename, size)

	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" {
		contentType = "image/jpeg"
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		log.Printf("[service] MinIO Upload failure: filename=%s size=%s error=%v", filename, size, err)
		return "", fmt.Errorf("upload thumbnail to minio: %w", err)
	}

	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}

	finalURL := fmt.Sprintf("%s://%s/%s/%s", scheme, s.endpoint, s.bucket, key)
	log.Printf("[service] MinIO Upload completed: url=%s", finalURL)
	return finalURL, nil
}

func (s *MinioService) DeleteByURL(ctx context.Context, objectURL string) error {
	log.Printf("[service] MinIO DeleteByURL started: url=%s", objectURL)
	if s == nil || s.client == nil {
		return fmt.Errorf("minio service is not initialized")
	}

	if objectURL == "" {
		log.Printf("[service] MinIO DeleteByURL skipped: empty url")
		return nil
	}

	u, err := url.Parse(objectURL)
	if err != nil {
		log.Printf("[service] MinIO DeleteByURL url parse failure: url=%s error=%v", objectURL, err)
		return fmt.Errorf("parse object url: %w", err)
	}

	key := strings.TrimPrefix(u.Path, "/")
	bucketPrefix := s.bucket + "/"
	if strings.HasPrefix(key, bucketPrefix) {
		key = strings.TrimPrefix(key, bucketPrefix)
	}
	if key == "" || key == "." {
		log.Printf("[service] MinIO DeleteByURL invalid key failure: url=%s key=%s", objectURL, key)
		return fmt.Errorf("invalid object key from url: %s", objectURL)
	}

	err = s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		log.Printf("[service] MinIO DeleteByURL deletion failure: url=%s key=%s error=%v", objectURL, key, err)
		return fmt.Errorf("delete object from minio: %w", err)
	}

	log.Printf("[service] MinIO DeleteByURL completed: url=%s", objectURL)
	return nil
}
