package http

import (
	"time"

	"reqsentinel/api/internal/domain"
)

// DTO: hình dạng JSON trả về cho client, tách khỏi entity domain.

type userDTO struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	FullName     string     `json:"full_name"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	JobTitle     *string    `json:"job_title"`
	Organization *string    `json:"organization"`
	Phone        *string    `json:"phone"`
	Bio          *string    `json:"bio"`
	Locale       *string    `json:"locale"`
	Theme        *string    `json:"theme"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	ProjectCount int        `json:"project_count"`
	RunCount     int        `json:"run_count"`
}

func toUserDTO(u domain.User) userDTO {
	dto := userDTO{ID: u.ID, Email: u.Email, FullName: u.FullName, Role: string(u.Role), Status: string(u.Status),
		JobTitle: u.JobTitle, Organization: u.Organization, Phone: u.Phone, Bio: u.Bio,
		LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt, ProjectCount: u.ProjectCount, RunCount: u.RunCount}
	if u.Locale != nil {
		l := string(*u.Locale)
		dto.Locale = &l
	}
	if u.Theme != nil {
		t := string(*u.Theme)
		dto.Theme = &t
	}
	return dto
}

type projectDTO struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    *string   `json:"description"`
	OwnerID        *string   `json:"owner_id"`
	OwnerEmail     *string   `json:"owner_email"`
	CreatedAt      time.Time `json:"created_at"`
	UserStoryCount int       `json:"user_story_count"`
	PMRowCount     int       `json:"pm_row_count"`
	RunCount       int       `json:"run_count"`
	OpenIssueCount int       `json:"open_issue_count"`
	MyRole         string    `json:"my_role"`
}

func toProjectDTO(p domain.Project) projectDTO {
	return projectDTO{p.ID, p.Name, p.Description, p.OwnerID, p.OwnerEmail, p.CreatedAt, p.UserStoryCount,
		p.PMRowCount, p.RunCount, p.OpenIssueCount, string(p.MyRole)}
}

type criterionDTO struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type storyDTO struct {
	ID                 string         `json:"id"`
	Content            string         `json:"content"`
	Source             string         `json:"source"`
	AcceptanceCriteria []criterionDTO `json:"acceptance_criteria"`
	CreatedAt          time.Time      `json:"created_at"`
}

func toStoryDTO(s domain.UserStory) storyDTO {
	ac := make([]criterionDTO, 0, len(s.AcceptanceCriteria))
	for _, c := range s.AcceptanceCriteria {
		ac = append(ac, criterionDTO{c.ID, c.Content})
	}
	return storyDTO{s.ID, s.Content, s.Source, ac, s.CreatedAt}
}

type pmRowDTO struct {
	ID        string  `json:"id,omitempty"`
	Role      string  `json:"role"`
	Action    string  `json:"action"`
	Resource  string  `json:"resource"`
	Effect    string  `json:"effect"`
	Condition *string `json:"condition"`
	Scope     *string `json:"scope"`
}

func toPMRowDTO(r domain.PMRow) pmRowDTO {
	return pmRowDTO{r.ID, r.Role, r.Action, r.Resource, string(r.Effect), r.Condition, r.Scope}
}

func toPMRowDTOs(rows []domain.PMRow) []pmRowDTO {
	return mapSlice(rows, toPMRowDTO)
}

type memberDTO struct {
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role"`
	AddedAt  time.Time `json:"added_at"`
}

func toMemberDTO(m domain.Member) memberDTO {
	return memberDTO{m.UserID, m.Email, m.FullName, string(m.Role), m.AddedAt}
}

type invitationDTO struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	InviterName string    `json:"inviter_name"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func toInvitationDTO(i domain.Invitation) invitationDTO {
	return invitationDTO{i.ID, i.ProjectID, i.ProjectName, i.Email, string(i.Role), string(i.Status),
		i.InviterName, i.ExpiresAt, i.CreatedAt}
}

type termDTO struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Canonical   string    `json:"canonical"`
	Synonyms    []string  `json:"synonyms"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func toTermDTO(t domain.Term) termDTO {
	syn := t.Synonyms
	if syn == nil {
		syn = []string{}
	}
	return termDTO{t.ID, string(t.Kind), t.Canonical, syn, t.Description, t.CreatedAt}
}

type pmVersionDTO struct {
	ID        string     `json:"id"`
	Version   int        `json:"version"`
	Label     *string    `json:"label"`
	RowCount  int        `json:"row_count"`
	Rows      []pmRowDTO `json:"rows,omitempty"`
	CreatedBy *string    `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
}

func toPMVersionDTO(v domain.PMVersion) pmVersionDTO {
	return pmVersionDTO{ID: v.ID, Version: v.Version, Label: v.Label, RowCount: len(v.Rows), CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt}
}

type pmChangeDTO struct {
	Before pmRowDTO `json:"before"`
	After  pmRowDTO `json:"after"`
}

type pmDiffDTO struct {
	Added   []pmRowDTO    `json:"added"`
	Removed []pmRowDTO    `json:"removed"`
	Changed []pmChangeDTO `json:"changed"`
}

func toPMDiffDTO(d domain.PMDiff) pmDiffDTO {
	return pmDiffDTO{toPMRowDTOs(d.Added), toPMRowDTOs(d.Removed), mapSlice(d.Changed, func(c domain.PMRowChange) pmChangeDTO {
		return pmChangeDTO{toPMRowDTO(c.Before), toPMRowDTO(c.After)}
	})}
}

// ---------- Phân tích ----------

type entityDTO struct {
	Role       *string `json:"role"`
	Action     *string `json:"action"`
	Resource   *string `json:"resource"`
	Effect     *string `json:"effect"`
	Condition  *string `json:"condition"`
	Scope      *string `json:"scope"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
}

func toEntityDTO(e domain.ExtractedEntity) entityDTO {
	return entityDTO{e.Role, e.Action, e.Resource, e.Effect, e.Condition, e.Scope, e.Source, e.Confidence}
}

type alignmentDTO struct {
	Extracted       entityDTO `json:"extracted"`
	MatchedPMRow    *pmRowDTO `json:"matched_pm_row"`
	SimilarityScore float64   `json:"similarity_score"`
	IsMatched       bool      `json:"is_matched"`
}

func toAlignmentDTO(a domain.Alignment) alignmentDTO {
	dto := alignmentDTO{Extracted: toEntityDTO(a.Extracted), SimilarityScore: a.Score, IsMatched: a.IsMatched}
	if a.Matched != nil {
		m := toPMRowDTO(*a.Matched)
		dto.MatchedPMRow = &m
	}
	return dto
}

type conflictDTO struct {
	IssueID         *string     `json:"issue_id"`
	ConflictType    string      `json:"conflict_type"`
	Confidence      float64     `json:"confidence"`
	Explanation     string      `json:"explanation"`
	EvidenceUSAC    *string     `json:"evidence_us_ac"`
	EvidencePM      *string     `json:"evidence_pm"`
	RelatedEntities []entityDTO `json:"related_entities,omitempty"`
}

// analysisDTO giữ đúng hình dạng PipelineOutput cũ.
type analysisDTO struct {
	PipelineRunID     string         `json:"pipeline_run_id"`
	ExtractedEntities []entityDTO    `json:"extracted_entities"`
	AlignmentResults  []alignmentDTO `json:"alignment_results"`
	Conflicts         []conflictDTO  `json:"conflicts"`
	IsConsistent      bool           `json:"is_consistent"`
	Summary           string         `json:"summary"`
	ModelVersion      string         `json:"model_version"`
}

func toAnalysisDTO(run domain.Run) analysisDTO {
	res := run.Result
	dto := analysisDTO{PipelineRunID: run.ID, IsConsistent: res.IsConsistent, Summary: res.Summary,
		ModelVersion: res.ModelVersion, ExtractedEntities: mapSlice(res.Entities, toEntityDTO),
		AlignmentResults: mapSlice(res.Alignments, toAlignmentDTO), Conflicts: []conflictDTO{}}
	real := 0
	for _, c := range res.Conflicts {
		cd := conflictDTO{ConflictType: string(c.Type), Confidence: c.Confidence, Explanation: c.Explanation,
			EvidenceUSAC: c.EvidenceUSAC, EvidencePM: c.EvidencePM, RelatedEntities: mapSlice(c.Related, toEntityDTO)}
		if c.Type != domain.ConflictNormal {
			if real < len(run.IssueIDs) && run.IssueIDs[real] != "" {
				cd.IssueID = &run.IssueIDs[real]
			}
			real++
		}
		dto.Conflicts = append(dto.Conflicts, cd)
	}
	return dto
}

type runSummaryDTO struct {
	ID            string    `json:"id"`
	ProjectID     *string   `json:"project_id"`
	ProjectName   *string   `json:"project_name"`
	UserStoryID   *string   `json:"user_story_id"`
	UserStory     string    `json:"user_story"`
	IsConsistent  bool      `json:"is_consistent"`
	Summary       string    `json:"summary"`
	ConflictCount int       `json:"conflict_count"`
	ConflictTypes []string  `json:"conflict_types"`
	ModelVersion  *string   `json:"model_version"`
	CreatedBy     *string   `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

func toRunSummaryDTO(r domain.RunSummary) runSummaryDTO {
	types := mapSlice(r.ConflictTypes, func(t domain.ConflictType) string { return string(t) })
	return runSummaryDTO{r.ID, r.ProjectID, r.ProjectName, r.UserStoryID, r.UserStory, r.IsConsistent, r.Summary,
		len(types), types, r.ModelVersion, r.CreatedBy, r.CreatedAt}
}

func toStoredConflictDTO(c domain.StoredConflict) conflictDTO {
	return conflictDTO{IssueID: c.IssueID, ConflictType: string(c.Type), Confidence: c.Confidence,
		Explanation: c.Explanation, EvidenceUSAC: c.EvidenceUSAC, EvidencePM: c.EvidencePM}
}

type runDetailDTO struct {
	ID                 string         `json:"id"`
	ProjectID          *string        `json:"project_id"`
	UserStoryID        *string        `json:"user_story_id"`
	PMVersionID        *string        `json:"pm_version_id"`
	JobID              *string        `json:"job_id"`
	CreatedBy          *string        `json:"created_by"`
	UserStory          string         `json:"user_story"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
	PermissionMatrix   []pmRowDTO     `json:"permission_matrix"`
	AlignmentResults   []alignmentDTO `json:"alignment_results"`
	IsConsistent       bool           `json:"is_consistent"`
	Summary            *string        `json:"summary"`
	ModelVersion       *string        `json:"model_version"`
	Conflicts          []conflictDTO  `json:"conflicts"`
	ExtractedEntities  []entityDTO    `json:"extracted_entities"`
	CreatedAt          time.Time      `json:"created_at"`
}

func toRunDetailDTO(d domain.RunDetail) runDetailDTO {
	return runDetailDTO{d.ID, d.ProjectID, d.UserStoryID, d.PMVersionID, d.JobID, d.CreatedBy, d.UserStory,
		d.AcceptanceCriteria, toPMRowDTOs(d.PermissionMatrix), mapSlice(d.Alignments, toAlignmentDTO),
		d.IsConsistent, d.Summary, d.ModelVersion, mapSlice(d.Conflicts, toStoredConflictDTO),
		mapSlice(d.Entities, toEntityDTO), d.CreatedAt}
}

type jobDTO struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	Status       string     `json:"status"`
	Total        int        `json:"total"`
	Processed    int        `json:"processed"`
	Failed       int        `json:"failed"`
	Error        *string    `json:"error"`
	ModelVersion *string    `json:"model_version"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

func toJobDTO(j domain.AnalysisJob) jobDTO {
	return jobDTO{j.ID, j.ProjectID, string(j.Status), j.Total, j.Processed, j.Failed, j.Error, j.ModelVersion,
		j.CreatedBy, j.CreatedAt, j.StartedAt, j.FinishedAt}
}

// ---------- Conflict issue ----------

type issueDTO struct {
	ID            string     `json:"id"`
	UserStoryID   string     `json:"user_story_id"`
	UserStory     string     `json:"user_story"`
	ConflictType  string     `json:"conflict_type"`
	Explanation   string     `json:"explanation"`
	EvidenceUSAC  *string    `json:"evidence_us_ac"`
	EvidencePM    *string    `json:"evidence_pm"`
	Role          *string    `json:"role"`
	Action        *string    `json:"action"`
	Resource      *string    `json:"resource"`
	Status        string     `json:"status"`
	AutoResolved  bool       `json:"auto_resolved"`
	AssigneeID    *string    `json:"assignee_id"`
	AssigneeEmail *string    `json:"assignee_email"`
	AssigneeName  *string    `json:"assignee_name"`
	LastSeenRunID string     `json:"last_seen_run_id"`
	LastSeenAt    time.Time  `json:"last_seen_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	CommentCount  int        `json:"comment_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func toIssueDTO(i domain.Issue) issueDTO {
	return issueDTO{i.ID, i.UserStoryID, i.StoryContent, string(i.Type), i.Explanation, i.EvidenceUSAC, i.EvidencePM,
		i.Role, i.Action, i.Resource, string(i.Status), i.Status == domain.IssueResolved && i.ResolvedBy == nil,
		i.AssigneeID, i.AssigneeEmail, i.AssigneeName, i.LastSeenRunID, i.LastSeenAt, i.ResolvedAt,
		i.CommentCount, i.CreatedAt, i.UpdatedAt}
}

type activityDTO struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	ActorID   *string   `json:"actor_id"`
	ActorName *string   `json:"actor_name"`
	Body      *string   `json:"body"`
	From      *string   `json:"from"`
	To        *string   `json:"to"`
	RunID     *string   `json:"run_id"`
	CreatedAt time.Time `json:"created_at"`
}

func toActivityDTO(a domain.IssueActivity) activityDTO {
	return activityDTO{a.ID, string(a.Kind), a.ActorID, a.ActorName, a.Body, a.FromValue, a.ToValue, a.RunID, a.CreatedAt}
}

// ---------- Báo cáo ----------

type countDTO struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type coverageRowDTO struct {
	pmRowDTO
	Covered  bool     `json:"covered"`
	StoryIDs []string `json:"story_ids"`
}

type coverageCellDTO struct {
	Role      string `json:"role"`
	Resource  string `json:"resource"`
	Allow     int    `json:"allow"`
	Deny      int    `json:"deny"`
	Covered   int    `json:"covered"`
	Uncovered int    `json:"uncovered"`
}

type coverageDTO struct {
	Rows            []coverageRowDTO  `json:"rows"`
	Roles           []string          `json:"roles"`
	Resources       []string          `json:"resources"`
	Cells           []coverageCellDTO `json:"cells"`
	UnusedAllow     int               `json:"unused_allow"`
	AnalyzedStories int               `json:"analyzed_stories"`
}

func toCoverageDTO(c domain.Coverage) coverageDTO {
	return coverageDTO{
		Rows: mapSlice(c.Rows, func(r domain.CoverageRow) coverageRowDTO {
			return coverageRowDTO{toPMRowDTO(r.Row), r.Covered, r.StoryIDs}
		}),
		Roles: c.Roles, Resources: c.Resources,
		Cells: mapSlice(c.Cells, func(x domain.CoverageCell) coverageCellDTO {
			return coverageCellDTO{x.Role, x.Resource, x.Allow, x.Deny, x.Covered, x.Uncovered}
		}),
		UnusedAllow: c.UnusedAllow, AnalyzedStories: c.AnalyzedStories,
	}
}

type storyStatusDTO struct {
	ID           string     `json:"id"`
	Content      string     `json:"content"`
	LastRunID    *string    `json:"last_run_id"`
	LastRunAt    *time.Time `json:"last_run_at"`
	IsConsistent *bool      `json:"is_consistent"`
	OpenIssues   int        `json:"open_issues"`
}

type reportDTO struct {
	Project           projectDTO       `json:"project"`
	GeneratedAt       time.Time        `json:"generated_at"`
	StoriesTotal      int              `json:"stories_total"`
	StoriesAnalyzed   int              `json:"stories_analyzed"`
	StoriesConsistent int              `json:"stories_consistent"`
	ConsistencyRate   float64          `json:"consistency_rate"`
	IssuesByStatus    []countDTO       `json:"issues_by_status"`
	IssuesByType      []countDTO       `json:"issues_by_type"`
	OpenIssuesByRole  []countDTO       `json:"open_issues_by_role"`
	Stories           []storyStatusDTO `json:"stories"`
	UnusedAllow       int              `json:"unused_allow"`
}

func toReportDTO(r domain.ProjectReport) reportDTO {
	toCounts := func(items []domain.CountItem) []countDTO {
		return mapSlice(items, func(c domain.CountItem) countDTO { return countDTO{c.Key, c.Count} })
	}
	return reportDTO{
		Project: toProjectDTO(r.Project), GeneratedAt: r.GeneratedAt, StoriesTotal: r.StoriesTotal,
		StoriesAnalyzed: r.StoriesAnalyzed, StoriesConsistent: r.StoriesConsistent, ConsistencyRate: r.ConsistencyRate,
		IssuesByStatus: toCounts(r.IssuesByStatus), IssuesByType: toCounts(r.IssuesByType),
		OpenIssuesByRole: toCounts(r.OpenIssuesByRole), UnusedAllow: r.Coverage.UnusedAllow,
		Stories: mapSlice(r.Stories, func(s domain.StoryStatus) storyStatusDTO {
			return storyStatusDTO{s.Story.ID, s.Story.Content, s.LastRunID, s.LastRunAt, s.IsConsistent, s.OpenIssues}
		}),
	}
}

type comparisonDTO struct {
	Base         runSummaryDTO `json:"base"`
	Target       runSummaryDTO `json:"target"`
	Introduced   []conflictDTO `json:"introduced"`
	Resolved     []conflictDTO `json:"resolved"`
	Unchanged    []conflictDTO `json:"unchanged"`
	PMDiff       pmDiffDTO     `json:"pm_diff"`
	StoryChanged bool          `json:"story_changed"`
}

func runRef(d domain.RunDetail) runSummaryDTO {
	return runSummaryDTO{ID: d.ID, ProjectID: d.ProjectID, UserStoryID: d.UserStoryID, UserStory: d.UserStory,
		IsConsistent: d.IsConsistent, ModelVersion: d.ModelVersion, CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt,
		ConflictCount: len(d.Conflicts), ConflictTypes: mapSlice(d.Conflicts, func(c domain.StoredConflict) string { return string(c.Type) })}
}

// ---------- Admin ----------

type auditDTO struct {
	ID         int64          `json:"id"`
	ActorID    *string        `json:"actor_id"`
	ActorEmail *string        `json:"actor_email"`
	Action     string         `json:"action"`
	TargetType *string        `json:"target_type"`
	TargetID   *string        `json:"target_id"`
	Detail     map[string]any `json:"detail"`
	IP         *string        `json:"ip"`
	CreatedAt  time.Time      `json:"created_at"`
}

func toAuditDTO(e domain.AuditEntry) auditDTO {
	return auditDTO{e.ID, e.ActorID, e.ActorEmail, e.Action, e.TargetType, e.TargetID, e.Detail, e.IP, e.CreatedAt}
}

// datasetItemDTO cùng định dạng data/processed/*.json của ml-service.
type datasetItemDTO struct {
	ID                 string     `json:"id"`
	UserStory          string     `json:"user_story"`
	AcceptanceCriteria []string   `json:"acceptance_criteria"`
	PermissionMatrix   []pmRowDTO `json:"permission_matrix"`
	ConflictLabel      string     `json:"conflict_label"`
	Origin             string     `json:"origin"`
	Notes              string     `json:"notes"`
}

func toDatasetItemDTO(d domain.DatasetItem) datasetItemDTO {
	rows := mapSlice(d.PermissionMatrix, func(r domain.PMRow) pmRowDTO { r.ID = ""; return toPMRowDTO(r) })
	return datasetItemDTO{d.ID, d.UserStory, d.AcceptanceCriteria, rows, string(d.ConflictLabel), "reqsentinel-feedback", d.Notes}
}

func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}
	return out
}
