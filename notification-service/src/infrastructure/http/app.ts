import express, { type NextFunction, type Request, type Response } from "express";
import { NotFoundError, type NotificationInbox } from "../../application/notification-queries.js";
import type { Logger } from "../../application/ports.js";
import type { JwtVerifier } from "../auth/jwt-verifier.js";
import { serialize } from "../realtime/socket-gateway.js";

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

interface AuthedRequest extends Request {
  userId?: string;
}

/** REST: hộp thư thông báo của người dùng đang đăng nhập. */
export function createHttpApp(inbox: NotificationInbox, verifier: JwtVerifier, health: () => Promise<object>, log: Logger) {
  const app = express();
  app.disable("x-powered-by");
  app.use(express.json({ limit: "100kb" }));

  app.get("/health", async (_req, res) => {
    res.json(await health());
  });

  const auth = (req: AuthedRequest, res: Response, next: NextFunction) => {
    const header = req.header("authorization") ?? "";
    const userId = verifier.verify(header.startsWith("Bearer ") ? header.slice(7) : undefined);
    if (!userId) {
      res.status(401).json({ detail: "Not authenticated", code: "not_authenticated" });
      return;
    }
    req.userId = userId;
    next();
  };

  const router = express.Router();
  router.use(auth);

  router.get("/", async (req: AuthedRequest, res) => {
    const result = await inbox.list(req.userId!, {
      limit: Number(req.query.limit) || undefined,
      before: typeof req.query.before === "string" ? req.query.before : undefined,
      unreadOnly: req.query.unread === "true",
    });
    res.json({ items: result.items.map(serialize), unread_count: result.unreadCount });
  });

  router.get("/unread-count", async (req: AuthedRequest, res) => {
    res.json({ unread_count: await inbox.unreadCount(req.userId!) });
  });

  router.post("/read-all", async (req: AuthedRequest, res) => {
    res.json({ unread_count: await inbox.markAllRead(req.userId!) });
  });

  router.post("/:id/read", async (req: AuthedRequest, res) => {
    const id = String(req.params.id);
    if (!UUID.test(id)) {
      res.status(404).json({ detail: "Notification not found", code: "notification_not_found" });
      return;
    }
    res.json({ unread_count: await inbox.markRead(req.userId!, id) });
  });

  app.use("/api/notifications", router);

  app.use((_req, res) => {
    res.status(404).json({ detail: "Not found", code: "not_found" });
  });
  app.use((err: unknown, req: Request, res: Response, _next: NextFunction) => {
    if (err instanceof NotFoundError) {
      res.status(404).json({ detail: err.message, code: "notification_not_found" });
      return;
    }
    log.error("http error", { path: req.path, err: String(err) });
    res.status(500).json({ detail: "Internal server error", code: "internal_error" });
  });
  return app;
}
