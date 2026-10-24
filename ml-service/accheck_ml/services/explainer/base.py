from abc import ABC, abstractmethod
from accheck_ml.domain.entities.access_control import Conflict


class BaseExplainer(ABC):
    @abstractmethod
    async def explain(self, conflicts: list[Conflict]) -> str:
        ...
