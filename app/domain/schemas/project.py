from __future__ import annotations
from typing import Optional
from datetime import datetime
from pydantic import BaseModel, Field

from app.domain.enums.conflict import EffectType


class ProjectCreate(BaseModel):
    name: str = Field(min_length=1, max_length=255)
    description: Optional[str] = None


class ProjectUpdate(BaseModel):
    name: Optional[str] = Field(default=None, min_length=1, max_length=255)
    description: Optional[str] = None


class ProjectOut(BaseModel):
    id: str
    name: str
    description: Optional[str] = None
    created_at: Optional[datetime] = None
    user_story_count: int = 0
    pm_row_count: int = 0
    run_count: int = 0


class UserStoryCreate(BaseModel):
    content: str = Field(min_length=1)
    acceptance_criteria: list[str] = Field(default_factory=list)


class UserStoryUpdate(BaseModel):
    content: Optional[str] = None
    acceptance_criteria: Optional[list[str]] = None


class AcceptanceCriterionOut(BaseModel):
    id: str
    content: str


class UserStoryOut(BaseModel):
    id: str
    content: str
    source: str = "User Story"
    acceptance_criteria: list[AcceptanceCriterionOut] = Field(default_factory=list)
    created_at: Optional[datetime] = None


class PMRowCreate(BaseModel):
    role: str
    action: str
    resource: str
    effect: EffectType = EffectType.ALLOW
    condition: Optional[str] = None
    scope: Optional[str] = None
    raw_text: Optional[str] = None


class PMRowUpdate(BaseModel):
    role: Optional[str] = None
    action: Optional[str] = None
    resource: Optional[str] = None
    effect: Optional[EffectType] = None
    condition: Optional[str] = None
    scope: Optional[str] = None


class PMRowOut(BaseModel):
    id: str
    role: str
    action: str
    resource: str
    effect: str
    condition: Optional[str] = None
    scope: Optional[str] = None


class ProjectCheckRequest(BaseModel):
    user_story_id: str