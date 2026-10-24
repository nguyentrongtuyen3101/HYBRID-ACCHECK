<script setup>
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../application/container.js";
import { CONFLICT_BADGES } from "../../domain/conflicts.js";
import { useEnumLabel, useErrorText, useFormat } from "../composables/i18n.js";

const props = defineProps({ id: { type: String, required: true } });
const { t } = useI18n();
const router = useRouter();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime } = useFormat();
const run = ref(null);
const error = ref("");

function back() {
  if (window.history.state?.back) router.back();
  else router.push("/history");
}

onMounted(async () => {
  try {
    run.value = await api.analysis.run(props.id);
  } catch (e) {
    error.value = errorText(e);
  }
});
</script>

<template>
  <section>
    <button type="button" class="btn btn-secondary btn-sm mb-3" @click="back">{{ t("common.back") }}</button>
    <h1 class="mb-3">{{ t("run.title") }}</h1>
    <div class="card">
      <div v-if="error" class="alert alert-error">{{ error }}</div>
      <div v-else-if="!run" class="empty">{{ t("common.loading") }}</div>
      <template v-else>
        <div class="row mb-3">
          <span class="badge" :class="run.is_consistent ? 'b-ok' : 'b-err'">
            {{ run.is_consistent ? t("analysis.consistent") : t("analysis.conflictsFound") }}
          </span>
          <span class="muted mono">{{ run.id }}</span>
          <span class="muted">{{ dateTime(run.created_at) }}</span>
          <span v-if="run.model_version" class="muted mono">{{ t("analysis.model", { version: run.model_version }) }}</span>
        </div>
        <h2>{{ t("run.userStory") }}</h2>
        <p>{{ run.user_story }}</p>
        <h2 class="mt-3">{{ t("analysis.conflicts") }}</h2>
        <div v-for="(c, i) in run.conflicts" :key="i" class="card" style="padding: 12px; margin-bottom: 8px">
          <div class="row">
            <span class="badge" :class="CONFLICT_BADGES[c.conflict_type] || 'b-missing'">{{ label("conflictType", c.conflict_type) }}</span>
            <RouterLink
              v-if="c.issue_id && run.project_id"
              class="muted"
              :to="{ path: `/projects/${run.project_id}`, query: { tab: 'conflicts', issue: c.issue_id } }"
            >
              {{ t("analysis.openInBoard") }}
            </RouterLink>
          </div>
          <p class="mt-2">{{ c.explanation }}</p>
          <p v-if="c.evidence_us_ac" class="muted">{{ t("analysis.evidenceStory", { value: c.evidence_us_ac }) }}</p>
          <p v-if="c.evidence_pm" class="muted">{{ t("analysis.evidencePm", { value: c.evidence_pm }) }}</p>
        </div>
        <div v-if="!run.conflicts.length" class="muted">{{ t("run.noConflicts") }}</div>
        <h2 class="mt-3">{{ t("analysis.extracted") }}</h2>
        <div v-for="(e, i) in run.extracted_entities" :key="i" class="mono">
          {{ e.role || "—" }} · {{ e.action || "—" }} · {{ e.resource || "—" }}
        </div>
        <span v-if="!run.extracted_entities.length" class="muted">—</span>
        <h2 class="mt-3">{{ t("analysis.summary") }}</h2>
        <pre class="summary">{{ run.summary || "" }}</pre>
      </template>
    </div>
  </section>
</template>
