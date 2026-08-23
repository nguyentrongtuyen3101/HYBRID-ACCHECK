FROM python:3.12-slim

# Cài các gói hệ thống cần thiết (torch, transformers thường cần build-essential + git)
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy requirements trước để tận dụng Docker layer cache (chỉ rebuild khi requirements đổi)
COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir -r requirements.txt

# Copy toàn bộ source code
COPY . .

# Hugging Face Spaces yêu cầu container LẮNG NGHE ĐÚNG PORT 7860, không được đổi
ENV PORT=7860
EXPOSE 7860

# HF Spaces chạy container với user không phải root theo mặc định ở 1 số base image,
# tạo thư mục cache cho HuggingFace model (tránh lỗi permission khi tải model runtime)
ENV HF_HOME=/app/.cache/huggingface
RUN mkdir -p /app/.cache/huggingface && chmod -R 777 /app/.cache

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "7860"]
