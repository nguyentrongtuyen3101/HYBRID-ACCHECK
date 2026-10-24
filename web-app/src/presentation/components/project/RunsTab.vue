<script setup>
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../../application/container.js";
import { CONFLICT_BADGES } from "../../../domain/conflicts.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showToast } from "../../composables/ui.js";
import AppModal from "../common/AppModal.vue";
import PMDiff from "./PMDiff.vue";

const props = defineProps({ projectId: { type: String, required: true } });
const { t } = useI18n();
const router = useRouter();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime } = useFormat();
const items = ref(null);
const error = ref("");
const selected = ref([]); // tối đa 2 run để so sánh
const comparison = ref(null);

onMounted(async () => {
  try {
    items.value = (await api.analysis.projectRuns(props.projectId, 50)).items;
  } catch (e) {
    error.value = errorText(e);
  }
});

function toggle(id) {
  selected.value = selected.value.includes(id) ? selected.value.filter((x) => x !== id) : [...selected.value, id].slice(-2);
}

async function compare() {
  // So sánh run cũ hơn (base) với run mới hơn (target)
  const [a, b] = selected.value.map((id) => items.value.find((r) => r.id === id));
  const [base, target] = new Date(a.created_at) <= new Date(b.created_at) ? [a, b] : [b, a];
  try {
    comparison.value = await api.analysis.compare(props.projectId, base.id, target.id);
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

const groups = [
  ["introduced", "diff-del"],
  ["resolved", "diff-add"],
  ["unchanged", ""],
];
</script>

<template>
  <div class="card">
    <div class="row-between mb-3">
      <h2 style="margin: 0">{{ t("runs.title") }}</h2>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="selected.length !== 2" @click="compare">
        {{ t("runs.compareSelected", { count: selected.length }) }}
      </button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-else-if="items === null" class="empty">{{ t("common.loading") }}</div>
    <div v-else-if="!items.length" class="empty">{{ t("runs.empty") }}</div>
    <table v-else class="table">
      <thead>
        <tr>
          <th></th>
          <th>{{ t("history.status") }}</th>
          <th>{{ t("history.userStory") }}</th>
          <th>{{ t("analysis.conflicts") }}</th>
          <th>{{ t("history.when") }}</th>
          <th class="col-actions"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="r in items" :key="r.id">
          <td><input type="checkbox" :checked="selected.includes(r.id)" :aria-label="t('runs.select')" @change="toggle(r.id)" /></td>
          <td>
            <span class="badge" :class="r.is_consistent ? 'b-ok' : 'b-err'">{{ r.is_consistent ? t("runs.ok") : t("runs.conflict") }}</span>
          </td>
          <td>{{ r.user_story }}</td>
          <td class="muted">{{ r.conflict_types.map((c) => label("conflictType", c)).join(", ") || "—" }}</td>
          <td class="muted">{{ dateTime(r.created_at) }}</td>
          <td class="col-actions">
            <button type="button" class="btn btn-secondary btn-sm" @click="router.push(`/runs/${r.id}`)">{{ t("common.view") }}</button>
          </td>
        </tr>
      </tbody>
    </table>

    <AppModal v-if="comparison" :title="t('runs.compareTitle')" max-width="760px">
      <p class="muted">
        {{ dateTime(comparison.base.created_at) }} → {{ dateTime(comparison.target.created_at) }}
        <span v-if="comparison.story_changed"> · {{ t("runs.storyChanged") }}</span>
      </p>
      <h2 class="mt-3">{{ t("analysis.conflicts") }}</h2>
      <div class="stack">
        <template v-for="[group, cls] in groups" :key="group">
          <div v-for="(c, i) in comparison[group]" :key="group + i" class="diff-row" :class="cls">
            <strong>{{ t(`runs.${group}`) }}</strong>
            <span class="badge" :class="CONFLICT_BADGES[c.conflict_type]">{{ label("conflictType", c.conflict_type) }}</span>
            {{ c.explanation }}
          </div>
        </template>
        <div v-if="!comparison.introduced.length && !comparison.resolved.length && !comparison.unchanged.length" class="muted">
          {{ t("runs.noConflictsEither") }}
        </div>
      </div>
      <h2 class="mt-3">{{ t("runs.pmChanges") }}</h2>
      <PMDiff :diff="comparison.pm_diff" />
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="comparison = null">{{ t("common.close") }}</button>
      </template>
    </AppModal>
  </div>
</template>
