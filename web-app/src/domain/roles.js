// Quyền theo role trong project — khớp domain.Permission bên api-service.
// UI dùng để ẩn/hiện thao tác; server vẫn là nơi kiểm tra quyền thật.

export const PROJECT_ROLES = ["owner", "editor", "viewer"];

export const Permission = Object.freeze({
  View: "view",
  Comment: "comment",
  Edit: "edit",
  RunAnalysis: "run_analysis",
  Triage: "triage",
  ManageMembers: "manage_members",
  ManageProject: "manage_project",
});

const ROLE_PERMISSIONS = {
  viewer: [Permission.View, Permission.Comment],
  editor: [Permission.View, Permission.Comment, Permission.Edit, Permission.RunAnalysis, Permission.Triage],
  owner: Object.values(Permission),
};

export function can(role, permission) {
  return (ROLE_PERMISSIONS[role] || []).includes(permission);
}

// Khớp enum user_role / user_status / term_kind / permission_effect của api-service
export const SYSTEM_ROLES = ["admin", "user"];
export const USER_STATUSES = ["active", "disabled"];
export const TERM_KINDS = ["role", "resource", "action"];
export const EFFECTS = ["Allow", "Deny"];
