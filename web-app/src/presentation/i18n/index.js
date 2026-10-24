import { createI18n } from "vue-i18n";
import en from "./locales/en.json";
import ja from "./locales/ja.json";
import vi from "./locales/vi.json";

// Tiếng Anh làm ngôn ngữ dự phòng khi thiếu khoá ở ngôn ngữ khác.
export function createAppI18n(locale) {
  return createI18n({
    legacy: false,
    locale,
    fallbackLocale: "en",
    messages: { vi, en, ja },
    missingWarn: import.meta.env.DEV,
    fallbackWarn: false,
  });
}
