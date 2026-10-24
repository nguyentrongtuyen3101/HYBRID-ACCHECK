// Mỗi module bọc 1 nhóm endpoint của api-service / notification-service.
const enc = encodeURIComponent;
const qs = (params) => {
  const s = new URLSearchParams(Object.entries(params).filter(([, v]) => v !== undefined && v !== null && v !== ""));
  return s.toString() ? `?${s}` : "";
};

export const createAuthApi = (http) => ({
  login: (email, password) => http.post("/auth/login", { email, password }),
  register: (body) => http.post("/auth/register", body),
  me: () => http.get("/auth/me"),
  updateProfile: (patch) => http.patch("/auth/me", patch),
  changePassword: (current, next) => http.post("/auth/change-password", { current_password: current, new_password: next }),
  forgotPassword: (email) => http.post("/auth/forgot-password", { email }),
  resetPassword: (token, password) => http.post("/auth/reset-password", { token, new_password: password }),
});

export const createProjectApi = (http) => ({
  list: () => http.get("/projects"),
  get: (id) => http.get(`/projects/${id}`),
  create: (name, description) => http.post("/projects", { name, description }),
  update: (id, body) => http.patch(`/projects/${id}`, body),
  remove: (id) => http.del(`/projects/${id}`),
});

export const createMemberApi = (http) => ({
  list: (pid) => http.get(`/projects/${pid}/members`),
  changeRole: (pid, userId, role) => http.patch(`/projects/${pid}/members/${userId}`, { role }),
  remove: (pid, userId) => http.del(`/projects/${pid}/members/${userId}`),
  invitations: (pid) => http.get(`/projects/${pid}/invitations`),
  invite: (pid, email, role) => http.post(`/projects/${pid}/invitations`, { email, role }),
  revoke: (pid, invitationId) => http.del(`/projects/${pid}/invitations/${invitationId}`),
  preview: (token) => http.get(`/invitations/${enc(token)}`),
  accept: (token) => http.post(`/invitations/${enc(token)}/accept`),
});

export const createRequirementApi = (http, baseUrl) => ({
  addStory: (pid, content, criteria) => http.post(`/projects/${pid}/user-stories`, { content, acceptance_criteria: criteria }),
  updateStory: (pid, id, body) => http.patch(`/projects/${pid}/user-stories/${id}`, body),
  deleteStory: (pid, id) => http.del(`/projects/${pid}/user-stories/${id}`),
  addRow: (pid, row) => http.post(`/projects/${pid}/permission-matrix`, row),
  updateRow: (pid, id, row) => http.patch(`/projects/${pid}/permission-matrix/${id}`, row),
  deleteRow: (pid, id) => http.del(`/projects/${pid}/permission-matrix/${id}`),
  importExcel: (pid, file) => {
    const fd = new FormData();
    fd.append("file", file);
    return http.post(`/projects/${pid}/import/excel`, fd);
  },
  templateUrl: `${baseUrl}/projects/template/excel`,
  terms: (pid) => http.get(`/projects/${pid}/terms`),
  createTerm: (pid, term) => http.post(`/projects/${pid}/terms`, term),
  updateTerm: (pid, id, term) => http.put(`/projects/${pid}/terms/${id}`, term),
  deleteTerm: (pid, id) => http.del(`/projects/${pid}/terms/${id}`),
  termSuggestions: (pid) => http.get(`/projects/${pid}/terms/suggestions`),
  versions: (pid) => http.get(`/projects/${pid}/pm-versions`),
  saveVersion: (pid, label) => http.post(`/projects/${pid}/pm-versions`, { label }),
  diffVersions: (pid, from, to) => http.get(`/projects/${pid}/pm-versions/diff${qs({ from, to })}`),
  restoreVersion: (pid, id) => http.post(`/projects/${pid}/pm-versions/${id}/restore`),
});

export const createAnalysisApi = (http) => ({
  checkStory: (pid, storyId) => http.post(`/projects/${pid}/check`, { user_story_id: storyId }),
  projectRuns: (pid, limit = 30) => http.get(`/projects/${pid}/runs${qs({ limit })}`),
  compare: (pid, base, target) => http.get(`/projects/${pid}/runs/compare${qs({ base, target })}`),
  startJob: (pid) => http.post(`/projects/${pid}/analysis-jobs`),
  jobs: (pid) => http.get(`/projects/${pid}/analysis-jobs`),
  job: (pid, id) => http.get(`/projects/${pid}/analysis-jobs/${id}`),
  history: (limit = 30) => http.get(`/check/history${qs({ limit })}`),
  run: (id) => http.get(`/check/history/${id}`),
});

export const createConflictApi = (http) => ({
  list: (pid, filters) => http.get(`/projects/${pid}/conflicts${qs(filters)}`),
  get: (pid, id) => http.get(`/projects/${pid}/conflicts/${id}`),
  update: (pid, id, body) => http.patch(`/projects/${pid}/conflicts/${id}`, body),
  comment: (pid, id, body) => http.post(`/projects/${pid}/conflicts/${id}/comments`, { body }),
});

export const createReportApi = (http) => ({
  report: (pid) => http.get(`/projects/${pid}/report`),
  coverage: (pid) => http.get(`/projects/${pid}/coverage`),
  downloadExcel: (pid) => http.download(`/projects/${pid}/report/excel`),
});

export const createAdminApi = (http) => ({
  stats: () => http.get("/admin/stats"),
  users: (params) => http.get(`/admin/users${qs(params)}`),
  createUser: (body) => http.post("/admin/users", body),
  updateUser: (id, body) => http.patch(`/admin/users/${id}`, body),
  resetPassword: (id, password) => http.post(`/admin/users/${id}/reset-password`, { new_password: password }),
  auditLogs: (params) => http.get(`/admin/audit-logs${qs(params)}`),
  downloadDataset: () => http.download("/admin/dataset/export"),
});

export const createSystemApi = (http) => ({
  health: () => http.get("/health"),
});

export const createNotificationApi = (http) => ({
  list: (params) => http.get(`/notifications${qs(params)}`),
  markRead: (id) => http.post(`/notifications/${id}/read`),
  markAllRead: () => http.post("/notifications/read-all"),
});
