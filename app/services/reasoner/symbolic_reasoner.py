"""
Symbolic Conflict Detection — Hybrid-ACCheck.
"""
from __future__ import annotations

from loguru import logger

from app.domain.enums.conflict import ConflictType
from app.domain.entities.access_control import Conflict
from app.services.aligner.base import AlignmentResult
from app.services.acrg.graph import AccessControlRequirementGraph
from app.services.reasoner.base import BaseReasoner

_ACTION_SYNONYMS = {
    "generate": "create",
    "add": "create",
    "register": "create",
    "build": "create",
    "see": "view",
    "access": "view",
    "find": "view",
    "check": "view",
    "monitor": "view",
    "look": "view",
    "read": "view",
    "modify": "edit",
    "customise": "edit",
    "customize": "edit",
    "set": "edit",
    "maintain": "edit",
    "update": "edit",
    "remove": "delete",
    "cancel": "delete",
    "grant": "approve",
    "authorise": "approve",
    "authorize": "approve",
    "confirm": "approve",
    "decline": "reject",
    "release": "publish",
    "allocate": "assign",
    "download": "export",
    "filter": "search",
    "administer": "manage",
}

_TRIVIAL_CONDITION_KEYWORDS = (
    "logged in",
    "log in",
    "login",
    "authenticated",
    "must be logged",
    "signed in",
    "sign in",
    "session",
)


def _normalize_action(action: str | None) -> str:
    if not action:
        return ""
    a = action.lower().strip().split()[0]
    return _ACTION_SYNONYMS.get(a, a)


def _is_trivial_condition(condition: str | None) -> bool:
    if not condition:
        return True
    c = condition.lower().strip()
    return any(kw in c for kw in _TRIVIAL_CONDITION_KEYWORDS)


class SymbolicReasoner(BaseReasoner):
    async def detect(
        self,
        alignment_results: list[AlignmentResult],
        acrg: AccessControlRequirementGraph,
    ) -> list[Conflict]:
        logger.info("[Reasoner] Running symbolic conflict detection")
        conflicts: list[Conflict] = []

        for ar in alignment_results:
            # Bỏ entity extract không đủ
            if not ar.extracted.role or not ar.extracted.action:
                continue

            if not ar.is_matched:
                conflicts.append(
                    Conflict(
                        conflict_type=ConflictType.MISSING_PERMISSION,
                        confidence=0.9,
                        explanation=(
                            f"US/AC yêu cầu quyền '{ar.extracted.action}' trên "
                            f"'{ar.extracted.resource}' cho role '{ar.extracted.role}' "
                            f"nhưng không tìm thấy dòng tương ứng trong Permission Matrix."
                        ),
                        evidence_us_ac=(
                            f"{ar.extracted.role} | {ar.extracted.action} | {ar.extracted.resource}"
                        ),
                        evidence_pm=None,
                        related_entities=[ar.extracted],
                    )
                )
                continue

            pm = ar.matched_pm_row
            if pm is None:
                continue

            if ar.extracted.effect and pm.effect and ar.extracted.effect != pm.effect:
                conflicts.append(
                    Conflict(
                        conflict_type=ConflictType.EFFECT_CONFLICT,
                        confidence=0.95,
                        explanation=(
                            f"Effect conflict: US/AC yêu cầu {ar.extracted.effect.value} "
                            f"nhưng Permission Matrix cấp {pm.effect.value}."
                        ),
                        evidence_us_ac=str(ar.extracted.effect.value),
                        evidence_pm=str(pm.effect.value),
                        related_entities=[ar.extracted],
                    )
                )

            if (
                ar.extracted.scope
                and pm.scope
                and ar.extracted.scope.lower() != pm.scope.lower()
            ):
                conflicts.append(
                    Conflict(
                        conflict_type=ConflictType.SCOPE_CONFLICT,
                        confidence=0.9,
                        explanation=(
                            f"Scope conflict: US/AC yêu cầu scope '{ar.extracted.scope}' "
                            f"nhưng Permission Matrix cấp scope '{pm.scope}'."
                        ),
                        evidence_us_ac=ar.extracted.scope,
                        evidence_pm=pm.scope,
                        related_entities=[ar.extracted],
                    )
                )

            if pm.condition and not _is_trivial_condition(pm.condition):
                us_cond = (ar.extracted.condition or "").strip()
                pm_cond = (pm.condition or "").strip()
                if us_cond.lower() != pm_cond.lower():
                    conflicts.append(
                        Conflict(
                            conflict_type=ConflictType.CONDITION_CONFLICT,
                            confidence=0.85,
                            explanation=(
                                f"Condition conflict: Permission Matrix yêu cầu điều kiện "
                                f"'{pm.condition}'"
                                + (
                                    f" nhưng US/AC nêu '{ar.extracted.condition}'."
                                    if ar.extracted.condition
                                    else " nhưng US/AC không đề cập điều kiện này."
                                )
                            ),
                            evidence_us_ac=ar.extracted.condition,
                            evidence_pm=pm.condition,
                            related_entities=[ar.extracted],
                        )
                    )

            us_action = _normalize_action(ar.extracted.action)
            pm_action = _normalize_action(pm.action)
            if us_action and pm_action and us_action != pm_action:
                conflicts.append(
                    Conflict(
                        conflict_type=ConflictType.ACTION_CONFLICT,
                        confidence=0.88,
                        explanation=(
                            f"Action conflict: US/AC yêu cầu action '{ar.extracted.action}' "
                            f"nhưng PM cấp '{pm.action}'."
                        ),
                        evidence_us_ac=ar.extracted.action,
                        evidence_pm=pm.action,
                        related_entities=[ar.extracted],
                    )
                )

        if not conflicts:
            conflicts.append(
                Conflict(
                    conflict_type=ConflictType.NORMAL,
                    confidence=0.95,
                    explanation="Không phát hiện xung đột. Yêu cầu phân quyền nhất quán.",
                )
            )

        return conflicts