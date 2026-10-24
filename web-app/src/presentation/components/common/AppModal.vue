<script setup>
defineProps({
  title: { type: String, required: true },
  maxWidth: { type: String, default: "480px" },
});
defineEmits(["close"]);
</script>

<template>
  <div class="modal-backdrop" role="dialog" aria-modal="true">
    <div class="modal" :style="{ maxWidth }">
      <h3>{{ title }}</h3>
      <slot />
      <div class="modal-actions">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>
