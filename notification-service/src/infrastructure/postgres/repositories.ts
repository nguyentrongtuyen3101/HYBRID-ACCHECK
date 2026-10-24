import type pg from "pg";
import type { NotificationRepository, ProcessedEventStore } from "../../application/ports.js";
import type { Notification, NotificationDraft, NotificationParams } from "../../domain/notification.js";

interface NotificationRow {
  id: string;
  user_id: string;
  event_id: string;
  type: string;
  title: string;
  body: string;
  link: string | null;
  params: NotificationParams | null;
  read_at: Date | null;
  created_at: Date;
}

const toNotification = (r: NotificationRow): Notification => ({
  id: r.id,
  userId: r.user_id,
  eventId: r.event_id,
  type: r.type,
  title: r.title,
  body: r.body,
  link: r.link,
  params: r.params ?? {},
  readAt: r.read_at,
  createdAt: r.created_at,
});

export class PgNotificationRepository implements NotificationRepository {
  constructor(private readonly pool: pg.Pool) {}

  async createMany(eventId: string, drafts: NotificationDraft[]): Promise<Notification[]> {
    const created: Notification[] = [];
    for (const d of drafts) {
      const { rows } = await this.pool.query<NotificationRow>(
        `INSERT INTO notification.notifications (user_id, event_id, type, title, body, link, params)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         ON CONFLICT (event_id, user_id) DO NOTHING
         RETURNING *`,
        [d.userId, eventId, d.type, d.title, d.body, d.link, JSON.stringify(d.params)],
      );
      if (rows[0]) created.push(toNotification(rows[0]));
    }
    return created;
  }

  async listForUser(userId: string, opts: { limit: number; before?: Date; unreadOnly?: boolean }) {
    const { rows } = await this.pool.query<NotificationRow>(
      `SELECT * FROM notification.notifications
       WHERE user_id = $1
         AND ($2::timestamptz IS NULL OR created_at < $2)
         AND (NOT $3 OR read_at IS NULL)
       ORDER BY created_at DESC LIMIT $4`,
      [userId, opts.before ?? null, opts.unreadOnly ?? false, opts.limit],
    );
    return rows.map(toNotification);
  }

  async unreadCount(userId: string): Promise<number> {
    const { rows } = await this.pool.query<{ n: string }>(
      "SELECT count(*) AS n FROM notification.notifications WHERE user_id = $1 AND read_at IS NULL",
      [userId],
    );
    return Number(rows[0]?.n ?? 0);
  }

  async markRead(userId: string, id: string): Promise<boolean> {
    const { rowCount } = await this.pool.query(
      `UPDATE notification.notifications SET read_at = COALESCE(read_at, now())
       WHERE id = $1 AND user_id = $2`,
      [id, userId],
    );
    return (rowCount ?? 0) > 0;
  }

  async markAllRead(userId: string): Promise<number> {
    const { rowCount } = await this.pool.query(
      "UPDATE notification.notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL",
      [userId],
    );
    return rowCount ?? 0;
  }
}

export class PgProcessedEventStore implements ProcessedEventStore {
  constructor(private readonly pool: pg.Pool) {}

  async isProcessed(eventId: string): Promise<boolean> {
    const { rowCount } = await this.pool.query("SELECT 1 FROM notification.processed_events WHERE event_id = $1", [
      eventId,
    ]);
    return (rowCount ?? 0) > 0;
  }

  async markProcessed(eventId: string, type: string): Promise<void> {
    await this.pool.query(
      "INSERT INTO notification.processed_events (event_id, type) VALUES ($1, $2) ON CONFLICT DO NOTHING",
      [eventId, type],
    );
  }
}
