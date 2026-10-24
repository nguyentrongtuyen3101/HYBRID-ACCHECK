// Port: interface mà use case cần; infrastructure hiện thực (Dependency Inversion).
import type { EmailMessage, Notification, NotificationDraft, RealtimeSignal } from "../domain/notification.js";

export interface NotificationRepository {
  /** Lưu thông báo; bỏ qua bản trùng (event_id + user_id), trả về các bản thực sự được tạo. */
  createMany(eventId: string, drafts: NotificationDraft[]): Promise<Notification[]>;
  listForUser(userId: string, opts: { limit: number; before?: Date; unreadOnly?: boolean }): Promise<Notification[]>;
  unreadCount(userId: string): Promise<number>;
  markRead(userId: string, id: string): Promise<boolean>;
  markAllRead(userId: string): Promise<number>;
}

/** Đánh dấu sự kiện đã xử lý để không xử lý lại khi broker giao lại (at-least-once). */
export interface ProcessedEventStore {
  isProcessed(eventId: string): Promise<boolean>;
  markProcessed(eventId: string, type: string): Promise<void>;
}

export interface Mailer {
  send(message: EmailMessage, eventId: string): Promise<void>;
}

export interface RealtimePublisher {
  notification(n: Notification, unreadCount: number): void;
  signal(s: RealtimeSignal): void;
}

export interface Logger {
  info(msg: string, meta?: Record<string, unknown>): void;
  warn(msg: string, meta?: Record<string, unknown>): void;
  error(msg: string, meta?: Record<string, unknown>): void;
}
