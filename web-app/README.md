# web-app (Vue 3 + Vite)

Giao diện ReqSentinel. `src/assets/app.css` là CSS gốc giữ nguyên; style cho màn hình mới nằm ở `extra.css`.

## Cấu trúc

```text
src/
  domain/          luật thuần cho UI: quyền theo role, vòng đời conflict, định dạng, ngôn ngữ/theme hợp lệ
  infrastructure/  http client, lưu phiên (localStorage), socket realtime, module API theo tài nguyên, browser/ (lang, theme của <html>)
  application/     session, notification center, preferences (ngôn ngữ + theme), container.js (composition root, provide/inject)
  presentation/    views/, components/{common,project,admin}/, composables/{ui,i18n}.js, i18n/locales/{vi,en,ja}.json
  router.js, main.js, App.vue
```

Presentation chỉ lấy dependency qua `useContainer()`; không gọi `fetch` hay import infrastructure trực tiếp.

## Ngôn ngữ và theme

- Không viết chữ hiển thị trực tiếp trong component: mọi text đi qua `t("nhóm.key")` (vue-i18n), bản dịch ở `src/presentation/i18n/locales/`.
  Ba file phải có cùng bộ key; giá trị enum hiển thị qua `useEnumLabel()` (`enums.<nhóm>.<giá trị>`), lỗi API qua `useErrorText()` (`errors.<code>`).
  Ký tự `@` trong bản dịch phải viết `{'@'}`.
- Ngôn ngữ/theme ưu tiên: profile của user → lựa chọn lưu trên máy → trình duyệt/hệ điều hành. Đổi khi đã đăng nhập thì lưu lên profile.
- Theme tối: `src/assets/theme-dark.css` ghi đè biến màu khi `<html data-theme="dark">`; màu mới trong `extra.css` dùng biến (`--surface`, `--hover`, ...) để tự đổi theo theme.
- Dropdown: dùng `components/common/CustomSelect.vue` (menu teleport ra `body` nên không bị cắt trong bảng/modal), không dùng `<select>`.

## Màn hình

- Đăng nhập / đăng ký, quên & đặt lại mật khẩu, trang nhận lời mời (`/invitations/:token`)
- Projects; trong project có các tab: User Stories, Permission Matrix (kèm phiên bản), Dictionary, Run analysis
  (1 story hoặc cả project, tiến độ realtime), Conflicts (bảng xử lý + chi tiết), Runs (so sánh 2 lần chạy),
  Report (bao phủ, xuất Excel), Members (mời qua email), Settings
- History, Run detail, Admin (thống kê, user, audit log, xuất dataset)
- Profile (`/profile`): thông tin cá nhân, ngôn ngữ, theme, đổi mật khẩu
- Chuông thông báo realtime trên header

## Chạy

```bash
npm install
npm run dev     # http://localhost:5173; proxy /api → :8000, /api/notifications + /socket.io → :8002
npm run build   # dist/ (production phục vụ bằng nginx, xem nginx.conf và Dockerfile)
```

Target proxy khi dev đổi bằng `API_PROXY_TARGET`, `NOTIFY_PROXY_TARGET`.
Chỉ đặt `VITE_API_BASE`, `VITE_NOTIFY_API_BASE`, `VITE_NOTIFY_URL` khi backend ở domain khác (nhớ bật CORS).
