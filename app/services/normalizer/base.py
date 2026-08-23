from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Optional

from app.domain.entities.access_control import ExtractedEntity


@dataclass
class NormalizedEntity:
    """Kết quả sau Semantic Normalization (internal)."""
    original: ExtractedEntity
    normalized_role: str
    normalized_action: str
    normalized_resource: str
    # Vector embedding dùng cho so sánh cosine similarity ở bước Alignment
    role_embedding: Optional[list[float]] = field(default=None, repr=False)
    action_embedding: Optional[list[float]] = field(default=None, repr=False)
    resource_embedding: Optional[list[float]] = field(default=None, repr=False)


class BaseNormalizer(ABC):
    @abstractmethod
    async def normalize(self, entities: list[ExtractedEntity]) -> list[NormalizedEntity]:
        """Chuẩn hóa + sinh embedding cho Role/Action/Resource của từng entity."""
        ...

    @abstractmethod
    async def embed_text(self, text: str) -> list[float]:
        """Sinh embedding cho 1 chuỗi text bất kỳ (dùng để embed Permission Matrix)."""
        ...

    @staticmethod
    def cosine_similarity(vec_a: list[float], vec_b: list[float]) -> float:
        """Cosine similarity thuần Python, không phụ thuộc numpy/torch ở interface layer."""
        if not vec_a or not vec_b or len(vec_a) != len(vec_b):
            return 0.0
        dot = sum(a * b for a, b in zip(vec_a, vec_b))
        norm_a = sum(a * a for a in vec_a) ** 0.5
        norm_b = sum(b * b for b in vec_b) ** 0.5
        if norm_a == 0 or norm_b == 0:
            return 0.0
        return dot / (norm_a * norm_b)