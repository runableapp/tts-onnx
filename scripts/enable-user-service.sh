#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="tts-onnx.service"
systemctl --user enable --now "${SERVICE_NAME}"
systemctl --user status "${SERVICE_NAME}" --no-pager
