<script setup>
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../application/container.js";
import { useErrorText } from "../composables/i18n.js";
import AuthCard from "../components/common/AuthCard.vue";

const { t } = useI18n();
const { session } = useContainer();
const errorText = useErrorText();
const email = ref("");
const busy = ref(false);
const sent = ref(false);
const error = ref("");

async function submit() {
  busy.value = true;
  error.value = "";
  try {
    await session.forgotPassword(email.value);
    sent.value = true;
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <AuthCard>
    <h2>{{ t("auth.forgotTitle") }}</h2>
    <div v-if="sent" class="alert alert-ok">{{ t("auth.forgotSent", { email }) }}</div>
    <form v-else class="stack" @submit.prevent="submit">
      <p class="muted">{{ t("auth.forgotIntro") }}</p>
      <div>
        <label class="label" for="forgot-email">{{ t("auth.email") }}</label>
        <input id="forgot-email" v-model="email" class="field" type="email" autocomplete="username" required />
      </div>
      <button type="submit" class="btn btn-primary" :disabled="busy">{{ t("auth.sendLink") }}</button>
    </form>
    <div v-if="error" class="alert alert-error mt-3">{{ error }}</div>
    <RouterLink class="muted mt-3" style="display: block" to="/login">{{ t("auth.backToSignIn") }}</RouterLink>
  </AuthCard>
</template>
