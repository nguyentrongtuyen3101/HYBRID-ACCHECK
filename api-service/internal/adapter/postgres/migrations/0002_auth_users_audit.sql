-- Auth nội bộ: bảng users với 2 role admin/user.
-- Sau này nếu chuyển sang IdP ngoài (Keycloak, Auth0...) thì giữ bảng này làm
-- "local profile", map theo external_subject.
CREATE TABLE IF NOT EXISTS users (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email            varchar(255) NOT NULL,
    password_hash    text,
    full_name        varchar(255) NOT NULL DEFAULT '',
    role             varchar(20)  NOT NULL DEFAULT 'user'
                     CHECK (role IN ('admin', 'user')),
    status           varchar(20)  NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active', 'disabled')),
    external_subject varchar(255),
    last_login_at    timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email_lower ON users (lower(email));
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_external_subject ON users (external_subject)
    WHERE external_subject IS NOT NULL;

-- Mỗi project thuộc về 1 user. Project cũ (owner_id NULL) được gán cho admin
-- đầu tiên khi bootstrap.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS owner_id uuid REFERENCES users (id);
CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects (owner_id, created_at DESC);

ALTER TABLE pipeline_runs ADD COLUMN IF NOT EXISTS created_by uuid REFERENCES users (id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_created_by ON pipeline_runs (created_by, created_at DESC);

CREATE TABLE IF NOT EXISTS audit_logs (
    id          bigserial PRIMARY KEY,
    actor_id    uuid REFERENCES users (id) ON DELETE SET NULL,
    actor_email varchar(255),
    action      varchar(64) NOT NULL,
    target_type varchar(32),
    target_id   varchar(64),
    detail      jsonb,
    ip          varchar(64),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action, created_at DESC);
