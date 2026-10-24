// Truy cập trình duyệt: ngôn ngữ, chế độ sáng/tối, thuộc tính của <html>, localStorage.
import { matchLocale } from "../../domain/preferences.js";

const PREFS_KEY = "reqsentinel.preferences";

export const browserEnvironment = {
  browserLocale: () => matchLocale(navigator.languages?.length ? navigator.languages : [navigator.language]),
  prefersDark: () => window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? false,
};

export const documentAppearance = {
  locale(locale) {
    document.documentElement.lang = locale;
  },
  theme(theme) {
    document.documentElement.dataset.theme = theme;
  },
};

export const preferenceStorage = {
  load() {
    try {
      return JSON.parse(localStorage.getItem(PREFS_KEY)) || {};
    } catch {
      return {};
    }
  },
  save(prefs) {
    try {
      localStorage.setItem(PREFS_KEY, JSON.stringify(prefs));
    } catch {
      /* chế độ riêng tư / hết dung lượng: bỏ qua */
    }
  },
};
