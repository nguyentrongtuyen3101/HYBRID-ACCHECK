import type { EmailTemplate, Locale } from "../../domain/notification.js";

export interface RenderedEmail {
  subject: string;
  html: string;
  text: string;
}

const escape = (v: unknown): string =>
  String(v ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");

const TIME_ZONES: Record<Locale, string> = { vi: "Asia/Ho_Chi_Minh", en: "UTC", ja: "Asia/Tokyo" };

function formatDate(iso: unknown, locale: Locale): string {
  const d = new Date(String(iso));
  if (Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat(locale, { dateStyle: "long", timeStyle: "short", timeZone: TIME_ZONES[locale] }).format(d);
}

interface Strings {
  roles: Record<string, string>;
  orOpen: string;
  invitation: {
    subject: (p: string) => string;
    title: (p: string) => string;
    intro: (inviter: string, project: string, role: string) => string;
    noAccount: string;
    button: string;
    expires: (date: string) => string;
  };
  reset: {
    subject: string;
    title: string;
    greeting: (name: string) => string;
    body: string;
    button: string;
    ignore: string;
  };
}

const STRINGS: Record<Locale, Strings> = {
  vi: {
    roles: { owner: "Chủ sở hữu", editor: "Biên tập viên", viewer: "Người xem" },
    orOpen: "Hoặc mở liên kết này:",
    invitation: {
      subject: (p) => `Bạn được mời tham gia ${p} trên ReqSentinel`,
      title: (p) => `Tham gia ${p}`,
      intro: (i, p, r) => `<strong>${i}</strong> đã mời bạn vào dự án <strong>${p}</strong> với vai trò <strong>${r}</strong>.`,
      noAccount: "Nếu chưa có tài khoản, hãy đăng ký bằng chính địa chỉ email này rồi chấp nhận lời mời.",
      button: "Chấp nhận lời mời",
      expires: (d) => `Lời mời hết hạn lúc ${d}.`,
    },
    reset: {
      subject: "Đặt lại mật khẩu ReqSentinel",
      title: "Đặt lại mật khẩu",
      greeting: (n) => `Xin chào ${n || "bạn"},`,
      body: "Chúng tôi nhận được yêu cầu đặt lại mật khẩu của bạn. Liên kết dưới đây có hiệu lực trong 30 phút và chỉ dùng được một lần.",
      button: "Đặt lại mật khẩu",
      ignore: "Nếu bạn không yêu cầu, hãy bỏ qua email này — mật khẩu của bạn không thay đổi.",
    },
  },
  en: {
    roles: { owner: "Owner", editor: "Editor", viewer: "Viewer" },
    orOpen: "Or open this link:",
    invitation: {
      subject: (p) => `You're invited to ${p} on ReqSentinel`,
      title: (p) => `Join ${p}`,
      intro: (i, p, r) => `<strong>${i}</strong> invited you to the project <strong>${p}</strong> as <strong>${r}</strong>.`,
      noAccount: "If you don't have an account yet, create one with this email address, then accept the invitation.",
      button: "Accept invitation",
      expires: (d) => `This invitation expires on ${d}.`,
    },
    reset: {
      subject: "Reset your ReqSentinel password",
      title: "Reset your password",
      greeting: (n) => `Hi ${n || "there"},`,
      body: "We received a request to reset your password. The link below is valid for 30 minutes and can be used once.",
      button: "Reset password",
      ignore: "If you did not request this, you can ignore this email — your password stays the same.",
    },
  },
  ja: {
    roles: { owner: "オーナー", editor: "編集者", viewer: "閲覧者" },
    orOpen: "または次のリンクを開いてください:",
    invitation: {
      subject: (p) => `ReqSentinel の「${p}」に招待されました`,
      title: (p) => `「${p}」に参加`,
      intro: (i, p, r) => `<strong>${i}</strong> さんがあなたをプロジェクト <strong>${p}</strong> に <strong>${r}</strong> として招待しました。`,
      noAccount: "アカウントをお持ちでない場合は、このメールアドレスで登録してから招待を承認してください。",
      button: "招待を承認する",
      expires: (d) => `この招待の有効期限は ${d} です。`,
    },
    reset: {
      subject: "ReqSentinel のパスワード再設定",
      title: "パスワードの再設定",
      greeting: (n) => `${n || "ユーザー"} 様`,
      body: "パスワード再設定のリクエストを受け付けました。下のリンクは 30 分間有効で、1 回のみ使用できます。",
      button: "パスワードを再設定",
      ignore: "お心当たりがない場合は、このメールを無視してください。パスワードは変更されません。",
    },
  },
};

/** Khung email cùng phong cách UI (viền đen, nút đen). */
function layout(locale: Locale, title: string, paragraphs: string[], action: { label: string; url: string }, footer?: string) {
  return `<!doctype html><html lang="${locale}"><body style="margin:0;padding:24px;background:#fafafa;font-family:Inter,Arial,sans-serif;color:#111">
  <div style="max-width:520px;margin:0 auto;background:#fff;border:1.5px solid #1a1a1a;border-radius:4px;box-shadow:2px 2px 0 #1a1a1a;padding:24px">
    <div style="font-weight:700;font-size:15px;margin-bottom:16px">ReqSentinel</div>
    <h1 style="font-size:18px;margin:0 0 12px">${escape(title)}</h1>
    ${paragraphs.map((p) => `<p style="font-size:14px;line-height:1.5;margin:0 0 10px">${p}</p>`).join("")}
    <p style="margin:24px 0"><a href="${escape(action.url)}" style="background:#1a1a1a;color:#fff;padding:10px 20px;border-radius:999px;text-decoration:none;font-weight:600;display:inline-block">${escape(action.label)}</a></p>
    <p style="font-size:12px;color:#555">${escape(STRINGS[locale].orOpen)}<br><a href="${escape(action.url)}" style="color:#1a1a1a;word-break:break-all">${escape(action.url)}</a></p>
    ${footer ? `<p style="font-size:12px;color:#555;margin-top:20px">${escape(footer)}</p>` : ""}
  </div></body></html>`;
}

const renderers: Record<EmailTemplate, (d: Record<string, unknown>, locale: Locale) => RenderedEmail> = {
  project_invitation: (d, locale) => {
    const t = STRINGS[locale];
    const role = t.roles[String(d.role)] ?? String(d.role);
    const expires = formatDate(d.expiresAt, locale);
    const footer = expires ? t.invitation.expires(expires) : undefined;
    return {
      subject: t.invitation.subject(String(d.projectName)),
      html: layout(
        locale,
        t.invitation.title(String(d.projectName)),
        [t.invitation.intro(escape(d.inviterName), escape(d.projectName), escape(role)), escape(t.invitation.noAccount)],
        { label: t.invitation.button, url: String(d.acceptUrl) },
        footer,
      ),
      text:
        `${t.invitation.intro(String(d.inviterName), String(d.projectName), role).replace(/<\/?strong>/g, "")}\n\n` +
        `${t.invitation.button}: ${d.acceptUrl}\n${footer ?? ""}\n`,
    };
  },

  password_reset: (d, locale) => {
    const t = STRINGS[locale].reset;
    return {
      subject: t.subject,
      html: layout(locale, t.title, [escape(t.greeting(String(d.name || ""))), escape(t.body)],
        { label: t.button, url: String(d.resetUrl) }, t.ignore),
      text: `${t.greeting(String(d.name || ""))}\n\n${t.body}\n\n${t.button}: ${d.resetUrl}\n\n${t.ignore}\n`,
    };
  },
};

export function renderEmail(template: EmailTemplate, data: Record<string, unknown>, locale: Locale): RenderedEmail {
  return renderers[template](data, locale);
}
