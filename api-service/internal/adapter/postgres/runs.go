package postgres

import (
	"context"
	"encoding/json"

	"reqsentinel/api/internal/domain"
)

type RunRepo struct{ db *DB }

func NewRunRepo(db *DB) *RunRepo { return &RunRepo{db: db} }

// Định dạng lưu trong cột raw_result (giữ tương thích với dữ liệu từ bản Python).
type entityJSON struct {
	Role       *string `json:"role"`
	Action     *string `json:"action"`
	Resource   *string `json:"resource"`
	Effect     *string `json:"effect"`
	Condition  *string `json:"condition"`
	Scope      *string `json:"scope"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
}

type alignmentJSON struct {
	Extracted entityJSON    `json:"extracted"`
	Matched   *domain.PMRow `json:"matched_pm_row"`
	Score     float64       `json:"similarity_score"`
	IsMatched bool          `json:"is_matched"`
}

type rawResultJSON struct {
	PermissionMatrix []domain.PMRow  `json:"permission_matrix"`
	Alignments       []alignmentJSON `json:"alignment_results"`
}

func toEntityJSON(e domain.ExtractedEntity) entityJSON {
	return entityJSON{e.Role, e.Action, e.Resource, e.Effect, e.Condition, e.Scope, e.Source, e.Confidence}
}

func fromEntityJSON(e entityJSON) domain.ExtractedEntity {
	return domain.ExtractedEntity{Role: e.Role, Action: e.Action, Resource: e.Resource, Effect: e.Effect,
		Condition: e.Condition, Scope: e.Scope, Source: e.Source, Confidence: e.Confidence}
}

func (r *RunRepo) Save(ctx context.Context, run domain.Run) (string, error) {
	criteria := run.Input.AcceptanceCriteria
	if criteria == nil {
		criteria = []string{}
	}
	criteriaJSON, err := json.Marshal(criteria)
	if err != nil {
		return "", err
	}
	raw := rawResultJSON{PermissionMatrix: run.Input.PermissionMatrix, Alignments: []alignmentJSON{}}
	for _, a := range run.Result.Alignments {
		raw.Alignments = append(raw.Alignments, alignmentJSON{toEntityJSON(a.Extracted), a.Matched, a.Score, a.IsMatched})
	}
	if raw.PermissionMatrix == nil {
		raw.PermissionMatrix = []domain.PMRow{}
	}
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	var modelVersion *string
	if run.Result.ModelVersion != "" {
		modelVersion = &run.Result.ModelVersion
	}

	err = r.db.WithinTx(ctx, func(ctx context.Context) error {
		q := r.db.q(ctx)
		if _, err := q.Exec(ctx, `
			INSERT INTO pipeline_runs
				(id, project_id, created_by, user_story_id, job_id, pm_version_id, model_version,
				 user_story_content, acceptance_criteria, is_consistent, summary, raw_result)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			run.ID, run.ProjectID, run.CreatedBy, run.UserStoryID, run.JobID, run.PMVersionID, modelVersion,
			run.Input.UserStory, criteriaJSON, run.Result.IsConsistent, run.Result.Summary, rawJSON); err != nil {
			return err
		}
		for _, e := range run.Result.Entities {
			source := e.Source
			if source == "" {
				source = "User Story"
			}
			if _, err := q.Exec(ctx, `
				INSERT INTO extracted_entities
					(pipeline_run_id, role, action, resource, effect, condition, scope, source, confidence, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, clock_timestamp())`,
				run.ID, e.Role, e.Action, e.Resource, e.Effect, e.Condition, e.Scope, source, e.Confidence); err != nil {
				return err
			}
		}
		for i, c := range run.Result.RealConflicts() {
			var issueID *string
			if i < len(run.IssueIDs) && run.IssueIDs[i] != "" {
				issueID = &run.IssueIDs[i]
			}
			if _, err := q.Exec(ctx, `
				INSERT INTO conflict_records
					(pipeline_run_id, issue_id, conflict_type, confidence, explanation, evidence_us_ac, evidence_pm, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, clock_timestamp())`,
				run.ID, issueID, string(c.Type), c.Confidence, c.Explanation, c.EvidenceUSAC, c.EvidencePM); err != nil {
				return err
			}
		}
		return nil
	})
	return run.ID, err
}

func (r *RunRepo) List(ctx context.Context, f domain.RunFilter) ([]domain.RunSummary, error) {
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT r.id, r.project_id, p.name, r.user_story_id, left(r.user_story_content, 120), r.is_consistent,
		       left(COALESCE(r.summary, ''), 200),
		       COALESCE((SELECT array_agg(c.conflict_type::text ORDER BY c.created_at)
		                 FROM conflict_records c WHERE c.pipeline_run_id = r.id), '{}'),
		       r.model_version, r.created_by, r.created_at
		FROM pipeline_runs r
		LEFT JOIN projects p ON p.id = r.project_id
		WHERE ($1::uuid IS NULL OR r.created_by = $1 OR EXISTS (
		          SELECT 1 FROM project_members m WHERE m.project_id = r.project_id AND m.user_id = $1))
		  AND ($2::uuid IS NULL OR r.project_id = $2)
		ORDER BY r.created_at DESC
		LIMIT $3`, f.VisibleTo, f.ProjectID, f.Limit)
	if err != nil {
		return nil, err
	}
	return collect(rows, func(s scanner) (domain.RunSummary, error) {
		var rs domain.RunSummary
		var types []string
		err := s.Scan(&rs.ID, &rs.ProjectID, &rs.ProjectName, &rs.UserStoryID, &rs.UserStory, &rs.IsConsistent,
			&rs.Summary, &types, &rs.ModelVersion, &rs.CreatedBy, &rs.CreatedAt)
		for _, t := range types {
			rs.ConflictTypes = append(rs.ConflictTypes, domain.ConflictType(t))
		}
		if rs.ConflictTypes == nil {
			rs.ConflictTypes = []domain.ConflictType{}
		}
		return rs, err
	})
}

const runCoreSelect = `
	SELECT r.id, r.project_id, r.user_story_id, r.pm_version_id, r.job_id, r.created_by, r.user_story_content,
	       COALESCE(r.acceptance_criteria::text, '[]'), r.is_consistent, r.summary, r.model_version,
	       COALESCE(r.raw_result::text, '{}'), r.created_at
	FROM pipeline_runs r`

func scanRunCore(s scanner) (domain.RunDetail, error) {
	var d domain.RunDetail
	var criteria, raw string
	if err := s.Scan(&d.ID, &d.ProjectID, &d.UserStoryID, &d.PMVersionID, &d.JobID, &d.CreatedBy, &d.UserStory,
		&criteria, &d.IsConsistent, &d.Summary, &d.ModelVersion, &raw, &d.CreatedAt); err != nil {
		return d, err
	}
	d.AcceptanceCriteria = []string{}
	_ = json.Unmarshal([]byte(criteria), &d.AcceptanceCriteria)
	var rr rawResultJSON
	_ = json.Unmarshal([]byte(raw), &rr)
	d.PermissionMatrix = rr.PermissionMatrix
	if d.PermissionMatrix == nil {
		d.PermissionMatrix = []domain.PMRow{}
	}
	d.Alignments = []domain.Alignment{}
	for _, a := range rr.Alignments {
		d.Alignments = append(d.Alignments, domain.Alignment{Extracted: fromEntityJSON(a.Extracted),
			Matched: a.Matched, Score: a.Score, IsMatched: a.IsMatched})
	}
	return d, nil
}

func (r *RunRepo) Get(ctx context.Context, id string) (domain.RunDetail, error) {
	d, err := scanRunCore(r.db.q(ctx).QueryRow(ctx, runCoreSelect+` WHERE r.id = $1`, id))
	if err != nil {
		return d, notFound(err)
	}
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT issue_id, conflict_type, confidence, explanation, evidence_us_ac, evidence_pm
		FROM conflict_records WHERE pipeline_run_id = $1 ORDER BY created_at, id`, id)
	if err != nil {
		return d, err
	}
	if d.Conflicts, err = collect(rows, func(s scanner) (domain.StoredConflict, error) {
		var c domain.StoredConflict
		var t string
		err := s.Scan(&c.IssueID, &t, &c.Confidence, &c.Explanation, &c.EvidenceUSAC, &c.EvidencePM)
		c.Type = domain.ConflictType(t)
		return c, err
	}); err != nil {
		return d, err
	}
	rows, err = r.db.q(ctx).Query(ctx, `
		SELECT role, action, resource, effect, condition, scope, source, confidence
		FROM extracted_entities WHERE pipeline_run_id = $1 ORDER BY created_at, id`, id)
	if err != nil {
		return d, err
	}
	d.Entities, err = collect(rows, func(s scanner) (domain.ExtractedEntity, error) {
		var e domain.ExtractedEntity
		err := s.Scan(&e.Role, &e.Action, &e.Resource, &e.Effect, &e.Condition, &e.Scope, &e.Source, &e.Confidence)
		return e, err
	})
	return d, err
}

func (r *RunRepo) LatestPerStory(ctx context.Context, projectID string) ([]domain.RunDetail, error) {
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT DISTINCT ON (r.user_story_id)
		       r.id, r.project_id, r.user_story_id, r.pm_version_id, r.job_id, r.created_by, r.user_story_content,
		       COALESCE(r.acceptance_criteria::text, '[]'), r.is_consistent, r.summary, r.model_version,
		       COALESCE(r.raw_result::text, '{}'), r.created_at
		FROM pipeline_runs r
		WHERE r.project_id = $1
		  AND r.user_story_id IN (SELECT id FROM user_stories WHERE project_id = $1)
		ORDER BY r.user_story_id, r.created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanRunCore)
}
