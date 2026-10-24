// Package http là adapter REST (/api/v1): parse request → gọi use case → map sang DTO.
// Không chứa logic nghiệp vụ.
package http

import (
	"context"
	"log/slog"
	"net/http"

	"reqsentinel/api/internal/app/admin"
	"reqsentinel/api/internal/app/analysis"
	"reqsentinel/api/internal/app/auth"
	"reqsentinel/api/internal/app/conflicts"
	"reqsentinel/api/internal/app/projects"
	"reqsentinel/api/internal/app/reports"
	"reqsentinel/api/internal/app/requirements"
	"reqsentinel/api/internal/app/users"
)

type Config struct {
	AppName, AppVersion string
	CORSAllowedOrigins  []string
	TrustProxy          bool
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type MLHealth interface {
	Health(ctx context.Context) (map[string]any, error)
}

type Services struct {
	Auth         *auth.Service
	Users        *users.Service
	Projects     *projects.Service
	Requirements *requirements.Service
	Analysis     *analysis.Service
	Conflicts    *conflicts.Service
	Reports      *reports.Service
	Admin        *admin.Service
}

type Server struct {
	cfg Config
	log *slog.Logger
	db  Pinger
	ml  MLHealth
	Services
	auth *auth.Service
}

func NewServer(cfg Config, svc Services, db Pinger, ml MLHealth, log *slog.Logger) *Server {
	return &Server{cfg: cfg, log: log, db: db, ml: ml, Services: svc, auth: svc.Auth}
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	writeError(w, r, s.log, err)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	const v1 = "/api/v1"
	h := func(pattern string, fn http.HandlerFunc) { mux.HandleFunc(pattern, fn) }
	p := func(method, path string) string { return method + " " + v1 + path }

	// Public
	h(p("GET", "/health"), s.health)
	h(p("POST", "/auth/register"), s.register)
	h(p("POST", "/auth/login"), s.login)
	h(p("POST", "/auth/forgot-password"), s.forgotPassword)
	h(p("POST", "/auth/reset-password"), s.resetPassword)
	h(p("GET", "/invitations/{token}"), s.previewInvitation)
	h(p("GET", "/projects/template/excel"), s.excelTemplate)
	h(p("GET", "/check/examples"), s.examples)

	// Tài khoản
	h(p("GET", "/auth/me"), s.authed(s.me))
	h(p("PATCH", "/auth/me"), s.authed(s.updateMe))
	h(p("POST", "/auth/change-password"), s.authed(s.changePassword))
	h(p("POST", "/invitations/{token}/accept"), s.authed(s.acceptInvitation))

	// Project & thành viên
	h(p("GET", "/projects"), s.authed(s.listProjects))
	h(p("POST", "/projects"), s.authed(s.createProject))
	h(p("GET", "/projects/{id}"), s.authed(s.getProject))
	h(p("PATCH", "/projects/{id}"), s.authed(s.updateProject))
	h(p("DELETE", "/projects/{id}"), s.authed(s.deleteProject))
	h(p("GET", "/projects/{id}/members"), s.authed(s.listMembers))
	h(p("PATCH", "/projects/{id}/members/{userID}"), s.authed(s.changeMemberRole))
	h(p("DELETE", "/projects/{id}/members/{userID}"), s.authed(s.removeMember))
	h(p("GET", "/projects/{id}/invitations"), s.authed(s.listInvitations))
	h(p("POST", "/projects/{id}/invitations"), s.authed(s.invite))
	h(p("DELETE", "/projects/{id}/invitations/{invitationID}"), s.authed(s.revokeInvitation))

	// Yêu cầu: user story, PM, từ điển, phiên bản PM, import
	h(p("POST", "/projects/{id}/import/excel"), s.authed(s.importExcel))
	h(p("POST", "/projects/{id}/user-stories"), s.authed(s.addStory))
	h(p("PATCH", "/projects/{id}/user-stories/{storyID}"), s.authed(s.updateStory))
	h(p("DELETE", "/projects/{id}/user-stories/{storyID}"), s.authed(s.deleteStory))
	h(p("POST", "/projects/{id}/permission-matrix"), s.authed(s.addPMRow))
	h(p("PATCH", "/projects/{id}/permission-matrix/{rowID}"), s.authed(s.updatePMRow))
	h(p("DELETE", "/projects/{id}/permission-matrix/{rowID}"), s.authed(s.deletePMRow))
	h(p("GET", "/projects/{id}/terms"), s.authed(s.listTerms))
	h(p("POST", "/projects/{id}/terms"), s.authed(s.createTerm))
	h(p("GET", "/projects/{id}/terms/suggestions"), s.authed(s.termSuggestions))
	h(p("PUT", "/projects/{id}/terms/{termID}"), s.authed(s.updateTerm))
	h(p("DELETE", "/projects/{id}/terms/{termID}"), s.authed(s.deleteTerm))
	h(p("GET", "/projects/{id}/pm-versions"), s.authed(s.listPMVersions))
	h(p("POST", "/projects/{id}/pm-versions"), s.authed(s.savePMVersion))
	h(p("GET", "/projects/{id}/pm-versions/diff"), s.authed(s.diffPMVersions))
	h(p("POST", "/projects/{id}/pm-versions/{versionID}/restore"), s.authed(s.restorePMVersion))

	// Phân tích
	h(p("POST", "/projects/{id}/check"), s.authed(s.checkStory))
	h(p("GET", "/projects/{id}/runs"), s.authed(s.projectRuns))
	h(p("GET", "/projects/{id}/runs/compare"), s.authed(s.compareRuns))
	h(p("GET", "/projects/{id}/analysis-jobs"), s.authed(s.listJobs))
	h(p("POST", "/projects/{id}/analysis-jobs"), s.authed(s.startJob))
	h(p("GET", "/projects/{id}/analysis-jobs/{jobID}"), s.authed(s.getJob))
	h(p("POST", "/check"), s.authed(s.adhocCheck))
	h(p("GET", "/check/history"), s.authed(s.history))
	h(p("GET", "/check/history/{runID}"), s.authed(s.runDetail))

	// Xử lý conflict
	h(p("GET", "/projects/{id}/conflicts"), s.authed(s.listIssues))
	h(p("GET", "/projects/{id}/conflicts/{issueID}"), s.authed(s.getIssue))
	h(p("PATCH", "/projects/{id}/conflicts/{issueID}"), s.authed(s.updateIssue))
	h(p("POST", "/projects/{id}/conflicts/{issueID}/comments"), s.authed(s.commentIssue))

	// Báo cáo
	h(p("GET", "/projects/{id}/report"), s.authed(s.report))
	h(p("GET", "/projects/{id}/report/excel"), s.authed(s.reportExcel))
	h(p("GET", "/projects/{id}/coverage"), s.authed(s.coverage))

	// Admin
	h(p("GET", "/admin/stats"), s.adminOnly(s.adminStats))
	h(p("GET", "/admin/users"), s.adminOnly(s.adminListUsers))
	h(p("POST", "/admin/users"), s.adminOnly(s.adminCreateUser))
	h(p("PATCH", "/admin/users/{userID}"), s.adminOnly(s.adminUpdateUser))
	h(p("POST", "/admin/users/{userID}/reset-password"), s.adminOnly(s.adminResetPassword))
	h(p("GET", "/admin/audit-logs"), s.adminOnly(s.adminAuditLogs))
	h(p("GET", "/admin/dataset/export"), s.adminOnly(s.adminDatasetExport))

	h("/", func(w http.ResponseWriter, r *http.Request) {
		writeDetail(w, http.StatusNotFound, "not_found", "Not found")
	})

	return s.recoverer(s.requestLogger(s.cors(mux)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{"status": "ok", "service": s.cfg.AppName, "version": s.cfg.AppVersion, "database": "ok"}
	if err := s.db.Ping(r.Context()); err != nil {
		resp["status"], resp["database"] = "degraded", "unavailable"
	}
	if ml, err := s.ml.Health(r.Context()); err != nil {
		resp["status"], resp["ml_service"] = "degraded", "unavailable"
	} else {
		resp["ml_service"] = ml
	}
	writeJSON(w, http.StatusOK, resp)
}
