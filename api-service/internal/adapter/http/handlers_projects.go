package http

import (
	"net/http"

	"reqsentinel/api/internal/domain"
)

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.Projects.List(r.Context(), actorFrom(r), r.URL.Query().Get("mine") == "true")
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(items, toProjectDTO))
}

type projectBody struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var body projectBody
	if !decodeJSON(w, r, &body) {
		return
	}
	name := ""
	if body.Name != nil {
		name = *body.Name
	}
	p, err := s.Projects.Create(r.Context(), actorFrom(r), name, body.Description, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectDTO(p))
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	d, err := s.Projects.Get(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project":           toProjectDTO(d.Project),
		"user_stories":      mapSlice(d.Stories, toStoryDTO),
		"permission_matrix": toPMRowDTOs(d.PM),
	})
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	var body projectBody
	if !decodeJSON(w, r, &body) {
		return
	}
	p, err := s.Projects.Update(r.Context(), actorFrom(r), id, body.Name, body.Description, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectDTO(p))
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	if err := s.Projects.Delete(r.Context(), actorFrom(r), id, s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Thành viên ----------

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	members, err := s.Projects.Members(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(members, toMemberDTO))
}

func (s *Server) changeMemberRole(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	userID, ok := pathID(w, r, "userID", "member_not_found", "Member not found")
	if !ok {
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.Projects.ChangeMemberRole(r.Context(), actorFrom(r), id, userID, domain.ProjectRole(body.Role), s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	userID, ok := pathID(w, r, "userID", "member_not_found", "Member not found")
	if !ok {
		return
	}
	if err := s.Projects.RemoveMember(r.Context(), actorFrom(r), id, userID, s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Lời mời ----------

func (s *Server) listInvitations(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	items, err := s.Projects.PendingInvitations(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(items, toInvitationDTO))
}

func (s *Server) invite(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	inv, err := s.Projects.Invite(r.Context(), actorFrom(r), id, body.Email, domain.ProjectRole(body.Role), s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toInvitationDTO(inv))
}

func (s *Server) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	invID, ok := pathID(w, r, "invitationID", "invitation_not_found", "Invitation not found")
	if !ok {
		return
	}
	if err := s.Projects.RevokeInvitation(r.Context(), actorFrom(r), id, invID, s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) previewInvitation(w http.ResponseWriter, r *http.Request) {
	inv, err := s.Projects.PreviewInvitation(r.Context(), r.PathValue("token"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toInvitationDTO(inv))
}

func (s *Server) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	p, err := s.Projects.AcceptInvitation(r.Context(), actorFrom(r), r.PathValue("token"), s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectDTO(p))
}
