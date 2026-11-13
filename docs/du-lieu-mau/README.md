# Dữ liệu mẫu

Ba file Excel để thử chức năng **Nhập Excel** (tab User Story của một dự án). File được tạo từ chính mẫu `Tải mẫu Excel` của hệ thống: sheet `UserStories` (content, acceptance_criteria — nhiều AC cách nhau bằng `;`) và sheet `PermissionMatrix` (role, action, resource, effect, scope, condition).

Cách dùng: tạo dự án mới → tab **User Story** → **Nhập Excel** → chọn file → tab **Chạy phân tích** → **Phân tích tất cả user story**. Mỗi file được thiết kế để ra đủ 5 loại xung đột và vài story nhất quán; kết quả dưới đây là kết quả thực tế khi chạy với mô hình `rule-based+all-MiniLM-L6-v2`.

## thu-vien-sach.xlsx — Thư viện sách điện tử

| User story | Kết quả mong đợi |
|---|---|
| As a member, I want to borrow books so that I can read them at home. | Nhất quán |
| As a librarian, I want to add new books to the catalog. | Nhất quán |
| As a member, I want to view my own borrowing history. | Xung đột phạm vi |
| As a guest, I want to borrow books. | Xung đột hiệu lực |
| As a librarian, I want to remove damaged books from the catalog. | Xung đột hành động |
| As a member, I want to renew borrowed books. | Xung đột điều kiện |
| As an administrator, I want to export the monthly lending statistics. | Thiếu quyền |

## dao-tao-truc-tuyen.xlsx — Hệ thống đào tạo trực tuyến (LMS)

| User story | Kết quả mong đợi |
|---|---|
| As a teacher, I want to create courses so that students can enroll. | Nhất quán |
| As a student, I want to submit my own assignments. | Nhất quán |
| As a student, I want to view my own grades. | Xung đột phạm vi |
| As a student, I want to edit course content. | Xung đột hiệu lực |
| As a teacher, I want to delete courses. | Xung đột hành động |
| As a teacher, I want to grade assignments. | Xung đột điều kiện |
| As a parent, I want to pay the tuition fees. | Thiếu quyền |

## kho-van-giao-hang.xlsx — Quản lý kho vận và giao hàng

| User story | Kết quả mong đợi |
|---|---|
| As a warehouse staff, I want to record incoming goods so that the stock is up to date. | Nhất quán |
| As a driver, I want to view assigned deliveries. | Xung đột phạm vi |
| As a driver, I want to update my own deliveries. | Nhất quán |
| As a driver, I want to adjust the inventory. | Xung đột hiệu lực |
| As a warehouse manager, I want to approve stock adjustments. | Xung đột điều kiện |
| As a warehouse manager, I want to delete suppliers. | Xung đột hành động |
| As an accountant, I want to export the delivery cost report. | Thiếu quyền |

## Dự án đã seed sẵn trong DB

Bốn dự án dưới đây đã có sẵn user story, ma trận phân quyền, từ điển thuật ngữ và phiên bản ma trận v1 (chủ sở hữu: tài khoản Nguyễn Trọng Tuyền). Chỉ cần mở dự án và bấm **Phân tích tất cả user story**.

| Dự án | User story | Dòng ma trận | Thuật ngữ |
|---|---|---|---|
| Website bán hàng – Phân hệ quản trị | 8 | 10 | 5 |
| CMS Tin tức nội bộ | 7 | 6 | 3 |
| Ứng dụng đặt lịch khám | 6 | 5 | 3 |
| Hệ thống quản lý nhân sự (HRM) | 6 | 6 | 3 |
