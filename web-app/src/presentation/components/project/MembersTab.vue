<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../../application/container.js";
import { can, Permission, PROJECT_ROLES } from "../../../domain/roles.js";
import { useEnumLabel, useErrorText, useFormat } from "../../composables/i18n.js";
import { showConfirm, showToast } from "../../composables/ui.js";
import CustomSelect from "../common/CustomSelect.vue";

const props = defineProps({
  projectId: { type: String, required: true },
  role: { type: String, required: true },
  members: { type: Array, required: true },
});
const emit = defineEmits(["changed"]);
const { t } = useI18n();
const router = useRouter();
const { api, session } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime } = useFormat();
const canManage = computed(() => can(props.role, Permission.ManageMembers));
const me = computed(() => session.user.value?.id);
const roleOptions = computed(() => PROJECT_ROLES.map((r) => ({ value: r, label: label("projectRole", r) })));

const invitations = ref([]);
const form = reactive({ email: "", role: "editor" });
const inviting = ref(false);

async function loadInvitations() {
  if (!canManage.value) return;
  try {
    invitations.value = await api.members.invitations(props.projectId);
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function invite() {
  if (!form.email.trim()) return;
  inviting.value = true;
  try {
    const inv = await api.members.invite(props.projectId, form.email.trim(), form.role);
    showToast(t("members.invitationSent", { email: inv.email }), "success");
    form.email = "";
    await loadInvitations();
  } catch (e) {
    showToast(errorText(e), "error");
  } finally {
    inviting.value = false;
  }
}

async function revoke(inv) {
  if (!(await showConfirm(t("members.revokeConfirm", { email: inv.email }), t("members.revokeTitle")))) return;
  try {
    await api.members.revoke(props.projectId, inv.id);
    await loadInvitations();
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function changeRole(m, role) {
  if (role === m.role) return;
  if (!(await showConfirm(t("members.roleConfirm", { email: m.email, role: label("projectRole", role) }), t("members.roleTitle")))) return;
  try {
    await api.members.changeRole(props.projectId, m.user_id, role);
    showToast(t("members.roleUpdated"), "success");
    emit("changed");
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

async function remove(m) {
  const self = m.user_id === me.value;
  const ok = await showConfirm(
    self ? t("members.leaveConfirm") : t("members.removeConfirm", { email: m.email }),
    self ? t("members.leaveTitle") : t("members.removeTitle")
  );
  if (!ok) return;
  try {
    await api.members.remove(props.projectId, m.user_id);
    if (self) {
      showToast(t("members.left"), "success");
      router.push("/");
    } else {
      showToast(t("members.removed"), "success");
      emit("changed");
    }
  } catch (e) {
    showToast(errorText(e), "error");
  }
}

onMounted(loadInvitations);
</script>

<template>
  <div>
    <div class="card mb-3">
      <h2>{{ t("members.title") }}</h2>
      <table class="table">
        <thead>
          <tr>
            <th>{{ t("members.member") }}</th>
            <th>{{ t("members.role") }}</th>
            <th>{{ t("members.joined") }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in members" :key="m.user_id">
            <td>
              <div><strong>{{ m.full_name || "—" }}</strong><span v-if="m.user_id === me" class="muted"> {{ t("common.you") }}</span></div>
              <div class="muted">{{ m.email }}</div>
            </td>
            <td>
              <CustomSelect v-if="canManage" :model-value="m.role" :options="roleOptions" @update:model-value="(r) => changeRole(m, r)" />
              <span v-else class="badge b-user">{{ label("projectRole", m.role) }}</span>
            </td>
            <td class="muted">{{ dateTime(m.added_at) }}</td>
            <td class="col-actions">
              <button v-if="canManage || m.user_id === me" type="button" class="btn btn-danger" @click="remove(m)">
                {{ m.user_id === me ? t("common.leave") : t("common.remove") }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="muted mt-2">
        <div v-for="r in PROJECT_ROLES" :key="r"><strong>{{ label("projectRole", r) }}</strong> — {{ t(`roleDescriptions.${r}`) }}</div>
      </div>
    </div>

    <div v-if="canManage" class="card">
      <h2>{{ t("members.inviteTitle") }}</h2>
      <p class="muted mb-3">{{ t("members.inviteHint") }}</p>
      <form class="toolbar mb-3" @submit.prevent="invite">
        <input v-model="form.email" class="field" style="flex: 1" type="email" :placeholder="t('members.emailPlaceholder')" required />
        <CustomSelect v-model="form.role" :options="roleOptions" :aria-label="t('members.role')" />
        <button type="submit" class="btn btn-primary btn-sm" :disabled="inviting">
          {{ inviting ? t("members.sending") : t("members.send") }}
        </button>
      </form>
      <h2>{{ t("members.pendingTitle") }}</h2>
      <div v-if="!invitations.length" class="empty">{{ t("members.noPending") }}</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>{{ t("auth.email") }}</th>
            <th>{{ t("members.role") }}</th>
            <th>{{ t("members.invitedBy") }}</th>
            <th>{{ t("members.expires") }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="inv in invitations" :key="inv.id">
            <td>{{ inv.email }}</td>
            <td><span class="badge b-user">{{ label("projectRole", inv.role) }}</span></td>
            <td class="muted">{{ inv.inviter_name }}</td>
            <td class="muted">{{ dateTime(inv.expires_at) }}</td>
            <td class="col-actions"><button type="button" class="btn btn-danger" @click="revoke(inv)">{{ t("members.revoke") }}</button></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
