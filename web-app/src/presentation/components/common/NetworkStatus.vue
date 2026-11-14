<script setup>
// Chỉ báo mạng chung cho cả app: thanh tải mảnh ở mép trên khi request chạy lâu,
// nhãn "mạng chậm" khi quá vài giây, dải báo khi trình duyệt mất mạng / có mạng lại.
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";

const { t } = useI18n();
const { activity } = useContainer();

const online = ref(typeof navigator === "undefined" ? true : navigator.onLine);
const backOnline = ref(false);
let backTimer;

function goOffline() {
  online.value = false;
  backOnline.value = false;
}
function goOnline() {
  online.value = true;
  backOnline.value = true;
  clearTimeout(backTimer);
  backTimer = setTimeout(() => (backOnline.value = false), 3000);
}
onMounted(() => {
  window.addEventListener("offline", goOffline);
  window.addEventListener("online", goOnline);
});
onBeforeUnmount(() => {
  window.removeEventListener("offline", goOffline);
  window.removeEventListener("online", goOnline);
  clearTimeout(backTimer);
});
</script>

<template>
  <div v-if="activity.state.busy" class="net-bar" role="progressbar" :aria-label="t('network.loading')">
    <span class="net-bar-fill"></span>
  </div>
  <Transition name="net-fade">
    <div v-if="!online" class="net-banner net-banner-off" role="alert">
      <span class="net-dot"></span>{{ t("network.offline") }}
    </div>
    <div v-else-if="backOnline" class="net-banner net-banner-on" role="status">
      <span class="net-dot"></span>{{ t("network.backOnline") }}
    </div>
    <div v-else-if="activity.state.slow" class="net-pill" role="status">
      <span class="net-spinner" aria-hidden="true"></span>{{ t("network.slow") }}
    </div>
  </Transition>
</template>
