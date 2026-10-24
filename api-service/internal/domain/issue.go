package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"
)

// Issue là 1 conflict "sống" qua nhiều lần chạy của cùng user story.
// Mỗi lần chạy sinh conflict_records (occurrence) trỏ về issue tương ứng.
type IssueStatus string

const (
	IssueOpen          IssueStatus = "open"
	IssueAcknowledged  IssueStatus = "acknowledged"
	IssueResolved      IssueStatus = "resolved"
	IssueFalsePositive IssueStatus = "false_positive"
)

var issueTransitions = map[IssueStatus][]IssueStatus{
	IssueOpen:          {IssueAcknowledged, IssueResolved, IssueFalsePositive},
	IssueAcknowledged:  {IssueOpen, IssueResolved, IssueFalsePositive},
	IssueResolved:      {IssueOpen},
	IssueFalsePositive: {IssueOpen},
}

func (s IssueStatus) Valid() bool { _, ok := issueTransitions[s]; return ok }

func (s IssueStatus) CanTransitionTo(t IssueStatus) bool {
	for _, allowed := range issueTransitions[s] {
		if allowed == t {
			return true
		}
	}
	return false
}

// Active: issue còn cần xử lý.
func (s IssueStatus) Active() bool { return s == IssueOpen || s == IssueAcknowledged }

type Issue struct {
	ID             string
	ProjectID      string
	UserStoryID    string
	StoryContent   string
	Fingerprint    string
	Type           ConflictType
	Explanation    string
	EvidenceUSAC   *string
	EvidencePM     *string
	Role           *string
	Action         *string
	Resource       *string
	Status         IssueStatus
	AssigneeID     *string
	AssigneeEmail  *string
	AssigneeName   *string
	FirstSeenRunID string
	LastSeenRunID  string
	LastSeenAt     time.Time
	ResolvedBy     *string // nil + Status=resolved nghĩa là hệ thống tự đóng
	ResolvedAt     *time.Time
	CommentCount   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// HumanReviewed: đã có người xác nhận (dùng để xuất dataset train lại).
func (i Issue) HumanReviewed() bool {
	switch i.Status {
	case IssueAcknowledged, IssueFalsePositive:
		return true
	case IssueResolved:
		return i.ResolvedBy != nil
	}
	return false
}

type ActivityKind string

const (
	ActivityDetected      ActivityKind = "detected"
	ActivityReopened      ActivityKind = "reopened"
	ActivityAutoResolved  ActivityKind = "auto_resolved"
	ActivityStatusChanged ActivityKind = "status_changed"
	ActivityAssigned      ActivityKind = "assigned"
	ActivityComment       ActivityKind = "comment"
)

type IssueActivity struct {
	ID        string
	IssueID   string
	ActorID   *string
	ActorName *string
	Kind      ActivityKind
	Body      *string
	FromValue *string
	ToValue   *string
	RunID     *string
	CreatedAt time.Time
}

type IssueFilter struct {
	ProjectID   string
	Status      string
	Type        string
	AssigneeID  string
	UserStoryID string
	Limit       int
	Offset      int
}

// Fingerprint nhận diện cùng 1 conflict giữa các lần chạy.
func Fingerprint(storyKey string, c DetectedConflict) string {
	parts := []string{storyKey, string(c.Type), norm(c.EvidenceUSAC), norm(c.EvidencePM)}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:16])
}

func norm(s *string) string {
	if s == nil {
		return ""
	}
	return strings.ToLower(strings.Join(strings.Fields(*s), " "))
}

func ValidateComment(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > 5000 {
		return "", Validation("invalid_comment", "Comment must be 1-5000 characters")
	}
	return body, nil
}

// IssueSync là kết quả đồng bộ issue của 1 user story sau 1 lần chạy.
type IssueSync struct {
	Created    []Issue
	Updated    []Issue
	Activities []IssueActivity
	// RunIssueIDs[i] là issue của RealConflicts()[i]
	RunIssueIDs []string
}

// SyncIssues áp luật nghiệp vụ:
//   - conflict mới → tạo issue "open"
//   - conflict đã có, issue đang "resolved" → mở lại
//   - "false_positive" / "acknowledged" giữ nguyên (người dùng đã đánh giá)
//   - issue open/acknowledged không còn xuất hiện → hệ thống tự "resolved"
func SyncIssues(projectID, storyID, runID string, existing []Issue, detected []DetectedConflict,
	now time.Time, newID func() string) IssueSync {

	res := IssueSync{RunIssueIDs: make([]string, len(detected))}
	byFP := map[string]*Issue{}
	for i := range existing {
		byFP[existing[i].Fingerprint] = &existing[i]
	}
	created := map[string]*Issue{}
	updated := map[string]bool{}
	seen := map[string]bool{}
	runRef := &runID

	activity := func(issueID string, kind ActivityKind, from, to *string) {
		res.Activities = append(res.Activities, IssueActivity{
			ID: newID(), IssueID: issueID, Kind: kind, FromValue: from, ToValue: to, RunID: runRef, CreatedAt: now,
		})
	}

	for i, c := range detected {
		fp := Fingerprint(storyID, c)
		seen[fp] = true
		if iss, ok := created[fp]; ok {
			res.RunIssueIDs[i] = iss.ID
			continue
		}
		if iss, ok := byFP[fp]; ok {
			iss.LastSeenRunID, iss.LastSeenAt, iss.UpdatedAt = runID, now, now
			if iss.Status == IssueResolved {
				from := string(iss.Status)
				iss.Status, iss.ResolvedBy, iss.ResolvedAt = IssueOpen, nil, nil
				activity(iss.ID, ActivityReopened, &from, strPtr(string(IssueOpen)))
			}
			updated[iss.ID] = true
			res.RunIssueIDs[i] = iss.ID
			continue
		}
		role, action, resource := relatedFields(c)
		iss := Issue{
			ID: newID(), ProjectID: projectID, UserStoryID: storyID, Fingerprint: fp,
			Type: c.Type, Explanation: c.Explanation, EvidenceUSAC: c.EvidenceUSAC, EvidencePM: c.EvidencePM,
			Role: role, Action: action, Resource: resource, Status: IssueOpen,
			FirstSeenRunID: runID, LastSeenRunID: runID, LastSeenAt: now, CreatedAt: now, UpdatedAt: now,
		}
		created[fp] = &iss
		res.Created = append(res.Created, iss)
		res.RunIssueIDs[i] = iss.ID
		activity(iss.ID, ActivityDetected, nil, strPtr(string(IssueOpen)))
	}

	for i := range existing {
		iss := &existing[i]
		if seen[iss.Fingerprint] || !iss.Status.Active() {
			continue
		}
		from := string(iss.Status)
		iss.Status, iss.ResolvedBy, iss.ResolvedAt, iss.UpdatedAt = IssueResolved, nil, &now, now
		updated[iss.ID] = true
		activity(iss.ID, ActivityAutoResolved, &from, strPtr(string(IssueResolved)))
	}

	for i := range existing {
		if updated[existing[i].ID] {
			res.Updated = append(res.Updated, existing[i])
		}
	}
	return res
}

func relatedFields(c DetectedConflict) (role, action, resource *string) {
	if len(c.Related) == 0 {
		return nil, nil, nil
	}
	e := c.Related[0]
	return e.Role, e.Action, e.Resource
}

func strPtr(s string) *string { return &s }
