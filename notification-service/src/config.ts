function env(key: string, def = ""): string {
  const v = process.env[key];
  return v === undefined || v === "" ? def : v;
}

/**
 * Chuẩn hoá DATABASE_URL để dùng chung được với api-service:
 * - chấp nhận dạng SQLAlchemy cũ postgresql+asyncpg://
 * - node-postgres v8 coi sslmode=require như verify-full (bắt buộc CA tin cậy), còn pgx bên Go
 *   theo libpq (mã hoá, không kiểm CA). Bật uselibpqcompat để 2 service hiểu giống nhau
 *   (vd Supabase dùng CA riêng). Muốn kiểm CA: sslmode=verify-full&sslrootcert=/đường/dẫn/ca.crt
 */
export function normalizeDatabaseUrl(raw: string): string {
  const url = raw.replace(/^postgresql\+\w+:\/\//, "postgres://");
  if (/[?&]sslmode=(require|prefer)\b/.test(url) && !/[?&]uselibpqcompat=/.test(url)) {
    return `${url}&uselibpqcompat=true`;
  }
  return url;
}

function required(key: string): string {
  const v = env(key);
  if (!v) throw new Error(`${key} is required`);
  return v;
}

export interface Config {
  port: number;
  databaseUrl: string;
  rabbitmqUrl: string;
  jwtSecret: string;
  corsOrigins: string[];
  smtp: { host: string; port: number; secure: boolean; user: string; pass: string; from: string };
  maxRetries: number;
}

export function loadConfig(): Config {
  // Nạp .env khi chạy local (biến môi trường có sẵn được ưu tiên); container truyền env trực tiếp
  try {
    process.loadEnvFile(".env");
  } catch {
    /* không có file .env */
  }
  const jwtSecret = required("JWT_SECRET");
  if (jwtSecret.length < 32) throw new Error("JWT_SECRET must be at least 32 characters (same as api-service)");
  return {
    port: Number(env("PORT", "8002")),
    databaseUrl: normalizeDatabaseUrl(required("DATABASE_URL")),
    rabbitmqUrl: required("RABBITMQ_URL"),
    jwtSecret,
    corsOrigins: env("CORS_ALLOWED_ORIGINS")
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean),
    smtp: {
      host: env("SMTP_HOST"),
      port: Number(env("SMTP_PORT", "587")),
      secure: env("SMTP_SECURE", "false") === "true",
      user: env("SMTP_USER"),
      pass: env("SMTP_PASSWORD"),
      // Gmail chỉ cho gửi bằng chính tài khoản đăng nhập → mặc định lấy SMTP_USER làm địa chỉ gửi
      from: env("MAIL_FROM", env("SMTP_USER") ? `ReqSentinel <${env("SMTP_USER")}>` : "ReqSentinel <no-reply@reqsentinel.local>"),
    },
    maxRetries: Number(env("EVENT_MAX_RETRIES", "5")),
  };
}
