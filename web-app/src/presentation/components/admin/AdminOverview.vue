<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showToast } from "../../composables/ui.js";
import BarList from "../common/BarList.vue";

const { t } = useI18n();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { percent } = useFormat();
const stats = ref(null);
const exporting = ref(false);

const cards = computed(() => {
  const s = stats.value;
  if (!s) return [];
  const rate = s.runs.total ? (s.runs.consistent / s.runs.total) * 100 : 0;
  return [
    [t("adminOverview.users"), s.users.total, t("adminOverview.usersSub", { active: s.users.active, admins: s.users.admins })],
    [t("adminOverview.disabledUsers"), s.users.disabled, ""],
    [t("adminOverview.projects"), s.projects, ""],
    [t("adminOverview.runs"), s.runs.total, t("adminOverview.runsSub", { count: s.runs.last_7_days })],
    [t("adminOverview.consistentRuns"), percent(rate), t("adminOverview.consistentSub", { count: s.runs.inconsistent })],
    [t("adminOverview.openConflicts"), s.open_issues, t("adminOverview.openConflictsSub")],
  ];
});
const conflicts = computed(() =>
  Object.entries(stats.value?.conflicts_by_type || {})
    .sort((a, b) => b[1] - a[1])
    .map(([k, v]) => [label("conflictType", k), v])
);
const models = computed(() => Object.entries(stats.value?.runs_by_model || {}).sort((a, b) => b[1] - a[1]));
const perDay = computed(() => (stats.value?.runs_per_day || []).map((d) => [d.day.slice(5), d.count]));

async function exportDataset() {
  exporting.value = true;
  try {
    const { blob, filename } = await api.admin.downloadDataset();
    const url = URL.createObjectURL(blob);
    Object.assign(document.createElement("a"), { href: url, download: filename }).click();
    URL.revokeObjectURL(url);
  } catch (e) {
    showToast(errorText(e), "error");
  } finally {
    exporting.value = false;
  }
}

onMounted(async () => {
  try {
    stats.value = await api.admin.stats();
  } catch (e) {
    showToast(errorText(e), "error");
  }
});
</script>

<template>
  <div v-if="!stats" class="empty">{{ t("common.loading") }}</div>
  <div v-else>
    <div class="stat-grid mb-3">
      <div v-for="[name, value, sub] in cards" :key="name" class="card stat">
        <div class="stat-label">{{ name }}</div>
        <div class="stat-value">{{ value }}</div>
        <div class="muted">{{ sub }}</div>
      </div>
    </div>
    <div class="grid-2 mb-3">
      <div class="card">
        <h2>{{ t("adminOverview.conflictsByType") }}</h2>
        <BarList :entries="conflicts" :empty-text="t('adminOverview.noConflicts')" />
      </div>
      <div class="card">
        <h2>{{ t("adminOverview.runsPerDay") }}</h2>
        <BarList :entries="perDay" :empty-text="t('adminOverview.noRuns')" />
      </div>
    </div>
    <div class="card">
      <div class="row-between mb-3">
        <div>
          <h2 style="margin: 0">{{ t("adminOverview.feedbackTitle") }}</h2>
          <p class="muted">{{ t("adminOverview.feedbackHint") }}</p>
        </div>
        <button type="button" class="btn btn-primary btn-sm" :disabled="exporting" @click="exportDataset">
          {{ exporting ? t("report.exporting") : t("adminOverview.exportDataset") }}
        </button>
      </div>
      <h2>{{ t("adminOverview.runsByModel") }}</h2>
      <BarList :entries="models" :empty-text="t('adminOverview.noRuns')" />
    </div>
  </div>
</template>
