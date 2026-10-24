package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"reqsentinel/api/internal/app/conflicts"
	"reqsentinel/api/internal/domain"
)

func (s *Server) listIssues(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	q := r.URL.Query()
	f := domain.IssueFilter{ProjectID: id, Status: q.Get("status"), Type: q.Get("type"),
		Limit: queryInt(r, "limit", 50, 1, 200), Offset: queryInt(r, "offset", 0, 0, 1_000_000)}
	for key, dst := range map[string]*string{"assignee_id": &f.AssigneeID, "user_story_id": &f.UserStoryID} {
		if v := q.Get(key); v != "" {
			if v == "me" && key == "assignee_id" {
				v = actorFrom(r).ID
			}
			if !validUUID(v) {
				writeDetail(w, http.StatusUnprocessableEntity, "invalid_param", "Invalid "+key)
				return
			}
			*dst = v
		}
	}
	items, total, err := s.Conflicts.List(r.Context(), actorFrom(r), f)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapSlice(items, toIssueDTO), "total": total})
}

func (s *Server) getIssue(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	issueID, ok := pathID(w, r, "issueID", "conflict_not_found", "Conflict not found")
	if !ok {
		return
	}
	d, err := s.Conflicts.Get(r.Context(), actorFrom(r), id, issueID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issue": toIssueDTO(d.Issue), "activities": mapSlice(d.Activities, toActivityDTO)})
}

func (s *Server) updateIssue(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	issueID, ok := pathID(w, r, "issueID", "conflict_not_found", "Conflict not found")
	if !ok {
		return
	}
	// raw để phân biệt "không gửi assignee_id" với "assignee_id: null" (bỏ giao)
	var body map[string]json.RawMessage
	if !decodeJSON(w, r, &body) {
		return
	}
	var in conflicts.UpdateInput
	if raw, ok := body["status"]; ok {
		var status string
		if err := json.Unmarshal(raw, &status); err != nil || !domain.IssueStatus(status).Valid() {
			writeDetail(w, http.StatusUnprocessableEntity, "invalid_issue_status", "status must be open, acknowledged, resolved or false_positive")
			return
		}
		st := domain.IssueStatus(status)
		in.Status = &st
	}
	if raw, ok := body["assignee_id"]; ok {
		in.AssigneeSet = true
		if strings.TrimSpace(string(raw)) != "null" {
			var assignee string
			if err := json.Unmarshal(raw, &assignee); err != nil || !validUUID(assignee) {
				writeDetail(w, http.StatusUnprocessableEntity, "invalid_assignee", "Invalid assignee_id")
				return
			}
			in.AssigneeID = &assignee
		}
	}
	iss, err := s.Conflicts.Update(r.Context(), actorFrom(r), id, issueID, in, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toIssueDTO(iss))
}

func (s *Server) commentIssue(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	issueID, ok := pathID(w, r, "issueID", "conflict_not_found", "Conflict not found")
	if !ok {
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	act, err := s.Conflicts.Comment(r.Context(), actorFrom(r), id, issueID, body.Body)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toActivityDTO(act))
}
