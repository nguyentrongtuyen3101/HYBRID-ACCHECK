<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useContainer } from "../../../application/container.js";
import NotificationBell from "./NotificationBell.vue";
import UserMenu from "./UserMenu.vue";
import BrandLogo from "./BrandLogo.vue";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { session } = useContainer();

const active = computed(() => {
  if (route.name === "history" || route.name === "run") return "history";
  if (route.name === "admin") return "admin";
  if (route.name === "profile") return "profile";
  return "projects";
});
</script>

<template>
  <header class="app-header">
    <div class="app-header-inner">
      <div class="brand" @click="router.push('/')">
        <BrandLogo />
      </div>
      <nav class="nav">
        <button type="button" :class="{ active: active === 'projects' }" @click="router.push('/')">{{ t("nav.projects") }}</button>
        <button type="button" :class="{ active: active === 'history' }" @click="router.push('/history')">{{ t("nav.history") }}</button>
        <button v-if="session.isAdmin()" type="button" :class="{ active: active === 'admin' }" @click="router.push('/admin')">
          {{ t("nav.admin") }}
        </button>
      </nav>
      <div v-if="session.user.value" class="header-tools">
        <NotificationBell />
        <UserMenu />
      </div>
    </div>
  </header>
</template>
