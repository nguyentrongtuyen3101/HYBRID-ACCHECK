# Hybrid-ACCheck Backend

Hệ thống AI hỗ trợ Business Analyst tự động phát hiện xung đột và thiếu sót trong yêu cầu phân quyền dựa trên **User Story**, **Acceptance Criteria** và **Permission Matrix**.

> Đồ án tốt nghiệp + bài báo hội nghị khoa học quốc gia

---

## Kiến trúc thư mục (Clean & Scalable)

```
hybrid-accheck/
├── app/
│   ├── api/                  # Layer API (FastAPI routers)
│   │   └── v1/
│   │       ├── endpoints/    # Mỗi endpoint 1 file (health, check, ...)
│   │       └── router.py
│   ├── core/                 # Config, logging, security
│   ├── domain/               # Domain layer (thuần business)
│   │   ├── entities/
│   │   ├── enums/            # ConflictType, EffectType, ArtifactSource
│   │   └── schemas/          # Pydantic models
│   ├── services/             # Application services (mỗi thành phần 1 module)
│   │   ├── extractor/        # DeBERTa Token Classification
│   │   ├── normalizer/       # Sentence-BERT
│   │   ├── aligner/          # Weighted similarity matching
│   │   ├── acrg/             # Access-Control Requirement Graph
│   │   ├── reasoner/         # Symbolic rules + graph reasoning
│   │   ├── explainer/        # Rule template explanation
│   │   └── pipeline.py       # Orchestrator chính
│   ├── infrastructure/       # DB, ML models, external services (sẽ mở rộng)
│   │   ├── db/
│   │   └── ml/
│   ├── utils/
│   └── main.py
├── tests/
├── scripts/
├── configs/
├── data/                     # raw, processed, models
├── docs/
├── requirements.txt
└── README.md
```

### Nguyên tắc chia module
- **1 thành phần = 1 folder** trong `services/`
- Mỗi service có `base.py` (interface) + implementation cụ thể → dễ thay thế / mock / ablation study
- Domain schemas tách biệt → API và service không phụ thuộc lẫn nhau
- Infrastructure (DB, model loading) để riêng → dễ scale sau này

---

## Pipeline

```
US + AC  →  DeBERTa Extractor  →  SBERT Normalization  →  Cross-Artifact Alignment
         →  ACRG  →  Security Reasoning  →  Conflict/Normal  →  Explanation
```

---

## Chạy nhanh (không cần DB)

```bash
# 1. Tạo virtualenv
python -m venv .venv
source .venv/bin/activate   # Windows: .venv\Scripts\activate

# 2. Cài dependency
pip install -r requirements.txt

# 3. Copy env
cp .env.example .env

# 4. Chạy server
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

Swagger UI: http://localhost:8000/docs

### Test endpoint chính

```bash
curl -X POST http://localhost:8000/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{
    "user_story": "Sales may view assigned customers",
    "acceptance_criteria": ["Sales staff can only view customers assigned to them"],
    "permission_matrix": [
      {
        "role": "Sales Staff",
        "action": "View",
        "resource": "Customer",
        "effect": "Allow",
        "scope": "Assigned"
      }
    ]
  }'
```

---

## Các bước tiếp theo (theo đề tài)

1. **Dataset**: Mở rộng CrUISE-AC → thêm Permission Matrix + Conflict Label
2. **Fine-tune DeBERTa** Token Classification (ROLE, ACTION, RESOURCE, EFFECT, CONDITION, SCOPE)
3. **Tích hợp Sentence-BERT** thật + domain synonym dictionary
4. **DB**: Tạo database Postgres + SQLAlchemy models + Alembic
5. **Baseline**: Rule-based, LLM-only, AI-only
6. **Ablation study** + evaluation (Precision / Recall / F1 / Macro-F1)
7. **Demo UI** cho BA

---

## Conflict Types được hỗ trợ

| Label                  | Mô tả                                      |
|------------------------|--------------------------------------------|
| Normal                 | Nhất quán                                  |
| Missing Permission     | US/AC yêu cầu nhưng PM không có            |
| Action Conflict        | Hành động khác nhau                        |
| Effect Conflict        | Allow ↔ Deny                               |
| Scope Conflict         | Own / Assigned / All khác nhau             |
| Condition Conflict     | Điều kiện cấp quyền không khớp             |
