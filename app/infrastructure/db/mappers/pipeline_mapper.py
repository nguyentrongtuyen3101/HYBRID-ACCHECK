"""
Mapper: Domain Entity ↔ ORM Model
Chỉ nằm ở Infrastructure layer.
"""
from app.domain.entities.access_control import (
    ExtractedEntity,
    Conflict,
    PipelineRun,
)
from app.domain.enums.conflict import ConflictType, EffectType, ArtifactSource
from app.infrastructure.db.models import (
    PipelineRunORM,
    ExtractedEntityRecordORM,
    ConflictRecordORM,
)


def extracted_entity_to_orm(
    entity: ExtractedEntity, pipeline_run_id: str
) -> ExtractedEntityRecordORM:
    return ExtractedEntityRecordORM(
        id=entity.id,
        pipeline_run_id=pipeline_run_id,
        role=entity.role,
        action=entity.action,
        resource=entity.resource,
        effect=entity.effect.value if entity.effect else None,
        condition=entity.condition,
        scope=entity.scope,
        source=entity.source.value,
        confidence=entity.confidence,
    )


def conflict_to_orm(conflict: Conflict, pipeline_run_id: str) -> ConflictRecordORM:
    return ConflictRecordORM(
        id=conflict.id,
        pipeline_run_id=pipeline_run_id,
        conflict_type=conflict.conflict_type.value,
        confidence=conflict.confidence,
        explanation=conflict.explanation,
        evidence_us_ac=conflict.evidence_us_ac,
        evidence_pm=conflict.evidence_pm,
    )


def pipeline_run_to_orm(run: PipelineRun) -> PipelineRunORM:
    orm = PipelineRunORM(
        id=run.id,
        project_id=run.project_id,
        user_story_content=run.user_story_content,
        acceptance_criteria=run.acceptance_criteria,
        is_consistent=run.is_consistent,
        summary=run.summary,
    )
    return orm


def orm_to_extracted_entity(orm: ExtractedEntityRecordORM) -> ExtractedEntity:
    return ExtractedEntity(
        id=orm.id,
        role=orm.role,
        action=orm.action,
        resource=orm.resource,
        effect=EffectType(orm.effect) if orm.effect else None,
        condition=orm.condition,
        scope=orm.scope,
        source=ArtifactSource(orm.source),
        confidence=orm.confidence,
    )


def orm_to_conflict(orm: ConflictRecordORM) -> Conflict:
    return Conflict(
        id=orm.id,
        conflict_type=ConflictType(orm.conflict_type),
        confidence=orm.confidence,
        explanation=orm.explanation,
        evidence_us_ac=orm.evidence_us_ac,
        evidence_pm=orm.evidence_pm,
    )


def orm_to_pipeline_run(orm: PipelineRunORM) -> PipelineRun:
    return PipelineRun(
        id=orm.id,
        project_id=orm.project_id,
        user_story_content=orm.user_story_content,
        acceptance_criteria=orm.acceptance_criteria or [],
        is_consistent=orm.is_consistent,
        summary=orm.summary,
        extracted_entities=[orm_to_extracted_entity(e) for e in orm.extracted_entities],
        conflicts=[orm_to_conflict(c) for c in orm.conflicts],
        created_at=orm.created_at,
        updated_at=orm.updated_at,
    )
