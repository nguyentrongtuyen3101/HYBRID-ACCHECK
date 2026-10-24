package http

import (
	"encoding/json"
	"net/http"
	"time"

	"reqsentinel/api/internal/app/auth"
	"reqsentinel/api/internal/app/users"
	"reqsentinel/api/internal/domain"
)

type sessionDTO struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        userDTO   `json:"user"`
}

func toSessionDTO(s auth.Session) sessionDTO {
	return sessionDTO{s.Token, "bearer", s.ExpiresAt, toUserDTO(s.User)}
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	sess, err := s.Auth.Register(r.Context(), auth.RegisterInput{Email: body.Email, Password: body.Password,
		FullName: body.FullName}, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSessionDTO(sess))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	sess, err := s.Auth.Login(r.Context(), body.Email, body.Password, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSessionDTO(sess))
}

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.Auth.ForgotPassword(r.Context(), body.Email, s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.Auth.ResetPassword(r.Context(), body.Token, body.NewPassword, s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, err := s.Auth.Me(r.Context(), actorFrom(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(u))
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	// raw để phân biệt "không gửi" với "gửi null" (locale / theme = null → theo trình duyệt / hệ điều hành)
	var body map[string]json.RawMessage
	if !decodeJSON(w, r, &body) {
		return
	}
	var p domain.ProfilePatch
	fields := map[string]**string{"full_name": &p.FullName, "job_title": &p.JobTitle,
		"organization": &p.Organization, "phone": &p.Phone, "bio": &p.Bio}
	optional := map[string]*domain.OptionalString{"locale": &p.Locale, "theme": &p.Theme}
	for key, raw := range body {
		var value *string
		if string(raw) != "null" {
			value = new(string)
			if json.Unmarshal(raw, value) != nil {
				writeDetail(w, http.StatusUnprocessableEntity, "invalid_param", "Invalid "+key)
				return
			}
		}
		if dst, ok := fields[key]; ok {
			*dst = value
		} else if dst, ok := optional[key]; ok {
			*dst = domain.OptionalString{Set: true, Value: value}
		}
	}
	u, err := s.Auth.UpdateProfile(r.Context(), actorFrom(r), p)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(u))
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.Auth.ChangePassword(r.Context(), actorFrom(r), body.CurrentPassword, body.NewPassword, s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Admin: user ----------

func (s *Server) adminListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pageSize := queryInt(r, "page_size", 20, 1, 100)
	page := queryInt(r, "page", 1, 1, 1_000_000)
	items, total, err := s.Users.List(r.Context(), domain.UserFilter{Query: q.Get("q"), Role: q.Get("role"),
		Status: q.Get("status"), Limit: pageSize, Offset: (page - 1) * pageSize})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapSlice(items, toUserDTO), "total": total,
		"page": page, "page_size": pageSize})
}

func (s *Server) adminCreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	u, err := s.Users.Create(r.Context(), actorFrom(r), users.CreateInput{Email: body.Email, Password: body.Password,
		FullName: body.FullName, Role: domain.SystemRole(body.Role)}, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserDTO(u))
}

func (s *Server) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "userID", "user_not_found", "User not found")
	if !ok {
		return
	}
	var body struct {
		FullName *string `json:"full_name"`
		Role     *string `json:"role"`
		Status   *string `json:"status"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	patch := domain.UserPatch{FullName: body.FullName}
	if body.Role != nil {
		role := domain.SystemRole(*body.Role)
		patch.Role = &role
	}
	if body.Status != nil {
		status := domain.UserStatus(*body.Status)
		patch.Status = &status
	}
	u, err := s.Users.Update(r.Context(), actorFrom(r), id, patch, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(u))
}

func (s *Server) adminResetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "userID", "user_not_found", "User not found")
	if !ok {
		return
	}
	var body struct {
		NewPassword string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.Users.ResetPassword(r.Context(), actorFrom(r), id, body.NewPassword, s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
