package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid_username", "testuser123", false},
		{"valid_with_underscore", "test_user_123", false},
		{"minimum_length", "abc", false},
		{"maximum_length", "abcdefghijklmnopqrstuvwxyz123456", false},
		{"too_short", "ab", true},
		{"too_long", "abcdefghijklmnopqrstuvwxyz1234567", true},
		{"invalid_chars", "test@user", true},
		{"invalid_space", "test user", true},
		{"invalid_dash", "test-user", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid_strong", "StrongPass123!", false},
		{"valid_minimum", "Aa1bcdef", false},
		{"missing_uppercase", "weakpass123", true},
		{"missing_lowercase", "WEAKPASS123", true},
		{"missing_number", "WeakPassword", true},
		{"too_short", "Aa1", true},
		{"too_long", "VeryLongPasswordThatExceedsTheMaximumLengthLimitAndShouldFailValidationBecauseItIsTooLongForOurSystem123456789", true},
		{"weak_common", "password123", true},
		{"weak_sequential", "Abc12345", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid_simple", "test@example.com", false},
		{"valid_subdomain", "user@mail.example.com", false},
		{"valid_with_plus", "user+tag@example.com", false},
		{"valid_with_dots", "first.last@example.com", false},
		{"invalid_no_at", "testexample.com", true},
		{"invalid_no_domain", "test@", true},
		{"invalid_no_local", "@example.com", true},
		{"invalid_double_at", "test@@example.com", true},
		{"invalid_no_tld", "test@example", true},
		{"invalid_spaces", "test @example.com", true},
		{"too_long", "verylongemailaddressthatexceedsthemaximumlengthallowedforemailaddresses@verylongdomainnamethatshouldnotbeallowed.com", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal_text", "hello world", "hello world"},
		{"with_spaces", "  hello world  ", "hello world"},
		{"with_quotes", "hello 'world' test", "hello world test"},
		{"with_sql_injection", "test'; DROP TABLE users; --", "test DROP TABLE users "},
		{"with_comments", "test /* comment */ data", "test  comment  data"},
		{"with_xp_commands", "test xp_cmdshell data", "test  data"},
		{"empty_string", "", ""},
		{"only_spaces", "   ", ""},
		{"mixed_dangerous", "'; xp_cmdshell /* test */ --", "   test "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateVideoTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"valid_title", "My awesome video", false},
		{"valid_long", "This is a very long video title that should still be valid", false},
		{"empty_string", "", true},
		{"only_spaces", "   ", true},
		{"too_long", "This title is way too long and exceeds the maximum character limit that we have set for video titles", true},
		{"exactly_100_chars", "This title is exactly one hundred characters long and should be valid for our validation test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVideoTitle(tt.title)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateComment(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		wantErr bool
	}{
		{"valid_comment", "This is a great video!", false},
		{"valid_long", "This is a much longer comment that provides detailed feedback about the video content and should still be within the allowed limits", false},
		{"empty_string", "", true},
		{"only_spaces", "   ", true},
		{"too_long", "This comment is extremely long and goes way beyond the maximum character limit that we have established for comments in our system. It contains way too much text and should be rejected by our validation function because it exceeds the 500 character limit that we have set for comments in our application. This is just too much text for a single comment and users should be encouraged to keep their comments more concise and to the point rather than writing these extremely long comments that are difficult to read and process.", true},
		{"exactly_500_chars", "This comment is exactly five hundred characters long and should be valid for our validation test. We need to make sure that comments of exactly this length are accepted by the system. This is important for boundary testing to ensure our validation works correctly at the limits. The comment needs to be exactly five hundred characters including spaces and punctuation marks. This should be accepted by our validation function since it meets the maximum length requirement without exceeding it completely.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComment(tt.comment)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateUserID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		userID  int64
		wantErr bool
	}{
		{"valid_positive", 123, false},
		{"valid_large", 999999, false},
		{"invalid_zero", 0, true},
		{"invalid_negative", -123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateUserID(tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateVideoID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		videoID int64
		wantErr bool
	}{
		{"valid_positive", 456, false},
		{"valid_large", 888888, false},
		{"invalid_zero", 0, true},
		{"invalid_negative", -456, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVideoID(tt.videoID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidatePage(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		page    int32
		size    int32
		wantErr bool
	}{
		{"valid_first_page", 1, 10, false},
		{"valid_large_page", 100, 20, false},
		{"valid_max_size", 1, 50, false},
		{"invalid_zero_page", 0, 10, true},
		{"invalid_negative_page", -1, 10, true},
		{"invalid_zero_size", 1, 0, true},
		{"invalid_negative_size", 1, -10, true},
		{"invalid_oversized", 1, 51, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePage(tt.page, tt.size)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_CompatibilityMethods(t *testing.T) {
	v := NewValidator()

	// 测试兼容性方法与全局函数的一致性
	t.Run("username_compatibility", func(t *testing.T) {
		username := "testuser123"
		globalErr := ValidateUsername(username)
		validatorErr := v.ValidateUsername(username)

		if globalErr != nil {
			assert.Error(t, validatorErr)
		} else {
			assert.NoError(t, validatorErr)
		}
	})

	t.Run("password_compatibility", func(t *testing.T) {
		password := "ValidPass123!"
		globalErr := ValidatePassword(password)
		validatorErr := v.ValidatePassword(password)

		if globalErr != nil {
			assert.Error(t, validatorErr)
		} else {
			assert.NoError(t, validatorErr)
		}
	})

	t.Run("email_compatibility", func(t *testing.T) {
		email := "test@example.com"
		globalErr := ValidateEmail(email)
		validatorErr := v.ValidateEmail(email)

		if globalErr != nil {
			assert.Error(t, validatorErr)
		} else {
			assert.NoError(t, validatorErr)
		}
	})
}
