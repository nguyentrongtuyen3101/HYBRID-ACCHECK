import { describe, expect, it } from "vitest";
import { normalizeDatabaseUrl } from "../src/config.js";

describe("normalizeDatabaseUrl", () => {
  it("makes sslmode=require behave like libpq (same as the Go api-service)", () => {
    expect(normalizeDatabaseUrl("postgres://u:p@h:5432/db?sslmode=require")).toBe(
      "postgres://u:p@h:5432/db?sslmode=require&uselibpqcompat=true",
    );
  });
  it("keeps explicit settings and plain URLs untouched", () => {
    const strict = "postgres://u:p@h/db?sslmode=verify-full&sslrootcert=/ca.crt";
    expect(normalizeDatabaseUrl(strict)).toBe(strict);
    expect(normalizeDatabaseUrl("postgres://u:p@h/db")).toBe("postgres://u:p@h/db");
    expect(normalizeDatabaseUrl("postgres://u:p@h/db?sslmode=require&uselibpqcompat=false")).toContain("uselibpqcompat=false");
  });
  it("accepts the old SQLAlchemy scheme", () => {
    expect(normalizeDatabaseUrl("postgresql+asyncpg://u:p@h/db")).toBe("postgres://u:p@h/db");
  });
});
