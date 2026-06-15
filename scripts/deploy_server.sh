#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy/server"
BACKEND_DIR="${ROOT_DIR}/backend/code"
FRONTEND_DIR="${ROOT_DIR}/frontend"
ENV_FILE="${DEPLOY_DIR}/netweaver-server.env"
CONTROLLER_BASE="${DEPLOY_DIR}/controller"
CONSOLE_BASE="${DEPLOY_DIR}/console"
BIN_DIR="${CONTROLLER_BASE}/bin"
WEB_DIR="${CONSOLE_BASE}/web"
LOG_DIR="${CONTROLLER_BASE}/logs"
RUN_DIR="${DEPLOY_DIR}/run"

log() {
  printf '[%s] %s\n' "$(date +'%Y-%m-%d %H:%M:%S')" "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "missing required command: $1"
    exit 1
  fi
}

prepare_dirs() {
  mkdir -p "$BIN_DIR" "$WEB_DIR" "$LOG_DIR" "$RUN_DIR"
}

write_env_template() {
  if [[ -f "$ENV_FILE" ]]; then
    return
  fi

  cat > "$ENV_FILE" <<'EOF'
# NetWeaver server deployment environment
# The script is only responsible for reading this file, building the project,
# and generating the runnable deployment directory.
# The real deployment behavior is decided by the values you fill in here.
#
# Read this file in four groups:
# 1. Service listen addresses: optional, change only when you need different ports.
# 2. Security values: must be changed before real deployment.
# 3. Network / relay values: relay address must be changed before multi-machine deployment.
# 4. Frontend build values: usually keep the defaults when controller hosts the console.

# 1) Service listen addresses. Optional.
# Controller HTTP listen address on the server itself.
# Most teams can keep :8080.
NETWEAVER_HTTP_ADDR=:8080

# UDP relay listen address on the server itself.
# If you change this port, also change NETWEAVER_RELAY_PORT below to the same value.
NETWEAVER_RELAY_LISTEN=:9000

# URL base used when controller serves the frontend console.
# With the default value, the browser entry becomes: http://<server-ip>:8080/console/
NETWEAVER_CONSOLE_BASE=/console

# 2) Security values.
# Dashboard username. Optional. Keeping admin is allowed, but changing it is safer.
NETWEAVER_DASHBOARD_USERNAME=admin

# Dashboard password. Must be changed before real deployment.
NETWEAVER_DASHBOARD_PASSWORD=change-me

# Dashboard JWT signing secret. Must be changed before real deployment.
NETWEAVER_JWT_SECRET=netweaver-dashboard-dev-secret

# Shared secret used by nodes when calling protected node APIs.
# Must be changed before real deployment, and all nodes must use the same value.
NETWEAVER_NODE_PSK=netweaver-dev-psk

# 3) Network / relay values.
# Must be changed to the real server IP before multi-machine deployment.
# Do not keep 127.0.0.1 unless controller and every node are on the same machine.
NETWEAVER_RELAY_ADDR=127.0.0.1

# Public relay port reported to nodes.
# Usually keep it the same as the port in NETWEAVER_RELAY_LISTEN.
NETWEAVER_RELAY_PORT=9000

# STUN server list. Optional.
# The default public STUN list is enough for most classroom or lab tests.
# Only change it when your network environment requires your own STUN plan.
NETWEAVER_STUN_SERVERS=stun.l.google.com:19302,stun1.l.google.com:19302,stun2.l.google.com:19302

# 4) Frontend build values.
# Keep VITE_API_BASE_URL empty when frontend is hosted by controller on the same server.
# Then the browser will continue requesting same-origin /api/... routes.
VITE_API_BASE_URL=

# Frontend static asset base path.
# If NETWEAVER_CONSOLE_BASE is /console, keep this as /console/.
VITE_APP_BASE=/console/
EOF

  log "created environment template: $ENV_FILE"
}

load_env() {
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
}

build_backend() {
  log "building controller binary"
  (
    cd "$BACKEND_DIR"
    go test ./...
    go build -o "$BIN_DIR/netweaver-controller" ./cmd/controller
  )
}

build_frontend() {
  log "installing frontend dependencies"
  (
    cd "$FRONTEND_DIR"
    npm ci
  )

  log "building frontend console"
  rm -rf "$WEB_DIR"
  mkdir -p "$WEB_DIR"
  (
    cd "$FRONTEND_DIR"
    VITE_API_BASE_URL="${VITE_API_BASE_URL:-}" \
    VITE_APP_BASE="${VITE_APP_BASE:-/console/}" \
    npm run build -- --outDir "$WEB_DIR"
  )
}

write_run_script() {
  cat > "$RUN_DIR/start-controller.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT_DIR}/netweaver-server.env"
BIN="${ROOT_DIR}/controller/bin/netweaver-controller"
WEB_DIR="${ROOT_DIR}/console/web"
LOG_DIR="${ROOT_DIR}/controller/logs"
mkdir -p "$LOG_DIR"
set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a
exec "$BIN" \
  -addr "${NETWEAVER_HTTP_ADDR:-:8080}" \
  -relay-addr "${NETWEAVER_RELAY_LISTEN:-:9000}" \
  -console-dir "$WEB_DIR" \
  -console-base "${NETWEAVER_CONSOLE_BASE:-/console}"
EOF
  chmod +x "$RUN_DIR/start-controller.sh"
}

write_systemd_template() {
  cat > "$DEPLOY_DIR/netweaver-controller.service" <<EOF
[Unit]
Description=NetWeaver Controller
After=network.target

[Service]
Type=simple
WorkingDirectory=${DEPLOY_DIR}
ExecStart=${RUN_DIR}/start-controller.sh
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
}

print_summary() {
  cat <<EOF

Deployment prepared successfully.

Controller binary:
  ${BIN_DIR}/netweaver-controller

Frontend static files:
  ${WEB_DIR}

Environment file:
  ${ENV_FILE}

Start controller manually:
  ${RUN_DIR}/start-controller.sh

Optional systemd unit file:
  ${DEPLOY_DIR}/netweaver-controller.service
EOF
}

main() {
  require_cmd go
  require_cmd npm
  prepare_dirs
  write_env_template
  load_env
  build_backend
  build_frontend
  write_run_script
  write_systemd_template
  print_summary
}

main "$@"
