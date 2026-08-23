from abc import ABC, abstractmethod
from app.domain.entities.access_control import ExtractedEntity


class BaseExtractor(ABC):
    """Interface cho Requirement Extractor."""

    @abstractmethod
    async def extract(self, text: str, source: str = "User Story") -> list[ExtractedEntity]:
        """Trích xuất Role, Action, Resource, Effect, Condition, Scope từ text."""
        ...
