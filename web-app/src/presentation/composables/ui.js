// Toast + hộp thoại xác nhận dùng chung toàn app.
import { reactive } from "vue";

export const toasts = reactive([]);
let nextId = 1;

export function closeToast(id) {
  const t = toasts.find((x) => x.id === id);
  if (!t || t.leaving) return;
  t.leaving = true;
  setTimeout(() => {
    const i = toasts.findIndex((x) => x.id === id);
    if (i >= 0) toasts.splice(i, 1);
  }, 200);
}

export function showToast(message, type = "info", duration = 3200) {
  const id = nextId++;
  toasts.push({ id, message, type, leaving: false });
  if (duration > 0) setTimeout(() => closeToast(id), duration);
}

// title rỗng → hộp thoại dùng tiêu đề mặc định đã dịch (common.confirm)
export const confirmState = reactive({ open: false, title: "", message: "", resolve: null });

export function showConfirm(message, title = "") {
  return new Promise((resolve) => {
    Object.assign(confirmState, { open: true, title, message, resolve });
  });
}

export function closeConfirm(result) {
  const resolve = confirmState.resolve;
  Object.assign(confirmState, { open: false, resolve: null });
  if (resolve) resolve(result);
}
