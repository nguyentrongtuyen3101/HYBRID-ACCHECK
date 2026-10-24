-- 1) Chuẩn hoá các cột trạng thái / role / loại từ varchar + CHECK sang ENUM của PostgreSQL.
--    Dữ liệu cũ được chuyển nguyên giá trị (giá trị lạ sẽ làm migration dừng thay vì mất dữ liệu).
-- 2) Thêm thông tin cá nhân + tuỳ chọn giao diện cho trang Profile.

DO $$
BEGIN
    CREATE TYPE user_role AS ENUM ('admin', 'user');
    CREATE TYPE user_status AS ENUM ('active', 'disabled');
    CREATE TYPE project_role AS ENUM ('owner', 'editor', 'viewer');
    CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'revoked');
    CREATE TYPE term_kind AS ENUM ('role', 'resource', 'action');
    CREATE TYPE job_status AS ENUM ('queued', 'running', 'completed', 'failed');
    CREATE TYPE issue_status AS ENUM ('open', 'acknowledged', 'resolved', 'false_positive');
    CREATE TYPE issue_activity_kind AS ENUM ('detected', 'reopened', 'auto_resolved', 'status_changed', 'assigned', 'comment');
    CREATE TYPE permission_effect AS ENUM ('Allow', 'Deny');
    CREATE TYPE conflict_type AS ENUM ('Normal', 'Missing Permission', 'Action Conflict', 'Effect Conflict',
                                       'Scope Conflict', 'Condition Conflict');
    CREATE TYPE artifact_source AS ENUM ('User Story', 'Acceptance Criteria', 'Permission Matrix');
    CREATE TYPE ui_locale AS ENUM ('vi', 'en', 'ja');
    CREATE TYPE ui_theme AS ENUM ('light', 'dark');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ---------- users ----------
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check, DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users ALTER COLUMN role DROP DEFAULT, ALTER COLUMN status DROP DEFAULT;
ALTER TABLE users
    ALTER COLUMN role TYPE user_role USING role::text::user_role,
    ALTER COLUMN status TYPE user_status USING status::text::user_status;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'user', ALTER COLUMN status SET DEFAULT 'active';

-- ---------- thành viên & lời mời ----------
ALTER TABLE project_members DROP CONSTRAINT IF EXISTS project_members_role_check;
ALTER TABLE project_members ALTER COLUMN role TYPE project_role USING role::text::project_role;

ALTER TABLE project_invitations
    DROP CONSTRAINT IF EXISTS project_invitations_role_check,
    DROP CONSTRAINT IF EXISTS project_invitations_status_check;
ALTER TABLE project_invitations ALTER COLUMN status DROP DEFAULT;
ALTER TABLE project_invitations
    ALTER COLUMN role TYPE project_role USING role::text::project_role,
    ALTER COLUMN status TYPE invitation_status USING status::text::invitation_status;
ALTER TABLE project_invitations ALTER COLUMN status SET DEFAULT 'pending';

-- ---------- yêu cầu ----------
ALTER TABLE user_stories ALTER COLUMN source DROP DEFAULT;
ALTER TABLE user_stories ALTER COLUMN source TYPE artifact_source USING source::text::artifact_source;
ALTER TABLE user_stories ALTER COLUMN source SET DEFAULT 'User Story';

ALTER TABLE permission_matrix_rows ALTER COLUMN effect TYPE permission_effect USING effect::text::permission_effect;

ALTER TABLE project_terms DROP CONSTRAINT IF EXISTS project_terms_kind_check;
ALTER TABLE project_terms ALTER COLUMN kind TYPE term_kind USING kind::text::term_kind;

-- ---------- phân tích ----------
ALTER TABLE analysis_jobs DROP CONSTRAINT IF EXISTS analysis_jobs_status_check;
ALTER TABLE analysis_jobs ALTER COLUMN status TYPE job_status USING status::text::job_status;

ALTER TABLE extracted_entities
    ALTER COLUMN effect TYPE permission_effect USING effect::text::permission_effect,
    ALTER COLUMN source TYPE artifact_source USING source::text::artifact_source;

ALTER TABLE conflict_records ALTER COLUMN conflict_type TYPE conflict_type USING conflict_type::text::conflict_type;

-- ---------- xử lý conflict ----------
ALTER TABLE conflict_issues DROP CONSTRAINT IF EXISTS conflict_issues_status_check;
ALTER TABLE conflict_issues ALTER COLUMN status DROP DEFAULT;
ALTER TABLE conflict_issues
    ALTER COLUMN status TYPE issue_status USING status::text::issue_status,
    ALTER COLUMN conflict_type TYPE conflict_type USING conflict_type::text::conflict_type;
ALTER TABLE conflict_issues ALTER COLUMN status SET DEFAULT 'open';

ALTER TABLE conflict_activities ALTER COLUMN kind TYPE issue_activity_kind USING kind::text::issue_activity_kind;

-- ---------- Profile ----------
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS job_title    varchar(100),
    ADD COLUMN IF NOT EXISTS organization varchar(150),
    ADD COLUMN IF NOT EXISTS phone        varchar(30),
    ADD COLUMN IF NOT EXISTS bio          text,
    ADD COLUMN IF NOT EXISTS locale       ui_locale,
    ADD COLUMN IF NOT EXISTS theme        ui_theme;
