import { computed, reactive } from "vue";

/**
 * Trạng thái phiên đăng nhập + các thao tác tài khoản.
 * @param {{ authApi: object, storage: { load: Function, save: Function } }} deps
 */
export function createSession({ authApi, storage }) {
  const saved = storage.load();
  const state = reactive({
    token: saved?.access_token || null,
    expiresAt: saved?.expires_at || null,
    user: saved?.user || null,
  });

  const persist = () =>
    storage.save(state.token ? { access_token: state.token, expires_at: state.expiresAt, user: state.user } : null);

  function apply(data) {
    state.token = data.access_token;
    state.expiresAt = data.expires_at;
    state.user = data.user;
    persist();
  }

  const isLoggedIn = () => !!state.token && (!state.expiresAt || new Date(state.expiresAt) > new Date());

  return {
    state,
    user: computed(() => state.user),
    isLoggedIn,
    isAdmin: () => state.user?.role === "admin",
    token: () => (isLoggedIn() ? state.token : null),

    async login(email, password) {
      apply(await authApi.login(email.trim(), password));
    },
    async register({ email, password, fullName }) {
      apply(await authApi.register({ email: email.trim(), password, full_name: fullName.trim() }));
    },
    logout() {
      state.token = state.expiresAt = state.user = null;
      persist();
    },
    /** Đồng bộ lại profile/role từ server (role có thể đã bị admin đổi). */
    async refresh() {
      try {
        state.user = await authApi.me();
        persist();
      } catch {
        /* giữ thông tin cũ; 401 đã được http client xử lý */
      }
    },
    /** @param {object} patch các field profile cần đổi (full_name, job_title, locale, theme...) */
    async updateProfile(patch) {
      state.user = await authApi.updateProfile(patch);
      persist();
      return state.user;
    },
    changePassword: (current, next) => authApi.changePassword(current, next),
    forgotPassword: (email) => authApi.forgotPassword(email.trim()),
    resetPassword: (token, password) => authApi.resetPassword(token, password),
  };
}
