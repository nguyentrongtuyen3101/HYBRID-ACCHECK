// Hợp đồng sự kiện do api-service (Go) phát qua RabbitMQ (xem postgres.EventMessage bên Go).

export interface EventActor {
  id: string;
  email: string;
  name: string;
}

export interface DomainEvent {
  id: string;
  type: string;
  occurred_at: string;
  actor: EventActor | null;
  recipients: string[]; // user id nhận thông báo in-app
  data: Record<string, unknown>;
}

export const EventTypes = {
  MemberInvited: "project.member_invited",
  MemberJoined: "project.member_joined",
  MemberRoleChanged: "project.member_role_changed",
  MemberRemoved: "project.member_removed",
  ConflictAssigned: "conflict.assigned",
  ConflictStatusChanged: "conflict.status_changed",
  ConflictCommented: "conflict.commented",
  JobProgress: "analysis.job_progress",
  JobFinished: "analysis.job_finished",
  PasswordResetRequested: "auth.password_reset_requested",
} as const;

export class InvalidEventError extends Error {}

/** Kiểm tra hình dạng tối thiểu của message nhận từ broker. */
export function parseEvent(raw: unknown): DomainEvent {
  if (typeof raw !== "object" || raw === null) throw new InvalidEventError("event must be an object");
  const e = raw as Record<string, unknown>;
  if (typeof e.id !== "string" || typeof e.type !== "string") throw new InvalidEventError("event id/type missing");
  const recipients = Array.isArray(e.recipients) ? e.recipients.filter((r): r is string => typeof r === "string") : [];
  const data = typeof e.data === "object" && e.data !== null ? (e.data as Record<string, unknown>) : {};
  const actor = typeof e.actor === "object" && e.actor !== null ? (e.actor as EventActor) : null;
  return {
    id: e.id,
    type: e.type,
    occurred_at: typeof e.occurred_at === "string" ? e.occurred_at : new Date().toISOString(),
    actor,
    recipients,
    data,
  };
}

/** Đọc field kiểu string trong data (thiếu → chuỗi rỗng). */
export function str(data: Record<string, unknown>, key: string): string {
  const v = data[key];
  return typeof v === "string" ? v : v === undefined || v === null ? "" : String(v);
}

export function num(data: Record<string, unknown>, key: string): number {
  const v = Number(data[key]);
  return Number.isFinite(v) ? v : 0;
}
