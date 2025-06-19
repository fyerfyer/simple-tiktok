package security

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"unicode"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// 常见弱密码模式 - 只检查完全匹配或作为独立词汇的弱密码
	weakPatterns = []string{
		"123456", "123456789", "12345678", "12345",
		"1234567", "qwerty", "abc123", "admin",
		"letmein", "welcome", "654321", "qwerty123",
	}

	// 连续字符模式 - 只检查明显的序列模式
	sequentialPatterns = []string{
		"qwertyuiop", "asdfghjkl", "zxcvbnm",
		"0123456789", "abcdefghijklmnopqrstuvwxyz",
		"qwerty", "asdf", "zxcv",
		"12345", "23456", "34567", "45678", "56789",
	}
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

	// 基本字符类型检查
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// 计算字符类型数量
	charTypeCount := 0
	if hasUpper {
		charTypeCount++
	}
	if hasLower {
		charTypeCount++
	}
	if hasNumber {
		charTypeCount++
	}
	if hasSpecial {
		charTypeCount++
	}

	// 密码强度要求：
	// 1. 长度 >= 8: 至少需要3种字符类型
	// 2. 长度 < 8: 至少需要3种字符类型
	if charTypeCount < 3 {
		return errors.New("password must contain at least 3 character types (uppercase, lowercase, number, special)")
	}

	// 检查是否包含完整的常见弱密码模式
	lowerPassword := strings.ToLower(password)
	for _, weak := range weakPatterns {
		if strings.Contains(lowerPassword, weak) && len(weak) >= 5 {
			return errors.New("password contains common weak patterns")
		}
	}

	// 检查明显的连续字符模式（只检查长度>=5的模式）
	for _, pattern := range sequentialPatterns {
		if len(pattern) >= 5 && strings.Contains(lowerPassword, pattern) {
			return errors.New("password contains sequential characters")
		}
	}

	// 检查重复字符（超过3个相同字符连续出现）
	if hasRepeatingChars(password, 4) {
		return errors.New("password contains too many repeating characters")
	}

	// 检查是否全为数字或全为字母（只对单一字符类型严格要求）
	if charTypeCount == 1 {
		if isAllDigits(password) {
			return errors.New("password cannot be all numbers")
		}
		if isAllLetters(password) {
			return errors.New("password cannot be all letters")
		}
	}

	return nil
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) error {
	log.Printf("DEBUG: Validating email: %s", email)
	log.Printf("DEBUG: Email length: %d", len(email))

	if len(email) > 254 {
		log.Printf("DEBUG: Email too long, length: %d", len(email))
		return errors.New("email too long")
	}

	if !emailRegex.MatchString(email) {
		log.Printf("DEBUG: Email format invalid")
		return errors.New("invalid email format")
	}

	log.Printf("DEBUG: Email validation passed")
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

// hasRepeatingChars 检查是否有重复字符
func hasRepeatingChars(password string, maxRepeat int) bool {
	if len(password) < maxRepeat {
		return false
	}

	for i := 0; i <= len(password)-maxRepeat; i++ {
		char := password[i]
		count := 1
		for j := i + 1; j < len(password) && j < i+maxRepeat; j++ {
			if password[j] == char {
				count++
			} else {
				break
			}
		}
		if count >= maxRepeat {
			return true
		}
	}
	return false
}

// isAllDigits 检查是否全为数字
func isAllDigits(s string) bool {
	for _, char := range s {
		if !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}

// isAllLetters 检查是否全为字母
func isAllLetters(s string) bool {
	for _, char := range s {
		if !unicode.IsLetter(char) {
			return false
		}
	}
	return true
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

// ValidateVideoTitle 验证视频标题
func (v *Validator) ValidateVideoTitle(title string) error {
	return ValidateVideoTitle(title)
}

// ValidateComment 验证评论内容
func (v *Validator) ValidateComment(content string) error {
	return ValidateComment(content)
}
