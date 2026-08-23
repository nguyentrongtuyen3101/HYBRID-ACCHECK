from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Optional

from app.domain.entities.access_control import ExtractedEntity, PermissionMatrixEntry
from app.services.normalizer.base import NormalizedEntity


@dataclass
class AlignmentResult:
    extracted: ExtractedEntity
    matched_pm_row: Optional[PermissionMatrixEntry] = None
    similarity_score: float = 0.0
    is_matched: bool = False


class BaseAligner(ABC):
    @abstractmethod
    async def align(
        self,
        normalized_entities: list[NormalizedEntity],
        permission_matrix: list[PermissionMatrixEntry],
    ) -> list[AlignmentResult]:
        """Ghép nối entity đã normalize (có embedding) với dòng phù hợp nhất trong Permission Matrix."""
        ...