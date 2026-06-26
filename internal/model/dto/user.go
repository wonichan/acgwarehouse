package dto

// UserCredentialRequest 表示注册/登录请求体。
type UserCredentialRequest struct {
	Username string `json:"username" vd:"len($) >= 3 && len($) <= 32"`
	Password string `json:"password" vd:"len($) >= 6"`
}

// LoginResponse 表示登录响应数据。
type LoginResponse struct {
	Token string `json:"token"`
}

// UserResponse 表示用户公开响应数据。
type UserResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}
