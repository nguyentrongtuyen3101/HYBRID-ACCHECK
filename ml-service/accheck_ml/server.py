"""
ML inference service — stateless, không có DB.

API service (Go) gọi vào đây để chạy pipeline Hybrid-ACCheck, sau đó tự lưu
kết quả. Model chỉ được load 1 lần khi process khởi động.

Chạy: accheck-ml-serve  (hoặc uvicorn accheck_ml.server:app --port 8001)
"""
import os
from contextlib import asynccontextmanager

from fastapi import Depends, FastAPI, Header, HTTPException
from loguru import logger

from accheck_ml.core.config import get_settings
from accheck_ml.core.logging import setup_logging
from accheck_ml.domain.schemas.access_control import PipelineInput, PipelineOutput
from accheck_ml.services.pipeline import HybridACCheckPipeline

_state: dict = {"pipeline": None, "ready": False}


def get_pipeline() -> HybridACCheckPipeline:
    if _state["pipeline"] is None:
        _state["pipeline"] = HybridACCheckPipeline()
    return _state["pipeline"]


@asynccontextmanager
async def lifespan(app: FastAPI):
    setup_logging()
    settings = get_settings()
    pipeline = get_pipeline()
    if settings.WARMUP_ON_STARTUP:
        logger.info("[ML] Warming up models...")
        pipeline.warmup()
    _state["ready"] = True
    logger.info(f"[ML] Ready (extractor={pipeline.extractor_name})")
    yield


def verify_internal_token(x_internal_token: str | None = Header(default=None)) -> None:
    expected = get_settings().INTERNAL_TOKEN
    if expected and x_internal_token != expected:
        raise HTTPException(status_code=401, detail="Invalid internal token")


settings = get_settings()
app = FastAPI(
    title=settings.APP_NAME,
    version=settings.APP_VERSION,
    description="Inference service cho Hybrid-ACCheck (DeBERTa + SBERT + Symbolic Reasoner)",
    lifespan=lifespan,
)


@app.get("/health", tags=["Health"])
async def health():
    pipeline = get_pipeline()
    return {
        "status": "ok" if _state["ready"] else "starting",
        "service": settings.APP_NAME,
        "version": settings.APP_VERSION,
        "extractor": pipeline.extractor_name,
        "model_version": pipeline.model_version,
        "sbert_model": settings.SBERT_MODEL_NAME,
    }


@app.post(
    "/v1/analyze",
    response_model=PipelineOutput,
    dependencies=[Depends(verify_internal_token)],
    tags=["Analyze"],
    summary="Chạy pipeline phát hiện xung đột cho 1 user story",
)
async def analyze(payload: PipelineInput) -> PipelineOutput:
    logger.info(
        f"Analyze: US={payload.user_story[:60]!r}, "
        f"AC={len(payload.acceptance_criteria)}, PM={len(payload.permission_matrix)}"
    )
    return await get_pipeline().run(payload)


def main() -> None:
    import uvicorn

    uvicorn.run(
        "accheck_ml.server:app",
        host=os.getenv("HOST", "0.0.0.0"),
        port=int(os.getenv("PORT", "8001")),
        workers=1,
    )


if __name__ == "__main__":
    main()
