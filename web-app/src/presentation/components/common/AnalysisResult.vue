<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { CONFLICT_BADGES } from "../../../domain/conflicts.js";
import { useEnumLabel } from "../../composables/i18n.js";

const props = defineProps({
  data: { type: Object, required: true },
  projectId: { type: String, default: "" },
});
const { t } = useI18n();
const label = useEnumLabel();
const realCount = computed(() => (props.data.conflicts || []).filter((c) => c.conflict_type !== "Normal").length);
</script>

<template>
  <div class="row mb-3">
    <span v-if="data.is_consistent" class="badge b-ok">{{ t("analysis.consistent") }}</span>
    <span v-else class="badge b-err">{{ t("analysis.conflictsFound") }}</span>
    <span class="muted">{{ t("analysis.conflictCount", { count: realCount }, realCount) }}</span>
    <span v-if="data.pipeline_run_id" class="muted mono">{{ t("analysis.saved", { id: data.pipeline_run_id.slice(0, 8) }) }}</span>
    <span v-if="data.model_version" class="muted mono">{{ t("analysis.model", { version: data.model_version }) }}</span>
  </div>
  <h2>{{ t("analysis.conflicts") }}</h2>
  <div v-for="(c, i) in data.conflicts" :key="i" class="card" style="padding: 10px; margin-bottom: 8px">
    <div class="row mb-3">
      <span class="badge" :class="CONFLICT_BADGES[c.conflict_type] || 'b-missing'">{{ label("conflictType", c.conflict_type) }}</span>
      <span class="muted">{{ Number(c.confidence).toFixed(2) }}</span>
      <RouterLink v-if="c.issue_id && projectId" class="muted" :to="{ query: { tab: 'conflicts', issue: c.issue_id } }">
        {{ t("analysis.openInBoard") }}
      </RouterLink>
    </div>
    <div>{{ c.explanation }}</div>
    <div v-if="c.evidence_us_ac" class="muted mt-2">{{ t("analysis.evidenceStory", { value: c.evidence_us_ac }) }}</div>
    <div v-if="c.evidence_pm" class="muted">{{ t("analysis.evidencePm", { value: c.evidence_pm }) }}</div>
  </div>
  <div v-if="!data.conflicts?.length" class="muted">{{ t("common.none") }}</div>
  <h2 class="mt-3">{{ t("analysis.extracted") }}</h2>
  <div v-for="(e, i) in data.extracted_entities" :key="i" class="mono">
    {{ e.role || "—" }} · {{ e.action || "—" }} · {{ e.resource || "—" }}{{ e.scope ? " · " + e.scope : "" }}
  </div>
  <span v-if="!data.extracted_entities?.length" class="muted">—</span>
  <h2 class="mt-3">{{ t("analysis.summary") }}</h2>
  <pre class="summary">{{ data.summary || "" }}</pre>
</template>
