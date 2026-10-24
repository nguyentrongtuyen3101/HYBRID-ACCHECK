-- Nghiệp vụ giai đoạn 1+2: thành viên project, lời mời, quên mật khẩu,
-- từ điển thuật ngữ, phiên bản permission matrix, job phân tích,
-- quy trình xử lý conflict và outbox sự kiện.

-- ---------- Thành viên & lời mời ----------
CREATE TABLE IF NOT EXISTS project_members (
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       varchar(20) NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_project_members_user ON project_members (user_id);

-- Chủ project hiện có trở thành owner
INSERT INTO project_members (project_id, user_id, role)
SELECT id, owner_id, 'owner' FROM projects WHERE owner_id IS NOT NULL
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS project_invitations (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    email        varchar(255) NOT NULL,
    role         varchar(20) NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    token_hash   varchar(64) NOT NULL UNIQUE,
    status       varchar(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'revoked')),
    invited_by   uuid NOT NULL REFERENCES users (id),
    expires_at   timestamptz NOT NULL,
    responded_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_invitations_project ON project_invitations (project_id, status);

CREATE TABLE IF NOT EXISTS password_resets (
    token_hash varchar(64) PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- ---------- Từ điển thuật ngữ ----------
CREATE TABLE IF NOT EXISTS project_terms (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    kind        varchar(20) NOT NULL CHECK (kind IN ('role', 'resource', 'action')),
    canonical   varchar(255) NOT NULL,
    synonyms    text[] NOT NULL DEFAULT '{}',
    description text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_project_terms ON project_terms (project_id, kind, lower(canonical));

-- ---------- Phiên bản permission matrix ----------
CREATE TABLE IF NOT EXISTS pm_versions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    version    integer NOT NULL,
    checksum   varchar(64) NOT NULL,
    rows       jsonb NOT NULL,
    label      varchar(255),
    created_by uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, version)
);

-- ---------- Job phân tích cả project ----------
CREATE TABLE IF NOT EXISTS analysis_jobs (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    created_by    uuid NOT NULL REFERENCES users (id),
    status        varchar(20) NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    total         integer NOT NULL DEFAULT 0,
    processed     integer NOT NULL DEFAULT 0,
    failed        integer NOT NULL DEFAULT 0,
    error         text,
    model_version varchar(100),
    created_at    timestamptz NOT NULL DEFAULT now(),
    started_at    timestamptz,
    finished_at   timestamptz
);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_queue ON analysis_jobs (status, created_at);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_project ON analysis_jobs (project_id, created_at DESC);

-- ---------- Run: gắn user story, job, phiên bản PM, phiên bản model ----------
ALTER TABLE pipeline_runs ADD COLUMN IF NOT EXISTS user_story_id uuid;
ALTER TABLE pipeline_runs ADD COLUMN IF NOT EXISTS job_id uuid;
ALTER TABLE pipeline_runs ADD COLUMN IF NOT EXISTS pm_version_id uuid;
ALTER TABLE pipeline_runs ADD COLUMN IF NOT EXISTS model_version varchar(100);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_story ON pipeline_runs (project_id, user_story_id, created_at DESC);

-- ---------- Quy trình xử lý conflict ----------
CREATE TABLE IF NOT EXISTS conflict_issues (
    id                uuid PRIMARY KEY,
    project_id        uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_story_id     uuid NOT NULL REFERENCES user_stories (id) ON DELETE CASCADE,
    fingerprint       varchar(64) NOT NULL,
    conflict_type     varchar(50) NOT NULL,
    explanation       text NOT NULL,
    evidence_us_ac    text,
    evidence_pm       text,
    role              varchar(150),
    action            varchar(100),
    resource          varchar(150),
    status            varchar(20) NOT NULL DEFAULT 'open'
                      CHECK (status IN ('open', 'acknowledged', 'resolved', 'false_positive')),
    assignee_id       uuid REFERENCES users (id) ON DELETE SET NULL,
    first_seen_run_id uuid NOT NULL,
    last_seen_run_id  uuid NOT NULL,
    last_seen_at      timestamptz NOT NULL,
    resolved_by       uuid REFERENCES users (id) ON DELETE SET NULL,
    resolved_at       timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, user_story_id, fingerprint)
);
CREATE INDEX IF NOT EXISTS idx_conflict_issues_project ON conflict_issues (project_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_conflict_issues_assignee ON conflict_issues (assignee_id) WHERE assignee_id IS NOT NULL;

ALTER TABLE conflict_records ADD COLUMN IF NOT EXISTS issue_id uuid REFERENCES conflict_issues (id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS conflict_activities (
    id         uuid PRIMARY KEY,
    issue_id   uuid NOT NULL REFERENCES conflict_issues (id) ON DELETE CASCADE,
    actor_id   uuid REFERENCES users (id) ON DELETE SET NULL,
    kind       varchar(32) NOT NULL,
    body       text,
    from_value varchar(255),
    to_value   varchar(255),
    run_id     uuid,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_conflict_activities_issue ON conflict_activities (issue_id, created_at);

-- ---------- Outbox: sự kiện chờ đẩy sang RabbitMQ ----------
CREATE TABLE IF NOT EXISTS outbox_events (
    id           uuid PRIMARY KEY,
    event_type   varchar(64) NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    attempts     integer NOT NULL DEFAULT 0,
    last_error   text
);
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events (created_at) WHERE published_at IS NULL;
