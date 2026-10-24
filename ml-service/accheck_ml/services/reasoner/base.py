from abc import ABC, abstractmethod
from accheck_ml.domain.entities.access_control import Conflict
from accheck_ml.services.aligner.base import AlignmentResult
from accheck_ml.services.acrg.graph import AccessControlRequirementGraph


class BaseReasoner(ABC):
    @abstractmethod
    async def detect(
        self,
        alignment_results: list[AlignmentResult],
        acrg: AccessControlRequirementGraph,
    ) -> list[Conflict]:
        ...
