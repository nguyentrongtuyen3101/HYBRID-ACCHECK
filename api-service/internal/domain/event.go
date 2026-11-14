package domain

import "time"

// Event là sự kiện nghiệp vụ gửi sang notification service.
// Tự chứa đủ dữ liệu (người nhận, email, tên project...) để service khác không phải gọi ngược lại.
type Event struct {
	ID         string
	Type       string
	OccurredAt time.Time
	Actor      *Actor
	Recipients []string // user id nhận thông báo in-app
	Data       map[string]any
}

const (
	EventMemberInvited          = "project.member_invited"
	EventMemberJoined           = "project.member_joined"
	EventMemberRoleChanged      = "project.member_role_changed"
	EventMemberRemoved          = "project.member_removed"
	EventIssueAssigned          = "conflict.assigned"
	EventIssueStatusChanged     = "conflict.status_changed"
	EventIssueCommented         = "conflict.commented"
	EventJobProgress            = "analysis.job_progress"
	EventJobFinished            = "analysis.job_finished"
	EventPasswordResetRequested = "auth.password_reset_requested"

	// Thao tác tác động lên chính tài khoản người dùng → luôn báo qua email.
	EventAccountCreated       = "account.created"          // admin tạo tài khoản
	EventAccountRoleChanged   = "account.role_changed"     // admin đổi vai trò hệ thống
	EventAccountStatusChanged = "account.status_changed"   // admin khóa / mở khóa
	EventPasswordChanged      = "account.password_changed" // tự đổi, đặt lại qua link, hoặc admin đặt lại
)

// RecipientData: thông tin người nhận email đi kèm sự kiện (notification-service không đọc bảng users).
func RecipientData(email, name string, locale Locale) map[string]any {
	return map[string]any{"email": email, "name": name, "locale": locale}
}

// Recipients loại người thực hiện và id trùng.
func Recipients(actorID string, ids ...*string) []string {
	seen := map[string]bool{actorID: true}
	out := []string{}
	for _, id := range ids {
		if id != nil && *id != "" && !seen[*id] {
			seen[*id] = true
			out = append(out, *id)
		}
	}
	return out
}

// ---------- Audit ----------

type AuditEntry struct {
	ID         int64
	ActorID    *string
	ActorEmail *string
	Action     string
	TargetType *string
	TargetID   *string
	Detail     map[string]any
	IP         *string
	CreatedAt  time.Time
}

type AuditFilter struct {
	Action  string
	ActorID *string
	Limit   int
	Offset  int
}

type DailyCount struct {
	Day   string
	Count int
}

type AdminStats struct {
	UsersTotal, UsersActive, UsersDisabled, Admins int
	Projects                                       int
	RunsTotal, RunsLast7Days, RunsConsistent       int
	OpenIssues                                     int
	ConflictsByType                                map[string]int
	RunsByModel                                    map[string]int
	RunsPerDay                                     []DailyCount
}
