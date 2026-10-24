<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { JOB_STATUS_BADGES } from "../../../domain/conflicts.js";
import { can, Permission } from "../../../domain/roles.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showToast } from "../../composables/ui.js";
import AnalysisResult from "../common/AnalysisResult.vue";
import CustomSelect from "../common/CustomSelect.vue";

const props = defineProps({
  projectId: { type: String, required: true },
  role: { type: String, required: true },
  stories: { type: Array, required: true },
});
const emit = defineEmits(["analyzed"]);
const { t } = useI18n();
const { api, notifications } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime } = useFormat();
const canRun = computed(() => can(props.role, Permission.RunAnalysis));

// ---------- 1 user story ----------
const storyId = ref("");
const running = ref(false);
const error = ref("");
const result = ref(null);
const options = computed(() => props.stories.map((s) => ({ value: s.id, label: s.content.slice(0, 80) })));

watch(
  () => props.stories,
  (list) => {
    if (!list.some((s) => s.id === storyId.value)) storyId.value = list[0]?.id || "";
  },
  { immediate: true }
);

async function run() {
  error.value = "";
  result.value = null;
  if (!storyId.value) {
    error.value = t("check.selectStory");
    return;
  }
  running.value = true;
  try {
    const data = await api.analysis.checkStory(props.projectId, storyId.value);
    result.value = data;
    showToast(data.is_consistent ? t("check.doneConsistent") : t("check.doneConflicts"), data.is_consistent ? "success" : "error");
    emit("analyzed");
  } catch (e) {
    error.value = errorText(e);
  } finally {
    running.value = false;
  }
}

// ---------- Cả project (job nền) ----------
const jobs = ref([]);
const activeJob = computed(() => jobs.value.find((j) => j.status === "queued" || j.status === "running"));
const progress = computed(() => {
  const j = activeJob.value;
  return j && j.total ? Math.round((j.processed / j.total) * 100) : 0;
});
let pollTimer;

async function loadJobs() {
  try {
    const wasActive = !!activeJob.value;
    jobs.value = await api.analysis.jobs(props.projectId);
    if (wasActive && !activeJob.value) emit("analyzed");
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function startJob() {
  try {
    const job = await api.analysis.startJob(props.projectId);
    jobs.value = [job, ...jobs.value];
    showToast(t("check.jobStarted", { count: job.total }), "info");
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

// Tiến độ đến realtime; nếu mất kết nối socket thì hỏi lại định kỳ
const offProgress = notifications.onSignal("job:progress", (p) => {
  if (p.project_id !== props.projectId) return;
  const job = jobs.value.find((j) => j.id === p.job_id);
  if (job) Object.assign(job, { status: p.status, processed: p.processed, failed: p.failed, total: p.total });
  if (p.status === "completed" || p.status === "failed") loadJobs();
});
watch(activeJob, (job) => {
  clearInterval(pollTimer);
  if (job) pollTimer = setInterval(() => !notifications.state.connected && loadJobs(), 3000);
});

onMounted(loadJobs);
onBeforeUnmount(() => {
  offProgress();
  clearInterval(pollTimer);
});
</script>

<template>
  <div>
    <div class="card mb-3">
      <div class="row-between mb-3">
        <div>
          <h2 style="margin: 0">{{ t("check.projectTitle") }}</h2>
          <p class="muted">{{ t("check.projectHint") }}</p>
        </div>
        <button v-if="canRun" type="button" class="btn btn-primary" :disabled="!!activeJob || !stories.length" @click="startJob">
          {{ t("check.analyzeAll") }}
        </button>
      </div>
      <div v-if="activeJob" class="stack">
        <div class="row-between">
          <span>{{ activeJob.status === "queued" ? t("check.waiting") : t("check.analyzing") }}</span>
          <span class="muted">{{ activeJob.processed }}/{{ activeJob.total }} · {{ progress }}%</span>
        </div>
        <div class="progress"><div class="progress-fill" :style="{ width: `${progress}%` }"></div></div>
      </div>
      <table v-if="jobs.length" class="table mt-3">
        <thead>
          <tr>
            <th>{{ t("check.status") }}</th>
            <th>{{ t("check.stories") }}</th>
            <th>{{ t("check.failed") }}</th>
            <th>{{ t("check.model") }}</th>
            <th>{{ t("check.started") }}</th>
            <th>{{ t("check.finished") }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="j in jobs.slice(0, 5)" :key="j.id">
            <td><span class="badge" :class="JOB_STATUS_BADGES[j.status]">{{ label("jobStatus", j.status) }}</span></td>
            <td>{{ j.processed }}/{{ j.total }}</td>
            <td>{{ j.failed }}</td>
            <td class="mono-sm">{{ j.model_version || "—" }}</td>
            <td class="muted">{{ dateTime(j.started_at || j.created_at) }}</td>
            <td class="muted">{{ dateTime(j.finished_at) || "—" }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="card">
      <h2>{{ t("check.singleTitle") }}</h2>
      <p class="muted mb-3">{{ t("check.singleHint") }}</p>
      <div v-if="!canRun" class="muted">{{ t("check.viewerHint") }}</div>
      <div v-else class="stack">
        <div>
          <label class="label">{{ t("check.userStory") }}</label>
          <CustomSelect v-model="storyId" :options="options" :placeholder="t('check.noStories')" />
        </div>
        <button type="button" class="btn btn-primary" :disabled="running" @click="run">
          {{ running ? t("check.running") : t("check.run") }}
        </button>
        <div v-if="error" class="alert alert-error">{{ error }}</div>
        <div v-if="result"><AnalysisResult :data="result" :project-id="projectId" /></div>
      </div>
    </div>
  </div>
</template>
