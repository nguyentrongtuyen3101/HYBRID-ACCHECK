<script setup>
// Wordmark: "Re q [S] entinel" — chữ S tự vẽ (SVG): cao vượt các chữ khác, nét vuông bo nhẹ, nghiêng, màu cam có vệt sáng.
// Phần chữ còn lại lấy màu chữ của nền (đen trên nền sáng, trắng trên header tối). Dòng mô tả nhỏ bên dưới.
defineProps({ tagline: { type: Boolean, default: true } });
</script>

<template>
  <span class="wordmark" role="img" aria-label="ReqSentinel">
    <span class="wm-name" aria-hidden="true"
      >Req<svg class="wm-s" viewBox="0 0 60 100" focusable="false">
        <path class="wm-s-body" d="M52 12 H21 Q10 12 10 23 V37 Q10 48 21 48 H39 Q50 48 50 59 V77 Q50 88 39 88 H6" />
        <path class="wm-s-shine" d="M35 6 H40 L34 18 H29 Z" /></svg
      >entinel</span
    >
    <span v-if="tagline" class="wm-tag" aria-hidden="true">Access-control checker</span>
  </span>
</template>
