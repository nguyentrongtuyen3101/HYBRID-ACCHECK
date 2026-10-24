package http

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"reqsentinel/api/internal/domain"
)

type actorKey struct{}

func actorFrom(r *http.Request) domain.Actor {
	a, _ := r.Context().Value(actorKey{}).(domain.Actor)
	return a
}

// authed: yêu cầu Bearer token hợp lệ; use case Authenticate đọc lại user từ DB
// nên khoá tài khoản / đổi role có hiệu lực ngay.
func (s *Server) authed(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeDetail(w, http.StatusUnauthorized, "not_authenticated", "Not authenticated")
			return
		}
		u, err := s.auth.Authenticate(r.Context(), token)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, u.Actor())))
	}
}

func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return s.authed(func(w http.ResponseWriter, r *http.Request) {
		if !actorFrom(r).IsAdmin() {
			writeDetail(w, http.StatusForbidden, "admin_required", "Admin role required")
			return
		}
		next(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.log.Info("http", "method", r.Method, "path", r.URL.Path, "status", rec.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.Error("panic", "path", r.URL.Path, "panic", v)
				writeDetail(w, http.StatusInternalServerError, "internal_error", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// cors chỉ bật khi cấu hình CORS_ALLOWED_ORIGINS (FE gọi qua proxy cùng origin thì không cần).
func (s *Server) cors(next http.Handler) http.Handler {
	if len(s.cfg.CORSAllowedOrigins) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (slices.Contains(s.cfg.CORSAllowedOrigins, "*") || slices.Contains(s.cfg.CORSAllowedOrigins, origin)) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
