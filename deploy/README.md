# ReqSentinel (Hybrid-ACCheck)

Hệ thống AI hỗ trợ Business Analyst phát hiện **xung đột / thiếu sót** trong yêu cầu phân quyền dựa trên
**User Story**, **Acceptance Criteria** và **Permission Matrix**, kèm quy trình làm việc nhóm để xử lý chúng.

> Đồ án tốt nghiệp · Pipeline hybrid: DeBERTa + SBERT + Alignment + ACRG + Symbolic Reasoner

Thư mục `deploy/` chứa mọi thứ để chạy **toàn bộ hệ thống** (docker compose + biến môi trường).
Chi tiết từng service xem README trong thư mục của service đó.
Hướng dẫn sử dụng cho người dùng cuối: [docs/huong-dan-su-dung](../docs/huong-dan-su-dung/README.md).

---

## Kiến trúc

```text
 Trình duyệt ──────────────┐
   │ HTTP          WebSocket│
   ▼                        ▼
┌──────────────┐  ┌───────────────────────┐   email (SMTP)
│ web-app      │  │ notification-service  │ ───────────────▶ người dùng
│ Vue 3 + nginx│  │ Node.js / TypeScript  │
└──────┬───────┘  └───────────▲───────────┘
       │ /api/v1               │ sự kiện (RabbitMQ, topic exchange)
       ▼                       │
┌──────────────┐  outbox ──────┘          ┌──────────────────┐
│ api-service  │ ── POST /v1/analyze ───▶ │ ml-service       │
│ Go           │                          │ Python / FastAPI │
└──────┬───────┘                          └──────────────────┘
       ▼
   PostgreSQL (api-service: schema public · notification-service: schema notification)
```

| Service | Công nghệ | Trách nhiệm | README |
|---------|-----------|-------------|--------|
| `web-app/` | Vue 3 + Vite, nginx | Toàn bộ UI; giữ nguyên CSS gốc | [web-app/README.md](../web-app/README.md) |
| `api-service/` | Go | Nghiệp vụ + dữ liệu: auth, project, thành viên, yêu cầu, phân tích, conflict, báo cáo, admin | [api-service/README.md](../api-service/README.md) |
| `ml-service/` | Python (package `accheck-ml`) | Chỉ suy luận model; train/đánh giá chạy offline bằng CLI | [ml-service/README.md](../ml-service/README.md) |
| `notification-service/` | Node.js / TypeScript | Nhận sự kiện → thông báo in-app (realtime qua Socket.IO) + email | [notification-service/README.md](../notification-service/README.md) |

Cả 3 backend đều theo Clean Architecture: **domain** (luật nghiệp vụ thuần) ← **application/use case**
(chỉ phụ thuộc interface/port) ← **adapter/infrastructure** (DB, HTTP, broker, SMTP…), nối với nhau ở composition root (`main`).

---

## Nghiệp vụ

| Nhóm | Chức năng |
|------|-----------|
| Tài khoản | Đăng ký, đăng nhập (JWT), đổi mật khẩu, **quên mật khẩu qua email**, role hệ thống `admin` / `user` |
| Cộng tác | **Thành viên project** với role `owner` / `editor` / `viewer`, **mời qua email** (người chưa có tài khoản đăng ký từ link mời) |
| Yêu cầu | User story + AC, permission matrix, import Excel, **từ điển thuật ngữ** (role/resource/action + từ đồng nghĩa) |
| Phân tích | Chạy 1 story, **chạy cả project ở nền** (tiến độ realtime), lưu `model_version` của mỗi lần chạy |
| Xử lý conflict | Mỗi conflict là 1 issue sống qua nhiều lần chạy: `open → acknowledged → resolved / false_positive`, giao người xử lý, bình luận, tự đóng khi không còn phát hiện, tự mở lại khi xuất hiện lại |
| Phiên bản | **Tự lưu phiên bản permission matrix** trước mỗi lần phân tích, so sánh phiên bản, khôi phục; **so sánh 2 lần chạy** |
| Báo cáo | Tỉ lệ nhất quán, conflict theo trạng thái / loại / role, **ma trận bao phủ** (quyền Allow không story nào dùng → nghi cấp thừa), **xuất Excel** |
| Thông báo | In-app realtime (chuông thông báo) + email: được mời, có người tham gia, đổi role, bị xoá khỏi project, được giao conflict, conflict đổi trạng thái / có bình luận, job phân tích xong |
| Admin | Thống kê hệ thống, quản lý user, audit log, **xuất dataset** từ đánh giá của BA để train lại model |

Quyền trong project:

| Thao tác | viewer | editor | owner |
|----------|:------:|:------:|:-----:|
| Xem project, kết quả, báo cáo; bình luận conflict | ✓ | ✓ | ✓ |
| Sửa story / PM / từ điển, import, chạy phân tích, xử lý & giao conflict | | ✓ | ✓ |
| Mời / đổi role / xoá thành viên, đổi tên, xoá project | | | ✓ |

Admin hệ thống có quyền owner trên mọi project. Project luôn phải còn ít nhất 1 owner.

---

## Chạy toàn bộ bằng Docker Compose

```bash
cd deploy
cp .env.example .env
# Điền JWT_SECRET (>= 32 ký tự, vd `openssl rand -hex 32`) và ADMIN_PASSWORD
docker compose up --build
```

| URL | Mô tả |
|-----|-------|
| http://localhost:8080 | Web UI |
| http://localhost:8000/api/v1/health | API Go (trạng thái DB + ML service) |
| http://localhost:8025 | Mailpit — xem email mời / reset mật khẩu khi chạy dev |
| http://localhost:15673 | RabbitMQ management (guest / guest) |

- Email mặc định đi vào Mailpit. Muốn gửi email thật thì điền `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `MAIL_FROM` trong `.env`.
- Cổng publish ra máy đổi được bằng `WEB_PORT`, `API_PORT`, `POSTGRES_PORT`, `RABBITMQ_UI_PORT`, `MAILPIT_UI_PORT`.
- Lần đầu ML service tải SBERT (~90MB) vào `ml-service/data/models`; checkpoint DeBERTa đặt ở `ml-service/data/models/deberta-extractor/`.

Chạy từng service khi dev: xem README của service (cần Postgres + RabbitMQ chạy sẵn).

---

## Luồng sự kiện

1. api-service ghi thay đổi dữ liệu **và** sự kiện vào bảng `outbox_events` trong cùng 1 transaction.
2. Relay trong api-service đẩy sự kiện lên exchange `reqsentinel.events` (topic, publisher confirm).
   Sự kiện phát ra khi chưa có consumer nào được giữ ở `reqsentinel.events.unrouted`.
3. notification-service đọc queue `notification-service.events`, áp luật phân phối, lưu thông báo, đẩy realtime,
   gửi email. Xử lý idempotent theo `event.id`; lỗi tạm thời thì retry có backoff, quá số lần thì vào DLQ.

| Sự kiện | In-app | Email |
|---------|--------|-------|
| `project.member_invited` | người được mời (nếu đã có tài khoản) | ✓ link chấp nhận lời mời |
| `project.member_joined` | người mời | |
| `project.member_role_changed`, `project.member_removed` | thành viên bị đổi role / bị xoá | |
| `conflict.assigned` | người được giao | |
| `conflict.status_changed`, `conflict.commented` | người được giao + người đã tham gia | |
| `analysis.job_progress` | chỉ realtime (không lưu) | |
| `analysis.job_finished` | người chạy job | |
| `auth.password_reset_requested` | | ✓ link đặt lại mật khẩu |

---

## Dùng Supabase làm database

1. Tạo project trên Supabase, chọn region gần nhất (vd Singapore).
2. **Connect** → mục **Session pooler** → copy URI (`postgres://postgres.<ref>:<password>@aws-0-<region>.pooler.supabase.com:5432/postgres`),
   thêm `?sslmode=require`. Không dùng *Direct connection* (chỉ IPv6) và *Transaction pooler* cổng 6543
   (không hỗ trợ prepared statement và advisory lock mà migration dùng).
   Mật khẩu có ký tự đặc biệt thì phải URL-encode (vd `@` → `%40`).
3. Đặt chuỗi đó vào `DATABASE_URL` của **cả** `api-service/.env` và `notification-service/.env`.
4. Chạy api-service: migration tự tạo bảng và tự bật Row Level Security cho mọi bảng trong `public`, nên
   Data API của Supabase (role `anon` / `authenticated`) không đọc được dữ liệu. Có thể tắt hẳn Data API ở
   *Project Settings → Data API* vì app không dùng tới.

## Nâng cấp từ bản cũ

api-service dùng lại DB cũ: migration chạy tự động khi khởi động và chỉ **thêm** bảng/cột (không xoá dữ liệu).
Project cũ chưa có chủ được gán cho admin đầu tiên; chủ project hiện có trở thành `owner`.
Nên backup DB (hoặc tạo branch trên Neon) trước lần chạy đầu với DB production.

---

## Troubleshooting

| Lỗi | Cách xử lý |
|-----|------------|
| Header hiện `ML offline` / phân tích trả 503 | ML service chưa chạy hoặc đang load model; kiểm tra `ML_SERVICE_URL`, `ML_SERVICE_TOKEN` trùng `INTERNAL_TOKEN` |
| Không nhận thông báo / email | Kiểm tra `RABBITMQ_URL` ở api-service và notification-service; sự kiện chưa gửi nằm trong bảng `outbox_events` |
| Socket không kết nối | `JWT_SECRET` của notification-service phải trùng api-service |
| API không start: `JWT_SECRET ...` | Đặt `JWT_SECRET` ≥ 32 ký tự |
| Không có tài khoản admin | Đặt `ADMIN_EMAIL` + `ADMIN_PASSWORD` rồi restart api-service |

Đồ án học thuật — không dùng model/data nhạy cảm production khi chưa audit bảo mật.
