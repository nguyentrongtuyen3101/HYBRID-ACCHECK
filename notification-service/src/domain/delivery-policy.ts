import { type DomainEvent, EventTypes, num, str } from "./event.js";
import { type DeliveryPlan, type NotificationDraft, type NotificationParams, emptyPlan, toLocale } from "./notification.js";

/**
 * Luật nghiệp vụ: mỗi loại sự kiện → ai nhận gì, qua kênh nào.
 * Thêm loại sự kiện mới = thêm 1 rule vào bảng (không sửa chỗ xử lý chung).
 *
 * Thông báo in-app lưu `type` + `params` để client hiển thị theo ngôn ngữ người dùng;
 * title/body tiếng Anh chỉ là bản dự phòng. Không đưa link chứa token vào params.
 */
export type DeliveryRule = (event: DomainEvent) => DeliveryPlan;

function actorName(e: DomainEvent): string {
  return e.actor?.name || e.actor?.email || "Someone";
}

function toEach(
  e: DomainEvent,
  params: NotificationParams,
  draft: { title: string; body: string; link: string | null },
): NotificationDraft[] {
  return e.recipients.map((userId) => ({ userId, type: e.type, params, ...draft }));
}

function projectLink(e: DomainEvent, query = ""): string {
  return `/projects/${str(e.data, "project_id")}${query}`;
}

function conflictLink(e: DomainEvent): string {
  return projectLink(e, `?tab=conflicts&issue=${str(e.data, "issue_id")}`);
}

/** Lấy phần path từ URL tuyệt đối để dùng làm link trong app. */
function pathOf(url: string): string | null {
  try {
    const u = new URL(url);
    return u.pathname + u.search;
  } catch {
    return null;
  }
}

/** Tham số chung của thông báo về conflict. */
function conflictParams(e: DomainEvent) {
  return {
    actor: actorName(e),
    project: str(e.data, "project_name"),
    conflict_type: str(e.data, "conflict_type"),
    explanation: str(e.data, "explanation"),
  };
}

export const deliveryRules: Record<string, DeliveryRule> = {
  [EventTypes.MemberInvited]: (e) => {
    const params = { actor: actorName(e), project: str(e.data, "project_name"), role: str(e.data, "role") };
    return {
      notifications: toEach(e, params, {
        title: `You were invited to ${params.project}`,
        body: `${params.actor} invited you to join as ${params.role}.`,
        link: pathOf(str(e.data, "accept_url")),
      }),
      emails: [
        {
          to: str(e.data, "email"),
          template: "project_invitation",
          locale: toLocale(e.data.locale),
          data: {
            projectName: params.project,
            inviterName: str(e.data, "inviter_name") || params.actor,
            role: params.role,
            acceptUrl: str(e.data, "accept_url"),
            expiresAt: str(e.data, "expires_at"),
          },
        },
      ],
      signals: [],
    };
  },

  [EventTypes.MemberJoined]: (e) => {
    const params = {
      member: str(e.data, "member_name") || actorName(e),
      project: str(e.data, "project_name"),
      role: str(e.data, "role"),
    };
    return {
      ...emptyPlan(),
      notifications: toEach(e, params, {
        title: `${params.member} joined ${params.project}`,
        body: `They accepted your invitation as ${params.role}.`,
        link: projectLink(e, "?tab=members"),
      }),
    };
  },

  [EventTypes.MemberRoleChanged]: (e) => {
    const params = { actor: actorName(e), project: str(e.data, "project_name"), role: str(e.data, "role") };
    return {
      ...emptyPlan(),
      notifications: toEach(e, params, {
        title: `Your role in ${params.project} changed`,
        body: `${params.actor} changed your role to ${params.role}.`,
        link: projectLink(e),
      }),
    };
  },

  [EventTypes.MemberRemoved]: (e) => {
    const params = { actor: actorName(e), project: str(e.data, "project_name") };
    return {
      ...emptyPlan(),
      notifications: toEach(e, params, {
        title: `You were removed from ${params.project}`,
        body: `${params.actor} removed you from the project.`,
        link: null,
      }),
    };
  },

  [EventTypes.ConflictAssigned]: (e) => {
    const params = conflictParams(e);
    return {
      ...emptyPlan(),
      notifications: toEach(e, params, {
        title: `${params.actor} assigned you a ${params.conflict_type}`,
        body: `${params.project}: ${params.explanation}`,
        link: conflictLink(e),
      }),
    };
  },

  [EventTypes.ConflictStatusChanged]: (e) => {
    const params = { ...conflictParams(e), from: str(e.data, "from"), to: str(e.data, "to") };
    return {
      ...emptyPlan(),
      notifications: toEach(e, params, {
        title: `${params.actor} marked a ${params.conflict_type} as ${params.to}`,
        body: `${params.project}: ${params.explanation}`,
        link: conflictLink(e),
      }),
    };
  },

  [EventTypes.ConflictCommented]: (e) => {
    const params = { ...conflictParams(e), comment: str(e.data, "comment") };
    return {
      ...emptyPlan(),
      notifications: toEach(e, params, {
        title: `${params.actor} commented on a ${params.conflict_type}`,
        body: params.comment,
        link: conflictLink(e),
      }),
    };
  },

  // Tiến độ job: chỉ đẩy realtime, không lưu thành thông báo
  [EventTypes.JobProgress]: (e) => ({
    ...emptyPlan(),
    signals: e.recipients.map((userId) => ({ userId, channel: "job:progress", payload: e.data })),
  }),

  [EventTypes.JobFinished]: (e) => {
    const failed = str(e.data, "status") === "failed";
    const params = {
      project: str(e.data, "project_name"),
      status: str(e.data, "status"),
      succeeded: num(e.data, "processed") - num(e.data, "failed"),
      total: num(e.data, "total"),
      open_issues: num(e.data, "open_issues"),
      error: str(e.data, "error"),
    };
    return {
      notifications: toEach(e, params, {
        title: failed ? `Analysis of ${params.project} failed` : `Analysis of ${params.project} finished`,
        body: failed
          ? params.error
          : `${params.succeeded}/${params.total} stories analyzed · ${params.open_issues} open conflicts`,
        link: projectLink(e, "?tab=conflicts"),
      }),
      emails: [],
      signals: e.recipients.map((userId) => ({ userId, channel: "job:progress", payload: e.data })),
    };
  },

  [EventTypes.PasswordResetRequested]: (e) => ({
    ...emptyPlan(),
    emails: [
      {
        to: str(e.data, "email"),
        template: "password_reset",
        locale: toLocale(e.data.locale),
        data: { name: str(e.data, "name"), resetUrl: str(e.data, "reset_url"), expiresAt: str(e.data, "expires_at") },
      },
    ],
  }),
};

/** Sự kiện không có rule → không gửi gì (service khác có thể quan tâm). */
export function planDelivery(event: DomainEvent, rules: Record<string, DeliveryRule> = deliveryRules): DeliveryPlan {
  const rule = rules[event.type];
  if (!rule) return emptyPlan();
  const plan = rule(event);
  return { ...plan, emails: plan.emails.filter((m) => m.to.includes("@")) };
}
