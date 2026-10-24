import { reactive } from "vue";

/**
 * Hộp thông báo: tải danh sách, nhận realtime, đánh dấu đã đọc.
 * Tín hiệu không lưu (vd tiến độ job) được phát lại cho các màn hình đăng ký qua onSignal.
 * @param {{ notificationApi: object, connect: (token: string) => { on: Function, close: Function } }} deps
 */
export function createNotificationCenter({ notificationApi, connect }) {
  const state = reactive({ items: [], unread: 0, loaded: false, connected: false });
  const listeners = new Map(); // channel → Set<handler>
  let socket = null;

  function emit(channel, payload) {
    for (const handler of listeners.get(channel) || []) handler(payload);
  }

  async function load() {
    try {
      const data = await notificationApi.list({ limit: 20 });
      state.items = data.items;
      state.unread = data.unread_count;
      state.loaded = true;
    } catch {
      state.loaded = true; // notification-service tắt không làm hỏng app chính
    }
  }

  return {
    state,
    async start(token) {
      this.stop();
      await load();
      socket = connect(token);
      socket.on("connect", () => (state.connected = true));
      socket.on("disconnect", () => (state.connected = false));
      socket.on("notification", ({ notification, unread_count }) => {
        state.items = [notification, ...state.items.filter((n) => n.id !== notification.id)].slice(0, 50);
        state.unread = unread_count;
        emit("notification", notification);
      });
      socket.on("job:progress", (payload) => emit("job:progress", payload));
    },
    stop() {
      socket?.close();
      socket = null;
      state.items = [];
      state.unread = 0;
      state.connected = false;
    },
    /** Đăng ký nhận tín hiệu realtime; trả về hàm huỷ đăng ký. */
    onSignal(channel, handler) {
      if (!listeners.has(channel)) listeners.set(channel, new Set());
      listeners.get(channel).add(handler);
      return () => listeners.get(channel).delete(handler);
    },
    async markRead(n) {
      if (n.read_at) return;
      const { unread_count } = await notificationApi.markRead(n.id);
      n.read_at = new Date().toISOString();
      state.unread = unread_count;
    },
    async markAllRead() {
      await notificationApi.markAllRead();
      const now = new Date().toISOString();
      state.items.forEach((n) => (n.read_at ||= now));
      state.unread = 0;
    },
  };
}
