package do

import "time"

const (
	// UserRoleUser 表示普通用户角色。
	UserRoleUser UserRole = "user"
	// UserRoleAdmin 表示管理员角色。
	UserRoleAdmin UserRole = "admin"
)

// UserRole 定义用户权限角色。
type UserRole string

// User 表示用户领域对象。
type User struct {
	ID                 int64
	Username           string
	Password           string
	PasswordHash       string
	Role               UserRole
	Nickname           string
	FavoriteTags       string
	Bio                string
	PublicProfile      bool
	EmailNotifications bool
	SyncCollections    bool
	CreatedAt          time.Time
}

// LoginResult 表示登录业务结果。
type LoginResult struct {
	Token string
}

// IsValid 判断角色是否属于受支持的集合。
func (r UserRole) IsValid() bool {
	return r == UserRoleUser || r == UserRoleAdmin
}

// Public 清除不应向外暴露的敏感字段并补齐公开默认值。
func (u User) Public() User {
	u.Password = ""
	u.PasswordHash = ""
	if u.Nickname == "" {
		u.Nickname = u.Username
	}
	return u
}
