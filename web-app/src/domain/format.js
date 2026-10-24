// Định dạng thuần (không phụ thuộc Vue); ngôn ngữ truyền vào từ tầng presentation.

export function formatDateTime(iso, locale) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat(locale, { dateStyle: "short", timeStyle: "short" }).format(d);
}

const UNITS = [
  ["year", 365 * 86400],
  ["month", 30 * 86400],
  ["week", 7 * 86400],
  ["day", 86400],
  ["hour", 3600],
  ["minute", 60],
];

export function timeAgo(iso, locale, now = Date.now()) {
  if (!iso) return "";
  const seconds = Math.round((new Date(iso).getTime() - now) / 1000);
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
  for (const [unit, size] of UNITS) {
    if (Math.abs(seconds) >= size) return rtf.format(Math.round(seconds / size), unit);
  }
  return rtf.format(0, "second");
}

export function formatPercent(value, locale) {
  return new Intl.NumberFormat(locale, { style: "percent", maximumFractionDigits: 0 }).format((Number(value) || 0) / 100);
}

export function splitLines(raw) {
  return String(raw || "")
    .split("\n")
    .map((s) => s.trim())
    .filter(Boolean);
}

/** Chữ viết tắt cho avatar: 2 từ cuối của tên (tên tiếng Việt: "Nguyễn Trọng Tuyển" → "TT"), không có tên thì lấy email. */
export function initialsOf(user) {
  const source = (user?.full_name || user?.email || "?").trim();
  return source
    .split(/\s+/)
    .slice(-2)
    .map((w) => w[0])
    .join("")
    .toUpperCase();
}
