package app

import (
	"context"

	"reqsentinel/api/internal/domain"
)

// Authorizer kiểm tra quyền của actor trong 1 project (dùng chung cho mọi use case).
type Authorizer struct {
	Projects ProjectRepository
	Members  MemberRepository
}

// Require trả về project + role nếu actor có quyền perm.
// Không phải thành viên → NotFound (không lộ sự tồn tại của project).
func (a Authorizer) Require(ctx context.Context, actor domain.Actor, projectID string, perm domain.Permission) (domain.Project, domain.ProjectRole, error) {
	p, err := a.Projects.Get(ctx, projectID)
	if err != nil {
		return p, "", notFoundAs(err, "project_not_found", "Project not found")
	}
	m, err := a.Members.Get(ctx, projectID, actor.ID)
	if err != nil {
		return p, "", err
	}
	role, ok := domain.EffectiveRole(actor, m)
	if !ok {
		return p, "", domain.NotFound("project_not_found", "Project not found")
	}
	if !role.Can(perm) {
		return p, role, domain.Forbidden("project_forbidden", "You do not have permission to do this in this project")
	}
	p.MyRole = role
	return p, role, nil
}

// notFoundAs đổi lỗi NotFound chung của repository thành thông báo cụ thể.
func notFoundAs(err error, code, msg string) error {
	if domain.IsNotFound(err) {
		return domain.NotFound(code, msg)
	}
	return err
}

// NotFoundAs dùng cho các package use case con.
func NotFoundAs(err error, code, msg string) error { return notFoundAs(err, code, msg) }

// Audit gom thông tin chung của 1 dòng audit log.
func Audit(actor *domain.Actor, ip, action, targetType, targetID string, detail map[string]any) domain.AuditEntry {
	e := domain.AuditEntry{Action: action, Detail: detail}
	if actor != nil {
		e.ActorID, e.ActorEmail = &actor.ID, &actor.Email
	}
	if ip != "" {
		e.IP = &ip
	}
	if targetType != "" {
		e.TargetType = &targetType
	}
	if targetID != "" {
		e.TargetID = &targetID
	}
	return e
}

// RequestMeta là thông tin request mà use case cần cho audit (không phụ thuộc HTTP).
type RequestMeta struct {
	IP string
}

// NewEvent tạo sự kiện với id/thời điểm từ IDGenerator/Clock.
func NewEvent(newID IDGenerator, now Clock, typ string, actor *domain.Actor, recipients []string, data map[string]any) domain.Event {
	if recipients == nil {
		recipients = []string{}
	}
	return domain.Event{ID: newID(), Type: typ, OccurredAt: now(), Actor: actor, Recipients: recipients, Data: data}
}
