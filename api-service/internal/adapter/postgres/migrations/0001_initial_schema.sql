-- Schema gốc (trước đây tạo bởi SQLAlchemy scripts/init_db.py).
-- Dùng IF NOT EXISTS để chạy an toàn trên DB đã có dữ liệu từ bản Python.

CREATE TABLE IF NOT EXISTS projects (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        varchar(255) NOT NULL,
    description text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_stories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid REFERENCES projects (id),
    content    text NOT NULL,
    source     varchar(50) NOT NULL DEFAULT 'User Story',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS acceptance_criteria (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_story_id uuid NOT NULL REFERENCES user_stories (id),
    content       text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS permission_matrix_rows (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid REFERENCES projects (id),
    role       varchar(150) NOT NULL,
    action     varchar(100) NOT NULL,
    resource   varchar(150) NOT NULL,
    effect     varchar(20) NOT NULL,
    condition  text,
    scope      varchar(100),
    raw_text   text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          uuid REFERENCES projects (id),
    user_story_content  text NOT NULL,
    acceptance_criteria json,
    is_consistent       boolean NOT NULL DEFAULT true,
    summary             text,
    raw_result          json,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS extracted_entities (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_run_id uuid NOT NULL REFERENCES pipeline_runs (id),
    role            varchar(150),
    action          varchar(100),
    resource        varchar(150),
    effect          varchar(20),
    condition       text,
    scope           varchar(100),
    source          varchar(50) NOT NULL,
    confidence      double precision NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS conflict_records (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_run_id uuid NOT NULL REFERENCES pipeline_runs (id),
    conflict_type   varchar(50) NOT NULL,
    confidence      double precision NOT NULL DEFAULT 0,
    explanation     text NOT NULL,
    evidence_us_ac  text,
    evidence_pm     text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_stories_project ON user_stories (project_id);
CREATE INDEX IF NOT EXISTS idx_acceptance_criteria_story ON acceptance_criteria (user_story_id);
CREATE INDEX IF NOT EXISTS idx_pm_rows_project ON permission_matrix_rows (project_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_project_created ON pipeline_runs (project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_created ON pipeline_runs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_extracted_entities_run ON extracted_entities (pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_conflict_records_run ON conflict_records (pipeline_run_id);
