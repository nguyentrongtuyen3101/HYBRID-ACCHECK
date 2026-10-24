/** Lỗi từ API: `code` + `params` để giao diện dịch (i18n); `message` là câu tiếng Anh dự phòng. */
export class HttpError extends Error {
  constructor(status, message, code = "", params = {}) {
    super(message);
    this.status = status;
    this.code = code;
    this.params = params;
  }
}

/**
 * HTTP client dùng chung: gắn token, chuẩn hoá lỗi {"detail": ...}, xử lý 401.
 * @param {{ baseUrl: string, getToken: () => string|null, onUnauthorized: () => void }} opts
 */
export function createHttpClient({ baseUrl, getToken, onUnauthorized }) {
  async function send(method, path, { body, raw = false } = {}) {
    const token = getToken();
    const headers = {};
    if (body !== undefined && !(body instanceof FormData)) headers["Content-Type"] = "application/json";
    if (token) headers.Authorization = `Bearer ${token}`;
    const res = await fetch(baseUrl + path, {
      method,
      headers,
      body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body),
    });
    if (res.status === 401 && token) {
      onUnauthorized();
      throw new HttpError(401, "Session expired, please log in again", "session_expired");
    }
    if (!res.ok) {
      let detail = res.statusText;
      let code = "";
      let params = {};
      try {
        const data = await res.json();
        if (data?.detail) detail = typeof data.detail === "string" ? data.detail : JSON.stringify(data.detail);
        code = data?.code || "";
        params = data?.params || {};
      } catch {
        /* body không phải JSON */
      }
      throw new HttpError(res.status, detail, code, params);
    }
    if (raw) return res;
    if (res.status === 204) return null;
    const text = await res.text();
    return text ? JSON.parse(text) : null;
  }

  return {
    get: (path) => send("GET", path),
    post: (path, body) => send("POST", path, { body }),
    put: (path, body) => send("PUT", path, { body }),
    patch: (path, body) => send("PATCH", path, { body }),
    del: (path) => send("DELETE", path),
    /** Tải file kèm token (Excel, JSON) → Blob + tên file. */
    async download(path) {
      const res = await send("GET", path, { raw: true });
      const disposition = res.headers.get("content-disposition") || "";
      const match = disposition.match(/filename\*=UTF-8''([^;]+)/) || disposition.match(/filename="([^"]+)"/);
      return { blob: await res.blob(), filename: match ? decodeURIComponent(match[1]) : "download" };
    },
  };
}
