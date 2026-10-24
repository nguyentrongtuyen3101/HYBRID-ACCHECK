// Composition root: nối infrastructure vào use case.
import { createServer } from "node:http";
import { HandleEventUseCase } from "./application/handle-event.js";
import { NotificationInbox } from "./application/notification-queries.js";
import { loadConfig } from "./config.js";
import { JwtVerifier } from "./infrastructure/auth/jwt-verifier.js";
import { createHttpApp } from "./infrastructure/http/app.js";
import { consoleLogger as log } from "./infrastructure/logger.js";
import { LogMailer, SmtpMailer } from "./infrastructure/mail/mailers.js";
import { createPool, migrate } from "./infrastructure/postgres/database.js";
import { PgNotificationRepository, PgProcessedEventStore } from "./infrastructure/postgres/repositories.js";
import { EventConsumer } from "./infrastructure/rabbitmq/event-consumer.js";
import { SocketGateway } from "./infrastructure/realtime/socket-gateway.js";

async function main() {
  const cfg = loadConfig();
  const pool = createPool(cfg.databaseUrl);
  await migrate(pool, log);

  const verifier = new JwtVerifier(cfg.jwtSecret);
  const notifications = new PgNotificationRepository(pool);
  const inbox = new NotificationInbox(notifications);
  const mailer = cfg.smtp.host ? new SmtpMailer(cfg.smtp, log) : new LogMailer(log);
  if (!cfg.smtp.host) log.warn("SMTP_HOST not set: emails are only logged");

  let consumer: EventConsumer | undefined;
  const health = async () => {
    let database = "ok";
    await pool.query("SELECT 1").catch(() => (database = "unavailable"));
    const rabbitmq = consumer?.connected ? "ok" : "unavailable";
    const ok = database === "ok" && rabbitmq === "ok";
    return { status: ok ? "ok" : "degraded", service: "ReqSentinel Notification Service", database, rabbitmq };
  };

  const httpServer = createServer(createHttpApp(inbox, verifier, health, log));
  const realtime = new SocketGateway(httpServer, verifier, cfg.corsOrigins, log);
  const handleEvent = new HandleEventUseCase({
    notifications,
    processed: new PgProcessedEventStore(pool),
    mailer,
    realtime,
    log,
  });
  consumer = new EventConsumer(cfg.rabbitmqUrl, handleEvent, log, cfg.maxRetries);

  httpServer.listen(cfg.port, () => log.info("notification service listening", { port: cfg.port }));
  void consumer.start();

  const shutdown = async (signal: string) => {
    log.info("shutting down", { signal });
    await consumer?.stop();
    await realtime.close();
    httpServer.close();
    await pool.end();
    process.exit(0);
  };
  process.on("SIGINT", () => void shutdown("SIGINT"));
  process.on("SIGTERM", () => void shutdown("SIGTERM"));
}

main().catch((err) => {
  log.error("fatal", { err: String(err?.stack ?? err) });
  process.exit(1);
});
