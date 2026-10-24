<script setup>
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../../application/container.js";
import { useErrorText, useFormat } from "../../composables/i18n.js";
import { showToast } from "../../composables/ui.js";

const { t, te } = useI18n();
const router = useRouter();
const { notifications } = useContainer();
const { ago } = useFormat();
const errorText = useErrorText();
const state = notifications.state;
const open = ref(false);

/** Dựng câu thông báo theo ngôn ngữ hiện tại từ type + params; thiếu bản dịch thì dùng câu gốc. */
function text(n, part) {
  const key = `notifications.${n.type}.${part}`;
  if (!te(key)) return part === "title" ? n.title : n.body;
  const p = { ...(n.params || {}) };
  for (const [field, group] of [["role", "projectRole"], ["conflict_type", "conflictType"], ["to", "issueStatus"], ["from", "issueStatus"]]) {
    if (p[field] && te(`enums.${group}.${p[field]}`)) p[field] = t(`enums.${group}.${p[field]}`);
  }
  if (n.type === "analysis.job_finished" && p.status === "failed" && part === "title") return t("notifications.analysis.job_failed.title", p);
  if (n.type === "analysis.job_finished" && p.status === "failed" && part === "body") return p.error || "";
  return t(key, p);
}

const toggle = () => (open.value = !open.value);
// Đóng khi click ra ngoài (pha capture để mở menu tài khoản cũng đóng menu này)
const root = ref(null);
const onOutside = (e) => root.value && !root.value.contains(e.target) && (open.value = false);

async function openItem(n) {
  open.value = false;
  try {
    await notifications.markRead(n);
  } catch {
    /* không chặn điều hướng */
  }
  if (n.link) router.push(n.link);
}

async function markAll() {
  try {
    await notifications.markAllRead();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

// Có thông báo mới khi đang mở app → toast nhỏ
const unsubscribe = notifications.onSignal("notification", (n) => showToast(text(n, "title"), "info"));
onMounted(() => document.addEventListener("click", onOutside, true));
onBeforeUnmount(() => {
  unsubscribe();
  document.removeEventListener("click", onOutside, true);
});
</script>

<template>
  <div ref="root" class="bell">
    <button type="button" class="bell-btn" :aria-label="t('notifications.ariaLabel', { count: state.unread })" @click="toggle">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
        <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
      </svg>
      <span v-if="state.unread" class="bell-count">{{ state.unread > 99 ? "99+" : state.unread }}</span>
    </button>
    <div v-if="open" class="bell-menu">
      <div class="row-between bell-head">
        <strong>{{ t("notifications.title") }}</strong>
        <button v-if="state.unread" type="button" class="link-btn" @click="markAll">{{ t("notifications.markAllRead") }}</button>
      </div>
      <div v-if="!state.items.length" class="empty" style="margin: 10px">{{ t("notifications.empty") }}</div>
      <button
        v-for="n in state.items"
        :key="n.id"
        type="button"
        class="bell-item"
        :class="{ unread: !n.read_at }"
        @click="openItem(n)"
      >
        <span class="bell-title">{{ text(n, "title") }}</span>
        <span v-if="text(n, 'body')" class="bell-body">{{ text(n, "body") }}</span>
        <span class="bell-time">{{ ago(n.created_at) }}</span>
      </button>
    </div>
  </div>
</template>
