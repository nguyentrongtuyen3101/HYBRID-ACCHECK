<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { LOCALES } from "../../../domain/preferences.js";
import CustomSelect from "./CustomSelect.vue";

const LOGO = "https://i.pinimg.com/736x/1b/67/57/1b67571055528fcd7a09ade1f3a03c1c.jpg";
const { t } = useI18n();
const { preferences } = useContainer();
const localeOptions = computed(() => LOCALES.map((l) => ({ value: l, label: t(`languages.short.${l}`) })));
</script>

<template>
  <div class="auth-wrap">
    <div class="card auth-card">
      <div class="row-between" style="margin-bottom: 18px; align-items: flex-start">
        <div class="brand">
          <div class="brand-mark">
            <img class="brand-logo" :src="LOGO" alt="Logo" />
          </div>
          <div>
            <div class="brand-title">{{ t("app.name") }}</div>
            <div class="brand-sub">{{ t("app.tagline") }}</div>
          </div>
        </div>
        <div class="header-tools">
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
      </div>
      <slot />
    </div>
  </div>
</template>
