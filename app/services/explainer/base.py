from abc import ABC, abstractmethod
from app.domain.entities.access_control import Conflict


class BaseExplainer(ABC):
    @abstractmethod
    async def explain(self, conflicts: list[Conflict]) -> str:
        ...
