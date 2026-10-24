// Package conflicts: quy trình xử lý conflict (trạng thái, giao người, bình luận).
package conflicts

import (
	"context"
	"time"

	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/domain"
)

type Deps struct {
	Issues  app.IssueRepository
	Members app.MemberRepository
	Auth    app.Authorizer
	Events  app.EventPublisher
	Tx      app.TxManager
	Audit   app.AuditLog
	Clock   app.Clock
	NewID   app.IDGenerator
}

type Service struct{ d Deps }

func New(d Deps) *Service { return &Service{d: d} }

func (s *Service) List(ctx context.Context, actor domain.Actor, f domain.IssueFilter) ([]domain.Issue, int, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, f.ProjectID, domain.PermView); err != nil {
		return nil, 0, err
	}
	if f.Status != "" && !domain.IssueStatus(f.Status).Valid() && f.Status != "active" {
		return nil, 0, domain.Validation("invalid_status_filter", "Invalid status filter")
	}
	return s.d.Issues.List(ctx, f)
}

type Detail struct {
	Issue      domain.Issue
	Activities []domain.IssueActivity
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, projectID, issueID string) (Detail, error) {
	if _, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermView); err != nil {
		return Detail{}, err
	}
	iss, err := s.d.Issues.Get(ctx, projectID, issueID)
	if err != nil {
		return Detail{}, app.NotFoundAs(err, "conflict_not_found", "Conflict not found")
	}
	acts, err := s.d.Issues.Activities(ctx, issueID)
	return Detail{Issue: iss, Activities: acts}, err
}

type UpdateInput struct {
	Status      *domain.IssueStatus
	AssigneeSet bool    // có gửi field assignee_id
	AssigneeID  *string // nil = bỏ giao
}

func (s *Service) Update(ctx context.Context, actor domain.Actor, projectID, issueID string, in UpdateInput, meta app.RequestMeta) (domain.Issue, error) {
	p, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermTriage)
	if err != nil {
		return domain.Issue{}, err
	}
	var iss domain.Issue
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		iss, err = s.d.Issues.Get(ctx, projectID, issueID)
		if err != nil {
			return app.NotFoundAs(err, "conflict_not_found", "Conflict not found")
		}
		now := s.d.Clock()
		var acts []domain.IssueActivity
		var events []domain.Event
		actorID := actor.ID

		if in.Status != nil && *in.Status != iss.Status {
			if !iss.Status.CanTransitionTo(*in.Status) {
				return domain.Validation("invalid_transition", "Cannot change status from "+string(iss.Status)+" to "+string(*in.Status)).
					With(map[string]any{"from": iss.Status, "to": *in.Status})
			}
			from, to := string(iss.Status), string(*in.Status)
			iss.Status = *in.Status
			if iss.Status == domain.IssueResolved {
				iss.ResolvedBy, iss.ResolvedAt = &actorID, &now
			} else {
				iss.ResolvedBy, iss.ResolvedAt = nil, nil
			}
			acts = append(acts, s.activity(iss.ID, actor, domain.ActivityStatusChanged, nil, &from, &to, now))
			watchers, err := s.d.Issues.Participants(ctx, iss.ID)
			if err != nil {
				return err
			}
			events = append(events, s.event(domain.EventIssueStatusChanged, actor, p.Name, iss,
				recipients(actor.ID, iss.AssigneeID, watchers), map[string]any{"from": from, "to": to}))
		}

		if in.AssigneeSet && !sameID(in.AssigneeID, iss.AssigneeID) {
			if in.AssigneeID != nil {
				m, err := s.d.Members.Get(ctx, projectID, *in.AssigneeID)
				if err != nil {
					return err
				}
				if m == nil || !m.Role.Can(domain.PermTriage) {
					return domain.Validation("invalid_assignee", "Assignee must be an owner or editor of this project")
				}
			}
			iss.AssigneeID = in.AssigneeID
			acts = append(acts, s.activity(iss.ID, actor, domain.ActivityAssigned, nil, nil, in.AssigneeID, now))
			if in.AssigneeID != nil {
				events = append(events, s.event(domain.EventIssueAssigned, actor, p.Name, iss,
					domain.Recipients(actor.ID, in.AssigneeID), nil))
			}
		}

		if len(acts) == 0 {
			return nil
		}
		iss.UpdatedAt = now
		if err := s.d.Issues.Update(ctx, []domain.Issue{iss}); err != nil {
			return err
		}
		if err := s.d.Issues.AddActivities(ctx, acts); err != nil {
			return err
		}
		return s.d.Events.Publish(ctx, events...)
	})
	if err != nil {
		return iss, err
	}
	s.d.Audit.Record(ctx, app.Audit(&actor, meta.IP, "conflict.update", "conflict", iss.ID,
		map[string]any{"status": iss.Status, "assignee_id": iss.AssigneeID}))
	return s.d.Issues.Get(ctx, projectID, issueID)
}

func (s *Service) Comment(ctx context.Context, actor domain.Actor, projectID, issueID, body string) (domain.IssueActivity, error) {
	p, _, err := s.d.Auth.Require(ctx, actor, projectID, domain.PermComment)
	if err != nil {
		return domain.IssueActivity{}, err
	}
	body, err = domain.ValidateComment(body)
	if err != nil {
		return domain.IssueActivity{}, err
	}
	var act domain.IssueActivity
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		iss, err := s.d.Issues.Get(ctx, projectID, issueID)
		if err != nil {
			return app.NotFoundAs(err, "conflict_not_found", "Conflict not found")
		}
		watchers, err := s.d.Issues.Participants(ctx, iss.ID)
		if err != nil {
			return err
		}
		act = s.activity(iss.ID, actor, domain.ActivityComment, &body, nil, nil, s.d.Clock())
		if err := s.d.Issues.AddActivities(ctx, []domain.IssueActivity{act}); err != nil {
			return err
		}
		return s.d.Events.Publish(ctx, s.event(domain.EventIssueCommented, actor, p.Name, iss,
			recipients(actor.ID, iss.AssigneeID, watchers), map[string]any{"comment": truncate(body, 200)}))
	})
	return act, err
}

func (s *Service) activity(issueID string, actor domain.Actor, kind domain.ActivityKind, body, from, to *string, at time.Time) domain.IssueActivity {
	id, name := actor.ID, actor.DisplayName()
	return domain.IssueActivity{ID: s.d.NewID(), IssueID: issueID, ActorID: &id, ActorName: &name, Kind: kind,
		Body: body, FromValue: from, ToValue: to, CreatedAt: at}
}

func (s *Service) event(typ string, actor domain.Actor, projectName string, iss domain.Issue, to []string, extra map[string]any) domain.Event {
	data := map[string]any{
		"project_id": iss.ProjectID, "project_name": projectName, "issue_id": iss.ID,
		"conflict_type": iss.Type, "status": iss.Status, "explanation": truncate(iss.Explanation, 200),
	}
	for k, v := range extra {
		data[k] = v
	}
	return app.NewEvent(s.d.NewID, s.d.Clock, typ, &actor, to, data)
}

func recipients(actorID string, assignee *string, watchers []string) []string {
	ids := []*string{assignee}
	for i := range watchers {
		ids = append(ids, &watchers[i])
	}
	return domain.Recipients(actorID, ids...)
}

func sameID(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
