package postgres

import (
	"context"
	"strings"
	"time"

	"reqsentinel/api/internal/domain"
)

type ProjectRepo struct{ db *DB }

func NewProjectRepo(db *DB) *ProjectRepo { return &ProjectRepo{db: db} }

const projectSelect = `
	SELECT p.id, p.name, p.description, p.owner_id, u.email, p.created_at,
	       (SELECT count(*) FROM user_stories s WHERE s.project_id = p.id),
	       (SELECT count(*) FROM permission_matrix_rows r WHERE r.project_id = p.id),
	       (SELECT count(*) FROM pipeline_runs pr WHERE pr.project_id = p.id),
	       (SELECT count(*) FROM conflict_issues ci WHERE ci.project_id = p.id AND ci.status IN ('open', 'acknowledged')),
	       m.role
	FROM projects p
	LEFT JOIN users u ON u.id = p.owner_id`

func scanProject(s scanner) (domain.Project, error) {
	var p domain.Project
	var role *string
	err := s.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.OwnerEmail, &p.CreatedAt,
		&p.UserStoryCount, &p.PMRowCount, &p.RunCount, &p.OpenIssueCount, &role)
	if role != nil {
		p.MyRole = domain.ProjectRole(*role)
	}
	return p, err
}

func (r *ProjectRepo) List(ctx context.Context, memberID *string) ([]domain.Project, error) {
	rows, err := r.db.q(ctx).Query(ctx, projectSelect+`
		LEFT JOIN project_members m ON m.project_id = p.id AND m.user_id = $1
		WHERE ($1::uuid IS NULL OR m.user_id IS NOT NULL)
		ORDER BY p.created_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanProject)
}

func (r *ProjectRepo) Get(ctx context.Context, id string) (domain.Project, error) {
	p, err := scanProject(r.db.q(ctx).QueryRow(ctx, projectSelect+`
		LEFT JOIN project_members m ON false
		WHERE p.id = $1`, id))
	return p, notFound(err)
}

func (r *ProjectRepo) Create(ctx context.Context, ownerID, name string, description *string) (domain.Project, error) {
	var id string
	if err := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO projects (name, description, owner_id) VALUES ($1, $2, $3) RETURNING id`,
		strings.TrimSpace(name), description, ownerID).Scan(&id); err != nil {
		return domain.Project{}, err
	}
	return r.Get(ctx, id)
}

func (r *ProjectRepo) Update(ctx context.Context, id string, name, description *string) error {
	return affected(r.db.q(ctx).Exec(ctx, `
		UPDATE projects SET
			name        = COALESCE($2, name),
			description = COALESCE($3, description),
			updated_at  = now()
		WHERE id = $1`, id, name, description))
}

// Delete xoá project và dữ liệu con. Các bảng mới có ON DELETE CASCADE;
// các bảng từ schema cũ (không cascade) được xoá tường minh.
func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		stmts := []string{
			`DELETE FROM conflict_records WHERE pipeline_run_id IN (SELECT id FROM pipeline_runs WHERE project_id = $1)`,
			`DELETE FROM extracted_entities WHERE pipeline_run_id IN (SELECT id FROM pipeline_runs WHERE project_id = $1)`,
			`DELETE FROM pipeline_runs WHERE project_id = $1`,
			`DELETE FROM conflict_issues WHERE project_id = $1`,
			`DELETE FROM acceptance_criteria WHERE user_story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`,
			`DELETE FROM user_stories WHERE project_id = $1`,
			`DELETE FROM permission_matrix_rows WHERE project_id = $1`,
		}
		for _, s := range stmts {
			if _, err := r.db.q(ctx).Exec(ctx, s, id); err != nil {
				return err
			}
		}
		return affected(r.db.q(ctx).Exec(ctx, `DELETE FROM projects WHERE id = $1`, id))
	})
}

func (r *ProjectRepo) AssignOrphansTo(ctx context.Context, userID string) (int64, error) {
	var n int64
	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		tag, err := r.db.q(ctx).Exec(ctx, `UPDATE projects SET owner_id = $1 WHERE owner_id IS NULL`, userID)
		if err != nil {
			return err
		}
		n = tag.RowsAffected()
		_, err = r.db.q(ctx).Exec(ctx, `
			INSERT INTO project_members (project_id, user_id, role)
			SELECT id, $1, 'owner' FROM projects WHERE owner_id = $1
			ON CONFLICT DO NOTHING`, userID)
		return err
	})
	return n, err
}

// ---------- Thành viên ----------

type MemberRepo struct{ db *DB }

func NewMemberRepo(db *DB) *MemberRepo { return &MemberRepo{db: db} }

const memberSelect = `
	SELECT m.project_id, m.user_id, u.email, u.full_name, m.role, m.created_at
	FROM project_members m JOIN users u ON u.id = m.user_id`

func scanMember(s scanner) (domain.Member, error) {
	var m domain.Member
	var role string
	err := s.Scan(&m.ProjectID, &m.UserID, &m.Email, &m.FullName, &role, &m.AddedAt)
	m.Role = domain.ProjectRole(role)
	return m, err
}

func (r *MemberRepo) Get(ctx context.Context, projectID, userID string) (*domain.Member, error) {
	m, err := scanMember(r.db.q(ctx).QueryRow(ctx, memberSelect+` WHERE m.project_id = $1 AND m.user_id = $2`, projectID, userID))
	if err != nil {
		if domain.IsNotFound(notFound(err)) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *MemberRepo) List(ctx context.Context, projectID string) ([]domain.Member, error) {
	rows, err := r.db.q(ctx).Query(ctx, memberSelect+` WHERE m.project_id = $1
		ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'editor' THEN 1 ELSE 2 END, u.email`, projectID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanMember)
}

func (r *MemberRepo) Add(ctx context.Context, m domain.Member) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, $3)`, m.ProjectID, m.UserID, string(m.Role))
	if isUniqueViolation(err) {
		return domain.Conflict("already_member", "User is already a member of this project")
	}
	return err
}

func (r *MemberRepo) UpdateRole(ctx context.Context, projectID, userID string, role domain.ProjectRole) error {
	return affected(r.db.q(ctx).Exec(ctx,
		`UPDATE project_members SET role = $3 WHERE project_id = $1 AND user_id = $2`, projectID, userID, string(role)))
}

func (r *MemberRepo) Remove(ctx context.Context, projectID, userID string) error {
	return affected(r.db.q(ctx).Exec(ctx,
		`DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, projectID, userID))
}

func (r *MemberRepo) CountOwners(ctx context.Context, projectID string) (int, error) {
	var n int
	// Khoá các dòng owner để 2 thao tác đồng thời không cùng xoá owner cuối
	err := r.db.q(ctx).QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT 1 FROM project_members WHERE project_id = $1 AND role = 'owner' FOR UPDATE
		) owners`, projectID).Scan(&n)
	return n, err
}

// ---------- Lời mời ----------

type InvitationRepo struct{ db *DB }

func NewInvitationRepo(db *DB) *InvitationRepo { return &InvitationRepo{db: db} }

const invitationSelect = `
	SELECT i.id, i.project_id, p.name, i.email, i.role, i.status, i.invited_by,
	       COALESCE(NULLIF(u.full_name, ''), u.email), i.expires_at, i.created_at, i.responded_at
	FROM project_invitations i
	JOIN projects p ON p.id = i.project_id
	JOIN users u ON u.id = i.invited_by`

func scanInvitation(s scanner) (domain.Invitation, error) {
	var inv domain.Invitation
	var role, status string
	err := s.Scan(&inv.ID, &inv.ProjectID, &inv.ProjectName, &inv.Email, &role, &status, &inv.InvitedBy,
		&inv.InviterName, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt)
	inv.Role, inv.Status = domain.ProjectRole(role), domain.InvitationStatus(status)
	return inv, err
}

func (r *InvitationRepo) Create(ctx context.Context, inv domain.Invitation, tokenHash string) (domain.Invitation, error) {
	var id string
	if err := r.db.q(ctx).QueryRow(ctx, `
		INSERT INTO project_invitations (project_id, email, role, token_hash, status, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		inv.ProjectID, inv.Email, string(inv.Role), tokenHash, string(inv.Status), inv.InvitedBy, inv.ExpiresAt).
		Scan(&id); err != nil {
		return inv, err
	}
	return r.Get(ctx, id)
}

func (r *InvitationRepo) Get(ctx context.Context, id string) (domain.Invitation, error) {
	inv, err := scanInvitation(r.db.q(ctx).QueryRow(ctx, invitationSelect+` WHERE i.id = $1`, id))
	return inv, notFound(err)
}

func (r *InvitationRepo) GetByTokenHash(ctx context.Context, hash string) (domain.Invitation, error) {
	inv, err := scanInvitation(r.db.q(ctx).QueryRow(ctx, invitationSelect+` WHERE i.token_hash = $1`, hash))
	return inv, notFound(err)
}

func (r *InvitationRepo) ListPending(ctx context.Context, projectID string) ([]domain.Invitation, error) {
	rows, err := r.db.q(ctx).Query(ctx, invitationSelect+`
		WHERE i.project_id = $1 AND i.status = 'pending' ORDER BY i.created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanInvitation)
}

func (r *InvitationRepo) FindPending(ctx context.Context, projectID, email string) (*domain.Invitation, error) {
	inv, err := scanInvitation(r.db.q(ctx).QueryRow(ctx, invitationSelect+`
		WHERE i.project_id = $1 AND lower(i.email) = lower($2) AND i.status = 'pending'
		ORDER BY i.created_at DESC LIMIT 1`, projectID, email))
	if err != nil {
		if domain.IsNotFound(notFound(err)) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepo) SetStatus(ctx context.Context, id string, status domain.InvitationStatus, at time.Time) error {
	return affected(r.db.q(ctx).Exec(ctx,
		`UPDATE project_invitations SET status = $2, responded_at = $3 WHERE id = $1`, id, string(status), at))
}
