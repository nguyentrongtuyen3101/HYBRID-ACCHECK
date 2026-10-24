package domain

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type SystemRole string

const (
	SystemRoleAdmin SystemRole = "admin"
	SystemRoleUser  SystemRole = "user"
)

func (r SystemRole) Valid() bool { return r == SystemRoleAdmin || r == SystemRoleUser }

type UserStatus string

const (
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
)

func (s UserStatus) Valid() bool { return s == UserActive || s == UserDisabled }

// Locale là ngôn ngữ giao diện / email của người dùng.
type Locale string

const (
	LocaleVI      Locale = "vi"
	LocaleEN      Locale = "en"
	LocaleJA      Locale = "ja"
	DefaultLocale        = LocaleVI
)

func (l Locale) Valid() bool { return l == LocaleVI || l == LocaleEN || l == LocaleJA }

type Theme string

const (
	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"
)

func (t Theme) Valid() bool { return t == ThemeLight || t == ThemeDark }

// Profile là thông tin cá nhân người dùng tự chỉnh sửa.
type Profile struct {
	FullName     string
	JobTitle     *string
	Organization *string
	Phone        *string
	Bio          *string
	Locale       *Locale // nil = theo trình duyệt
	Theme        *Theme  // nil = theo hệ điều hành
}

type User struct {
	ID           string
	Email        string
	Role         SystemRole
	Status       UserStatus
	PasswordHash *string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Profile

	// Chỉ có giá trị khi liệt kê ở trang admin
	ProjectCount int
	RunCount     int
}

func (u User) IsActive() bool { return u.Status == UserActive }

func (u User) Actor() Actor {
	return Actor{ID: u.ID, Email: u.Email, Name: u.FullName, Role: u.Role, Locale: u.PreferredLocale()}
}

// PreferredLocale: ngôn ngữ dùng cho email gửi tới user này.
func (u User) PreferredLocale() Locale {
	if u.Locale != nil {
		return *u.Locale
	}
	return DefaultLocale
}

// Actor là người đang thực hiện thao tác (user đã đăng nhập).
type Actor struct {
	ID     string
	Email  string
	Name   string
	Role   SystemRole
	Locale Locale
}

func (a Actor) IsAdmin() bool { return a.Role == SystemRoleAdmin }

func (a Actor) DisplayName() string {
	if a.Name != "" {
		return a.Name
	}
	return a.Email
}

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func ValidateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 255 {
		return Validation("invalid_email", "Invalid email")
	}
	return nil
}

const (
	MinPasswordLen = 8
	MaxPasswordLen = 72 // giới hạn của bcrypt (byte)
)

func ValidatePassword(pw string) error {
	if utf8.RuneCountInString(pw) < MinPasswordLen || len(pw) > MaxPasswordLen {
		return Validation("password_length", fmt.Sprintf("password must be %d-%d characters", MinPasswordLen, MaxPasswordLen)).
			With(map[string]any{"min": MinPasswordLen, "max": MaxPasswordLen})
	}
	return nil
}

type UserFilter struct {
	Query  string
	Role   string
	Status string
	Limit  int
	Offset int
}

type UserPatch struct {
	FullName *string
	Role     *SystemRole
	Status   *UserStatus
}

// ProfilePatch: field nil = giữ nguyên. Với các field tuỳ chọn, chuỗi rỗng = xoá.
type ProfilePatch struct {
	FullName     *string
	JobTitle     *string
	Organization *string
	Phone        *string
	Bio          *string
	Locale       OptionalString
	Theme        OptionalString
}

var phonePattern = regexp.MustCompile(`^\+?[0-9 ().-]{6,30}$`)

// Normalize cắt khoảng trắng và kiểm tra độ dài / định dạng từng field.
func (p *ProfilePatch) Normalize() error {
	limits := []struct {
		field *string
		max   int
		code  string
	}{
		{p.FullName, 255, "invalid_full_name"},
		{p.JobTitle, 100, "invalid_job_title"},
		{p.Organization, 150, "invalid_organization"},
		{p.Bio, 1000, "invalid_bio"},
	}
	for _, l := range limits {
		if l.field == nil {
			continue
		}
		*l.field = strings.TrimSpace(*l.field)
		if utf8.RuneCountInString(*l.field) > l.max {
			return Validation(l.code, fmt.Sprintf("must be at most %d characters", l.max)).With(map[string]any{"max": l.max})
		}
	}
	if p.Phone != nil {
		*p.Phone = strings.TrimSpace(*p.Phone)
		if *p.Phone != "" && !phonePattern.MatchString(*p.Phone) {
			return Validation("invalid_phone", "Invalid phone number")
		}
	}
	if p.Locale.Set && p.Locale.Value != nil && !Locale(*p.Locale.Value).Valid() {
		return Validation("invalid_locale", "Language must be vi, en or ja")
	}
	if p.Theme.Set && p.Theme.Value != nil && !Theme(*p.Theme.Value).Valid() {
		return Validation("invalid_theme", "Theme must be light or dark")
	}
	return nil
}
