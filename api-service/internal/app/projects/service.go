// Package projects: project, thành viên và lời mời.
package projects

import (
	"context"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

type Deps struct {
	Projects    app.ProjectRepository
	Members     app.MemberRepository
	Invitations app.InvitationRepository
	Users       app.UserRepository
	Stories     app.StoryRepository
	PM          app.PMRepository
	Auth        app.Authorizer
	Opaque      app.OpaqueTokens
	Events      app.EventPublisher
	Tx          app.TxManager
	Audit       app.AuditLog
	Clock       app.Clock
	NewID       app.IDGenerator
	WebURL      string // link chấp nhận lời mời trong email
}

type Service struct{ d Deps }

func New(d Deps) *Service { return &Service{d: d} }

// ---------- Project ----------

func (s *Service) List(ctx context.Context, actor domain.Actor, onlyMine bool) ([]domain.Project, error) {
	var member *string
	if !actor.IsAdmin() || onlyMine {
		member = &actor.ID
	}
	// Repository điền MyRole khi lọc theo thành viên; admin luôn là owner
	items, err := s.d.Projects.List(ctx, member)
	if err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		for i := range items {
			items[i].MyRole = domain.ProjectOwner
		}
	}
	return items, nil
}

type Detail struct {
	Project domain.Project
	Stories []domain.UserStory
	PM      []domain.PMRow
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id string) (Detail, error) {
	p, _, err := s.d.Auth.Require(ctx, actor, id, domain.PermView)
	if err != nil {
		return Detail{}, err
	}
	stories, err := s.d.Stories.List(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	rows, err := s.d.PM.List(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	return Detail{Project: p, Stories: stories, PM: rows}, nil
}

func (s *Service) Create(ctx context.Context, actor domain.Actor, name string, description *string, meta app.RequestMeta) (domain.Project, error) {
	name, err := domain.ValidateProjectName(name)
	if err != nil {
		return domain.Project{}, err
	}
	var p domain.Project
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if p, err = s.d.Projects.Create(ctx, actor.ID, name, description); err != nil {
			return err
		}
		return s.d.Members.Add(ctx, domain.Member{ProjectID: p.ID, UserID: actor.ID, Role: domain.ProjectOwner})
	})
	if err != nil {
		return p, err
	}
	p.MyRole = domain.ProjectOwner
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.create", "project", p.ID, map[string]any{"name": p.Name}))
	return p, nil
}

func (s *Service) Update(ctx context.Context, actor domain.Actor, id string, name, description *string, meta app.RequestMeta) (domain.Project, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, id, domain.PermManageProject); err != nil {
		return domain.Project{}, err
	}
	if name != nil {
		n, err := domain.ValidateProjectName(*name)
		if err != nil {
			return domain.Project{}, err
		}
		name = &n
	}
	if err := s.d.Projects.Update(ctx, id, name, description); err != nil {
		return domain.Project{}, err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.update", "project", id, nil))
	p, _, err := s.d.Auth.Require(ctx, actor, id, domain.PermView)
	return p, err
}

func (s *Service) Delete(ctx context.Context, actor domain.Actor, id string, meta app.RequestMeta) error {
	p, _, err := s.d.Auth.Require(ctx, actor, id, domain.PermManageProject)
	if err != nil {
		return err
	}
	if err := s.d.Projects.Delete(ctx, id); err != nil {
		return err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.delete", "project", id, map[string]any{"name": p.Name}))
	return nil
}

// ---------- Thành viên ----------

func (s *Service) Members(ctx context.Context, actor domain.Actor, projectID string) ([]domain.Member, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermView); err != nil {
		return nil, err
	}
	return s.d.Members.List(ctx, projectID)
}

func (s *Service) ChangeMemberRole(ctx context.Context, actor domain.Actor, projectID, userID string, role domain.ProjectRole, meta app.RequestMeta) error {
	if err := domain.ValidateInviteRole(role); err != nil {
		return err
	}
	p, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermManageMembers)
	if err != nil {
		return err
	}
	return s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		target, err := s.member(ctx, projectID, userID)
		if err != nil {
			return err
		}
		owners, err := s.d.Members.CountOwners(ctx, projectID)
		if err != nil {
			return err
		}
		if err := domain.EnsureOwnerRemains(target, &role, owners); err != nil {
			return err
		}
		if err := s.d.Members.UpdateRole(ctx, projectID, userID, role); err != nil {
			return err
		}
		s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.member_role_changed", "project", projectID,
			map[string]any{"user_id": userID, "from": target.Role, "to": role}))
		return s.d.Events.Publish(ctx, app.NewEvent(s.d.NewID, s.d.Clock, domain.EventMemberRoleChanged, &actor,
			domain.Recipients(actor.ID, &userID), map[string]any{
				"project_id": projectID, "project_name": p.Name, "role": role,
			}))
	})
}

// RemoveMember: owner xoá thành viên, hoặc thành viên tự rời project.
func (s *Service) RemoveMember(ctx context.Context, actor domain.Actor, projectID, userID string, meta app.RequestMeta) error {
	perm := domain.PermManageMembers
	if userID == actor.ID {
		perm = domain.PermView
	}
	p, _, err := s.d.Auth.Require(ctx, actor, projectID, perm)
	if err != nil {
		return err
	}
	return s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		target, err := s.member(ctx, projectID, userID)
		if err != nil {
			return err
		}
		owners, err := s.d.Members.CountOwners(ctx, projectID)
		if err != nil {
			return err
		}
		if err := domain.EnsureOwnerRemains(target, nil, owners); err != nil {
			return err
		}
		if err := s.d.Members.Remove(ctx, projectID, userID); err != nil {
			return err
		}
		s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.member_removed", "project", projectID,
			map[string]any{"user_id": userID}))
		if userID == actor.ID {
			return nil
		}
		return s.d.Events.Publish(ctx, app.NewEvent(s.d.NewID, s.d.Clock, domain.EventMemberRemoved, &actor,
			domain.Recipients(actor.ID, &userID), map[string]any{"project_id": projectID, "project_name": p.Name}))
	})
}

func (s *Service) member(ctx context.Context, projectID, userID string) (domain.Member, error) {
	m, err := s.d.Members.Get(ctx, projectID, userID)
	if err != nil {
		return domain.Member{}, err
	}
	if m == nil {
		return domain.Member{}, domain.NotFound("member_not_found", "Member not found")
	}
	return *m, nil
}

// ---------- Lời mời ----------

func (s *Service) Invite(ctx context.Context, actor domain.Actor, projectID, email string, role domain.ProjectRole, meta app.RequestMeta) (domain.Invitation, error) {
	email = domain.NormalizeEmail(email)
	if err := domain.ValidateEmail(email); err != nil {
		return domain.Invitation{}, err
	}
	if err := domain.ValidateInviteRole(role); err != nil {
		return domain.Invitation{}, err
	}
	p, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermManageMembers)
	if err != nil {
		return domain.Invitation{}, err
	}

	// Người được mời đã có tài khoản thì gửi thêm thông báo in-app
	var inviteeID *string
	locale := actor.Locale // người nhận chưa có tài khoản → email theo ngôn ngữ của người mời
	if u, err := s.d.Users.GetByEmail(ctx, email); err == nil {
		inviteeID, locale = &u.ID, u.PreferredLocale()
		if m, err := s.d.Members.Get(ctx, projectID, u.ID); err != nil {
			return domain.Invitation{}, err
		} else if m != nil {
			return domain.Invitation{}, domain.Conflict("already_member", "This user is already a member of the project")
		}
	} else if !domain.IsNotFound(err) {
		return domain.Invitation{}, err
	}

	var inv domain.Invitation
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		// Mời lại cùng email → thu hồi lời mời cũ, tạo lời mời mới (link mới)
		if old, err := s.d.Invitations.FindPending(ctx, projectID, email); err != nil {
			return err
		} else if old != nil {
			if err := s.d.Invitations.SetStatus(ctx, old.ID, domain.InvitationRevoked, s.d.Clock()); err != nil {
				return err
			}
		}
		raw, hash := s.d.Opaque.New()
		now := s.d.Clock()
		inv, err = s.d.Invitations.Create(ctx, domain.Invitation{
			ProjectID: projectID, Email: email, Role: role, Status: domain.InvitationPending,
			InvitedBy: actor.ID, ExpiresAt: now.Add(domain.InvitationTTL),
		}, hash)
		if err != nil {
			return err
		}
		inv.ProjectName, inv.InviterName = p.Name, actor.DisplayName()
		return s.d.Events.Publish(ctx, app.NewEvent(s.d.NewID, s.d.Clock, domain.EventMemberInvited, &actor,
			domain.Recipients(actor.ID, inviteeID), map[string]any{
				"invitation_id": inv.ID,
				"project_id":    projectID,
				"project_name":  p.Name,
				"email":         email,
				"role":          role,
				"inviter_name":  actor.DisplayName(),
				"accept_url":    s.d.WebURL + "/invitations/" + raw,
				"locale":        locale,
				"expires_at":    inv.ExpiresAt,
			}))
	})
	if err != nil {
		return inv, err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.member_invited", "project", projectID,
		map[string]any{"email": email, "role": role}))
	return inv, nil
}

func (s *Service) PendingInvitations(ctx context.Context, actor domain.Actor, projectID string) ([]domain.Invitation, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermManageMembers); err != nil {
		return nil, err
	}
	return s.d.Invitations.ListPending(ctx, projectID)
}

func (s *Service) RevokeInvitation(ctx context.Context, actor domain.Actor, projectID, invitationID string, meta app.RequestMeta) error {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermManageMembers); err != nil {
		return err
	}
	inv, err := s.d.Invitations.Get(ctx, invitationID)
	if err != nil || inv.ProjectID != projectID {
		return domain.NotFound("invitation_not_found", "Invitation not found")
	}
	if inv.Status != domain.InvitationPending {
		return domain.Conflict("invitation_not_pending", "Invitation is no longer pending")
	}
	if err := s.d.Invitations.SetStatus(ctx, inv.ID, domain.InvitationRevoked, s.d.Clock()); err != nil {
		return err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.invitation_revoked", "project", projectID,
		map[string]any{"email": inv.Email}))
	return nil
}

// PreviewInvitation cho trang chấp nhận lời mời (chưa cần đăng nhập).
func (s *Service) PreviewInvitation(ctx context.Context, token string) (domain.Invitation, error) {
	inv, err := s.d.Invitations.GetByTokenHash(ctx, s.d.Opaque.Hash(token))
	if err != nil {
		return inv, domain.NotFound("invitation_not_found", "Invitation not found")
	}
	if inv.Status == domain.InvitationPending && inv.Expired(s.d.Clock()) {
		return inv, domain.Conflict("invitation_expired", "This invitation has expired")
	}
	return inv, nil
}

func (s *Service) AcceptInvitation(ctx context.Context, actor domain.Actor, token string, meta app.RequestMeta) (domain.Project, error) {
	inv, err := s.d.Invitations.GetByTokenHash(ctx, s.d.Opaque.Hash(token))
	if err != nil {
		return domain.Project{}, domain.NotFound("invitation_not_found", "Invitation not found")
	}
	if err := inv.CheckAcceptable(actor.Email, s.d.Clock()); err != nil {
		return domain.Project{}, err
	}
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		existing, err := s.d.Members.Get(ctx, inv.ProjectID, actor.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			if err := s.d.Members.Add(ctx, domain.Member{ProjectID: inv.ProjectID, UserID: actor.ID,
				Role: inv.Role}); err != nil {
				return err
			}
		}
		if err := s.d.Invitations.SetStatus(ctx, inv.ID, domain.InvitationAccepted, s.d.Clock()); err != nil {
			return err
		}
		return s.d.Events.Publish(ctx, app.NewEvent(s.d.NewID, s.d.Clock, domain.EventMemberJoined, &actor,
			domain.Recipients(actor.ID, &inv.InvitedBy), map[string]any{
				"project_id": inv.ProjectID, "project_name": inv.ProjectName,
				"member_name": actor.DisplayName(), "role": inv.Role,
			}))
	})
	if err != nil {
		return domain.Project{}, err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.member_joined", "project", inv.ProjectID,
		map[string]any{"role": inv.Role}))
	p, _, err := s.d.Auth.Require(ctx, actor, inv.ProjectID, domain.PermView)
	return p, err
}
