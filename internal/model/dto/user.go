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

// UserProfileUpdateRequest 表示当前用户资料与偏好更新请求体。
type UserProfileUpdateRequest struct {
	Nickname           string `json:"nickname"`
	FavoriteTags       string `json:"favorite_tags"`
	Bio                string `json:"bio"`
	PublicProfile      bool   `json:"public_profile"`
	EmailNotifications bool   `json:"email_notifications"`
	SyncCollections    bool   `json:"sync_collections"`
}

// UserPasswordUpdateRequest 表示当前用户密码修改请求体。
type UserPasswordUpdateRequest struct {
	OldPassword string `json:"old_password" vd:"len($) >= 6"`
	NewPassword string `json:"new_password" vd:"len($) >= 6"`
}

// UserResponse 表示用户公开响应数据。
type UserResponse struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	CreatedAt          string `json:"created_at"`
	Nickname           string `json:"nickname"`
	FavoriteTags       string `json:"favorite_tags"`
	Bio                string `json:"bio"`
	PublicProfile      bool   `json:"public_profile"`
	EmailNotifications bool   `json:"email_notifications"`
	SyncCollections    bool   `json:"sync_collections"`
	Points             int64  `json:"points"`
}

// MonthlyCheckInsResponse 表示用户月度签到查询响应数据。
type MonthlyCheckInsResponse struct {
	Dates       []string `json:"dates"`
	TotalPoints int64    `json:"total_points"`
}
