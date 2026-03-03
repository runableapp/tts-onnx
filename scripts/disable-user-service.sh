#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="tts-onnx.service"
systemctl --user disable --now "${SERVICE_NAME}" || true
echo "Disabled ${SERVICE_NAME}"
