// Package frontendserver serves the built Vue SPA and proxies backend API requests.
package frontendserver

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config 定义前端静态服务与后端代理地址。
type Config struct {
	DistDir             string
	BackendURL          string
	MaxRequestBodyBytes int64
}

// NewHandler 创建先代理 /api/* 再回退到 SPA index 的 HTTP 处理器。
func NewHandler(cfg Config) (http.Handler, error) {
	backendURL, err := url.Parse(strings.TrimSpace(cfg.BackendURL))
	if err != nil {
		return nil, err
	}
	if backendURL.Scheme == "" || backendURL.Host == "" {
		return nil, errors.New("frontend backend url must include scheme and host")
	}
	if backendURL.Scheme != "http" && backendURL.Scheme != "https" {
		return nil, errors.New("frontend backend url must use http or https")
	}
	indexPath := filepath.Join(cfg.DistDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(backendURL)
	files := http.FileServer(http.Dir(cfg.DistDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if exceedsBodyLimit(r, cfg.MaxRequestBodyBytes) {
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
			if cfg.MaxRequestBodyBytes > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxRequestBodyBytes)
			}
			proxy.ServeHTTP(w, r)
			return
		}
		if shouldServeIndex(cfg.DistDir, r) {
			http.ServeFile(w, r, indexPath)
			return
		}
		files.ServeHTTP(w, r)
	}), nil
}

// shouldServeIndex 判断请求是否需要回退到 Vue Router 入口文件。
func shouldServeIndex(distDir string, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	path := filepath.Clean("/" + r.URL.Path)
	info, err := os.Stat(filepath.Join(distDir, path))
	return err != nil || info.IsDir()
}

// exceedsBodyLimit 判断带 Content-Length 的 API 请求是否超过代理上限。
func exceedsBodyLimit(r *http.Request, limit int64) bool {
	return limit > 0 && r.ContentLength > limit
}
