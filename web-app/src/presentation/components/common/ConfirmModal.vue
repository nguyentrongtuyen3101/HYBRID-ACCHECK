<script setup>
import { nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { closeConfirm, confirmState } from "../../composables/ui.js";

const { t } = useI18n();

const confirmBtn = ref(null);
watch(
  () => confirmState.open,
  async (open) => {
    if (open) {
      await nextTick();
      confirmBtn.value?.focus();
    }
  }
);
</script>

<template>
  <div id="modal-overlay" :class="{ hidden: !confirmState.open }" role="dialog" aria-modal="true" @click.self="closeConfirm(false)">
    <div class="modal">
      <h3>{{ confirmState.title || t("common.confirm") }}</h3>
      <p>{{ confirmState.message }}</p>
      <div class="modal-actions">
        <button type="button" class="btn btn-secondary" @click="closeConfirm(false)">{{ t("common.cancel") }}</button>
        <button ref="confirmBtn" type="button" class="btn btn-primary" @click="closeConfirm(true)">{{ t("common.confirm") }}</button>
      </div>
    </div>
  </div>
</template>
