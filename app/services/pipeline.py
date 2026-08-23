"""
Orchestrator chính của Hybrid-ACCheck.
Pipeline: Extract (DeBERTa + RuleBased fallback) → Normalize (SBERT)
          → Align (weighted) → ACRG → Reason → Explain
"""
from loguru import logger

from app.domain.entities.access_control import (
    ExtractedEntity,
    PermissionMatrixEntry,
)
from app.domain.enums.conflict import ArtifactSource, ConflictType
from app.domain.schemas.access_control import (
    PipelineInput,
    PipelineOutput,
    ExtractedEntitySchema,
    AlignmentResultSchema,
    ConflictResultSchema,
    PermissionMatrixRowSchema,
)
from app.services.extractor.deberta_extractor import DeBERTaExtractor
from app.services.extractor.rule_based_extractor import RuleBasedExtractor
from app.services.normalizer.sbert_normalizer import SBERTNormalizer
from app.services.aligner.weighted_aligner import WeightedAligner
from app.services.acrg.graph import AccessControlRequirementGraph
from app.services.reasoner.symbolic_reasoner import SymbolicReasoner
from app.services.explainer.rule_explainer import RuleExplainer


class HybridACCheckPipeline:
    def __init__(self):
        self.extractor = DeBERTaExtractor()
        self.fallback_extractor = RuleBasedExtractor()
        self.normalizer = SBERTNormalizer()
        self.aligner = WeightedAligner(normalizer=self.normalizer)
        self.reasoner = SymbolicReasoner()
        self.explainer = RuleExplainer()

    async def _extract_with_fallback(self, text: str, source: str) -> list[ExtractedEntity]:
        """DeBERTa chính; nếu thiếu Role/Action/Resource thì fallback RuleBased."""
        entities = await self.extractor.extract(text, source=source)
        if entities:
            e = entities[0]
            if e.role and e.action and e.resource:
                return entities
        fb = await self.fallback_extractor.extract(text, source=source)
        if fb:
            logger.info("[Pipeline] Fallback to RuleBasedExtractor")
            return fb
        return entities or []

    async def run(self, payload: PipelineInput) -> PipelineOutput:
        logger.info("[Pipeline] Starting Hybrid-ACCheck")

        # 1. Extract
        entities: list[ExtractedEntity] = []
        us_entities = await self._extract_with_fallback(
            payload.user_story, source=ArtifactSource.USER_STORY.value
        )
        entities.extend(us_entities)

        for ac in payload.acceptance_criteria:
            ac_entities = await self._extract_with_fallback(
                ac, source=ArtifactSource.ACCEPTANCE_CRITERIA.value
            )
            entities.extend(ac_entities)

        # Chỉ giữ entity đủ Role + Action + Resource (bỏ AC dạng mô tả UI)
        entities = [
            e for e in entities
            if e.role and e.action and e.resource
        ]
        logger.info(f"[Pipeline] Usable entities after filter: {len(entities)}")

        # 2. Normalize
        normalized_entities = await self.normalizer.normalize(entities)

        # 3. Convert PM schema → Domain Entity
        pm_entries = [
            PermissionMatrixEntry(
                role=row.role,
                action=row.action,
                resource=row.resource,
                effect=row.effect,
                condition=row.condition,
                scope=row.scope,
                raw_text=row.raw_text,
            )
            for row in payload.permission_matrix
        ]

        # 4. Align
        alignment_results = await self.aligner.align(normalized_entities, pm_entries)

        # 5. ACRG
        acrg = AccessControlRequirementGraph()
        acrg.build_from_alignment(alignment_results, pm_entries)

        # 6. Reason
        conflicts = await self.reasoner.detect(alignment_results, acrg)

        # 7. Explain
        summary = await self.explainer.explain(conflicts)

        is_consistent = all(c.conflict_type == ConflictType.NORMAL for c in conflicts)

        logger.info(f"[Pipeline] Done. Consistent={is_consistent}, Conflicts={len(conflicts)}")

        return PipelineOutput(
            extracted_entities=[
                ExtractedEntitySchema(
                    role=e.role,
                    action=e.action,
                    resource=e.resource,
                    effect=e.effect,
                    condition=e.condition,
                    scope=e.scope,
                    source=e.source,
                    confidence=e.confidence,
                )
                for e in entities
            ],
            alignment_results=[
                AlignmentResultSchema(
                    extracted=ExtractedEntitySchema(
                        role=ar.extracted.role,
                        action=ar.extracted.action,
                        resource=ar.extracted.resource,
                        effect=ar.extracted.effect,
                        condition=ar.extracted.condition,
                        scope=ar.extracted.scope,
                        source=ar.extracted.source,
                        confidence=ar.extracted.confidence,
                    ),
                    matched_pm_row=PermissionMatrixRowSchema(
                        role=ar.matched_pm_row.role,
                        action=ar.matched_pm_row.action,
                        resource=ar.matched_pm_row.resource,
                        effect=ar.matched_pm_row.effect,
                        condition=ar.matched_pm_row.condition,
                        scope=ar.matched_pm_row.scope,
                    )
                    if ar.matched_pm_row
                    else None,
                    similarity_score=ar.similarity_score,
                    is_matched=ar.is_matched,
                )
                for ar in alignment_results
            ],
            conflicts=[
                ConflictResultSchema(
                    conflict_type=c.conflict_type,
                    confidence=c.confidence,
                    explanation=c.explanation,
                    evidence_us_ac=c.evidence_us_ac,
                    evidence_pm=c.evidence_pm,
                    related_entities=[
                        ExtractedEntitySchema(
                            role=e.role,
                            action=e.action,
                            resource=e.resource,
                            effect=e.effect,
                            condition=e.condition,
                            scope=e.scope,
                            source=e.source,
                            confidence=e.confidence,
                        )
                        for e in c.related_entities
                    ],
                )
                for c in conflicts
            ],
            is_consistent=is_consistent,
            summary=summary,
        )