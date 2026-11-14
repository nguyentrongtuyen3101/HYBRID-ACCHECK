// Luật thuần cho job phân tích nền (không phụ thuộc Vue / HTTP).

export const isJobActive = (job) => job?.status === "queued" || job?.status === "running";
export const isJobDone = (status) => status === "completed" || status === "failed";

/**
 * Áp tín hiệu tiến độ realtime vào job đang hiển thị.
 * Tín hiệu có thể đến sai thứ tự (notification-service xử lý song song nhiều sự kiện), nên:
 * - không đưa job đã kết thúc quay lại trạng thái đang chạy;
 * - bỏ qua tín hiệu cũ hơn tiến độ đang hiển thị.
 * Trả về true nếu đã cập nhật.
 */
export function applyJobProgress(job, signal) {
  if (!job || !isJobActive(job)) return false;
  if (!isJobDone(signal.status) && (signal.processed ?? 0) < (job.processed ?? 0)) return false;
  Object.assign(job, {
    status: signal.status,
    processed: signal.processed,
    failed: signal.failed,
    total: signal.total,
  });
  return true;
}
