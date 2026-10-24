// Package reports: báo cáo project, ma trận bao phủ và xuất Excel.
package reports

import (
	"context"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

type Deps struct {
	Stories app.StoryRepository
	PM      app.PMRepository
	Terms   app.TermRepository
	Runs    app.RunRepository
	Issues  app.IssueRepository
	Auth    app.Authorizer
	Sheets  app.Spreadsheet
	Clock   app.Clock
}

type Service struct{ d Deps }

func New(d Deps) *Service { return &Service{d: d} }

func (s *Service) Coverage(ctx context.Context, actor domain.Actor, projectID string) (domain.Coverage, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermView); err != nil {
		return domain.Coverage{}, err
	}
	cov, _, err := s.coverage(ctx, projectID)
	return cov, err
}

// coverage so khớp PM hiện tại (đã chuẩn hoá theo từ điển) với alignment của lần chạy mới nhất mỗi story.
func (s *Service) coverage(ctx context.Context, projectID string) (domain.Coverage, []domain.RunDetail, error) {
	rows, err := s.d.PM.List(ctx, projectID)
	if err != nil {
		return domain.Coverage{}, nil, err
	}
	terms, err := s.d.Terms.List(ctx, projectID)
	if err != nil {
		return domain.Coverage{}, nil, err
	}
	n := domain.NewTermNormalizer(terms)
	normalized := make([]domain.PMRow, len(rows))
	for i, r := range rows {
		normalized[i] = n.NormalizeRow(r)
	}
	latest, err := s.d.Runs.LatestPerStory(ctx, projectID)
	if err != nil {
		return domain.Coverage{}, nil, err
	}
	stories := make([]domain.StoryAlignments, 0, len(latest))
	for _, run := range latest {
		if run.UserStoryID != nil {
			stories = append(stories, domain.StoryAlignments{StoryID: *run.UserStoryID, Alignments: run.Alignments})
		}
	}
	return domain.ComputeCoverage(normalized, stories), latest, nil
}

func (s *Service) Report(ctx context.Context, actor domain.Actor, projectID string) (domain.ProjectReport, error) {
	p, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermView)
	if err != nil {
		return domain.ProjectReport{}, err
	}
	stories, err := s.d.Stories.List(ctx, projectID)
	if err != nil {
		return domain.ProjectReport{}, err
	}
	cov, latest, err := s.coverage(ctx, projectID)
	if err != nil {
		return domain.ProjectReport{}, err
	}
	issues, _, err := s.d.Issues.List(ctx, domain.IssueFilter{ProjectID: projectID, Limit: 10000})
	if err != nil {
		return domain.ProjectReport{}, err
	}
	latestByStory := map[string]domain.RunSummary{}
	for _, r := range latest {
		if r.UserStoryID != nil {
			latestByStory[*r.UserStoryID] = domain.RunSummary{ID: r.ID, IsConsistent: r.IsConsistent, CreatedAt: r.CreatedAt}
		}
	}
	return domain.BuildReport(p, stories, latestByStory, issues, cov, s.d.Clock()), nil
}

func (s *Service) ReportExcel(ctx context.Context, actor domain.Actor, projectID string) ([]byte, string, error) {
	rep, err := s.Report(ctx, actor, projectID)
	if err != nil {
		return nil, "", err
	}
	data, err := s.d.Sheets.ProjectReport(rep)
	return data, rep.Project.Name, err
}
