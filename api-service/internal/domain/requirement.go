package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type AcceptanceCriterion struct {
	ID      string
	Content string
}

type UserStory struct {
	ID                 string
	ProjectID          string
	Content            string
	Source             string
	AcceptanceCriteria []AcceptanceCriterion
	CreatedAt          time.Time
}

func (s UserStory) CriteriaTexts() []string {
	out := make([]string, 0, len(s.AcceptanceCriteria))
	for _, ac := range s.AcceptanceCriteria {
		out = append(out, ac.Content)
	}
	return out
}

// CleanCriteria bỏ khoảng trắng và dòng rỗng.
func CleanCriteria(in []string) []string {
	out := []string{}
	for _, c := range in {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

func ValidateStoryContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", Validation("story_content_required", "User story content is required")
	}
	return content, nil
}

// ---------- Permission matrix ----------

type Effect string

const (
	EffectAllow Effect = "Allow"
	EffectDeny  Effect = "Deny"
)

func (e Effect) Valid() bool { return e == EffectAllow || e == EffectDeny }

type PMRow struct {
	ID        string  `json:"id,omitempty"`
	Role      string  `json:"role"`
	Action    string  `json:"action"`
	Resource  string  `json:"resource"`
	Effect    Effect  `json:"effect"`
	Condition *string `json:"condition"`
	Scope     *string `json:"scope"`
}

// Key xác định 1 quyền: role + action + resource (không phân biệt hoa thường).
func (r PMRow) Key() string {
	return strings.ToLower(strings.TrimSpace(r.Role) + "|" + strings.TrimSpace(r.Action) + "|" + strings.TrimSpace(r.Resource))
}

func (r *PMRow) Normalize() error {
	r.Role, r.Action, r.Resource = strings.TrimSpace(r.Role), strings.TrimSpace(r.Action), strings.TrimSpace(r.Resource)
	if r.Role == "" || r.Action == "" || r.Resource == "" {
		return Validation("pm_fields_required", "Role, Action, Resource are required")
	}
	if r.Effect == "" {
		r.Effect = EffectAllow
	}
	if !r.Effect.Valid() {
		return Validation("invalid_effect", "Effect must be Allow or Deny")
	}
	r.Condition, r.Scope = trimOptional(r.Condition), trimOptional(r.Scope)
	return nil
}

func trimOptional(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// OptionalString phân biệt "không gửi" với "gửi null" trong PATCH.
type OptionalString struct {
	Set   bool
	Value *string
}

type PMRowPatch struct {
	Role, Action, Resource *string
	Effect                 *Effect
	Condition, Scope       OptionalString
}

// ---------- Version của Permission Matrix ----------

type PMVersion struct {
	ID        string
	ProjectID string
	Version   int
	Checksum  string
	Rows      []PMRow
	Label     *string
	CreatedBy *string
	CreatedAt time.Time
}

// PMChecksum: checksum không phụ thuộc thứ tự dòng, bỏ qua ID.
func PMChecksum(rows []PMRow) string {
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		r.ID = ""
		b, _ := json.Marshal(r)
		lines = append(lines, string(b))
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

type PMRowChange struct {
	Before PMRow
	After  PMRow
}

type PMDiff struct {
	Added   []PMRow
	Removed []PMRow
	Changed []PMRowChange // cùng role/action/resource nhưng khác effect/scope/condition
}

func DiffPM(from, to []PMRow) PMDiff {
	d := PMDiff{Added: []PMRow{}, Removed: []PMRow{}, Changed: []PMRowChange{}}
	old := map[string]PMRow{}
	for _, r := range from {
		old[r.Key()] = r
	}
	seen := map[string]bool{}
	for _, r := range to {
		k := r.Key()
		seen[k] = true
		before, ok := old[k]
		switch {
		case !ok:
			d.Added = append(d.Added, r)
		case !samePermission(before, r):
			d.Changed = append(d.Changed, PMRowChange{Before: before, After: r})
		}
	}
	for _, r := range from {
		if !seen[r.Key()] {
			d.Removed = append(d.Removed, r)
		}
	}
	return d
}

func samePermission(a, b PMRow) bool {
	return a.Effect == b.Effect && eqOpt(a.Scope, b.Scope) && eqOpt(a.Condition, b.Condition)
}

func eqOpt(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return strings.EqualFold(strings.TrimSpace(*a), strings.TrimSpace(*b))
}
