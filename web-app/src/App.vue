<script setup>
import { onBeforeUnmount, onMounted, watch } from "vue";
import { useRoute } from "vue-router";
import { useContainer } from "./application/container.js";
import { closeConfirm, confirmState } from "./presentation/composables/ui.js";
import AppHeader from "./presentation/components/common/AppHeader.vue";
import ConfirmModal from "./presentation/components/common/ConfirmModal.vue";
import ToastContainer from "./presentation/components/common/ToastContainer.vue";

const route = useRoute();
const { session, notifications, preferences } = useContainer();

// Đăng nhập → đồng bộ profile (ngôn ngữ, theme) + mở kết nối thông báo realtime; đăng xuất → đóng
watch(
  () => session.state.token,
  async (token) => {
    if (token && session.isLoggedIn()) {
      notifications.start(token);
      await session.refresh();
      preferences.attachUser(session.user.value, (patch) => session.updateProfile(patch));
    } else {
      notifications.stop();
      preferences.detachUser();
    }
  },
  { immediate: true }
);

function onKeydown(e) {
  if (e.key === "Escape" && confirmState.open) closeConfirm(false);
}
onMounted(() => document.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => {
  document.removeEventListener("keydown", onKeydown);
  notifications.stop();
});
</script>

<template>
  <RouterView v-if="route.meta.public" :key="route.fullPath" />
  <template v-else>
    <AppHeader />
    <div class="container">
      <RouterView :key="route.path" />
    </div>
  </template>
  <ToastContainer />
  <ConfirmModal />
</template>
