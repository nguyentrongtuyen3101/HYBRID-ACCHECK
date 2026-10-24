<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { useErrorText, useFormat } from "../../composables/i18n.js";
import CustomSelect from "../common/CustomSelect.vue";

// Khớp các action api-service ghi vào audit_logs
const ACTIONS = [
  "auth.login",
  "auth.login_failed",
  "auth.register",
  "auth.change_password",
  "auth.password_reset_requested",
  "auth.password_reset",
  "user.create",
  "user.update",
  "user.reset_password",
  "project.create",
  "project.update",
  "project.delete",
  "project.import_excel",
  "project.member_invited",
  "project.member_joined",
  "project.member_role_changed",
  "project.member_removed",
  "project.invitation_revoked",
  "project.pm_restored",
  "conflict.update",
  "analysis.run",
  "analysis.job_started",
];
const LIMIT = 50;

const { t, te } = useI18n();
const { api } = useContainer();
const errorText = useErrorText();
const { dateTime } = useFormat();

const action = ref("");
const offset = ref(0);
const total = ref(0);
const items = ref(null);
const error = ref("");

const actionName = (a) => (te(`auditActions.${a}`) ? t(`auditActions.${a}`) : a);
const actionOptions = computed(() => [{ value: "", label: t("adminAudit.allActions") }, ...ACTIONS.map((a) => ({ value: a, label: actionName(a) }))]);

function detailText(entry) {
  if (!entry.detail) return "";
  return Object.entries(entry.detail)
    .map(([k, v]) => `${k}: ${typeof v === "object" ? JSON.stringify(v) : v}`)
    .join(" · ");
}

async function load() {
  try {
    const data = await api.admin.auditLogs({ limit: LIMIT, offset: offset.value, action: action.value });
    total.value = data.total;
    items.value = data.items;
    error.value = "";
  } catch (e) {
    error.value = errorText(e);
  }
}

watch(action, () => {
  offset.value = 0;
  load();
});
watch(offset, load);
onMounted(load);
</script>

<template>
  <div class="card">
    <div class="toolbar mb-3">
      <CustomSelect v-model="action" :options="actionOptions" :aria-label="t('adminAudit.action')" />
    </div>

    <div class="table-scroll">
      <div v-if="error" class="alert alert-error">{{ error }}</div>
      <div v-else-if="items === null" class="empty">{{ t("common.loading") }}</div>
      <div v-else-if="!items.length" class="empty">{{ t("adminAudit.empty") }}</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>{{ t("adminAudit.when") }}</th>
            <th>{{ t("adminAudit.actor") }}</th>
            <th>{{ t("adminAudit.action") }}</th>
            <th>{{ t("adminAudit.target") }}</th>
            <th>{{ t("adminAudit.detail") }}</th>
            <th>{{ t("adminAudit.ip") }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in items" :key="e.id">
            <td class="muted">{{ dateTime(e.created_at) }}</td>
            <td>{{ e.actor_email || "—" }}</td>
            <td><span :title="e.action">{{ actionName(e.action) }}</span></td>
            <td class="mono-sm">{{ e.target_type || "" }} {{ (e.target_id || "").slice(0, 8) }}</td>
            <td class="muted">{{ detailText(e) }}</td>
            <td class="mono-sm">{{ e.ip || "" }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pager">
      <span class="muted">{{ t("adminAudit.range", { from: total ? offset + 1 : 0, to: offset + (items?.length || 0), total }) }}</span>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="offset <= 0" :aria-label="t('common.prev')" @click="offset = Math.max(0, offset - LIMIT)">←</button>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="offset + LIMIT >= total" :aria-label="t('common.next')" @click="offset += LIMIT">→</button>
    </div>
  </div>
</template>
