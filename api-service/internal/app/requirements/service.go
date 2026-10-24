// Package requirements: user story, permission matrix, import Excel,
// từ điển thuật ngữ và phiên bản permission matrix.
package requirements

import (
	"context"
	"io"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

type Deps struct {
	Stories    app.StoryRepository
	PM         app.PMRepository
	Terms      app.TermRepository
	PMVersions app.PMVersionRepository
	Auth       app.Authorizer
	Sheets     app.Spreadsheet
	Tx         app.TxManager
	Audit      app.AuditLog
}

type Service struct{ d Deps }

func New(d Deps) *Service { return &Service{d: d} }

func (s *Service) require(ctx context.Context, actor domain.Actor, projectID string, perm domain.Permission) error {
	_, _, err := s.d.Auth.Require(ctx, actor, projectID, perm)
	return err
}

// ---------- User stories ----------

func (s *Service) AddStory(ctx context.Context, actor domain.Actor, projectID, content string, criteria []string) (domain.UserStory, error) {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return domain.UserStory{}, err
	}
	content, err := domain.ValidateStoryContent(content)
	if err != nil {
		return domain.UserStory{}, err
	}
	var id string
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		id, err = s.d.Stories.Add(ctx, projectID, content, domain.CleanCriteria(criteria))
		return err
	})
	if err != nil {
		return domain.UserStory{}, err
	}
	return s.d.Stories.Get(ctx, projectID, id)
}

func (s *Service) UpdateStory(ctx context.Context, actor domain.Actor, projectID, storyID string, content *string, criteria *[]string) (domain.UserStory, error) {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return domain.UserStory{}, err
	}
	if content != nil {
		c, err := domain.ValidateStoryContent(*content)
		if err != nil {
			return domain.UserStory{}, err
		}
		content = &c
	}
	if criteria != nil {
		clean := domain.CleanCriteria(*criteria)
		criteria = &clean
	}
	err := s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		return s.d.Stories.Update(ctx, projectID, storyID, content, criteria)
	})
	if err != nil {
		return domain.UserStory{}, app.NotFoundAs(err, "story_not_found", "User story not found")
	}
	return s.d.Stories.Get(ctx, projectID, storyID)
}

func (s *Service) DeleteStory(ctx context.Context, actor domain.Actor, projectID, storyID string) error {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return err
	}
	return app.NotFoundAs(s.d.Stories.Delete(ctx, projectID, storyID), "story_not_found", "User story not found")
}

// ---------- Permission matrix ----------

func (s *Service) AddPMRow(ctx context.Context, actor domain.Actor, projectID string, row domain.PMRow) (domain.PMRow, error) {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return row, err
	}
	if err := row.Normalize(); err != nil {
		return row, err
	}
	return s.d.PM.Add(ctx, projectID, row)
}

func (s *Service) UpdatePMRow(ctx context.Context, actor domain.Actor, projectID, rowID string, p domain.PMRowPatch) (domain.PMRow, error) {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return domain.PMRow{}, err
	}
	if p.Effect != nil && !p.Effect.Valid() {
		return domain.PMRow{}, domain.Validation("invalid_effect", "Effect must be Allow or Deny")
	}
	row, err := s.d.PM.Update(ctx, projectID, rowID, p)
	return row, app.NotFoundAs(err, "pm_row_not_found", "PM row not found")
}

func (s *Service) DeletePMRow(ctx context.Context, actor domain.Actor, projectID, rowID string) error {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return err
	}
	return app.NotFoundAs(s.d.PM.Delete(ctx, projectID, rowID), "pm_row_not_found", "PM row not found")
}

// ---------- Excel ----------

func (s *Service) ImportTemplate() ([]byte, error) { return s.d.Sheets.ImportTemplate() }

type ImportResult struct {
	Stories, Criteria, PMRows int
	Warnings                  []string
}

func (s *Service) Import(ctx context.Context, actor domain.Actor, projectID string, r io.Reader, meta app.RequestMeta) (ImportResult, error) {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return ImportResult{}, err
	}
	data, err := s.d.Sheets.ParseImport(r)
	if err != nil {
		return ImportResult{}, domain.Validation("invalid_excel", err.Error())
	}
	res := ImportResult{Warnings: data.Warnings}
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		for _, st := range data.Stories {
			criteria := domain.CleanCriteria(st.AcceptanceCriteria)
			if _, err := s.d.Stories.Add(ctx, projectID, st.Content, criteria); err != nil {
				return err
			}
			res.Stories++
			res.Criteria += len(criteria)
		}
		for _, row := range data.PMRows {
			if err := row.Normalize(); err != nil {
				res.Warnings = append(res.Warnings, err.Error())
				continue
			}
			if _, err := s.d.PM.Add(ctx, projectID, row); err != nil {
				return err
			}
			res.PMRows++
		}
		return nil
	})
	if err != nil {
		return ImportResult{}, err
	}
	if res.Warnings == nil {
		res.Warnings = []string{}
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.import_excel", "project", projectID, map[string]any{
		"user_stories": res.Stories, "acceptance_criteria": res.Criteria, "pm_rows": res.PMRows,
	}))
	return res, nil
}

// ---------- Từ điển thuật ngữ ----------

func (s *Service) Terms(ctx context.Context, actor domain.Actor, projectID string) ([]domain.Term, error) {
	if err := s.require(ctx, actor, projectID, domain.PermView); err != nil {
		return nil, err
	}
	return s.d.Terms.List(ctx, projectID)
}

func (s *Service) SaveTerm(ctx context.Context, actor domain.Actor, t domain.Term) (domain.Term, error) {
	if err := s.require(ctx, actor, t.ProjectID, domain.PermEdit); err != nil {
		return t, err
	}
	if err := t.Normalize(); err != nil {
		return t, err
	}
	if t.ID == "" {
		return s.d.Terms.Add(ctx, t)
	}
	saved, err := s.d.Terms.Update(ctx, t)
	return saved, app.NotFoundAs(err, "term_not_found", "Term not found")
}

func (s *Service) DeleteTerm(ctx context.Context, actor domain.Actor, projectID, id string) error {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return err
	}
	return app.NotFoundAs(s.d.Terms.Delete(ctx, projectID, id), "term_not_found", "Term not found")
}

// TermSuggestions: role/resource/action trong PM chưa có trong từ điển.
func (s *Service) TermSuggestions(ctx context.Context, actor domain.Actor, projectID string) ([]domain.TermSuggestion, error) {
	if err := s.require(ctx, actor, projectID, domain.PermView); err != nil {
		return nil, err
	}
	terms, err := s.d.Terms.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	rows, err := s.d.PM.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return domain.SuggestTerms(domain.NewTermNormalizer(terms), rows), nil
}

// ---------- Phiên bản permission matrix ----------

func (s *Service) PMVersions(ctx context.Context, actor domain.Actor, projectID string) ([]domain.PMVersion, error) {
	if err := s.require(ctx, actor, projectID, domain.PermView); err != nil {
		return nil, err
	}
	return s.d.PMVersions.List(ctx, projectID)
}

// DiffPMVersions so sánh 2 phiên bản; toID rỗng = so với PM hiện tại.
func (s *Service) DiffPMVersions(ctx context.Context, actor domain.Actor, projectID, fromID, toID string) (domain.PMDiff, error) {
	if err := s.require(ctx, actor, projectID, domain.PermView); err != nil {
		return domain.PMDiff{}, err
	}
	from, err := s.d.PMVersions.Get(ctx, projectID, fromID)
	if err != nil {
		return domain.PMDiff{}, app.NotFoundAs(err, "version_not_found", "Version not found")
	}
	var to []domain.PMRow
	if toID == "" {
		if to, err = s.d.PM.List(ctx, projectID); err != nil {
			return domain.PMDiff{}, err
		}
	} else {
		v, err := s.d.PMVersions.Get(ctx, projectID, toID)
		if err != nil {
			return domain.PMDiff{}, app.NotFoundAs(err, "version_not_found", "Version not found")
		}
		to = v.Rows
	}
	return domain.DiffPM(from.Rows, to), nil
}

// SaveVersion chụp lại PM hiện tại (gọi tự động trước mỗi lần phân tích, hoặc thủ công kèm nhãn).
// Không đổi gì so với phiên bản gần nhất thì dùng lại phiên bản đó.
func (s *Service) SaveVersion(ctx context.Context, actor domain.Actor, projectID string, label *string) (domain.PMVersion, error) {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return domain.PMVersion{}, err
	}
	return SnapshotPM(ctx, s.d.PM, s.d.PMVersions, projectID, &actor.ID, label)
}

func (s *Service) RestoreVersion(ctx context.Context, actor domain.Actor, projectID, versionID string, meta app.RequestMeta) error {
	if err := s.require(ctx, actor, projectID, domain.PermEdit); err != nil {
		return err
	}
	v, err := s.d.PMVersions.Get(ctx, projectID, versionID)
	if err != nil {
		return app.NotFoundAs(err, "version_not_found", "Version not found")
	}
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		// Lưu trạng thái hiện tại trước khi ghi đè để có thể quay lại
		if _, err := SnapshotPM(ctx, s.d.PM, s.d.PMVersions, projectID, &actor.ID, nil); err != nil {
			return err
		}
		return s.d.PM.ReplaceAll(ctx, projectID, v.Rows)
	})
	if err != nil {
		return err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "project.pm_restored", "project", projectID,
		map[string]any{"version": v.Version}))
	return nil
}

// SnapshotPM dùng chung với use case phân tích.
func SnapshotPM(ctx context.Context, pm app.PMRepository, versions app.PMVersionRepository, projectID string, by, label *string) (domain.PMVersion, error) {
	rows, err := pm.List(ctx, projectID)
	if err != nil {
		return domain.PMVersion{}, err
	}
	checksum := domain.PMChecksum(rows)
	latest, err := versions.Latest(ctx, projectID)
	if err != nil {
		return domain.PMVersion{}, err
	}
	if latest != nil && latest.Checksum == checksum && label == nil {
		return *latest, nil
	}
	next := 1
	if latest != nil {
		next = latest.Version + 1
	}
	for i := range rows {
		rows[i].ID = ""
	}
	return versions.Create(ctx, domain.PMVersion{ProjectID: projectID, Version: next, Checksum: checksum,
		Rows: rows, Label: label, CreatedBy: by})
}
