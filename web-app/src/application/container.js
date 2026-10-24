// Composition root của FE: nối infrastructure (HTTP, storage, socket) vào application.
// Presentation chỉ lấy dependency qua useContainer(), không tự tạo client/fetch.
import { inject } from "vue";
import {
  createAdminApi,
  createAnalysisApi,
  createAuthApi,
  createConflictApi,
  createMemberApi,
  createNotificationApi,
  createProjectApi,
  createReportApi,
  createRequirementApi,
  createSystemApi,
} from "../infrastructure/api/index.js";
import { createHttpClient } from "../infrastructure/http/httpClient.js";
import { connectNotificationSocket } from "../infrastructure/realtime/notificationSocket.js";
import { browserEnvironment, documentAppearance, preferenceStorage } from "../infrastructure/browser/environment.js";
import { sessionStorageAdapter } from "../infrastructure/storage/sessionStorage.js";
import { createNotificationCenter } from "./notifications.js";
import { createPreferences } from "./preferences.js";
import { createSession } from "./session.js";

export const CONTAINER_KEY = Symbol("container");

/**
 * @param {{ onSessionExpired: () => void, onLocaleChange: (locale: string) => void }} hooks
 *   onLocaleChange: presentation đổi ngôn ngữ của i18n
 */
export function createContainer({ onSessionExpired, onLocaleChange }) {
  const apiBase = import.meta.env.VITE_API_BASE || "/api/v1";
  const notifyBase = import.meta.env.VITE_NOTIFY_API_BASE || "/api";

  let session;
  const httpOpts = {
    getToken: () => session?.token(),
    onUnauthorized: () => {
      session.logout();
      onSessionExpired();
    },
  };
  const http = createHttpClient({ baseUrl: apiBase, ...httpOpts });
  const notifyHttp = createHttpClient({ baseUrl: notifyBase, ...httpOpts });

  const api = {
    auth: createAuthApi(http),
    projects: createProjectApi(http),
    members: createMemberApi(http),
    requirements: createRequirementApi(http, apiBase),
    analysis: createAnalysisApi(http),
    conflicts: createConflictApi(http),
    reports: createReportApi(http),
    admin: createAdminApi(http),
    system: createSystemApi(http),
  };
  session = createSession({ authApi: api.auth, storage: sessionStorageAdapter });
  const notifications = createNotificationCenter({
    notificationApi: createNotificationApi(notifyHttp),
    connect: connectNotificationSocket,
  });
  const preferences = createPreferences({
    storage: preferenceStorage,
    environment: browserEnvironment,
    apply: {
      locale: (l) => {
        documentAppearance.locale(l);
        onLocaleChange(l);
      },
      theme: documentAppearance.theme,
    },
  });
  return { api, session, notifications, preferences };
}

export function useContainer() {
  return inject(CONTAINER_KEY);
}
