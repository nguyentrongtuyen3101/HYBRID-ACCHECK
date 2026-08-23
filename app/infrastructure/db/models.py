"""
SQLAlchemy ORM Models – chỉ nằm ở Infrastructure layer.
Không chứa business logic.
"""
from datetime import datetime
from typing import Optional
from uuid import uuid4

from sqlalchemy import (
    String,
    Text,
    Float,
    Boolean,
    ForeignKey,
    JSON,
    DateTime,
    func,
)
from sqlalchemy.orm import Mapped, mapped_column, relationship
from sqlalchemy.dialects.postgresql import UUID

from app.infrastructure.db.base import Base


class ProjectORM(Base):
    __tablename__ = "projects"

    id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), primary_key=True, default=lambda: str(uuid4())
    )
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    description: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False
    )

    user_stories: Mapped[list["UserStoryORM"]] = relationship(back_populates="project")
    permission_matrix_rows: Mapped[list["PermissionMatrixRowORM"]] = relationship(
        back_populates="project"
    )
    pipeline_runs: Mapped[list["PipelineRunORM"]] = relationship(back_populates="project")


class UserStoryORM(Base):
    __tablename__ = "user_stories"

    id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), primary_key=True, default=lambda: str(uuid4())
    )
    project_id: Mapped[Optional[str]] = mapped_column(
        UUID(as_uuid=False), ForeignKey("projects.id"), nullable=True
    )
    content: Mapped[str] = mapped_column(Text, nullable=False)
    source: Mapped[str] = mapped_column(String(50), nullable=False, default="User Story")
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False
    )

    project: Mapped[Optional["ProjectORM"]] = relationship(back_populates="user_stories")
    acceptance_criteria: Mapped[list["AcceptanceCriterionORM"]] = relationship(
        back_populates="user_story", cascade="all, delete-orphan"
    )


class AcceptanceCriterionORM(Base):
    __tablename__ = "acceptance_criteria"

    id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), primary_key=True, default=lambda: str(uuid4())
    )
    user_story_id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), ForeignKey("user_stories.id"), nullable=False
    )
    content: Mapped[str] = mapped_column(Text, nullable=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False
    )

    user_story: Mapped["UserStoryORM"] = relationship(back_populates="acceptance_criteria")


class PermissionMatrixRowORM(Base):
    __tablename__ = "permission_matrix_rows"

    id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), primary_key=True, default=lambda: str(uuid4())
    )
    project_id: Mapped[Optional[str]] = mapped_column(
        UUID(as_uuid=False), ForeignKey("projects.id"), nullable=True
    )
    role: Mapped[str] = mapped_column(String(150), nullable=False)
    action: Mapped[str] = mapped_column(String(100), nullable=False)
    resource: Mapped[str] = mapped_column(String(150), nullable=False)
    effect: Mapped[str] = mapped_column(String(20), nullable=False)
    condition: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    scope: Mapped[Optional[str]] = mapped_column(String(100), nullable=True)
    raw_text: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False
    )

    project: Mapped[Optional["ProjectORM"]] = relationship(
        back_populates="permission_matrix_rows"
    )


class PipelineRunORM(Base):
    __tablename__ = "pipeline_runs"

    id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), primary_key=True, default=lambda: str(uuid4())
    )
    project_id: Mapped[Optional[str]] = mapped_column(
        UUID(as_uuid=False), ForeignKey("projects.id"), nullable=True
    )
    user_story_content: Mapped[str] = mapped_column(Text, nullable=False)
    acceptance_criteria: Mapped[Optional[list]] = mapped_column(JSON, nullable=True)
    is_consistent: Mapped[bool] = mapped_column(Boolean, default=True)
    summary: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    raw_result: Mapped[Optional[dict]] = mapped_column(JSON, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False
    )

    project: Mapped[Optional["ProjectORM"]] = relationship(back_populates="pipeline_runs")
    conflicts: Mapped[list["ConflictRecordORM"]] = relationship(
        back_populates="pipeline_run", cascade="all, delete-orphan"
    )
    extracted_entities: Mapped[list["ExtractedEntityRecordORM"]] = relationship(
        back_populates="pipeline_run", cascade="all, delete-orphan"
    )


class ExtractedEntityRecordORM(Base):
    __tablename__ = "extracted_entities"

    id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), primary_key=True, default=lambda: str(uuid4())
    )
    pipeline_run_id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), ForeignKey("pipeline_runs.id"), nullable=False
    )
    role: Mapped[Optional[str]] = mapped_column(String(150), nullable=True)
    action: Mapped[Optional[str]] = mapped_column(String(100), nullable=True)
    resource: Mapped[Optional[str]] = mapped_column(String(150), nullable=True)
    effect: Mapped[Optional[str]] = mapped_column(String(20), nullable=True)
    condition: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    scope: Mapped[Optional[str]] = mapped_column(String(100), nullable=True)
    source: Mapped[str] = mapped_column(String(50), nullable=False)
    confidence: Mapped[float] = mapped_column(Float, default=0.0)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False
    )

    pipeline_run: Mapped["PipelineRunORM"] = relationship(
        back_populates="extracted_entities"
    )


class ConflictRecordORM(Base):
    __tablename__ = "conflict_records"

    id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), primary_key=True, default=lambda: str(uuid4())
    )
    pipeline_run_id: Mapped[str] = mapped_column(
        UUID(as_uuid=False), ForeignKey("pipeline_runs.id"), nullable=False
    )
    conflict_type: Mapped[str] = mapped_column(String(50), nullable=False)
    confidence: Mapped[float] = mapped_column(Float, default=0.0)
    explanation: Mapped[str] = mapped_column(Text, nullable=False)
    evidence_us_ac: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    evidence_pm: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False
    )

    pipeline_run: Mapped["PipelineRunORM"] = relationship(back_populates="conflicts")
