# Đề xuất nghiệp vụ cho ReqSentinel

Hiện tại hệ thống có: project → user story/AC + permission matrix → chạy phân tích → xem lịch sử,
cộng với auth (admin/user), trang admin và audit log. Dưới đây là các nghiệp vụ nên làm tiếp,
xếp theo thứ tự ưu tiên (giá trị / công sức).

## Về role: giữ 2 role hệ thống, thêm role theo project

Không nên thêm nhiều role hệ thống. Nhu cầu thật của BA là **làm việc nhóm trong 1 project**, nên tách 2 tầng:

| Tầng | Role | Ý nghĩa |
|------|------|---------|
| Hệ thống (`users.role`) | `admin`, `user` | Như hiện tại |
| Project (`project_members.role`) | `owner`, `editor`, `viewer` | Owner: quản lý thành viên, xoá project. Editor: sửa US/PM, chạy phân tích. Viewer: chỉ xem kết quả (vd PO, khách hàng, reviewer). |

Có thể thêm role hệ thống `auditor` (chỉ đọc mọi project + audit log) nếu cần cho QA/kiểm toán.

## Giai đoạn 1 — nên làm ngay

1. **Thành viên project (collaboration)**
   Bảng `project_members(project_id, user_id, role, invited_by)`; mời bằng email; thay điều kiện
   "chủ project" ở `loadProject` bằng kiểm tra membership + role.

2. **Quy trình xử lý conflict**
   Mỗi conflict có trạng thái `open → acknowledged → resolved` hoặc `false_positive`, kèm người xử lý,
   comment, thời điểm. Màn hình "Conflict board" theo project.
   Đây là phần giá trị nhất: biến công cụ "báo lỗi" thành công cụ "quản lý lỗi".
   Conflict bị đánh `false_positive` chính là **dữ liệu có nhãn** để train lại model (xem mục 6).

3. **Chạy phân tích toàn bộ project + báo cáo**
   Nút "Analyze all stories"; báo cáo theo project: % nhất quán, conflict theo loại / theo role /
   theo resource; xuất Excel/PDF để gửi khách hàng hoặc đính kèm tài liệu đặc tả.

4. **Version Permission Matrix & so sánh run**
   Snapshot PM mỗi lần chạy (đã lưu trong `raw_result`, chỉ cần tách bảng), diff 2 phiên bản PM,
   so sánh 2 lần chạy: conflict nào mới xuất hiện / đã hết.

## Giai đoạn 2 — tăng chất lượng phân tích

5. **Từ điển Role / Resource / Action theo project**
   Danh sách chuẩn + từ đồng nghĩa ("Sales staff" = "Sales employee"). Dùng để chuẩn hoá trước
   khi gửi sang ML (tăng độ chính xác alignment) và cảnh báo role lạ không có trong từ điển.

6. **Vòng lặp feedback → dataset → train lại (admin)**
   Admin xuất dữ liệu đã được BA xác nhận (resolved / false positive) thành dataset; chạy
   `accheck-ml-train`; lưu `model_version` vào mỗi run để biết kết quả do model nào sinh ra.

7. **Ma trận bao phủ (coverage)**
   Heatmap Role × Resource: dòng PM nào không được user story nào nhắc tới → cảnh báo
   **"Unused / over-granted permission"** (chiều ngược lại của Missing Permission — rất có giá trị
   về bảo mật: quyền cấp thừa).

8. **Chạy phân tích bất đồng bộ (job queue)**
   Bắt buộc khi có mục 3. Go đẩy job vào Redis/RabbitMQ, worker gọi ML, UI xem tiến độ.
   Giới hạn số job đồng thời để ML service không quá tải.

## Giai đoạn 3 — vận hành & tích hợp

9. **Quên mật khẩu qua email, refresh token, thu hồi phiên, 2FA cho admin.**
10. **Hạn mức (quota)** số lần phân tích / ngày theo user — trang admin chỉnh được.
11. **API key cá nhân** để tích hợp CI hoặc script; **import từ Jira / Azure DevOps** thay vì Excel.
12. **Thông báo** (email / Slack) khi lần chạy phát hiện conflict, khi được mời vào project.
13. **Thùng rác**: xoá mềm project / user story, admin khôi phục được.
14. **Cấu hình hệ thống trong trang admin**: ngưỡng alignment, bật/tắt đăng ký… lưu DB thay vì biến môi trường.

## Thứ tự gợi ý

1 → 2 → 3 là bộ ba nên làm cùng nhau (cộng tác + xử lý conflict + báo cáo) — đủ để demo một quy trình
nghiệp vụ hoàn chỉnh: BA nhập yêu cầu → hệ thống phát hiện → nhóm xử lý → xuất báo cáo.
Mục 6 và 7 là điểm cộng về mặt học thuật (cải thiện model từ phản hồi người dùng, phát hiện cấp quyền thừa).
