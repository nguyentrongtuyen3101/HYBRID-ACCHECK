package excel

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestTemplateRoundTrip(t *testing.T) {
	data, err := BuildTemplate()
	if err != nil {
		t.Fatal(err)
	}
	res, err := Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.UserStories) != 2 || len(res.PermissionMatrix) != 2 || len(res.Warnings) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if got := res.UserStories[0].AcceptanceCriteria; len(got) != 2 || got[1] != "Staff must be logged in." {
		t.Fatalf("acceptance criteria not split on ';': %q", got)
	}
	if pm := res.PermissionMatrix[0]; pm.Effect != "Allow" || pm.Scope == nil || *pm.Scope != "assigned" || pm.Condition != nil {
		t.Fatalf("unexpected PM row: %+v", pm)
	}
}

func TestParseSkipsInvalidRows(t *testing.T) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "UserStories")
	f.SetSheetRow("UserStories", "A1", &[]any{"Content", "acceptance_criteria"})
	f.SetSheetRow("UserStories", "A2", &[]any{"", "orphan AC"})
	f.SetSheetRow("UserStories", "A4", &[]any{"As a user, I want X", "a\nb"})
	f.NewSheet("PermissionMatrix")
	f.SetSheetRow("PermissionMatrix", "A1", &[]any{"role", "action", "resource", "effect"})
	f.SetSheetRow("PermissionMatrix", "A2", &[]any{"Admin", "delete", "User", "DENY"})
	f.SetSheetRow("PermissionMatrix", "A3", &[]any{"Admin", "", "User", "Allow"})
	buf, _ := f.WriteToBuffer()

	res, err := Parse(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.UserStories) != 1 || len(res.UserStories[0].AcceptanceCriteria) != 2 {
		t.Fatalf("stories: %+v", res.UserStories)
	}
	if len(res.PermissionMatrix) != 1 || res.PermissionMatrix[0].Effect != "Deny" {
		t.Fatalf("pm: %+v", res.PermissionMatrix)
	}
	if len(res.Warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %q", res.Warnings)
	}
}
