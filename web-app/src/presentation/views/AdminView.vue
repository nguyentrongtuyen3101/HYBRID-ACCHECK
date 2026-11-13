<script setup>
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import AdminOverview from "../components/admin/AdminOverview.vue";
import AdminUsers from "../components/admin/AdminUsers.vue";
import AdminAudit from "../components/admin/AdminAudit.vue";
import PageHeader from "../components/common/PageHeader.vue";

const { t } = useI18n();
const tab = ref("overview");
const tabs = ["overview", "users", "audit"];
</script>

<template>
  <section>
    <PageHeader :eyebrow="t('page.admin.eyebrow')" :title="t('page.admin.title')" :dim="t('page.admin.dim')" :subtitle="t('admin.subtitle')" />
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
