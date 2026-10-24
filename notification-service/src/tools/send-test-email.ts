// Kiểm tra cấu hình SMTP: xác thực với server rồi gửi 1 email mẫu (template lời mời thật).
// Chạy: npm run mail:test -- you@example.com [vi|en|ja]
import nodemailer from "nodemailer";
import { loadConfig } from "../config.js";
import { toLocale } from "../domain/notification.js";
import { consoleLogger as log } from "../infrastructure/logger.js";
import { SmtpMailer } from "../infrastructure/mail/mailers.js";

async function main() {
  const to = process.argv[2];
  if (!to?.includes("@")) throw new Error("Usage: npm run mail:test -- you@example.com [vi|en|ja]");
  const { smtp } = loadConfig();
  if (!smtp.host) throw new Error("SMTP_HOST is empty in .env");

  const transport = nodemailer.createTransport({
    host: smtp.host,
    port: smtp.port,
    secure: smtp.secure,
    auth: smtp.user ? { user: smtp.user, pass: smtp.pass } : undefined,
  });
  await transport.verify();
  log.info("SMTP login ok", { host: smtp.host, port: smtp.port, user: smtp.user });

  await new SmtpMailer(smtp, log).send(
    {
      to,
      template: "project_invitation",
      locale: toLocale(process.argv[3]),
      data: {
        projectName: "SMTP test",
        inviterName: "ReqSentinel",
        role: "viewer",
        acceptUrl: "http://localhost:5173/",
        expiresAt: new Date(Date.now() + 7 * 86400_000).toISOString(),
      },
    },
    "smtp-test",
  );
}

main().catch((err) => {
  log.error("SMTP test failed", { err: String(err?.message ?? err) });
  process.exit(1);
});
