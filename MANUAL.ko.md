# TTS ONNX 매뉴얼 (사용 및 설정)

이 문서는 `linux-tts-onnx` 를 런타임에서 사용하고 설정하는 방법을 설명합니다.  
소스 코드 컴파일/빌드 절차는 포함하지 않습니다.

---

### 1) 이 문서의 전제

- `tts` 실행 파일이 이미 있어야 합니다 (`./bin/tts` 또는 `/usr/local/bin/tts`).
- Linux 환경 기준이며, 예시는 Ubuntu 명령어를 사용합니다.

### 2) 런타임 의존성

```bash
sudo apt update
sudo apt install -y libc6 libstdc++6 libgcc-s1 ca-certificates alsa-utils
```

설명:

- `ca-certificates`: 원격 모델 다운로드에 필요
- `alsa-utils`: 스피커 재생(`aplay`)에 필요

### 3) 기본 데이터 경로

- 모델 폴더: `~/.local/share/tts-onnx/models`
- 매니페스트: `~/.local/share/tts-onnx/models/manifest.json`
- 상태 경로: `~/.local/state/tts-onnx`
- 캐시 경로: `~/.cache/tts-onnx`

### 4) 초기 설정 (CLI)

완전 초기화가 필요하면:

```bash
rm -rf ~/.local/share/tts-onnx
```

권장 모델 설치:

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
./bin/tts --install-remote-id vits-mimic3-ko_KO-kss_low
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

모델 다운로드 URL:

- 전체 릴리스 목록: `https://github.com/k2-fsa/sherpa-onnx/releases/tag/tts-models`
- 영어 (`kitten-nano-en-v0_1-fp16`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2`
- 한국어 (`vits-mimic3-ko_KO-kss_low`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-mimic3-ko_KO-kss_low.tar.bz2`
- 일본어 멀티랭 (`kokoro-multi-lang-v1_0`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-multi-lang-v1_0.tar.bz2`

수동 설치( `--install-remote-id` 없이 URL 지정):

1. 서비스 실행:

```bash
./bin/tts --service --config ./config/config.sherpa.yaml
```

2. API로 URL 직접 지정하여 설치:

```bash
curl -X POST http://127.0.0.1:18741/v1/models/install \
  -H 'content-type: application/json' \
  -d '{
    "lang":"en",
    "model_id":"kitten-nano-en-v0_1-fp16",
    "version":"v0_1-fp16",
    "url":"https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2"
  }'
```

3. 설치 확인:

```bash
curl -fsS http://127.0.0.1:18741/v1/models
```

### 4.1) 원격 모델 조회 (`--remote-models`)

전체 원격 모델 목록:

```bash
./bin/tts --remote-models
```

언어별 필터:

```bash
./bin/tts --remote-models --lang en
./bin/tts --remote-models --lang ko
./bin/tts --remote-models --lang ja
```

여기서 `--lang`은 선택사항입니다. 생략하면 전체 원격 모델을 출력합니다.

출력 필드 의미:

- `lang`: 추론된 언어 태그 (또는 `unknown`)
- `id`: 모델 패키지 ID (`--install-remote-id`에 사용)
- `version`: 추론된 버전
- `url`: 아카이브 직접 다운로드 URL

`lang=unknown`이면 `--lang`을 명시해서 설치:

```bash
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

### 4.2) 모델 다운로드/설치 방법

방법 A (권장): 모델 ID로 설치

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
./bin/tts --install-remote-id vits-mimic3-ko_KO-kss_low
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

방법 B: URL을 직접 지정해서 설치 (서비스 API)

```bash
curl -X POST http://127.0.0.1:18741/v1/models/install \
  -H 'content-type: application/json' \
  -d '{
    "lang":"en",
    "model_id":"kitten-nano-en-v0_1-fp16",
    "version":"v0_1-fp16",
    "url":"https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2"
  }'
```

### 4.3) 다운로드 실패/재시도

다음과 같은 일시적 네트워크/CDN 오류가 보이면:

- `download failed with status 503`

잠시 후 동일 명령을 다시 실행하세요:

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
```

설치 후에는 아래로 정상 설치 여부를 확인할 수 있습니다:

```bash
./bin/tts --voice-list --lang en
./bin/tts --voice-list --lang ko
./bin/tts --voice-list --lang ja
```

일부 설치에서 `--lang`이 필요한 이유:

- 일부 멀티랭 모델은 ID만으로 언어 추론이 안 되거나 `lang=unknown`으로 표시됩니다.
- 이 경우 `--lang <language>`로 설치 대상 언어를 명시합니다.

### 5) 설치 확인

```bash
./bin/tts --voice-list
./bin/tts --voice-list --lang en
./bin/tts --voice-list --lang ko
./bin/tts --voice-list --lang ja
```

### 6) 합성/재생 (파일 저장 없이 스피커 재생)

```bash
./bin/tts "Hello, this is a test."
./bin/tts --lang ko "안녕하세요. 테스트입니다."
./bin/tts --lang ja "こんにちは。テストです。"
```

합성에서는 `--lang`이 선택사항입니다. 생략하면 `--voice` 또는 설치된 첫 모델 기준으로 자동 선택됩니다.

모델을 명시해서 사용:

```bash
./bin/tts --voice kitten-nano-en-v0_1-fp16 "English test"
./bin/tts --voice vits-mimic3-ko_KO-kss_low "한국어 테스트"
./bin/tts --voice v1_0 "日本語テスト"
```

멀티 스피커 모델에서 SID 사용:

```bash
./bin/tts --lang ko --voice 6 "speaker id six test"
```

### 7) 필요할 때만 파일 저장

```bash
./bin/tts --out ./out.wav "save this audio"
```

### 8) 서비스 설정 (빌드 과정 제외)

사용자 설정 파일 준비:

```bash
mkdir -p ~/.config/tts-onnx
cp ./config/config.sherpa.yaml ~/.config/tts-onnx/config.yaml
```

서비스 설치/시작:

```bash
bash ./scripts/install-user-unit.sh
bash ./scripts/enable-user-service.sh
```

헬스 체크:

```bash
curl -fsS http://127.0.0.1:18741/v1/health
```

로그 확인:

```bash
journalctl --user -u tts-onnx.service -f
```

서비스 중지/비활성화:

```bash
bash ./scripts/disable-user-service.sh
```

### 9) 서비스 API 빠른 사용 예

```bash
curl -X POST http://127.0.0.1:18741/v1/speak \
  -H 'content-type: application/json' \
  -d '{"text":"hello world","lang":"en","format":"wav"}' \
  --output out.wav
```

설치 모델 조회:

```bash
curl -fsS http://127.0.0.1:18741/v1/models
```

### 10) 문제 해결

- **`aplay not found in PATH`**
  - `alsa-utils` 설치 필요
- **`cannot infer language for remote model ...`**
  - 설치 명령에 `--lang <language>` 추가
- **`no supported model file found in ...`**
  - 해당 모델 재설치
- **스피커에서 소리가 안 나옴**
  - 시스템 오디오 출력 장치/볼륨 및 `aplay` 동작 확인
