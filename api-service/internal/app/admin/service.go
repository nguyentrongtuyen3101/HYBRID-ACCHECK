// Package admin: thống kê hệ thống, audit log, xuất dataset từ phản hồi của BA.
package admin

import (
	"context"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

type Deps struct {
	Stats app.StatsRepository
	Audit app.AuditRepository
}

type Service struct{ d Deps }

func New(d Deps) *Service { return &Service{d: d} }

func (s *Service) Stats(ctx context.Context) (domain.AdminStats, error) {
	return s.d.Stats.AdminStats(ctx)
}

func (s *Service) AuditLogs(ctx context.Context, f domain.AuditFilter) ([]domain.AuditEntry, int, error) {
	return s.d.Audit.List(ctx, f)
}

// DatasetExport chuyển các story đã được BA đánh giá thành dataset cùng định dạng
// data/processed/*.json của ml-service, để train lại model.
func (s *Service) DatasetExport(ctx context.Context) ([]domain.DatasetItem, error) {
	reviewed, err := s.d.Stats.ReviewedStories(ctx)
	if err != nil {
		return nil, err
	}
	items := []domain.DatasetItem{}
	for _, rs := range reviewed {
		label, ok := domain.LabelFromIssues(rs.Issues)
		if !ok {
			continue
		}
		items = append(items, domain.DatasetItem{
			ID:                 "FB_" + rs.StoryID[:8],
			UserStory:          rs.Run.UserStory,
			AcceptanceCriteria: rs.Run.AcceptanceCriteria,
			PermissionMatrix:   rs.Run.PermissionMatrix,
			ConflictLabel:      label,
			Notes:              "project " + rs.ProjectID + ", run " + rs.Run.ID,
		})
	}
	return items, nil
}
