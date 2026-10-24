<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useContainer } from "../../application/container.js";
import { can, Permission } from "../../domain/roles.js";
import { useEnumLabel, useErrorText } from "../composables/i18n.js";
import { showConfirm, showToast } from "../composables/ui.js";
import StoriesTab from "../components/project/StoriesTab.vue";
import MatrixTab from "../components/project/MatrixTab.vue";
import DictionaryTab from "../components/project/DictionaryTab.vue";
import CheckTab from "../components/project/CheckTab.vue";
import ConflictsTab from "../components/project/ConflictsTab.vue";
import RunsTab from "../components/project/RunsTab.vue";
import ReportTab from "../components/project/ReportTab.vue";
import MembersTab from "../components/project/MembersTab.vue";
import SettingsTab from "../components/project/SettingsTab.vue";

const props = defineProps({ id: { type: String, required: true } });
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();

const detail = ref(null);
const members = ref([]);
const error = ref("");
const runsVersion = ref(0);

const role = computed(() => detail.value?.project.my_role || "viewer");
const tabs = computed(() =>
  [
    ["stories"],
    ["matrix"],
    ["dictionary"],
    ["check"],
    ["conflicts", detail.value?.project.open_issue_count],
    ["runs"],
    ["report"],
    ["members"],
    can(role.value, Permission.ManageProject) && ["settings"],
  ].filter(Boolean)
);
const tab = computed(() => {
  const value = String(route.query.tab || "stories");
  return tabs.value.some(([k]) => k === value) ? value : "stories";
});

function setTab(key) {
  router.replace({ query: { tab: key } });
}

async function refresh() {
  try {
    const [d, m] = await Promise.all([api.projects.get(props.id), api.members.list(props.id)]);
    detail.value = d;
    members.value = m;
    error.value = "";
  } catch (e) {
    error.value = errorText(e);
  }
}

async function deleteProject() {
  if (!(await showConfirm(t("project.deleteConfirm"), t("projects.deleteTitle")))) return;
  try {
    await api.projects.remove(props.id);
    showToast(t("projects.deleted"), "success");
    router.push("/");
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

function onAnalyzed() {
  runsVersion.value++;
  refresh();
}

onMounted(refresh);
</script>

<template>
  <section>
    <div class="row-between mb-3">
      <div>
        <button type="button" class="btn btn-secondary btn-sm" @click="router.push('/')">{{ t("project.backToProjects") }}</button>
        <h1 class="mt-2">
          {{ detail?.project.name || t("project.fallbackTitle") }}
          <span v-if="detail" class="badge b-user" style="margin-left: 6px; vertical-align: middle">{{ label("projectRole", role) }}</span>
        </h1>
        <p class="muted">{{ detail?.project.description || "" }}</p>
      </div>
      <button v-if="can(role, Permission.ManageProject)" type="button" class="btn btn-danger" @click="deleteProject">
        {{ t("project.deleteButton") }}
      </button>
    </div>

    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-else-if="!detail" class="empty">{{ t("common.loading") }}</div>

    <div v-else>
      <div class="tabs tabs-scroll">
        <button v-for="[key, count] in tabs" :key="key" type="button" :class="{ active: tab === key }" @click="setTab(key)">
          {{ t(`project.tabs.${key}`) }}<span v-if="count" class="tab-count">{{ count }}</span>
        </button>
      </div>

      <StoriesTab v-if="tab === 'stories'" :project-id="id" :role="role" :stories="detail.user_stories" @changed="refresh" />
      <MatrixTab v-else-if="tab === 'matrix'" :project-id="id" :role="role" :rows="detail.permission_matrix" @changed="refresh" />
      <DictionaryTab v-else-if="tab === 'dictionary'" :project-id="id" :role="role" />
      <CheckTab v-else-if="tab === 'check'" :project-id="id" :role="role" :stories="detail.user_stories" @analyzed="onAnalyzed" />
      <ConflictsTab
        v-else-if="tab === 'conflicts'"
        :project-id="id"
        :role="role"
        :members="members"
        :stories="detail.user_stories"
        @changed="refresh"
      />
      <RunsTab v-else-if="tab === 'runs'" :key="runsVersion" :project-id="id" />
      <ReportTab v-else-if="tab === 'report'" :project-id="id" />
      <MembersTab v-else-if="tab === 'members'" :project-id="id" :role="role" :members="members" @changed="refresh" />
      <SettingsTab v-else-if="tab === 'settings'" :project="detail.project" @changed="refresh" />
    </div>
  </section>
</template>
