package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
)

var (
	// ErrInvalidToken 表示令牌签名或结构非法。
	ErrInvalidToken = pkgerrors.New("jwt: invalid token")
	// ErrExpiredToken 表示令牌已经过期。
	ErrExpiredToken = pkgerrors.New("jwt: expired token")
)

const (
	jwtType      = "JWT"
	jwtAlgorithm = "HS256"
	jwtPartCount = 3
)

// Claims 保存访问令牌中的认证信息。
type Claims struct {
	UserID   int64     `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	ExpireAt time.Time `json:"-"`
}

// Manager 负责 HS256 JWT 的签发与解析。
type Manager struct {
	secret   []byte
	duration time.Duration
}

type tokenHeader struct {
	Type      string `json:"typ"`
	Algorithm string `json:"alg"`
}

type tokenPayload struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	ExpireAt int64  `json:"exp"`
}

// NewManager 创建 JWT 管理器。
func NewManager(secret string, duration time.Duration) *Manager {
	return &Manager{secret: []byte(secret), duration: duration}
}

// Sign 签发 HS256 JWT。
func (m *Manager) Sign(claims Claims, now time.Time) (string, error) {
	expireAt := claims.ExpireAt
	if expireAt.IsZero() {
		expireAt = now.Add(m.duration)
	}
	payload := tokenPayload{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		ExpireAt: expireAt.Unix(),
	}
	head, err := encodeJSON(tokenHeader{Type: jwtType, Algorithm: jwtAlgorithm})
	if err != nil {
		return "", pkgerrors.WithMessage(err, "encode jwt header")
	}
	body, err := encodeJSON(payload)
	if err != nil {
		return "", pkgerrors.WithMessage(err, "encode jwt payload")
	}
	signingInput := head + "." + body
	return signingInput + "." + sign(signingInput, m.secret), nil
}

// Parse 校验并解析 HS256 JWT。
func (m *Manager) Parse(token string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != jwtPartCount {
		return Claims{}, pkgerrors.WithMessage(ErrInvalidToken, "split jwt")
	}
	header, err := decodeHeader(parts[0])
	if err != nil {
		return Claims{}, pkgerrors.WithMessage(err, "decode jwt header")
	}
	if header.Type != jwtType || header.Algorithm != jwtAlgorithm {
		return Claims{}, pkgerrors.WithMessage(ErrInvalidToken, "validate jwt header")
	}
	signingInput := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(sign(signingInput, m.secret))) {
		return Claims{}, pkgerrors.WithMessage(ErrInvalidToken, "verify jwt signature")
	}
	payload, err := decodePayload(parts[1])
	if err != nil {
		return Claims{}, pkgerrors.WithMessage(err, "decode jwt payload")
	}
	if now.Unix() >= payload.ExpireAt {
		return Claims{}, pkgerrors.WithMessage(ErrExpiredToken, "check jwt exp")
	}
	return Claims{
		UserID:   payload.UserID,
		Username: payload.Username,
		Role:     payload.Role,
		ExpireAt: time.Unix(payload.ExpireAt, 0).UTC(),
	}, nil
}

// encodeJSON 将对象序列化为 JWT base64url 片段。
func encodeJSON(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", pkgerrors.WithMessage(err, "marshal jwt json")
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// decodeHeader 解析并校验 JWT header 片段的 JSON 结构。
func decodeHeader(encoded string) (tokenHeader, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return tokenHeader{}, pkgerrors.WithMessage(ErrInvalidToken, "decode base64 header")
	}
	var header tokenHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return tokenHeader{}, pkgerrors.WithMessage(ErrInvalidToken, "unmarshal header")
	}
	return header, nil
}

// decodePayload 解析 JWT payload 片段。
func decodePayload(encoded string) (tokenPayload, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return tokenPayload{}, pkgerrors.WithMessage(ErrInvalidToken, "decode base64 payload")
	}
	var payload tokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return tokenPayload{}, pkgerrors.WithMessage(ErrInvalidToken, "unmarshal payload")
	}
	if payload.UserID == 0 || payload.Username == "" || payload.Role == "" || payload.ExpireAt == 0 {
		return tokenPayload{}, pkgerrors.WithMessage(ErrInvalidToken, "validate payload")
	}
	return payload, nil
}

// sign 生成 HS256 签名片段。
func sign(signingInput string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
