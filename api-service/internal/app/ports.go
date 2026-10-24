// Package app chứa use case. Chỉ phụ thuộc domain và các port (interface)
// khai báo ở đây; adapter (postgres, http, rabbitmq, ml...) hiện thực các port này.
package app

import (
	"context"
	"io"
	"time"

	"reqsentinel/api/internal/domain"
)

// ---------- Hạ tầng chung ----------

// TxManager chạy fn trong 1 transaction; repository tự lấy tx từ ctx.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// EventPublisher ghi sự kiện (outbox) trong cùng transaction với thay đổi dữ liệu.
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.Event) error
}

// AuditLog ghi nhật ký; không tham gia transaction và không làm hỏng thao tác chính khi lỗi.
type AuditLog interface {
	Record(ctx context.Context, e domain.AuditEntry)
}

type Clock func() time.Time

type IDGenerator func() string

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash *string, password string) bool
}

type TokenIssuer interface {
	Issue(userID string, role domain.SystemRole) (token string, expiresAt time.Time, err error)
	Verify(token string) (userID string, err error)
}

// OpaqueTokens sinh token ngẫu nhiên (mời thành viên, reset mật khẩu); DB chỉ lưu hash.
type OpaqueTokens interface {
	New() (raw, hash string)
	Hash(raw string) string
}

type LoginLimiter interface {
	Blocked(key string) bool
	Fail(key string)
	Reset(key string)
}

// Analyzer là ML service.
type Analyzer interface {
	Analyze(ctx context.Context, in domain.AnalysisInput) (domain.AnalysisResult, error)
}

type Spreadsheet interface {
	ImportTemplate() ([]byte, error)
	ParseImport(r io.Reader) (ImportData, error)
	ProjectReport(rep domain.ProjectReport) ([]byte, error)
}

type ImportData struct {
	Stories  []ImportStory
	PMRows   []domain.PMRow
	Warnings []string
}

type ImportStory struct {
	Content            string
	AcceptanceCriteria []string
}

// ---------- Repository ----------

type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	List(ctx context.Context, f domain.UserFilter) ([]domain.User, int, error)
	Update(ctx context.Context, id string, p domain.UserPatch) (domain.User, error)
	UpdateProfile(ctx context.Context, id string, p domain.ProfilePatch) (domain.User, error)
	SetPassword(ctx context.Context, id, hash string) error
	TouchLastLogin(ctx context.Context, id string) error
	CountActiveAdmins(ctx context.Context) (int, error)
}

type PasswordResetRepository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	// Consume đánh dấu token đã dùng, trả về user id; lỗi nếu không hợp lệ/hết hạn.
	Consume(ctx context.Context, tokenHash string, now time.Time) (string, error)
}

type ProjectRepository interface {
	// List: memberID nil = mọi project; khác nil = project user là thành viên (kèm MyRole).
	List(ctx context.Context, memberID *string) ([]domain.Project, error)
	Get(ctx context.Context, id string) (domain.Project, error)
	Create(ctx context.Context, ownerID, name string, description *string) (domain.Project, error)
	Update(ctx context.Context, id string, name, description *string) error
	Delete(ctx context.Context, id string) error
	AssignOrphansTo(ctx context.Context, userID string) (int64, error)
}

type MemberRepository interface {
	Get(ctx context.Context, projectID, userID string) (*domain.Member, error)
	List(ctx context.Context, projectID string) ([]domain.Member, error)
	Add(ctx context.Context, m domain.Member) error
	UpdateRole(ctx context.Context, projectID, userID string, role domain.ProjectRole) error
	Remove(ctx context.Context, projectID, userID string) error
	CountOwners(ctx context.Context, projectID string) (int, error)
}

type InvitationRepository interface {
	Create(ctx context.Context, inv domain.Invitation, tokenHash string) (domain.Invitation, error)
	Get(ctx context.Context, id string) (domain.Invitation, error)
	GetByTokenHash(ctx context.Context, hash string) (domain.Invitation, error)
	ListPending(ctx context.Context, projectID string) ([]domain.Invitation, error)
	FindPending(ctx context.Context, projectID, email string) (*domain.Invitation, error)
	SetStatus(ctx context.Context, id string, status domain.InvitationStatus, at time.Time) error
}

type StoryRepository interface {
	List(ctx context.Context, projectID string) ([]domain.UserStory, error)
	Get(ctx context.Context, projectID, storyID string) (domain.UserStory, error)
	Add(ctx context.Context, projectID, content string, criteria []string) (string, error)
	Update(ctx context.Context, projectID, storyID string, content *string, criteria *[]string) error
	Delete(ctx context.Context, projectID, storyID string) error
}

type PMRepository interface {
	List(ctx context.Context, projectID string) ([]domain.PMRow, error)
	Add(ctx context.Context, projectID string, r domain.PMRow) (domain.PMRow, error)
	Update(ctx context.Context, projectID, rowID string, p domain.PMRowPatch) (domain.PMRow, error)
	Delete(ctx context.Context, projectID, rowID string) error
	ReplaceAll(ctx context.Context, projectID string, rows []domain.PMRow) error
}

type PMVersionRepository interface {
	Latest(ctx context.Context, projectID string) (*domain.PMVersion, error)
	Create(ctx context.Context, v domain.PMVersion) (domain.PMVersion, error)
	List(ctx context.Context, projectID string) ([]domain.PMVersion, error)
	Get(ctx context.Context, projectID, id string) (domain.PMVersion, error)
}

type TermRepository interface {
	List(ctx context.Context, projectID string) ([]domain.Term, error)
	Add(ctx context.Context, t domain.Term) (domain.Term, error)
	Update(ctx context.Context, t domain.Term) (domain.Term, error)
	Delete(ctx context.Context, projectID, id string) error
}

type RunRepository interface {
	Save(ctx context.Context, run domain.Run) (string, error)
	List(ctx context.Context, f domain.RunFilter) ([]domain.RunSummary, error)
	Get(ctx context.Context, id string) (domain.RunDetail, error)
	// LatestPerStory: lần chạy mới nhất của mỗi user story trong project.
	LatestPerStory(ctx context.Context, projectID string) ([]domain.RunDetail, error)
}

type IssueRepository interface {
	ListByStory(ctx context.Context, projectID, storyID string) ([]domain.Issue, error)
	List(ctx context.Context, f domain.IssueFilter) ([]domain.Issue, int, error)
	Get(ctx context.Context, projectID, id string) (domain.Issue, error)
	Create(ctx context.Context, issues []domain.Issue) error
	Update(ctx context.Context, issues []domain.Issue) error
	AddActivities(ctx context.Context, acts []domain.IssueActivity) error
	Activities(ctx context.Context, issueID string) ([]domain.IssueActivity, error)
	// Participants: những người đã bình luận / thao tác trên issue.
	Participants(ctx context.Context, issueID string) ([]string, error)
}

type JobRepository interface {
	Create(ctx context.Context, j domain.AnalysisJob) (domain.AnalysisJob, error)
	Get(ctx context.Context, id string) (domain.AnalysisJob, error)
	ListByProject(ctx context.Context, projectID string, limit int) ([]domain.AnalysisJob, error)
	ActiveForProject(ctx context.Context, projectID string) (*domain.AnalysisJob, error)
	// ClaimNext lấy 1 job đang chờ và chuyển sang running (an toàn khi nhiều worker).
	ClaimNext(ctx context.Context, now time.Time) (*domain.AnalysisJob, error)
	Save(ctx context.Context, j domain.AnalysisJob) error
	FailInterrupted(ctx context.Context, now time.Time) (int64, error)
}

type AuditRepository interface {
	Insert(ctx context.Context, e domain.AuditEntry) error
	List(ctx context.Context, f domain.AuditFilter) ([]domain.AuditEntry, int, error)
}

type StatsRepository interface {
	AdminStats(ctx context.Context) (domain.AdminStats, error)
	ReviewedStories(ctx context.Context) ([]ReviewedStory, error)
}

// ReviewedStory: lần chạy mới nhất của 1 story kèm các issue (để xuất dataset).
type ReviewedStory struct {
	ProjectID string
	StoryID   string
	Run       domain.RunDetail
	Issues    []domain.Issue
}
