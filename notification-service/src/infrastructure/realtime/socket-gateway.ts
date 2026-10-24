import type { Server as HttpServer } from "node:http";
import { Server } from "socket.io";
import type { Logger, RealtimePublisher } from "../../application/ports.js";
import type { Notification, RealtimeSignal } from "../../domain/notification.js";
import type { JwtVerifier } from "../auth/jwt-verifier.js";

const room = (userId: string) => `user:${userId}`;

/** Đẩy thông báo realtime qua Socket.IO; mỗi user 1 room, xác thực bằng access token của api-service. */
export class SocketGateway implements RealtimePublisher {
  private readonly io: Server;

  constructor(httpServer: HttpServer, verifier: JwtVerifier, corsOrigins: string[], log: Logger) {
    this.io = new Server(httpServer, {
      path: "/socket.io",
      cors: corsOrigins.length ? { origin: corsOrigins } : undefined,
    });
    this.io.use((socket, next) => {
      const token = (socket.handshake.auth as { token?: string } | undefined)?.token;
      const userId = verifier.verify(token);
      if (!userId) return next(new Error("unauthorized"));
      socket.data.userId = userId;
      next();
    });
    this.io.on("connection", (socket) => {
      const userId = socket.data.userId as string;
      void socket.join(room(userId));
      log.info("socket connected", { userId, sockets: this.io.engine.clientsCount });
    });
  }

  notification(n: Notification, unreadCount: number): void {
    this.io.to(room(n.userId)).emit("notification", { notification: serialize(n), unread_count: unreadCount });
  }

  signal(s: RealtimeSignal): void {
    this.io.to(room(s.userId)).emit(s.channel, s.payload);
  }

  close(): Promise<void> {
    return new Promise((resolve) => this.io.close(() => resolve()));
  }
}

export function serialize(n: Notification) {
  return {
    id: n.id,
    type: n.type,
    title: n.title,
    body: n.body,
    params: n.params,
    link: n.link,
    read_at: n.readAt,
    created_at: n.createdAt,
  };
}
