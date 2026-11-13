<script setup>
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../application/container.js";
import { can, Permission } from "../../domain/roles.js";
import { useEnumLabel, useErrorText } from "../composables/i18n.js";
import { showConfirm, showToast } from "../composables/ui.js";
import PageHeader from "../components/common/PageHeader.vue";

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
    <PageHeader :eyebrow="t('page.projects.eyebrow')" :title="t('page.projects.title')" :dim="t('page.projects.dim')" :subtitle="t('projects.subtitle')" />

    <div class="card mb-3">
      <p class="eyebrow">{{ t("projects.createTitle") }}</p>
      <div class="create-row">
        <div>
          <label class="label" for="new-project-name">{{ t("projects.name") }}</label>
          <input id="new-project-name" v-model="form.name" class="field" :placeholder="t('projects.namePlaceholder')" @keyup.enter="create" />
        </div>
        <div>
          <label class="label" for="new-project-desc">{{ t("projects.description") }}</label>
          <input id="new-project-desc" v-model="form.description" class="field" :placeholder="t('projects.descriptionPlaceholder')" @keyup.enter="create" />
        </div>
        <button type="button" class="btn btn-primary btn-arrow" @click="create">{{ t("projects.createButton") }}</button>
      </div>
    </div>

    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-else-if="projects === null" class="empty">{{ t("common.loading") }}</div>
    <div v-else-if="!projects.length" class="empty">{{ t("projects.empty") }}</div>
    <div v-else class="project-grid">
      <article v-for="(p, i) in projects" :key="p.id" class="project-card" @click="router.push(`/projects/${p.id}`)">
        <div class="row-between">
          <span class="project-no">{{ String(i + 1).padStart(2, "0") }}</span>
          <span class="badge" :class="p.my_role === 'owner' ? 'b-missing' : 'b-user'">{{ label("projectRole", p.my_role) }}</span>
        </div>
        <h3>{{ p.name }}</h3>
        <p class="meta">{{ p.description || t("projects.noDescription") }}</p>
        <p v-if="p.owner_email && p.owner_id !== session.user.value?.id" class="meta mt-2">{{ t("projects.createdBy", { email: p.owner_email }) }}</p>
        <div class="project-foot">
          <div class="project-figures">
            <div><b>{{ p.user_story_count }}</b><span>{{ t("projects.fig.stories") }}</span></div>
            <div><b>{{ p.pm_row_count }}</b><span>{{ t("projects.fig.rows") }}</span></div>
            <div><b>{{ p.run_count }}</b><span>{{ t("projects.fig.runs") }}</span></div>
            <div><b :class="{ hot: p.open_issue_count }">{{ p.open_issue_count }}</b><span>{{ t("projects.fig.open") }}</span></div>
          </div>
          <button v-if="can(p.my_role, Permission.ManageProject)" type="button" class="btn btn-danger btn-sm" @click.stop="remove(p)">
            {{ t("common.delete") }}
          </button>
        </div>
      </article>
    </div>
  </section>
</template>
