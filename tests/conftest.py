"""
Cấu hình dùng chung cho test suite.

Unit test KHÔNG nên phụ thuộc vào việc tải model SBERT thật (nặng, cần
mạng, chậm). Fixture `mock_sbert_embed` (autouse) sẽ patch
SBERTNormalizer.embed_text bằng 1 hàm embedding giả lập nhưng vẫn phản ánh
đúng ngữ nghĩa cơ bản (từ trùng nhau → similarity cao) để test logic
Alignment/Reasoning mà không cần model thật.

Muốn test với model SBERT thật (có tải weight, chạy chậm hơn) → đặt trong
tests/integration/ và không dùng fixture autouse này (hoặc override lại).
"""
import pytest

_VOCAB = [
    "sale", "staff", "employee", "customer", "view", "read", "edit", "update",
    "delete", "create", "approve", "reject", "manage", "assigned", "own",
    "all", "admin", "order", "product", "invoice", "report",
]


def _fake_embed(text: str) -> list[float]:
    words = {w.rstrip("s") for w in (text or "").lower().split()}
    return [1.0 if w in words else 0.0 for w in _VOCAB]


@pytest.fixture(autouse=True)
def mock_sbert_embed(monkeypatch):
    """Tự động patch SBERTNormalizer.embed_text cho mọi unit test."""
    from app.services.normalizer.sbert_normalizer import SBERTNormalizer

    async def _mock_embed_text(self, text: str) -> list[float]:
        return _fake_embed(text)

    monkeypatch.setattr(SBERTNormalizer, "embed_text", _mock_embed_text)