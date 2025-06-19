package utils

import (
	"fmt"
	"net/http"

	"github.com/go-kratos/kratos/v2/errors"
	"go-backend/api/common/v1"
)

// 预定义错误
var (
	ErrInvalidParam     = NewBadRequestError(v1.ErrorCode_PARAM_ERROR, "invalid parameter")
	ErrTokenInvalid     = NewUnauthorizedError(v1.ErrorCode_TOKEN_INVALID, "invalid token")
	ErrTokenExpired     = NewUnauthorizedError(v1.ErrorCode_TOKEN_EXPIRED, "token expired")
	ErrPermissionDenied = NewForbiddenError(v1.ErrorCode_PERMISSION_DENIED, "permission denied")
	ErrServerError      = NewInternalError(v1.ErrorCode_SERVER_ERROR, "internal server error")

	// 用户相关错误
	ErrUserNotFound   = NewNotFoundError(v1.ErrorCode_USER_NOT_EXIST, "user not found")
	ErrUserExists     = NewBadRequestError(v1.ErrorCode_USER_EXIST, "user already exists")
	ErrPasswordError  = NewBadRequestError(v1.ErrorCode_PASSWORD_ERROR, "invalid password")
	ErrRegisterFailed = NewBadRequestError(v1.ErrorCode_REGISTER_FAILED, "register failed")

	// 关系相关错误
	ErrAlreadyFollow = NewBadRequestError(v1.ErrorCode_ALREADY_FOLLOW, "already followed")
	ErrNotFollow     = NewBadRequestError(v1.ErrorCode_NOT_FOLLOW, "not following")

	// 视频相关错误
	ErrVideoNotFound   = NewNotFoundError(v1.ErrorCode_VIDEO_NOT_EXIST, "video not found")
	ErrVideoUploadFail = NewBadRequestError(v1.ErrorCode_VIDEO_UPLOAD_FAIL, "video upload failed")
	ErrVideoFormatErr  = NewBadRequestError(v1.ErrorCode_VIDEO_FORMAT_ERR, "invalid video format")
	ErrVideoSizeErr    = NewBadRequestError(v1.ErrorCode_VIDEO_SIZE_ERR, "video size too large")
)

// NewBadRequestError 创建400错误
func NewBadRequestError(code v1.ErrorCode, message string) *errors.Error {
	return errors.New(http.StatusBadRequest, code.String(), message)
}

// NewUnauthorizedError 创建401错误
func NewUnauthorizedError(code v1.ErrorCode, message string) *errors.Error {
	return errors.New(http.StatusUnauthorized, code.String(), message)
}

// NewForbiddenError 创建403错误
func NewForbiddenError(code v1.ErrorCode, message string) *errors.Error {
	return errors.New(http.StatusForbidden, code.String(), message)
}

// NewNotFoundError 创建404错误
func NewNotFoundError(code v1.ErrorCode, message string) *errors.Error {
	return errors.New(http.StatusNotFound, code.String(), message)
}

// NewInternalError 创建500错误
func NewInternalError(code v1.ErrorCode, message string) *errors.Error {
	return errors.New(http.StatusInternalServerError, code.String(), message)
}

// WrapError 包装错误
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// IsKratosError 检查是否为Kratos错误
func IsKratosError(err error) bool {
	_, ok := err.(*errors.Error)
	return ok
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) v1.ErrorCode {
	if kratosErr, ok := err.(*errors.Error); ok {
		// 从错误原因中解析错误码
		switch kratosErr.Reason {
		case v1.ErrorCode_PARAM_ERROR.String():
			return v1.ErrorCode_PARAM_ERROR
		case v1.ErrorCode_TOKEN_INVALID.String():
			return v1.ErrorCode_TOKEN_INVALID
		case v1.ErrorCode_USER_NOT_EXIST.String():
			return v1.ErrorCode_USER_NOT_EXIST
		case v1.ErrorCode_USER_EXIST.String():
			return v1.ErrorCode_USER_EXIST
		case v1.ErrorCode_VIDEO_NOT_EXIST.String():
			return v1.ErrorCode_VIDEO_NOT_EXIST
		case v1.ErrorCode_VIDEO_UPLOAD_FAIL.String():
			return v1.ErrorCode_VIDEO_UPLOAD_FAIL
		case v1.ErrorCode_VIDEO_FORMAT_ERR.String():
			return v1.ErrorCode_VIDEO_FORMAT_ERR
		case v1.ErrorCode_VIDEO_SIZE_ERR.String():
			return v1.ErrorCode_VIDEO_SIZE_ERR
		default:
			return v1.ErrorCode_SERVER_ERROR
		}
	}
	return v1.ErrorCode_SERVER_ERROR
}
