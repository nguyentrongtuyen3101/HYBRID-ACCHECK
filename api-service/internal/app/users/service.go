// Package users: quản trị tài khoản (chỉ admin) và tạo admin đầu tiên.
package users

import (
	"context"
	"log/slog"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

type Deps struct {
	Users    app.UserRepository
	Projects app.ProjectRepository
	Hasher   app.PasswordHasher
	Audit    app.AuditLog
}

type Service struct{ d Deps }

func New(d Deps) *Service { return &Service{d: d} }

func (s *Service) List(ctx context.Context, f domain.UserFilter) ([]domain.User, int, error) {
	return s.d.Users.List(ctx, f)
}

type CreateInput struct {
	Email, Password, FullName string
	Role                      domain.SystemRole
}

func (s *Service) Create(ctx context.Context, actor domain.Actor, in CreateInput, meta app.RequestMeta) (domain.User, error) {
	email := domain.NormalizeEmail(in.Email)
	if in.Role == "" {
		in.Role = domain.SystemRoleUser
	}
	if err := domain.ValidateEmail(email); err != nil {
		return domain.User{}, err
	}
	if !in.Role.Valid() {
		return domain.User{}, domain.Validation("invalid_system_role", "Role must be admin or user")
	}
	if err := domain.ValidatePassword(in.Password); err != nil {
		return domain.User{}, err
	}
	hash, err := s.d.Hasher.Hash(in.Password)
	if err != nil {
		return domain.User{}, err
	}
	u, err := s.d.Users.Create(ctx, domain.User{Email: email, Profile: domain.Profile{FullName: in.FullName}, Role: in.Role, PasswordHash: &hash})
	if err != nil {
		return u, err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "user.create", "user", u.ID,
		map[string]any{"email": u.Email, "role": u.Role}))
	return u, nil
}

func (s *Service) Update(ctx context.Context, actor domain.Actor, id string, p domain.UserPatch, meta app.RequestMeta) (domain.User, error) {
	if p.Role != nil && !p.Role.Valid() {
		return domain.User{}, domain.Validation("invalid_system_role", "Role must be admin or user")
	}
	if p.Status != nil && !p.Status.Valid() {
		return domain.User{}, domain.Validation("invalid_user_status", "Status must be active or disabled")
	}
	target, err := s.d.Users.GetByID(ctx, id)
	if err != nil {
		return target, app.NotFoundAs(err, "user_not_found", "User not found")
	}
	demoting := p.Role != nil && *p.Role != domain.SystemRoleAdmin
	disabling := p.Status != nil && *p.Status != domain.UserActive
	if id == actor.ID && (demoting || disabling) {
		return target, domain.Validation("cannot_change_self", "You cannot demote or disable your own account")
	}
	if target.Role == domain.SystemRoleAdmin && target.IsActive() && (demoting || disabling) {
		n, err := s.d.Users.CountActiveAdmins(ctx)
		if err != nil {
			return target, err
		}
		if n <= 1 {
			return target, domain.Validation("last_admin", "Cannot remove the last active admin")
		}
	}
	u, err := s.d.Users.Update(ctx, id, p)
	if err != nil {
		return u, app.NotFoundAs(err, "user_not_found", "User not found")
	}
	changes := map[string]any{}
	if u.Role != target.Role {
		changes["role"] = map[string]any{"from": target.Role, "to": u.Role}
	}
	if u.Status != target.Status {
		changes["status"] = map[string]any{"from": target.Status, "to": u.Status}
	}
	if u.FullName != target.FullName {
		changes["full_name"] = u.FullName
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "user.update", "user", u.ID, changes))
	return u, nil
}

func (s *Service) ResetPassword(ctx context.Context, actor domain.Actor, id, password string, meta app.RequestMeta) error {
	if err := domain.ValidatePassword(password); err != nil {
		return err
	}
	hash, err := s.d.Hasher.Hash(password)
	if err != nil {
		return err
	}
	if err := s.d.Users.SetPassword(ctx, id, hash); err != nil {
		return app.NotFoundAs(err, "user_not_found", "User not found")
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "user.reset_password", "user", id, nil))
	return nil
}

// BootstrapAdmin tạo admin đầu tiên nếu chưa có, rồi gán project cũ (chưa có chủ) cho admin.
func (s *Service) BootstrapAdmin(ctx context.Context, email, password, name string, log *slog.Logger) error {
	n, err := s.d.Users.CountActiveAdmins(ctx)
	if err != nil || n > 0 {
		return err
	}
	if email == "" || password == "" {
		log.Warn("no admin account exists; set ADMIN_EMAIL and ADMIN_PASSWORD to create one on startup")
		return nil
	}
	if err := domain.ValidatePassword(password); err != nil {
		return err
	}
	u, err := s.d.Users.GetByEmail(ctx, email)
	switch {
	case domain.IsNotFound(err):
		hash, err := s.d.Hasher.Hash(password)
		if err != nil {
			return err
		}
		if u, err = s.d.Users.Create(ctx, domain.User{Email: domain.NormalizeEmail(email), Profile: domain.Profile{FullName: name},
			Role: domain.SystemRoleAdmin, PasswordHash: &hash}); err != nil {
			return err
		}
		log.Info("created bootstrap admin", "email", u.Email)
	case err != nil:
		return err
	default:
		role, status := domain.SystemRoleAdmin, domain.UserActive
		if u, err = s.d.Users.Update(ctx, u.ID, domain.UserPatch{Role: &role, Status: &status}); err != nil {
			return err
		}
		log.Info("promoted existing user to admin", "email", u.Email)
	}
	assigned, err := s.d.Projects.AssignOrphansTo(ctx, u.ID)
	if err != nil {
		return err
	}
	if assigned > 0 {
		log.Info("assigned legacy projects to admin", "count", assigned)
	}
	return nil
}
