from fastapi import APIRouter, Depends, File, HTTPException, UploadFile
from fastapi.responses import Response
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import get_pipeline
from app.domain.enums.conflict import EffectType
from app.domain.schemas.access_control import (
    PipelineInput,
    PipelineOutput,
    PermissionMatrixRowSchema,
)
from app.domain.schemas.project import (
    ProjectCreate,
    ProjectUpdate,
    ProjectOut,
    UserStoryCreate,
    UserStoryUpdate,
    UserStoryOut,
    AcceptanceCriterionOut,
    PMRowCreate,
    PMRowUpdate,
    PMRowOut,
    ProjectCheckRequest,
)
from app.infrastructure.db.session import get_db
from app.infrastructure.db import project_repo as repo
from app.infrastructure.db.repository import save_pipeline_run
from app.infrastructure.db.excel_io import build_template_bytes, parse_import_excel
from app.services.pipeline import HybridACCheckPipeline

router = APIRouter(prefix="/projects", tags=["Projects"])


def _project_out(p, us_c=0, pm_c=0, run_c=0) -> ProjectOut:
    return ProjectOut(
        id=p.id,
        name=p.name,
        description=p.description,
        created_at=p.created_at,
        user_story_count=us_c,
        pm_row_count=pm_c,
        run_count=run_c,
    )


@router.get("/template/excel", summary="Tai file Excel mau US+AC+PM")
async def download_excel_template():
    data = build_template_bytes()
    return Response(
        content=data,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={
            "Content-Disposition": 'attachment; filename="reqsentinel_import_template.xlsx"'
        },
    )


@router.get("", response_model=list[ProjectOut])
async def list_projects(db: AsyncSession = Depends(get_db)):
    items = await repo.list_projects(db)
    result = []
    for p in items:
        us_c, pm_c, run_c = await repo.count_related(db, p.id)
        result.append(_project_out(p, us_c, pm_c, run_c))
    return result


@router.post("", response_model=ProjectOut, status_code=201)
async def create_project(body: ProjectCreate, db: AsyncSession = Depends(get_db)):
    p = await repo.create_project(db, body.name, body.description)
    return _project_out(p)


@router.get("/{project_id}")
async def get_project(project_id: str, db: AsyncSession = Depends(get_db)):
    p = await repo.get_project(db, project_id)
    if not p:
        raise HTTPException(404, "Project not found")
    us_c, pm_c, run_c = await repo.count_related(db, p.id)
    return {
        "project": _project_out(p, us_c, pm_c, run_c),
        "user_stories": [
            UserStoryOut(
                id=us.id,
                content=us.content,
                source=us.source,
                created_at=us.created_at,
                acceptance_criteria=[
                    AcceptanceCriterionOut(id=ac.id, content=ac.content)
                    for ac in us.acceptance_criteria
                ],
            )
            for us in p.user_stories
        ],
        "permission_matrix": [
            PMRowOut(
                id=r.id,
                role=r.role,
                action=r.action,
                resource=r.resource,
                effect=r.effect,
                condition=r.condition,
                scope=r.scope,
            )
            for r in p.permission_matrix_rows
        ],
    }


@router.patch("/{project_id}", response_model=ProjectOut)
async def update_project(
    project_id: str, body: ProjectUpdate, db: AsyncSession = Depends(get_db)
):
    p = await repo.get_project(db, project_id)
    if not p:
        raise HTTPException(404, "Project not found")
    p = await repo.update_project(db, p, body.name, body.description)
    us_c, pm_c, run_c = await repo.count_related(db, p.id)
    return _project_out(p, us_c, pm_c, run_c)


@router.delete("/{project_id}", status_code=204)
async def delete_project(project_id: str, db: AsyncSession = Depends(get_db)):
    ok = await repo.delete_project(db, project_id)
    if not ok:
        raise HTTPException(404, "Project not found")


@router.post("/{project_id}/import/excel", summary="Import US+AC+PM tu Excel")
async def import_excel(
    project_id: str,
    file: UploadFile = File(...),
    db: AsyncSession = Depends(get_db),
):
    p = await repo.get_project(db, project_id)
    if not p:
        raise HTTPException(404, "Project not found")

    if not file.filename or not file.filename.lower().endswith((".xlsx", ".xlsm")):
        raise HTTPException(400, "Chi nhan file .xlsx")

    raw = await file.read()
    if not raw:
        raise HTTPException(400, "File trong")

    parsed = parse_import_excel(raw)
    created_us = 0
    created_pm = 0
    created_ac = 0

    for us in parsed["user_stories"]:
        await repo.add_user_story(
            db, project_id, us["content"], us["acceptance_criteria"]
        )
        created_us += 1
        created_ac += len(us["acceptance_criteria"])

    for row in parsed["permission_matrix"]:
        await repo.add_pm_row(db, project_id, row)
        created_pm += 1

    logger.info(
        f"Import project={project_id} us={created_us} ac={created_ac} pm={created_pm}"
    )
    return {
        "project_id": project_id,
        "user_stories_imported": created_us,
        "acceptance_criteria_imported": created_ac,
        "pm_rows_imported": created_pm,
        "warnings": parsed["errors"],
    }


@router.post("/{project_id}/user-stories", response_model=UserStoryOut, status_code=201)
async def add_user_story(
    project_id: str, body: UserStoryCreate, db: AsyncSession = Depends(get_db)
):
    p = await repo.get_project(db, project_id)
    if not p:
        raise HTTPException(404, "Project not found")
    us = await repo.add_user_story(db, project_id, body.content, body.acceptance_criteria)
    return UserStoryOut(
        id=us.id,
        content=us.content,
        source=us.source,
        created_at=us.created_at,
        acceptance_criteria=[
            AcceptanceCriterionOut(id=ac.id, content=ac.content)
            for ac in us.acceptance_criteria
        ],
    )


@router.patch("/{project_id}/user-stories/{us_id}", response_model=UserStoryOut)
async def update_user_story(
    project_id: str, us_id: str, body: UserStoryUpdate, db: AsyncSession = Depends(get_db)
):
    us = await repo.update_user_story(
        db, project_id, us_id, body.content, body.acceptance_criteria
    )
    if not us:
        raise HTTPException(404, "User story not found")
    return UserStoryOut(
        id=us.id,
        content=us.content,
        source=us.source,
        created_at=us.created_at,
        acceptance_criteria=[
            AcceptanceCriterionOut(id=ac.id, content=ac.content)
            for ac in us.acceptance_criteria
        ],
    )


@router.delete("/{project_id}/user-stories/{us_id}", status_code=204)
async def delete_user_story(
    project_id: str, us_id: str, db: AsyncSession = Depends(get_db)
):
    ok = await repo.delete_user_story(db, project_id, us_id)
    if not ok:
        raise HTTPException(404, "User story not found")


@router.post("/{project_id}/permission-matrix", response_model=PMRowOut, status_code=201)
async def add_pm_row(
    project_id: str, body: PMRowCreate, db: AsyncSession = Depends(get_db)
):
    p = await repo.get_project(db, project_id)
    if not p:
        raise HTTPException(404, "Project not found")
    row = await repo.add_pm_row(
        db,
        project_id,
        {
            "role": body.role,
            "action": body.action,
            "resource": body.resource,
            "effect": body.effect,
            "condition": body.condition,
            "scope": body.scope,
            "raw_text": body.raw_text,
        },
    )
    return PMRowOut(
        id=row.id,
        role=row.role,
        action=row.action,
        resource=row.resource,
        effect=row.effect,
        condition=row.condition,
        scope=row.scope,
    )


@router.patch("/{project_id}/permission-matrix/{row_id}", response_model=PMRowOut)
async def update_pm_row(
    project_id: str, row_id: str, body: PMRowUpdate, db: AsyncSession = Depends(get_db)
):
    row = await repo.update_pm_row(
        db,
        project_id,
        row_id,
        body.model_dump(exclude_unset=True),
    )
    if not row:
        raise HTTPException(404, "PM row not found")
    return PMRowOut(
        id=row.id,
        role=row.role,
        action=row.action,
        resource=row.resource,
        effect=row.effect,
        condition=row.condition,
        scope=row.scope,
    )


@router.delete("/{project_id}/permission-matrix/{row_id}", status_code=204)
async def delete_pm_row(
    project_id: str, row_id: str, db: AsyncSession = Depends(get_db)
):
    ok = await repo.delete_pm_row(db, project_id, row_id)
    if not ok:
        raise HTTPException(404, "PM row not found")


@router.post("/{project_id}/check", response_model=PipelineOutput)
async def check_project_story(
    project_id: str,
    body: ProjectCheckRequest,
    db: AsyncSession = Depends(get_db),
    pipeline: HybridACCheckPipeline = Depends(get_pipeline),
):
    p = await repo.get_project(db, project_id)
    if not p:
        raise HTTPException(404, "Project not found")

    us = next((u for u in p.user_stories if u.id == body.user_story_id), None)
    if not us:
        raise HTTPException(404, "User story not found in this project")

    if not p.permission_matrix_rows:
        raise HTTPException(400, "Project has no permission matrix rows")

    payload = PipelineInput(
        user_story=us.content,
        acceptance_criteria=[ac.content for ac in us.acceptance_criteria],
        permission_matrix=[
            PermissionMatrixRowSchema(
                role=r.role,
                action=r.action,
                resource=r.resource,
                effect=EffectType(r.effect),
                condition=r.condition,
                scope=r.scope,
            )
            for r in p.permission_matrix_rows
        ],
        project_id=project_id,
    )

    logger.info(f"Project check project={project_id} us={body.user_story_id}")
    result = await pipeline.run(payload)

    try:
        run_id = await save_pipeline_run(db, payload, result)
        result.pipeline_run_id = run_id
    except Exception as e:
        logger.warning(f"Could not save run: {e}")

    return result


@router.get("/{project_id}/runs")
async def project_runs(
    project_id: str, limit: int = 20, db: AsyncSession = Depends(get_db)
):
    p = await repo.get_project(db, project_id)
    if not p:
        raise HTTPException(404, "Project not found")
    runs = sorted(p.pipeline_runs, key=lambda r: r.created_at or "", reverse=True)[:limit]
    return {
        "items": [
            {
                "id": r.id,
                "user_story": r.user_story_content[:120],
                "is_consistent": r.is_consistent,
                "summary": (r.summary or "")[:200],
                "created_at": r.created_at.isoformat() if r.created_at else None,
            }
            for r in runs
        ]
    }