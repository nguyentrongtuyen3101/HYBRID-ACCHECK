import nodemailer, { type Transporter } from "nodemailer";
import type { Logger, Mailer } from "../../application/ports.js";
import type { EmailMessage } from "../../domain/notification.js";
import { renderEmail } from "./templates.js";

export interface SmtpConfig {
  host: string;
  port: number;
  secure: boolean;
  user: string;
  pass: string;
  from: string;
}

export class SmtpMailer implements Mailer {
  private readonly transport: Transporter;

  constructor(
    private readonly cfg: SmtpConfig,
    private readonly log: Logger,
  ) {
    this.transport = nodemailer.createTransport({
      host: cfg.host,
      port: cfg.port,
      secure: cfg.secure,
      auth: cfg.user ? { user: cfg.user, pass: cfg.pass } : undefined,
    });
  }

  async send(message: EmailMessage, eventId: string): Promise<void> {
    const email = renderEmail(message.template, message.data, message.locale);
    const info = await this.transport.sendMail({
      from: this.cfg.from,
      to: message.to,
      subject: email.subject,
      html: email.html,
      text: email.text,
      headers: { "X-ReqSentinel-Event": eventId },
    });
    this.log.info("email sent", { eventId, template: message.template, to: message.to, messageId: info.messageId });
  }
}

/** Dùng khi chưa cấu hình SMTP (dev): chỉ ghi log nội dung email. */
export class LogMailer implements Mailer {
  constructor(private readonly log: Logger) {}

  async send(message: EmailMessage, eventId: string): Promise<void> {
    const email = renderEmail(message.template, message.data, message.locale);
    this.log.warn("SMTP not configured, email not sent", {
      eventId,
      to: message.to,
      subject: email.subject,
      text: email.text,
    });
  }
}
