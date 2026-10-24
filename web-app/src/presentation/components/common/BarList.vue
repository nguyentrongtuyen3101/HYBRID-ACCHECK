<script setup>
import { computed } from "vue";

const props = defineProps({
  entries: { type: Array, required: true }, // [[label, value], ...]
  emptyText: { type: String, default: "No data." },
});
const max = computed(() => Math.max(1, ...props.entries.map(([, v]) => v)));
</script>

<template>
  <div class="stack">
    <div v-if="!entries.length" class="muted">{{ emptyText }}</div>
    <div v-for="[label, value] in entries" :key="label" class="bar-row">
      <span>{{ label }}</span>
      <div class="bar-track"><div class="bar-fill" :style="{ width: `${(value / max) * 100}%` }"></div></div>
      <span class="muted">{{ value }}</span>
    </div>
  </div>
</template>
