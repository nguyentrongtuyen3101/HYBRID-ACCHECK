// Package auth: đăng ký, đăng nhập, xác thực token, đổi / quên mật khẩu.
package auth

import (
	"context"
	"time"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

const passwordResetTTL = 30 * time.Minute

type Deps struct {
	Users       app.UserRepository
	Resets      app.PasswordResetRepository
	Hasher      app.PasswordHasher
	Tokens      app.TokenIssuer
	Opaque      app.OpaqueTokens
	Limiter     app.LoginLimiter
	Events      app.EventPublisher
	Tx          app.TxManager
	Audit       app.AuditLog
	Clock       app.Clock
	NewID       app.IDGenerator
	AllowSignup bool
	WebURL      string // link trong email reset mật khẩu
}

type Service struct{ d Deps }

func New(d Deps) *Service { return &Service{d: d} }

type Session struct {
	Token     string
	ExpiresAt time.Time
	User      domain.User
}

type RegisterInput struct {
	Email, Password, FullName string
}

func (s *Service) Register(ctx context.Context, in RegisterInput, meta app.RequestMeta) (Session, error) {
	if !s.d.AllowSignup {
		return Session{}, domain.Forbidden("signup_disabled", "Self sign-up is disabled. Contact an administrator.")
	}
	email := domain.NormalizeEmail(in.Email)
	if err := domain.ValidateEmail(email); err != nil {
		return Session{}, err
	}
	if err := domain.ValidatePassword(in.Password); err != nil {
		return Session{}, err
	}
	hash, err := s.d.Hasher.Hash(in.Password)
	if err != nil {
		return Session{}, err
	}
	u, err := s.d.Users.Create(ctx, domain.User{Email: email, Profile: domain.Profile{FullName: in.FullName}, Role: domain.SystemRoleUser, PasswordHash: &hash})
	if err != nil {
		return Session{}, err
	}
	actor := u.Actor()
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "auth.register", "user", u.ID, nil))
	return s.session(u)
}

func (s *Service) Login(ctx context.Context, email, password string, meta app.RequestMeta) (Session, error) {
	email = domain.NormalizeEmail(email)
	key := meta.IP + "|" + email
	if s.d.Limiter.Blocked(key) {
		return Session{}, domain.RateLimited("too_many_login_attempts", "Too many failed login attempts. Try again later.")
	}
	u, err := s.d.Users.GetByEmail(ctx, email)
	if err != nil && !domain.IsNotFound(err) {
		return Session{}, err
	}
	var hash *string
	if err == nil {
		hash = u.PasswordHash
	}
	// Luôn so sánh hash (kể cả khi email không tồn tại) để thời gian phản hồi như nhau
	if !s.d.Hasher.Compare(hash, password) {
		s.d.Limiter.Fail(key)
		s.d.Audit.Record(ctx, app.Audit(nil, meta.IP, "auth.login_failed", "user", "", map[string]any{"email": email}))
		return Session{}, domain.Unauthorized("invalid_credentials", "Incorrect email or password")
	}
	if !u.IsActive() {
		return Session{}, domain.Forbidden("account_disabled", "Account is disabled")
	}
	s.d.Limiter.Reset(key)
	if err := s.d.Users.TouchLastLogin(ctx, u.ID); err != nil {
		return Session{}, err
	}
	now := s.d.Clock()
	u.LastLoginAt = &now
	actor := u.Actor()
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "auth.login", "user", u.ID, nil))
	return s.session(u)
}

func (s *Service) session(u domain.User) (Session, error) {
	token, exp, err := s.d.Tokens.Issue(u.ID, u.Role)
	if err != nil {
		return Session{}, err
	}
	return Session{Token: token, ExpiresAt: exp, User: u}, nil
}

// Authenticate xác thực token và đọc lại user từ DB, để khoá tài khoản /
// đổi role có hiệu lực ngay mà không cần chờ token hết hạn.
func (s *Service) Authenticate(ctx context.Context, token string) (domain.User, error) {
	userID, err := s.d.Tokens.Verify(token)
	if err != nil {
		return domain.User{}, domain.Unauthorized("invalid_token", "Invalid or expired token")
	}
	u, err := s.d.Users.GetByID(ctx, userID)
	if domain.IsNotFound(err) {
		return u, domain.Unauthorized("invalid_token", "Invalid or expired token")
	}
	if err != nil {
		return u, err
	}
	if !u.IsActive() {
		return u, domain.Forbidden("account_disabled", "Account is disabled")
	}
	return u, nil
}

func (s *Service) Me(ctx context.Context, actor domain.Actor) (domain.User, error) {
	return s.d.Users.GetByID(ctx, actor.ID)
}

func (s *Service) UpdateProfile(ctx context.Context, actor domain.Actor, p domain.ProfilePatch) (domain.User, error) {
	if err := p.Normalize(); err != nil {
		return domain.User{}, err
	}
	return s.d.Users.UpdateProfile(ctx, actor.ID, p)
}

func (s *Service) ChangePassword(ctx context.Context, actor domain.Actor, current, next string, meta app.RequestMeta) error {
	u, err := s.d.Users.GetByID(ctx, actor.ID)
	if err != nil {
		return err
	}
	if !s.d.Hasher.Compare(u.PasswordHash, current) {
		return domain.Validation("wrong_current_password", "Current password is incorrect")
	}
	if err := domain.ValidatePassword(next); err != nil {
		return err
	}
	if next == current {
		return domain.Validation("password_unchanged", "New password must differ from the current one")
	}
	hash, err := s.d.Hasher.Hash(next)
	if err != nil {
		return err
	}
	if err := s.d.Users.SetPassword(ctx, u.ID, hash); err != nil {
		return err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "auth.change_password", "user", u.ID, nil))
	return nil
}

// ForgotPassword luôn trả nil để không lộ email nào đã đăng ký.
func (s *Service) ForgotPassword(ctx context.Context, email string, meta app.RequestMeta) error {
	email = domain.NormalizeEmail(email)
	u, err := s.d.Users.GetByEmail(ctx, email)
	if domain.IsNotFound(err) || (err == nil && !u.IsActive()) {
		return nil
	}
	if err != nil {
		return err
	}
	raw, hash := s.d.Opaque.New()
	expires := s.d.Clock().Add(passwordResetTTL)
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.d.Resets.Create(ctx, u.ID, hash, expires); err != nil {
			return err
		}
		return s.d.Events.Publish(ctx, app.NewEvent(s.d.NewID, s.d.Clock, domain.EventPasswordResetRequested, nil, nil,
			map[string]any{
				"email":      u.Email,
				"name":       u.FullName,
				"reset_url":  s.d.WebURL + "/reset-password?token=" + raw,
				"locale":     u.PreferredLocale(),
				"expires_at": expires,
			}))
	})
	if err != nil {
		return err
	}
	s.d.Audit.Record(ctx, app.Audit(nil, meta.IP, "auth.password_reset_requested", "user", u.ID, nil))
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string, meta app.RequestMeta) error {
	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := s.d.Hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	var userID string
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		id, err := s.d.Resets.Consume(ctx, s.d.Opaque.Hash(token), s.d.Clock())
		if err != nil {
			return domain.Validation("reset_link_invalid", "Reset link is invalid or has expired")
		}
		userID = id
		return s.d.Users.SetPassword(ctx, id, hash)
	})
	if err != nil {
		return err
	}
	s.d.Audit.Record(ctx, app.Audit(nil, meta.IP, "auth.password_reset", "user", userID, nil))
	return nil
}
