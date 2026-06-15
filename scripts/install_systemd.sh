#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy/server"
SERVICE_SRC="${DEPLOY_DIR}/netweaver-controller.service"
SERVICE_DST="/etc/systemd/system/netweaver-controller.service"

if [[ $EUID -ne 0 ]]; then
  echo "please run as root" >&2
  exit 1
fi

if [[ ! -f "$SERVICE_SRC" ]]; then
  echo "service file not found: $SERVICE_SRC" >&2
  echo "run scripts/deploy_server.sh first" >&2
  exit 1
fi

install -Dm644 "$SERVICE_SRC" "$SERVICE_DST"
systemctl daemon-reload
systemctl enable netweaver-controller.service
echo "installed: $SERVICE_DST"
echo "start with: systemctl start netweaver-controller.service"
echo "status with: systemctl status netweaver-controller.service"
