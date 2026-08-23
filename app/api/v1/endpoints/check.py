from fastapi import APIRouter, Depends, HTTPException
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import get_pipeline
from app.domain.schemas.access_control import PipelineInput, PipelineOutput
from app.infrastructure.db.session import get_db
from app.infrastructure.db.repository import (
    save_pipeline_run,
    list_pipeline_runs,
    get_pipeline_run,
)
from app.services.pipeline import HybridACCheckPipeline

router = APIRouter(prefix="/check", tags=["Conflict Check"])


@router.post("", response_model=PipelineOutput, summary="Phát hiện xung đột phân quyền")
async def check_conflict(
    payload: PipelineInput,
    pipeline: HybridACCheckPipeline = Depends(get_pipeline),
    db: AsyncSession = Depends(get_db),
) -> PipelineOutput:
    logger.info(
        f"Check request: US={payload.user_story[:60]!r}..., "
        f"AC={len(payload.acceptance_criteria)}, PM={len(payload.permission_matrix)}"
    )
    result = await pipeline.run(payload)

    try:
        run_id = await save_pipeline_run(db, payload, result)
        result.pipeline_run_id = run_id
        logger.info(f"Saved pipeline_run id={run_id}")
    except Exception as e:
        logger.warning(f"Could not save to DB: {e}")

    return result


@router.get("/history", summary="Lịch sử các lần check")
async def check_history(
    limit: int = 20,
    db: AsyncSession = Depends(get_db),
):
    runs = await list_pipeline_runs(db, limit=limit)
    return {
        "items": [
            {
                "id": r.id,
                "user_story": r.user_story_content[:120],
                "is_consistent": r.is_consistent,
                "conflict_count": len(r.conflicts),
                "conflict_types": [c.conflict_type for c in r.conflicts],
                "created_at": r.created_at.isoformat() if r.created_at else None,
            }
            for r in runs
        ]
    }


@router.get("/history/{run_id}", summary="Chi tiết một lần check")
async def check_history_detail(
    run_id: str,
    db: AsyncSession = Depends(get_db),
):
    run = await get_pipeline_run(db, run_id)
    if not run:
        raise HTTPException(status_code=404, detail="Run not found")
    return {
        "id": run.id,
        "user_story": run.user_story_content,
        "acceptance_criteria": run.acceptance_criteria,
        "is_consistent": run.is_consistent,
        "summary": run.summary,
        "raw_result": run.raw_result,
        "conflicts": [
            {
                "conflict_type": c.conflict_type,
                "confidence": c.confidence,
                "explanation": c.explanation,
                "evidence_us_ac": c.evidence_us_ac,
                "evidence_pm": c.evidence_pm,
            }
            for c in run.conflicts
        ],
        "extracted_entities": [
            {
                "role": e.role,
                "action": e.action,
                "resource": e.resource,
                "effect": e.effect,
                "scope": e.scope,
                "confidence": e.confidence,
            }
            for e in run.extracted_entities
        ],
        "created_at": run.created_at.isoformat() if run.created_at else None,
    }


@router.get("/examples", summary="Mẫu input demo")
async def get_examples():
    return {
        "examples": [
            {
                "id": "normal",
                "label": "Normal — consistent",
                "user_story": "As a Sales Staff, I want to view assigned customers so that I can manage my accounts.",
                "acceptance_criteria": [
                    "System displays only customers assigned to the logged-in staff."
                ],
                "permission_matrix": [
                    {
                        "role": "Sales Staff",
                        "action": "view",
                        "resource": "Customer",
                        "effect": "Allow",
                        "condition": None,
                        "scope": "assigned",
                    }
                ],
            },
            {
                "id": "effect",
                "label": "Effect Conflict",
                "user_story": "As a Guest, I want to view product catalog so that I can browse items.",
                "acceptance_criteria": [],
                "permission_matrix": [
                    {
                        "role": "Guest",
                        "action": "view",
                        "resource": "Product Catalog",
                        "effect": "Deny",
                        "condition": None,
                        "scope": "all",
                    }
                ],
            },
        ]
    }