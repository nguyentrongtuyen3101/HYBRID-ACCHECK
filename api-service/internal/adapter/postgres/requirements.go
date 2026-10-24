package postgres

import (
	"context"
	"encoding/json"

	"reqsentinel/api/internal/domain"
)

// ---------- User stories ----------

type StoryRepo struct{ db *DB }

func NewStoryRepo(db *DB) *StoryRepo { return &StoryRepo{db: db} }

func (r *StoryRepo) List(ctx context.Context, projectID string) ([]domain.UserStory, error) {
	return r.query(ctx, `WHERE s.project_id = $1`, projectID)
}

func (r *StoryRepo) Get(ctx context.Context, projectID, storyID string) (domain.UserStory, error) {
	list, err := r.query(ctx, `WHERE s.project_id = $1 AND s.id = $2`, projectID, storyID)
	if err != nil {
		return domain.UserStory{}, err
	}
	if len(list) == 0 {
		return domain.UserStory{}, domain.ErrNotFound
	}
	return list[0], nil
}

func (r *StoryRepo) query(ctx context.Context, where string, args ...any) ([]domain.UserStory, error) {
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT s.id, s.project_id, s.content, s.source, s.created_at, ac.id, ac.content
		FROM user_stories s
		LEFT JOIN acceptance_criteria ac ON ac.user_story_id = s.id
		`+where+`
		ORDER BY s.created_at, s.id, ac.created_at, ac.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.UserStory{}
	for rows.Next() {
		var us domain.UserStory
		var acID, acContent *string
		if err := rows.Scan(&us.ID, &us.ProjectID, &us.Content, &us.Source, &us.CreatedAt, &acID, &acContent); err != nil {
			return nil, err
		}
		if n := len(out); n == 0 || out[n-1].ID != us.ID {
			us.AcceptanceCriteria = []domain.AcceptanceCriterion{}
			out = append(out, us)
		}
		if acID != nil {
			last := &out[len(out)-1]
			last.AcceptanceCriteria = append(last.AcceptanceCriteria, domain.AcceptanceCriterion{ID: *acID, Content: *acContent})
		}
	}
	return out, rows.Err()
}

// clock_timestamp() (không phải now()) để giữ đúng thứ tự khi insert nhiều dòng trong 1 transaction.
func (r *StoryRepo) Add(ctx context.Context, projectID, content string, criteria []string) (string, error) {
	var id string
	if err := r.db.q(ctx).QueryRow(ctx, `
		INSERT INTO user_stories (project_id, content, source, created_at)
		VALUES ($1, $2, 'User Story', clock_timestamp()) RETURNING id`, projectID, content).Scan(&id); err != nil {
		return "", err
	}
	return id, r.insertCriteria(ctx, id, criteria)
}

func (r *StoryRepo) insertCriteria(ctx context.Context, storyID string, criteria []string) error {
	for _, c := range criteria {
		if _, err := r.db.q(ctx).Exec(ctx, `
			INSERT INTO acceptance_criteria (user_story_id, content, created_at)
			VALUES ($1, $2, clock_timestamp())`, storyID, c); err != nil {
			return err
		}
	}
	return nil
}

func (r *StoryRepo) Update(ctx context.Context, projectID, storyID string, content *string, criteria *[]string) error {
	if err := affected(r.db.q(ctx).Exec(ctx, `
		UPDATE user_stories SET content = COALESCE($3, content), updated_at = now()
		WHERE id = $1 AND project_id = $2`, storyID, projectID, content)); err != nil {
		return err
	}
	if criteria == nil {
		return nil
	}
	if _, err := r.db.q(ctx).Exec(ctx, `DELETE FROM acceptance_criteria WHERE user_story_id = $1`, storyID); err != nil {
		return err
	}
	return r.insertCriteria(ctx, storyID, *criteria)
}

func (r *StoryRepo) Delete(ctx context.Context, projectID, storyID string) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.q(ctx).Exec(ctx, `
			DELETE FROM acceptance_criteria WHERE user_story_id IN
				(SELECT id FROM user_stories WHERE id = $1 AND project_id = $2)`, storyID, projectID); err != nil {
			return err
		}
		return affected(r.db.q(ctx).Exec(ctx,
			`DELETE FROM user_stories WHERE id = $1 AND project_id = $2`, storyID, projectID))
	})
}

// ---------- Permission matrix ----------

type PMRepo struct{ db *DB }

func NewPMRepo(db *DB) *PMRepo { return &PMRepo{db: db} }

const pmColumns = `id, role, action, resource, effect, condition, scope`

func scanPM(s scanner) (domain.PMRow, error) {
	var r domain.PMRow
	var effect string
	err := s.Scan(&r.ID, &r.Role, &r.Action, &r.Resource, &effect, &r.Condition, &r.Scope)
	r.Effect = domain.Effect(effect)
	return r, err
}

func (r *PMRepo) List(ctx context.Context, projectID string) ([]domain.PMRow, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT `+pmColumns+` FROM permission_matrix_rows
		WHERE project_id = $1 ORDER BY created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanPM)
}

func (r *PMRepo) Add(ctx context.Context, projectID string, row domain.PMRow) (domain.PMRow, error) {
	return scanPM(r.db.q(ctx).QueryRow(ctx, `
		INSERT INTO permission_matrix_rows (project_id, role, action, resource, effect, condition, scope, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, clock_timestamp())
		RETURNING `+pmColumns,
		projectID, row.Role, row.Action, row.Resource, string(row.Effect), row.Condition, row.Scope))
}

func (r *PMRepo) Update(ctx context.Context, projectID, rowID string, p domain.PMRowPatch) (domain.PMRow, error) {
	var effect *string
	if p.Effect != nil {
		e := string(*p.Effect)
		effect = &e
	}
	row, err := scanPM(r.db.q(ctx).QueryRow(ctx, `
		UPDATE permission_matrix_rows SET
			role       = COALESCE($3, role),
			action     = COALESCE($4, action),
			resource   = COALESCE($5, resource),
			effect     = COALESCE($6, effect),
			condition  = CASE WHEN $7::boolean THEN $8::text ELSE condition END,
			scope      = CASE WHEN $9::boolean THEN $10::text ELSE scope END,
			updated_at = now()
		WHERE id = $1 AND project_id = $2
		RETURNING `+pmColumns,
		rowID, projectID, p.Role, p.Action, p.Resource, effect,
		p.Condition.Set, p.Condition.Value, p.Scope.Set, p.Scope.Value))
	return row, notFound(err)
}

func (r *PMRepo) Delete(ctx context.Context, projectID, rowID string) error {
	return affected(r.db.q(ctx).Exec(ctx,
		`DELETE FROM permission_matrix_rows WHERE id = $1 AND project_id = $2`, rowID, projectID))
}

func (r *PMRepo) ReplaceAll(ctx context.Context, projectID string, rows []domain.PMRow) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.q(ctx).Exec(ctx, `DELETE FROM permission_matrix_rows WHERE project_id = $1`, projectID); err != nil {
			return err
		}
		for _, row := range rows {
			if _, err := r.Add(ctx, projectID, row); err != nil {
				return err
			}
		}
		return nil
	})
}

// ---------- Phiên bản PM ----------

type PMVersionRepo struct{ db *DB }

func NewPMVersionRepo(db *DB) *PMVersionRepo { return &PMVersionRepo{db: db} }

const versionColumns = `id, project_id, version, checksum, rows, label, created_by, created_at`

func scanVersion(s scanner) (domain.PMVersion, error) {
	var v domain.PMVersion
	var raw []byte
	if err := s.Scan(&v.ID, &v.ProjectID, &v.Version, &v.Checksum, &raw, &v.Label, &v.CreatedBy, &v.CreatedAt); err != nil {
		return v, err
	}
	err := json.Unmarshal(raw, &v.Rows)
	return v, err
}

func (r *PMVersionRepo) Latest(ctx context.Context, projectID string) (*domain.PMVersion, error) {
	v, err := scanVersion(r.db.q(ctx).QueryRow(ctx, `SELECT `+versionColumns+` FROM pm_versions
		WHERE project_id = $1 ORDER BY version DESC LIMIT 1`, projectID))
	if err != nil {
		if domain.IsNotFound(notFound(err)) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *PMVersionRepo) Create(ctx context.Context, v domain.PMVersion) (domain.PMVersion, error) {
	raw, err := json.Marshal(v.Rows)
	if err != nil {
		return v, err
	}
	return scanVersion(r.db.q(ctx).QueryRow(ctx, `
		INSERT INTO pm_versions (project_id, version, checksum, rows, label, created_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+versionColumns,
		v.ProjectID, v.Version, v.Checksum, raw, v.Label, v.CreatedBy))
}

func (r *PMVersionRepo) List(ctx context.Context, projectID string) ([]domain.PMVersion, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT `+versionColumns+` FROM pm_versions
		WHERE project_id = $1 ORDER BY version DESC`, projectID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanVersion)
}

func (r *PMVersionRepo) Get(ctx context.Context, projectID, id string) (domain.PMVersion, error) {
	v, err := scanVersion(r.db.q(ctx).QueryRow(ctx, `SELECT `+versionColumns+` FROM pm_versions
		WHERE project_id = $1 AND id = $2`, projectID, id))
	return v, notFound(err)
}

// ---------- Từ điển ----------

type TermRepo struct{ db *DB }

func NewTermRepo(db *DB) *TermRepo { return &TermRepo{db: db} }

const termColumns = `id, project_id, kind, canonical, synonyms, description, created_at`

func scanTerm(s scanner) (domain.Term, error) {
	var t domain.Term
	var kind string
	err := s.Scan(&t.ID, &t.ProjectID, &kind, &t.Canonical, &t.Synonyms, &t.Description, &t.CreatedAt)
	t.Kind = domain.TermKind(kind)
	return t, err
}

func (r *TermRepo) List(ctx context.Context, projectID string) ([]domain.Term, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT `+termColumns+` FROM project_terms
		WHERE project_id = $1 ORDER BY kind, lower(canonical)`, projectID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanTerm)
}

func (r *TermRepo) Add(ctx context.Context, t domain.Term) (domain.Term, error) {
	saved, err := scanTerm(r.db.q(ctx).QueryRow(ctx, `
		INSERT INTO project_terms (project_id, kind, canonical, synonyms, description)
		VALUES ($1, $2, $3, $4, $5) RETURNING `+termColumns,
		t.ProjectID, string(t.Kind), t.Canonical, t.Synonyms, t.Description))
	if isUniqueViolation(err) {
		return saved, domain.Conflict("term_exists", "This term already exists in the dictionary")
	}
	return saved, err
}

func (r *TermRepo) Update(ctx context.Context, t domain.Term) (domain.Term, error) {
	saved, err := scanTerm(r.db.q(ctx).QueryRow(ctx, `
		UPDATE project_terms SET kind = $3, canonical = $4, synonyms = $5, description = $6, updated_at = now()
		WHERE id = $1 AND project_id = $2 RETURNING `+termColumns,
		t.ID, t.ProjectID, string(t.Kind), t.Canonical, t.Synonyms, t.Description))
	if isUniqueViolation(err) {
		return saved, domain.Conflict("term_exists", "This term already exists in the dictionary")
	}
	return saved, notFound(err)
}

func (r *TermRepo) Delete(ctx context.Context, projectID, id string) error {
	return affected(r.db.q(ctx).Exec(ctx, `DELETE FROM project_terms WHERE id = $1 AND project_id = $2`, id, projectID))
}
