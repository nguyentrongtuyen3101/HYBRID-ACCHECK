<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { can, Permission, TERM_KINDS } from "../../../domain/roles.js";
import { useEnumLabel, useErrorText } from "../../composables/i18n.js";
import { showConfirm, showToast } from "../../composables/ui.js";
import AppModal from "../common/AppModal.vue";
import CustomSelect from "../common/CustomSelect.vue";

const props = defineProps({
  projectId: { type: String, required: true },
  role: { type: String, required: true },
});
const { t } = useI18n();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const reqs = api.requirements;
const canEdit = computed(() => can(props.role, Permission.Edit));

const kindOptions = computed(() => TERM_KINDS.map((k) => ({ value: k, label: label("termKind", k) })));
const terms = ref(null);
const suggestions = ref([]);
const form = reactive({ kind: "role", canonical: "", synonyms: "" });
const editing = ref(null);

const parseSynonyms = (s) => s.split(",").map((x) => x.trim()).filter(Boolean);

async function load() {
  try {
    [terms.value, suggestions.value] = await Promise.all([reqs.terms(props.projectId), reqs.termSuggestions(props.projectId)]);
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function create(term = { kind: form.kind, canonical: form.canonical, synonyms: parseSynonyms(form.synonyms) }) {
  try {
    await reqs.createTerm(props.projectId, term);
    Object.assign(form, { canonical: "", synonyms: "" });
    showToast(t("dictionary.added", { term: term.canonical }), "success");
    await load();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function saveEdit() {
  const term = editing.value;
  try {
    await reqs.updateTerm(props.projectId, term.id, { kind: term.kind, canonical: term.canonical, synonyms: parseSynonyms(term.synonyms) });
    editing.value = null;
    showToast(t("dictionary.updated"), "success");
    await load();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function remove(term) {
  if (!(await showConfirm(t("dictionary.deleteConfirm", { term: term.canonical }), t("dictionary.deleteTitle")))) return;
  try {
    await reqs.deleteTerm(props.projectId, term.id);
    await load();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

onMounted(load);
</script>

<template>
  <div>
    <div class="card mb-3">
      <h2>{{ t("dictionary.title") }}</h2>
      <p class="muted mb-3">{{ t("dictionary.hint") }}</p>
      <div v-if="terms === null" class="empty">{{ t("common.loading") }}</div>
      <div v-else-if="!terms.length" class="empty">{{ t("dictionary.empty") }}</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>{{ t("dictionary.kind") }}</th>
            <th>{{ t("dictionary.canonical") }}</th>
            <th>{{ t("dictionary.synonyms") }}</th>
            <th v-if="canEdit" class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="term in terms" :key="term.id">
            <td><span class="badge b-user">{{ label("termKind", term.kind) }}</span></td>
            <td><strong>{{ term.canonical }}</strong></td>
            <td class="muted">{{ term.synonyms.join(", ") || "—" }}</td>
            <td v-if="canEdit" class="col-actions">
              <div class="row">
                <button class="btn btn-secondary btn-sm" @click="editing = { ...term, synonyms: term.synonyms.join(', ') }">{{ t("common.edit") }}</button>
                <button class="btn btn-danger" @click="remove(term)">{{ t("common.delete") }}</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <template v-if="canEdit">
        <h2 class="mt-3">{{ t("dictionary.addTitle") }}</h2>
        <div class="toolbar">
          <CustomSelect v-model="form.kind" :options="kindOptions" :aria-label="t('dictionary.kind')" />
          <input v-model="form.canonical" class="field" :placeholder="t('dictionary.canonical')" />
          <input v-model="form.synonyms" class="field" style="flex: 1" :placeholder="t('dictionary.synonymsPlaceholder')" />
          <button type="button" class="btn btn-primary btn-sm" @click="create()">{{ t("common.add") }}</button>
        </div>
      </template>
    </div>

    <div v-if="suggestions.length" class="card">
      <h2>{{ t("dictionary.suggestionsTitle") }}</h2>
      <p class="muted mb-3">{{ t("dictionary.suggestionsHint") }}</p>
      <div class="chip-list">
        <span v-for="s in suggestions" :key="s.kind + s.value" class="chip">
          <span class="muted">{{ label("termKind", s.kind) }}</span> {{ s.value }}
          <span class="muted">×{{ s.count }}</span>
          <button v-if="canEdit" type="button" class="link-btn" @click="create({ kind: s.kind, canonical: s.value, synonyms: [] })">
            {{ t("dictionary.quickAdd") }}
          </button>
        </span>
      </div>
    </div>

    <AppModal v-if="editing" :title="t('dictionary.editTitle')" max-width="460px">
      <div class="stack">
        <div>
          <label class="label">{{ t("dictionary.kind") }}</label>
          <CustomSelect v-model="editing.kind" :options="kindOptions" />
        </div>
        <div><label class="label">{{ t("dictionary.canonical") }}</label><input v-model="editing.canonical" class="field" /></div>
        <div><label class="label">{{ t("dictionary.synonymsPlaceholder") }}</label><input v-model="editing.synonyms" class="field" /></div>
      </div>
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="editing = null">{{ t("common.cancel") }}</button>
        <button type="button" class="btn btn-primary" @click="saveEdit">{{ t("common.save") }}</button>
      </template>
    </AppModal>
  </div>
</template>
