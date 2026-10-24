package domain

import (
	"sort"
	"strings"
	"time"
)

// ---------- Ma trận bao phủ ----------

// StoryAlignments là các alignment của lần chạy gần nhất của 1 user story.
type StoryAlignments struct {
	StoryID    string
	Alignments []Alignment
}

type CoverageRow struct {
	Row      PMRow
	Covered  bool
	StoryIDs []string
}

type CoverageCell struct {
	Role, Resource string
	Allow, Deny    int
	Covered        int
	Uncovered      int
}

type Coverage struct {
	Rows      []CoverageRow
	Roles     []string
	Resources []string
	Cells     []CoverageCell
	// Quyền Allow không user story nào cần tới → nghi cấp thừa quyền
	UnusedAllow     int
	AnalyzedStories int
}

func ComputeCoverage(rows []PMRow, stories []StoryAlignments) Coverage {
	usedBy := map[string][]string{}
	for _, s := range stories {
		for _, a := range s.Alignments {
			if a.IsMatched && a.Matched != nil {
				k := a.Matched.Key()
				usedBy[k] = appendUnique(usedBy[k], s.StoryID)
			}
		}
	}

	cov := Coverage{Rows: []CoverageRow{}, Cells: []CoverageCell{}, AnalyzedStories: len(stories)}
	cells := map[string]*CoverageCell{}
	roles, resources := map[string]string{}, map[string]string{}
	for _, r := range rows {
		ids := usedBy[r.Key()]
		cr := CoverageRow{Row: r, Covered: len(ids) > 0, StoryIDs: ids}
		if cr.StoryIDs == nil {
			cr.StoryIDs = []string{}
		}
		cov.Rows = append(cov.Rows, cr)
		if !cr.Covered && r.Effect == EffectAllow {
			cov.UnusedAllow++
		}

		roles[strings.ToLower(r.Role)] = r.Role
		resources[strings.ToLower(r.Resource)] = r.Resource
		key := strings.ToLower(r.Role) + "|" + strings.ToLower(r.Resource)
		cell, ok := cells[key]
		if !ok {
			cell = &CoverageCell{Role: r.Role, Resource: r.Resource}
			cells[key] = cell
		}
		if r.Effect == EffectDeny {
			cell.Deny++
		} else {
			cell.Allow++
		}
		if cr.Covered {
			cell.Covered++
		} else {
			cell.Uncovered++
		}
	}
	cov.Roles, cov.Resources = sortedValues(roles), sortedValues(resources)
	for _, c := range cells {
		cov.Cells = append(cov.Cells, *c)
	}
	sort.Slice(cov.Cells, func(i, j int) bool {
		if cov.Cells[i].Role != cov.Cells[j].Role {
			return cov.Cells[i].Role < cov.Cells[j].Role
		}
		return cov.Cells[i].Resource < cov.Cells[j].Resource
	})
	return cov
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func sortedValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// ---------- So sánh 2 lần chạy ----------

type RunComparison struct {
	Introduced   []StoredConflict
	Resolved     []StoredConflict
	Unchanged    []StoredConflict
	PMDiff       PMDiff
	StoryChanged bool
}

func CompareRuns(base, target RunDetail) RunComparison {
	key := func(c StoredConflict) string {
		return Fingerprint("", DetectedConflict{Type: c.Type, EvidenceUSAC: c.EvidenceUSAC, EvidencePM: c.EvidencePM})
	}
	before := map[string]StoredConflict{}
	for _, c := range base.Conflicts {
		before[key(c)] = c
	}
	cmp := RunComparison{Introduced: []StoredConflict{}, Resolved: []StoredConflict{}, Unchanged: []StoredConflict{}}
	seen := map[string]bool{}
	for _, c := range target.Conflicts {
		k := key(c)
		seen[k] = true
		if _, ok := before[k]; ok {
			cmp.Unchanged = append(cmp.Unchanged, c)
		} else {
			cmp.Introduced = append(cmp.Introduced, c)
		}
	}
	for _, c := range base.Conflicts {
		if !seen[key(c)] {
			cmp.Resolved = append(cmp.Resolved, c)
		}
	}
	cmp.PMDiff = DiffPM(base.PermissionMatrix, target.PermissionMatrix)
	cmp.StoryChanged = strings.TrimSpace(base.UserStory) != strings.TrimSpace(target.UserStory) ||
		strings.Join(base.AcceptanceCriteria, "\n") != strings.Join(target.AcceptanceCriteria, "\n")
	return cmp
}

// ---------- Báo cáo project ----------

type StoryStatus struct {
	Story        UserStory
	LastRunID    *string
	LastRunAt    *time.Time
	IsConsistent *bool
	OpenIssues   int
}

type CountItem struct {
	Key   string
	Count int
}

type ProjectReport struct {
	Project           Project
	GeneratedAt       time.Time
	StoriesTotal      int
	StoriesAnalyzed   int
	StoriesConsistent int
	ConsistencyRate   float64 // % story đã phân tích mà nhất quán
	IssuesByStatus    []CountItem
	IssuesByType      []CountItem
	OpenIssuesByRole  []CountItem
	Stories           []StoryStatus
	Issues            []Issue
	Coverage          Coverage
}

func BuildReport(p Project, stories []UserStory, latest map[string]RunSummary, issues []Issue,
	cov Coverage, now time.Time) ProjectReport {

	rep := ProjectReport{Project: p, GeneratedAt: now, StoriesTotal: len(stories), Issues: issues, Coverage: cov}
	openByStory := map[string]int{}
	byStatus, byType, byRole := map[string]int{}, map[string]int{}, map[string]int{}
	for _, iss := range issues {
		byStatus[string(iss.Status)]++
		byType[string(iss.Type)]++
		if iss.Status.Active() {
			openByStory[iss.UserStoryID]++
			role := "(unknown)"
			if iss.Role != nil && *iss.Role != "" {
				role = *iss.Role
			}
			byRole[role]++
		}
	}
	for _, s := range stories {
		st := StoryStatus{Story: s, OpenIssues: openByStory[s.ID]}
		if run, ok := latest[s.ID]; ok {
			id, at, consistent := run.ID, run.CreatedAt, run.IsConsistent
			st.LastRunID, st.LastRunAt, st.IsConsistent = &id, &at, &consistent
			rep.StoriesAnalyzed++
			if consistent {
				rep.StoriesConsistent++
			}
		}
		rep.Stories = append(rep.Stories, st)
	}
	if rep.StoriesAnalyzed > 0 {
		rep.ConsistencyRate = float64(rep.StoriesConsistent) * 100 / float64(rep.StoriesAnalyzed)
	}
	rep.IssuesByStatus = toCountItems(byStatus)
	rep.IssuesByType = toCountItems(byType)
	rep.OpenIssuesByRole = toCountItems(byRole)
	return rep
}

func toCountItems(m map[string]int) []CountItem {
	out := make([]CountItem, 0, len(m))
	for k, v := range m {
		out = append(out, CountItem{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// ---------- Xuất dataset từ phản hồi người dùng ----------

type DatasetItem struct {
	ID                 string
	UserStory          string
	AcceptanceCriteria []string
	PermissionMatrix   []PMRow
	ConflictLabel      ConflictType
	Notes              string
}

// LabelFromIssues suy ra nhãn từ đánh giá của BA trên lần chạy gần nhất.
// ok=false khi chưa có issue nào được người xem xét.
func LabelFromIssues(issues []Issue) (ConflictType, bool) {
	reviewed := false
	for _, iss := range issues {
		if !iss.HumanReviewed() {
			continue
		}
		reviewed = true
		if iss.Status != IssueFalsePositive {
			return iss.Type, true
		}
	}
	if reviewed {
		return ConflictNormal, true
	}
	return "", false
}
