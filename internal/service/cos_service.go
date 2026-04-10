package service

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
)

type COSService struct {
	client    *cos.Client
	bucketURL string
}

func NewCOSService(cfg *config.COSConfig) (*COSService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cos config is required")
	}
	if cfg.BucketURL == "" {
		return nil, fmt.Errorf("cos bucket_url is required")
	}

	secretID := os.Getenv("COS_SECRET_ID")
	if secretID == "" {
		secretID = cfg.SecretID
	}
	secretKey := os.Getenv("COS_SECRET_KEY")
	if secretKey == "" {
		secretKey = cfg.SecretKey
	}

	u, err := url.Parse(cfg.BucketURL)
	if err != nil {
		return nil, fmt.Errorf("parse cos bucket_url: %w", err)
	}

	b := &cos.BaseURL{BucketURL: u}
	httpClient := &http.Client{}
	if secretID != "" && secretKey != "" {
		httpClient.Transport = &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		}
	}
	client := cos.NewClient(b, httpClient)

	return &COSService{
		client:    client,
		bucketURL: strings.TrimRight(cfg.BucketURL, "/"),
	}, nil
}

func (s *COSService) Upload(ctx context.Context, filename, size string, data []byte) (string, error) {
	log.Printf("[service] COS Upload started: filename=%s size=%s", filename, size)
	key := fmt.Sprintf("thumbnails/%s-%s.jpg", filename, size)

	_, err := s.client.Object.Put(ctx, key, bytes.NewReader(data), &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: "image/jpeg",
		},
		ACLHeaderOptions: &cos.ACLHeaderOptions{
			XCosACL: "public-read",
		},
	})
	if err != nil {
		log.Printf("[service] COS Upload failure: filename=%s size=%s error=%v", filename, size, err)
		return "", fmt.Errorf("upload thumbnail to cos: %w", err)
	}

	// Ensure bucketURL has https:// protocol
	uploadURL := s.bucketURL
	if !strings.HasPrefix(uploadURL, "http://") && !strings.HasPrefix(uploadURL, "https://") {
		uploadURL = "https://" + uploadURL
	}

	finalURL := fmt.Sprintf("%s/%s", uploadURL, key)
	log.Printf("[service] COS Upload completed: url=%s", finalURL)
	return finalURL, nil
}

func (s *COSService) DeleteByURL(ctx context.Context, objectURL string) error {
	log.Printf("[service] COS DeleteByURL started: url=%s", objectURL)
	if s == nil || s.client == nil {
		return fmt.Errorf("cos service is not initialized")
	}

	if objectURL == "" {
		log.Printf("[service] COS DeleteByURL skipped: empty url")
		return nil
	}

	u, err := url.Parse(objectURL)
	if err != nil {
		log.Printf("[service] COS DeleteByURL url parse failure: url=%s error=%v", objectURL, err)
		return fmt.Errorf("parse object url: %w", err)
	}

	key := strings.TrimPrefix(path.Clean(u.Path), "/")
	if key == "" || key == "." {
		log.Printf("[service] COS DeleteByURL invalid key failure: url=%s key=%s", objectURL, key)
		return fmt.Errorf("invalid object key from url: %s", objectURL)
	}

	_, err = s.client.Object.Delete(ctx, key)
	if err != nil {
		log.Printf("[service] COS DeleteByURL deletion failure: url=%s key=%s error=%v", objectURL, key, err)
		return fmt.Errorf("delete object from cos: %w", err)
	}

	log.Printf("[service] COS DeleteByURL completed: url=%s", objectURL)
	return nil
}
