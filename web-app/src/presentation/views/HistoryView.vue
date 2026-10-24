<script setup>
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../application/container.js";
import { useEnumLabel, useErrorText, useFormat } from "../composables/i18n.js";

const { t } = useI18n();
const router = useRouter();
const { api } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime } = useFormat();
const items = ref(null);
const error = ref("");

onMounted(async () => {
  try {
    items.value = (await api.analysis.history(30)).items;
  } catch (e) {
    error.value = errorText(e);
  }
});
</script>

<template>
  <section>
    <h1 class="mb-3">{{ t("history.title") }}</h1>
    <p class="muted mb-3">{{ t("history.subtitle") }}</p>
    <div class="card">
      <div v-if="error" class="alert alert-error">{{ error }}</div>
      <div v-else-if="items === null" class="empty">{{ t("common.loading") }}</div>
      <div v-else-if="!items.length" class="empty">{{ t("history.empty") }}</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>{{ t("history.status") }}</th>
            <th>{{ t("history.project") }}</th>
            <th>{{ t("history.types") }}</th>
            <th>{{ t("history.userStory") }}</th>
            <th>{{ t("history.when") }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in items" :key="r.id">
            <td>
              <span class="badge" :class="r.is_consistent ? 'b-ok' : 'b-err'">{{ r.is_consistent ? t("runs.ok") : t("runs.conflict") }}</span>
            </td>
            <td class="muted">{{ r.project_name || t("history.adhoc") }}</td>
            <td class="muted">{{ r.conflict_types.map((c) => label("conflictType", c)).join(", ") || "—" }}</td>
            <td>{{ r.user_story }}</td>
            <td class="muted">{{ dateTime(r.created_at) }}</td>
            <td class="col-actions">
              <button type="button" class="btn btn-secondary btn-sm" @click="router.push(`/runs/${r.id}`)">{{ t("common.view") }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
