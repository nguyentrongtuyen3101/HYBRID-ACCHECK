from __future__ import annotations
from uuid import uuid4

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.infrastructure.db.models import (
    ProjectORM,
    UserStoryORM,
    AcceptanceCriterionORM,
    PermissionMatrixRowORM,
    PipelineRunORM,
)


async def list_projects(session: AsyncSession) -> list[ProjectORM]:
    q = select(ProjectORM).order_by(ProjectORM.created_at.desc())
    return list((await session.execute(q)).scalars().all())


async def get_project(session: AsyncSession, project_id: str) -> ProjectORM | None:
    q = (
        select(ProjectORM)
        .options(
            selectinload(ProjectORM.user_stories).selectinload(UserStoryORM.acceptance_criteria),
            selectinload(ProjectORM.permission_matrix_rows),
            selectinload(ProjectORM.pipeline_runs).selectinload(PipelineRunORM.conflicts),
        )
        .where(ProjectORM.id == project_id)
    )
    return (await session.execute(q)).scalar_one_or_none()


async def create_project(session: AsyncSession, name: str, description: str | None) -> ProjectORM:
    p = ProjectORM(id=str(uuid4()), name=name, description=description)
    session.add(p)
    await session.flush()
    return p


async def update_project(
    session: AsyncSession, project: ProjectORM, name: str | None, description: str | None
) -> ProjectORM:
    if name is not None:
        project.name = name
    if description is not None:
        project.description = description
    await session.flush()
    return project


async def delete_project(session: AsyncSession, project_id: str) -> bool:
    p = await get_project(session, project_id)
    if not p:
        return False
    for us in list(p.user_stories):
        await session.delete(us)
    for row in list(p.permission_matrix_rows):
        await session.delete(row)
    for run in list(p.pipeline_runs):
        await session.delete(run)
    await session.delete(p)
    await session.flush()
    return True


async def add_user_story(
    session: AsyncSession,
    project_id: str,
    content: str,
    acceptance_criteria: list[str],
) -> UserStoryORM:
    us = UserStoryORM(
        id=str(uuid4()),
        project_id=project_id,
        content=content,
        source="User Story",
    )
    session.add(us)
    await session.flush()
    for ac_text in acceptance_criteria:
        if not ac_text.strip():
            continue
        session.add(
            AcceptanceCriterionORM(
                id=str(uuid4()),
                user_story_id=us.id,
                content=ac_text.strip(),
            )
        )
    await session.flush()
    q = (
        select(UserStoryORM)
        .options(selectinload(UserStoryORM.acceptance_criteria))
        .where(UserStoryORM.id == us.id)
    )
    return (await session.execute(q)).scalar_one()


async def update_user_story(
    session: AsyncSession,
    project_id: str,
    us_id: str,
    content: str | None,
    acceptance_criteria: list[str] | None,
) -> UserStoryORM | None:
    q = (
        select(UserStoryORM)
        .options(selectinload(UserStoryORM.acceptance_criteria))
        .where(UserStoryORM.id == us_id, UserStoryORM.project_id == project_id)
    )
    us = (await session.execute(q)).scalar_one_or_none()
    if not us:
        return None
    if content is not None:
        us.content = content
    if acceptance_criteria is not None:
        for ac in list(us.acceptance_criteria):
            await session.delete(ac)
        await session.flush()
        for ac_text in acceptance_criteria:
            if not ac_text.strip():
                continue
            session.add(
                AcceptanceCriterionORM(
                    id=str(uuid4()),
                    user_story_id=us.id,
                    content=ac_text.strip(),
                )
            )
    await session.flush()
    q2 = (
        select(UserStoryORM)
        .options(selectinload(UserStoryORM.acceptance_criteria))
        .where(UserStoryORM.id == us.id)
    )
    return (await session.execute(q2)).scalar_one()


async def delete_user_story(session: AsyncSession, project_id: str, us_id: str) -> bool:
    q = select(UserStoryORM).where(
        UserStoryORM.id == us_id, UserStoryORM.project_id == project_id
    )
    us = (await session.execute(q)).scalar_one_or_none()
    if not us:
        return False
    await session.delete(us)
    await session.flush()
    return True


async def add_pm_row(session: AsyncSession, project_id: str, data: dict) -> PermissionMatrixRowORM:
    effect = data["effect"]
    if not isinstance(effect, str):
        effect = effect.value
    row = PermissionMatrixRowORM(
        id=str(uuid4()),
        project_id=project_id,
        role=data["role"],
        action=data["action"],
        resource=data["resource"],
        effect=effect,
        condition=data.get("condition"),
        scope=data.get("scope"),
        raw_text=data.get("raw_text"),
    )
    session.add(row)
    await session.flush()
    return row


async def update_pm_row(
    session: AsyncSession, project_id: str, row_id: str, data: dict
) -> PermissionMatrixRowORM | None:
    q = select(PermissionMatrixRowORM).where(
        PermissionMatrixRowORM.id == row_id,
        PermissionMatrixRowORM.project_id == project_id,
    )
    row = (await session.execute(q)).scalar_one_or_none()
    if not row:
        return None
    if "role" in data and data["role"] is not None:
        row.role = data["role"]
    if "action" in data and data["action"] is not None:
        row.action = data["action"]
    if "resource" in data and data["resource"] is not None:
        row.resource = data["resource"]
    if "effect" in data and data["effect"] is not None:
        eff = data["effect"]
        row.effect = eff if isinstance(eff, str) else eff.value
    if "condition" in data:
        row.condition = data["condition"]
    if "scope" in data:
        row.scope = data["scope"]
    await session.flush()
    return row


async def delete_pm_row(session: AsyncSession, project_id: str, row_id: str) -> bool:
    q = select(PermissionMatrixRowORM).where(
        PermissionMatrixRowORM.id == row_id,
        PermissionMatrixRowORM.project_id == project_id,
    )
    row = (await session.execute(q)).scalar_one_or_none()
    if not row:
        return False
    await session.delete(row)
    await session.flush()
    return True


async def count_related(session: AsyncSession, project_id: str) -> tuple[int, int, int]:
    us_c = (
        await session.execute(
            select(func.count()).select_from(UserStoryORM).where(UserStoryORM.project_id == project_id)
        )
    ).scalar() or 0
    pm_c = (
        await session.execute(
            select(func.count())
            .select_from(PermissionMatrixRowORM)
            .where(PermissionMatrixRowORM.project_id == project_id)
        )
    ).scalar() or 0
    run_c = (
        await session.execute(
            select(func.count()).select_from(PipelineRunORM).where(PipelineRunORM.project_id == project_id)
        )
    ).scalar() or 0
    return int(us_c), int(pm_c), int(run_c)