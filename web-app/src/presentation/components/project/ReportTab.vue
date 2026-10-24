<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showToast } from "../../composables/ui.js";
import BarList from "../common/BarList.vue";

const props = defineProps({ projectId: { type: String, required: true } });
const { t } = useI18n();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime, percent } = useFormat();
const report = ref(null);
const coverage = ref(null);
const exporting = ref(false);

onMounted(async () => {
  try {
    [report.value, coverage.value] = await Promise.all([api.reports.report(props.projectId), api.reports.coverage(props.projectId)]);
  } catch (e) {
    showToast(errorText(e), "error");
  }
});

const pairs = (items, group) => (items || []).map((c) => [group ? label(group, c.key) : c.key, c.count]);
const cellOf = computed(() => {
  const map = {};
  for (const c of coverage.value?.cells || []) map[`${c.role}|${c.resource}`] = c;
  return map;
});
const unused = computed(() => (coverage.value?.rows || []).filter((r) => !r.covered && r.effect === "Allow"));
const openTotal = computed(() => (report.value?.stories || []).reduce((n, s) => n + s.open_issues, 0));

function cellClass(c) {
  if (!c) return "cov-empty";
  if (!c.allow) return "cov-deny";
  return c.uncovered ? (c.covered ? "cov-partial" : "cov-unused") : "cov-full";
}

async function exportExcel() {
  exporting.value = true;
  try {
    const { blob, filename } = await api.reports.downloadExcel(props.projectId);
    const url = URL.createObjectURL(blob);
    Object.assign(document.createElement("a"), { href: url, download: filename }).click();
    URL.revokeObjectURL(url);
  } catch (e) {
    showToast(errorText(e), "error");
  } finally {
    exporting.value = false;
  }
}
</script>

<template>
  <div v-if="!report || !coverage" class="empty">{{ t("common.loading") }}</div>
  <div v-else>
    <div class="row-between mb-3">
      <p class="muted">{{ t("report.basedOn", { date: dateTime(report.generated_at) }) }}</p>
      <button type="button" class="btn btn-primary btn-sm" :disabled="exporting" @click="exportExcel">
        {{ exporting ? t("report.exporting") : t("report.export") }}
      </button>
    </div>

    <div class="stat-grid mb-3">
      <div class="card stat">
        <div class="stat-label">{{ t("report.storiesAnalyzed") }}</div>
        <div class="stat-value">{{ report.stories_analyzed }}/{{ report.stories_total }}</div>
      </div>
      <div class="card stat">
        <div class="stat-label">{{ t("report.consistency") }}</div>
        <div class="stat-value">{{ percent(report.consistency_rate) }}</div>
        <div class="muted">{{ t("report.consistentStories", { count: report.stories_consistent }) }}</div>
      </div>
      <div class="card stat">
        <div class="stat-label">{{ t("report.openConflicts") }}</div>
        <div class="stat-value">{{ openTotal }}</div>
      </div>
      <div class="card stat">
        <div class="stat-label">{{ t("report.unusedAllow") }}</div>
        <div class="stat-value">{{ report.unused_allow }}</div>
        <div class="muted">{{ t("report.overGranting") }}</div>
      </div>
    </div>

    <div class="grid-3 mb-3">
      <div class="card">
        <h2>{{ t("report.byStatus") }}</h2>
        <BarList :entries="pairs(report.issues_by_status, 'issueStatus')" :empty-text="t('report.noConflicts')" />
      </div>
      <div class="card">
        <h2>{{ t("report.byType") }}</h2>
        <BarList :entries="pairs(report.issues_by_type, 'conflictType')" :empty-text="t('report.noConflicts')" />
      </div>
      <div class="card">
        <h2>{{ t("report.byRole") }}</h2>
        <BarList :entries="pairs(report.open_issues_by_role)" :empty-text="t('report.noOpenConflicts')" />
      </div>
    </div>

    <div class="card mb-3">
      <h2>{{ t("report.coverageTitle") }}</h2>
      <p class="muted mb-3">{{ t("report.coverageHint", { count: coverage.analyzed_stories }) }}</p>
      <div v-if="!coverage.roles.length" class="empty">{{ t("matrix.empty") }}</div>
      <div v-else class="table-scroll">
        <table class="table coverage">
          <thead>
            <tr>
              <th>{{ t("report.roleResource") }}</th>
              <th v-for="res in coverage.resources" :key="res">{{ res }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="role in coverage.roles" :key="role">
              <th>{{ role }}</th>
              <td v-for="res in coverage.resources" :key="res" :class="cellClass(cellOf[`${role}|${res}`])">
                <template v-if="cellOf[`${role}|${res}`]">
                  {{ cellOf[`${role}|${res}`].covered }}/{{ cellOf[`${role}|${res}`].allow + cellOf[`${role}|${res}`].deny }}
                </template>
              </td>
            </tr>
          </tbody>
        </table>
        <div class="row muted mt-2">
          <span class="legend cov-full"></span> {{ t("report.legendFull") }}
          <span class="legend cov-partial"></span> {{ t("report.legendPartial") }}
          <span class="legend cov-unused"></span> {{ t("report.legendUnused") }}
          <span class="legend cov-deny"></span> {{ t("report.legendDeny") }}
        </div>
      </div>
      <template v-if="unused.length">
        <h2 class="mt-3">{{ t("report.unusedTitle") }}</h2>
        <div v-for="r in unused" :key="r.id" class="diff-row diff-chg">
          <strong>{{ r.role }} · {{ r.action }} · {{ r.resource }}</strong>
          <span class="muted">{{ [r.scope && t("diff.scope", { scope: r.scope }), r.condition].filter(Boolean).join(", ") }}</span>
        </div>
      </template>
    </div>

    <div class="card">
      <h2>{{ t("report.storiesTitle") }}</h2>
      <table class="table">
        <thead>
          <tr>
            <th>{{ t("history.status") }}</th>
            <th>{{ t("history.userStory") }}</th>
            <th>{{ t("report.openConflicts") }}</th>
            <th>{{ t("report.lastAnalyzed") }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in report.stories" :key="s.id">
            <td>
              <span v-if="s.is_consistent === null" class="badge b-disabled">{{ t("report.notAnalyzed") }}</span>
              <span v-else class="badge" :class="s.is_consistent ? 'b-ok' : 'b-err'">
                {{ s.is_consistent ? t("analysis.consistent") : t("analysis.conflictsFound") }}
              </span>
            </td>
            <td class="col-content">{{ s.content }}</td>
            <td>
              <RouterLink v-if="s.open_issues" :to="{ query: { tab: 'conflicts' } }">{{ s.open_issues }}</RouterLink>
              <span v-else class="muted">0</span>
            </td>
            <td class="muted">{{ dateTime(s.last_run_at) || "—" }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
