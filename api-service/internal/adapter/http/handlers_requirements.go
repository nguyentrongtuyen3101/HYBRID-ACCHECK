package http

import (
	"net/http"
	"strings"

	"reqsentinel/api/internal/domain"
)

const maxExcelSize = 10 << 20

func (s *Server) excelTemplate(w http.ResponseWriter, r *http.Request) {
	data, err := s.Requirements.ImportTemplate()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="reqsentinel_import_template.xlsx"`)
	w.Write(data)
}

func (s *Server) importExcel(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxExcelSize+1<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeDetail(w, http.StatusBadRequest, "file_missing", "Missing file (max 10MB)")
		return
	}
	defer file.Close()
	name := strings.ToLower(header.Filename)
	if !strings.HasSuffix(name, ".xlsx") && !strings.HasSuffix(name, ".xlsm") {
		writeDetail(w, http.StatusBadRequest, "file_type", "Only .xlsx files are accepted")
		return
	}
	if header.Size == 0 {
		writeDetail(w, http.StatusBadRequest, "file_empty", "File is empty")
		return
	}
	res, err := s.Requirements.Import(r.Context(), actorFrom(r), id, file, s.meta(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project_id": id, "user_stories_imported": res.Stories, "acceptance_criteria_imported": res.Criteria,
		"pm_rows_imported": res.PMRows, "warnings": res.Warnings,
	})
}

// ---------- User stories ----------

func (s *Server) addStory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	var body struct {
		Content            string   `json:"content"`
		AcceptanceCriteria []string `json:"acceptance_criteria"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	us, err := s.Requirements.AddStory(r.Context(), actorFrom(r), id, body.Content, body.AcceptanceCriteria)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toStoryDTO(us))
}

func (s *Server) updateStory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	storyID, ok := pathID(w, r, "storyID", "story_not_found", "User story not found")
	if !ok {
		return
	}
	var body struct {
		Content            *string   `json:"content"`
		AcceptanceCriteria *[]string `json:"acceptance_criteria"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	us, err := s.Requirements.UpdateStory(r.Context(), actorFrom(r), id, storyID, body.Content, body.AcceptanceCriteria)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toStoryDTO(us))
}

func (s *Server) deleteStory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	storyID, ok := pathID(w, r, "storyID", "story_not_found", "User story not found")
	if !ok {
		return
	}
	if err := s.Requirements.DeleteStory(r.Context(), actorFrom(r), id, storyID); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Permission matrix ----------

func (s *Server) addPMRow(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	var body pmRowDTO
	if !decodeJSON(w, r, &body) {
		return
	}
	row, err := s.Requirements.AddPMRow(r.Context(), actorFrom(r), id, domain.PMRow{Role: body.Role, Action: body.Action,
		Resource: body.Resource, Effect: domain.Effect(body.Effect), Condition: body.Condition, Scope: body.Scope})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPMRowDTO(row))
}

func (s *Server) updatePMRow(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	rowID, ok := pathID(w, r, "rowID", "pm_row_not_found", "PM row not found")
	if !ok {
		return
	}
	// map để phân biệt "không gửi" và "gửi null" (xoá condition/scope)
	var body map[string]*string
	if !decodeJSON(w, r, &body) {
		return
	}
	patch := domain.PMRowPatch{Role: nonEmpty(body["role"]), Action: nonEmpty(body["action"]), Resource: nonEmpty(body["resource"])}
	if e := nonEmpty(body["effect"]); e != nil {
		effect := domain.Effect(*e)
		patch.Effect = &effect
	}
	if v, present := body["condition"]; present {
		patch.Condition = domain.OptionalString{Set: true, Value: v}
	}
	if v, present := body["scope"]; present {
		patch.Scope = domain.OptionalString{Set: true, Value: v}
	}
	row, err := s.Requirements.UpdatePMRow(r.Context(), actorFrom(r), id, rowID, patch)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPMRowDTO(row))
}

func nonEmpty(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	t := strings.TrimSpace(*v)
	return &t
}

func (s *Server) deletePMRow(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	rowID, ok := pathID(w, r, "rowID", "pm_row_not_found", "PM row not found")
	if !ok {
		return
	}
	if err := s.Requirements.DeletePMRow(r.Context(), actorFrom(r), id, rowID); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Từ điển ----------

type termBody struct {
	Kind        string   `json:"kind"`
	Canonical   string   `json:"canonical"`
	Synonyms    []string `json:"synonyms"`
	Description *string  `json:"description"`
}

func (s *Server) listTerms(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	terms, err := s.Requirements.Terms(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(terms, toTermDTO))
}

func (s *Server) saveTerm(w http.ResponseWriter, r *http.Request, projectID, termID string, status int) {
	var body termBody
	if !decodeJSON(w, r, &body) {
		return
	}
	t, err := s.Requirements.SaveTerm(r.Context(), actorFrom(r), domain.Term{ID: termID, ProjectID: projectID,
		Kind: domain.TermKind(body.Kind), Canonical: body.Canonical, Synonyms: body.Synonyms, Description: body.Description})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, status, toTermDTO(t))
}

func (s *Server) createTerm(w http.ResponseWriter, r *http.Request) {
	if id, ok := pathID(w, r, "id", "project_not_found", "Project not found"); ok {
		s.saveTerm(w, r, id, "", http.StatusCreated)
	}
}

func (s *Server) updateTerm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	if termID, ok := pathID(w, r, "termID", "term_not_found", "Term not found"); ok {
		s.saveTerm(w, r, id, termID, http.StatusOK)
	}
}

func (s *Server) deleteTerm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	termID, ok := pathID(w, r, "termID", "term_not_found", "Term not found")
	if !ok {
		return
	}
	if err := s.Requirements.DeleteTerm(r.Context(), actorFrom(r), id, termID); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) termSuggestions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	items, err := s.Requirements.TermSuggestions(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	type suggestionDTO struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
		Count int    `json:"count"`
	}
	writeJSON(w, http.StatusOK, mapSlice(items, func(t domain.TermSuggestion) suggestionDTO {
		return suggestionDTO{string(t.Kind), t.Value, t.Count}
	}))
}

// ---------- Phiên bản PM ----------

func (s *Server) listPMVersions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	versions, err := s.Requirements.PMVersions(r.Context(), actorFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(versions, toPMVersionDTO))
}

func (s *Server) savePMVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	var body struct {
		Label *string `json:"label"`
	}
	if r.ContentLength != 0 && !decodeJSON(w, r, &body) {
		return
	}
	if body.Label != nil && strings.TrimSpace(*body.Label) == "" {
		body.Label = nil
	}
	v, err := s.Requirements.SaveVersion(r.Context(), actorFrom(r), id, body.Label)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPMVersionDTO(v))
}

func (s *Server) diffPMVersions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if !validUUID(from) || (to != "" && !validUUID(to)) {
		writeDetail(w, http.StatusUnprocessableEntity, "invalid_version_ids", "from (and optional to) must be version ids")
		return
	}
	d, err := s.Requirements.DiffPMVersions(r.Context(), actorFrom(r), id, from, to)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPMDiffDTO(d))
}

func (s *Server) restorePMVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id", "project_not_found", "Project not found")
	if !ok {
		return
	}
	versionID, ok := pathID(w, r, "versionID", "version_not_found", "Version not found")
	if !ok {
		return
	}
	if err := s.Requirements.RestoreVersion(r.Context(), actorFrom(r), id, versionID, s.meta(r)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
