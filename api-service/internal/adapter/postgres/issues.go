package postgres

import (
	"context"
	"time"

	"reqsentinel/api/internal/domain"
)

type IssueRepo struct{ db *DB }

func NewIssueRepo(db *DB) *IssueRepo { return &IssueRepo{db: db} }

const issueSelect = `
	SELECT i.id, i.project_id, i.user_story_id, left(s.content, 200), i.fingerprint, i.conflict_type, i.explanation,
	       i.evidence_us_ac, i.evidence_pm, i.role, i.action, i.resource, i.status,
	       i.assignee_id, a.email, NULLIF(a.full_name, ''),
	       i.first_seen_run_id, i.last_seen_run_id, i.last_seen_at, i.resolved_by, i.resolved_at,
	       (SELECT count(*) FROM conflict_activities ca WHERE ca.issue_id = i.id AND ca.kind = 'comment'),
	       i.created_at, i.updated_at
	FROM conflict_issues i
	JOIN user_stories s ON s.id = i.user_story_id
	LEFT JOIN users a ON a.id = i.assignee_id`

func scanIssue(s scanner) (domain.Issue, error) {
	var i domain.Issue
	var typ, status string
	err := s.Scan(&i.ID, &i.ProjectID, &i.UserStoryID, &i.StoryContent, &i.Fingerprint, &typ, &i.Explanation,
		&i.EvidenceUSAC, &i.EvidencePM, &i.Role, &i.Action, &i.Resource, &status,
		&i.AssigneeID, &i.AssigneeEmail, &i.AssigneeName,
		&i.FirstSeenRunID, &i.LastSeenRunID, &i.LastSeenAt, &i.ResolvedBy, &i.ResolvedAt,
		&i.CommentCount, &i.CreatedAt, &i.UpdatedAt)
	i.Type, i.Status = domain.ConflictType(typ), domain.IssueStatus(status)
	return i, err
}

func (r *IssueRepo) ListByStory(ctx context.Context, projectID, storyID string) ([]domain.Issue, error) {
	rows, err := r.db.q(ctx).Query(ctx, issueSelect+`
		WHERE i.project_id = $1 AND i.user_story_id = $2 ORDER BY i.created_at FOR UPDATE OF i`, projectID, storyID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanIssue)
}

func (r *IssueRepo) List(ctx context.Context, f domain.IssueFilter) ([]domain.Issue, int, error) {
	const where = `
		WHERE i.project_id = $1
		  AND ($2::text = '' OR i.status::text = $2 OR ($2 = 'active' AND i.status IN ('open', 'acknowledged')))
		  AND ($3::text = '' OR i.conflict_type::text = $3)
		  AND ($4 = '' OR i.assignee_id::text = $4)
		  AND ($5 = '' OR i.user_story_id::text = $5)`
	args := []any{f.ProjectID, f.Status, f.Type, f.AssigneeID, f.UserStoryID}
	var total int
	if err := r.db.q(ctx).QueryRow(ctx, `SELECT count(*) FROM conflict_issues i`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.q(ctx).Query(ctx, issueSelect+where+`
		ORDER BY CASE i.status WHEN 'open' THEN 0 WHEN 'acknowledged' THEN 1 ELSE 2 END, i.updated_at DESC
		LIMIT $6 OFFSET $7`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	issues, err := collect(rows, scanIssue)
	return issues, total, err
}

func (r *IssueRepo) Get(ctx context.Context, projectID, id string) (domain.Issue, error) {
	iss, err := scanIssue(r.db.q(ctx).QueryRow(ctx, issueSelect+` WHERE i.project_id = $1 AND i.id = $2`, projectID, id))
	return iss, notFound(err)
}

func (r *IssueRepo) Create(ctx context.Context, issues []domain.Issue) error {
	for _, i := range issues {
		if _, err := r.db.q(ctx).Exec(ctx, `
			INSERT INTO conflict_issues
				(id, project_id, user_story_id, fingerprint, conflict_type, explanation, evidence_us_ac, evidence_pm,
				 role, action, resource, status, assignee_id, first_seen_run_id, last_seen_run_id, last_seen_at,
				 created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $17)`,
			i.ID, i.ProjectID, i.UserStoryID, i.Fingerprint, string(i.Type), i.Explanation, i.EvidenceUSAC, i.EvidencePM,
			i.Role, i.Action, i.Resource, string(i.Status), i.AssigneeID, i.FirstSeenRunID, i.LastSeenRunID,
			i.LastSeenAt, i.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *IssueRepo) Update(ctx context.Context, issues []domain.Issue) error {
	for _, i := range issues {
		if err := affected(r.db.q(ctx).Exec(ctx, `
			UPDATE conflict_issues SET
				status = $2, assignee_id = $3, last_seen_run_id = $4, last_seen_at = $5,
				resolved_by = $6, resolved_at = $7, updated_at = $8
			WHERE id = $1`,
			i.ID, string(i.Status), i.AssigneeID, i.LastSeenRunID, i.LastSeenAt, i.ResolvedBy, i.ResolvedAt,
			i.UpdatedAt)); err != nil {
			return err
		}
	}
	return nil
}

func (r *IssueRepo) AddActivities(ctx context.Context, acts []domain.IssueActivity) error {
	for _, a := range acts {
		if _, err := r.db.q(ctx).Exec(ctx, `
			INSERT INTO conflict_activities (id, issue_id, actor_id, kind, body, from_value, to_value, run_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			a.ID, a.IssueID, a.ActorID, string(a.Kind), a.Body, a.FromValue, a.ToValue, a.RunID, a.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *IssueRepo) Activities(ctx context.Context, issueID string) ([]domain.IssueActivity, error) {
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT a.id, a.issue_id, a.actor_id, COALESCE(NULLIF(u.full_name, ''), u.email), a.kind, a.body,
		       a.from_value, a.to_value, a.run_id, a.created_at
		FROM conflict_activities a LEFT JOIN users u ON u.id = a.actor_id
		WHERE a.issue_id = $1 ORDER BY a.created_at, a.id`, issueID)
	if err != nil {
		return nil, err
	}
	return collect(rows, func(s scanner) (domain.IssueActivity, error) {
		var a domain.IssueActivity
		var kind string
		err := s.Scan(&a.ID, &a.IssueID, &a.ActorID, &a.ActorName, &kind, &a.Body, &a.FromValue, &a.ToValue,
			&a.RunID, &a.CreatedAt)
		a.Kind = domain.ActivityKind(kind)
		return a, err
	})
}

func (r *IssueRepo) Participants(ctx context.Context, issueID string) ([]string, error) {
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT DISTINCT actor_id::text FROM conflict_activities WHERE issue_id = $1 AND actor_id IS NOT NULL`, issueID)
	if err != nil {
		return nil, err
	}
	return collect(rows, func(s scanner) (string, error) {
		var id string
		return id, s.Scan(&id)
	})
}

// ---------- Job phân tích ----------

type JobRepo struct{ db *DB }

func NewJobRepo(db *DB) *JobRepo { return &JobRepo{db: db} }

const jobSelect = `
	SELECT j.id, j.project_id, p.name, j.created_by, j.status, j.total, j.processed, j.failed, j.error,
	       j.model_version, j.created_at, j.started_at, j.finished_at
	FROM analysis_jobs j JOIN projects p ON p.id = j.project_id`

func scanJob(s scanner) (domain.AnalysisJob, error) {
	var j domain.AnalysisJob
	var status string
	err := s.Scan(&j.ID, &j.ProjectID, &j.ProjectName, &j.CreatedBy, &status, &j.Total, &j.Processed, &j.Failed,
		&j.Error, &j.ModelVersion, &j.CreatedAt, &j.StartedAt, &j.FinishedAt)
	j.Status = domain.JobStatus(status)
	return j, err
}

func (r *JobRepo) Create(ctx context.Context, j domain.AnalysisJob) (domain.AnalysisJob, error) {
	var id string
	if err := r.db.q(ctx).QueryRow(ctx, `
		INSERT INTO analysis_jobs (project_id, created_by, status, total) VALUES ($1, $2, $3, $4) RETURNING id`,
		j.ProjectID, j.CreatedBy, string(j.Status), j.Total).Scan(&id); err != nil {
		return j, err
	}
	return r.Get(ctx, id)
}

func (r *JobRepo) Get(ctx context.Context, id string) (domain.AnalysisJob, error) {
	j, err := scanJob(r.db.q(ctx).QueryRow(ctx, jobSelect+` WHERE j.id = $1`, id))
	return j, notFound(err)
}

func (r *JobRepo) ListByProject(ctx context.Context, projectID string, limit int) ([]domain.AnalysisJob, error) {
	rows, err := r.db.q(ctx).Query(ctx, jobSelect+` WHERE j.project_id = $1 ORDER BY j.created_at DESC LIMIT $2`,
		projectID, limit)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanJob)
}

func (r *JobRepo) ActiveForProject(ctx context.Context, projectID string) (*domain.AnalysisJob, error) {
	j, err := scanJob(r.db.q(ctx).QueryRow(ctx, jobSelect+`
		WHERE j.project_id = $1 AND j.status IN ('queued', 'running') LIMIT 1`, projectID))
	if err != nil {
		if domain.IsNotFound(notFound(err)) {
			return nil, nil
		}
		return nil, err
	}
	return &j, nil
}

func (r *JobRepo) ClaimNext(ctx context.Context, now time.Time) (*domain.AnalysisJob, error) {
	var id string
	err := r.db.q(ctx).QueryRow(ctx, `
		UPDATE analysis_jobs SET status = 'running', started_at = $1
		WHERE id = (
			SELECT id FROM analysis_jobs WHERE status = 'queued'
			ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED
		)
		RETURNING id`, now).Scan(&id)
	if err != nil {
		if domain.IsNotFound(notFound(err)) {
			return nil, nil
		}
		return nil, err
	}
	j, err := r.Get(ctx, id)
	return &j, err
}

func (r *JobRepo) Save(ctx context.Context, j domain.AnalysisJob) error {
	return affected(r.db.q(ctx).Exec(ctx, `
		UPDATE analysis_jobs SET status = $2, total = $3, processed = $4, failed = $5, error = $6,
			model_version = $7, finished_at = $8
		WHERE id = $1`,
		j.ID, string(j.Status), j.Total, j.Processed, j.Failed, j.Error, j.ModelVersion, j.FinishedAt))
}

func (r *JobRepo) FailInterrupted(ctx context.Context, now time.Time) (int64, error) {
	tag, err := r.db.q(ctx).Exec(ctx, `
		UPDATE analysis_jobs SET status = 'failed', error = 'interrupted (service restarted)', finished_at = $1
		WHERE status = 'running'`, now)
	return tag.RowsAffected(), err
}
