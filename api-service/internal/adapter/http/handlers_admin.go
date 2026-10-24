package http

import (
	"net/http"
	"strings"

	"reqsentinel/api/internal/domain"
)

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.Admin.Stats(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users": map[string]int{"total": st.UsersTotal, "active": st.UsersActive, "disabled": st.UsersDisabled,
			"admins": st.Admins},
		"projects": st.Projects,
		"runs": map[string]int{"total": st.RunsTotal, "last_7_days": st.RunsLast7Days,
			"consistent": st.RunsConsistent, "inconsistent": st.RunsTotal - st.RunsConsistent},
		"open_issues":       st.OpenIssues,
		"conflicts_by_type": st.ConflictsByType,
		"runs_by_model":     st.RunsByModel,
		"runs_per_day": mapSlice(st.RunsPerDay, func(d domain.DailyCount) map[string]any {
			return map[string]any{"day": d.Day, "count": d.Count}
		}),
	})
}

func (s *Server) adminAuditLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.AuditFilter{Action: strings.TrimSpace(q.Get("action")),
		Limit: queryInt(r, "limit", 50, 1, 200), Offset: queryInt(r, "offset", 0, 0, 1_000_000)}
	if actor := q.Get("actor_id"); actor != "" {
		if !validUUID(actor) {
			writeDetail(w, http.StatusUnprocessableEntity, "invalid_param", "Invalid actor_id")
			return
		}
		f.ActorID = &actor
	}
	items, total, err := s.Admin.AuditLogs(r.Context(), f)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapSlice(items, toAuditDTO), "total": total})
}

func (s *Server) adminDatasetExport(w http.ResponseWriter, r *http.Request) {
	items, err := s.Admin.DatasetExport(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="reqsentinel_feedback_dataset.json"`)
	writeJSON(w, http.StatusOK, mapSlice(items, toDatasetItemDTO))
}
