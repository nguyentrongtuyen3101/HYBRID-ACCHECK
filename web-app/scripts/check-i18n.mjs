// Kiểm tra bản dịch: npm run i18n:check
//  1. mọi key dùng trong code (t("..."), te("...")) có trong cả 3 file locale
//  2. 3 file có cùng bộ key, cùng biến {placeholder}
//  3. mọi mã lỗi backend (api-service, notification-service) có bản dịch errors.<code>
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const LOCALES = ["vi", "en", "ja"];

const flat = (obj, prefix = "", out = {}) => {
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === "object") flat(v, key, out);
    else out[key] = String(v);
  }
  return out;
};
const locales = Object.fromEntries(
  LOCALES.map((l) => [l, flat(JSON.parse(fs.readFileSync(path.join(root, "src/presentation/i18n/locales", `${l}.json`), "utf8")))])
);

const files = (dir, exts) =>
  fs.readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) return e.name === "node_modules" ? [] : files(p, exts);
    return exts.some((x) => e.name.endsWith(x)) ? [p] : [];
  });

const used = new Set();
for (const f of files(path.join(root, "src"), [".vue", ".js"])) {
  for (const m of fs.readFileSync(f, "utf8").matchAll(/\bte?\(\s*["']([a-zA-Z0-9_.]+)["']/g)) used.add(m[1]);
}

// Mã lỗi backend: domain.Validation("code", …), writeDetail(w, status, "code", …), code: "…" (Node)
const codes = new Set();
const backends = [
  [path.join(root, "../api-service"), [".go"], /(?:Validation|NotFound|Forbidden|Conflict|Unauthorized|Unavailable|RateLimited|NotFoundAs\([^,]+,|writeDetail\([^,]+,[^,]+,)\s*\(?\s*"([a-z_]+)"/g],
  [path.join(root, "../notification-service/src"), [".ts"], /code:\s*"([a-z_]+)"/g],
];
for (const [dir, exts, re] of backends) {
  if (!fs.existsSync(dir)) continue;
  for (const f of files(dir, exts)) {
    if (f.endsWith("_test.go") || f.includes(".test.")) continue;
    for (const m of fs.readFileSync(f, "utf8").matchAll(re)) codes.add(m[1]);
  }
}
for (const c of codes) used.add(`errors.${c}`);

const placeholders = (s) => [...new Set([...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]))].sort().join(",");
const problems = [];
for (const l of LOCALES) {
  const missing = [...used].filter((k) => !(k in locales[l]));
  if (missing.length) problems.push(`[${l}] thiếu ${missing.length} key: ${missing.join(", ")}`);
}
const ref = locales.en;
for (const l of LOCALES.filter((x) => x !== "en")) {
  const onlyRef = Object.keys(ref).filter((k) => !(k in locales[l]));
  const extra = Object.keys(locales[l]).filter((k) => !(k in ref));
  if (onlyRef.length) problems.push(`[${l}] thiếu so với en: ${onlyRef.join(", ")}`);
  if (extra.length) problems.push(`[${l}] thừa so với en: ${extra.join(", ")}`);
  for (const k of Object.keys(ref)) {
    if (k in locales[l] && placeholders(ref[k]) !== placeholders(locales[l][k])) problems.push(`[${l}] biến {…} khác en ở ${k}`);
  }
}

console.log(`key dùng trong code: ${used.size} (gồm ${codes.size} mã lỗi backend) · key mỗi file: ${Object.keys(ref).length}`);
if (problems.length) {
  console.error(problems.join("\n"));
  process.exit(1);
}
console.log("OK — 3 ngôn ngữ đủ và khớp nhau");
