"""
Semantic Normalization dùng Sentence-BERT.

Đúng theo tài liệu đề cương (mục 6.2 - Semantic Normalization):
- Nhận các cụm từ Role/Action/Resource được trích xuất từ Requirement Extractor
- Sinh vector embedding bằng Sentence-BERT (all-MiniLM-L6-v2)
- Dùng cosine similarity để nhận diện các cách nói khác nhau nhưng cùng nghĩa
  (vd "Sales employee" ≈ "Sales staff", "view" ≈ "read")

Model được load 1 lần duy nhất (singleton, lazy-load) để tránh tốn thời gian
load lại mỗi request.
"""
import asyncio
from functools import lru_cache

from loguru import logger

from app.core.config import get_settings
from app.domain.entities.access_control import ExtractedEntity
from app.services.normalizer.base import BaseNormalizer, NormalizedEntity


class SBERTNormalizer(BaseNormalizer):
    def __init__(self, model_name: str | None = None):
        settings = get_settings()
        self.model_name = model_name or settings.SBERT_MODEL_NAME
        self._model = None
        # Cache embedding theo text đã chuẩn hóa để tránh encode lại nhiều lần
        self._embedding_cache: dict[str, list[float]] = {}

    def _get_model(self):
        """Lazy-load model SBERT (chỉ import torch/sentence-transformers khi thực sự cần)."""
        if self._model is None:
            from sentence_transformers import SentenceTransformer

            logger.info(f"[Normalizer] Loading SBERT model: {self.model_name}")
            settings = get_settings()
            self._model = SentenceTransformer(
                self.model_name, cache_folder=settings.MODEL_CACHE_DIR
            )
        return self._model

    async def embed_text(self, text: str) -> list[float]:
        """Sinh embedding cho 1 chuỗi text, có cache để tránh encode trùng lặp."""
        clean_text = (text or "").strip().lower()
        if not clean_text:
            return []

        if clean_text in self._embedding_cache:
            return self._embedding_cache[clean_text]

        # SentenceTransformer.encode là hàm sync (CPU/GPU-bound) → chạy trong
        # thread pool để không block event loop của FastAPI
        loop = asyncio.get_event_loop()
        model = self._get_model()
        vector = await loop.run_in_executor(
            None, lambda: model.encode(clean_text, normalize_embeddings=True).tolist()
        )
        self._embedding_cache[clean_text] = vector
        return vector

    async def normalize(self, entities: list[ExtractedEntity]) -> list[NormalizedEntity]:
        logger.info(f"[Normalizer] Normalizing {len(entities)} entities via SBERT")
        results: list[NormalizedEntity] = []

        for ent in entities:
            normalized_role = (ent.role or "").strip()
            normalized_action = (ent.action or "").strip().lower()
            normalized_resource = (ent.resource or "").strip()

            role_embedding = await self.embed_text(normalized_role) if normalized_role else None
            action_embedding = (
                await self.embed_text(normalized_action) if normalized_action else None
            )
            resource_embedding = (
                await self.embed_text(normalized_resource) if normalized_resource else None
            )

            results.append(
                NormalizedEntity(
                    original=ent,
                    normalized_role=normalized_role,
                    normalized_action=normalized_action,
                    normalized_resource=normalized_resource,
                    role_embedding=role_embedding,
                    action_embedding=action_embedding,
                    resource_embedding=resource_embedding,
                )
            )
        return results

    def load_model(self) -> None:
        """Gọi trước (vd lúc startup) để tránh cold-start lâu ở request đầu tiên."""
        self._get_model()