import { createApp } from "vue";
import App from "./App.vue";
import { CONTAINER_KEY, createContainer } from "./application/container.js";
import { createAppI18n } from "./presentation/i18n/index.js";
import { createAppRouter } from "./router.js";
import "./assets/app.css";
import "./assets/extra.css";
import "./assets/theme-dark.css";

// Composition root: nối infrastructure + application + presentation.
const i18n = createAppI18n("en");
let router;
const container = createContainer({
  onSessionExpired: () => router.push({ name: "login", query: { next: router.currentRoute.value.fullPath } }),
  onLocaleChange: (locale) => (i18n.global.locale.value = locale),
});
router = createAppRouter(container.session);

createApp(App).provide(CONTAINER_KEY, container).use(i18n).use(router).mount("#app");
