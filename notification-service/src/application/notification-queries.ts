import type { Notification } from "../domain/notification.js";
import type { NotificationRepository } from "./ports.js";

export class NotFoundError extends Error {}

/** Use case đọc / đánh dấu đã đọc thông báo của người dùng hiện tại. */
export class NotificationInbox {
  constructor(private readonly repo: NotificationRepository) {}

  async list(userId: string, opts: { limit?: number; before?: string; unreadOnly?: boolean }) {
    const limit = Math.min(Math.max(opts.limit ?? 20, 1), 100);
    const before = opts.before ? new Date(opts.before) : undefined;
    const items: Notification[] = await this.repo.listForUser(userId, {
      limit,
      before: before && !Number.isNaN(before.getTime()) ? before : undefined,
      unreadOnly: opts.unreadOnly,
    });
    return { items, unreadCount: await this.repo.unreadCount(userId) };
  }

  unreadCount(userId: string): Promise<number> {
    return this.repo.unreadCount(userId);
  }

  async markRead(userId: string, id: string): Promise<number> {
    if (!(await this.repo.markRead(userId, id))) throw new NotFoundError("Notification not found");
    return this.repo.unreadCount(userId);
  }

  async markAllRead(userId: string): Promise<number> {
    await this.repo.markAllRead(userId);
    return 0;
  }
}
