// Package analysis: chạy phân tích (1 story, ad-hoc, cả project qua job nền),
// lịch sử và so sánh các lần chạy.
package analysis

import (
	"context"
	"fmt"
	"log/slog"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/app/requirements"
	"reqsentinel/api/internal/domain"
)

type Deps struct {
	Analyzer   app.Analyzer
	Runs       app.RunRepository
	Issues     app.IssueRepository
	Jobs       app.JobRepository
	Stories    app.StoryRepository
	PM         app.PMRepository
	PMVersions app.PMVersionRepository
	Terms      app.TermRepository
	Projects   app.ProjectRepository
	Members    app.MemberRepository
	Auth       app.Authorizer
	Events     app.EventPublisher
	Tx         app.TxManager
	Audit      app.AuditLog
	Clock      app.Clock
	NewID      app.IDGenerator
	Log        *slog.Logger
}

type Service struct{ d Deps }

func New(d Deps) *Service { return &Service{d: d} }

// ---------- Chạy 1 user story ----------

func (s *Service) RunStory(ctx context.Context, actor domain.Actor, projectID, storyID string, meta app.RequestMeta) (domain.Run, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermRunAnalysis); err != nil {
		return domain.Run{}, err
	}
	story, err := s.d.Stories.Get(ctx, projectID, storyID)
	if err != nil {
		return domain.Run{}, app.NotFoundAs(err, "story_not_found", "User story not found in this project")
	}
	ctxData, err := s.loadProjectContext(ctx, projectID)
	if err != nil {
		return domain.Run{}, err
	}
	run, err := s.analyzeStory(ctx, projectID, story, ctxData, actor.ID, nil)
	if err != nil {
		return run, err
	}
	s.auditRun(ctx, actor, meta, run)
	return run, nil
}

// RunAdhoc: phân tích dữ liệu nhập tay, không gắn user story (không tạo issue).
func (s *Service) RunAdhoc(ctx context.Context, actor domain.Actor, in domain.AnalysisInput, projectID *string, meta app.RequestMeta) (domain.Run, error) {
	if err := in.Validate(); err != nil {
		return domain.Run{}, err
	}
	if projectID != nil {
		if _, _, err := s.d.Auth.Require(ctx, actor, *projectID, domain.PermRunAnalysis); err != nil {
			return domain.Run{}, err
		}
	}
	result, err := s.d.Analyzer.Analyze(ctx, in)
	if err != nil {
		return domain.Run{}, err
	}
	run := domain.Run{ID: s.d.NewID(), ProjectID: projectID, CreatedBy: actor.ID, Input: in, Result: result,
		IssueIDs: make([]string, len(result.RealConflicts()))}
	if _, err := s.d.Runs.Save(ctx, run); err != nil {
		return run, err
	}
	s.auditRun(ctx, actor, meta, run)
	return run, nil
}

type projectContext struct {
	rows       []domain.PMRow
	normalizer *domain.TermNormalizer
}

func (s *Service) loadProjectContext(ctx context.Context, projectID string) (projectContext, error) {
	rows, err := s.d.PM.List(ctx, projectID)
	if err != nil {
		return projectContext{}, err
	}
	if len(rows) == 0 {
		return projectContext{}, domain.Validation("project_no_pm", "Project has no permission matrix rows")
	}
	terms, err := s.d.Terms.List(ctx, projectID)
	if err != nil {
		return projectContext{}, err
	}
	return projectContext{rows: rows, normalizer: domain.NewTermNormalizer(terms)}, nil
}

// analyzeStory: chuẩn hoá theo từ điển → gọi ML → (1 transaction) chụp phiên bản PM,
// đồng bộ conflict issue, lưu run.
func (s *Service) analyzeStory(ctx context.Context, projectID string, story domain.UserStory, pc projectContext,
	actorID string, jobID *string) (domain.Run, error) {

	in := domain.AnalysisInput{
		UserStory: story.Content, AcceptanceCriteria: story.CriteriaTexts(), PermissionMatrix: pc.rows,
	}.Normalize(pc.normalizer)

	result, err := s.d.Analyzer.Analyze(ctx, in)
	if err != nil {
		return domain.Run{}, err
	}

	storyID := story.ID
	run := domain.Run{ID: s.d.NewID(), ProjectID: &projectID, UserStoryID: &storyID, JobID: jobID,
		CreatedBy: actorID, Input: in, Result: result}

	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		version, err := requirements.SnapshotPM(ctx, s.d.PM, s.d.PMVersions, projectID, &actorID, nil)
		if err != nil {
			return err
		}
		run.PMVersionID = &version.ID

		existing, err := s.d.Issues.ListByStory(ctx, projectID, storyID)
		if err != nil {
			return err
		}
		sync := domain.SyncIssues(projectID, storyID, run.ID, existing, result.RealConflicts(), s.d.Clock(), s.d.NewID)
		if err := s.d.Issues.Create(ctx, sync.Created); err != nil {
			return err
		}
		if err := s.d.Issues.Update(ctx, sync.Updated); err != nil {
			return err
		}
		run.IssueIDs = sync.RunIssueIDs
		if _, err := s.d.Runs.Save(ctx, run); err != nil {
			return err
		}
		return s.d.Issues.AddActivities(ctx, sync.Activities)
	})
	return run, err
}

func (s *Service) auditRun(ctx context.Context, actor domain.Actor, meta app.RequestMeta, run domain.Run) {
	types := []domain.ConflictType{}
	for _, c := range run.Result.RealConflicts() {
		types = append(types, c.Type)
	}
	detail := map[string]any{"is_consistent": run.Result.IsConsistent, "conflict_types": types}
	if run.ProjectID != nil {
		detail["project_id"] = *run.ProjectID
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "analysis.run", "run", run.ID, detail))
}

// ---------- Job phân tích cả project ----------

func (s *Service) StartJob(ctx context.Context, actor domain.Actor, projectID string, meta app.RequestMeta) (domain.AnalysisJob, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermRunAnalysis); err != nil {
		return domain.AnalysisJob{}, err
	}
	stories, err := s.d.Stories.List(ctx, projectID)
	if err != nil {
		return domain.AnalysisJob{}, err
	}
	if len(stories) == 0 {
		return domain.AnalysisJob{}, domain.Validation("project_no_stories", "Project has no user stories")
	}
	if _, err := s.loadProjectContext(ctx, projectID); err != nil {
		return domain.AnalysisJob{}, err
	}
	if active, err := s.d.Jobs.ActiveForProject(ctx, projectID); err != nil {
		return domain.AnalysisJob{}, err
	} else if active != nil {
		return domain.AnalysisJob{}, domain.Conflict("analysis_job_running", "An analysis job is already running for this project")
	}
	job, err := s.d.Jobs.Create(ctx, domain.AnalysisJob{ProjectID: projectID, CreatedBy: actor.ID,
		Status: domain.JobQueued, Total: len(stories)})
	if err != nil {
		return job, err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "analysis.job_started", "project", projectID,
		map[string]any{"job_id": job.ID, "stories": job.Total}))
	return job, nil
}

func (s *Service) Jobs(ctx context.Context, actor domain.Actor, projectID string) ([]domain.AnalysisJob, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermView); err != nil {
		return nil, err
	}
	return s.d.Jobs.ListByProject(ctx, projectID, 20)
}

func (s *Service) Job(ctx context.Context, actor domain.Actor, projectID, jobID string) (domain.AnalysisJob, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermView); err != nil {
		return domain.AnalysisJob{}, err
	}
	job, err := s.d.Jobs.Get(ctx, jobID)
	if err != nil || job.ProjectID != projectID {
		return job, domain.NotFound("job_not_found", "Job not found")
	}
	return job, nil
}

// RecoverInterruptedJobs: job đang chạy khi service tắt đột ngột → đánh dấu failed.
func (s *Service) RecoverInterruptedJobs(ctx context.Context) error {
	n, err := s.d.Jobs.FailInterrupted(ctx, s.d.Clock())
	if n > 0 {
		s.d.Log.Warn("marked interrupted analysis jobs as failed", "count", n)
	}
	return err
}

// ProcessNextJob lấy 1 job đang chờ và chạy hết; trả false nếu không có job.
func (s *Service) ProcessNextJob(ctx context.Context) (bool, error) {
	job, err := s.d.Jobs.ClaimNext(ctx, s.d.Clock())
	if err != nil || job == nil {
		return false, err
	}
	s.d.Log.Info("analysis job started", "job_id", job.ID, "project_id", job.ProjectID)

	fail := func(msg string) (bool, error) {
		job.Status, job.Error = domain.JobFailed, &msg
		return true, s.finishJob(ctx, job)
	}
	stories, err := s.d.Stories.List(ctx, job.ProjectID)
	if err != nil {
		return fail(err.Error())
	}
	pc, err := s.loadProjectContext(ctx, job.ProjectID)
	if err != nil {
		return fail(err.Error())
	}
	job.Total, job.Processed, job.Failed = len(stories), 0, 0

	var lastErr error
	for _, story := range stories {
		if ctx.Err() != nil {
			return fail("service is shutting down")
		}
		run, err := s.analyzeStory(ctx, job.ProjectID, story, pc, job.CreatedBy, &job.ID)
		if err != nil {
			job.Failed++
			lastErr = err
			s.d.Log.Warn("job story failed", "job_id", job.ID, "story_id", story.ID, "err", err)
		} else if run.Result.ModelVersion != "" {
			mv := run.Result.ModelVersion
			job.ModelVersion = &mv
		}
		job.Processed++
		if err := s.d.Jobs.Save(ctx, *job); err != nil {
			return true, err
		}
		s.publish(ctx, domain.EventJobProgress, job, nil)
	}

	job.Status = domain.JobCompleted
	if job.Failed == job.Total && lastErr != nil {
		msg := fmt.Sprintf("all stories failed: %v", lastErr)
		job.Status, job.Error = domain.JobFailed, &msg
	}
	return true, s.finishJob(ctx, job)
}

func (s *Service) finishJob(ctx context.Context, job *domain.AnalysisJob) error {
	now := s.d.Clock()
	job.FinishedAt = &now
	if err := s.d.Jobs.Save(ctx, *job); err != nil {
		return err
	}
	_, openIssues, err := s.d.Issues.List(ctx, domain.IssueFilter{ProjectID: job.ProjectID, Status: string(domain.IssueOpen), Limit: 1})
	if err != nil {
		return err
	}
	s.publish(ctx, domain.EventJobFinished, job, map[string]any{"open_issues": openIssues})
	s.d.Log.Info("analysis job finished", "job_id", job.ID, "status", job.Status, "failed", job.Failed)
	return nil
}

func (s *Service) publish(ctx context.Context, typ string, job *domain.AnalysisJob, extra map[string]any) {
	p, err := s.d.Projects.Get(ctx, job.ProjectID)
	if err != nil {
		s.d.Log.Warn("publish job event", "err", err)
		return
	}
	data := map[string]any{
		"job_id": job.ID, "project_id": job.ProjectID, "project_name": p.Name, "status": job.Status,
		"total": job.Total, "processed": job.Processed, "failed": job.Failed,
	}
	if job.Error != nil {
		data["error"] = *job.Error
	}
	for k, v := range extra {
		data[k] = v
	}
	ev := app.NewEvent(s.d.NewID, s.d.Clock, typ, nil, []string{job.CreatedBy}, data)
	if err := s.d.Events.Publish(ctx, ev); err != nil {
		s.d.Log.Warn("publish job event", "err", err)
	}
}

// ---------- Lịch sử ----------

func (s *Service) History(ctx context.Context, actor domain.Actor, onlyMine bool, limit int) ([]domain.RunSummary, error) {
	f := domain.RunFilter{Limit: limit}
	if !actor.IsAdmin() || onlyMine {
		f.VisibleTo = &actor.ID
	}
	return s.d.Runs.List(ctx, f)
}

func (s *Service) ProjectRuns(ctx context.Context, actor domain.Actor, projectID string, limit int) ([]domain.RunSummary, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermView); err != nil {
		return nil, err
	}
	return s.d.Runs.List(ctx, domain.RunFilter{ProjectID: &projectID, Limit: limit})
}

// RunDetail: admin, người chạy, hoặc thành viên project được xem.
func (s *Service) RunDetail(ctx context.Context, actor domain.Actor, runID string) (domain.RunDetail, error) {
	run, err := s.d.Runs.Get(ctx, runID)
	if err != nil {
		return run, app.NotFoundAs(err, "run_not_found", "Run not found")
	}
	if actor.IsAdmin() || (run.CreatedBy != nil && *run.CreatedBy == actor.ID) {
		return run, nil
	}
	if run.ProjectID != nil {
		if _, _, err := s.d.Auth.Require(ctx, actor, *run.ProjectID, domain.PermView); err == nil {
			return run, nil
		}
	}
	return domain.RunDetail{}, domain.NotFound("run_not_found", "Run not found")
}

func (s *Service) CompareRuns(ctx context.Context, actor domain.Actor, projectID, baseID, targetID string) (domain.RunComparison, domain.RunDetail, domain.RunDetail, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermView); err != nil {
		return domain.RunComparison{}, domain.RunDetail{}, domain.RunDetail{}, err
	}
	load := func(id string) (domain.RunDetail, error) {
		r, err := s.d.Runs.Get(ctx, id)
		if err != nil || r.ProjectID == nil || *r.ProjectID != projectID {
			return r, domain.NotFound("run_not_found", "Run not found")
		}
		return r, nil
	}
	base, err := load(baseID)
	if err != nil {
		return domain.RunComparison{}, base, domain.RunDetail{}, err
	}
	target, err := load(targetID)
	if err != nil {
		return domain.RunComparison{}, base, target, err
	}
	return domain.CompareRuns(base, target), base, target, nil
}
