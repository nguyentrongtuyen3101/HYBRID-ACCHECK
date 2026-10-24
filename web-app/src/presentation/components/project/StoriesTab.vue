<script setup>
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { splitLines } from "../../../domain/format.js";
import { can, Permission } from "../../../domain/roles.js";
import { useErrorText } from "../../composables/i18n.js";
import { showConfirm, showToast } from "../../composables/ui.js";
import AppModal from "../common/AppModal.vue";

const props = defineProps({
  projectId: { type: String, required: true },
  role: { type: String, required: true },
  stories: { type: Array, required: true },
});
const emit = defineEmits(["changed"]);
const { t } = useI18n();
const { api } = useContainer();
const errorText = useErrorText();
const reqs = api.requirements;
const canEdit = computed(() => can(props.role, Permission.Edit));

const form = reactive({ content: "", ac: "" });
const editing = ref(null); // { id, content, ac }

async function run(action, successKey) {
  try {
    await action();
    if (successKey) showToast(t(successKey), "success");
    emit("changed");
    return true;
  } catch (e) {
    showToast(errorText(e), "error");
    return false;
  }
}

async function add() {
  const content = form.content.trim();
  if (!content) {
    showToast(t("stories.contentRequired"), "error");
    return;
  }
  if (await run(() => reqs.addStory(props.projectId, content, splitLines(form.ac)), "stories.added")) {
    form.content = "";
    form.ac = "";
  }
}

async function remove(us) {
  if (!(await showConfirm(t("stories.deleteConfirm"), t("stories.deleteTitle")))) return;
  await run(() => reqs.deleteStory(props.projectId, us.id), "stories.deleted");
}

function openEdit(us) {
  editing.value = { id: us.id, content: us.content, ac: us.acceptance_criteria.map((a) => a.content).join("\n") };
}

async function saveEdit() {
  const body = { content: editing.value.content.trim(), acceptance_criteria: splitLines(editing.value.ac) };
  if (await run(() => reqs.updateStory(props.projectId, editing.value.id, body), "stories.updated")) editing.value = null;
}

async function importExcel(event) {
  const file = event.target.files?.[0];
  event.target.value = "";
  if (!file) return;
  try {
    const data = await reqs.importExcel(props.projectId, file);
    showToast(
      t("stories.imported", {
        stories: data.user_stories_imported,
        criteria: data.acceptance_criteria_imported || 0,
        rows: data.pm_rows_imported,
      }) + (data.warnings?.length ? " " + t("stories.importWarnings", { count: data.warnings.length }) : ""),
      "success"
    );
    emit("changed");
  } catch (e) {
    showToast(errorText(e), "error");
  }
}
</script>

<template>
  <div>
    <div class="card">
      <div class="row-between mb-3">
        <h2 style="margin: 0">{{ t("stories.title") }}</h2>
        <div class="row">
          <a class="btn btn-secondary btn-sm btn-download-excel" :href="reqs.templateUrl" download="reqsentinel_import_template.xlsx">
            <svg class="icon-download" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M12 3v12" />
              <path d="M7 10l5 5 5-5" />
              <path d="M5 21h14" />
            </svg>
            {{ t("stories.downloadTemplate") }}
          </a>
          <label v-if="canEdit" class="btn btn-secondary btn-sm" style="cursor: pointer; margin: 0">
            {{ t("stories.importExcel") }}
            <input type="file" accept=".xlsx,.xlsm" hidden @change="importExcel" />
          </label>
        </div>
      </div>

      <div class="mb-3">
        <div v-if="!stories.length" class="empty">{{ canEdit ? t("stories.emptyEditable") : t("stories.empty") }}</div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>{{ t("stories.content") }}</th>
              <th>{{ t("stories.ac") }}</th>
              <th v-if="canEdit" class="col-actions"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="us in stories" :key="us.id">
              <td class="col-content">{{ us.content }}</td>
              <td class="col-ac muted">
                <template v-if="us.acceptance_criteria.length">
                  <div v-for="ac in us.acceptance_criteria" :key="ac.id">{{ ac.content }}</div>
                </template>
                <template v-else>—</template>
              </td>
              <td v-if="canEdit" class="col-actions">
                <div class="row">
                  <button class="btn btn-secondary btn-sm" @click="openEdit(us)">{{ t("common.edit") }}</button>
                  <button class="btn btn-danger" @click="remove(us)">{{ t("common.delete") }}</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <template v-if="canEdit">
        <h2>{{ t("stories.addTitle") }}</h2>
        <div class="stack">
          <div>
            <label class="label">{{ t("stories.content") }}</label>
            <textarea v-model="form.content" class="field" :placeholder="t('stories.contentPlaceholder')"></textarea>
          </div>
          <div>
            <label class="label">{{ t("stories.acHint") }}</label>
            <textarea v-model="form.ac" class="field" rows="2"></textarea>
          </div>
          <button type="button" class="btn btn-primary" @click="add">{{ t("stories.addButton") }}</button>
        </div>
      </template>
    </div>

    <AppModal v-if="editing" :title="t('stories.editTitle')">
      <div class="stack">
        <div>
          <label class="label">{{ t("stories.content") }}</label>
          <textarea v-model="editing.content" class="field"></textarea>
        </div>
        <div>
          <label class="label">{{ t("stories.acEditHint") }}</label>
          <textarea v-model="editing.ac" class="field" rows="3"></textarea>
        </div>
      </div>
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="editing = null">{{ t("common.cancel") }}</button>
        <button type="button" class="btn btn-primary" @click="saveEdit">{{ t("common.save") }}</button>
      </template>
    </AppModal>
  </div>
</template>
