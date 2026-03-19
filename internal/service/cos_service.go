package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
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

func (s *COSService) Upload(ctx context.Context, imageID int64, size string, data []byte) (string, error) {
	key := fmt.Sprintf("thumbnails/%d_%s.jpg", imageID, size)

	_, err := s.client.Object.Put(ctx, key, bytes.NewReader(data), &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: "image/jpeg",
		},
		ACLHeaderOptions: &cos.ACLHeaderOptions{
			XCosACL: "public-read",
		},
	})
	if err != nil {
		return "", fmt.Errorf("upload thumbnail to cos: %w", err)
	}

	// Ensure bucketURL has https:// protocol
	uploadURL := s.bucketURL
	if !strings.HasPrefix(uploadURL, "http://") && !strings.HasPrefix(uploadURL, "https://") {
		uploadURL = "https://" + uploadURL
	}

	return fmt.Sprintf("%s/%s", uploadURL, key), nil
}
