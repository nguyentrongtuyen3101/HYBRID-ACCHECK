<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { LOCALES } from "../../../domain/preferences.js";
import CustomSelect from "./CustomSelect.vue";
import BrandLogo from "./BrandLogo.vue";

const { t } = useI18n();
const { preferences } = useContainer();
const localeOptions = computed(() => LOCALES.map((l) => ({ value: l, label: t(`languages.short.${l}`) })));
// Các con số trên nửa trái: [giá trị, khoá nhãn]
const POINTS = [
  ["5", "auth.hero.conflictTypes"],
  ["4", "auth.hero.steps"],
  ["3", "auth.hero.languages"],
];
</script>

<template>
  <div class="auth-wrap">
    <aside class="auth-hero">
      <p class="eyebrow">{{ t("app.tagline") }}</p>
      <div>
        <h1 class="auth-hero-title">Req<span>Sentinel</span></h1>
        <p class="auth-hero-lead">{{ t("auth.hero.lead") }}</p>
      </div>
      <div class="auth-points">
        <div v-for="[value, key] in POINTS" :key="key">
          <b>{{ value }}</b>
          <span>{{ t(key) }}</span>
        </div>
      </div>
    </aside>

    <main class="auth-panel">
      <div class="auth-panel-top">
        <div class="lang-switch">
          <CustomSelect
            :model-value="preferences.state.locale"
            :options="localeOptions"
            :aria-label="t('header.language')"
            @update:model-value="preferences.setLocale"
          />
        </div>
        <button
          type="button"
          class="icon-btn"
          :aria-label="preferences.state.theme === 'dark' ? t('header.themeLight') : t('header.themeDark')"
          @click="preferences.toggleTheme"
        >
          {{ preferences.state.theme === "dark" ? "☀" : "☾" }}
        </button>
      </div>
      <div class="auth-card">
        <div class="brand">
        <BrandLogo />
        </div>
        <slot />
      </div>
    </main>
  </div>
</template>
