<script setup>
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useContainer } from "../../../application/container.js";
import { CONFLICT_BADGES, CONFLICT_TYPES, ISSUE_STATUS_BADGES, ISSUE_STATUSES } from "../../../domain/conflicts.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showToast } from "../../composables/ui.js";
import CustomSelect from "../common/CustomSelect.vue";
import ConflictDetail from "./ConflictDetail.vue";

const props = defineProps({
  projectId: { type: String, required: true },
  role: { type: String, required: true },
  members: { type: Array, required: true },
  stories: { type: Array, required: true },
});
const emit = defineEmits(["changed"]);
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { ago } = useFormat();

const filters = reactive({ status: "active", type: "", assignee_id: "", user_story_id: "" });
const items = ref(null);
const total = ref(0);
const openIssueId = computed(() => (typeof route.query.issue === "string" ? route.query.issue : null));

const statusOptions = computed(() => [
  { value: "active", label: t("conflicts.filterActive") },
  { value: "", label: t("conflicts.filterAllStatuses") },
  ...ISSUE_STATUSES.map((s) => ({ value: s, label: label("issueStatus", s) })),
]);
const typeOptions = computed(() => [
  { value: "", label: t("conflicts.filterAllTypes") },
  ...CONFLICT_TYPES.map((c) => ({ value: c, label: label("conflictType", c) })),
]);
const assigneeOptions = computed(() => [
  { value: "", label: t("conflicts.filterAnyone") },
  { value: "me", label: t("conflicts.filterMe") },
  ...props.members.map((m) => ({ value: m.user_id, label: m.full_name || m.email })),
]);
const storyOptions = computed(() => [
  { value: "", label: t("conflicts.filterAllStories") },
  ...props.stories.map((s) => ({ value: s.id, label: s.content.slice(0, 60) })),
]);

async function load() {
  try {
    const data = await api.conflicts.list(props.projectId, { ...filters, limit: 200 });
    items.value = data.items;
    total.value = data.total;
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

function openIssue(id) {
  router.replace({ query: { ...route.query, tab: "conflicts", issue: id } });
}
function closeIssue() {
  const { issue: _, ...rest } = route.query;
  router.replace({ query: rest });
}
function onIssueChanged() {
  load();
  emit("changed");
}

watch(filters, load);
onMounted(load);
</script>

<template>
  <div class="card">
    <div class="row-between mb-3">
      <div>
        <h2 style="margin: 0">{{ t("conflicts.title") }}</h2>
        <p class="muted">{{ t("conflicts.hint") }}</p>
      </div>
      <div class="toolbar">
        <CustomSelect v-model="filters.status" :options="statusOptions" :aria-label="t('conflicts.status')" />
        <CustomSelect v-model="filters.type" :options="typeOptions" :aria-label="t('conflicts.type')" />
        <CustomSelect v-model="filters.assignee_id" :options="assigneeOptions" :aria-label="t('conflicts.assignee')" />
        <CustomSelect v-model="filters.user_story_id" :options="storyOptions" :aria-label="t('conflicts.userStory')" />
      </div>
    </div>

    <div v-if="items === null" class="empty">{{ t("common.loading") }}</div>
    <div v-else-if="!items.length" class="empty">{{ t("conflicts.empty") }}</div>
    <table v-else class="table">
      <thead>
        <tr>
          <th>{{ t("conflicts.status") }}</th>
          <th>{{ t("conflicts.type") }}</th>
          <th>{{ t("conflicts.roleResource") }}</th>
          <th>{{ t("conflicts.userStory") }}</th>
          <th>{{ t("conflicts.assignee") }}</th>
          <th>{{ t("conflicts.updated") }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="i in items" :key="i.id" class="clickable" @click="openIssue(i.id)">
          <td>
            <span class="badge" :class="ISSUE_STATUS_BADGES[i.status]">{{ label("issueStatus", i.status) }}</span>
            <span v-if="i.auto_resolved" class="muted"> {{ t("conflicts.auto") }}</span>
          </td>
          <td><span class="badge" :class="CONFLICT_BADGES[i.conflict_type]">{{ label("conflictType", i.conflict_type) }}</span></td>
          <td>{{ i.role || "—" }} · {{ i.resource || "—" }}</td>
          <td class="col-content">{{ i.user_story }}</td>
          <td class="muted">{{ i.assignee_name || i.assignee_email || "—" }}</td>
          <td class="muted">
            {{ ago(i.updated_at) }}<span v-if="i.comment_count"> · 💬 {{ i.comment_count }}</span>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-if="items && total > items.length" class="muted mt-2">{{ t("conflicts.showing", { shown: items.length, total }) }}</p>

    <ConflictDetail
      v-if="openIssueId"
      :key="openIssueId"
      :project-id="projectId"
      :issue-id="openIssueId"
      :role="role"
      :members="members"
      @close="closeIssue"
      @changed="onIssueChanged"
    />
  </div>
</template>
