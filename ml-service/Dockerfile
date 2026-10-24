FROM python:3.12-slim

WORKDIR /srv

# Torch bản CPU (~200MB) thay vì bản CUDA mặc định (vài GB).
# Có GPU thì đổi index-url sang bản cu12x tương ứng.
COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir torch --index-url https://download.pytorch.org/whl/cpu \
    && pip install --no-cache-dir -r requirements.txt

# Cài chính package accheck_ml (deps đã có ở layer trên → layer này rebuild rất nhanh)
COPY pyproject.toml .
COPY accheck_ml ./accheck_ml
RUN pip install --no-cache-dir --no-deps .

# data/models/deberta-extractor (nếu có) được bundle vào image;
# hoặc mount volume vào /srv/data/models khi chạy.
COPY data ./data

ENV HF_HOME=/srv/.cache/huggingface \
    MODEL_CACHE_DIR=/srv/data/models \
    DEBERTA_MODEL_DIR=/srv/data/models/deberta-extractor \
    PORT=8001
RUN mkdir -p /srv/.cache/huggingface /srv/data/models && chmod -R 777 /srv/.cache /srv/data

EXPOSE 8001
CMD ["accheck-ml-serve"]
