from loguru import logger
from app.domain.enums.conflict import ConflictType
from app.domain.entities.access_control import Conflict
from app.services.explainer.base import BaseExplainer


class RuleExplainer(BaseExplainer):
    async def explain(self, conflicts: list[Conflict]) -> str:
        logger.info(f"[Explainer] Generating explanation for {len(conflicts)} conflicts")

        if not conflicts:
            return "Không có conflict nào được phát hiện."

        lines = ["=== Báo cáo xung đột phân quyền (Hybrid-ACCheck) ===\n"]
        for i, c in enumerate(conflicts, 1):
            lines.append(f"{i}. [{c.conflict_type.value}] (confidence: {c.confidence:.2f})")
            lines.append(f"   → {c.explanation}")
            if c.evidence_us_ac:
                lines.append(f"   Evidence US/AC: {c.evidence_us_ac}")
            if c.evidence_pm:
                lines.append(f"   Evidence PM  : {c.evidence_pm}")
            lines.append("")

        real_conflicts = [c for c in conflicts if c.conflict_type != ConflictType.NORMAL]
        if real_conflicts:
            lines.append(f"Tổng cộng: {len(real_conflicts)} xung đột cần xử lý.")
        else:
            lines.append("Hệ thống phân quyền nhất quán với yêu cầu nghiệp vụ.")

        return "\n".join(lines)
