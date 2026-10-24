import { createRouter, createWebHistory } from "vue-router";

import LoginView from "./presentation/views/LoginView.vue";
import ForgotPasswordView from "./presentation/views/ForgotPasswordView.vue";
import ResetPasswordView from "./presentation/views/ResetPasswordView.vue";
import InvitationView from "./presentation/views/InvitationView.vue";
import ProjectsView from "./presentation/views/ProjectsView.vue";
import ProjectView from "./presentation/views/ProjectView.vue";
import HistoryView from "./presentation/views/HistoryView.vue";
import RunView from "./presentation/views/RunView.vue";
import AdminView from "./presentation/views/AdminView.vue";
import ProfileView from "./presentation/views/ProfileView.vue";

/** @param {{ isLoggedIn: () => boolean, isAdmin: () => boolean }} session */
export function createAppRouter(session) {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: "/login", name: "login", component: LoginView, meta: { public: true, guestOnly: true } },
      { path: "/forgot-password", name: "forgot", component: ForgotPasswordView, meta: { public: true, guestOnly: true } },
      { path: "/reset-password", name: "reset", component: ResetPasswordView, meta: { public: true } },
      { path: "/invitations/:token", name: "invitation", component: InvitationView, props: true, meta: { public: true } },
      { path: "/", name: "projects", component: ProjectsView },
      { path: "/projects/:id", name: "project", component: ProjectView, props: true },
      { path: "/history", name: "history", component: HistoryView },
      { path: "/runs/:id", name: "run", component: RunView, props: true },
      { path: "/profile", name: "profile", component: ProfileView },
      { path: "/admin", name: "admin", component: AdminView, meta: { admin: true } },
      { path: "/:pathMatch(.*)*", redirect: "/" },
    ],
  });

  router.beforeEach((to) => {
    if (to.meta.public) {
      return to.meta.guestOnly && session.isLoggedIn() ? { path: "/" } : true;
    }
    if (!session.isLoggedIn()) return { name: "login", query: { next: to.fullPath } };
    if (to.meta.admin && !session.isAdmin()) return { path: "/" };
    return true;
  });
  return router;
}
