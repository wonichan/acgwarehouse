package repository

import (
	"context"
	stderrors "errors"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
	"github.com/yachiyo/acgwarehouse/internal/ports"
)

var (
	// ErrUserNotFound 表示用户不存在。
	ErrUserNotFound = ports.ErrUserNotFound
	// ErrUsernameExists 表示用户名已经存在。
	ErrUsernameExists = ports.ErrUsernameExists
)

// UserRepository 提供用户持久化访问。
type UserRepository struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewUserRepository 创建用户仓储。
func NewUserRepository(readDB *gorm.DB, writeDB *gorm.DB) *UserRepository {
	return &UserRepository{readDB: readDB, writeDB: writeDB}
}

// FindByUsername 按用户名查询用户。
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (do.User, error) {
	var user po.User
	err := r.readDB.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return do.User{}, pkgerrors.WithMessage(ErrUserNotFound, "find user by username")
	}
	if err != nil {
		return do.User{}, pkgerrors.WithMessage(err, "find user by username")
	}
	return toDO(user), nil
}

// FindByID 按用户 ID 查询用户。
func (r *UserRepository) FindByID(ctx context.Context, id int64) (do.User, error) {
	var user po.User
	err := r.readDB.WithContext(ctx).First(&user, id).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return do.User{}, pkgerrors.WithMessage(ErrUserNotFound, "find user by id")
	}
	if err != nil {
		return do.User{}, pkgerrors.WithMessage(err, "find user by id")
	}
	return toDO(user), nil
}

// Create 创建用户记录。
func (r *UserRepository) Create(ctx context.Context, user do.User) (do.User, error) {
	created := toPO(user)
	if err := r.writeDB.WithContext(ctx).Create(&created).Error; err != nil {
		if isUniqueConstraintError(err) {
			return do.User{}, pkgerrors.WithMessage(ErrUsernameExists, "create user")
		}
		return do.User{}, pkgerrors.WithMessage(err, "create user")
	}
	return toDO(created), nil
}

// toDO 将持久化对象转换为领域对象。
func toDO(user po.User) do.User {
	return do.User{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Role:         do.UserRole(user.Role),
		CreatedAt:    user.CreatedAt,
	}
}

// toPO 将领域对象转换为持久化对象。
func toPO(user do.User) po.User {
	return po.User{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
		CreatedAt:    user.CreatedAt,
	}
}

// isUniqueConstraintError 判断数据库错误是否为唯一约束冲突。
func isUniqueConstraintError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "constraint failed")
}
