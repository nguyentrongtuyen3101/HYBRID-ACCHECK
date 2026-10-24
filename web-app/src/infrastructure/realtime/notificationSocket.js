import { io } from "socket.io-client";

/**
 * Kết nối Socket.IO tới notification-service (cùng origin, qua proxy /socket.io).
 * @returns {{ on: Function, close: Function }}
 */
export function connectNotificationSocket(token, url = import.meta.env.VITE_NOTIFY_URL || undefined) {
  const socket = io(url, { path: "/socket.io", auth: { token }, transports: ["websocket", "polling"] });
  return {
    on: (event, handler) => socket.on(event, handler),
    close: () => socket.close(),
  };
}
