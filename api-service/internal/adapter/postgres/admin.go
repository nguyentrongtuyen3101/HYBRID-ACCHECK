package postgres

import (
	"context"
	"encoding/json"
	"log/slog"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

// ---------- Audit ----------

type AuditRepo struct {
	db  *DB
	log *slog.Logger
}

func NewAuditRepo(db *DB, log *slog.Logger) *AuditRepo { return &AuditRepo{db: db, log: log} }

func (r *AuditRepo) Insert(ctx context.Context, e domain.AuditEntry) error {
	var detail []byte
	if e.Detail != nil {
		var err error
		if detail, err = json.Marshal(e.Detail); err != nil {
			return err
		}
	}
	// Luôn dùng pool (không theo transaction): lỗi audit không được làm hỏng transaction nghiệp vụ
	_, err := r.db.Pool.Exec(context.WithoutCancel(ctx), `
		INSERT INTO audit_logs (actor_id, actor_email, action, target_type, target_id, detail, ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ActorID, e.ActorEmail, e.Action, e.TargetType, e.TargetID, detail, e.IP)
	return err
}

// Record hiện thực app.AuditLog: lỗi chỉ ghi log.
func (r *AuditRepo) Record(ctx context.Context, e domain.AuditEntry) {
	if err := r.Insert(ctx, e); err != nil {
		r.log.Warn("audit log failed", "action", e.Action, "err", err)
	}
}

func (r *AuditRepo) List(ctx context.Context, f domain.AuditFilter) ([]domain.AuditEntry, int, error) {
	const where = ` WHERE ($1 = '' OR action = $1) AND ($2::uuid IS NULL OR actor_id = $2)`
	var total int
	if err := r.db.q(ctx).QueryRow(ctx, `SELECT count(*) FROM audit_logs`+where, f.Action, f.ActorID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT id, actor_id, actor_email, action, target_type, target_id, detail::text, ip, created_at
		FROM audit_logs`+where+`
		ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`, f.Action, f.ActorID, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	items, err := collect(rows, func(s scanner) (domain.AuditEntry, error) {
		var e domain.AuditEntry
		var detail *string
		err := s.Scan(&e.ID, &e.ActorID, &e.ActorEmail, &e.Action, &e.TargetType, &e.TargetID, &detail, &e.IP, &e.CreatedAt)
		if detail != nil {
			_ = json.Unmarshal([]byte(*detail), &e.Detail)
		}
		return e, err
	})
	return items, total, err
}

// ---------- Thống kê ----------

type StatsRepo struct {
	db     *DB
	runs   *RunRepo
	issues *IssueRepo
}

func NewStatsRepo(db *DB, runs *RunRepo, issues *IssueRepo) *StatsRepo {
	return &StatsRepo{db: db, runs: runs, issues: issues}
}

func (r *StatsRepo) AdminStats(ctx context.Context) (domain.AdminStats, error) {
	var st domain.AdminStats
	q := r.db.q(ctx)
	if err := q.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users),
			(SELECT count(*) FROM users WHERE status = 'active'),
			(SELECT count(*) FROM users WHERE status = 'disabled'),
			(SELECT count(*) FROM users WHERE role = 'admin'),
			(SELECT count(*) FROM projects),
			(SELECT count(*) FROM pipeline_runs),
			(SELECT count(*) FROM pipeline_runs WHERE created_at >= now() - interval '7 days'),
			(SELECT count(*) FROM pipeline_runs WHERE is_consistent),
			(SELECT count(*) FROM conflict_issues WHERE status IN ('open', 'acknowledged'))`,
	).Scan(&st.UsersTotal, &st.UsersActive, &st.UsersDisabled, &st.Admins, &st.Projects,
		&st.RunsTotal, &st.RunsLast7Days, &st.RunsConsistent, &st.OpenIssues); err != nil {
		return st, err
	}
	var err error
	if st.ConflictsByType, err = countMap(ctx, q,
		`SELECT conflict_type, count(*) FROM conflict_records GROUP BY conflict_type`); err != nil {
		return st, err
	}
	if st.RunsByModel, err = countMap(ctx, q,
		`SELECT COALESCE(model_version, 'unknown'), count(*) FROM pipeline_runs GROUP BY 1`); err != nil {
		return st, err
	}
	rows, err := q.Query(ctx, `
		SELECT to_char(d, 'YYYY-MM-DD'), count(r.id)
		FROM generate_series(current_date - 13, current_date, interval '1 day') AS d
		LEFT JOIN pipeline_runs r ON r.created_at >= d AND r.created_at < d + interval '1 day'
		GROUP BY d ORDER BY d`)
	if err != nil {
		return st, err
	}
	st.RunsPerDay, err = collect(rows, func(s scanner) (domain.DailyCount, error) {
		var dc domain.DailyCount
		return dc, s.Scan(&dc.Day, &dc.Count)
	})
	return st, err
}

func countMap(ctx context.Context, q querier, sql string) (map[string]int, error) {
	rows, err := q.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out[k] = n
	}
	return out, rows.Err()
}

// ReviewedStories: các story có ít nhất 1 issue đã được người dùng đánh giá,
// kèm lần chạy mới nhất và toàn bộ issue của story đó.
func (r *StatsRepo) ReviewedStories(ctx context.Context) ([]app.ReviewedStory, error) {
	rows, err := r.db.q(ctx).Query(ctx, `
		SELECT DISTINCT project_id::text, user_story_id::text FROM conflict_issues
		WHERE status IN ('acknowledged', 'false_positive') OR (status = 'resolved' AND resolved_by IS NOT NULL)`)
	if err != nil {
		return nil, err
	}
	type key struct{ project, story string }
	keys, err := collect(rows, func(s scanner) (key, error) {
		var k key
		return k, s.Scan(&k.project, &k.story)
	})
	if err != nil {
		return nil, err
	}
	out := []app.ReviewedStory{}
	for _, k := range keys {
		run, err := scanRunCore(r.db.q(ctx).QueryRow(ctx, runCoreSelect+`
			WHERE r.project_id = $1 AND r.user_story_id = $2 ORDER BY r.created_at DESC LIMIT 1`, k.project, k.story))
		if err != nil {
			if domain.IsNotFound(notFound(err)) {
				continue
			}
			return nil, err
		}
		issues, _, err := r.issues.List(ctx, domain.IssueFilter{ProjectID: k.project, UserStoryID: k.story, Limit: 1000})
		if err != nil {
			return nil, err
		}
		out = append(out, app.ReviewedStory{ProjectID: k.project, StoryID: k.story, Run: run, Issues: issues})
	}
	return out, nil
}
