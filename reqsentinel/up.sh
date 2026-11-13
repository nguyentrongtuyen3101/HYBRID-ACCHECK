#!/usr/bin/env bash
# Chạy toàn bộ ReqSentinel bằng 1 lệnh.
#
#   ./reqsentinel/up.sh            build image còn thiếu (lần lượt từng cái) rồi chạy tất cả
#   ./reqsentinel/up.sh rebuild    build lại mọi image sau khi sửa code / git pull (layer torch vẫn dùng cache)
#   ./reqsentinel/up.sh rebuild web            chỉ build lại 1 hoặc vài service: ml | api | notification | web
#   ./reqsentinel/up.sh down       dừng tất cả (giữ dữ liệu)
#   ./reqsentinel/up.sh logs [svc] xem log
#   ./reqsentinel/up.sh status     trạng thái + RAM từng container
#
# Vì sao không dùng thẳng `docker compose up --build`: lệnh đó build 4 image song song
# (torch + Go + 2 lần npm) nên RAM dồn lên cùng lúc. Script này build từng image một và
# dừng build nếu RAM trống của máy xuống dưới MIN_FREE_MB.
set -euo pipefail
cd "$(dirname "$0")"

MIN_FREE_MB=${MIN_FREE_MB:-700}           # dưới ngưỡng này thì dừng build cho an toàn
SERVICES=(ml api notification web)        # ml build trước: nặng nhất, làm lúc RAM còn nhiều nhất

free_mb() { awk '/MemAvailable/ {print int($2/1024)}' /proc/meminfo 2>/dev/null || echo 99999; }
c() { docker compose "$@"; }

need_env() {
  if [[ ! -f .env ]]; then
    cp .env.example .env
    echo "Đã tạo reqsentinel/.env từ .env.example. Điền JWT_SECRET (>= 32 ký tự) và ADMIN_PASSWORD rồi chạy lại."
    exit 1
  fi
}

check_memory() {
  local docker_mb free
  docker_mb=$(docker info --format '{{.MemTotal}}' | awk '{print int($1/1048576)}')
  free=$(free_mb)
  echo "RAM cấp cho Docker: ${docker_mb} MB · RAM trống của máy: ${free} MB"
  if (( docker_mb < 2500 )); then
    echo "⚠ Docker được cấp dưới 2,5 GB. Docker Desktop → Settings → Resources → Memory, đặt 3–4 GB."
  fi
  if (( free < MIN_FREE_MB + 500 )); then
    echo "⚠ Máy còn ít RAM trống (${free} MB). Nên đóng bớt Chrome/ứng dụng nặng trước khi build."
    read -r -p "Vẫn tiếp tục? [y/N] " ans
    [[ $ans == [yY] ]] || exit 1
  fi
}

# Build 1 image, theo dõi RAM trống mỗi 2 giây; tụt dưới ngưỡng thì dừng build.
build_one() {
  local svc=$1 pid low=0
  echo "▶ Build $svc ..."
  c build "$svc" &
  pid=$!
  while kill -0 "$pid" 2>/dev/null; do
    if (( $(free_mb) < MIN_FREE_MB )); then
      low=$((low + 1))
      if (( low >= 3 )); then
        echo "✖ RAM trống còn $(free_mb) MB — dừng build $svc để bảo vệ máy. Đóng bớt ứng dụng rồi chạy lại."
        kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null || true
        exit 1
      fi
    else
      low=0
    fi
    sleep 2
  done
  wait "$pid"
}

build_missing() {
  local force=${1:-0}
  for svc in "${SERVICES[@]}"; do
    if (( force )) || ! docker image inspect "reqsentinel-$svc:latest" >/dev/null 2>&1; then
      build_one "$svc"
    else
      echo "✓ reqsentinel-$svc:latest đã có — bỏ qua build"
    fi
  done
}

wait_ready() {
  local port url i
  port=$(grep -E '^API_PORT=' .env 2>/dev/null | cut -d= -f2 || true); url="http://localhost:${port:-8000}/api/v1/health"
  echo "Đợi các service sẵn sàng (ml-service nạp model lần đầu mất 1–2 phút)..."
  for i in $(seq 1 90); do
    if curl -fs "$url" 2>/dev/null | grep -q '"status":"ok"'; then
      echo "✓ Tất cả đã sẵn sàng"; return
    fi
    sleep 2
  done
  echo "⚠ Chưa thấy trạng thái ok sau 3 phút — xem ./reqsentinel/up.sh logs"
}

show_status() {
  c ps --format 'table {{.Service}}\t{{.Status}}'
  echo
  docker stats --no-stream --format 'table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}' $(c ps -q) 2>/dev/null || true
}

case "${1:-up}" in
  up)
    need_env; check_memory; build_missing 0
    c up -d --no-build
    wait_ready; show_status
    cat <<'EOF'

  Giao diện   http://localhost:8080
  API         http://localhost:8000/api/v1/health
  Email dev   http://localhost:8025
  RabbitMQ    http://localhost:15673   (guest / guest)
EOF
    ;;
  rebuild)
    shift
    need_env; check_memory
    if (( $# )); then                       # chỉ build lại các service được nêu tên
      for svc in "$@"; do
        [[ " ${SERVICES[*]} " == *" $svc "* ]] || { echo "Không có service '$svc' (chọn: ${SERVICES[*]})"; exit 1; }
        build_one "$svc"
      done
      c up -d --no-build "$@"
    else
      build_missing 1
      c up -d --no-build
    fi
    wait_ready; show_status
    ;;
  down)   c down ;;
  logs)   shift; c logs -f --tail=100 "$@" ;;
  status) show_status ;;
  *)      sed -n '2,10p' "$0"; exit 1 ;;
esac
