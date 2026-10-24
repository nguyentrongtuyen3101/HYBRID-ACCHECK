<script setup>
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import AdminOverview from "../components/admin/AdminOverview.vue";
import AdminUsers from "../components/admin/AdminUsers.vue";
import AdminAudit from "../components/admin/AdminAudit.vue";

const { t } = useI18n();
const tab = ref("overview");
const tabs = ["overview", "users", "audit"];
</script>

<template>
  <section>
    <div class="row-between mb-3">
      <div>
        <h1>{{ t("admin.title") }}</h1>
        <p class="muted">{{ t("admin.subtitle") }}</p>
      </div>
    </div>
    <div class="tabs">
      <button v-for="key in tabs" :key="key" type="button" :class="{ active: tab === key }" @click="tab = key">
        {{ t(`admin.tabs.${key}`) }}
      </button>
    </div>
    <!-- v-if để mỗi lần mở tab là dữ liệu mới -->
    <AdminOverview v-if="tab === 'overview'" />
    <AdminUsers v-else-if="tab === 'users'" />
    <AdminAudit v-else />
  </section>
</template>
