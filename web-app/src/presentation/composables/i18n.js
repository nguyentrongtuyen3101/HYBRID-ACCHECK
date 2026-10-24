// Tiện ích i18n dùng chung cho các component.
import { useI18n } from "vue-i18n";
import { formatDateTime, formatPercent, timeAgo } from "../../domain/format.js";

/** Định dạng ngày giờ / thời gian tương đối theo ngôn ngữ đang chọn. */
export function useFormat() {
  const { locale } = useI18n();
  return {
    dateTime: (iso) => formatDateTime(iso, locale.value),
    ago: (iso) => timeAgo(iso, locale.value),
    percent: (v) => formatPercent(v, locale.value),
  };
}

/** Dịch lỗi từ API theo `code` (+ `params`); không có bản dịch thì dùng câu gốc. */
export function useErrorText() {
  const { t, te } = useI18n();
  return (err) => {
    if (err?.code && te(`errors.${err.code}`)) return t(`errors.${err.code}`, err.params || {});
    return err?.message || t("errors.unknown");
  };
}

/** Nhãn hiển thị của giá trị enum, vd label("issueStatus", "open"). */
export function useEnumLabel() {
  const { t, te } = useI18n();
  return (group, value) => (value && te(`enums.${group}.${value}`) ? t(`enums.${group}.${value}`) : value || "");
}
