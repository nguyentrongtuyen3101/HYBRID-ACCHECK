<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { CONFLICT_BADGES, ISSUE_STATUS_BADGES, nextStatuses } from "../../../domain/conflicts.js";
import { can, Permission } from "../../../domain/roles.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showToast } from "../../composables/ui.js";
import CustomSelect from "../common/CustomSelect.vue";

const props = defineProps({
  projectId: { type: String, required: true },
  issueId: { type: String, required: true },
  role: { type: String, required: true },
  members: { type: Array, required: true },
});
const emit = defineEmits(["close", "changed"]);
const { t } = useI18n();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime, ago } = useFormat();

const issue = ref(null);
const activities = ref([]);
const error = ref("");
const comment = ref("");
const busy = ref(false);

const canTriage = computed(() => can(props.role, Permission.Triage));
const canComment = computed(() => can(props.role, Permission.Comment));
// Chỉ owner/editor được giao xử lý
const assigneeOptions = computed(() => [
  { value: "", label: t("conflictDetail.unassigned") },
  ...props.members
    .filter((m) => can(m.role, Permission.Triage))
    .map((m) => ({ value: m.user_id, label: `${m.full_name || m.email} (${label("projectRole", m.role)})` })),
]);
const memberName = (id) => {
  const m = props.members.find((x) => x.user_id === id);
  return m ? m.full_name || m.email : t("conflictDetail.formerMember");
};

async function load() {
  try {
    const data = await api.conflicts.get(props.projectId, props.issueId);
    issue.value = data.issue;
    activities.value = data.activities;
  } catch (e) {
    error.value = errorText(e);
  }
}

async function update(body, message) {
  busy.value = true;
  try {
    issue.value = await api.conflicts.update(props.projectId, props.issueId, body);
    showToast(message, "success");
    await load();
    emit("changed");
  } catch (e) {
    showToast(errorText(e), "error");
  } finally {
    busy.value = false;
  }
}

const setStatus = (status) => update({ status }, t("conflictDetail.statusChanged", { status: label("issueStatus", status) }));
const setAssignee = (id) => update({ assignee_id: id || null }, t("conflictDetail.assigneeChanged"));

async function addComment() {
  if (!comment.value.trim()) return;
  busy.value = true;
  try {
    await api.conflicts.comment(props.projectId, props.issueId, comment.value);
    comment.value = "";
    await load();
    emit("changed");
  } catch (e) {
    showToast(errorText(e), "error");
  } finally {
    busy.value = false;
  }
}

function describe(a) {
  if (a.kind === "status_changed") {
    return t("activity.status_changed", { from: label("issueStatus", a.from), to: label("issueStatus", a.to) });
  }
  if (a.kind === "assigned") return a.to ? t("activity.assigned", { name: memberName(a.to) }) : t("activity.unassigned");
  return t(`activity.${a.kind}`);
}

onMounted(load);
</script>

<template>
  <div class="modal-backdrop" role="dialog" aria-modal="true" @click.self="emit('close')">
    <div class="modal drawer">
      <div class="row-between mb-3">
        <h3 style="margin: 0">{{ t("conflictDetail.title") }}</h3>
        <button type="button" class="btn btn-secondary btn-sm" @click="emit('close')">{{ t("common.close") }}</button>
      </div>
      <div v-if="error" class="alert alert-error">{{ error }}</div>
      <div v-else-if="!issue" class="empty">{{ t("common.loading") }}</div>
      <template v-else>
        <div class="row mb-3">
          <span class="badge" :class="CONFLICT_BADGES[issue.conflict_type]">{{ label("conflictType", issue.conflict_type) }}</span>
          <span class="badge" :class="ISSUE_STATUS_BADGES[issue.status]">{{ label("issueStatus", issue.status) }}</span>
          <span v-if="issue.auto_resolved" class="muted">{{ t("conflictDetail.autoResolved") }}</span>
        </div>
        <p>{{ issue.explanation }}</p>
        <p v-if="issue.evidence_us_ac" class="muted">{{ t("analysis.evidenceStory", { value: issue.evidence_us_ac }) }}</p>
        <p v-if="issue.evidence_pm" class="muted">{{ t("analysis.evidencePm", { value: issue.evidence_pm }) }}</p>
        <p class="mt-2"><strong>{{ t("conflictDetail.userStory") }}</strong> {{ issue.user_story }}</p>
        <p class="muted">
          {{ t("conflictDetail.lastDetected", { date: dateTime(issue.last_seen_at) }) }} ·
          <RouterLink :to="`/runs/${issue.last_seen_run_id}`">{{ t("conflictDetail.viewRun") }}</RouterLink>
        </p>

        <div class="stack mt-3">
          <div v-if="canTriage" class="row">
            <span class="label" style="margin: 0">{{ t("conflictDetail.moveTo") }}</span>
            <button
              v-for="s in nextStatuses(issue.status)"
              :key="s"
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="busy"
              @click="setStatus(s)"
            >
              {{ label("issueStatus", s) }}
            </button>
          </div>
          <div>
            <label class="label">{{ t("conflicts.assignee") }}</label>
            <CustomSelect
              :model-value="issue.assignee_id || ''"
              :options="assigneeOptions"
              :disabled="!canTriage || busy"
              @update:model-value="setAssignee"
            />
          </div>
        </div>

        <h2 class="mt-3">{{ t("conflictDetail.activity") }}</h2>
        <ul class="timeline">
          <li v-for="a in activities" :key="a.id" :class="{ 'timeline-comment': a.kind === 'comment' }">
            <div>
              <strong>{{ a.actor_name || t("conflictDetail.system") }}</strong> {{ describe(a) }}
              <span class="muted"> · {{ ago(a.created_at) }}</span>
            </div>
            <div v-if="a.body" class="timeline-body">{{ a.body }}</div>
          </li>
        </ul>
        <div v-if="canComment" class="stack mt-3">
          <textarea v-model="comment" class="field" rows="2" :placeholder="t('conflictDetail.commentPlaceholder')"></textarea>
          <button type="button" class="btn btn-primary btn-sm" :disabled="busy || !comment.trim()" @click="addComment">
            {{ t("conflictDetail.comment") }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>
