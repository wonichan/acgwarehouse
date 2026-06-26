package cos

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	qcos "github.com/tencentyun/cos-go-sdk-v5"

	"github.com/yachiyo/acgwarehouse/internal/conf"
)

const (
	placeholderSecretID  = "COS_SECRET_ID_PLACEHOLDER"
	placeholderSecretKey = "COS_SECRET_KEY_PLACEHOLDER"
	listMaxKeys          = 1000
)

var (
	// ErrInvalidCredential 表示 COS 凭证缺失或仍为占位符。
	ErrInvalidCredential = pkgerrors.New("cos: invalid credential")
)

// Object 表示 COS 对象列表中的元数据。
type Object struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// Client 封装腾讯云 COS SDK 客户端。
type Client struct {
	sdk    *qcos.Client
	domain string
}

// NewClient 创建腾讯云 COS 客户端。
func NewClient(cfg conf.COSConfig) (*Client, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, pkgerrors.WithMessage(err, "validate cos config")
	}
	bucketURL, err := url.Parse(bucketEndpoint(cfg))
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "parse cos bucket url")
	}
	return &Client{
		sdk: qcos.NewClient(&qcos.BaseURL{BucketURL: bucketURL}, &http.Client{
			Timeout: 100 * time.Second,
			Transport: &qcos.AuthorizationTransport{
				SecretID:  cfg.SecretID,
				SecretKey: cfg.SecretKey,
			},
		}),
		domain: strings.TrimRight(cfg.Domain, "/"),
	}, nil
}

// ValidateConfig 校验 COS 同步所需配置。
func ValidateConfig(cfg conf.COSConfig) error {
	if isPlaceholder(cfg.SecretID, placeholderSecretID) || isPlaceholder(cfg.SecretKey, placeholderSecretKey) {
		return pkgerrors.WithMessage(ErrInvalidCredential, "cos credential is placeholder")
	}
	if strings.TrimSpace(cfg.Bucket) == "" || strings.TrimSpace(cfg.Region) == "" || strings.TrimSpace(cfg.Domain) == "" {
		return pkgerrors.New("cos bucket, region and domain must be configured")
	}
	return nil
}

// NormalizePrefix 将配置中的 COS 前缀标准化为 ListObjects 所需格式。
func NormalizePrefix(prefix string) string {
	trimmed := strings.Trim(strings.TrimSpace(prefix), "/")
	if trimmed == "" {
		return ""
	}
	return trimmed + "/"
}

// ListObjects 分页列举指定前缀下的全部 COS 对象。
func (c *Client) ListObjects(ctx context.Context, prefix string) ([]Object, error) {
	if c == nil || c.sdk == nil {
		return nil, pkgerrors.New("cos client is nil")
	}
	marker := ""
	objects := make([]Object, 0)
	for {
		result, _, err := c.sdk.Bucket.Get(ctx, &qcos.BucketGetOptions{
			Prefix:       NormalizePrefix(prefix),
			Marker:       marker,
			MaxKeys:      listMaxKeys,
			EncodingType: "url",
		})
		if err != nil {
			return nil, pkgerrors.WithMessage(err, "list cos objects")
		}
		for _, item := range result.Contents {
			object, err := decodeObject(item)
			if err != nil {
				return nil, pkgerrors.WithMessage(err, "decode cos object")
			}
			objects = append(objects, object)
		}
		if !result.IsTruncated {
			break
		}
		marker = result.NextMarker
	}
	return objects, nil
}

// ObjectURL 返回对象可公开访问的完整 URL。
func (c *Client) ObjectURL(key string) string {
	return strings.TrimRight(c.domain, "/") + "/" + strings.TrimLeft(key, "/")
}

// bucketEndpoint 根据 bucket 与 region 构造 COS bucket URL。
func bucketEndpoint(cfg conf.COSConfig) string {
	return "https://" + cfg.Bucket + ".cos." + cfg.Region + ".myqcloud.com"
}

// decodeObject 将 SDK 对象元数据转换为内部对象。
func decodeObject(item qcos.Object) (Object, error) {
	key, err := qcos.DecodeURIComponent(item.Key)
	if err != nil {
		return Object{}, pkgerrors.WithMessage(err, "decode object key")
	}
	lastModified, err := time.Parse(time.RFC3339, item.LastModified)
	if err != nil {
		return Object{}, pkgerrors.WithMessage(err, "parse object last modified")
	}
	return Object{Key: key, Size: int64(item.Size), LastModified: lastModified.UTC()}, nil
}

// isPlaceholder 判断凭证是否为空或占位符。
func isPlaceholder(value string, placeholder string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed == "" || trimmed == placeholder
}
