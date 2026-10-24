from enum import Enum


class ConflictType(str, Enum):
    """Các loại conflict chính theo đề tài."""
    NORMAL = "Normal"
    MISSING_PERMISSION = "Missing Permission"
    ACTION_CONFLICT = "Action Conflict"
    EFFECT_CONFLICT = "Effect Conflict"          # Allow/Deny
    SCOPE_CONFLICT = "Scope Conflict"
    CONDITION_CONFLICT = "Condition Conflict"


class EffectType(str, Enum):
    ALLOW = "Allow"
    DENY = "Deny"


class ArtifactSource(str, Enum):
    """Nguồn xuất phát của một entity trong ACRG."""
    USER_STORY = "User Story"
    ACCEPTANCE_CRITERIA = "Acceptance Criteria"
    PERMISSION_MATRIX = "Permission Matrix"
