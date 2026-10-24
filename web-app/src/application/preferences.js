import { reactive } from "vue";
import { LOCALES, THEMES } from "../domain/preferences.js";

/**
 * Ngôn ngữ + theme của giao diện.
 * Thứ tự ưu tiên: profile của user (nếu đã đặt) → lựa chọn lưu trên máy → trình duyệt / hệ điều hành.
 * @param {{
 *   storage: { load: () => {locale?: string, theme?: string}, save: (p: object) => void },
 *   environment: { browserLocale: () => string, prefersDark: () => boolean },
 *   apply: { locale: (l: string) => void, theme: (t: string) => void },
 * }} deps
 */
export function createPreferences({ storage, environment, apply }) {
  const saved = storage.load();
  const state = reactive({
    locale: LOCALES.includes(saved.locale) ? saved.locale : environment.browserLocale(),
    theme: THEMES.includes(saved.theme) ? saved.theme : environment.prefersDark() ? "dark" : "light",
  });
  apply.locale(state.locale);
  apply.theme(state.theme);

  let persistRemote = null; // (patch) => Promise, gắn khi đã đăng nhập

  function set(key, value) {
    if (state[key] === value) return;
    state[key] = value;
    apply[key](value);
    storage.save({ locale: state.locale, theme: state.theme });
    persistRemote?.({ [key]: value }).catch(() => {
      /* lưu lên profile thất bại không ảnh hưởng giao diện */
    });
  }

  return {
    state,
    setLocale: (l) => LOCALES.includes(l) && set("locale", l),
    setTheme: (t) => THEMES.includes(t) && set("theme", t),
    toggleTheme() {
      set("theme", state.theme === "dark" ? "light" : "dark");
    },
    /**
     * Sau khi đăng nhập: áp dụng tuỳ chọn đã lưu trong profile và bật đồng bộ lên server.
     * Profile chưa có tuỳ chọn nào thì lưu lựa chọn hiện tại lên (email gửi theo ngôn ngữ này).
     */
    attachUser(user, saveToProfile) {
      persistRemote = null;
      if (user?.locale && user.locale !== state.locale) set("locale", user.locale);
      if (user?.theme && user.theme !== state.theme) set("theme", user.theme);
      persistRemote = saveToProfile;
      const missing = {};
      if (user && !user.locale) missing.locale = state.locale;
      if (user && !user.theme) missing.theme = state.theme;
      if (Object.keys(missing).length) {
        saveToProfile?.(missing).catch(() => {
          /* như trên */
        });
      }
    },
    detachUser() {
      persistRemote = null;
    },
  };
}
