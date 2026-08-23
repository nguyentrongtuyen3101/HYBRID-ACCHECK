from abc import ABC, abstractmethod
from app.domain.entities.access_control import Conflict
from app.services.aligner.base import AlignmentResult
from app.services.acrg.graph import AccessControlRequirementGraph


class BaseReasoner(ABC):
    @abstractmethod
    async def detect(
        self,
        alignment_results: list[AlignmentResult],
        acrg: AccessControlRequirementGraph,
    ) -> list[Conflict]:
        ...
