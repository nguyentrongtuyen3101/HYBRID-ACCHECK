package excel

import (
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

// Sheets hiện thực app.Spreadsheet.
type Sheets struct{}

func (Sheets) ImportTemplate() ([]byte, error) { return BuildTemplate() }

func (Sheets) ParseImport(r io.Reader) (app.ImportData, error) {
	res, err := Parse(r)
	if err != nil {
		return app.ImportData{}, err
	}
	data := app.ImportData{Warnings: res.Warnings}
	for _, s := range res.UserStories {
		data.Stories = append(data.Stories, app.ImportStory{Content: s.Content, AcceptanceCriteria: s.AcceptanceCriteria})
	}
	for _, p := range res.PermissionMatrix {
		data.PMRows = append(data.PMRows, domain.PMRow{Role: p.Role, Action: p.Action, Resource: p.Resource,
			Effect: domain.Effect(p.Effect), Scope: p.Scope, Condition: p.Condition})
	}
	return data, nil
}

// ProjectReport xuất báo cáo: Summary, Conflicts, Stories, Coverage.
func (Sheets) ProjectReport(rep domain.ProjectReport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	header, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1A1A1A"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	wrap, err := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Vertical: "top", WrapText: true}})
	if err != nil {
		return nil, err
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}

	table := func(sheet string, headers []string, widths []float64, rows [][]any) error {
		if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
			return err
		}
		last, _ := excelize.CoordinatesToCellName(len(headers), 1)
		f.SetCellStyle(sheet, "A1", last, header)
		for i, row := range rows {
			cell, _ := excelize.CoordinatesToCellName(1, i+2)
			if err := f.SetSheetRow(sheet, cell, &row); err != nil {
				return err
			}
		}
		if len(rows) > 0 {
			end, _ := excelize.CoordinatesToCellName(len(headers), len(rows)+1)
			f.SetCellStyle(sheet, "A2", end, wrap)
		}
		for i, w := range widths {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheet, col, col, w)
		}
		return f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	}

	// --- Summary ---
	f.SetSheetName("Sheet1", "Summary")
	summary := [][]any{
		{"Project", rep.Project.Name},
		{"Generated at", rep.GeneratedAt.Format("2006-01-02 15:04")},
		{"User stories", rep.StoriesTotal},
		{"Stories analyzed", rep.StoriesAnalyzed},
		{"Stories consistent", rep.StoriesConsistent},
		{"Consistency rate", fmt.Sprintf("%.1f%%", rep.ConsistencyRate)},
		{"Permission rows not used by any story (Allow)", rep.Coverage.UnusedAllow},
		{},
		{"Conflicts by status"},
	}
	for _, c := range rep.IssuesByStatus {
		summary = append(summary, []any{c.Key, c.Count})
	}
	summary = append(summary, []any{}, []any{"Conflicts by type"})
	for _, c := range rep.IssuesByType {
		summary = append(summary, []any{c.Key, c.Count})
	}
	summary = append(summary, []any{}, []any{"Open conflicts by role"})
	for _, c := range rep.OpenIssuesByRole {
		summary = append(summary, []any{c.Key, c.Count})
	}
	for i, row := range summary {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		f.SetSheetRow("Summary", cell, &row)
		if len(row) == 1 {
			f.SetCellStyle("Summary", cell, cell, bold)
		}
	}
	f.SetColWidth("Summary", "A", "A", 46)
	f.SetColWidth("Summary", "B", "B", 30)

	// --- Conflicts ---
	f.NewSheet("Conflicts")
	var issueRows [][]any
	for _, i := range rep.Issues {
		issueRows = append(issueRows, []any{string(i.Status), string(i.Type), deref(i.Role), deref(i.Action),
			deref(i.Resource), i.StoryContent, i.Explanation, deref(i.EvidenceUSAC), deref(i.EvidencePM),
			deref(i.AssigneeEmail), i.LastSeenAt.Format("2006-01-02 15:04")})
	}
	if err := table("Conflicts", []string{"Status", "Type", "Role", "Action", "Resource", "User story",
		"Explanation", "Evidence US/AC", "Evidence PM", "Assignee", "Last detected"},
		[]float64{14, 18, 16, 12, 18, 50, 50, 24, 24, 24, 17}, issueRows); err != nil {
		return nil, err
	}

	// --- Stories ---
	f.NewSheet("Stories")
	var storyRows [][]any
	for _, s := range rep.Stories {
		status := "Not analyzed"
		if s.IsConsistent != nil {
			status = map[bool]string{true: "Consistent", false: "Conflicts"}[*s.IsConsistent]
		}
		last := ""
		if s.LastRunAt != nil {
			last = s.LastRunAt.Format("2006-01-02 15:04")
		}
		storyRows = append(storyRows, []any{s.Story.Content, strings.Join(s.Story.CriteriaTexts(), "\n"), status,
			s.OpenIssues, last})
	}
	if err := table("Stories", []string{"User story", "Acceptance criteria", "Status", "Open conflicts", "Last analyzed"},
		[]float64{60, 50, 14, 14, 17}, storyRows); err != nil {
		return nil, err
	}

	// --- Coverage ---
	f.NewSheet("Coverage")
	var covRows [][]any
	for _, r := range rep.Coverage.Rows {
		covRows = append(covRows, []any{r.Row.Role, r.Row.Action, r.Row.Resource, string(r.Row.Effect),
			deref(r.Row.Scope), deref(r.Row.Condition), map[bool]string{true: "Yes", false: "No"}[r.Covered], len(r.StoryIDs)})
	}
	if err := table("Coverage", []string{"Role", "Action", "Resource", "Effect", "Scope", "Condition",
		"Used by a story", "Stories"}, []float64{18, 12, 20, 9, 12, 24, 15, 9}, covRows); err != nil {
		return nil, err
	}

	f.SetActiveSheet(0)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
