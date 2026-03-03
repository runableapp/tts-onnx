#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

if ! command -v task >/dev/null 2>&1; then
  echo "error: Taskfile runner 'task' is not installed." >&2
  echo "install: https://taskfile.dev/installation/" >&2
  exit 1
fi

task release:build
echo "built: ./bin/tts"
