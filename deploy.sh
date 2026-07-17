#!/usr/bin/env bash
# One-click deploy for go-order-management-system (stage-10).
# Target: Ubuntu/Debian cloud VM with root or passwordless sudo.
# Stack: Docker Compose (API + MySQL + Redis + RabbitMQ + migrate),
#        Nginx reverse proxy, frontend static build.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

FRONTEND_PORT="${FRONTEND_PORT:-8880}"
API_PORT="${API_PORT:-8082}"
NGINX_SITE="${NGINX_SITE:-/etc/nginx/sites-available/go-order-management-system}"
NGINX_LINK="${NGINX_LINK:-/etc/nginx/sites-enabled/go-order-management-system}"
SKIP_SYSTEM_UPDATE="${SKIP_SYSTEM_UPDATE:-0}"
SKIP_DOCKER_MIRROR="${SKIP_DOCKER_MIRROR:-0}"
SKIP_FIREWALL="${SKIP_FIREWALL:-0}"
SKIP_FRONTEND="${SKIP_FRONTEND:-0}"
NONINTERACTIVE="${NONINTERACTIVE:-0}"
COMPOSE_CMD="${COMPOSE_CMD:-}"

echo_step() {
  echo -e "${GREEN}[$1] $2${NC}"
}

echo_warn() {
  echo -e "${YELLOW}[warn] $1${NC}"
}

echo_err() {
  echo -e "${RED}[error] $1${NC}" >&2
}

need_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    if command -v sudo >/dev/null 2>&1; then
      exec sudo -E bash "$0" "$@"
    fi
    echo_err "Please run as root or with sudo."
    exit 1
  fi
}

detect_compose() {
  if [[ -n "${COMPOSE_CMD}" ]]; then
    return
  fi
  if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
  else
    echo_err "docker compose is not available after Docker install."
    exit 1
  fi
}

install_base_packages() {
  echo_step 1 "Install base packages"
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  if [[ "${SKIP_SYSTEM_UPDATE}" != "1" ]]; then
    apt-get upgrade -y
  fi
  apt-get install -y \
    ca-certificates \
    curl \
    git \
    gnupg \
    ufw \
    nginx \
    openssl \
    lsb-release
}

install_docker() {
  echo_step 2 "Ensure Docker Engine is installed"
  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    echo "Docker is already available."
    return
  fi

  if ! command -v docker >/dev/null 2>&1; then
    curl -fsSL https://get.docker.com | sh
  fi

  systemctl enable --now docker
  if ! docker info >/dev/null 2>&1; then
    echo_err "Docker installed but daemon is not ready."
    exit 1
  fi
}

configure_docker_mirror() {
  if [[ "${SKIP_DOCKER_MIRROR}" == "1" ]]; then
    echo_step 3 "Skip Docker registry mirror configuration"
    return
  fi

  echo_step 3 "Configure Docker registry mirrors (optional CN accelerate)"
  mkdir -p /etc/docker
  if [[ -f /etc/docker/daemon.json ]]; then
    echo_warn "/etc/docker/daemon.json already exists; leave it unchanged."
    return
  fi

  cat >/etc/docker/daemon.json <<'EOD'
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://mirror.ccs.tencentyun.com"
  ]
}
EOD
  systemctl restart docker
}

configure_firewall() {
  if [[ "${SKIP_FIREWALL}" == "1" ]]; then
    echo_step 4 "Skip firewall configuration"
    return
  fi

  echo_step 4 "Configure UFW (22/80)"
  ufw allow OpenSSH >/dev/null 2>&1 || ufw allow 22/tcp
  ufw allow 80/tcp
  ufw --force enable
}

prepare_env_file() {
  echo_step 5 "Prepare .env"
  if [[ ! -f .env.example ]]; then
    echo_err ".env.example is missing."
    exit 1
  fi

  if [[ ! -f .env ]]; then
    cp .env.example .env
    echo "Created .env from .env.example"
  fi

  # Compose requires MYSQL_PASSWORD and JWT_SECRET.
  if ! grep -qE '^MYSQL_PASSWORD=.+' .env || grep -qE '^MYSQL_PASSWORD=(your_password|your-password)?$' .env; then
    local mysql_password
    mysql_password="$(openssl rand -base64 18 | tr -d '/+=' | cut -c1-20)"
    if grep -qE '^MYSQL_PASSWORD=' .env; then
      sed -i "s|^MYSQL_PASSWORD=.*|MYSQL_PASSWORD=${mysql_password}|" .env
    else
      echo "MYSQL_PASSWORD=${mysql_password}" >>.env
    fi
    echo "Generated MYSQL_PASSWORD in .env"
  fi

  if ! grep -qE '^JWT_SECRET=.+' .env || grep -qE '^JWT_SECRET=(replace_with_a_32_plus_chars_random_secret|replace-with-at-least-32-random-characters)?$' .env; then
    local jwt_secret
    jwt_secret="$(openssl rand -base64 48 | tr -d '/+=' | cut -c1-48)"
    if grep -qE '^JWT_SECRET=' .env; then
      sed -i "s|^JWT_SECRET=.*|JWT_SECRET=${jwt_secret}|" .env
    else
      echo "JWT_SECRET=${jwt_secret}" >>.env
    fi
    echo "Generated JWT_SECRET in .env"
  fi

  # Align RabbitMQ URL with compose service name for in-cluster use is handled by compose.yml.
  # Keep local .env values for host tools; ensure common stage-10 keys exist.
  ensure_env_key "DB_SLOW_THRESHOLD" "200ms"
  ensure_env_key "DB_LOG_LEVEL" "warn"
  ensure_env_key "INVENTORY_RECONCILE_ENABLED" "true"
  ensure_env_key "INVENTORY_RECONCILE_INTERVAL" "5m"
  ensure_env_key "INVENTORY_RECONCILE_TIMEOUT" "3s"
  ensure_env_key "ORDER_TIMEOUT_DELAY" "30m"
  ensure_env_key "RABBITMQ_USER" "order_app"
  ensure_env_key "RABBITMQ_PASSWORD" "order_dev_password"
  ensure_env_key "JWT_EXPIRE_HOURS" "24"
  # Mainland-friendly Docker build defaults (Compose passes these as build args).
  ensure_env_key "GOPROXY" "https://goproxy.cn,direct"
  ensure_env_key "GOSUMDB" "sum.golang.google.cn"
  ensure_env_key "APK_MIRROR" "mirrors.aliyun.com"

  if [[ "${NONINTERACTIVE}" != "1" ]]; then
    echo
    echo "Current required secrets are ready in .env"
    echo "Optional: edit .env now (MySQL password, JWT, RabbitMQ, reconcile, slow SQL)."
    read -r -p "Press Enter to continue deployment..." _
  fi
}

ensure_env_key() {
  local key="$1"
  local value="$2"
  if ! grep -qE "^${key}=" .env; then
    echo "${key}=${value}" >>.env
  fi
}

start_backend_stack() {
  echo_step 6 "Build and start Docker stack (app/mysql/redis/rabbitmq/migrate)"
  detect_compose
  export DOCKER_BUILDKIT=1
  # Prefer values already in .env; export fallbacks for compose interpolation.
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
  export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
  export GOSUMDB="${GOSUMDB:-sum.golang.google.cn}"
  export APK_MIRROR="${APK_MIRROR:-mirrors.aliyun.com}"
  echo "Build args: GOPROXY=${GOPROXY} GOSUMDB=${GOSUMDB} APK_MIRROR=${APK_MIRROR}"
  # shellcheck disable=SC2086
  ${COMPOSE_CMD} --env-file .env up -d --build --wait
  # shellcheck disable=SC2086
  ${COMPOSE_CMD} --env-file .env ps
}

install_node_if_needed() {
  if command -v node >/dev/null 2>&1 && command -v npm >/dev/null 2>&1; then
    return
  fi
  echo "Installing Node.js 20.x for frontend build..."
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  apt-get install -y nodejs
}

build_frontend() {
  if [[ "${SKIP_FRONTEND}" == "1" ]]; then
    echo_step 7 "Skip frontend build"
    return
  fi

  echo_step 7 "Build frontend static assets"
  install_node_if_needed
  pushd fronted >/dev/null
  # Mainland npm mirror; override with NPM_REGISTRY=https://registry.npmjs.org if needed.
  npm config set registry "${NPM_REGISTRY:-https://registry.npmmirror.com}"
  if [[ -f package-lock.json ]]; then
    npm ci
  else
    npm install
  fi
  npm run build
  popd >/dev/null

  if [[ ! -f fronted/dist/index.html ]]; then
    echo_err "Frontend build failed: fronted/dist/index.html not found."
    exit 1
  fi
}

configure_nginx() {
  echo_step 8 "Configure Nginx reverse proxy"
  local frontend_root="${ROOT_DIR}/fronted/dist"
  local use_static=0
  if [[ -f "${frontend_root}/index.html" ]]; then
    use_static=1
  fi

  if [[ "${use_static}" -eq 1 ]]; then
    cat >"${NGINX_SITE}" <<NGX
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    client_max_body_size 10m;

    # Backend health and metrics
    location = /ping {
        proxy_pass http://127.0.0.1:${API_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Request-ID \$request_id;
    }

    location = /live {
        proxy_pass http://127.0.0.1:${API_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location = /readyz {
        proxy_pass http://127.0.0.1:${API_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location = /metrics {
        proxy_pass http://127.0.0.1:${API_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # API
    location ^~ /api/ {
        proxy_pass http://127.0.0.1:${API_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Request-ID \$request_id;
        proxy_set_header Authorization \$http_authorization;
        proxy_read_timeout 60s;
    }

    # Frontend SPA
    root ${frontend_root};
    index index.html;

    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
NGX
  else
    echo_warn "Frontend dist missing; proxy SPA to Vite/dev host on ${FRONTEND_PORT}"
    cat >"${NGINX_SITE}" <<NGX
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    location ^~ /api/ {
        proxy_pass http://127.0.0.1:${API_PORT};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Authorization \$http_authorization;
    }

    location = /ping { proxy_pass http://127.0.0.1:${API_PORT}; }
    location = /live { proxy_pass http://127.0.0.1:${API_PORT}; }
    location = /readyz { proxy_pass http://127.0.0.1:${API_PORT}; }
    location = /metrics { proxy_pass http://127.0.0.1:${API_PORT}; }

    location / {
        proxy_pass http://127.0.0.1:${FRONTEND_PORT};
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
NGX
  fi

  rm -f /etc/nginx/sites-enabled/default
  ln -sfn "${NGINX_SITE}" "${NGINX_LINK}"
  nginx -t
  systemctl enable --now nginx
  systemctl reload nginx
}

verify_deployment() {
  echo_step 9 "Verify health endpoints"
  local ok=1
  for path in /ping /live /readyz; do
    if curl -fsS "http://127.0.0.1${path}" >/dev/null; then
      echo "OK  http://127.0.0.1${path}"
    else
      echo_warn "FAIL http://127.0.0.1${path}"
      ok=0
    fi
  done

  if curl -fsS "http://127.0.0.1/metrics" >/dev/null; then
    echo "OK  http://127.0.0.1/metrics"
  else
    echo_warn "FAIL http://127.0.0.1/metrics"
    ok=0
  fi

  if curl -fsS "http://127.0.0.1:${API_PORT}/readyz" >/dev/null; then
    echo "OK  backend direct :${API_PORT}/readyz"
  else
    echo_warn "Backend :${API_PORT}/readyz is not ready yet"
    ok=0
  fi

  if [[ "${ok}" -ne 1 ]]; then
    echo_warn "Some checks failed. Inspect with: ${COMPOSE_CMD} --env-file .env logs --tail=200"
  fi
}

print_summary() {
  local public_ip
  public_ip="$(curl -fsS --max-time 5 ifconfig.me 2>/dev/null || true)"
  if [[ -z "${public_ip}" ]]; then
    public_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  fi
  if [[ -z "${public_ip}" ]]; then
    public_ip="YOUR_SERVER_IP"
  fi

  echo
  echo -e "${GREEN}=== Deploy finished ===${NC}"
  echo "Project dir : ${ROOT_DIR}"
  echo "Frontend    : http://${public_ip}/"
  echo "API base    : http://${public_ip}/api/v1"
  echo "Health      : http://${public_ip}/readyz"
  echo "Metrics     : http://${public_ip}/metrics"
  echo
  echo "Useful commands:"
  echo "  ${COMPOSE_CMD} --env-file .env ps"
  echo "  ${COMPOSE_CMD} --env-file .env logs -f app"
  echo "  ${COMPOSE_CMD} --env-file .env down"
  echo
  echo "First-time tips:"
  echo "  1) Register an admin user via API or UI, then assign admin role if needed."
  echo "  2) Keep .env private; never commit real secrets."
  echo "  3) Stage-10 includes Redis inventory prededuct, operation logs, reconcile worker, and /metrics."
  echo "  4) MySQL host port is not published by default; only app :${API_PORT} is exposed locally."
}

main() {
  need_root "$@"
  echo "=== Start one-click deploy: go-order-management-system ==="
  install_base_packages
  install_docker
  configure_docker_mirror
  configure_firewall
  prepare_env_file
  start_backend_stack
  build_frontend
  configure_nginx
  verify_deployment
  print_summary
}

main "$@"
