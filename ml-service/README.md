# ml-service (Python, package `accheck-ml`)

Suy luận pipeline Hybrid-ACCheck. Stateless, không có DB: api-service gửi input, nhận kết quả rồi tự lưu.
Model được load **1 lần** khi khởi động. Train / đánh giá chạy offline bằng CLI, tách khỏi service đang chạy.

## Pipeline

```text
US + AC
  → DeBERTa Extractor (+ RuleBased fallback; không có checkpoint thì chỉ dùng RuleBased)
  → SBERT Normalizer
  → Weighted Aligner (threshold ~0.55)
  → ACRG
  → Symbolic Reasoner
  → Explainer
```

| Label | Ý nghĩa |
|-------|---------|
| Normal | Nhất quán |
| Missing Permission | US/AC yêu cầu nhưng không match PM |
| Action Conflict | Action khác nhau |
| Effect Conflict | Allow ↔ Deny |
| Scope Conflict | own / assigned / all không khớp |
| Condition Conflict | Điều kiện PM không khớp US/AC |

Mỗi kết quả kèm `model_version` (vd `rule-based+all-MiniLM-L6-v2`, `deberta@202610241200+all-MiniLM-L6-v2`;
ghi đè bằng `MODEL_VERSION`), để biết lần chạy nào do model nào sinh ra.

## Chạy

```bash
python3 -m venv .venv && source .venv/bin/activate
pip install torch --index-url https://download.pytorch.org/whl/cpu   # hoặc bản CUDA
pip install -e ".[dev]"
cp .env.example .env
accheck-ml-serve           # http://localhost:8001 · Swagger: /docs
pytest
```

API: `GET /health`, `POST /v1/analyze` (header `X-Internal-Token` khi đặt `INTERNAL_TOKEN`).

## Train / đánh giá

```bash
pip install -e ".[train]"
accheck-ml-labels --dataset data/processed/hybrid_accheck_dataset_ecommerce_cms_200.json \
                  --output data/processed/bio_token_labels.json
accheck-ml-train  --bio-file data/processed/bio_token_labels.json \
                  --output-dir data/models/deberta-extractor --epochs 10 --batch-size 8 --lr 1e-5
accheck-ml-eval   --dataset data/processed/hybrid_accheck_dataset_ecommerce_cms_200.json
```

Dữ liệu phản hồi từ BA: trang Admin → *Export dataset (JSON)* xuất các story đã được đánh giá, cùng định dạng
`data/processed/*.json`, gộp vào dataset rồi train lại. Train xong chỉ cần restart ml-service.

## Docker

```bash
docker build -t reqsentinel-ml .
docker run -p 8001:8001 -e INTERNAL_TOKEN=... -v $PWD/data/models:/srv/data/models reqsentinel-ml
```

Image dùng torch bản CPU; checkpoint trong `data/models` được bundle vào image hoặc mount volume.
