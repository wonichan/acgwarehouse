package service

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

var (
	// ErrUserNotFound 表示用户不存在。
	ErrUserNotFound = repository.ErrUserNotFound
	// ErrUsernameExists 表示用户名已经存在。
	ErrUsernameExists = repository.ErrUsernameExists
	// ErrInvalidCredential 表示登录凭据非法。
	ErrInvalidCredential = pkgerrors.New("service: invalid credential")
	// ErrInvalidUserInput 表示用户输入非法。
	ErrInvalidUserInput = pkgerrors.New("service: invalid user input")
)

const (
	minUsernameLength = 3
	maxUsernameLength = 32
	minPasswordLength = 6
	bcryptCost        = 12
)

// UserRepository 定义用户服务依赖的仓储能力。
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (do.User, error)
	FindByID(ctx context.Context, id int64) (do.User, error)
	Create(ctx context.Context, user do.User) (do.User, error)
}

// UserService 提供用户注册、登录和当前用户查询能力。
type UserService struct {
	repo       UserRepository
	jwtManager *jwtpkg.Manager
}

// NewUserService 创建用户服务。
func NewUserService(repo UserRepository, jwtManager *jwtpkg.Manager) *UserService {
	return &UserService{repo: repo, jwtManager: jwtManager}
}

// Register 注册普通用户或管理员用户。
func (s *UserService) Register(ctx context.Context, user do.User) (do.User, error) {
	prepared, err := prepareNewUser(user)
	if err != nil {
		return do.User{}, pkgerrors.WithMessage(err, "prepare new user")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(prepared.Password), bcryptCost)
	if err != nil {
		return do.User{}, pkgerrors.WithMessage(err, "hash password")
	}
	prepared.PasswordHash = string(hash)
	prepared.Password = ""
	created, err := s.repo.Create(ctx, prepared)
	if err != nil {
		return do.User{}, pkgerrors.WithMessage(err, "create user")
	}
	return created.Public(), nil
}

// Login 校验用户密码并签发访问令牌。
func (s *UserService) Login(ctx context.Context, input do.User) (do.LoginResult, error) {
	user, err := s.repo.FindByUsername(ctx, strings.TrimSpace(input.Username))
	if stderrors.Is(err, repository.ErrUserNotFound) {
		return do.LoginResult{}, pkgerrors.WithMessage(ErrInvalidCredential, "find login user")
	}
	if err != nil {
		return do.LoginResult{}, pkgerrors.WithMessage(err, "find login user")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return do.LoginResult{}, pkgerrors.WithMessage(ErrInvalidCredential, "compare password")
	}
	now := time.Now().UTC()
	token, err := s.jwtManager.Sign(jwtpkg.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(user.Role),
	}, now)
	if err != nil {
		return do.LoginResult{}, pkgerrors.WithMessage(err, "sign login token")
	}
	return do.LoginResult{Token: token}, nil
}

// CurrentUser 获取当前用户公开信息。
func (s *UserService) CurrentUser(ctx context.Context, userID int64) (do.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return do.User{}, pkgerrors.WithMessage(err, "find current user")
	}
	return user.Public(), nil
}

// EnsureAdmin 创建首个管理员，若用户名已存在则保持幂等跳过。
func (s *UserService) EnsureAdmin(ctx context.Context, username string, password string) error {
	_, err := s.Register(ctx, do.User{Username: username, Password: password, Role: do.UserRoleAdmin})
	if stderrors.Is(err, repository.ErrUsernameExists) {
		return nil
	}
	if err != nil {
		return pkgerrors.WithMessage(err, "register bootstrap admin")
	}
	return nil
}

// prepareNewUser 校验并规范化待创建用户。
func prepareNewUser(user do.User) (do.User, error) {
	user.Username = strings.TrimSpace(user.Username)
	if !isValidUsername(user.Username) || len(user.Password) < minPasswordLength {
		return do.User{}, pkgerrors.WithMessage(ErrInvalidUserInput, "validate user credential")
	}
	if user.Role == "" {
		user.Role = do.UserRoleUser
	}
	if !user.Role.IsValid() {
		return do.User{}, pkgerrors.WithMessage(ErrInvalidUserInput, "validate user role")
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	return user, nil
}

// isValidUsername 判断用户名长度是否满足规则。
func isValidUsername(username string) bool {
	return len(username) >= minUsernameLength && len(username) <= maxUsernameLength
}
