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

const s = (v: unknown): string => String(v ?? "");

const TIME_ZONES: Record<Locale, string> = { vi: "Asia/Ho_Chi_Minh", en: "UTC", ja: "Asia/Tokyo" };

function formatDate(iso: unknown, locale: Locale): string {
  const d = new Date(s(iso));
  if (Number.isNaN(d.getTime())) return "";
  if (locale === "vi") {
    // Intl "vi" cho dạng "lúc 21:30 14 tháng 11, 2026" khó đọc → "21:30, 14/11/2026"
    const p = Object.fromEntries(
      new Intl.DateTimeFormat("en-GB", {
        day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit", hour12: false,
        timeZone: TIME_ZONES.vi,
      }).formatToParts(d).map((x) => [x.type, x.value]),
    );
    return `${p.hour}:${p.minute}, ${p.day}/${p.month}/${p.year}`;
  }
  return new Intl.DateTimeFormat(locale, { dateStyle: "long", timeStyle: "short", timeZone: TIME_ZONES[locale] }).format(d);
}

// ---------------------------------------------------------------- Chuỗi theo ngôn ngữ

interface Common {
  tagline: string;
  greeting: (name: string) => string;
  orOpen: string;
  footer: string;
  labels: { project: string; role: string; actor: string; inviter: string; time: string; conflictType: string; email: string; systemRole: string };
  roles: Record<string, string>;
  roleDescriptions: Record<string, string>;
  systemRoles: Record<string, string>;
  conflictTypes: Record<string, string>;
  eyebrows: { invitation: string; permission: string; task: string; security: string; account: string };
}

const COMMON: Record<Locale, Common> = {
  vi: {
    tagline: "Access-control checker",
    greeting: (n) => `Xin chào ${n || "bạn"},`,
    orOpen: "Nếu nút không bấm được, mở liên kết này:",
    footer: "Email tự động từ ReqSentinel. Vui lòng không trả lời email này.",
    labels: {
      project: "Dự án", role: "Vai trò", actor: "Người thực hiện", inviter: "Người mời", time: "Thời gian",
      conflictType: "Loại xung đột", email: "Email đăng nhập", systemRole: "Vai trò hệ thống",
    },
    roles: { owner: "Chủ sở hữu", editor: "Biên tập viên", viewer: "Người xem" },
    roleDescriptions: {
      owner: "toàn quyền: thành viên, cài đặt, xóa dự án",
      editor: "sửa yêu cầu, chạy phân tích, xử lý xung đột",
      viewer: "xem kết quả và bình luận",
    },
    systemRoles: { admin: "Quản trị viên", user: "Người dùng" },
    conflictTypes: {
      "Missing Permission": "Thiếu quyền", "Action Conflict": "Xung đột hành động", "Effect Conflict": "Xung đột hiệu lực",
      "Scope Conflict": "Xung đột phạm vi", "Condition Conflict": "Xung đột điều kiện",
    },
    eyebrows: { invitation: "Lời mời dự án", permission: "Thay đổi quyền", task: "Nhiệm vụ mới", security: "Bảo mật tài khoản", account: "Tài khoản" },
  },
  en: {
    tagline: "Access-control checker",
    greeting: (n) => `Hi ${n || "there"},`,
    orOpen: "If the button doesn't work, open this link:",
    footer: "Automated email from ReqSentinel. Please do not reply.",
    labels: {
      project: "Project", role: "Role", actor: "Changed by", inviter: "Invited by", time: "Time",
      conflictType: "Conflict type", email: "Sign-in email", systemRole: "System role",
    },
    roles: { owner: "Owner", editor: "Editor", viewer: "Viewer" },
    roleDescriptions: {
      owner: "full control: members, settings, delete project",
      editor: "edit requirements, run analysis, triage conflicts",
      viewer: "read results and comment",
    },
    systemRoles: { admin: "Admin", user: "User" },
    conflictTypes: {},
    eyebrows: { invitation: "Project invitation", permission: "Permission change", task: "New assignment", security: "Account security", account: "Account" },
  },
  ja: {
    tagline: "Access-control checker",
    greeting: (n) => `${n || "ユーザー"} 様`,
    orOpen: "ボタンが開けない場合は次のリンクを開いてください:",
    footer: "このメールは ReqSentinel から自動送信されています。返信しないでください。",
    labels: {
      project: "プロジェクト", role: "ロール", actor: "実行者", inviter: "招待者", time: "日時",
      conflictType: "コンフリクトの種類", email: "ログインメール", systemRole: "システムロール",
    },
    roles: { owner: "オーナー", editor: "編集者", viewer: "閲覧者" },
    roleDescriptions: {
      owner: "すべての権限:メンバー、設定、プロジェクト削除",
      editor: "要件の編集、分析の実行、コンフリクトの対応",
      viewer: "結果の閲覧とコメント",
    },
    systemRoles: { admin: "管理者", user: "ユーザー" },
    conflictTypes: {
      "Missing Permission": "権限不足", "Action Conflict": "アクションの競合", "Effect Conflict": "効果の競合",
      "Scope Conflict": "スコープの競合", "Condition Conflict": "条件の競合",
    },
    eyebrows: { invitation: "プロジェクトへの招待", permission: "権限の変更", task: "新しい担当", security: "アカウントのセキュリティ", account: "アカウント" },
  },
};

const b = (v: unknown) => `<strong>${escape(v)}</strong>`;

// ---------------------------------------------------------------- Nội dung từng mẫu

interface Content {
  subject: string;
  eyebrow: string;
  title: string;
  greeting?: string;
  intro: string; // HTML đã escape
  details?: [string, string][];
  quote?: string;
  note?: string;
  action?: { label: string; url: string };
  secondary?: { label: string; url: string };
  footerNote?: string;
}

type Builder = (d: Record<string, unknown>, locale: Locale, c: Common) => Content;

const roleName = (c: Common, r: unknown) => c.roles[s(r)] ?? s(r);
const sysRoleName = (c: Common, r: unknown) => c.systemRoles[s(r)] ?? s(r);
const change = (from: string, to: string) => (from && from !== to ? `${from} → ${to}` : to);

const builders: Record<EmailTemplate, Builder> = {
  project_invitation: (d, locale, c) => {
    const expires = formatDate(d.expiresAt, locale);
    const role = roleName(c, d.role);
    const t = {
      vi: {
        subject: `Bạn được mời tham gia ${s(d.projectName)} trên ReqSentinel`,
        title: `Tham gia ${s(d.projectName)}`,
        intro: `${b(d.inviterName)} đã mời bạn vào dự án ${b(d.projectName)} với vai trò ${b(role)}.`,
        note: "Nếu chưa có tài khoản, hãy đăng ký bằng chính địa chỉ email này rồi chấp nhận lời mời.",
        button: "Chấp nhận lời mời",
        expires: `Lời mời hết hạn lúc ${expires}.`,
      },
      en: {
        subject: `You're invited to ${s(d.projectName)} on ReqSentinel`,
        title: `Join ${s(d.projectName)}`,
        intro: `${b(d.inviterName)} invited you to the project ${b(d.projectName)} as ${b(role)}.`,
        note: "If you don't have an account yet, sign up with this email address, then accept the invitation.",
        button: "Accept invitation",
        expires: `This invitation expires on ${expires}.`,
      },
      ja: {
        subject: `ReqSentinel の「${s(d.projectName)}」に招待されました`,
        title: `「${s(d.projectName)}」に参加`,
        intro: `${b(d.inviterName)} さんがあなたをプロジェクト ${b(d.projectName)} に ${b(role)} として招待しました。`,
        note: "アカウントをお持ちでない場合は、このメールアドレスで登録してから招待を承認してください。",
        button: "招待を承認する",
        expires: `この招待の有効期限は ${expires} です。`,
      },
    }[locale];
    return {
      subject: t.subject, eyebrow: c.eyebrows.invitation, title: t.title, intro: t.intro,
      details: [[c.labels.project, s(d.projectName)], [c.labels.role, role], [c.labels.inviter, s(d.inviterName)]],
      note: t.note, action: { label: t.button, url: s(d.acceptUrl) }, footerNote: expires ? t.expires : undefined,
    };
  },

  password_reset: (d, locale, c) => {
    const t = {
      vi: {
        subject: "Đặt lại mật khẩu ReqSentinel", title: "Đặt lại mật khẩu",
        intro: "Chúng tôi nhận được yêu cầu đặt lại mật khẩu của bạn. Liên kết dưới đây có hiệu lực trong 30 phút và chỉ dùng được một lần.",
        button: "Đặt lại mật khẩu", note: "Nếu bạn không yêu cầu, hãy bỏ qua email này. Mật khẩu của bạn không thay đổi.",
      },
      en: {
        subject: "Reset your ReqSentinel password", title: "Reset your password",
        intro: "We received a request to reset your password. The link below is valid for 30 minutes and can be used once.",
        button: "Reset password", note: "If you did not request this, ignore this email. Your password stays the same.",
      },
      ja: {
        subject: "ReqSentinel のパスワード再設定", title: "パスワードの再設定",
        intro: "パスワード再設定のリクエストを受け付けました。下のリンクは 30 分間有効で、1 回のみ使用できます。",
        button: "パスワードを再設定", note: "お心当たりがない場合は、このメールを無視してください。パスワードは変更されません。",
      },
    }[locale];
    return {
      subject: t.subject, eyebrow: c.eyebrows.security, title: t.title, greeting: c.greeting(s(d.name)),
      intro: escape(t.intro), action: { label: t.button, url: s(d.resetUrl) }, note: t.note,
    };
  },

  member_role_changed: (d, locale, c) => {
    const role = roleName(c, d.role);
    const from = d.fromRole ? roleName(c, d.fromRole) : "";
    const desc = c.roleDescriptions[s(d.role)] ?? "";
    const t = {
      vi: {
        subject: `Vai trò của bạn trong ${s(d.projectName)} đã thay đổi`, title: "Vai trò của bạn đã thay đổi",
        intro: `${b(d.actorName)} đã đổi vai trò của bạn trong dự án ${b(d.projectName)} thành ${b(role)}.`,
        note: desc ? `Với vai trò ${role}, bạn có thể: ${desc}.` : "", button: "Mở dự án",
      },
      en: {
        subject: `Your role in ${s(d.projectName)} has changed`, title: "Your role has changed",
        intro: `${b(d.actorName)} changed your role in the project ${b(d.projectName)} to ${b(role)}.`,
        note: desc ? `As ${role}, you can ${desc}.` : "", button: "Open project",
      },
      ja: {
        subject: `「${s(d.projectName)}」でのロールが変更されました`, title: "ロールが変更されました",
        intro: `${b(d.actorName)} さんがプロジェクト ${b(d.projectName)} でのあなたのロールを ${b(role)} に変更しました。`,
        note: desc ? `${role}としてできること: ${desc}。` : "", button: "プロジェクトを開く",
      },
    }[locale];
    return {
      subject: t.subject, eyebrow: c.eyebrows.permission, title: t.title, greeting: c.greeting(s(d.name)), intro: t.intro,
      details: [[c.labels.project, s(d.projectName)], [c.labels.role, change(from, role)], [c.labels.actor, s(d.actorName)],
        [c.labels.time, formatDate(d.at, locale)]],
      note: t.note || undefined, action: d.url ? { label: t.button, url: s(d.url) } : undefined,
    };
  },

  member_removed: (d, locale, c) => {
    const t = {
      vi: {
        subject: `Bạn không còn là thành viên của ${s(d.projectName)}`, title: "Bạn đã được gỡ khỏi dự án",
        intro: `${b(d.actorName)} đã gỡ bạn khỏi dự án ${b(d.projectName)}. Bạn không còn xem hay chỉnh sửa được dữ liệu của dự án này.`,
        note: "Nếu đây là nhầm lẫn, hãy liên hệ chủ dự án để được mời lại.", button: "Xem các dự án của bạn",
      },
      en: {
        subject: `You were removed from ${s(d.projectName)}`, title: "You were removed from a project",
        intro: `${b(d.actorName)} removed you from the project ${b(d.projectName)}. You can no longer view or edit its data.`,
        note: "If this is a mistake, ask a project owner to invite you again.", button: "See your projects",
      },
      ja: {
        subject: `「${s(d.projectName)}」から削除されました`, title: "プロジェクトから削除されました",
        intro: `${b(d.actorName)} さんがあなたをプロジェクト ${b(d.projectName)} から削除しました。このプロジェクトのデータは閲覧・編集できなくなりました。`,
        note: "誤りの場合は、プロジェクトのオーナーに再招待を依頼してください。", button: "プロジェクト一覧を見る",
      },
    }[locale];
    return {
      subject: t.subject, eyebrow: c.eyebrows.permission, title: t.title, greeting: c.greeting(s(d.name)), intro: t.intro,
      details: [[c.labels.project, s(d.projectName)], [c.labels.role, d.role ? roleName(c, d.role) : "—"],
        [c.labels.actor, s(d.actorName)], [c.labels.time, formatDate(d.at, locale)]],
      note: t.note, action: d.url ? { label: t.button, url: s(d.url) } : undefined,
    };
  },

  conflict_assigned: (d, locale, c) => {
    const type = c.conflictTypes[s(d.conflictType)] ?? s(d.conflictType);
    const t = {
      vi: {
        subject: `Bạn được giao xử lý một xung đột trong ${s(d.projectName)}`, title: "Bạn có một xung đột cần xử lý",
        intro: `${b(d.actorName)} đã giao cho bạn xử lý một ${b(type)} trong dự án ${b(d.projectName)}.`, button: "Mở xung đột",
        actor: "Người giao",
      },
      en: {
        subject: `You've been assigned a conflict in ${s(d.projectName)}`, title: "A conflict needs your attention",
        intro: `${b(d.actorName)} assigned you a ${b(type)} in the project ${b(d.projectName)}.`, button: "Open conflict",
        actor: "Assigned by",
      },
      ja: {
        subject: `「${s(d.projectName)}」のコンフリクト対応が割り当てられました`, title: "対応が必要なコンフリクトがあります",
        intro: `${b(d.actorName)} さんがプロジェクト ${b(d.projectName)} の ${b(type)} をあなたに割り当てました。`, button: "コンフリクトを開く",
        actor: "割り当てた人",
      },
    }[locale];
    return {
      subject: t.subject, eyebrow: c.eyebrows.task, title: t.title, greeting: c.greeting(s(d.name)), intro: t.intro,
      details: [[c.labels.project, s(d.projectName)], [c.labels.conflictType, type], [t.actor, s(d.actorName)],
        [c.labels.time, formatDate(d.at, locale)]],
      quote: s(d.explanation) || undefined, action: d.url ? { label: t.button, url: s(d.url) } : undefined,
    };
  },

  account_created: (d, locale, c) => {
    const t = {
      vi: {
        subject: "Tài khoản ReqSentinel của bạn đã được tạo", title: "Chào mừng đến với ReqSentinel",
        intro: `Quản trị viên ${b(d.actorName)} đã tạo tài khoản ReqSentinel cho bạn.`,
        note: "Vì lý do bảo mật, mật khẩu không được gửi qua email. Hãy dùng mật khẩu quản trị viên cung cấp, hoặc tự đặt mật khẩu mới bằng liên kết bên dưới.",
        button: "Đăng nhập", secondary: "Tự đặt mật khẩu", actor: "Người tạo",
      },
      en: {
        subject: "Your ReqSentinel account is ready", title: "Welcome to ReqSentinel",
        intro: `Admin ${b(d.actorName)} created a ReqSentinel account for you.`,
        note: "For security, passwords are never sent by email. Use the password your admin gave you, or set your own with the link below.",
        button: "Sign in", secondary: "Set your own password", actor: "Created by",
      },
      ja: {
        subject: "ReqSentinel のアカウントが作成されました", title: "ReqSentinel へようこそ",
        intro: `管理者 ${b(d.actorName)} さんがあなたの ReqSentinel アカウントを作成しました。`,
        note: "セキュリティのため、パスワードはメールで送信されません。管理者から受け取ったパスワードを使うか、下のリンクからご自身で設定してください。",
        button: "ログイン", secondary: "パスワードを自分で設定", actor: "作成者",
      },
    }[locale];
    return {
      subject: t.subject, eyebrow: c.eyebrows.account, title: t.title, greeting: c.greeting(s(d.name)), intro: t.intro,
      details: [[c.labels.email, s(d.email)], [c.labels.systemRole, sysRoleName(c, d.role)], [t.actor, s(d.actorName)]],
      note: t.note, action: { label: t.button, url: s(d.url) },
      secondary: d.forgotUrl ? { label: t.secondary, url: s(d.forgotUrl) } : undefined,
    };
  },

  account_role_changed: (d, locale, c) => {
    const promoted = s(d.role) === "admin";
    const role = sysRoleName(c, d.role);
    const t = {
      vi: {
        subject: "Vai trò hệ thống của bạn đã thay đổi",
        title: promoted ? "Bạn đã trở thành quản trị viên" : "Bạn không còn là quản trị viên",
        intro: `${b(d.actorName)} đã đổi vai trò hệ thống của bạn thành ${b(role)}.`,
        note: promoted
          ? "Bạn có thể quản lý người dùng, xem nhật ký kiểm toán và có quyền chủ sở hữu trên mọi dự án."
          : "Các chức năng quản trị không còn khả dụng; vai trò của bạn trong từng dự án giữ nguyên.",
        button: "Mở ReqSentinel",
      },
      en: {
        subject: "Your system role has changed",
        title: promoted ? "You are now an admin" : "You are no longer an admin",
        intro: `${b(d.actorName)} changed your system role to ${b(role)}.`,
        note: promoted
          ? "You can now manage users, read the audit log and act as owner on every project."
          : "Admin features are no longer available; your role in each project stays the same.",
        button: "Open ReqSentinel",
      },
      ja: {
        subject: "システムロールが変更されました",
        title: promoted ? "管理者になりました" : "管理者ではなくなりました",
        intro: `${b(d.actorName)} さんがあなたのシステムロールを ${b(role)} に変更しました。`,
        note: promoted
          ? "ユーザー管理、監査ログの閲覧ができ、すべてのプロジェクトでオーナー権限を持ちます。"
          : "管理機能は利用できなくなりました。各プロジェクトでのロールは変わりません。",
        button: "ReqSentinel を開く",
      },
    }[locale];
    return {
      subject: t.subject, eyebrow: c.eyebrows.permission, title: t.title, greeting: c.greeting(s(d.name)), intro: t.intro,
      details: [[c.labels.systemRole, change(d.fromRole ? sysRoleName(c, d.fromRole) : "", role)],
        [c.labels.actor, s(d.actorName)], [c.labels.time, formatDate(d.at, locale)]],
      note: t.note, action: d.url ? { label: t.button, url: s(d.url) } : undefined,
    };
  },

  account_status_changed: (d, locale, c) => {
    const disabled = s(d.status) === "disabled";
    const t = {
      vi: disabled
        ? {
            subject: "Tài khoản ReqSentinel của bạn đã bị khóa", title: "Tài khoản đã bị khóa",
            intro: `Quản trị viên ${b(d.actorName)} đã khóa tài khoản của bạn. Bạn sẽ không đăng nhập được cho tới khi tài khoản được mở lại.`,
            note: "Nếu cho rằng đây là nhầm lẫn, hãy liên hệ quản trị viên hệ thống.", button: "",
          }
        : {
            subject: "Tài khoản ReqSentinel của bạn đã được mở khóa", title: "Tài khoản đã được mở khóa",
            intro: `Quản trị viên ${b(d.actorName)} đã mở khóa tài khoản của bạn. Bạn có thể đăng nhập lại bình thường.`,
            note: "", button: "Đăng nhập",
          },
      en: disabled
        ? {
            subject: "Your ReqSentinel account has been disabled", title: "Account disabled",
            intro: `Admin ${b(d.actorName)} disabled your account. You cannot sign in until it is enabled again.`,
            note: "If you think this is a mistake, contact your system admin.", button: "",
          }
        : {
            subject: "Your ReqSentinel account has been enabled", title: "Account enabled",
            intro: `Admin ${b(d.actorName)} enabled your account again. You can sign in as usual.`,
            note: "", button: "Sign in",
          },
      ja: disabled
        ? {
            subject: "ReqSentinel のアカウントが無効化されました", title: "アカウントが無効化されました",
            intro: `管理者 ${b(d.actorName)} さんがあなたのアカウントを無効化しました。再び有効化されるまでログインできません。`,
            note: "誤りと思われる場合は、システム管理者にお問い合わせください。", button: "",
          }
        : {
            subject: "ReqSentinel のアカウントが有効化されました", title: "アカウントが有効化されました",
            intro: `管理者 ${b(d.actorName)} さんがあなたのアカウントを再び有効化しました。通常どおりログインできます。`,
            note: "", button: "ログイン",
          },
    }[locale];
    return {
      subject: t.subject, eyebrow: c.eyebrows.security, title: t.title, greeting: c.greeting(s(d.name)), intro: t.intro,
      details: [[c.labels.actor, s(d.actorName)], [c.labels.time, formatDate(d.at, locale)]],
      note: t.note || undefined, action: t.button && d.url ? { label: t.button, url: s(d.url) } : undefined,
    };
  },

  password_changed: (d, locale, c) => {
    const by = s(d.by);
    const t = {
      vi: {
        subject: "Mật khẩu ReqSentinel của bạn vừa được thay đổi", title: "Mật khẩu đã được thay đổi",
        intro: {
          self: "Mật khẩu tài khoản của bạn vừa được đổi từ trang hồ sơ.",
          reset: "Mật khẩu tài khoản của bạn vừa được đặt lại qua liên kết gửi tới email này.",
          admin: `Quản trị viên ${b(d.actorName)} đã đặt lại mật khẩu cho tài khoản của bạn. Hãy đăng nhập bằng mật khẩu mới được cung cấp rồi đổi sang mật khẩu của riêng bạn.`,
        },
        note: "Nếu không phải bạn thực hiện, hãy đặt lại mật khẩu ngay và báo cho quản trị viên.",
        reset: "Đặt lại mật khẩu", login: "Đăng nhập",
      },
      en: {
        subject: "Your ReqSentinel password was changed", title: "Password changed",
        intro: {
          self: "Your account password was just changed from your profile page.",
          reset: "Your account password was just reset with a link sent to this email.",
          admin: `Admin ${b(d.actorName)} reset the password of your account. Sign in with the new password you were given, then change it to your own.`,
        },
        note: "If this wasn't you, reset your password right away and tell your admin.",
        reset: "Reset password", login: "Sign in",
      },
      ja: {
        subject: "ReqSentinel のパスワードが変更されました", title: "パスワードが変更されました",
        intro: {
          self: "プロフィールページからアカウントのパスワードが変更されました。",
          reset: "このメールに送信されたリンクからアカウントのパスワードが再設定されました。",
          admin: `管理者 ${b(d.actorName)} さんがあなたのアカウントのパスワードを再設定しました。受け取った新しいパスワードでログインし、ご自身のパスワードに変更してください。`,
        },
        note: "お心当たりがない場合は、すぐにパスワードを再設定し、管理者に連絡してください。",
        reset: "パスワードを再設定", login: "ログイン",
      },
    }[locale];
    const intro = by === "admin" ? t.intro.admin : escape(by === "reset" ? t.intro.reset : t.intro.self);
    const details: [string, string][] = [[c.labels.time, formatDate(d.at, locale)]];
    if (by === "admin") details.unshift([c.labels.actor, s(d.actorName)]);
    return {
      subject: t.subject, eyebrow: c.eyebrows.security, title: t.title, greeting: c.greeting(s(d.name)), intro, details,
      note: by === "admin" ? undefined : t.note,
      action: by === "admin" ? { label: t.login, url: s(d.url) } : { label: t.reset, url: s(d.forgotUrl) },
    };
  },
};

// ---------------------------------------------------------------- Khung email (cùng hệ thiết kế với web-app)

const C = {
  bg: "#efebe6", surface: "#f8f6f3", header: "#111111", headerText: "#f3f1ee", headerMuted: "#8d8984",
  accent: "#f26a1d", accentStrong: "#d95a10", ink: "#121212", text: "#151515", text2: "#3d3a37", muted: "#76706a", line: "#d8d1c9",
};
const DISPLAY = `Anton,Impact,'Arial Narrow Bold','Arial Narrow','Noto Sans JP','Hiragino Kaku Gothic ProN','Yu Gothic',sans-serif`;
const BODY = `Inter,'Segoe UI',Roboto,Helvetica,Arial,'Noto Sans JP','Hiragino Kaku Gothic ProN','Yu Gothic',sans-serif`;

function wordmark(): string {
  return `<span style="font-family:${DISPLAY};font-size:26px;line-height:1;letter-spacing:0.5px;color:${C.headerText}">Req<span style="color:${C.accent};font-size:32px;font-style:italic">S</span>entinel</span>`;
}

function button(a: { label: string; url: string }): string {
  return `<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:28px 0 8px"><tr>
<td style="background:${C.accent};border:1.5px solid ${C.ink};border-radius:4px">
<a href="${escape(a.url)}" style="display:inline-block;padding:13px 26px;font-family:${BODY};font-size:14px;font-weight:700;letter-spacing:0.4px;text-transform:uppercase;color:${C.ink};text-decoration:none">${escape(a.label)} &rarr;</a>
</td></tr></table>`;
}

function layout(locale: Locale, k: Content): string {
  const c = COMMON[locale];
  const p = (html: string, extra = "") =>
    `<p style="margin:0 0 14px;font-family:${BODY};font-size:15px;line-height:1.6;color:${C.text2};${extra}">${html}</p>`;
  const details = k.details?.filter(([, v]) => v).length
    ? `<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:20px 0;background:${C.bg};border-left:3px solid ${C.accent}">
${k.details
  .filter(([, v]) => v)
  .map(
    ([label, value]) => `<tr><td style="padding:10px 16px 10px 18px;width:38%;vertical-align:top;font-family:${BODY};font-size:11px;font-weight:700;letter-spacing:1px;text-transform:uppercase;color:${C.muted}">${escape(label)}</td>
<td style="padding:10px 16px 10px 0;vertical-align:top;font-family:${BODY};font-size:14px;font-weight:600;color:${C.text}">${escape(value)}</td></tr>`,
  )
  .join("")}
</table>`
    : "";
  const quote = k.quote
    ? `<div style="margin:0 0 18px;padding:12px 16px;border-left:3px solid ${C.text};background:#ffffff;font-family:${BODY};font-size:14px;line-height:1.55;font-style:italic;color:${C.text2}">${escape(k.quote)}</div>`
    : "";
  const action = k.action?.url ? button(k.action) : "";
  const secondary = k.secondary?.url
    ? `<p style="margin:6px 0 0;font-family:${BODY};font-size:13px"><a href="${escape(k.secondary.url)}" style="color:${C.accentStrong};font-weight:600">${escape(k.secondary.label)}</a></p>`
    : "";
  const fallback = k.action?.url
    ? `<p style="margin:18px 0 0;font-family:${BODY};font-size:12px;line-height:1.5;color:${C.muted}">${escape(c.orOpen)}<br><a href="${escape(k.action.url)}" style="color:${C.text};word-break:break-all">${escape(k.action.url)}</a></p>`
    : "";

  return `<!doctype html>
<html lang="${locale}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light"><meta name="supported-color-schemes" content="light">
<link href="https://fonts.googleapis.com/css2?family=Anton&amp;family=Inter:wght@400;600;700&amp;display=swap" rel="stylesheet">
<title>${escape(k.subject)}</title></head>
<body style="margin:0;padding:0;background:${C.bg}">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:${C.bg}"><tr><td align="center" style="padding:32px 12px">
<table role="presentation" width="560" cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:560px">
<tr><td style="background:${C.header};padding:22px 28px 18px">${wordmark()}
<div style="margin-top:6px;font-family:${BODY};font-size:10px;font-weight:600;letter-spacing:2px;text-transform:uppercase;color:${C.headerMuted}">${escape(c.tagline)}</div></td></tr>
<tr><td style="background:${C.accent};height:4px;line-height:4px;font-size:0">&nbsp;</td></tr>
<tr><td style="background:${C.surface};padding:32px 28px 28px;border-left:1px solid ${C.line};border-right:1px solid ${C.line}">
<div style="margin:0 0 10px;font-family:${BODY};font-size:11px;font-weight:700;letter-spacing:2px;text-transform:uppercase;color:${C.accentStrong}">&#9679;&nbsp;${escape(k.eyebrow)}</div>
<h1 style="margin:0 0 18px;font-family:${DISPLAY};font-weight:400;font-size:30px;line-height:1.15;text-transform:uppercase;color:${C.text}">${escape(k.title)}</h1>
${k.greeting ? p(escape(k.greeting), `color:${C.text};font-weight:600`) : ""}
${p(k.intro)}
${details}${quote}
${k.note ? p(escape(k.note), `font-size:13px;color:${C.muted}`) : ""}
${action}${secondary}${fallback}
${k.footerNote ? `<p style="margin:20px 0 0;font-family:${BODY};font-size:12px;color:${C.muted}">${escape(k.footerNote)}</p>` : ""}
</td></tr>
<tr><td style="background:${C.header};padding:16px 28px;font-family:${BODY};font-size:12px;line-height:1.5;color:${C.headerMuted}">${escape(c.footer)}</td></tr>
</table></td></tr></table></body></html>`;
}

const stripTags = (html: string) =>
  html
    .replace(/<[^>]+>/g, "")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"');

function plainText(locale: Locale, k: Content): string {
  const lines = [k.title.toUpperCase(), ""];
  if (k.greeting) lines.push(k.greeting, "");
  lines.push(stripTags(k.intro), "");
  for (const [label, value] of k.details ?? []) if (value) lines.push(`${label}: ${value}`);
  if (k.details?.length) lines.push("");
  if (k.quote) lines.push(`"${k.quote}"`, "");
  if (k.note) lines.push(k.note, "");
  if (k.action?.url) lines.push(`${k.action.label}: ${k.action.url}`);
  if (k.secondary?.url) lines.push(`${k.secondary.label}: ${k.secondary.url}`);
  if (k.footerNote) lines.push("", k.footerNote);
  lines.push("", "— ReqSentinel", COMMON[locale].footer);
  return lines.join("\n");
}

export function renderEmail(template: EmailTemplate, data: Record<string, unknown>, locale: Locale): RenderedEmail {
  const content = builders[template](data, locale, COMMON[locale]);
  return { subject: content.subject, html: layout(locale, content), text: plainText(locale, content) };
}
