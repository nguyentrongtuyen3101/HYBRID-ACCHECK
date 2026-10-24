<script setup>
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useContainer } from "../../application/container.js";
import { useErrorText } from "../composables/i18n.js";
import AuthCard from "../components/common/AuthCard.vue";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { session } = useContainer();
const errorText = useErrorText();

const tab = ref(route.query.tab === "register" ? "register" : "login");
const busy = ref(false);
const error = ref("");
const prefill = String(route.query.email || "");
const login = reactive({ email: prefill, password: "" });
const reg = reactive({ fullName: "", email: prefill, password: "" });
const emailInput = ref(null);

function nextUrl() {
  const next = String(route.query.next || "/");
  return next.startsWith("/") && !next.startsWith("//") ? next : "/";
}

function switchTab(value) {
  tab.value = value;
  error.value = "";
}

async function submit(action) {
  busy.value = true;
  error.value = "";
  try {
    await action();
    router.replace(nextUrl());
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}

const onLogin = () => submit(() => session.login(login.email, login.password));
const onRegister = () => submit(() => session.register(reg));

onMounted(() => emailInput.value?.focus());
</script>

<template>
  <AuthCard>
    <div class="tabs">
      <button type="button" :class="{ active: tab === 'login' }" @click="switchTab('login')">{{ t("auth.signIn") }}</button>
      <button type="button" :class="{ active: tab === 'register' }" @click="switchTab('register')">{{ t("auth.createAccount") }}</button>
    </div>

    <form v-if="tab === 'login'" class="stack" @submit.prevent="onLogin">
      <div>
        <label class="label" for="login-email">{{ t("auth.email") }}</label>
        <input id="login-email" ref="emailInput" v-model="login.email" class="field" type="email" autocomplete="username" required />
      </div>
      <div>
        <label class="label" for="login-password">{{ t("auth.password") }}</label>
        <input id="login-password" v-model="login.password" class="field" type="password" autocomplete="current-password" required />
      </div>
      <button type="submit" class="btn btn-primary" :disabled="busy">{{ t("auth.signIn") }}</button>
      <RouterLink class="muted" to="/forgot-password">{{ t("auth.forgotLink") }}</RouterLink>
    </form>

    <form v-else class="stack" @submit.prevent="onRegister">
      <div>
        <label class="label" for="reg-name">{{ t("auth.fullName") }}</label>
        <input id="reg-name" v-model="reg.fullName" class="field" autocomplete="name" />
      </div>
      <div>
        <label class="label" for="reg-email">{{ t("auth.email") }}</label>
        <input id="reg-email" v-model="reg.email" class="field" type="email" autocomplete="username" required />
      </div>
      <div>
        <label class="label" for="reg-password">{{ t("auth.passwordHint") }}</label>
        <input id="reg-password" v-model="reg.password" class="field" type="password" autocomplete="new-password" minlength="8" required />
      </div>
      <button type="submit" class="btn btn-primary" :disabled="busy">{{ t("auth.createAccount") }}</button>
    </form>

    <div v-if="error" class="alert alert-error mt-3">{{ error }}</div>
  </AuthCard>
</template>
