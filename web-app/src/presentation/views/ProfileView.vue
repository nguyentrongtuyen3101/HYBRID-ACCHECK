<script setup>
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../application/container.js";
import { LOCALES, THEMES } from "../../domain/preferences.js";
import { initialsOf } from "../../domain/format.js";
import { useErrorText, useFormat } from "../composables/i18n.js";
import { showToast } from "../composables/ui.js";
import CustomSelect from "../components/common/CustomSelect.vue";

const BIO_MAX = 1000;
const { t } = useI18n();
const { session, preferences } = useContainer();
const errorText = useErrorText();
const { dateTime } = useFormat();

const user = session.user;
const form = reactive({ full_name: "", job_title: "", organization: "", phone: "", bio: "" });
const saving = ref(false);
const pw = reactive({ current: "", next: "", confirm: "" });
const changingPw = ref(false);

watch(
  user,
  (u) => {
    if (!u) return;
    for (const key of Object.keys(form)) form[key] = u[key] || "";
  },
  { immediate: true }
);

const initials = computed(() => initialsOf(user.value));
const dirty = computed(() => Object.keys(form).some((k) => (form[k] || "") !== (user.value?.[k] || "")));
const localeOptions = computed(() => LOCALES.map((l) => ({ value: l, label: t(`languages.full.${l}`) })));

async function saveProfile() {
  saving.value = true;
  try {
    await session.updateProfile({ ...form });
    showToast(t("profile.saved"), "success");
  } catch (e) {
    showToast(errorText(e), "error");
  } finally {
    saving.value = false;
  }
}

async function changePassword() {
  if (pw.next !== pw.confirm) {
    showToast(t("auth.passwordsMismatch"), "error");
    return;
  }
  changingPw.value = true;
  try {
    await session.changePassword(pw.current, pw.next);
    Object.assign(pw, { current: "", next: "", confirm: "" });
    showToast(t("profile.passwordChanged"), "success");
  } catch (e) {
    showToast(errorText(e), "error");
  } finally {
    changingPw.value = false;
  }
}
</script>

<template>
  <section v-if="user">
    <div class="mb-3">
      <h1>{{ t("profile.title") }}</h1>
      <p class="muted">{{ t("profile.subtitle") }}</p>
    </div>

    <div class="profile-grid">
      <div class="card stack" style="align-items: center; text-align: center">
        <div class="avatar">{{ initials }}</div>
        <div>
          <h2 style="margin: 0">{{ user.full_name || user.email }}</h2>
          <div class="muted">{{ user.job_title || "" }}<template v-if="user.job_title && user.organization"> · </template>{{ user.organization || "" }}</div>
        </div>
        <span class="badge" :class="user.role === 'admin' ? 'b-admin' : 'b-user'">{{ t(`enums.systemRole.${user.role}`) }}</span>
        <div class="muted">{{ user.email }}</div>
        <div class="muted">{{ t("profile.memberSince", { date: dateTime(user.created_at) }) }}</div>
        <div v-if="user.last_login_at" class="muted">{{ t("profile.lastLogin", { date: dateTime(user.last_login_at) }) }}</div>
      </div>

      <div class="stack">
        <form class="card" @submit.prevent="saveProfile">
          <h2>{{ t("profile.personalInfo") }}</h2>
          <div class="form-grid mb-3">
            <div>
              <label class="label" for="pf-name">{{ t("profile.fullName") }}</label>
              <input id="pf-name" v-model="form.full_name" class="field" maxlength="255" autocomplete="name" />
            </div>
            <div>
              <label class="label" for="pf-email">{{ t("auth.email") }}</label>
              <input id="pf-email" class="field" :value="user.email" disabled />
            </div>
            <div>
              <label class="label" for="pf-phone">{{ t("profile.phone") }}</label>
              <input id="pf-phone" v-model="form.phone" class="field" maxlength="30" autocomplete="tel" placeholder="+84 …" />
            </div>
            <div>
              <label class="label" for="pf-title">{{ t("profile.jobTitle") }}</label>
              <input id="pf-title" v-model="form.job_title" class="field" maxlength="100" :placeholder="t('profile.jobTitlePlaceholder')" />
            </div>
            <div>
              <label class="label" for="pf-org">{{ t("profile.organization") }}</label>
              <input id="pf-org" v-model="form.organization" class="field" maxlength="150" />
            </div>
          </div>
          <div class="mb-3">
            <label class="label" for="pf-bio">{{ t("profile.bio") }}</label>
            <textarea id="pf-bio" v-model="form.bio" class="field" rows="3" :maxlength="BIO_MAX" :placeholder="t('profile.bioPlaceholder')"></textarea>
            <div class="char-count">{{ form.bio.length }}/{{ BIO_MAX }}</div>
          </div>
          <button type="submit" class="btn btn-primary" :disabled="saving || !dirty">
            {{ saving ? t("common.saving") : t("common.saveChanges") }}
          </button>
        </form>

        <div class="card">
          <h2>{{ t("profile.preferences") }}</h2>
          <p class="muted mb-3">{{ t("profile.preferencesHint") }}</p>
          <div class="form-grid">
            <div>
              <label class="label">{{ t("header.language") }}</label>
              <CustomSelect :model-value="preferences.state.locale" :options="localeOptions" @update:model-value="preferences.setLocale" />
            </div>
            <div>
              <label class="label">{{ t("profile.theme") }}</label>
              <div class="segmented" role="radiogroup">
                <button
                  v-for="theme in THEMES"
                  :key="theme"
                  type="button"
                  role="radio"
                  :aria-checked="preferences.state.theme === theme"
                  :class="{ active: preferences.state.theme === theme }"
                  @click="preferences.setTheme(theme)"
                >
                  {{ t(`profile.themes.${theme}`) }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <form class="card" @submit.prevent="changePassword">
          <h2>{{ t("profile.changePassword") }}</h2>
          <div class="form-grid mb-3">
            <div>
              <label class="label" for="pw-current">{{ t("profile.currentPassword") }}</label>
              <input id="pw-current" v-model="pw.current" class="field" type="password" autocomplete="current-password" required />
            </div>
            <div>
              <label class="label" for="pw-next">{{ t("auth.newPassword") }}</label>
              <input id="pw-next" v-model="pw.next" class="field" type="password" autocomplete="new-password" minlength="8" required />
            </div>
            <div>
              <label class="label" for="pw-confirm">{{ t("auth.confirmPassword") }}</label>
              <input id="pw-confirm" v-model="pw.confirm" class="field" type="password" autocomplete="new-password" required />
            </div>
          </div>
          <button type="submit" class="btn btn-secondary" :disabled="changingPw">{{ t("auth.changePassword") }}</button>
        </form>
      </div>
    </div>
  </section>
</template>
