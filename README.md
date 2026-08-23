# ReqSentinel (Hybrid-ACCheck)

Hệ thống AI hỗ trợ Business Analyst tự động phát hiện **xung đột / thiếu sót** trong yêu cầu phân quyền dựa trên **User Story**, **Acceptance Criteria** và **Permission Matrix**.

> Đồ án tốt nghiệp · Pipeline hybrid: DeBERTa + SBERT + Alignment + ACRG + Symbolic Reasoner

---

## Tính năng hiện có

- **Web UI (ReqSentinel)**: quản lý Project, User Story, Permission Matrix, chạy analysis, History, xem chi tiết run
- **REST API** (FastAPI + Swagger `/docs`)
- **Pipeline hybrid**: Extract → Normalize → Align → ACRG → Reason → Explain
- **Postgres**: lưu projects, user stories, AC, permission matrix, pipeline runs, conflicts, extracted entities
- **Fine-tuned DeBERTa** token classification + RuleBased fallback
- **SBERT** alignment + symbolic conflict rules

---

## Kiến trúc thư mục

```text
hybrid-accheck/
├── app/
│   ├── api/v1/endpoints/     # health, check, projects
│   ├── core/                 # config, logging
│   ├── domain/               # entities, enums, schemas
│   ├── services/
│   │   ├── extractor/        # DeBERTa + RuleBased
│   │   ├── normalizer/       # SBERT
│   │   ├── aligner/          # Weighted similarity
│   │   ├── acrg/             # Access-Control Requirement Graph
│   │   ├── reasoner/         # Symbolic conflict detection
│   │   ├── explainer/
│   │   └── pipeline.py
│   ├── infrastructure/db/    # SQLAlchemy models, session, repos
│   ├── static/               # UI: index.html, css/, js/
│   └── main.py
├── data/
│   ├── processed/            # dataset, bio labels
│   └── models/               # deberta-extractor checkpoint
├── scripts/                  # init_db, train, generate labels, evaluate
├── requirements.txt
├── .env.example
└── README.md
```

---

## Yêu cầu hệ thống

- Python **3.10+** (khuyến nghị 3.12)
- **PostgreSQL** (local hoặc Docker)
- ~2–4 GB RAM khi load DeBERTa + SBERT (GPU optional)

---

## Cài đặt từ đầu (sau khi `git pull`)

### 1. Clone / vào thư mục project

```bash
cd hybrid-accheck   # hoặc đường dẫn repo của bạn
```

### 2. Tạo virtualenv và cài dependency

```bash
python3 -m venv .venv
source .venv/bin/activate          # Windows: .venv\Scripts\activate

pip install --upgrade pip
pip install -r requirements.txt
```

Nếu thiếu package tokenizer DeBERTa:

```bash
pip install sentencepiece protobuf tiktoken
```

### 3. Cấu hình môi trường

```bash
cp .env.example .env
```

Sửa `.env` cho đúng Postgres của bạn:

```env
DATABASE_URL=postgresql+asyncpg://USER:PASSWORD@localhost:5432/hybrid_accheck
DEBUG=true
ALIGNMENT_THRESHOLD=0.55
```

Tạo database (nếu chưa có):

```bash
# psql hoặc Beekeeper
CREATE DATABASE hybrid_accheck;
```

Hoặc Docker:

```bash
docker run -d --name hybrid-pg \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=hybrid_accheck \
  -p 5432:5432 postgres:16
```

### 4. Tạo bảng (schema)

```bash
export PYTHONPATH=.
python scripts/init_db.py
```

### 5. Model ML

- Checkpoint DeBERTa fine-tune: `data/models/deberta-extractor/`
- SBERT (`all-MiniLM-L6-v2`) tải lần đầu qua Hugging Face (cần mạng)

Nếu chưa có checkpoint, pipeline vẫn chạy với **RuleBasedExtractor** fallback (accuracy thấp hơn).

### 6. Chạy server

```bash
source .venv/bin/activate
export PYTHONPATH=.
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

| URL | Mô tả |
|-----|--------|
| http://localhost:8000 | Web UI |
| http://localhost:8000/docs | Swagger API |
| http://localhost:8000/api/v1/health | Health check |

---

## API chính

| Method | Path | Mô tả |
|--------|------|--------|
| GET/POST | `/api/v1/projects` | Danh sách / tạo project |
| GET/PATCH/DELETE | `/api/v1/projects/{id}` | Chi tiết / sửa / xóa |
| POST/PATCH/DELETE | `/api/v1/projects/{id}/user-stories` | User stories + AC |
| POST/PATCH/DELETE | `/api/v1/projects/{id}/permission-matrix` | Dòng PM |
| POST | `/api/v1/projects/{id}/check` | Chạy pipeline (body: `user_story_id`) |
| GET | `/api/v1/projects/{id}/runs` | Lịch sử run theo project |
| POST | `/api/v1/check` | Check ad-hoc (không qua project) |
| GET | `/api/v1/check/history` | History global |
| GET | `/api/v1/check/history/{run_id}` | Chi tiết 1 run |

---

## Pipeline

```text
US + AC
  → DeBERTa Extractor (+ RuleBased fallback)
  → SBERT Normalizer
  → Weighted Aligner (threshold ~0.55)
  → ACRG
  → Symbolic Reasoner
  → Explainer
  → Lưu pipeline_runs / conflict_records / extracted_entities
```

### Conflict types

| Label | Ý nghĩa |
|-------|---------|
| Normal | Nhất quán |
| Missing Permission | US/AC yêu cầu nhưng không match PM |
| Action Conflict | Action khác nhau |
| Effect Conflict | Allow ↔ Deny |
| Scope Conflict | own / assigned / all không khớp |
| Condition Conflict | Điều kiện PM không khớp US/AC |

---

## Dataset & training (tùy chọn)

```bash
# Gán nhãn BIO
python scripts/generate_bio_labels.py \
  --dataset data/processed/hybrid_accheck_dataset_ecommerce_cms_200.json \
  --output data/processed/bio_token_labels.json

# Fine-tune DeBERTa
python scripts/train_deberta_extractor.py \
  --bio-file data/processed/bio_token_labels.json \
  --output-dir data/models/deberta-extractor \
  --epochs 10 --batch-size 8 --lr 1e-5

# Đánh giá baseline / hybrid
python scripts/evaluate_dataset.py ...
```

---

## UI – luồng dùng nhanh

1. **Projects** → Create project  
2. Mở project → tab **User Stories** / **Permission Matrix**  
3. Tab **Run analysis** → chọn US → Run (kết quả lưu DB)  
4. Tab **Runs** / menu **History** → View chi tiết  
5. **Edit** US/PM, **Settings** đổi tên project, **Delete** trên thẻ project  

---

## Troubleshooting

| Lỗi | Cách xử lý |
|-----|------------|
| `Could not save to DB` / History trống | Kiểm tra `DATABASE_URL`, Postgres đang chạy, đã `init_db` |
| DeBERTa load fail / tokenizer | `pip install sentencepiece protobuf tiktoken` |
| Import module | Luôn `export PYTHONPATH=.` từ thư mục gốc project |
| UI không cập nhật | Hard refresh `Ctrl+Shift+R` |
| CUDA OOM | Chạy CPU; giảm `batch-size` khi train |

---

## License / ghi chú

Đồ án học thuật — không dùng model/data nhạy cảm production khi chưa audit bảo mật.