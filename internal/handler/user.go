package handler

import (
	"context"
	stderrors "errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/dto"
	"github.com/yachiyo/acgwarehouse/internal/repository"
	"github.com/yachiyo/acgwarehouse/internal/service"
	apperrors "github.com/yachiyo/acgwarehouse/pkg/errors"
)

// UserHandler 处理用户与认证 HTTP 请求。
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler 创建用户处理器。
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Register 处理用户注册请求。
func (h *UserHandler) Register(c context.Context, ctx *app.RequestContext) {
	input, ok := bindCredential(c, ctx)
	if !ok {
		return
	}
	created, err := h.userService.Register(c, do.User{
		Username: input.Username,
		Password: input.Password,
		Role:     do.UserRoleUser,
	})
	if err != nil {
		writeUserError(c, ctx, err)
		return
	}
	Success(ctx, toUserResponse(created))
}

// Login 处理用户登录请求。
func (h *UserHandler) Login(c context.Context, ctx *app.RequestContext) {
	input, ok := bindCredential(c, ctx)
	if !ok {
		return
	}
	result, err := h.userService.Login(c, do.User{Username: input.Username, Password: input.Password})
	if err != nil {
		writeUserError(c, ctx, err)
		return
	}
	Success(ctx, dto.LoginResponse{Token: result.Token})
}

// Me 返回当前登录用户公开信息。
func (h *UserHandler) Me(c context.Context, ctx *app.RequestContext) {
	id, ok := requiredCurrentUserID(c, ctx)
	if !ok {
		return
	}
	user, err := h.userService.CurrentUser(c, id)
	if err != nil {
		writeUserError(c, ctx, err)
		return
	}
	Success(ctx, toUserResponse(user))
}

// UpdateMe 更新当前登录用户公开资料和偏好设置。
func (h *UserHandler) UpdateMe(c context.Context, ctx *app.RequestContext) {
	id, ok := requiredCurrentUserID(c, ctx)
	if !ok {
		return
	}
	var input dto.UserProfileUpdateRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "参数错误", err)
		return
	}
	updated, err := h.userService.UpdateCurrentUserProfile(c, id, do.User{
		Nickname:           input.Nickname,
		FavoriteTags:       input.FavoriteTags,
		Bio:                input.Bio,
		PublicProfile:      input.PublicProfile,
		EmailNotifications: input.EmailNotifications,
		SyncCollections:    input.SyncCollections,
	})
	if err != nil {
		writeUserError(c, ctx, err)
		return
	}
	Success(ctx, toUserResponse(updated))
}

// ChangePassword 更新当前登录用户密码。
func (h *UserHandler) ChangePassword(c context.Context, ctx *app.RequestContext) {
	id, ok := requiredCurrentUserID(c, ctx)
	if !ok {
		return
	}
	var input dto.UserPasswordUpdateRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "参数错误", err)
		return
	}
	if err := h.userService.ChangePassword(c, id, input.OldPassword, input.NewPassword); err != nil {
		writeUserError(c, ctx, err)
		return
	}
	Success(ctx, nil)
}

// requiredCurrentUserID 读取认证中间件注入的当前用户 ID。
func requiredCurrentUserID(c context.Context, ctx *app.RequestContext) (int64, bool) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		Fail(c, ctx, consts.StatusUnauthorized, apperrors.CodeUnauthorized, "请先登录", nil)
		return 0, false
	}
	id, ok := userID.(int64)
	if !ok {
		Fail(c, ctx, consts.StatusUnauthorized, apperrors.CodeUnauthorized, "请先登录", nil)
		return 0, false
	}
	return id, true
}

// bindCredential 绑定并校验用户凭据请求体。
func bindCredential(c context.Context, ctx *app.RequestContext) (dto.UserCredentialRequest, bool) {
	var input dto.UserCredentialRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "参数错误", err)
		return dto.UserCredentialRequest{}, false
	}
	return input, true
}

// writeUserError 将用户业务错误映射为统一响应。
func writeUserError(c context.Context, ctx *app.RequestContext, err error) {
	switch {
	case stderrors.Is(err, service.ErrInvalidUserInput):
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "参数不符合规则", err)
	case stderrors.Is(err, repository.ErrUsernameExists):
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "用户名已存在", err)
	case stderrors.Is(err, service.ErrInvalidCredential):
		Fail(c, ctx, consts.StatusUnauthorized, apperrors.CodeUnauthorized, "用户名或密码错误", err)
	case stderrors.Is(err, repository.ErrUserNotFound):
		Fail(c, ctx, consts.StatusNotFound, apperrors.CodeNotFound, "用户不存在", err)
	default:
		Fail(c, ctx, consts.StatusInternalServerError, apperrors.CodeInternal, "系统异常", err)
	}
}

// toUserResponse 将用户领域对象转换为 HTTP 响应 DTO。
func toUserResponse(user do.User) dto.UserResponse {
	return dto.UserResponse{
		ID:                 user.ID,
		Username:           user.Username,
		Role:               string(user.Role),
		CreatedAt:          FormatTime(user.CreatedAt),
		Nickname:           user.Nickname,
		FavoriteTags:       user.FavoriteTags,
		Bio:                user.Bio,
		PublicProfile:      user.PublicProfile,
		EmailNotifications: user.EmailNotifications,
		SyncCollections:    user.SyncCollections,
	}
}
