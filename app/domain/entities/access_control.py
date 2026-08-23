"""
Domain Entities – pure Python, không phụ thuộc framework nào.
"""
from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional
from uuid import uuid4

from app.domain.enums.conflict import ConflictType, EffectType, ArtifactSource


@dataclass
class Project:
    name: str
    description: Optional[str] = None
    id: str = field(default_factory=lambda: str(uuid4()))
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


@dataclass
class UserStory:
    content: str
    project_id: Optional[str] = None
    source: ArtifactSource = ArtifactSource.USER_STORY
    id: str = field(default_factory=lambda: str(uuid4()))
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


@dataclass
class AcceptanceCriterion:
    content: str
    user_story_id: str
    id: str = field(default_factory=lambda: str(uuid4()))
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


@dataclass
class PermissionMatrixEntry:
    role: str
    action: str
    resource: str
    effect: EffectType
    condition: Optional[str] = None
    scope: Optional[str] = None
    raw_text: Optional[str] = None
    project_id: Optional[str] = None
    id: str = field(default_factory=lambda: str(uuid4()))
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


@dataclass
class ExtractedEntity:
    role: Optional[str] = None
    action: Optional[str] = None
    resource: Optional[str] = None
    effect: Optional[EffectType] = None
    condition: Optional[str] = None
    scope: Optional[str] = None
    source: ArtifactSource = ArtifactSource.USER_STORY
    confidence: float = 0.0
    id: str = field(default_factory=lambda: str(uuid4()))


@dataclass
class Conflict:
    conflict_type: ConflictType
    explanation: str
    confidence: float = 0.0
    evidence_us_ac: Optional[str] = None
    evidence_pm: Optional[str] = None
    related_entities: list[ExtractedEntity] = field(default_factory=list)
    id: str = field(default_factory=lambda: str(uuid4()))


@dataclass
class PipelineRun:
    user_story_content: str
    acceptance_criteria: list[str] = field(default_factory=list)
    is_consistent: bool = True
    summary: Optional[str] = None
    project_id: Optional[str] = None
    extracted_entities: list[ExtractedEntity] = field(default_factory=list)
    conflicts: list[Conflict] = field(default_factory=list)
    id: str = field(default_factory=lambda: str(uuid4()))
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
