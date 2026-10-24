package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Project struct {
	ID          string
	Name        string
	Description *string
	OwnerID     *string // người tạo project
	OwnerEmail  *string
	CreatedAt   time.Time

	UserStoryCount int
	PMRowCount     int
	RunCount       int
	OpenIssueCount int
	MyRole         ProjectRole // role của người đang xem (điền ở tầng app)
}

func ValidateProjectName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return "", Validation("invalid_project_name", "Project name must be 1-255 characters")
	}
	return name, nil
}

// ---------- Role trong project ----------

type ProjectRole string

const (
	ProjectOwner  ProjectRole = "owner"
	ProjectEditor ProjectRole = "editor"
	ProjectViewer ProjectRole = "viewer"
)

func (r ProjectRole) Valid() bool {
	return r == ProjectOwner || r == ProjectEditor || r == ProjectViewer
}

// Permission là quyền thao tác trong 1 project.
type Permission int

const (
	PermView          Permission = iota // xem project, kết quả, báo cáo
	PermComment                         // bình luận conflict
	PermEdit                            // sửa user story / PM / từ điển, import
	PermRunAnalysis                     // chạy phân tích
	PermTriage                          // đổi trạng thái, giao conflict
	PermManageMembers                   // mời / đổi role / xoá thành viên
	PermManageProject                   // đổi tên, xoá project
)

var rolePermissions = map[ProjectRole]map[Permission]bool{
	ProjectViewer: {PermView: true, PermComment: true},
	ProjectEditor: {PermView: true, PermComment: true, PermEdit: true, PermRunAnalysis: true, PermTriage: true},
	ProjectOwner: {PermView: true, PermComment: true, PermEdit: true, PermRunAnalysis: true, PermTriage: true,
		PermManageMembers: true, PermManageProject: true},
}

func (r ProjectRole) Can(p Permission) bool { return rolePermissions[r][p] }

// EffectiveRole: admin hệ thống được coi là owner của mọi project.
func EffectiveRole(actor Actor, membership *Member) (ProjectRole, bool) {
	if actor.IsAdmin() {
		return ProjectOwner, true
	}
	if membership == nil {
		return "", false
	}
	return membership.Role, true
}

type Member struct {
	ProjectID string
	UserID    string
	Email     string
	FullName  string
	Role      ProjectRole
	AddedAt   time.Time
}

// EnsureOwnerRemains: không cho thao tác làm project mất owner cuối cùng.
func EnsureOwnerRemains(target Member, newRole *ProjectRole, ownerCount int) error {
	if target.Role != ProjectOwner {
		return nil
	}
	stillOwner := newRole != nil && *newRole == ProjectOwner
	if !stillOwner && ownerCount <= 1 {
		return Validation("last_owner", "A project must keep at least one owner")
	}
	return nil
}
