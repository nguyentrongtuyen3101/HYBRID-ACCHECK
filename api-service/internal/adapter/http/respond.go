package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// errorBody: {"detail": câu tiếng Anh, "code": mã để client dịch, "params": tham số bản dịch}.
type errorBody struct {
	Detail string         `json:"detail"`
	Code   string         `json:"code"`
	Params map[string]any `json:"params,omitempty"`
}

func writeDetail(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, errorBody{Detail: detail, Code: code})
}

var kindStatus = map[domain.ErrorKind]int{
	domain.KindValidation:   http.StatusUnprocessableEntity,
	domain.KindNotFound:     http.StatusNotFound,
	domain.KindForbidden:    http.StatusForbidden,
	domain.KindConflict:     http.StatusConflict,
	domain.KindUnauthorized: http.StatusUnauthorized,
	domain.KindUnavailable:  http.StatusServiceUnavailable,
	domain.KindRateLimited:  http.StatusTooManyRequests,
}

// writeError map lỗi nghiệp vụ sang HTTP status; lỗi không phân loại → 500 (không lộ chi tiết).
func writeError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	var de *domain.Error
	if errors.As(err, &de) {
		if status, ok := kindStatus[de.Kind]; ok {
			if de.Kind == domain.KindUnavailable {
				log.Error("dependency unavailable", "path", r.URL.Path, "err", err)
			}
			writeJSON(w, status, errorBody{Detail: de.Msg, Code: de.Code, Params: de.Params})
			return
		}
	}
	log.Error("internal error", "method", r.Method, "path", r.URL.Path, "err", err)
	writeDetail(w, http.StatusInternalServerError, "internal_error", "Internal server error")
}

const maxJSONBody = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			writeDetail(w, http.StatusUnprocessableEntity, "body_required", "Request body is required")
		} else {
			writeDetail(w, http.StatusUnprocessableEntity, "invalid_json", "Invalid JSON body: "+err.Error())
		}
		return false
	}
	return true
}

// pathID đọc path param dạng UUID; sai định dạng → 404 thay vì lỗi SQL.
func pathID(w http.ResponseWriter, r *http.Request, name, code, notFoundMsg string) (string, bool) {
	id := r.PathValue(name)
	if !validUUID(id) {
		writeDetail(w, http.StatusNotFound, code, notFoundMsg)
		return "", false
	}
	return id, true
}

func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

func queryInt(r *http.Request, key string, def, min, max int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return clamp(v, min, max)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (s *Server) meta(r *http.Request) app.RequestMeta {
	return app.RequestMeta{IP: s.clientIP(r)}
}

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
