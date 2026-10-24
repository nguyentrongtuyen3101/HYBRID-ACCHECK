package domain

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func ptr(s string) *string { return &s }

func seqID() func() string {
	n := 0
	return func() string { n++; return fmt.Sprintf("id-%d", n) }
}

func TestProjectRolePermissions(t *testing.T) {
	cases := []struct {
		role ProjectRole
		perm Permission
		want bool
	}{
		{ProjectViewer, PermView, true},
		{ProjectViewer, PermComment, true},
		{ProjectViewer, PermEdit, false},
		{ProjectViewer, PermTriage, false},
		{ProjectEditor, PermRunAnalysis, true},
		{ProjectEditor, PermTriage, true},
		{ProjectEditor, PermManageMembers, false},
		{ProjectOwner, PermManageProject, true},
	}
	for _, c := range cases {
		if got := c.role.Can(c.perm); got != c.want {
			t.Errorf("%s.Can(%d) = %v, want %v", c.role, c.perm, got, c.want)
		}
	}
	admin := Actor{ID: "a", Role: SystemRoleAdmin}
	if r, ok := EffectiveRole(admin, nil); !ok || r != ProjectOwner {
		t.Error("admin must act as owner on every project")
	}
	if _, ok := EffectiveRole(Actor{ID: "u", Role: SystemRoleUser}, nil); ok {
		t.Error("non-member must have no role")
	}
}

func TestEnsureOwnerRemains(t *testing.T) {
	owner := Member{Role: ProjectOwner}
	editor := ProjectEditor
	if EnsureOwnerRemains(owner, &editor, 1) == nil {
		t.Error("demoting the last owner must fail")
	}
	if EnsureOwnerRemains(owner, nil, 1) == nil {
		t.Error("removing the last owner must fail")
	}
	if EnsureOwnerRemains(owner, &editor, 2) != nil {
		t.Error("demoting one of two owners is allowed")
	}
	if EnsureOwnerRemains(Member{Role: ProjectViewer}, nil, 1) != nil {
		t.Error("removing a viewer is always allowed")
	}
}

func TestInvitationCheckAcceptable(t *testing.T) {
	now := time.Now()
	inv := Invitation{Email: "Bob@Example.com", Status: InvitationPending, ExpiresAt: now.Add(time.Hour)}
	if err := inv.CheckAcceptable("bob@example.com", now); err != nil {
		t.Errorf("valid invitation rejected: %v", err)
	}
	if KindOf(inv.CheckAcceptable("eve@example.com", now)) != KindForbidden {
		t.Error("other email must be forbidden")
	}
	if inv.CheckAcceptable("bob@example.com", now.Add(2*time.Hour)) == nil {
		t.Error("expired invitation accepted")
	}
	inv.Status = InvitationRevoked
	if inv.CheckAcceptable("bob@example.com", now) == nil {
		t.Error("revoked invitation accepted")
	}
}

func TestDiffPM(t *testing.T) {
	from := []PMRow{
		{Role: "Admin", Action: "delete", Resource: "User", Effect: EffectAllow},
		{Role: "Guest", Action: "view", Resource: "Catalog", Effect: EffectAllow},
		{Role: "Staff", Action: "edit", Resource: "Order", Effect: EffectAllow, Scope: ptr("own")},
	}
	to := []PMRow{
		{Role: "admin", Action: "Delete", Resource: "user", Effect: EffectAllow},    // không đổi (khác hoa thường)
		{Role: "Guest", Action: "view", Resource: "Catalog", Effect: EffectDeny},    // đổi effect
		{Role: "Staff", Action: "approve", Resource: "Refund", Effect: EffectAllow}, // thêm
	}
	d := DiffPM(from, to)
	if len(d.Added) != 1 || d.Added[0].Resource != "Refund" {
		t.Errorf("added = %+v", d.Added)
	}
	if len(d.Removed) != 1 || d.Removed[0].Resource != "Order" {
		t.Errorf("removed = %+v", d.Removed)
	}
	if len(d.Changed) != 1 || d.Changed[0].After.Effect != EffectDeny {
		t.Errorf("changed = %+v", d.Changed)
	}
	if PMChecksum(from) != PMChecksum([]PMRow{from[2], from[0], from[1]}) {
		t.Error("checksum must not depend on row order")
	}
	if PMChecksum(from) == PMChecksum(to) {
		t.Error("different matrices must have different checksums")
	}
}

func TestTermNormalizer(t *testing.T) {
	n := NewTermNormalizer([]Term{
		{Kind: TermRole, Canonical: "Sales Staff", Synonyms: []string{"sales employee", "employee"}},
		{Kind: TermAction, Canonical: "view", Synonyms: []string{"see", "read"}},
	})
	got := n.NormalizeText("As a Sales Employee, I want to see customers. An employee can read it.")
	want := "As a Sales Staff, I want to view customers. An Sales Staff can view it."
	if got != want {
		t.Errorf("NormalizeText:\n got %q\nwant %q", got, want)
	}
	row := n.NormalizeRow(PMRow{Role: "EMPLOYEE", Action: "Read", Resource: "Customer"})
	if row.Role != "Sales Staff" || row.Action != "view" || row.Resource != "Customer" {
		t.Errorf("NormalizeRow = %+v", row)
	}
	sugg := SuggestTerms(n, []PMRow{{Role: "Manager", Action: "view", Resource: "Report"}, {Role: "manager", Action: "see", Resource: "Report"}})
	if len(sugg) != 2 || sugg[0].Kind != TermResource || sugg[1].Value != "Manager" || sugg[1].Count != 2 {
		t.Errorf("SuggestTerms = %+v", sugg)
	}
	term := Term{Kind: TermRole, Canonical: " Admin ", Synonyms: []string{"admin", " ", "Administrator", "administrator"}}
	if err := term.Normalize(); err != nil || term.Canonical != "Admin" || len(term.Synonyms) != 1 {
		t.Errorf("Term.Normalize = %+v, %v", term, err)
	}
}

func conflict(typ ConflictType, us, pm string) DetectedConflict {
	return DetectedConflict{Type: typ, EvidenceUSAC: ptr(us), EvidencePM: ptr(pm), Explanation: string(typ),
		Related: []ExtractedEntity{{Role: ptr("Guest")}}}
}

func TestSyncIssuesLifecycle(t *testing.T) {
	now := time.Now()
	newID := seqID()
	effect := conflict(ConflictEffect, "Allow", "Deny")
	scope := conflict(ConflictScope, "own", "all")

	// Lần 1: 2 conflict mới → 2 issue open
	s1 := SyncIssues("p", "s", "run1", nil, []DetectedConflict{effect, scope}, now, newID)
	if len(s1.Created) != 2 || s1.Created[0].Status != IssueOpen || s1.Created[0].Role == nil {
		t.Fatalf("run1 created = %+v", s1.Created)
	}
	if s1.RunIssueIDs[0] != s1.Created[0].ID || s1.RunIssueIDs[1] != s1.Created[1].ID {
		t.Fatalf("run issue ids = %v", s1.RunIssueIDs)
	}

	// Người dùng đánh dấu scope là false positive
	existing := append([]Issue{}, s1.Created...)
	existing[1].Status = IssueFalsePositive

	// Lần 2: chỉ còn scope → effect tự resolved, scope giữ false_positive
	s2 := SyncIssues("p", "s", "run2", existing, []DetectedConflict{scope}, now, newID)
	if len(s2.Created) != 0 || len(s2.Updated) != 2 {
		t.Fatalf("run2 created=%d updated=%d", len(s2.Created), len(s2.Updated))
	}
	byID := map[string]Issue{}
	for _, i := range s2.Updated {
		byID[i.ID] = i
	}
	if e := byID[existing[0].ID]; e.Status != IssueResolved || e.ResolvedBy != nil || e.HumanReviewed() {
		t.Errorf("effect issue should be auto-resolved: %+v", e)
	}
	if sc := byID[existing[1].ID]; sc.Status != IssueFalsePositive || sc.LastSeenRunID != "run2" {
		t.Errorf("false positive must stay: %+v", sc)
	}

	// Lần 3: effect xuất hiện lại → mở lại
	existing = s2.Updated
	s3 := SyncIssues("p", "s", "run3", existing, []DetectedConflict{effect, effect}, now, newID)
	reopened := false
	for _, a := range s3.Activities {
		if a.Kind == ActivityReopened {
			reopened = true
		}
	}
	if !reopened || s3.RunIssueIDs[0] != s3.RunIssueIDs[1] {
		t.Errorf("effect must be reopened once and deduplicated: %+v", s3)
	}
}

func TestIssueTransitions(t *testing.T) {
	if !IssueOpen.CanTransitionTo(IssueResolved) || IssueResolved.CanTransitionTo(IssueFalsePositive) {
		t.Error("unexpected transition rules")
	}
	if IssueStatus("bogus").Valid() {
		t.Error("bogus status accepted")
	}
}

func TestComputeCoverage(t *testing.T) {
	rows := []PMRow{
		{Role: "Guest", Action: "view", Resource: "Catalog", Effect: EffectAllow},
		{Role: "Admin", Action: "delete", Resource: "User", Effect: EffectAllow},
		{Role: "Guest", Action: "delete", Resource: "Catalog", Effect: EffectDeny},
	}
	matched := rows[0]
	cov := ComputeCoverage(rows, []StoryAlignments{
		{StoryID: "s1", Alignments: []Alignment{{IsMatched: true, Matched: &matched}}},
		{StoryID: "s2", Alignments: []Alignment{{IsMatched: false}}},
	})
	if !cov.Rows[0].Covered || cov.Rows[1].Covered || cov.UnusedAllow != 1 {
		t.Errorf("coverage = %+v", cov)
	}
	if len(cov.Roles) != 2 || len(cov.Cells) != 2 {
		t.Errorf("roles=%v cells=%+v", cov.Roles, cov.Cells)
	}
}

func TestLabelFromIssues(t *testing.T) {
	user := "u"
	if _, ok := LabelFromIssues([]Issue{{Status: IssueOpen}}); ok {
		t.Error("unreviewed story must not be labeled")
	}
	if l, ok := LabelFromIssues([]Issue{{Status: IssueFalsePositive, Type: ConflictScope}}); !ok || l != ConflictNormal {
		t.Errorf("all false positives → Normal, got %q", l)
	}
	if l, ok := LabelFromIssues([]Issue{{Status: IssueResolved, ResolvedBy: &user, Type: ConflictEffect}}); !ok || l != ConflictEffect {
		t.Errorf("confirmed conflict → its type, got %q", l)
	}
}

func TestCompareRuns(t *testing.T) {
	a := RunDetail{UserStory: "x", Conflicts: []StoredConflict{{Type: ConflictEffect, EvidenceUSAC: ptr("Allow"), EvidencePM: ptr("Deny")}}}
	b := RunDetail{UserStory: "x", Conflicts: []StoredConflict{{Type: ConflictScope, EvidenceUSAC: ptr("own"), EvidencePM: ptr("all")}}}
	c := CompareRuns(a, b)
	if len(c.Introduced) != 1 || len(c.Resolved) != 1 || len(c.Unchanged) != 0 || c.StoryChanged {
		t.Errorf("comparison = %+v", c)
	}
}

func TestProfilePatchNormalize(t *testing.T) {
	name, phone, bad := "  Tuyen Nguyen ", "+84 362-851-330", "call me"
	p := ProfilePatch{FullName: &name, Phone: &phone, Locale: OptionalString{Set: true, Value: ptr("ja")}}
	if err := p.Normalize(); err != nil || *p.FullName != "Tuyen Nguyen" {
		t.Fatalf("valid profile rejected: %v", err)
	}
	if err := (&ProfilePatch{Phone: &bad}).Normalize(); KindOf(err) != KindValidation {
		t.Error("invalid phone accepted")
	}
	if err := (&ProfilePatch{Locale: OptionalString{Set: true, Value: ptr("fr")}}).Normalize(); err == nil {
		t.Error("unsupported locale accepted")
	}
	long := string(make([]rune, 101))
	var de *Error
	if err := (&ProfilePatch{JobTitle: &long}).Normalize(); !errors.As(err, &de) || de.Code != "invalid_job_title" || de.Params["max"] != 100 {
		t.Errorf("long job title: %v", err)
	}
	if err := ValidatePassword("short"); !errors.As(err, &de) || de.Code != "password_length" || de.Params["min"] != MinPasswordLen {
		t.Errorf("password error must carry code + params: %+v", err)
	}
}
