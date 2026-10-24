package postgres

import (
	"context"
	"strings"
	"time"

	"reqsentinel/api/internal/domain"
)

type UserRepo struct{ db *DB }

func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

const userColumns = `id, email, full_name, role, status, password_hash, last_login_at, created_at, updated_at,
	job_title, organization, phone, bio, locale, theme`

func scanUser(row scanner) (domain.User, error) {
	var u domain.User
	var role, status string
	var locale, theme *string
	err := row.Scan(&u.ID, &u.Email, &u.FullName, &role, &status, &u.PasswordHash, &u.LastLoginAt, &u.CreatedAt,
		&u.UpdatedAt, &u.JobTitle, &u.Organization, &u.Phone, &u.Bio, &locale, &theme)
	u.Role, u.Status = domain.SystemRole(role), domain.UserStatus(status)
	if locale != nil {
		l := domain.Locale(*locale)
		u.Locale = &l
	}
	if theme != nil {
		t := domain.Theme(*theme)
		u.Theme = &t
	}
	return u, err
}

func (r *UserRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {
	created, err := scanUser(r.db.q(ctx).QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING `+userColumns,
		domain.NormalizeEmail(u.Email), u.PasswordHash, strings.TrimSpace(u.FullName), string(u.Role)))
	if isUniqueViolation(err) {
		return created, domain.Conflict("email_taken", "Email already registered")
	}
	return created, err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (domain.User, error) {
	u, err := scanUser(r.db.q(ctx).QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	return u, notFound(err)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := scanUser(r.db.q(ctx).QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE lower(email) = $1`, domain.NormalizeEmail(email)))
	return u, notFound(err)
}

func (r *UserRepo) List(ctx context.Context, f domain.UserFilter) ([]domain.User, int, error) {
	const where = `
		WHERE ($1::text = '' OR u.email ILIKE '%' || $1 || '%' OR u.full_name ILIKE '%' || $1 || '%')
		  AND ($2::text = '' OR u.role::text = $2)
		  AND ($3::text = '' OR u.status::text = $3)`
	var total int
	if err := r.db.q(ctx).QueryRow(ctx, `SELECT count(*) FROM users u`+where, f.Query, f.Role, f.Status).
		Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT u.id, u.email, u.full_name, u.role, u.status, u.password_hash, u.last_login_at, u.created_at, u.updated_at,
		       u.job_title, u.organization, u.phone, u.bio, u.locale, u.theme,
		       (SELECT count(*) FROM project_members m WHERE m.user_id = u.id),
		       (SELECT count(*) FROM pipeline_runs pr WHERE pr.created_by = u.id)
		FROM users u`+where+`
		ORDER BY u.created_at DESC
		LIMIT $4 OFFSET $5`, f.Query, f.Role, f.Status, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	users, err := collect(rows, func(s scanner) (domain.User, error) {
		var counts struct{ projects, runs int }
		u, err := scanUser(scanTail{s, []any{&counts.projects, &counts.runs}})
		u.ProjectCount, u.RunCount = counts.projects, counts.runs
		return u, err
	})
	return users, total, err
}

func (r *UserRepo) Update(ctx context.Context, id string, p domain.UserPatch) (domain.User, error) {
	var fullName, role, status *string
	if p.FullName != nil {
		v := strings.TrimSpace(*p.FullName)
		fullName = &v
	}
	if p.Role != nil {
		v := string(*p.Role)
		role = &v
	}
	if p.Status != nil {
		v := string(*p.Status)
		status = &v
	}
	u, err := scanUser(r.db.q(ctx).QueryRow(ctx, `
		UPDATE users SET
			full_name  = COALESCE($2, full_name),
			role       = COALESCE($3, role),
			status     = COALESCE($4, status),
			updated_at = now()
		WHERE id = $1
		RETURNING `+userColumns, id, fullName, role, status))
	return u, notFound(err)
}

func (r *UserRepo) UpdateProfile(ctx context.Context, id string, p domain.ProfilePatch) (domain.User, error) {
	u, err := scanUser(r.db.q(ctx).QueryRow(ctx, `
		UPDATE users SET
			full_name    = COALESCE($2, full_name),
			job_title    = CASE WHEN $3::text IS NULL THEN job_title ELSE NULLIF($3, '') END,
			organization = CASE WHEN $4::text IS NULL THEN organization ELSE NULLIF($4, '') END,
			phone        = CASE WHEN $5::text IS NULL THEN phone ELSE NULLIF($5, '') END,
			bio          = CASE WHEN $6::text IS NULL THEN bio ELSE NULLIF($6, '') END,
			locale       = CASE WHEN $7::boolean THEN $8::text::ui_locale ELSE locale END,
			theme        = CASE WHEN $9::boolean THEN $10::text::ui_theme ELSE theme END,
			updated_at   = now()
		WHERE id = $1
		RETURNING `+userColumns,
		id, p.FullName, p.JobTitle, p.Organization, p.Phone, p.Bio,
		p.Locale.Set, p.Locale.Value, p.Theme.Set, p.Theme.Value))
	return u, notFound(err)
}

func (r *UserRepo) SetPassword(ctx context.Context, id, hash string) error {
	return affected(r.db.q(ctx).Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, hash))
}

func (r *UserRepo) TouchLastLogin(ctx context.Context, id string) error {
	_, err := r.db.q(ctx).Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

func (r *UserRepo) CountActiveAdmins(ctx context.Context) (int, error) {
	var n int
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM users WHERE role = 'admin' AND status = 'active'`).Scan(&n)
	return n, err
}

// ---------- Reset mật khẩu ----------

type PasswordResetRepo struct{ db *DB }

func NewPasswordResetRepo(db *DB) *PasswordResetRepo { return &PasswordResetRepo{db: db} }

func (r *PasswordResetRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`INSERT INTO password_resets (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`, tokenHash, userID, expiresAt)
	return err
}

func (r *PasswordResetRepo) Consume(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := r.db.q(ctx).QueryRow(ctx, `
		UPDATE password_resets SET used_at = $2
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
		RETURNING user_id`, tokenHash, now).Scan(&userID)
	if err != nil {
		return "", notFound(err)
	}
	// Token khác của cùng user hết hiệu lực luôn
	_, err = r.db.q(ctx).Exec(ctx,
		`UPDATE password_resets SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`, userID, now)
	return userID, err
}
