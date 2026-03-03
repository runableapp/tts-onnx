#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="tts-onnx.service"
UNIT_SRC="deploy/systemd/${SERVICE_NAME}"
UNIT_DST="${HOME}/.config/systemd/user/${SERVICE_NAME}"

mkdir -p "${HOME}/.config/systemd/user"
cp "${UNIT_SRC}" "${UNIT_DST}"
systemctl --user daemon-reload
echo "Installed ${UNIT_DST}"
