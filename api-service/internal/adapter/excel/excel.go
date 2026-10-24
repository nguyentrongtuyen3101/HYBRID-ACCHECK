// Package excel: file mẫu và import User Stories (kèm AC) / Permission Matrix.
// Port từ app/infrastructure/db/excel_io.py của bản Python.
package excel

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	sheetStories = "UserStories"
	sheetPM      = "PermissionMatrix"
	sheetReadme  = "README"
)

func BuildTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	border := []excelize.Border{
		{Type: "left", Color: "1A1A1A", Style: 1},
		{Type: "right", Color: "1A1A1A", Style: 1},
		{Type: "top", Color: "1A1A1A", Style: 1},
		{Type: "bottom", Color: "1A1A1A", Style: 1},
	}
	header, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Bold: true, Size: 14, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1A1A1A"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}
	body, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}
	bodyCenter, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	writeSheet := func(sheet string, rows [][]string, bodyStyle int, widths []float64) error {
		for r, row := range rows {
			for c, val := range row {
				cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
				if err := f.SetCellStr(sheet, cell, val); err != nil {
					return err
				}
				style := bodyStyle
				if r == 0 {
					style = header
				}
				if err := f.SetCellStyle(sheet, cell, cell, style); err != nil {
					return err
				}
			}
		}
		if err := f.SetRowHeight(sheet, 1, 28); err != nil {
			return err
		}
		for i, w := range widths {
			col, _ := excelize.ColumnNumberToName(i + 1)
			if err := f.SetColWidth(sheet, col, col, w); err != nil {
				return err
			}
		}
		return nil
	}

	if err := f.SetSheetName("Sheet1", sheetStories); err != nil {
		return nil, err
	}
	if err := writeSheet(sheetStories, [][]string{
		{"content", "acceptance_criteria"},
		{"As a Sales Staff, I want to view assigned customers so that I can manage my accounts.",
			"System shows only assigned customers.;Staff must be logged in."},
		{"As an Admin, I want to delete user accounts so that I can remove inactive users.",
			"Only inactive accounts can be deleted."},
	}, body, []float64{90, 55}); err != nil {
		return nil, err
	}

	if _, err := f.NewSheet(sheetPM); err != nil {
		return nil, err
	}
	if err := writeSheet(sheetPM, [][]string{
		{"role", "action", "resource", "effect", "scope", "condition"},
		{"Sales Staff", "view", "Customer", "Allow", "assigned", ""},
		{"Admin", "delete", "User Account", "Allow", "all", ""},
	}, bodyCenter, []float64{18, 12, 18, 10, 12, 28}); err != nil {
		return nil, err
	}

	if _, err := f.NewSheet(sheetReadme); err != nil {
		return nil, err
	}
	readme := []string{
		"Huong dan import ReqSentinel",
		"",
		"Sheet UserStories",
		"- content: bat buoc, 1 user story moi dong",
		"- acceptance_criteria: tuy chon; nhieu AC cach nhau bang dau ;",
		"",
		"Sheet PermissionMatrix",
		"- role, action, resource: bat buoc",
		"- effect: Allow hoac Deny (mac dinh Allow)",
		"- scope, condition: tuy chon",
	}
	for i, line := range readme {
		if err := f.SetCellStr(sheetReadme, fmt.Sprintf("A%d", i+1), line); err != nil {
			return nil, err
		}
	}
	title, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	if err != nil {
		return nil, err
	}
	f.SetCellStyle(sheetReadme, "A1", "A1", title)
	f.SetColWidth(sheetReadme, "A", "A", 90)

	f.SetActiveSheet(0)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type StoryRow struct {
	Content            string
	AcceptanceCriteria []string
}

type PMRow struct {
	Role, Action, Resource, Effect string
	Scope, Condition               *string
}

type ImportResult struct {
	UserStories      []StoryRow
	PermissionMatrix []PMRow
	Warnings         []string
}

func Parse(r io.Reader) (ImportResult, error) {
	var res ImportResult
	data, err := io.ReadAll(r)
	if err != nil {
		return res, err
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return res, fmt.Errorf("cannot read Excel file: %w", err)
	}
	defer f.Close()

	sheets := map[string]bool{}
	for _, s := range f.GetSheetList() {
		sheets[s] = true
	}

	if !sheets[sheetStories] {
		res.Warnings = append(res.Warnings, "Thieu sheet UserStories")
	} else if rows, err := f.GetRows(sheetStories); err != nil {
		return res, err
	} else if len(rows) == 0 {
		res.Warnings = append(res.Warnings, "Sheet UserStories trong")
	} else {
		parseStories(rows, &res)
	}

	if !sheets[sheetPM] {
		res.Warnings = append(res.Warnings, "Thieu sheet PermissionMatrix")
	} else if rows, err := f.GetRows(sheetPM); err != nil {
		return res, err
	} else if len(rows) == 0 {
		res.Warnings = append(res.Warnings, "Sheet PermissionMatrix trong")
	} else {
		parsePM(rows, &res)
	}
	return res, nil
}

func headerIndex(header []string) map[string]int {
	idx := map[string]int{}
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if _, exists := idx[key]; !exists {
			idx[key] = i
		}
	}
	return idx
}

func cell(row []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func blankRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func parseStories(rows [][]string, res *ImportResult) {
	idx := headerIndex(rows[0])
	if _, ok := idx["content"]; !ok {
		res.Warnings = append(res.Warnings, "UserStories thieu cot content")
		return
	}
	for n, row := range rows[1:] {
		if blankRow(row) {
			continue
		}
		content := cell(row, idx, "content")
		if content == "" {
			res.Warnings = append(res.Warnings, fmt.Sprintf("UserStories dong %d: content trong — bo qua", n+2))
			continue
		}
		var criteria []string
		raw := strings.ReplaceAll(cell(row, idx, "acceptance_criteria"), "\n", ";")
		for _, part := range strings.Split(raw, ";") {
			if p := strings.TrimSpace(part); p != "" {
				criteria = append(criteria, p)
			}
		}
		res.UserStories = append(res.UserStories, StoryRow{Content: content, AcceptanceCriteria: criteria})
	}
}

func parsePM(rows [][]string, res *ImportResult) {
	idx := headerIndex(rows[0])
	for _, col := range []string{"role", "action", "resource"} {
		if _, ok := idx[col]; !ok {
			res.Warnings = append(res.Warnings, "PermissionMatrix thieu role/action/resource")
			return
		}
	}
	optional := func(row []string, name string) *string {
		if v := cell(row, idx, name); v != "" {
			return &v
		}
		return nil
	}
	for n, row := range rows[1:] {
		if blankRow(row) {
			continue
		}
		role, action, resource := cell(row, idx, "role"), cell(row, idx, "action"), cell(row, idx, "resource")
		if role == "" || action == "" || resource == "" {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("PermissionMatrix dong %d: thieu role/action/resource — bo qua", n+2))
			continue
		}
		effect := "Allow"
		if e := strings.ToLower(cell(row, idx, "effect")); e == "deny" {
			effect = "Deny"
		}
		res.PermissionMatrix = append(res.PermissionMatrix, PMRow{
			Role: role, Action: action, Resource: resource, Effect: effect,
			Scope: optional(row, "scope"), Condition: optional(row, "condition"),
		})
	}
}
