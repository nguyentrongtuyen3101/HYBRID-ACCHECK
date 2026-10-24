package domain

import (
	"strings"
	"time"
)

type ConflictType string

const (
	ConflictNormal            ConflictType = "Normal"
	ConflictMissingPermission ConflictType = "Missing Permission"
	ConflictAction            ConflictType = "Action Conflict"
	ConflictEffect            ConflictType = "Effect Conflict"
	ConflictScope             ConflictType = "Scope Conflict"
	ConflictCondition         ConflictType = "Condition Conflict"
)

type AnalysisInput struct {
	UserStory          string
	AcceptanceCriteria []string
	PermissionMatrix   []PMRow
}

func (in AnalysisInput) Validate() error {
	if strings.TrimSpace(in.UserStory) == "" {
		return Validation("story_content_required", "user_story is required")
	}
	for _, r := range in.PermissionMatrix {
		if !r.Effect.Valid() {
			return Validation("invalid_effect", "permission_matrix effect must be Allow or Deny")
		}
	}
	return nil
}

// Normalize áp dụng từ điển project lên input trước khi gửi sang ML.
func (in AnalysisInput) Normalize(n *TermNormalizer) AnalysisInput {
	out := AnalysisInput{UserStory: n.NormalizeText(in.UserStory)}
	for _, ac := range in.AcceptanceCriteria {
		out.AcceptanceCriteria = append(out.AcceptanceCriteria, n.NormalizeText(ac))
	}
	for _, r := range in.PermissionMatrix {
		out.PermissionMatrix = append(out.PermissionMatrix, n.NormalizeRow(r))
	}
	return out
}

type ExtractedEntity struct {
	Role, Action, Resource, Effect, Condition, Scope *string
	Source                                           string
	Confidence                                       float64
}

type Alignment struct {
	Extracted ExtractedEntity
	Matched   *PMRow
	Score     float64
	IsMatched bool
}

type DetectedConflict struct {
	Type         ConflictType
	Confidence   float64
	Explanation  string
	EvidenceUSAC *string
	EvidencePM   *string
	Related      []ExtractedEntity
}

type AnalysisResult struct {
	Entities     []ExtractedEntity
	Alignments   []Alignment
	Conflicts    []DetectedConflict
	IsConsistent bool
	Summary      string
	ModelVersion string
}

// RealConflicts bỏ các kết quả "Normal".
func (r AnalysisResult) RealConflicts() []DetectedConflict {
	out := []DetectedConflict{}
	for _, c := range r.Conflicts {
		if c.Type != ConflictNormal {
			out = append(out, c)
		}
	}
	return out
}

// ---------- Run (1 lần phân tích) ----------

type Run struct {
	ID          string
	ProjectID   *string
	UserStoryID *string
	JobID       *string
	PMVersionID *string
	CreatedBy   string
	Input       AnalysisInput
	Result      AnalysisResult
	IssueIDs    []string // song song với Result.RealConflicts()
	CreatedAt   time.Time
}

type RunSummary struct {
	ID            string
	ProjectID     *string
	ProjectName   *string
	UserStoryID   *string
	UserStory     string
	IsConsistent  bool
	Summary       string
	ConflictTypes []ConflictType
	ModelVersion  *string
	CreatedBy     *string
	CreatedAt     time.Time
}

type StoredConflict struct {
	IssueID      *string
	Type         ConflictType
	Confidence   float64
	Explanation  string
	EvidenceUSAC *string
	EvidencePM   *string
}

type RunDetail struct {
	ID                 string
	ProjectID          *string
	UserStoryID        *string
	PMVersionID        *string
	JobID              *string
	CreatedBy          *string
	UserStory          string
	AcceptanceCriteria []string
	PermissionMatrix   []PMRow
	Alignments         []Alignment
	IsConsistent       bool
	Summary            *string
	ModelVersion       *string
	Conflicts          []StoredConflict
	Entities           []ExtractedEntity
	CreatedAt          time.Time
}

type RunFilter struct {
	VisibleTo *string // nil = tất cả (admin)
	ProjectID *string
	Limit     int
}

// ---------- Job phân tích cả project ----------

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
)

func (s JobStatus) Active() bool { return s == JobQueued || s == JobRunning }

type AnalysisJob struct {
	ID           string
	ProjectID    string
	ProjectName  string
	CreatedBy    string
	Status       JobStatus
	Total        int
	Processed    int
	Failed       int
	Error        *string
	ModelVersion *string
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}
