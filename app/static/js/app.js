const API = "/api/v1";

const state = {
  view: "projects",
  projectId: null,
  projectDetail: null,
  tab: "stories",
  lastResult: null,
  editUsId: null,
  editPmId: null,
};

const badgeMap = {
  Normal: "b-normal",
  "Missing Permission": "b-missing",
  "Action Conflict": "b-action",
  "Effect Conflict": "b-effect",
  "Scope Conflict": "b-scope",
  "Condition Conflict": "b-condition",
};

async function api(path, options = {}) {
  const res = await fetch(API + path, {
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });
  if (res.status === 204) return null;
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = { detail: text };
  }
  if (!res.ok) {
    const msg = data?.detail
      ? typeof data.detail === "string"
        ? data.detail
        : JSON.stringify(data.detail)
      : res.statusText;
    throw new Error(msg);
  }
  return data;
}

function $(id) {
  return document.getElementById(id);
}

function showToast(message, type = "info", duration = 3200) {
  const container = $("toast-container");
  if (!container) return;
  const el = document.createElement("div");
  el.className = `toast toast-${type}`;
  const iconMap = { success: "✓", error: "!", info: "i" };
  el.innerHTML = `
    <span class="toast-icon">${iconMap[type] || "i"}</span>
    <span class="toast-body">${escapeHtml(message)}</span>
    <button type="button" class="toast-close" aria-label="Close">×</button>
  `;
  const close = () => {
    el.classList.add("toast-out");
    setTimeout(() => el.remove(), 200);
  };
  el.querySelector(".toast-close").onclick = close;
  container.appendChild(el);
  if (duration > 0) setTimeout(close, duration);
}

let _confirmResolve = null;

function showConfirm(message, title = "Confirm") {
  return new Promise((resolve) => {
    _confirmResolve = resolve;
    $("modal-title").textContent = title;
    $("modal-message").textContent = message;
    $("modal-overlay").classList.remove("hidden");
    $("modal-confirm").focus();
  });
}

function closeModal(result) {
  $("modal-overlay").classList.add("hidden");
  if (_confirmResolve) {
    _confirmResolve(result);
    _confirmResolve = null;
  }
}

function setNav() {
  document.querySelectorAll(".nav button").forEach((b) => {
    b.classList.toggle("active", b.dataset.view === state.view);
  });
}

function showView(name) {
  state.view = name;
  setNav();
  $("view-projects").classList.toggle("hidden", name !== "projects");
  $("view-project").classList.toggle("hidden", name !== "project");
  $("view-history").classList.toggle("hidden", name !== "history");
  const run = $("view-run");
  if (run) run.classList.toggle("hidden", name !== "run");
}

function escapeHtml(s) {
  return String(s ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

async function loadProjects() {
  const el = $("projects-list");
  el.innerHTML = "<div class='empty'>Loading…</div>";
  try {
    const items = await api("/projects");
    if (!items.length) {
      el.innerHTML = "<div class='empty'>No projects yet. Create one above.</div>";
      return;
    }
    el.innerHTML = items
      .map(
        (p) => `
      <div class="project-card" data-id="${p.id}">
        <div class="row-between">
          <div>
            <h3>${escapeHtml(p.name)}</h3>
            <p class="meta">${escapeHtml(p.description || "No description")}</p>
            <p class="meta mt-2">${p.user_story_count} stories · ${p.pm_row_count} PM rows · ${p.run_count} runs</p>
          </div>
          <button type="button" class="btn btn-danger btn-sm" data-del-project="${p.id}">Delete</button>
        </div>
      </div>`
      )
      .join("");

    el.querySelectorAll(".project-card").forEach((card) => {
      card.onclick = (e) => {
        if (e.target.closest("[data-del-project]")) return;
        openProject(card.dataset.id);
      };
    });

    el.querySelectorAll("[data-del-project]").forEach((btn) => {
      btn.onclick = async (e) => {
        e.stopPropagation();
        const id = btn.dataset.delProject;
        const ok = await showConfirm(
          "Delete this project and all related data?",
          "Delete project"
        );
        if (!ok) return;
        try {
          await api(`/projects/${id}`, { method: "DELETE" });
          showToast("Project deleted", "success");
          if (state.projectId === id) {
            state.projectId = null;
            showView("projects");
          }
          await loadProjects();
        } catch (err) {
          showToast(err.message, "error");
        }
      };
    });
  } catch (e) {
    el.innerHTML = `<div class="alert alert-error">${escapeHtml(e.message)}</div>`;
    showToast(e.message, "error");
  }
}

async function createProject() {
  const name = $("new-project-name").value.trim();
  const description = $("new-project-desc").value.trim() || null;
  if (!name) {
    showToast("Project name is required", "error");
    return;
  }
  try {
    await api("/projects", {
      method: "POST",
      body: JSON.stringify({ name, description }),
    });
    $("new-project-name").value = "";
    $("new-project-desc").value = "";
    showToast("Project created", "success");
    await loadProjects();
  } catch (e) {
    showToast(e.message, "error");
  }
}

async function openProject(id) {
  state.projectId = id;
  showView("project");
  await refreshProjectDetail();
}

async function refreshProjectDetail() {
  const box = $("project-detail");
  box.innerHTML = "<div class='empty'>Loading…</div>";
  try {
    const data = await api(`/projects/${state.projectId}`);
    state.projectDetail = data;
    const p = data.project;
    $("project-title").textContent = p.name;
    $("project-sub").textContent = p.description || "";
    if ($("edit-project-name")) {
      $("edit-project-name").value = p.name;
      $("edit-project-desc").value = p.description || "";
    }
    renderStories(data.user_stories || []);
    renderMatrix(data.permission_matrix || []);
    fillCheckSelect(data.user_stories || []);
    await loadProjectRuns();
    setTab(state.tab);
    box.classList.add("hidden");
    $("project-body").classList.remove("hidden");
  } catch (e) {
    box.innerHTML = `<div class="alert alert-error">${escapeHtml(e.message)}</div>`;
    $("project-body").classList.add("hidden");
    showToast(e.message, "error");
  }
}

function setTab(tab) {
  state.tab = tab;
  document.querySelectorAll(".tabs button").forEach((b) => {
    b.classList.toggle("active", b.dataset.tab === tab);
  });
  ["stories", "matrix", "check", "runs", "settings"].forEach((t) => {
    const el = $(`tab-${t}`);
    if (el) el.classList.toggle("hidden", t !== tab);
  });
}

function renderStories(list) {
  const el = $("stories-list");
  if (!list.length) {
    el.innerHTML = "<div class='empty'>No user stories. Add one below.</div>";
    return;
  }
  el.innerHTML = `
    <table class="table">
      <thead><tr><th>Content</th><th>AC</th><th></th></tr></thead>
      <tbody>
        ${list
          .map(
            (us) => `
          <tr>
            <td>${escapeHtml(us.content)}</td>
            <td class="muted">${(us.acceptance_criteria || [])
              .map((a) => escapeHtml(a.content))
              .join("<br>") || "—"}</td>
            <td class="row">
              <button class="btn btn-secondary btn-sm" data-edit-us="${us.id}">Edit</button>
              <button class="btn btn-danger" data-del-us="${us.id}">Delete</button>
            </td>
          </tr>`
          )
          .join("")}
      </tbody>
    </table>`;
  el.querySelectorAll("[data-del-us]").forEach((btn) => {
    btn.onclick = async () => {
      const ok = await showConfirm("Delete this user story?", "Delete user story");
      if (!ok) return;
      try {
        await api(`/projects/${state.projectId}/user-stories/${btn.dataset.delUs}`, {
          method: "DELETE",
        });
        showToast("User story deleted", "success");
        await refreshProjectDetail();
      } catch (e) {
        showToast(e.message, "error");
      }
    };
  });
  el.querySelectorAll("[data-edit-us]").forEach((btn) => {
    btn.onclick = () => openEditUs(btn.dataset.editUs);
  });
}

async function addStory() {
  const content = $("us-content").value.trim();
  const acRaw = $("us-ac").value.trim();
  if (!content) {
    showToast("User story content required", "error");
    return;
  }
  const acceptance_criteria = acRaw
    ? acRaw.split("\n").map((s) => s.trim()).filter(Boolean)
    : [];
  try {
    await api(`/projects/${state.projectId}/user-stories`, {
      method: "POST",
      body: JSON.stringify({ content, acceptance_criteria }),
    });
    $("us-content").value = "";
    $("us-ac").value = "";
    showToast("User story added", "success");
    await refreshProjectDetail();
  } catch (e) {
    showToast(e.message, "error");
  }
}

function renderMatrix(list) {
  const el = $("matrix-list");
  if (!list.length) {
    el.innerHTML = "<div class='empty'>No permission matrix rows.</div>";
    return;
  }
  el.innerHTML = `
    <table class="table">
      <thead><tr><th>Role</th><th>Action</th><th>Resource</th><th>Effect</th><th>Scope</th><th>Condition</th><th></th></tr></thead>
      <tbody>
        ${list
          .map(
            (r) => `
          <tr>
            <td>${escapeHtml(r.role)}</td>
            <td>${escapeHtml(r.action)}</td>
            <td>${escapeHtml(r.resource)}</td>
            <td>${escapeHtml(r.effect)}</td>
            <td>${escapeHtml(r.scope || "—")}</td>
            <td>${escapeHtml(r.condition || "—")}</td>
            <td class="row">
              <button class="btn btn-secondary btn-sm" data-edit-pm="${r.id}">Edit</button>
              <button class="btn btn-danger" data-del-pm="${r.id}">Delete</button>
            </td>
          </tr>`
          )
          .join("")}
      </tbody>
    </table>`;
  el.querySelectorAll("[data-del-pm]").forEach((btn) => {
    btn.onclick = async () => {
      const ok = await showConfirm("Delete this permission matrix row?", "Delete row");
      if (!ok) return;
      try {
        await api(`/projects/${state.projectId}/permission-matrix/${btn.dataset.delPm}`, {
          method: "DELETE",
        });
        showToast("Row deleted", "success");
        await refreshProjectDetail();
      } catch (e) {
        showToast(e.message, "error");
      }
    };
  });
  el.querySelectorAll("[data-edit-pm]").forEach((btn) => {
    btn.onclick = () => openEditPm(btn.dataset.editPm);
  });
}

async function addPmRow() {
  const body = {
    role: $("pm-role").value.trim(),
    action: $("pm-action").value.trim(),
    resource: $("pm-resource").value.trim(),
    effect: $("pm-effect").value,
    scope: $("pm-scope").value.trim() || null,
    condition: $("pm-condition").value.trim() || null,
  };
  if (!body.role || !body.action || !body.resource) {
    showToast("Role, Action, Resource are required", "error");
    return;
  }
  try {
    await api(`/projects/${state.projectId}/permission-matrix`, {
      method: "POST",
      body: JSON.stringify(body),
    });
    ["pm-role", "pm-action", "pm-resource", "pm-scope", "pm-condition"].forEach(
      (id) => ($(id).value = "")
    );
    $("pm-effect").value = "Allow";
    showToast("Permission row added", "success");
    await refreshProjectDetail();
  } catch (e) {
    showToast(e.message, "error");
  }
}
function closeAllCustomSelects() {
  document.querySelectorAll(".custom-select.open").forEach((el) => {
    el.classList.remove("open");
    const t = el.querySelector(".custom-select-trigger");
    if (t) t.setAttribute("aria-expanded", "false");
  });
}

function fillCheckSelect(stories) {
  const wrap = $("check-us-wrap");
  const menu = $("check-us-menu");
  const trigger = $("check-us-trigger");
  const hidden = $("check-us");
  if (!wrap || !menu || !trigger || !hidden) return;

  if (!stories.length) {
    hidden.value = "";
    wrap.dataset.value = "";
    trigger.textContent = "No user stories";
    menu.innerHTML = `<div class="custom-select-option placeholder">No user stories</div>`;
    trigger.onclick = (e) => {
      e.stopPropagation();
      closeAllCustomSelects();
      wrap.classList.add("open");
      trigger.setAttribute("aria-expanded", "true");
    };
    return;
  }

  menu.innerHTML = stories
    .map(
      (s, i) =>
        `<div class="custom-select-option${i === 0 ? " selected" : ""}" role="option" data-value="${escapeHtml(String(s.id))}" data-label="${escapeHtml(s.content.slice(0, 80))}">${escapeHtml(s.content.slice(0, 80))}</div>`
    )
    .join("");

  const first = stories[0];
  hidden.value = first.id;
  wrap.dataset.value = first.id;
  trigger.textContent = first.content.slice(0, 80);

  menu.querySelectorAll(".custom-select-option").forEach((opt) => {
    opt.onclick = (e) => {
      e.stopPropagation();
      menu.querySelectorAll(".custom-select-option").forEach((o) => o.classList.remove("selected"));
      opt.classList.add("selected");
      hidden.value = opt.dataset.value;
      wrap.dataset.value = opt.dataset.value;
      trigger.textContent = opt.dataset.label;
      closeAllCustomSelects();
    };
  });

  trigger.onclick = (e) => {
    e.stopPropagation();
    const willOpen = !wrap.classList.contains("open");
    closeAllCustomSelects();
    if (willOpen) {
      wrap.classList.add("open");
      trigger.setAttribute("aria-expanded", "true");
    }
  };
}

function openEditUs(id) {
  const us = (state.projectDetail?.user_stories || []).find((x) => x.id === id);
  if (!us) return;
  state.editUsId = id;
  $("modal-us-content").value = us.content;
  $("modal-us-ac").value = (us.acceptance_criteria || []).map((a) => a.content).join("\n");
  $("modal-us").classList.remove("hidden");
}

async function saveEditUs() {
  try {
    const acRaw = $("modal-us-ac").value.trim();
    await api(`/projects/${state.projectId}/user-stories/${state.editUsId}`, {
      method: "PATCH",
      body: JSON.stringify({
        content: $("modal-us-content").value.trim(),
        acceptance_criteria: acRaw
          ? acRaw.split("\n").map((s) => s.trim()).filter(Boolean)
          : [],
      }),
    });
    $("modal-us").classList.add("hidden");
    showToast("User story updated", "success");
    await refreshProjectDetail();
  } catch (e) {
    showToast(e.message, "error");
  }
}

function openEditPm(id) {
  const r = (state.projectDetail?.permission_matrix || []).find((x) => x.id === id);
  if (!r) return;
  state.editPmId = id;
  $("modal-pm-role").value = r.role;
  $("modal-pm-action").value = r.action;
  $("modal-pm-resource").value = r.resource;
  $("modal-pm-effect").value = r.effect;
  $("modal-pm-scope").value = r.scope || "";
  $("modal-pm-condition").value = r.condition || "";
  $("modal-pm").classList.remove("hidden");
}

async function saveEditPm() {
  try {
    await api(`/projects/${state.projectId}/permission-matrix/${state.editPmId}`, {
      method: "PATCH",
      body: JSON.stringify({
        role: $("modal-pm-role").value.trim(),
        action: $("modal-pm-action").value.trim(),
        resource: $("modal-pm-resource").value.trim(),
        effect: $("modal-pm-effect").value,
        scope: $("modal-pm-scope").value.trim() || null,
        condition: $("modal-pm-condition").value.trim() || null,
      }),
    });
    $("modal-pm").classList.add("hidden");
    showToast("Permission row updated", "success");
    await refreshProjectDetail();
  } catch (e) {
    showToast(e.message, "error");
  }
}

async function saveProjectMeta() {
  try {
    await api(`/projects/${state.projectId}`, {
      method: "PATCH",
      body: JSON.stringify({
        name: $("edit-project-name").value.trim(),
        description: $("edit-project-desc").value.trim() || null,
      }),
    });
    showToast("Project updated", "success");
    await refreshProjectDetail();
  } catch (e) {
    showToast(e.message, "error");
  }
}

async function runProjectCheck() {
  const usId = $("check-us").value;
  const err = $("check-error");
  const out = $("check-result");
  err.classList.add("hidden");
  out.innerHTML = "";
  if (!usId) {
    err.textContent = "Select a user story";
    err.classList.remove("hidden");
    showToast("Select a user story", "error");
    return;
  }
  const btn = $("btn-run-check");
  btn.disabled = true;
  btn.textContent = "Running…";
  try {
    const data = await api(`/projects/${state.projectId}/check`, {
      method: "POST",
      body: JSON.stringify({ user_story_id: usId }),
    });
    state.lastResult = data;
    out.innerHTML = renderResultHtml(data);
    showToast(
      data.is_consistent ? "Analysis complete — consistent" : "Analysis complete — conflicts found",
      data.is_consistent ? "success" : "error"
    );
    await loadProjectRuns();
  } catch (e) {
    err.textContent = e.message;
    err.classList.remove("hidden");
    showToast(e.message, "error");
  } finally {
    btn.disabled = false;
    btn.textContent = "Run analysis";
  }
}

function renderResultHtml(data) {
  const real = (data.conflicts || []).filter((c) => c.conflict_type !== "Normal");
  const badge = data.is_consistent
    ? `<span class="badge b-ok">Consistent</span>`
    : `<span class="badge b-err">Conflicts found</span>`;
  const conflicts = (data.conflicts || [])
    .map((c) => {
      const cls = badgeMap[c.conflict_type] || "b-missing";
      return `<div class="card" style="padding:10px;margin-bottom:8px">
        <div class="row mb-3">
          <span class="badge ${cls}">${escapeHtml(c.conflict_type)}</span>
          <span class="muted">${Number(c.confidence).toFixed(2)}</span>
        </div>
        <div>${escapeHtml(c.explanation || "")}</div>
        ${c.evidence_us_ac ? `<div class="muted mt-2">US/AC: ${escapeHtml(c.evidence_us_ac)}</div>` : ""}
        ${c.evidence_pm ? `<div class="muted">PM: ${escapeHtml(c.evidence_pm)}</div>` : ""}
      </div>`;
    })
    .join("");

  const entities = (data.extracted_entities || [])
    .map(
      (e) =>
        `<div class="mono">${escapeHtml(e.role || "—")} · ${escapeHtml(
          e.action || "—"
        )} · ${escapeHtml(e.resource || "—")} ${
          e.scope ? "· " + escapeHtml(e.scope) : ""
        }</div>`
    )
    .join("") || "<span class='muted'>—</span>";

  return `
    <div class="row mb-3">${badge}
      <span class="muted">${real.length} conflict(s)</span>
      ${
        data.pipeline_run_id
          ? `<span class="muted mono">saved ${data.pipeline_run_id.slice(0, 8)}…</span>`
          : ""
      }
    </div>
    <h2>Conflicts</h2>
    ${conflicts || "<div class='muted'>None</div>"}
    <h2 class="mt-3">Extracted</h2>
    ${entities}
    <h2 class="mt-3">Summary</h2>
    <pre class="summary">${escapeHtml(data.summary || "")}</pre>
  `;
}

async function loadProjectRuns() {
  const el = $("runs-list");
  if (!state.projectId) return;
  try {
    const data = await api(`/projects/${state.projectId}/runs?limit=20`);
    if (!data.items || !data.items.length) {
      el.innerHTML = "<div class='empty'>No runs yet for this project.</div>";
      return;
    }
    el.innerHTML = `
      <table class="table">
        <thead><tr><th>Status</th><th>User story</th><th>When</th><th></th></tr></thead>
        <tbody>
          ${data.items
            .map(
              (r) => `
            <tr>
              <td><span class="badge ${r.is_consistent ? "b-ok" : "b-err"}">${
                r.is_consistent ? "OK" : "Conflict"
              }</span></td>
              <td>${escapeHtml(r.user_story)}</td>
              <td class="muted">${r.created_at ? r.created_at.replace("T", " ").slice(0, 19) : ""}</td>
              <td><button type="button" class="btn btn-secondary btn-sm" data-run="${r.id}">View</button></td>
            </tr>`
            )
            .join("")}
        </tbody>
      </table>`;
    el.querySelectorAll("[data-run]").forEach((btn) => {
      btn.onclick = () => openRunDetail(btn.dataset.run);
    });
  } catch (e) {
    el.innerHTML = `<div class="alert alert-error">${escapeHtml(e.message)}</div>`;
  }
}

async function openRunDetail(runId) {
  showView("run");
  const el = $("run-detail");
  el.innerHTML = "<div class='empty'>Loading…</div>";
  try {
    const data = await api(`/check/history/${runId}`);
    const badge = data.is_consistent
      ? `<span class="badge b-ok">Consistent</span>`
      : `<span class="badge b-err">Conflicts</span>`;
    const conflicts = (data.conflicts || [])
      .map((c) => {
        const cls = badgeMap[c.conflict_type] || "b-missing";
        return `<div class="card" style="padding:12px;margin-bottom:8px">
          <span class="badge ${cls}">${escapeHtml(c.conflict_type)}</span>
          <p class="mt-2">${escapeHtml(c.explanation || "")}</p>
          ${c.evidence_us_ac ? `<p class="muted">US/AC: ${escapeHtml(c.evidence_us_ac)}</p>` : ""}
          ${c.evidence_pm ? `<p class="muted">PM: ${escapeHtml(c.evidence_pm)}</p>` : ""}
        </div>`;
      })
      .join("") || "<div class='muted'>No conflicts</div>";
    const entities = (data.extracted_entities || [])
      .map(
        (e) =>
          `<div class="mono">${escapeHtml(e.role || "—")} · ${escapeHtml(e.action || "—")} · ${escapeHtml(e.resource || "—")}</div>`
      )
      .join("") || "<span class='muted'>—</span>";
    el.innerHTML = `
      <div class="row mb-3">${badge}
        <span class="muted mono">${escapeHtml(data.id)}</span>
        <span class="muted">${data.created_at ? data.created_at.replace("T", " ").slice(0, 19) : ""}</span>
      </div>
      <h2>User story</h2>
      <p>${escapeHtml(data.user_story)}</p>
      <h2 class="mt-3">Conflicts</h2>
      ${conflicts}
      <h2 class="mt-3">Extracted</h2>
      ${entities}
      <h2 class="mt-3">Summary</h2>
      <pre class="summary">${escapeHtml(data.summary || "")}</pre>
    `;
  } catch (e) {
    el.innerHTML = `<div class="alert alert-error">${escapeHtml(e.message)}</div>`;
  }
}

async function loadGlobalHistory() {
  const el = $("global-history");
  el.innerHTML = "<div class='empty'>Loading…</div>";
  try {
    const data = await api("/check/history?limit=30");
    if (!data.items || !data.items.length) {
      el.innerHTML = "<div class='empty'>No saved runs.</div>";
      return;
    }
    el.innerHTML = `
      <table class="table">
        <thead><tr><th>Status</th><th>Types</th><th>User story</th><th>When</th><th></th></tr></thead>
        <tbody>
          ${data.items
            .map(
              (r) => `
            <tr>
              <td><span class="badge ${r.is_consistent ? "b-ok" : "b-err"}">${
                r.is_consistent ? "OK" : "Conflict"
              }</span></td>
              <td class="muted">${escapeHtml((r.conflict_types || []).join(", ") || "—")}</td>
              <td>${escapeHtml(r.user_story)}</td>
              <td class="muted">${r.created_at ? r.created_at.replace("T", " ").slice(0, 19) : ""}</td>
              <td><button type="button" class="btn btn-secondary btn-sm" data-run="${r.id}">View</button></td>
            </tr>`
            )
            .join("")}
        </tbody>
      </table>`;
    el.querySelectorAll("[data-run]").forEach((btn) => {
      btn.onclick = () => openRunDetail(btn.dataset.run);
    });
  } catch (e) {
    el.innerHTML = `<div class="alert alert-error">${escapeHtml(
      e.message
    )} — check DATABASE_URL</div>`;
    showToast(e.message, "error");
  }
}

async function deleteCurrentProject() {
  if (!state.projectId) return;
  const ok = await showConfirm(
    "Delete this project and all related data (stories, matrix, runs)?",
    "Delete project"
  );
  if (!ok) return;
  try {
    await api(`/projects/${state.projectId}`, { method: "DELETE" });
    state.projectId = null;
    showToast("Project deleted", "success");
    showView("projects");
    await loadProjects();
  } catch (e) {
    showToast(e.message, "error");
  }
}

async function checkHealth() {
  try {
    const d = await api("/health");
    $("healthStatus").textContent = `${d.status} · v${d.version || ""}`;
    $("healthStatus").style.color = "#0f9d78";
  } catch {
    $("healthStatus").textContent = "API offline";
    $("healthStatus").style.color = "#c41e3a";
  }
}

function bind() {
  $("nav-projects").onclick = () => {
    showView("projects");
    loadProjects();
  };
  $("nav-history").onclick = () => {
    showView("history");
    loadGlobalHistory();
  };
  $("brand").onclick = () => {
    showView("projects");
    loadProjects();
  };
  $("btn-create-project").onclick = createProject;
  $("btn-add-story").onclick = addStory;
  $("btn-add-pm").onclick = addPmRow;
  $("btn-run-check").onclick = runProjectCheck;
  $("btn-delete-project").onclick = deleteCurrentProject;
  $("btn-back").onclick = () => {
    showView("projects");
    loadProjects();
  };
  if ($("btn-save-project")) $("btn-save-project").onclick = saveProjectMeta;
  if ($("btn-save-us")) $("btn-save-us").onclick = saveEditUs;
  if ($("btn-cancel-us")) $("btn-cancel-us").onclick = () => $("modal-us").classList.add("hidden");
  if ($("btn-save-pm")) $("btn-save-pm").onclick = saveEditPm;
  if ($("btn-cancel-pm")) $("btn-cancel-pm").onclick = () => $("modal-pm").classList.add("hidden");
  if ($("btn-back-run")) {
    $("btn-back-run").onclick = () => {
      if (state.projectId) showView("project");
      else {
        showView("history");
        loadGlobalHistory();
      }
    };
  }
  document.querySelectorAll(".tabs button").forEach((b) => {
    b.onclick = () => setTab(b.dataset.tab);
  });

  if ($("modal-cancel")) $("modal-cancel").onclick = () => closeModal(false);
  if ($("modal-confirm")) $("modal-confirm").onclick = () => closeModal(true);
  if ($("modal-overlay")) {
    $("modal-overlay").onclick = (e) => {
      if (e.target === $("modal-overlay")) closeModal(false);
    };
  }
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      if ($("modal-overlay") && !$("modal-overlay").classList.contains("hidden")) {
        closeModal(false);
      }
      if ($("modal-us") && !$("modal-us").classList.contains("hidden")) {
        $("modal-us").classList.add("hidden");
      }
      if ($("modal-pm") && !$("modal-pm").classList.contains("hidden")) {
        $("modal-pm").classList.add("hidden");
      }
      closeAllCustomSelects();
    }
  });
  document.addEventListener("click", () => {
    closeAllCustomSelects();
  });
}

document.addEventListener("DOMContentLoaded", () => {
  bind();
  checkHealth();
  showView("projects");
  loadProjects();
});