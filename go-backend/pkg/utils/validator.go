package utils

import (
	"errors"
	"regexp"
	"strings"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// 验证用户名
func (v *Validator) ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return errors.New("username length must be between 3 and 32")
	}

	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers and underscore")
	}

	return nil
}

// 验证密码
func (v *Validator) ValidatePassword(password string) error {
	if len(password) < 6 || len(password) > 20 {
		return errors.New("password length must be between 6 and 20")
	}

	return nil
}

// 验证邮箱
func (v *Validator) ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	return nil
}

// 验证视频标题
func (v *Validator) ValidateVideoTitle(title string) error {
	title = strings.TrimSpace(title)
	if len(title) == 0 {
		return errors.New("video title cannot be empty")
	}

	if len(title) > 50 {
		return errors.New("video title too long")
	}

	return nil
}

// 验证评论内容
func (v *Validator) ValidateComment(content string) error {
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return errors.New("comment content cannot be empty")
	}

	if len(content) > 200 {
		return errors.New("comment content too long")
	}

	return nil
}

// 验证用户ID
func (v *Validator) ValidateUserID(userID int64) error {
	if userID <= 0 {
		return errors.New("invalid user id")
	}

	return nil
}

// 验证视频ID
func (v *Validator) ValidateVideoID(videoID int64) error {
	if videoID <= 0 {
		return errors.New("invalid video id")
	}

	return nil
}

// 验证分页参数
func (v *Validator) ValidatePage(page, size int32) error {
	if page < 1 {
		return errors.New("page must be greater than 0")
	}

	if size < 1 || size > 50 {
		return errors.New("page size must be between 1 and 50")
	}

	return nil
}
