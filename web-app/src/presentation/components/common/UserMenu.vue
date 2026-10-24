<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContainer } from "../../../application/container.js";
import { initialsOf } from "../../../domain/format.js";
import { LOCALES, THEMES } from "../../../domain/preferences.js";

const { t } = useI18n();
const router = useRouter();
const { api, session, preferences } = useContainer();
const user = session.user;
const open = ref(false);

// Trạng thái hệ thống: chỉ báo trên avatar khi có sự cố, chi tiết nằm trong menu
const health = ref({ status: "loading" });
const healthText = computed(
  () =>
    ({
      loading: "…",
      ok: t("header.statusOk"),
      degraded: t("header.mlOfflineHint"),
      offline: t("header.apiOffline"),
    })[health.value.status]
);
const unhealthy = computed(() => health.value.status === "degraded" || health.value.status === "offline");

async function loadHealth() {
  try {
    const d = await api.system.health();
    health.value = { status: d.status === "ok" ? "ok" : "degraded", version: d.version };
  } catch {
    health.value = { status: "offline" };
  }
}

function go(path) {
  open.value = false;
  router.push(path);
}

function logout() {
  open.value = false;
  session.logout();
  router.push("/login");
}

// Đóng khi click ra ngoài (pha capture để menu chuông mở ra cũng đóng menu này)
const root = ref(null);
const onOutside = (e) => root.value && !root.value.contains(e.target) && (open.value = false);
const onKey = (e) => e.key === "Escape" && (open.value = false);
onMounted(() => {
  loadHealth();
  document.addEventListener("click", onOutside, true);
  document.addEventListener("keydown", onKey);
});
onBeforeUnmount(() => {
  document.removeEventListener("click", onOutside, true);
  document.removeEventListener("keydown", onKey);
});
</script>

<template>
  <div ref="root" class="account">
    <button
      type="button"
      class="account-btn"
      :class="{ active: open }"
      :aria-label="t('header.accountMenu')"
      :aria-expanded="open"
      :title="user.full_name || user.email"
      @click="open = !open"
    >
      {{ initialsOf(user) }}
      <span v-if="unhealthy" class="account-alert" :title="healthText"></span>
    </button>

    <div v-if="open" class="account-menu" role="menu">
      <div class="account-head">
        <div class="account-avatar">{{ initialsOf(user) }}</div>
        <div class="account-id">
          <div class="account-name">{{ user.full_name || user.email }}</div>
          <div class="account-email">{{ user.email }}</div>
        </div>
        <span class="badge" :class="session.isAdmin() ? 'b-admin' : 'b-user'">{{ t(`enums.systemRole.${user.role}`) }}</span>
      </div>

      <button type="button" class="account-item" role="menuitem" @click="go('/profile')">{{ t("header.profile") }}</button>

      <div class="account-row">
        <span>{{ t("header.language") }}</span>
        <div class="segmented segmented-sm" role="radiogroup">
          <button
            v-for="l in LOCALES"
            :key="l"
            type="button"
            role="radio"
            :aria-checked="preferences.state.locale === l"
            :title="t(`languages.full.${l}`)"
            :class="{ active: preferences.state.locale === l }"
            @click="preferences.setLocale(l)"
          >
            {{ t(`languages.short.${l}`) }}
          </button>
        </div>
      </div>
      <div class="account-row">
        <span>{{ t("profile.theme") }}</span>
        <div class="segmented segmented-sm" role="radiogroup">
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

      <div class="account-status" :class="health.status">
        <span class="status-dot"></span>
        <span>{{ healthText }}</span>
        <span v-if="health.version" class="muted">v{{ health.version }}</span>
      </div>

      <button type="button" class="account-item account-logout" role="menuitem" @click="logout">{{ t("header.logout") }}</button>
    </div>
  </div>
</template>
