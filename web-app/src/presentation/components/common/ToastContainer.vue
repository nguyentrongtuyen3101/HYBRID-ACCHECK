<script setup>
import { useI18n } from "vue-i18n";
import { closeToast, toasts } from "../../composables/ui.js";

const { t } = useI18n();

const icons = { success: "✓", error: "!", info: "i" };
</script>

<template>
  <div id="toast-container" aria-live="polite">
    <div v-for="toast in toasts" :key="toast.id" class="toast" :class="[`toast-${toast.type}`, { 'toast-out': toast.leaving }]">
      <span class="toast-icon">{{ icons[toast.type] || "i" }}</span>
      <span class="toast-body">{{ toast.message }}</span>
      <button type="button" class="toast-close" :aria-label="t('common.close')" @click="closeToast(toast.id)">×</button>
    </div>
  </div>
</template>
