package jwt_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

const testSecret = "test-secret"

type testTokenHeader struct {
	Type      string `json:"typ"`
	Algorithm string `json:"alg"`
}

type testTokenPayload struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	ExpireAt int64  `json:"exp"`
}

type testTokenParts struct {
	HeaderSegment  string
	PayloadSegment string
	Secret         string
}

func Test_Manager_Parse_returns_claims_when_token_valid(t *testing.T) {
	// Given
	manager := jwtpkg.NewManager(testSecret, time.Hour)
	issuedAt := time.Date(2026, 6, 26, 1, 2, 3, 0, time.UTC)
	claims := jwtpkg.Claims{
		UserID:   7,
		Username: "alice",
		Role:     "user",
		ExpireAt: issuedAt.Add(time.Hour),
	}

	// When
	token, err := manager.Sign(claims, issuedAt)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	got, err := manager.Parse(token, issuedAt.Add(time.Minute))

	// Then
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if got.UserID != claims.UserID {
		t.Fatalf("user id = %d, want %d", got.UserID, claims.UserID)
	}
	if got.Username != claims.Username {
		t.Fatalf("username = %q, want %q", got.Username, claims.Username)
	}
	if got.Role != claims.Role {
		t.Fatalf("role = %q, want %q", got.Role, claims.Role)
	}
}

func Test_Manager_Parse_rejects_expired_token(t *testing.T) {
	// Given
	manager := jwtpkg.NewManager(testSecret, time.Hour)
	issuedAt := time.Date(2026, 6, 26, 1, 2, 3, 0, time.UTC)
	token, err := manager.Sign(jwtpkg.Claims{
		UserID:   7,
		Username: "alice",
		Role:     "user",
		ExpireAt: issuedAt.Add(time.Hour),
	}, issuedAt)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	// When
	_, err = manager.Parse(token, issuedAt.Add(2*time.Hour))

	// Then
	if !errors.Is(err, jwtpkg.ErrExpiredToken) {
		t.Fatalf("error = %v, want expired token", err)
	}
}

func Test_Manager_Parse_rejects_tampered_token(t *testing.T) {
	// Given
	manager := jwtpkg.NewManager(testSecret, time.Hour)
	issuedAt := time.Date(2026, 6, 26, 1, 2, 3, 0, time.UTC)
	token, err := manager.Sign(jwtpkg.Claims{
		UserID:   7,
		Username: "alice",
		Role:     "user",
		ExpireAt: issuedAt.Add(time.Hour),
	}, issuedAt)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	// When
	_, err = manager.Parse(token+"tampered", issuedAt.Add(time.Minute))

	// Then
	if !errors.Is(err, jwtpkg.ErrInvalidToken) {
		t.Fatalf("error = %v, want invalid token", err)
	}
}

func Test_Manager_Parse_rejects_token_when_header_algorithm_is_not_hs256(t *testing.T) {
	// Given
	manager := jwtpkg.NewManager(testSecret, time.Hour)
	issuedAt := time.Date(2026, 6, 26, 1, 2, 3, 0, time.UTC)
	header := encodeTestHeader(t, testTokenHeader{Type: "JWT", Algorithm: "none"})
	payload := encodeTestPayload(t, validTestPayload(issuedAt.Add(time.Hour)))
	token := signTestToken(testTokenParts{HeaderSegment: header, PayloadSegment: payload, Secret: testSecret})

	// When
	_, err := manager.Parse(token, issuedAt.Add(time.Minute))

	// Then
	if !errors.Is(err, jwtpkg.ErrInvalidToken) {
		t.Fatalf("error = %v, want invalid token", err)
	}
}

func Test_Manager_Parse_rejects_token_when_header_type_is_not_jwt(t *testing.T) {
	// Given
	manager := jwtpkg.NewManager(testSecret, time.Hour)
	issuedAt := time.Date(2026, 6, 26, 1, 2, 3, 0, time.UTC)
	header := encodeTestHeader(t, testTokenHeader{Type: "JWS", Algorithm: "HS256"})
	payload := encodeTestPayload(t, validTestPayload(issuedAt.Add(time.Hour)))
	token := signTestToken(testTokenParts{HeaderSegment: header, PayloadSegment: payload, Secret: testSecret})

	// When
	_, err := manager.Parse(token, issuedAt.Add(time.Minute))

	// Then
	if !errors.Is(err, jwtpkg.ErrInvalidToken) {
		t.Fatalf("error = %v, want invalid token", err)
	}
}

func Test_Manager_Parse_rejects_token_when_header_is_malformed(t *testing.T) {
	// Given
	manager := jwtpkg.NewManager(testSecret, time.Hour)
	issuedAt := time.Date(2026, 6, 26, 1, 2, 3, 0, time.UTC)
	header := base64.RawURLEncoding.EncodeToString([]byte("not-json"))
	payload := encodeTestPayload(t, validTestPayload(issuedAt.Add(time.Hour)))
	token := signTestToken(testTokenParts{HeaderSegment: header, PayloadSegment: payload, Secret: testSecret})

	// When
	_, err := manager.Parse(token, issuedAt.Add(time.Minute))

	// Then
	if !errors.Is(err, jwtpkg.ErrInvalidToken) {
		t.Fatalf("error = %v, want invalid token", err)
	}
}

// validTestPayload 构造默认合法测试载荷。
func validTestPayload(expireAt time.Time) testTokenPayload {
	return testTokenPayload{UserID: 7, Username: "alice", Role: "user", ExpireAt: expireAt.Unix()}
}

// encodeTestHeader 将测试 JOSE header 编码为 JWT 片段。
func encodeTestHeader(t *testing.T, header testTokenHeader) string {
	t.Helper()
	data, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

// encodeTestPayload 将测试 payload 编码为 JWT 片段。
func encodeTestPayload(t *testing.T, payload testTokenPayload) string {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

// signTestToken 使用测试密钥为指定片段生成 HMAC 签名。
func signTestToken(parts testTokenParts) string {
	signingInput := parts.HeaderSegment + "." + parts.PayloadSegment
	mac := hmac.New(sha256.New, []byte(parts.Secret))
	mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
