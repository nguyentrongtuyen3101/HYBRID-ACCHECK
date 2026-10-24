<script setup>
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useContainer } from "../../application/container.js";
import { useErrorText } from "../composables/i18n.js";
import AuthCard from "../components/common/AuthCard.vue";

const { t } = useI18n();
const route = useRoute();
const { session } = useContainer();
const errorText = useErrorText();
const token = String(route.query.token || "");
const password = ref("");
const confirm = ref("");
const busy = ref(false);
const done = ref(false);
const error = ref(token ? "" : t("auth.resetMissingToken"));

async function submit() {
  if (password.value !== confirm.value) {
    error.value = t("auth.passwordsMismatch");
    return;
  }
  busy.value = true;
  error.value = "";
  try {
    await session.resetPassword(token, password.value);
    session.logout();
    done.value = true;
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <AuthCard>
    <h2>{{ t("auth.resetTitle") }}</h2>
    <div v-if="done" class="stack">
      <div class="alert alert-ok">{{ t("auth.resetDone") }}</div>
      <RouterLink class="btn btn-primary" to="/login" style="text-align: center">{{ t("auth.signIn") }}</RouterLink>
    </div>
    <form v-else-if="token" class="stack" @submit.prevent="submit">
      <div>
        <label class="label" for="new-password">{{ t("auth.newPassword") }}</label>
        <input id="new-password" v-model="password" class="field" type="password" autocomplete="new-password" minlength="8" required />
      </div>
      <div>
        <label class="label" for="confirm-password">{{ t("auth.confirmPassword") }}</label>
        <input id="confirm-password" v-model="confirm" class="field" type="password" autocomplete="new-password" required />
      </div>
      <button type="submit" class="btn btn-primary" :disabled="busy">{{ t("auth.changePassword") }}</button>
    </form>
    <div v-if="error" class="alert alert-error mt-3">{{ error }}</div>
    <RouterLink v-if="!done" class="muted mt-3" style="display: block" to="/forgot-password">{{ t("auth.requestNewLink") }}</RouterLink>
  </AuthCard>
</template>
