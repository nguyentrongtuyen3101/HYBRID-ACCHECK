/** Thông báo in-app của 1 người dùng. */
export interface Notification {
  id: string;
  userId: string;
  eventId: string;
  type: string;
  /** Câu tiếng Anh dự phòng; client dịch theo `type` + `params`. */
  title: string;
  body: string;
  params: NotificationParams;
  link: string | null;
  createdAt: Date;
  readAt: Date | null;
}

/** Tham số để client dựng câu thông báo theo ngôn ngữ của người dùng (i18n). */
export type NotificationParams = Record<string, string | number>;

/** Nội dung thông báo trước khi lưu (chưa có id / thời điểm). */
export interface NotificationDraft {
  userId: string;
  type: string;
  title: string;
  body: string;
  params: NotificationParams;
  link: string | null;
}

/** Email cần gửi; template được render ở tầng infrastructure. */
export interface EmailMessage {
  to: string;
  template: EmailTemplate;
  locale: Locale;
  data: Record<string, unknown>;
}

export type EmailTemplate = "project_invitation" | "password_reset";

export const LOCALES = ["vi", "en", "ja"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "vi";

export function toLocale(value: unknown): Locale {
  return LOCALES.includes(value as Locale) ? (value as Locale) : DEFAULT_LOCALE;
}

/** Sự kiện chỉ đẩy realtime, không lưu (vd tiến độ job). */
export interface RealtimeSignal {
  userId: string;
  channel: string;
  payload: Record<string, unknown>;
}

/** Kế hoạch gửi cho 1 sự kiện: thông báo in-app, email và tín hiệu realtime. */
export interface DeliveryPlan {
  notifications: NotificationDraft[];
  emails: EmailMessage[];
  signals: RealtimeSignal[];
}

export const emptyPlan = (): DeliveryPlan => ({ notifications: [], emails: [], signals: [] });
