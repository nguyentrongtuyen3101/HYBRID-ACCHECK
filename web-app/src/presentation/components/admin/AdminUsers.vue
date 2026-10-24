<script setup>
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { SYSTEM_ROLES, USER_STATUSES } from "../../../domain/roles.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showConfirm, showToast } from "../../composables/ui.js";
import AppModal from "../common/AppModal.vue";
import CustomSelect from "../common/CustomSelect.vue";

const { t } = useI18n();
const { api, session } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime } = useFormat();
const me = computed(() => session.user.value?.id);

const PAGE_SIZE = 20;
const filters = reactive({ q: "", role: "", status: "" });
const page = ref(1);
const total = ref(0);
const users = ref(null);
const error = ref("");
const pages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)));

const roleOptions = computed(() => SYSTEM_ROLES.map((r) => ({ value: r, label: label("systemRole", r) })));
const roleFilterOptions = computed(() => [{ value: "", label: t("adminUsers.allRoles") }, ...roleOptions.value]);
const statusFilterOptions = computed(() => [
  { value: "", label: t("adminUsers.allStatuses") },
  ...USER_STATUSES.map((s) => ({ value: s, label: label("userStatus", s) })),
]);

async function load() {
  try {
    const data = await api.admin.users({ page: page.value, page_size: PAGE_SIZE, ...filters, q: filters.q.trim() });
    total.value = data.total;
    users.value = data.items;
    error.value = "";
  } catch (e) {
    error.value = errorText(e);
  }
}

let searchTimer;
watch(
  () => ({ ...filters }),
  () => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      page.value = 1;
      load();
    }, 300);
  }
);
watch(page, load);
onMounted(load);

async function patchUser(id, body, successKey) {
  try {
    await api.admin.updateUser(id, body);
    showToast(t(successKey), "success");
  } catch (e) {
    showToast(errorText(e), "error");
  }
  await load();
}

async function changeRole(u, role) {
  if (role === u.role) return;
  if (await showConfirm(t("adminUsers.roleConfirm", { email: u.email, role: label("systemRole", role) }), t("adminUsers.roleTitle"))) {
    await patchUser(u.id, { role }, "adminUsers.roleUpdated");
  }
}

async function setStatus(u, status) {
  const disabling = status === "disabled";
  const ok = await showConfirm(
    t(disabling ? "adminUsers.disableConfirm" : "adminUsers.enableConfirm", { email: u.email }),
    t(disabling ? "adminUsers.disableTitle" : "adminUsers.enableTitle")
  );
  if (ok) await patchUser(u.id, { status }, disabling ? "adminUsers.disabled" : "adminUsers.enabled");
}

// ---- Tạo user ----
const creating = ref(null);
function openCreate() {
  creating.value = { email: "", full_name: "", password: "", role: "user" };
}
async function create() {
  try {
    await api.admin.createUser({ ...creating.value, email: creating.value.email.trim() });
    creating.value = null;
    showToast(t("adminUsers.created"), "success");
    page.value = 1;
    await load();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

// ---- Reset mật khẩu ----
const resetting = ref(null);
async function resetPassword() {
  try {
    await api.admin.resetPassword(resetting.value.id, resetting.value.password);
    resetting.value = null;
    showToast(t("adminUsers.passwordReset"), "success");
  } catch (e) {
    showToast(errorText(e), "error");
  }
}
</script>

<template>
  <div class="card">
    <div class="row-between mb-3">
      <div class="toolbar">
        <input v-model="filters.q" class="field" :placeholder="t('adminUsers.search')" />
        <CustomSelect v-model="filters.role" :options="roleFilterOptions" :aria-label="t('adminUsers.role')" />
        <CustomSelect v-model="filters.status" :options="statusFilterOptions" :aria-label="t('adminUsers.status')" />
      </div>
      <button type="button" class="btn btn-primary btn-sm" @click="openCreate">{{ t("adminUsers.newUser") }}</button>
    </div>

    <div class="table-scroll">
      <div v-if="error" class="alert alert-error">{{ error }}</div>
      <div v-else-if="users === null" class="empty">{{ t("common.loading") }}</div>
      <div v-else-if="!users.length" class="empty">{{ t("adminUsers.empty") }}</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>{{ t("adminUsers.user") }}</th>
            <th>{{ t("adminUsers.role") }}</th>
            <th>{{ t("adminUsers.status") }}</th>
            <th>{{ t("adminUsers.projects") }}</th>
            <th>{{ t("adminUsers.runs") }}</th>
            <th>{{ t("adminUsers.lastLogin") }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>
              <div>
                <strong>{{ u.full_name || "—" }}</strong>
                <span v-if="u.id === me" class="muted"> {{ t("common.you") }}</span>
              </div>
              <div class="muted">{{ u.email }}</div>
            </td>
            <td>
              <CustomSelect :model-value="u.role" :options="roleOptions" :disabled="u.id === me" @update:model-value="(r) => changeRole(u, r)" />
            </td>
            <td>
              <span class="badge" :class="u.status === 'active' ? 'b-ok' : 'b-disabled'">{{ label("userStatus", u.status) }}</span>
            </td>
            <td>{{ u.project_count }}</td>
            <td>{{ u.run_count }}</td>
            <td class="muted">{{ dateTime(u.last_login_at) || t("common.never") }}</td>
            <td class="col-actions">
              <div class="row">
                <button type="button" class="btn btn-secondary btn-sm" @click="resetting = { id: u.id, email: u.email, password: '' }">
                  {{ t("adminUsers.resetPassword") }}
                </button>
                <template v-if="u.id !== me">
                  <button v-if="u.status === 'active'" type="button" class="btn btn-danger" @click="setStatus(u, 'disabled')">
                    {{ t("adminUsers.disable") }}
                  </button>
                  <button v-else type="button" class="btn btn-secondary btn-sm" @click="setStatus(u, 'active')">{{ t("adminUsers.enable") }}</button>
                </template>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pager">
      <span class="muted">{{ t("adminUsers.pageInfo", { total, page, pages }) }}</span>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="page <= 1" :aria-label="t('common.prev')" @click="page--">←</button>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="page >= pages" :aria-label="t('common.next')" @click="page++">→</button>
    </div>

    <AppModal v-if="creating" :title="t('adminUsers.newUser')" max-width="440px">
      <div class="stack">
        <div><label class="label">{{ t("auth.email") }}</label><input v-model="creating.email" class="field" type="email" /></div>
        <div><label class="label">{{ t("auth.fullName") }}</label><input v-model="creating.full_name" class="field" /></div>
        <div>
          <label class="label">{{ t("adminUsers.initialPassword") }}</label>
          <input v-model="creating.password" class="field" type="password" autocomplete="new-password" />
        </div>
        <div>
          <label class="label">{{ t("adminUsers.role") }}</label>
          <CustomSelect v-model="creating.role" :options="roleOptions" />
        </div>
      </div>
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="creating = null">{{ t("common.cancel") }}</button>
        <button type="button" class="btn btn-primary" @click="create">{{ t("common.create") }}</button>
      </template>
    </AppModal>

    <AppModal v-if="resetting" :title="t('adminUsers.resetPassword')" max-width="420px">
      <p class="muted">{{ resetting.email }}</p>
      <div>
        <label class="label">{{ t("auth.newPassword") }}</label>
        <input v-model="resetting.password" class="field" type="password" autocomplete="new-password" />
      </div>
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="resetting = null">{{ t("common.cancel") }}</button>
        <button type="button" class="btn btn-primary" @click="resetPassword">{{ t("adminUsers.reset") }}</button>
      </template>
    </AppModal>
  </div>
</template>
