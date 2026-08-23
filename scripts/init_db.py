"""
Script tạo toàn bộ bảng trong database.
Chạy: python scripts/init_db.py
"""
import asyncio
from loguru import logger

from app.infrastructure.db.base import Base
from app.infrastructure.db.session import engine
from app.infrastructure.db import models  # noqa: F401


async def init_models() -> None:
    logger.info("Creating all tables...")
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    logger.info("Done! All tables created successfully.")


if __name__ == "__main__":
    asyncio.run(init_models())
