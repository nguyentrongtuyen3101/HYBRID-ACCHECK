# notification-service (Node.js / TypeScript)

Nhận sự kiện nghiệp vụ từ api-service qua RabbitMQ, rồi:

- lưu thông báo in-app (schema `notification`, không đọc bảng của service khác),
- đẩy realtime qua **Socket.IO** (xác thực bằng access token của api-service),
- gửi **email** (mời thành viên, đặt lại mật khẩu) qua SMTP.

## Cấu trúc (Clean Architecture)

```text
src/
  domain/            hợp đồng sự kiện, thông báo, delivery-policy.ts (sự kiện → ai nhận gì, qua kênh nào)
  application/       ports.ts (interface) + use case: HandleEventUseCase, NotificationInbox
  infrastructure/    postgres/ rabbitmq/ mail/ realtime/ http/ auth/
  main.ts            composition root
test/                test luật phân phối + use case bằng fake (không cần DB/SMTP/socket)
```

Thêm loại sự kiện mới: thêm 1 rule vào `deliveryRules` trong `src/domain/delivery-policy.ts`
(không phải sửa consumer hay use case).

## Độ tin cậy

- Idempotent theo `event.id` (bảng `processed_events`); thông báo trùng `(event_id, user_id)` không bị tạo lại.
- Lỗi tạm thời (SMTP, DB…) → retry có backoff (`EVENT_MAX_RETRIES`), quá số lần → `notification-service.events.dlq`.
- Message hỏng → DLQ ngay.
- Đọc cả queue `reqsentinel.events.unrouted` để không mất sự kiện phát ra trước khi service này chạy lần đầu.

## Chạy

```bash
npm install
cp .env.example .env   # DATABASE_URL, RABBITMQ_URL, JWT_SECRET (trùng api-service), SMTP_*
npm run dev            # http://localhost:8002
npm test
npm run build && npm start
```

`SMTP_HOST` để trống thì email chỉ được ghi log. Khi dev có thể dùng Mailpit (`SMTP_HOST=localhost`, `SMTP_PORT=1025`).

Gửi qua Gmail: bật xác minh 2 bước, tạo App Password tại https://myaccount.google.com/apppasswords, rồi đặt
`SMTP_HOST=smtp.gmail.com`, `SMTP_PORT=587`, `SMTP_USER=<gmail>`, `SMTP_PASSWORD=<app password>`.
Kiểm tra cấu hình: `npm run mail:test -- you@example.com [vi|en|ja]` (đăng nhập SMTP rồi gửi 1 email mẫu).

Email gửi theo `locale` có trong event (api-service lấy từ profile người nhận, mặc định `vi`); bản dịch ở `src/infrastructure/mail/templates.ts`.
Thông báo in-app lưu `type` + `params` (không lưu câu đã dịch) để web-app tự hiển thị theo ngôn ngữ đang chọn; `title`/`body` tiếng Anh chỉ là bản dự phòng.

## API

| Endpoint | Mô tả |
|----------|-------|
| `GET /health` | trạng thái DB + RabbitMQ |
| `GET /api/notifications?limit=&before=&unread=true` | danh sách + `unread_count` |
| `GET /api/notifications/unread-count` | số chưa đọc |
| `POST /api/notifications/{id}/read` | đánh dấu đã đọc |
| `POST /api/notifications/read-all` | đánh dấu tất cả đã đọc |
| Socket.IO `/socket.io` (`auth: { token }`) | sự kiện `notification` và `job:progress` |
