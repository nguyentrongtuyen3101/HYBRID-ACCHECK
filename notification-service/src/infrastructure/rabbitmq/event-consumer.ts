import amqp, { type Channel, type ChannelModel, type ConsumeMessage } from "amqplib";
import type { HandleEventUseCase } from "../../application/handle-event.js";
import type { Logger } from "../../application/ports.js";
import { InvalidEventError, parseEvent } from "../../domain/event.js";

// Phải khớp với api-service (internal/adapter/rabbitmq): cùng tên và cùng arguments.
export const EXCHANGE = "reqsentinel.events";
export const UNROUTED_EXCHANGE = "reqsentinel.events.unrouted";
export const UNROUTED_QUEUE = "reqsentinel.events.unrouted";

const QUEUE = "notification-service.events";
const DLX = "notification-service.dlx";
const DLQ = "notification-service.events.dlq";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** Nhận sự kiện từ RabbitMQ và chuyển cho use case HandleEvent. */
export class EventConsumer {
  private conn?: ChannelModel;
  private channel?: Channel;
  private stopped = false;
  connected = false;

  constructor(
    private readonly url: string,
    private readonly useCase: HandleEventUseCase,
    private readonly log: Logger,
    private readonly maxRetries: number,
  ) {}

  /** Kết nối và tự kết nối lại khi mất kết nối. */
  async start(): Promise<void> {
    let backoff = 1000;
    while (!this.stopped) {
      try {
        await this.connect();
        return;
      } catch (err) {
        this.log.warn("rabbitmq connect failed, retrying", { err: String(err), inMs: backoff });
        await sleep(backoff);
        backoff = Math.min(backoff * 2, 30_000);
      }
    }
  }

  private async connect(): Promise<void> {
    const conn = await amqp.connect(this.url);
    const ch = await conn.createChannel();
    this.conn = conn;
    this.channel = ch;

    await ch.assertExchange(UNROUTED_EXCHANGE, "fanout", { durable: true });
    await ch.assertQueue(UNROUTED_QUEUE, { durable: true });
    await ch.bindQueue(UNROUTED_QUEUE, UNROUTED_EXCHANGE, "");
    await ch.assertExchange(EXCHANGE, "topic", { durable: true, alternateExchange: UNROUTED_EXCHANGE });
    await ch.assertExchange(DLX, "fanout", { durable: true });
    await ch.assertQueue(DLQ, { durable: true });
    await ch.bindQueue(DLQ, DLX, "");
    await ch.assertQueue(QUEUE, { durable: true, deadLetterExchange: DLX });
    await ch.bindQueue(QUEUE, EXCHANGE, "#");
    await ch.prefetch(10);

    await ch.consume(QUEUE, (msg) => void this.onMessage(ch, msg));
    // Sự kiện phát ra trước khi queue của service này tồn tại được giữ ở đây
    await ch.consume(UNROUTED_QUEUE, (msg) => void this.onMessage(ch, msg));

    this.connected = true;
    this.log.info("rabbitmq consumer started", { queue: QUEUE });
    conn.on("close", () => {
      this.connected = false;
      if (!this.stopped) {
        this.log.warn("rabbitmq connection closed, reconnecting");
        void this.start();
      }
    });
    conn.on("error", (err) => this.log.error("rabbitmq error", { err: String(err) }));
  }

  private async onMessage(ch: Channel, msg: ConsumeMessage | null): Promise<void> {
    if (!msg) return;
    let eventId = msg.properties.messageId as string | undefined;
    try {
      const event = parseEvent(JSON.parse(msg.content.toString("utf8")));
      eventId = event.id;
      await this.useCase.execute(event);
      ch.ack(msg);
    } catch (err) {
      if (err instanceof InvalidEventError || err instanceof SyntaxError) {
        this.log.error("invalid event, moved to DLQ", { eventId, err: String(err) });
        ch.nack(msg, false, false);
        return;
      }
      const retries = Number(msg.properties.headers?.["x-retries"] ?? 0);
      if (retries >= this.maxRetries) {
        this.log.error("event failed too many times, moved to DLQ", { eventId, retries, err: String(err) });
        ch.nack(msg, false, false);
        return;
      }
      // Lỗi tạm thời (SMTP, DB...) → chờ rồi đưa lại vào queue với số lần thử tăng dần
      const delay = Math.min(1000 * 2 ** retries, 60_000);
      this.log.warn("event handling failed, retrying", { eventId, retries, delayMs: delay, err: String(err) });
      await sleep(delay);
      ch.sendToQueue(QUEUE, msg.content, {
        ...msg.properties,
        persistent: true,
        headers: { ...msg.properties.headers, "x-retries": retries + 1 },
      });
      ch.ack(msg);
    }
  }

  async stop(): Promise<void> {
    this.stopped = true;
    await this.channel?.close().catch(() => undefined);
    await this.conn?.close().catch(() => undefined);
  }
}
