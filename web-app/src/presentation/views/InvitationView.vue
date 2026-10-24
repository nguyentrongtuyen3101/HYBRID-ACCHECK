<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../application/container.js";
import { useEnumLabel, useErrorText, useFormat } from "../composables/i18n.js";
import AuthCard from "../components/common/AuthCard.vue";

const props = defineProps({ token: { type: String, required: true } });
const { t } = useI18n();
const router = useRouter();
const { api, session } = useContainer();
const errorText = useErrorText();
const label = useEnumLabel();
const { dateTime } = useFormat();

const invitation = ref(null);
const error = ref("");
const busy = ref(false);

const here = computed(() => `/invitations/${props.token}`);
const wrongAccount = computed(
  () => session.isLoggedIn() && invitation.value && session.user.value?.email?.toLowerCase() !== invitation.value.email.toLowerCase()
);

onMounted(async () => {
  try {
    invitation.value = await api.members.preview(props.token);
  } catch (e) {
    error.value = errorText(e);
  }
});

async function accept() {
  busy.value = true;
  error.value = "";
  try {
    const project = await api.members.accept(props.token);
    router.replace(`/projects/${project.id}`);
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}

function switchAccount() {
  session.logout();
  router.push({ name: "login", query: { next: here.value, email: invitation.value.email } });
}
</script>

<template>
  <AuthCard>
    <div v-if="!invitation && !error" class="empty">{{ t("common.loading") }}</div>
    <template v-if="invitation">
      <h2>{{ t("invitation.join", { project: invitation.project_name }) }}</h2>
      <p>
        {{ t("invitation.invited", { inviter: invitation.inviter_name, email: invitation.email }) }}
        <span class="badge b-user">{{ label("projectRole", invitation.role) }}</span>
      </p>
      <p class="muted">{{ t(`roleDescriptions.${invitation.role}`) }}</p>
      <p v-if="invitation.status !== 'pending'" class="alert alert-error">
        {{ t("invitation.notPending", { status: label("invitationStatus", invitation.status) }) }}
      </p>
      <template v-else>
        <p class="muted mb-3">{{ t("invitation.expires", { date: dateTime(invitation.expires_at) }) }}</p>
        <div v-if="!session.isLoggedIn()" class="stack">
          <RouterLink class="btn btn-primary" style="text-align: center" :to="{ name: 'login', query: { next: here, email: invitation.email } }">
            {{ t("invitation.signInToAccept") }}
          </RouterLink>
          <RouterLink
            class="btn btn-secondary"
            style="text-align: center"
            :to="{ name: 'login', query: { next: here, email: invitation.email, tab: 'register' } }"
          >
            {{ t("invitation.createAccount") }}
          </RouterLink>
        </div>
        <div v-else-if="wrongAccount" class="stack">
          <div class="alert alert-error">
            {{ t("invitation.wrongAccount", { current: session.user.value.email, email: invitation.email }) }}
          </div>
          <button type="button" class="btn btn-secondary" @click="switchAccount">{{ t("invitation.switchAccount") }}</button>
        </div>
        <button v-else type="button" class="btn btn-primary" :disabled="busy" @click="accept">{{ t("invitation.accept") }}</button>
      </template>
    </template>
    <div v-if="error" class="alert alert-error mt-3">{{ error }}</div>
    <RouterLink v-if="session.isLoggedIn()" class="muted mt-3" style="display: block" to="/">{{ t("invitation.backToProjects") }}</RouterLink>
  </AuthCard>
</template>
