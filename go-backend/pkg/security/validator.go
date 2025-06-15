package security

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"github.com/nbutton23/zxcvbn-go"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// ValidateUsername 验证用户名格式
func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return errors.New("username length must be between 3 and 32")
	}

	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers and underscore")
	}

	return nil
}

// ValidatePassword 验证密码强度
func ValidatePassword(password string) error {
	if len(password) < 6 || len(password) > 128 {
		return errors.New("password length must be between 6 and 128")
	}

	// 基本字符检查
	var hasUpper, hasLower, hasNumber bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber {
		return errors.New("password must contain uppercase, lowercase and number")
	}

	// 使用zxcvbn检查密码强度
	strength := zxcvbn.PasswordStrength(password, nil)
	if strength.Score < 2 {
		return errors.New("password is too weak, please use a stronger password")
	}

	return nil
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) error {
	if len(email) > 254 {
		return errors.New("email too long")
	}

	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	return nil
}

// SanitizeInput 清理输入内容
func SanitizeInput(input string) string {
	// 移除前后空白
	input = strings.TrimSpace(input)

	// 移除潜在的SQL注入字符
	dangerous := []string{"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_"}
	for _, d := range dangerous {
		input = strings.ReplaceAll(input, d, "")
	}

	return input
}

// ValidateVideoTitle 验证视频标题
func ValidateVideoTitle(title string) error {
	title = strings.TrimSpace(title)
	if len(title) == 0 {
		return errors.New("title cannot be empty")
	}
	if len(title) > 100 {
		return errors.New("title too long, max 100 characters")
	}
	return nil
}

// ValidateComment 验证评论内容
func ValidateComment(content string) error {
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return errors.New("comment cannot be empty")
	}
	if len(content) > 500 {
		return errors.New("comment too long, max 500 characters")
	}
	return nil
}

// 以下为兼容性方法，保持原有的Validator结构体方法

// ValidateUserID 验证用户ID
func (v *Validator) ValidateUserID(userID int64) error {
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	return nil
}

// ValidateVideoID 验证视频ID
func (v *Validator) ValidateVideoID(videoID int64) error {
	if videoID <= 0 {
		return errors.New("invalid video id")
	}
	return nil
}

// ValidatePage 验证分页参数
func (v *Validator) ValidatePage(page, size int32) error {
	if page < 1 {
		return errors.New("page must be greater than 0")
	}
	if size < 1 || size > 50 {
		return errors.New("page size must be between 1 and 50")
	}
	return nil
}

// ValidateUsername 兼容性方法
func (v *Validator) ValidateUsername(username string) error {
	return ValidateUsername(username)
}

// ValidatePassword 兼容性方法
func (v *Validator) ValidatePassword(password string) error {
	return ValidatePassword(password)
}

// ValidateEmail 兼容性方法
func (v *Validator) ValidateEmail(email string) error {
	return ValidateEmail(email)
}
