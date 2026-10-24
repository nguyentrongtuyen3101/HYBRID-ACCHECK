import { describe, expect, it } from "vitest";
import { HandleEventUseCase } from "../src/application/handle-event.js";
import type { Mailer, NotificationRepository, ProcessedEventStore, RealtimePublisher } from "../src/application/ports.js";
import { planDelivery } from "../src/domain/delivery-policy.js";
import { type DomainEvent, InvalidEventError, parseEvent } from "../src/domain/event.js";
import type { EmailMessage, Notification, NotificationDraft, RealtimeSignal } from "../src/domain/notification.js";
import { renderEmail } from "../src/infrastructure/mail/templates.js";

const event = (over: Partial<DomainEvent>): DomainEvent => ({
  id: "11111111-1111-4111-8111-111111111111",
  type: "project.member_invited",
  occurred_at: new Date().toISOString(),
  actor: { id: "owner", email: "owner@x.io", name: "Owner" },
  recipients: [],
  data: {},
  ...over,
});

describe("delivery policy", () => {
  it("sends an invitation email, plus an in-app notification when the invitee has an account", () => {
    const plan = planDelivery(
      event({
        recipients: ["invitee"],
        data: {
          project_name: "Shop",
          email: "bob@x.io",
          role: "editor",
          accept_url: "http://web/invitations/tok",
          inviter_name: "Owner",
        },
      }),
    );
    expect(plan.emails).toHaveLength(1);
    expect(plan.emails[0]).toMatchObject({ to: "bob@x.io", template: "project_invitation", locale: "vi" });
    // params dùng cho i18n phía client; không chứa link có token
    expect(plan.notifications[0]?.params).toEqual({ actor: "Owner", project: "Shop", role: "editor" });
    expect(plan.notifications).toEqual([
      expect.objectContaining({ userId: "invitee", title: "You were invited to Shop", link: "/invitations/tok" }),
    ]);
  });

  it("only pushes job progress in realtime, without storing it", () => {
    const plan = planDelivery(event({ type: "analysis.job_progress", recipients: ["u1"], data: { processed: 3 } }));
    expect(plan.notifications).toHaveLength(0);
    expect(plan.signals).toEqual([{ userId: "u1", channel: "job:progress", payload: { processed: 3 } }]);
  });

  it("links conflict notifications to the conflict board", () => {
    const plan = planDelivery(
      event({
        type: "conflict.assigned",
        recipients: ["dev"],
        data: { project_id: "p1", issue_id: "i1", conflict_type: "Effect Conflict", project_name: "Shop" },
      }),
    );
    expect(plan.notifications[0]?.link).toBe("/projects/p1?tab=conflicts&issue=i1");
  });

  it("ignores unknown event types and drops emails without a valid address", () => {
    expect(planDelivery(event({ type: "something.else", recipients: ["u"] }))).toEqual({
      notifications: [],
      emails: [],
      signals: [],
    });
    expect(planDelivery(event({ type: "auth.password_reset_requested", data: { email: "" } })).emails).toHaveLength(0);
  });

  it("rejects malformed messages", () => {
    expect(() => parseEvent({ type: "x" })).toThrow(InvalidEventError);
    expect(parseEvent({ id: "a", type: "b", recipients: ["u", 5] }).recipients).toEqual(["u"]);
  });
});

describe("email templates", () => {
  it("renders in the recipient's language", () => {
    const data = { projectName: "Shop", inviterName: "Alice", role: "editor", acceptUrl: "http://web/i/x" };
    expect(renderEmail("project_invitation", data, "vi").subject).toBe("Bạn được mời tham gia Shop trên ReqSentinel");
    expect(renderEmail("project_invitation", data, "ja").html).toContain("編集者");
    expect(renderEmail("password_reset", { resetUrl: "http://web/r" }, "en").subject).toBe("Reset your ReqSentinel password");
  });

  it("escapes user-provided values in HTML", () => {
    const email = renderEmail(
      "project_invitation",
      { projectName: "<script>alert(1)</script>", inviterName: "Eve", role: "viewer", acceptUrl: "http://web/invitations/x" },
      "en",
    );
    expect(email.html).not.toContain("<script>");
    expect(email.html).toContain("&lt;script&gt;");
    expect(email.text).toContain("http://web/invitations/x");
  });
});

// ---------- HandleEvent với fake (không cần DB / SMTP / socket) ----------

class FakeRepo implements NotificationRepository {
  rows: Notification[] = [];
  async createMany(eventId: string, drafts: NotificationDraft[]) {
    const created: Notification[] = [];
    for (const d of drafts) {
      if (this.rows.some((r) => r.eventId === eventId && r.userId === d.userId)) continue;
      const n = { ...d, id: `n${this.rows.length}`, eventId, createdAt: new Date(), readAt: null };
      this.rows.push(n);
      created.push(n);
    }
    return created;
  }
  async listForUser(userId: string) {
    return this.rows.filter((r) => r.userId === userId);
  }
  async unreadCount(userId: string) {
    return this.rows.filter((r) => r.userId === userId && !r.readAt).length;
  }
  async markRead() {
    return true;
  }
  async markAllRead() {
    return 0;
  }
}

function setup(failEmails = 0) {
  const repo = new FakeRepo();
  const processed = new Set<string>();
  const store: ProcessedEventStore = {
    isProcessed: async (id) => processed.has(id),
    markProcessed: async (id) => void processed.add(id),
  };
  const sent: EmailMessage[] = [];
  let failures = failEmails;
  const mailer: Mailer = {
    send: async (m) => {
      if (failures-- > 0) throw new Error("smtp down");
      sent.push(m);
    },
  };
  const pushed: { n?: Notification; s?: RealtimeSignal; unread?: number }[] = [];
  const realtime: RealtimePublisher = {
    notification: (n, unread) => pushed.push({ n, unread }),
    signal: (s) => pushed.push({ s }),
  };
  const log = { info: () => {}, warn: () => {}, error: () => {} };
  return { useCase: new HandleEventUseCase({ notifications: repo, processed: store, mailer, realtime, log }), repo, sent, pushed };
}

const invite = event({
  recipients: ["invitee"],
  data: { project_name: "Shop", email: "bob@x.io", role: "viewer", accept_url: "http://web/invitations/t" },
});

describe("HandleEventUseCase", () => {
  it("stores, pushes and emails once", async () => {
    const { useCase, repo, sent, pushed } = setup();
    await useCase.execute(invite);
    await useCase.execute(invite); // broker giao lại
    expect(repo.rows).toHaveLength(1);
    expect(sent).toHaveLength(1);
    expect(pushed).toEqual([expect.objectContaining({ unread: 1 })]);
  });

  it("retries a failed email without duplicating the in-app notification", async () => {
    const { useCase, repo, sent, pushed } = setup(1);
    await expect(useCase.execute(invite)).rejects.toThrow("smtp down");
    await useCase.execute(invite);
    expect(repo.rows).toHaveLength(1);
    expect(pushed).toHaveLength(1);
    expect(sent).toHaveLength(1);
  });
});
