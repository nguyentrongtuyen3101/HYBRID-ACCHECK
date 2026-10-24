<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { can, EFFECTS, Permission } from "../../../domain/roles.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showConfirm, showToast } from "../../composables/ui.js";
import AppModal from "../common/AppModal.vue";
import CustomSelect from "../common/CustomSelect.vue";
import PMDiff from "./PMDiff.vue";

const props = defineProps({
  projectId: { type: String, required: true },
  role: { type: String, required: true },
  rows: { type: Array, required: true },
});
const emit = defineEmits(["changed"]);
const { t } = useI18n();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime } = useFormat();
const reqs = api.requirements;
const canEdit = computed(() => can(props.role, Permission.Edit));

const effectOptions = computed(() => EFFECTS.map((e) => ({ value: e, label: label("effect", e) })));
const blank = () => ({ role: "", action: "", resource: "", effect: "Allow", scope: "", condition: "" });
const form = reactive(blank());
const editing = ref(null);

function payload(f) {
  return {
    role: f.role.trim(),
    action: f.action.trim(),
    resource: f.resource.trim(),
    effect: f.effect,
    scope: f.scope.trim() || null,
    condition: f.condition.trim() || null,
  };
}

async function attempt(action, successKey) {
  try {
    await action();
    showToast(t(successKey), "success");
    return true;
  } catch (e) {
    showToast(errorText(e), "error");
    return false;
  }
}

async function add() {
  const body = payload(form);
  if (!body.role || !body.action || !body.resource) {
    showToast(t("matrix.fieldsRequired"), "error");
    return;
  }
  if (await attempt(() => reqs.addRow(props.projectId, body), "matrix.added")) {
    Object.assign(form, blank());
    emit("changed");
  }
}

async function remove(r) {
  if (!(await showConfirm(t("matrix.deleteConfirm"), t("matrix.deleteTitle")))) return;
  if (await attempt(() => reqs.deleteRow(props.projectId, r.id), "matrix.deleted")) emit("changed");
}

function openEdit(r) {
  editing.value = { ...r, effect: r.effect || "Allow", scope: r.scope || "", condition: r.condition || "" };
}

async function saveEdit() {
  if (await attempt(() => reqs.updateRow(props.projectId, editing.value.id, payload(editing.value)), "matrix.updated")) {
    editing.value = null;
    emit("changed");
  }
}

// ---------- Phiên bản ----------
const versions = ref([]);
const versionLabel = ref("");
const diff = ref(null); // { version, data }

async function loadVersions() {
  try {
    versions.value = await reqs.versions(props.projectId);
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function saveVersion() {
  try {
    const v = await reqs.saveVersion(props.projectId, versionLabel.value.trim() || null);
    versionLabel.value = "";
    showToast(t("versions.saved", { version: v.version }), "success");
    await loadVersions();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function showDiff(v) {
  try {
    diff.value = { version: v, data: await reqs.diffVersions(props.projectId, v.id) };
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function restore(v) {
  if (!(await showConfirm(t("versions.restoreConfirm", { version: v.version }), t("versions.restoreTitle")))) return;
  try {
    await reqs.restoreVersion(props.projectId, v.id);
    showToast(t("versions.restored", { version: v.version }), "success");
    diff.value = null;
    emit("changed");
    await loadVersions();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

onMounted(loadVersions);
</script>

<template>
  <div>
    <div class="card mb-3">
      <h2>{{ t("matrix.title") }}</h2>
      <div class="mb-3">
        <div v-if="!rows.length" class="empty">{{ t("matrix.empty") }}</div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>{{ t("matrix.role") }}</th>
              <th>{{ t("matrix.action") }}</th>
              <th>{{ t("matrix.resource") }}</th>
              <th>{{ t("matrix.effect") }}</th>
              <th>{{ t("matrix.scope") }}</th>
              <th>{{ t("matrix.condition") }}</th>
              <th v-if="canEdit" class="col-actions"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.id">
              <td>{{ r.role }}</td>
              <td>{{ r.action }}</td>
              <td>{{ r.resource }}</td>
              <td>{{ label("effect", r.effect) }}</td>
              <td>{{ r.scope || "—" }}</td>
              <td>{{ r.condition || "—" }}</td>
              <td v-if="canEdit" class="col-actions">
                <div class="row">
                  <button class="btn btn-secondary btn-sm" @click="openEdit(r)">{{ t("common.edit") }}</button>
                  <button class="btn btn-danger" @click="remove(r)">{{ t("common.delete") }}</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <template v-if="canEdit">
        <h2>{{ t("matrix.addTitle") }}</h2>
        <div class="form-grid mb-3">
          <div><label class="label">{{ t("matrix.role") }}</label><input v-model="form.role" class="field" /></div>
          <div><label class="label">{{ t("matrix.action") }}</label><input v-model="form.action" class="field" /></div>
          <div><label class="label">{{ t("matrix.resource") }}</label><input v-model="form.resource" class="field" /></div>
          <div>
            <label class="label">{{ t("matrix.effect") }}</label>
            <CustomSelect v-model="form.effect" :options="effectOptions" />
          </div>
          <div><label class="label">{{ t("matrix.scope") }}</label><input v-model="form.scope" class="field" :placeholder="t('matrix.scopePlaceholder')" /></div>
          <div><label class="label">{{ t("matrix.condition") }}</label><input v-model="form.condition" class="field" /></div>
        </div>
        <button type="button" class="btn btn-primary" @click="add">{{ t("matrix.addButton") }}</button>
      </template>
    </div>

    <div class="card">
      <div class="row-between mb-3">
        <div>
          <h2 style="margin: 0">{{ t("versions.title") }}</h2>
          <p class="muted">{{ t("versions.hint") }}</p>
        </div>
        <div v-if="canEdit" class="toolbar">
          <input v-model="versionLabel" class="field" :placeholder="t('versions.labelPlaceholder')" />
          <button type="button" class="btn btn-secondary btn-sm" @click="saveVersion">{{ t("versions.save") }}</button>
        </div>
      </div>
      <div v-if="!versions.length" class="empty">{{ t("versions.empty") }}</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>{{ t("versions.version") }}</th>
            <th>{{ t("versions.label") }}</th>
            <th>{{ t("versions.rows") }}</th>
            <th>{{ t("versions.savedAt") }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in versions" :key="v.id">
            <td class="mono">v{{ v.version }}</td>
            <td>{{ v.label || "—" }}</td>
            <td>{{ v.row_count }}</td>
            <td class="muted">{{ dateTime(v.created_at) }}</td>
            <td class="col-actions">
              <div class="row">
                <button type="button" class="btn btn-secondary btn-sm" @click="showDiff(v)">{{ t("versions.compare") }}</button>
                <button v-if="canEdit" type="button" class="btn btn-secondary btn-sm" @click="restore(v)">{{ t("versions.restore") }}</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <AppModal v-if="editing" :title="t('matrix.editTitle')" max-width="520px">
      <div class="form-grid">
        <div><label class="label">{{ t("matrix.role") }}</label><input v-model="editing.role" class="field" /></div>
        <div><label class="label">{{ t("matrix.action") }}</label><input v-model="editing.action" class="field" /></div>
        <div><label class="label">{{ t("matrix.resource") }}</label><input v-model="editing.resource" class="field" /></div>
        <div>
          <label class="label">{{ t("matrix.effect") }}</label>
          <CustomSelect v-model="editing.effect" :options="effectOptions" />
        </div>
        <div><label class="label">{{ t("matrix.scope") }}</label><input v-model="editing.scope" class="field" /></div>
        <div><label class="label">{{ t("matrix.condition") }}</label><input v-model="editing.condition" class="field" /></div>
      </div>
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="editing = null">{{ t("common.cancel") }}</button>
        <button type="button" class="btn btn-primary" @click="saveEdit">{{ t("common.save") }}</button>
      </template>
    </AppModal>

    <AppModal v-if="diff" :title="t('versions.diffTitle', { version: diff.version.version })" max-width="720px">
      <PMDiff :diff="diff.data" />
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="diff = null">{{ t("common.close") }}</button>
        <button v-if="canEdit" type="button" class="btn btn-primary" @click="restore(diff.version)">
          {{ t("versions.restoreVersion", { version: diff.version.version }) }}
        </button>
      </template>
    </AppModal>
  </div>
</template>
