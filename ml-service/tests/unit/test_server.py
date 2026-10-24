from fastapi.testclient import TestClient

from accheck_ml.core.config import get_settings


def _client(monkeypatch, token: str = "") -> TestClient:
    monkeypatch.setenv("WARMUP_ON_STARTUP", "false")
    monkeypatch.setenv("INTERNAL_TOKEN", token)
    get_settings.cache_clear()
    from accheck_ml import server

    return TestClient(server.app)


PAYLOAD = {
    "user_story": "As a Guest, I want to view product catalog so that I can browse items.",
    "acceptance_criteria": [],
    "permission_matrix": [
        {"role": "Guest", "action": "view", "resource": "Product Catalog", "effect": "Deny", "scope": "all"}
    ],
}


def test_analyze_detects_effect_conflict(monkeypatch):
    with _client(monkeypatch) as client:
        res = client.post("/v1/analyze", json=PAYLOAD)
    assert res.status_code == 200
    body = res.json()
    assert body["is_consistent"] is False
    assert "Effect Conflict" in [c["conflict_type"] for c in body["conflicts"]]
    assert body["model_version"].startswith("rule-based+")


def test_analyze_requires_internal_token(monkeypatch):
    with _client(monkeypatch, token="secret") as client:
        assert client.post("/v1/analyze", json=PAYLOAD).status_code == 401
        ok = client.post("/v1/analyze", json=PAYLOAD, headers={"X-Internal-Token": "secret"})
    assert ok.status_code == 200
    get_settings.cache_clear()
