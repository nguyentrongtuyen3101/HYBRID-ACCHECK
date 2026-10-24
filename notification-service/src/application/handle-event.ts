import type { DomainEvent } from "../domain/event.js";
import { type DeliveryRule, deliveryRules, planDelivery } from "../domain/delivery-policy.js";
import type { Logger, Mailer, NotificationRepository, ProcessedEventStore, RealtimePublisher } from "./ports.js";

export interface HandleEventDeps {
  notifications: NotificationRepository;
  processed: ProcessedEventStore;
  mailer: Mailer;
  realtime: RealtimePublisher;
  log: Logger;
  rules?: Record<string, DeliveryRule>;
}

/**
 * Use case: nhận 1 sự kiện nghiệp vụ → lưu thông báo, đẩy realtime, gửi email.
 * Idempotent: sự kiện đã xử lý thì bỏ qua; thông báo trùng không bị tạo lại.
 */
export class HandleEventUseCase {
  constructor(private readonly deps: HandleEventDeps) {}

  async execute(event: DomainEvent): Promise<void> {
    const { notifications, processed, mailer, realtime, log } = this.deps;
    if (await processed.isProcessed(event.id)) {
      log.info("event already processed, skipping", { eventId: event.id, type: event.type });
      return;
    }
    const plan = planDelivery(event, this.deps.rules ?? deliveryRules);

    const created = await notifications.createMany(event.id, plan.notifications);
    for (const n of created) {
      realtime.notification(n, await notifications.unreadCount(n.userId));
    }
    for (const signal of plan.signals) {
      realtime.signal(signal);
    }
    // Email lỗi → ném lỗi để broker giao lại; thông báo đã lưu sẽ không bị tạo trùng
    for (const email of plan.emails) {
      await mailer.send(email, event.id);
    }

    await processed.markProcessed(event.id, event.type);
    if (created.length || plan.emails.length) {
      log.info("event delivered", {
        eventId: event.id,
        type: event.type,
        notifications: created.length,
        emails: plan.emails.length,
      });
    }
  }
}
