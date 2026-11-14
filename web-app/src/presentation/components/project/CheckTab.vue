<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { JOB_STATUS_BADGES } from "../../../domain/conflicts.js";
import { applyJobProgress, isJobActive, isJobDone } from "../../../domain/jobs.js";
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
const activeJob = computed(() => jobs.value.find(isJobActive));
// Bước đang sáng trong khối quy trình: chưa chạy → 1, đang phân tích → 3, có kết quả → 4
const STEPS = ["extract", "normalize", "align", "reason"];
const activeStep = computed(() => (running.value || activeJob.value ? 2 : result.value ? 3 : 0));
const progress = computed(() => {
  const j = activeJob.value;
  return j && j.total ? Math.round((j.processed / j.total) * 100) : 0;
});
let pollTimer;

async function loadJobs({ silent = false } = {}) {
  try {
    const wasActive = !!activeJob.value;
    jobs.value = await api.analysis.jobs(props.projectId);
    if (wasActive && !activeJob.value) emit("analyzed");
  } catch (e) {
    if (!silent) showToast(errorText(e), "error");
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

// Tiến độ đến realtime. Tín hiệu có thể đến sai thứ tự hoặc trễ, nên bỏ qua tín hiệu cũ
// (applyJobProgress) và luôn hỏi lại máy chủ định kỳ khi có job đang chạy, kể cả khi socket vẫn nối.
const offProgress = notifications.onSignal("job:progress", (p) => {
  if (p.project_id !== props.projectId) return;
  applyJobProgress(jobs.value.find((j) => j.id === p.job_id), p);
  if (isJobDone(p.status) || (p.total && p.processed >= p.total)) loadJobs({ silent: true });
});
watch(
  () => activeJob.value?.id,
  (id) => {
    clearInterval(pollTimer);
    if (id) pollTimer = setInterval(() => loadJobs({ silent: true }), notifications.state.connected ? 5000 : 3000);
  },
);

onMounted(loadJobs);
onBeforeUnmount(() => {
  offProgress();
  clearInterval(pollTimer);
});
</script>

<template>
  <div>
    <p class="eyebrow">{{ t("check.stepsTitle") }}</p>
    <div class="steps">
      <div v-for="(key, i) in STEPS" :key="key" class="step" :class="{ active: activeStep === i }">
        <span class="step-no">{{ i + 1 }}.</span>
        <span class="step-title">{{ t(`check.steps.${key}.title`) }}</span>
        <p class="step-body">{{ t(`check.steps.${key}.body`) }}</p>
      </div>
    </div>

    <div class="card mb-3">
      <div class="row-between mb-3">
        <div>
          <h2>{{ t("check.projectTitle") }}</h2>
          <p class="muted">{{ t("check.projectHint") }}</p>
        </div>
        <button v-if="canRun" type="button" class="btn btn-primary btn-arrow" :disabled="!!activeJob || !stories.length" @click="startJob">
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
