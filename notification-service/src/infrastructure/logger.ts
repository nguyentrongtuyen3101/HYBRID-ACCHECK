import type { Logger } from "../application/ports.js";

/** Log dạng JSON 1 dòng (dễ gom bằng Loki/ELK). */
export const consoleLogger: Logger = {
  info: (msg, meta) => console.log(JSON.stringify({ level: "info", time: new Date().toISOString(), msg, ...meta })),
  warn: (msg, meta) => console.warn(JSON.stringify({ level: "warn", time: new Date().toISOString(), msg, ...meta })),
  error: (msg, meta) => console.error(JSON.stringify({ level: "error", time: new Date().toISOString(), msg, ...meta })),
};
