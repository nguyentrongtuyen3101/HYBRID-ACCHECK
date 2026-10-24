import pg from "pg";
import type { Logger } from "../../application/ports.js";

// Service này chỉ dùng schema riêng "notification" (không đọc/ghi bảng của api-service).
const MIGRATIONS: { version: string; sql: string }[] = [
  {
    version: "001_init",
    sql: `
      CREATE SCHEMA IF NOT EXISTS notification;

      CREATE TABLE IF NOT EXISTS notification.notifications (
        id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id    uuid NOT NULL,
        event_id   uuid NOT NULL,
        type       varchar(64) NOT NULL,
        title      text NOT NULL,
        body       text NOT NULL DEFAULT '',
        link       text,
        read_at    timestamptz,
        created_at timestamptz NOT NULL DEFAULT now(),
        UNIQUE (event_id, user_id)
      );
      CREATE INDEX IF NOT EXISTS idx_notifications_user ON notification.notifications (user_id, created_at DESC);
      CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notification.notifications (user_id) WHERE read_at IS NULL;

      CREATE TABLE IF NOT EXISTS notification.processed_events (
        event_id     uuid PRIMARY KEY,
        type         varchar(64) NOT NULL,
        processed_at timestamptz NOT NULL DEFAULT now()
      );
    `,
  },
  {
    // Tham số để client hiển thị thông báo theo ngôn ngữ người dùng (i18n)
    version: "002_params",
    sql: `ALTER TABLE notification.notifications ADD COLUMN IF NOT EXISTS params jsonb NOT NULL DEFAULT '{}'`,
  },
];

export function createPool(url: string): pg.Pool {
  return new pg.Pool({ connectionString: url, max: 10 });
}

export async function migrate(pool: pg.Pool, log: Logger): Promise<void> {
  const client = await pool.connect();
  try {
    await client.query("SELECT pg_advisory_lock(72620012)");
    await client.query(`CREATE SCHEMA IF NOT EXISTS notification;
      CREATE TABLE IF NOT EXISTS notification.schema_migrations (
        version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`);
    for (const m of MIGRATIONS) {
      const { rowCount } = await client.query("SELECT 1 FROM notification.schema_migrations WHERE version = $1", [
        m.version,
      ]);
      if (rowCount) continue;
      await client.query("BEGIN");
      try {
        await client.query(m.sql);
        await client.query("INSERT INTO notification.schema_migrations (version) VALUES ($1)", [m.version]);
        await client.query("COMMIT");
      } catch (err) {
        await client.query("ROLLBACK");
        throw err;
      }
      log.info("applied migration", { version: m.version });
    }
  } finally {
    await client.query("SELECT pg_advisory_unlock(72620012)").catch(() => undefined);
    client.release();
  }
}
