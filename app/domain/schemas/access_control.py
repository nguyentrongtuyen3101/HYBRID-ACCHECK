"""
Pydantic schemas – dùng cho API layer và pipeline input/output.
Không chứa business logic.
"""
from typing import Optional
from pydantic import BaseModel, Field

from app.domain.enums.conflict import ConflictType, EffectType, ArtifactSource


# ---------- Request / Input ----------

class PermissionMatrixRowSchema(BaseModel):
    role: str
    action: str
    resource: str
    effect: EffectType
    condition: Optional[str] = None
    scope: Optional[str] = None
    raw_text: Optional[str] = None


class PipelineInput(BaseModel):
    user_story: str
    acceptance_criteria: list[str] = Field(default_factory=list)
    permission_matrix: list[PermissionMatrixRowSchema] = Field(default_factory=list)
    project_id: Optional[str] = None


# ---------- Internal / Response ----------

class ExtractedEntitySchema(BaseModel):
    role: Optional[str] = None
    action: Optional[str] = None
    resource: Optional[str] = None
    effect: Optional[EffectType] = None
    condition: Optional[str] = None
    scope: Optional[str] = None
    source: ArtifactSource = ArtifactSource.USER_STORY
    confidence: float = Field(default=0.0, ge=0.0, le=1.0)


class AlignmentResultSchema(BaseModel):
    extracted: ExtractedEntitySchema
    matched_pm_row: Optional[PermissionMatrixRowSchema] = None
    similarity_score: float = 0.0
    is_matched: bool = False


class ConflictResultSchema(BaseModel):
    conflict_type: ConflictType
    confidence: float = Field(ge=0.0, le=1.0)
    explanation: str
    evidence_us_ac: Optional[str] = None
    evidence_pm: Optional[str] = None
    related_entities: list[ExtractedEntitySchema] = Field(default_factory=list)


class PipelineOutput(BaseModel):
    extracted_entities: list[ExtractedEntitySchema]
    alignment_results: list[AlignmentResultSchema]
    conflicts: list[ConflictResultSchema]
    is_consistent: bool
    summary: str
    pipeline_run_id: Optional[str] = None
