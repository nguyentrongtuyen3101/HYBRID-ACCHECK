<script setup>
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../application/container.js";
import { can, Permission } from "../../domain/roles.js";
import { useEnumLabel, useErrorText } from "../composables/i18n.js";
import { showConfirm, showToast } from "../composables/ui.js";

const { t } = useI18n();
const router = useRouter();
const { api, session } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const projects = ref(null);
const error = ref("");
const form = reactive({ name: "", description: "" });

async function load() {
  error.value = "";
  try {
    projects.value = await api.projects.list();
  } catch (e) {
    error.value = errorText(e);
  }
}

async function create() {
  const name = form.name.trim();
  if (!name) {
    showToast(t("projects.nameRequired"), "error");
    return;
  }
  try {
    const p = await api.projects.create(name, form.description.trim() || null);
    form.name = "";
    form.description = "";
    showToast(t("projects.created"), "success");
    router.push(`/projects/${p.id}`);
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function remove(p) {
  if (!(await showConfirm(t("projects.deleteConfirm"), t("projects.deleteTitle")))) return;
  try {
    await api.projects.remove(p.id);
    showToast(t("projects.deleted"), "success");
    await load();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

onMounted(load);
</script>

<template>
  <section>
    <div class="row-between mb-3">
      <div>
        <h1>{{ t("projects.title") }}</h1>
        <p class="muted">{{ t("projects.subtitle") }}</p>
      </div>
    </div>

    <div class="card mb-3">
      <h2>{{ t("projects.createTitle") }}</h2>
      <div class="stack">
        <div>
          <label class="label">{{ t("projects.name") }}</label>
          <input v-model="form.name" class="field" :placeholder="t('projects.namePlaceholder')" @keyup.enter="create" />
        </div>
        <div>
          <label class="label">{{ t("projects.description") }}</label>
          <input v-model="form.description" class="field" :placeholder="t('projects.descriptionPlaceholder')" />
        </div>
        <div>
          <button type="button" class="btn btn-primary" @click="create">{{ t("projects.createButton") }}</button>
        </div>
      </div>
    </div>

    <div class="stack">
      <div v-if="error" class="alert alert-error">{{ error }}</div>
      <div v-else-if="projects === null" class="empty">{{ t("common.loading") }}</div>
      <div v-else-if="!projects.length" class="empty">{{ t("projects.empty") }}</div>
      <div v-for="p in projects" :key="p.id" class="project-card" @click="router.push(`/projects/${p.id}`)">
        <div class="row-between">
          <div>
            <h3>
              {{ p.name }}
              <span class="badge b-user" style="margin-left: 6px">{{ label("projectRole", p.my_role) }}</span>
            </h3>
            <p class="meta">{{ p.description || t("projects.noDescription") }}</p>
            <p class="meta mt-2">
              {{ t("projects.stats", { stories: p.user_story_count, rows: p.pm_row_count, runs: p.run_count }) }}
              <template v-if="p.open_issue_count"> · <strong>{{ t("projects.openConflicts", { count: p.open_issue_count }) }}</strong></template>
              <template v-if="p.owner_email && p.owner_id !== session.user.value?.id"> · {{ t("projects.createdBy", { email: p.owner_email }) }}</template>
            </p>
          </div>
          <button v-if="can(p.my_role, Permission.ManageProject)" type="button" class="btn btn-danger btn-sm" @click.stop="remove(p)">
            {{ t("common.delete") }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
