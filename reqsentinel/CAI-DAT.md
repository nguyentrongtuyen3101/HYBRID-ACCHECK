# Cài đặt ReqSentinel trên máy mới

Có 2 cách. **Chỉ cần chạy để dùng / demo → Cách 1** (cài 2 phần mềm, không cài thư viện nào).
**Cần sửa code từng service → Cách 2.**

Mọi lệnh bên dưới chạy ở **thư mục gốc của dự án** (thư mục chứa `api-service/`, `web-app/`, `reqsentinel/`…).
Trên Windows dùng **Git Bash** (cài cùng Git) cho các khối lệnh `bash`; khối `powershell` chạy trong PowerShell.

---

## Cách 1 — Chạy bằng Docker (khuyên dùng)

### 1. Cài phần mềm

Windows (PowerShell, chạy 1 lần rồi khởi động lại máy):

```powershell
winget install --id Git.Git -e
winget install --id Docker.DockerDesktop -e
```

Ubuntu / Debian:

```bash
sudo apt update && sudo apt install -y git curl
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER      # đăng xuất / đăng nhập lại để dùng docker không cần sudo
```

Mở Docker Desktop (Windows) và đợi biểu tượng cá voi chuyển sang trạng thái chạy.

### 2. Lấy code và chạy

```bash
git clone https://github.com/nguyentrongtuyen3101/HYBRID-ACCHECK.git
cd HYBRID-ACCHECK
./reqsentinel/up.sh
```

- Lần đầu script tạo `reqsentinel/.env` rồi dừng: mở file đó, điền `JWT_SECRET` (chuỗi bất kỳ ≥ 32 ký tự) và
  `ADMIN_PASSWORD`, rồi chạy lại `./reqsentinel/up.sh`.
- Lần chạy kế tiếp tải và build khoảng 2–3 GB (image nền, torch, model SBERT) — 5–15 phút tùy mạng.
  Từ lần sau khởi động chỉ vài giây.
- Xong mở http://localhost:8080, đăng nhập bằng `ADMIN_EMAIL` / `ADMIN_PASSWORD` trong `reqsentinel/.env`.
- Dừng: `./reqsentinel/up.sh down`. Xem thêm lệnh trong [README.md](README.md#chạy-toàn-bộ-bằng-docker-compose).

---

## Cách 2 — Cài thư viện để chạy / sửa từng service

| Service | Ngôn ngữ | Phiên bản cần |
|---|---|---|
| api-service | Go | 1.26 trở lên |
| ml-service | Python | 3.12 (không dùng 3.13+, torch chưa hỗ trợ đủ) |
| notification-service, web-app | Node.js | 22 trở lên |
| Postgres, RabbitMQ, Mailpit | Docker | Docker Desktop / Docker Engine |

### Bước 1. Cài công cụ

Windows (PowerShell, xong thì **mở lại** terminal để nhận PATH mới):

```powershell
winget install --id Git.Git -e
winget install --id Docker.DockerDesktop -e
winget install --id GoLang.Go -e
winget install --id Python.Python.3.12 -e
winget install --id OpenJS.NodeJS.LTS -e
```

Ubuntu 24.04:

```bash
sudo apt update && sudo apt install -y git curl python3.12 python3.12-venv
sudo snap install go --classic
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash && source ~/.bashrc && nvm install 22
curl -fsSL https://get.docker.com | sudo sh && sudo usermod -aG docker $USER
```

Kiểm tra: `go version`, `node -v`, `py -3.12 --version` (Windows) hoặc `python3.12 --version` (Linux), `docker version`.

### Bước 2. Bật Postgres, RabbitMQ, Mailpit (khớp sẵn với các file `.env.example`)

```bash
docker run -d --name rs-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=hybrid_accheck -p 5432:5432 postgres:16-alpine
docker run -d --name rs-rabbitmq -e RABBITMQ_DEFAULT_VHOST=reqsentinel -p 5672:5672 -p 15672:15672 rabbitmq:3-management
docker run -d --name rs-mailpit -p 1025:1025 -p 8025:8025 axllent/mailpit
```

Lần sau chỉ cần `docker start rs-postgres rs-rabbitmq rs-mailpit`.
Dùng Supabase thay Postgres thì bỏ lệnh đầu và sửa `DATABASE_URL` ở bước 3.

### Bước 3. Cài thư viện cho cả 4 service

Copy **nguyên khối** vào Git Bash (Windows) hoặc terminal (Linux):

```bash
# --- api-service (Go): tải module
(cd api-service && go mod download && cp -n .env.example .env)

# --- ml-service (Python): venv + torch bản CPU (~200 MB) + thư viện còn lại
(cd ml-service \
  && { py -3.12 -m venv .venv 2>/dev/null || python3.12 -m venv .venv; } \
  && { source .venv/Scripts/activate 2>/dev/null || source .venv/bin/activate; } \
  && python -m pip install --upgrade pip \
  && pip install torch --index-url https://download.pytorch.org/whl/cpu \
  && pip install -e ".[dev]" \
  && cp -n .env.example .env)

# --- notification-service (Node/TypeScript)
(cd notification-service && npm ci && cp -n .env.example .env)

# --- web-app (Vue)
(cd web-app && npm ci)
```

Sau khi cài, sửa 2 chỗ trong `.env`:
- `JWT_SECRET` của `api-service/.env` và `notification-service/.env` phải **giống nhau** (≥ 32 ký tự).
- `api-service/.env`: điền `ADMIN_EMAIL`, `ADMIN_PASSWORD` (tài khoản admin được tạo khi chạy lần đầu).
- Gửi email thật: điền `SMTP_*` trong `notification-service/.env`; để trống thì xem email ở http://localhost:8025
  (đặt `SMTP_HOST=localhost`, `SMTP_PORT=1025`).

Model SBERT (~90 MB) không có trên Git; ml-service tự tải vào `ml-service/data/models` ở lần chạy đầu.
Cần huấn luyện lại model thì cài thêm: `pip install -r ml-service/requirements-train.txt` (trong venv).

### Bước 4. Chạy (mỗi lệnh 1 cửa sổ terminal)

```bash
cd api-service && go run ./cmd/server                                   # http://localhost:8000
```

```bash
cd ml-service && { source .venv/Scripts/activate 2>/dev/null || source .venv/bin/activate; } && accheck-ml-serve   # :8001
```

```bash
cd notification-service && npm run dev                                  # :8002
```

```bash
cd web-app && npm run dev                                               # mở http://localhost:5173
```

### Kiểm thử

```bash
(cd api-service && go test ./...)
(cd notification-service && npm test)
(cd ml-service && { source .venv/Scripts/activate 2>/dev/null || source .venv/bin/activate; } && pytest)
(cd web-app && npm run build)
```

---

## Lỗi thường gặp

| Lỗi | Cách xử lý |
|---|---|
| `$'\r': command not found` khi chạy `up.sh` | File bị đổi sang xuống dòng Windows. Chạy `git rm --cached -r . && git reset --hard` để checkout lại theo `.gitattributes`. |
| `python` mở Microsoft Store | Dùng `py -3.12` như trong lệnh trên, hoặc tắt *App execution aliases* của Python trong Settings. |
| `port is already allocated` | Cổng đang bị chiếm. Cách 1: đổi `WEB_PORT`, `API_PORT`… trong `reqsentinel/.env`. Cách 2: tắt chương trình đang dùng cổng 5432 / 5672 / 8000. |
| Build Docker chậm hoặc máy đơ | Docker Desktop → Settings → Resources → Memory đặt 3–4 GB; đóng bớt ứng dụng. `up.sh` tự dừng build khi RAM trống < 700 MB. |
| `ml_service: unavailable` ở `/api/v1/health` | ml-service đang nạp model (1–2 phút lần đầu) hoặc chưa chạy. |
