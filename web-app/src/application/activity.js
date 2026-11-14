import { reactive } from "vue";

/**
 * Theo dõi các request đang chạy để hiện chỉ báo tải chung cho cả app.
 * - busy: có request chạy quá `showAfter` ms (request nhanh không làm thanh tải nhấp nháy)
 * - slow: có request chạy quá `slowAfter` ms (mạng chậm / máy chủ phản hồi lâu)
 * @param {{ showAfter?: number, slowAfter?: number }} [opts]
 */
export function createActivityTracker({ showAfter = 400, slowAfter = 5000 } = {}) {
  const state = reactive({ busy: false, slow: false });
  let pending = 0;
  let busyTimer;
  let slowTimer;

  function reset() {
    clearTimeout(busyTimer);
    clearTimeout(slowTimer);
    state.busy = false;
    state.slow = false;
  }

  return {
    state,
    start() {
      pending += 1;
      if (pending === 1) {
        busyTimer = setTimeout(() => (state.busy = true), showAfter);
        slowTimer = setTimeout(() => (state.slow = true), slowAfter);
      }
    },
    end() {
      pending = Math.max(0, pending - 1);
      if (pending === 0) reset();
    },
  };
}
