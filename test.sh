#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"
MODEL_DOWNLOAD_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/tts-onnx/models"
TTS_BIN="${TTS_BIN:-./bin/tts}"

echo "[1/9] Checking CLI binary (${TTS_BIN})"
if [[ ! -x "${TTS_BIN}" ]]; then
  echo "Error: ${TTS_BIN} not found or not executable."
  echo "This script no longer builds the CLI binary automatically."
  echo "Please prepare the binary first (for example: task release:build)."
  exit 1
fi

EN_TEXT="Although this local text to speech daemon is designed for desktop usage, it still needs to handle a fairly long sentence with stable pacing, clear articulation, and no abrupt glitches when running inference on CPU only environments."
KO_TEXT="이 로컬 텍스트 투 스피치 데몬은 데스크톱 환경에서 오프라인으로 동작하도록 설계되었지만, 비교적 긴 문장을 입력하더라도 발음이 뭉개지지 않고 속도 변화가 과도하지 않으며 자연스럽게 이어지는 음성을 안정적으로 생성해야 합니다."
JA_TEXT="これはにほんごのおんせいごうせいてすとです。ややながいぶんでも、しぜんなよみあげになることをかくにんします。"
ZH_TEXT="这是中文语音合成测试。我们希望语速稳定、发音清晰，并且听起来更自然。"

EN_MODEL_ID="kitten-nano-en-v0_1-fp16"
KO_MODEL_ID="vits-mimic3-ko_KO-kss_low"
JA_MODEL_ID="kokoro-int8-multi-lang-v1_0"
# Use a Japanese voice sid from kokoro v1_0 (jf_* range: 37-40, jm_*: 41).
JA_VOICE_SID="37"
ZH_MODEL_ID="vits-piper-zh_CN-huayan-medium"

ensure_model_installed() {
  local lang="$1"
  local model_id="$2"
  local voice_list

  voice_list="$("${TTS_BIN}" --voice-list --lang "${lang}" 2>/dev/null || true)"
  if [[ "${voice_list}" == *"${model_id}"* ]]; then
    echo "Model already installed (${lang}: ${model_id}) -> skipping download"
    return 0
  fi

  echo "Downloading model (${lang}: ${model_id}) -> ${MODEL_DOWNLOAD_DIR}"
  "${TTS_BIN}" --lang "${lang}" --install-remote-id "${model_id}"
}

echo "[2/9] Ensuring English model (${EN_MODEL_ID})"
ensure_model_installed "en" "${EN_MODEL_ID}"
echo "[3/9] Ensuring Korean model (${KO_MODEL_ID})"
ensure_model_installed "ko" "${KO_MODEL_ID}"
echo "[4/9] Ensuring Japanese model (${JA_MODEL_ID})"
ensure_model_installed "ja" "${JA_MODEL_ID}"
echo "[5/9] Ensuring Chinese model (${ZH_MODEL_ID})"
ensure_model_installed "zh" "${ZH_MODEL_ID}"

echo "[6/9] Playing English sample via speaker"
"${TTS_BIN}" --lang en --voice "${EN_MODEL_ID}" "${EN_TEXT}"

echo "[7/9] Playing Korean sample via speaker"
"${TTS_BIN}" --lang ko --voice "${KO_MODEL_ID}" "${KO_TEXT}"

echo "[8/9] Playing Japanese sample via speaker"
"${TTS_BIN}" --lang ja --voice "${JA_VOICE_SID}" "${JA_TEXT}"

echo "[9/9] Playing Chinese sample via speaker"
"${TTS_BIN}" --voice "${ZH_MODEL_ID}" "${ZH_TEXT}"

echo "Done. Audio was played via speaker (no wav files written)."
