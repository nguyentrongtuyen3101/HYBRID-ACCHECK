// Giá trị hợp lệ — khớp enum ui_locale / ui_theme của api-service.
export const LOCALES = ["vi", "en", "ja"];
export const THEMES = ["light", "dark"];
export const DEFAULT_LOCALE = "vi";

/** Chọn ngôn ngữ hỗ trợ gần nhất với tag của trình duyệt (vd "ja-JP" → "ja"). */
export function matchLocale(tags) {
  for (const tag of tags || []) {
    const base = String(tag).toLowerCase().split("-")[0];
    if (LOCALES.includes(base)) return base;
  }
  return DEFAULT_LOCALE;
}
