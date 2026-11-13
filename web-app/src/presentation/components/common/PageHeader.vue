<script setup>
// Đầu trang: nhãn cam + tiêu đề lớn 2 tông (phần dim màu xám) + mô tả; slot mặc định = nút hành động, slot "meta" = dòng trên tiêu đề.
defineProps({
  eyebrow: { type: String, default: "" },
  title: { type: String, required: true },
  dim: { type: String, default: "" },
  subtitle: { type: String, default: "" },
});
</script>

<template>
  <header class="page-head">
    <div>
      <div v-if="$slots.meta" class="page-meta"><slot name="meta" /></div>
      <p v-else-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
      <h1>
        {{ title }}<template v-if="dim"><br /><span class="dim">{{ dim }}</span></template>
      </h1>
      <p v-if="subtitle" class="page-sub">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.default" class="page-actions"><slot /></div>
  </header>
</template>
