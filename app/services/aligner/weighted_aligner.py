"""
Cross-Artifact Alignment (mục 6.3 tài liệu đề cương).

Ghép nối từng entity trích xuất được từ US/AC với dòng phù hợp nhất trong
Permission Matrix, dựa trên công thức weighted similarity:

    score = w_role * sim(role) + w_action * sim(action) + w_resource * sim(resource)

sim(...) là cosine similarity giữa 2 vector embedding SBERT, KHÔNG còn so
sánh chuỗi tuyệt đối (==) như bản skeleton trước đây.
"""
from loguru import logger

from app.core.config import get_settings
from app.domain.entities.access_control import PermissionMatrixEntry
from app.services.aligner.base import BaseAligner, AlignmentResult
from app.services.normalizer.base import BaseNormalizer, NormalizedEntity


class WeightedAligner(BaseAligner):
    def __init__(self, normalizer: BaseNormalizer):
        settings = get_settings()
        self.normalizer = normalizer
        self.threshold = settings.ALIGNMENT_THRESHOLD
        self.w_role = settings.ROLE_WEIGHT
        self.w_action = settings.ACTION_WEIGHT
        self.w_resource = settings.RESOURCE_WEIGHT
        # Cache embedding của các dòng Permission Matrix, key theo NỘI DUNG
        # (role, action, resource) — không dùng id(row) vì object có thể bị
        # garbage-collected và địa chỉ nhớ bị tái sử dụng cho object khác,
        # gây cache trả nhầm embedding giữa các mẫu khác nhau khi chạy hàng loạt.
        self._pm_embedding_cache: dict[tuple[str, str, str], tuple[list[float], list[float], list[float]]] = {}

    async def _embed_pm_row(
        self, row: PermissionMatrixEntry
    ) -> tuple[list[float], list[float], list[float]]:
        key = (row.role, row.action, row.resource)
        if key in self._pm_embedding_cache:
            return self._pm_embedding_cache[key]

        role_vec = await self.normalizer.embed_text(row.role)
        action_vec = await self.normalizer.embed_text(row.action)
        resource_vec = await self.normalizer.embed_text(row.resource)
        self._pm_embedding_cache[key] = (role_vec, action_vec, resource_vec)
        return role_vec, action_vec, resource_vec

    async def align(
        self,
        normalized_entities: list[NormalizedEntity],
        permission_matrix: list[PermissionMatrixEntry],
    ) -> list[AlignmentResult]:
        logger.info(
            f"[Aligner] Aligning {len(normalized_entities)} entities "
            f"with {len(permission_matrix)} PM rows"
        )

        results: list[AlignmentResult] = []
        for ne in normalized_entities:
            best_score = 0.0
            best_row: PermissionMatrixEntry | None = None

            for row in permission_matrix:
                score = await self._compute_score(ne, row)
                if score > best_score:
                    best_score = score
                    best_row = row

            is_matched = best_score >= self.threshold and best_row is not None
            results.append(
                AlignmentResult(
                    extracted=ne.original,
                    matched_pm_row=best_row if is_matched else None,
                    similarity_score=round(best_score, 4),
                    is_matched=is_matched,
                )
            )
        return results

    async def _compute_score(self, ne: NormalizedEntity, row: PermissionMatrixEntry) -> float:
        role_vec, action_vec, resource_vec = await self._embed_pm_row(row)

        role_sim = (
            self.normalizer.cosine_similarity(ne.role_embedding, role_vec)
            if ne.role_embedding
            else 0.0
        )
        action_sim = (
            self.normalizer.cosine_similarity(ne.action_embedding, action_vec)
            if ne.action_embedding
            else 0.0
        )
        resource_sim = (
            self.normalizer.cosine_similarity(ne.resource_embedding, resource_vec)
            if ne.resource_embedding
            else 0.0
        )

        return (
            self.w_role * role_sim
            + self.w_action * action_sim
            + self.w_resource * resource_sim
        )