import jwt from "jsonwebtoken";

/** Xác thực access token do api-service phát (HS256, cùng JWT_SECRET, iss=reqsentinel). */
export class JwtVerifier {
  constructor(private readonly secret: string) {}

  /** Trả về user id (sub) hoặc null nếu token không hợp lệ. */
  verify(token: string | undefined): string | null {
    if (!token) return null;
    try {
      const payload = jwt.verify(token, this.secret, { algorithms: ["HS256"], issuer: "reqsentinel" });
      return typeof payload === "object" && typeof payload.sub === "string" ? payload.sub : null;
    } catch {
      return null;
    }
  }
}
