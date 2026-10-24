import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// Dev: proxy sang các service backend (cùng origin → không cần CORS).
// Thứ tự quan trọng: /api/notifications phải đứng trước /api.
const api = process.env.API_PROXY_TARGET || "http://localhost:8000";
const notify = process.env.NOTIFY_PROXY_TARGET || "http://localhost:8002";

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      "/api/notifications": notify,
      "/socket.io": { target: notify, ws: true },
      "/api": api,
    },
  },
});
