package http

import (
	"net/http"
	"net/url"

	"reqsentinel/api/internal/domain"
)

func (s *Server) checkStory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	var body struct {
		UserStoryID string `json:"user_story_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if !validUUID(body.UserStoryID) {
		writeDetail(w, http.StatusNotFound, "story_not_found", "User story not found in this project")
		return
	}
	run, err := s.Analysis.RunStory(r.Context(), actorFrom(r), id, body.UserStoryID, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAnalysisDTO(run))
}

func (s *Server) adhocCheck(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserStory          string     `json:"user_story"`
		AcceptanceCriteria []string   `json:"acceptance_criteria"`
		PermissionMatrix   []pmRowDTO `json:"permission_matrix"`
		ProjectID          *string    `json:"project_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.ProjectID != nil && !validUUID(*body.ProjectID) {
		writeDetail(w, http.StatusNotFound, "project_not_found", "Project not found")
		return
	}
	in := domain.AnalysisInput{UserStory: body.UserStory, AcceptanceCriteria: body.AcceptanceCriteria}
	for _, row := range body.PermissionMatrix {
		in.PermissionMatrix = append(in.PermissionMatrix, domain.PMRow{Role: row.Role, Action: row.Action,
			Resource: row.Resource, Effect: domain.Effect(row.Effect), Condition: row.Condition, Scope: row.Scope})
	}
	run, err := s.Analysis.RunAdhoc(r.Context(), actorFrom(r), in, body.ProjectID, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAnalysisDTO(run))
}

func (s *Server) projectRuns(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	runs, err := s.Analysis.ProjectRuns(r.Context(), actorFrom(r), id, queryInt(r, "limit", 20, 1, 200))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapSlice(runs, toRunSummaryDTO)})
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	runs, err := s.Analysis.History(r.Context(), actorFrom(r), r.URL.Query().Get("mine") == "true",
		queryInt(r, "limit", 20, 1, 200))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapSlice(runs, toRunSummaryDTO)})
}

func (s *Server) runDetail(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runID", "run_not_found", "Run not found")
	if !ok {
		return
	}
	run, err := s.Analysis.RunDetail(r.Context(), actorFrom(r), runID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toRunDetailDTO(run))
}

func (s *Server) compareRuns(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	q := r.URL.Query()
	base, target := q.Get("base"), q.Get("target")
	if !validUUID(base) || !validUUID(target) {
		writeDetail(w, http.StatusUnprocessableEntity, "invalid_run_ids", "base and target must be run ids")
		return
	}
	cmp, b, t, err := s.Analysis.CompareRuns(r.Context(), actorFrom(r), id, base, target)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, comparisonDTO{
		Base: runRef(b), Target: runRef(t),
		Introduced: mapSlice(cmp.Introduced, toStoredConflictDTO), Resolved: mapSlice(cmp.Resolved, toStoredConflictDTO),
		Unchanged: mapSlice(cmp.Unchanged, toStoredConflictDTO), PMDiff: toPMDiffDTO(cmp.PMDiff), StoryChanged: cmp.StoryChanged,
	})
}

// ---------- Job ----------

func (s *Server) startJob(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	job, err := s.Analysis.StartJob(r.Context(), actorFrom(r), id, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, toJobDTO(job))
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	jobs, err := s.Analysis.Jobs(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(jobs, toJobDTO))
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	jobID, ok := pathID(w, r, "jobID", "job_not_found", "Job not found")
	if !ok {
		return
	}
	job, err := s.Analysis.Job(r.Context(), actorFrom(r), id, jobID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toJobDTO(job))
}

// ---------- Báo cáo ----------

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	rep, err := s.Reports.Report(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toReportDTO(rep))
}

func (s *Server) reportExcel(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	data, name, err := s.Reports.ReportExcel(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	filename := "reqsentinel-report-" + name + ".xlsx"
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="report.xlsx"; filename*=UTF-8''`+url.PathEscape(filename))
	w.Write(data)
}

func (s *Server) coverage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	cov, err := s.Reports.Coverage(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toCoverageDTO(cov))
}

func (s *Server) examples(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"examples": []map[string]any{
			{
				"id": "normal", "label": "Normal — consistent",
				"user_story":          "As a Sales Staff, I want to view assigned customers so that I can manage my accounts.",
				"acceptance_criteria": []string{"System displays only customers assigned to the logged-in staff."},
				"permission_matrix": []map[string]any{
					{"role": "Sales Staff", "action": "view", "resource": "Customer", "effect": "Allow", "condition": nil, "scope": "assigned"},
				},
			},
			{
				"id": "effect", "label": "Effect Conflict",
				"user_story":          "As a Guest, I want to view product catalog so that I can browse items.",
				"acceptance_criteria": []string{},
				"permission_matrix": []map[string]any{
					{"role": "Guest", "action": "view", "resource": "Product Catalog", "effect": "Deny", "condition": nil, "scope": "all"},
				},
			},
		},
	})
}
