// Loại conflict và vòng đời xử lý — khớp domain.IssueStatus bên api-service.

export const CONFLICT_TYPES = [
  "Missing Permission",
  "Action Conflict",
  "Effect Conflict",
  "Scope Conflict",
  "Condition Conflict",
];

export const CONFLICT_BADGES = {
  Normal: "b-normal",
  "Missing Permission": "b-missing",
  "Action Conflict": "b-action",
  "Effect Conflict": "b-effect",
  "Scope Conflict": "b-scope",
  "Condition Conflict": "b-condition",
};

/** Vòng đời xử lý conflict (nhãn hiển thị nằm trong i18n: enums.issueStatus.*). */
export const ISSUE_STATUSES = ["open", "acknowledged", "resolved", "false_positive"];

export const ISSUE_STATUS_BADGES = {
  open: "b-err",
  acknowledged: "b-missing",
  resolved: "b-ok",
  false_positive: "b-disabled",
};

const TRANSITIONS = {
  open: ["acknowledged", "resolved", "false_positive"],
  acknowledged: ["open", "resolved", "false_positive"],
  resolved: ["open"],
  false_positive: ["open"],
};

export function nextStatuses(status) {
  return TRANSITIONS[status] || [];
}

export const JOB_STATUSES = ["queued", "running", "completed", "failed"];

export const JOB_STATUS_BADGES = { queued: "b-missing", running: "b-missing", completed: "b-ok", failed: "b-err" };
