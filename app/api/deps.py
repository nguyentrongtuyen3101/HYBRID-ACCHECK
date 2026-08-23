from functools import lru_cache

from app.services.pipeline import HybridACCheckPipeline


@lru_cache
def get_pipeline() -> HybridACCheckPipeline:
    return HybridACCheckPipeline()