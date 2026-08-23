"""Persist pipeline results to Postgres."""
from __future__ import annotations

from uuid import uuid4

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.domain.schemas.access_control import PipelineInput, PipelineOutput
from app.infrastructure.db.models import (
    ConflictRecordORM,
    ExtractedEntityRecordORM,
    PipelineRunORM,
)


async def save_pipeline_run(
    session: AsyncSession,
    payload: PipelineInput,
    result: PipelineOutput,
) -> str:
    run_id = str(uuid4())

    run = PipelineRunORM(
        id=run_id,
        project_id=payload.project_id,
        user_story_content=payload.user_story,
        acceptance_criteria=payload.acceptance_criteria or [],
        is_consistent=result.is_consistent,
        summary=result.summary,
        raw_result={
            "permission_matrix": [r.model_dump(mode="json") for r in payload.permission_matrix],
            "alignment_results": [a.model_dump(mode="json") for a in result.alignment_results],
        },
    )
    session.add(run)

    for e in result.extracted_entities:
        session.add(
            ExtractedEntityRecordORM(
                id=str(uuid4()),
                pipeline_run_id=run_id,
                role=e.role,
                action=e.action,
                resource=e.resource,
                effect=e.effect.value if e.effect else None,
                condition=e.condition,
                scope=e.scope,
                source=e.source.value if e.source else "User Story",
                confidence=e.confidence,
            )
        )

    for c in result.conflicts:
        if c.conflict_type.value == "Normal":
            continue
        session.add(
            ConflictRecordORM(
                id=str(uuid4()),
                pipeline_run_id=run_id,
                conflict_type=c.conflict_type.value,
                confidence=c.confidence,
                explanation=c.explanation,
                evidence_us_ac=c.evidence_us_ac,
                evidence_pm=c.evidence_pm,
            )
        )

    await session.flush()
    return run_id


async def list_pipeline_runs(session: AsyncSession, limit: int = 20) -> list[PipelineRunORM]:
    q = (
        select(PipelineRunORM)
        .options(selectinload(PipelineRunORM.conflicts))
        .order_by(PipelineRunORM.created_at.desc())
        .limit(limit)
    )
    rows = (await session.execute(q)).scalars().all()
    return list(rows)


async def get_pipeline_run(session: AsyncSession, run_id: str) -> PipelineRunORM | None:
    q = (
        select(PipelineRunORM)
        .options(
            selectinload(PipelineRunORM.conflicts),
            selectinload(PipelineRunORM.extracted_entities),
        )
        .where(PipelineRunORM.id == run_id)
    )
    return (await session.execute(q)).scalar_one_or_none()