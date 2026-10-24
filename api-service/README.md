# api-service (Go)

Toàn bộ nghiệp vụ và dữ liệu của ReqSentinel: auth, project & thành viên, yêu cầu, phân tích (gọi ml-service),
xử lý conflict, báo cáo, admin. Phát sự kiện sang notification-service qua RabbitMQ (transactional outbox).

## Cấu trúc (Clean Architecture)

```text
cmd/server/main.go        composition root: nối adapter vào use case, chạy HTTP + worker + relay
internal/
  domain/                 entity + luật nghiệp vụ thuần (chỉ dùng thư viện chuẩn)
                          vd: quyền theo role, vòng đời conflict, đồng bộ issue, diff PM, từ điển, bao phủ
  app/                    use case; ports.go khai báo interface (repository, analyzer, publisher...)
    auth/ users/ projects/ requirements/ analysis/ conflicts/ reports/ admin/
  adapter/                hiện thực port
    postgres/             repository (pgx), migrations/*.sql, outbox
    http/                 REST /api/v1: parse request → gọi use case → DTO
    mlclient/             gọi ml-service
    rabbitmq/             relay outbox → exchange reqsentinel.events
    excel/                template, import, báo cáo Excel
    security/             bcrypt, JWT, token mời/reset, giới hạn đăng nhập sai
    worker/               chạy job phân tích nền
  config/
```

Hướng phụ thuộc: `adapter → app → domain`. Use case không import adapter; HTTP handler không chứa logic nghiệp vụ.

## Chạy dev

```bash
cp .env.example .env      # DATABASE_URL, JWT_SECRET, ADMIN_EMAIL, ADMIN_PASSWORD, RABBITMQ_URL, WEB_URL
go run ./cmd/server       # tự chạy migration khi start
go test ./...
```

Biến môi trường chính (xem `.env.example`):

| Biến | Ý nghĩa |
|------|---------|
| `DATABASE_URL` | Postgres (chấp nhận cả dạng `postgresql+asyncpg://` cũ) |
| `JWT_SECRET` | ≥ 32 ký tự; notification-service dùng cùng giá trị |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | tạo admin đầu tiên nếu DB chưa có admin |
| `ML_SERVICE_URL` / `ML_SERVICE_TOKEN` | ml-service và token nội bộ |
| `RABBITMQ_URL` | để trống = sự kiện chỉ nằm trong `outbox_events` |
| `WEB_URL` | gốc link trong email mời / reset mật khẩu |
| `JOB_WORKERS` | số worker chạy job phân tích cả project |
| `ALLOW_SIGNUP`, `TOKEN_TTL`, `TRUST_PROXY`, `CORS_ALLOWED_ORIGINS` | tuỳ chọn |

## API (`/api/v1`)

| Nhóm | Endpoint |
|------|----------|
| Public | `GET /health`, `POST /auth/register`, `POST /auth/login`, `POST /auth/forgot-password`, `POST /auth/reset-password`, `GET /invitations/{token}`, `GET /projects/template/excel` |
| Tài khoản | `GET/PATCH /auth/me`, `POST /auth/change-password`, `POST /invitations/{token}/accept` |
| Project | `GET/POST /projects`, `GET/PATCH/DELETE /projects/{id}` |
| Thành viên | `GET /projects/{id}/members`, `PATCH/DELETE /projects/{id}/members/{userID}`, `GET/POST /projects/{id}/invitations`, `DELETE /projects/{id}/invitations/{invitationID}` |
| Yêu cầu | `POST/PATCH/DELETE /projects/{id}/user-stories[/{storyID}]`, `POST/PATCH/DELETE /projects/{id}/permission-matrix[/{rowID}]`, `POST /projects/{id}/import/excel` |
| Từ điển | `GET/POST /projects/{id}/terms`, `PUT/DELETE /projects/{id}/terms/{termID}`, `GET /projects/{id}/terms/suggestions` |
| Phiên bản PM | `GET/POST /projects/{id}/pm-versions`, `GET /projects/{id}/pm-versions/diff?from=&to=`, `POST /projects/{id}/pm-versions/{versionID}/restore` |
| Phân tích | `POST /projects/{id}/check`, `GET /projects/{id}/runs`, `GET /projects/{id}/runs/compare?base=&target=`, `GET/POST /projects/{id}/analysis-jobs`, `GET /projects/{id}/analysis-jobs/{jobID}`, `POST /check`, `GET /check/history[/{runID}]` |
| Conflict | `GET /projects/{id}/conflicts`, `GET/PATCH /projects/{id}/conflicts/{issueID}`, `POST /projects/{id}/conflicts/{issueID}/comments` |
| Báo cáo | `GET /projects/{id}/report`, `GET /projects/{id}/report/excel`, `GET /projects/{id}/coverage` |
| Admin | `GET /admin/stats`, `GET/POST /admin/users`, `PATCH /admin/users/{userID}`, `POST /admin/users/{userID}/reset-password`, `GET /admin/audit-logs`, `GET /admin/dataset/export` |

Lỗi luôn trả dạng `{"detail": "...", "code": "password_length", "params": {"min": 8, "max": 128}}`:
`detail` là câu tiếng Anh để debug, FE dịch theo `code` (+ `params`) — thêm code mới thì thêm key `errors.<code>` vào 3 file locale của web-app.
HTTP status: 422 dữ liệu không hợp lệ, 403 không đủ quyền, 404 không tồn tại hoặc không phải thành viên, 409 xung đột trạng thái, 503 ml-service không sẵn sàng.

`PATCH /auth/me` nhận một phần của: `full_name`, `job_title`, `organization`, `phone`, `bio` (chuỗi rỗng = xoá), `locale` (`vi|en|ja`), `theme` (`light|dark`) — `null` để bỏ đặt locale/theme.
Email mời / đặt lại mật khẩu gửi theo `locale` của người nhận (chưa đặt thì `vi`).

## Migration

`internal/adapter/postgres/migrations/*.sql`, chạy tự động khi khởi động (có khoá advisory nên nhiều instance start cùng lúc vẫn an toàn).
`0001` khớp schema SQLAlchemy cũ (`IF NOT EXISTS`), `0002` users + audit, `0003` thành viên, lời mời, từ điển, phiên bản PM, job, conflict issue, outbox,
`0004` chuyển mọi cột trạng thái/loại sang Postgres ENUM (`user_role`, `user_status`, `project_role`, `invitation_status`, `term_kind`,
`job_status`, `issue_status`, `issue_activity_kind`, `permission_effect`, `conflict_type`, `artifact_source`, `ui_locale`, `ui_theme`) và thêm cột profile cho users.
Sau migrate, RLS được bật trên mọi bảng `public` (chặn Data API của Supabase; api-service kết nối bằng role owner nên không bị ảnh hưởng).

Thêm giá trị enum: `ALTER TYPE ... ADD VALUE` trong migration mới, rồi cập nhật hằng số tương ứng trong `internal/domain`.

Giới hạn hiện tại: khi khởi động, job đang `running` bị đánh dấu `failed` (giả định 1 instance chạy worker);
bộ đếm đăng nhập sai lưu in-memory (chạy nhiều instance thì chuyển sang Redis).
