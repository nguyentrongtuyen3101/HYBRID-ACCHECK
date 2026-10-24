// Lưu phiên đăng nhập ở localStorage.
const KEY = "reqsentinel.auth";

export const sessionStorageAdapter = {
  load() {
    try {
      const s = JSON.parse(localStorage.getItem(KEY));
      if (!s?.access_token) return null;
      if (s.expires_at && new Date(s.expires_at) <= new Date()) return null;
      return s;
    } catch {
      return null;
    }
  },
  save(session) {
    if (session) localStorage.setItem(KEY, JSON.stringify(session));
    else localStorage.removeItem(KEY);
  },
};
