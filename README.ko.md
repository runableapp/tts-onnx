# 오픈소스 프로젝트를 후원해 주세요 📚

저희 도구가 도움이 되었다면, 후원을 부탁드립니다!

[**❤️ Polar.sh로 후원하기**](https://buy.polar.sh/polar_cl_2RRA7HD1Pv9AFP7pAZRg8XDomwZc9WCLfHXaW0hdJZz)

---

# Linux TTS Daemon (Go)

안정적인 REST API, 모델 수명주기 관리, `systemd --user` 운영을 지원하는 Ubuntu 중심의 로컬 텍스트 음성 합성 데몬입니다.

## 현재 런타임 상태

- 기본 엔진은 로컬 개발 및 API 검증용 `mock` 입니다.
- 실제 `sherpa-onnx` Go 런타임이 통합되어 있으며, 네이티브 추론 실행 시 `config/config.sherpa.yaml` 을 사용합니다.
- KO/ZH/JA/EN 런타임은 로컬 매니페스트의 설치 모델과 sherpa 릴리스 원격 목록을 함께 사용합니다.

## 시스템 의존성

Ubuntu에서 `./bin/tts` 실행 시 필요한 런타임 의존성:

- `libc6`
- `libstdc++6`
- `libgcc-s1`
- `ca-certificates` (원격 모델 다운로드용)

선택 런타임 의존성:

- `alsa-utils` (로컬 스피커 재생 `aplay`)

소스에서 빌드할 때 필요한 의존성:

- `golang` (Go toolchain)
- `gcc` (sherpa 런타임 cgo 링크 단계에 필요)
- `task` ( `build.sh` / Taskfile 워크플로 사용 시)

Ubuntu 설치 예시:

```bash
sudo apt update
sudo apt install -y libc6 libstdc++6 libgcc-s1 ca-certificates alsa-utils golang gcc
```

## ONNX 및 모델 URL

- Sherpa ONNX TTS 문서:
  - https://k2-fsa.github.io/sherpa/onnx/tts/index.html
- Sherpa ONNX TTS 사전학습 모델:
  - https://k2-fsa.github.io/sherpa/onnx/tts/pretrained_models/index.html
  - https://github.com/k2-fsa/sherpa-onnx/releases/tag/tts-models

- Sherpa ONNX 전체 TTS 카탈로그:
  - https://k2-fsa.github.io/sherpa/onnx/tts/all/
- Sherpa ONNX Go 바인딩:
  - https://github.com/k2-fsa/sherpa-onnx-go

이 저장소에서 사용하는 고정 모델 아티팩트 URL:

- 한국어 (Mimic3 VITS):
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-mimic3-ko_KO-kss_low.tar.bz2
- 중국어 (Piper):
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-piper-zh_CN-huayan-medium.tar.bz2
- 일본어 (Kokoro int8 multi-lang):
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-int8-multi-lang-v1_0.tar.bz2
- 영어 (Kitten):
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2

## Sherpa 런타임 `.so` 위치

`linux-tts-onnx` 는 현재 동적 링크(완전 정적 링크 아님)로 빌드됩니다.

Sherpa 런타임 공유 라이브러리는 다음 Go 모듈 경로에 있습니다:

- `$(go env GOPATH)/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@<version>/lib/x86_64-unknown-linux-gnu/`

예상 파일:

- `libsherpa-onnx-c-api.so`
- `libonnxruntime.so`

빠른 확인:

- 라이브러리 목록: `ls "$(go env GOPATH)"/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@*/lib/x86_64-unknown-linux-gnu`
- 링크 확인: `ldd ./bin/tts`

## 모델 저장 경로

다운로드된 모델 저장 위치:

- `~/.local/share/tts-onnx/models`

현재 언어별 모델 경로:

- 영어:
  - `~/.local/share/tts-onnx/models/en/<version>/<model-id>/`
- 한국어:
  - `~/.local/share/tts-onnx/models/ko/<version>/`
- 중국어:
  - `~/.local/share/tts-onnx/models/zh/<version>/`
- 일본어:
  - `~/.local/share/tts-onnx/models/ja/<version>/<model-id>/`

보이스/모델 자산 파일 예시:

- `voices.bin`
- `tokens.txt`
- `model.onnx` 또는 `model.fp16.onnx`
- `espeak-ng-data/`

설치 모델 상태 추적 파일:

- `~/.local/share/tts-onnx/models/manifest.json`

빠른 확인:

- 서비스 기준 설치 모델 확인: `curl -fsS http://127.0.0.1:18741/v1/models`

## 빠른 시작

1. 빌드 및 실행:
   - `go build -o ./bin/tts ./cmd/tts`
   - `./bin/tts --service --config ./config/config.example.yaml`
2. 헬스 체크:
   - `curl http://127.0.0.1:18741/v1/health`
3. 음성 합성 테스트:
   - `curl -X POST http://127.0.0.1:18741/v1/speak -H 'content-type: application/json' -d '{"text":"hello world","lang":"en","format":"wav"}' --output out.wav`
4. 스피커 재생(서비스):
   - `config/config.sherpa.yaml` 에서 `play_on_speak: true` 로 설정하면 `/v1/speak` 호출 시 호스트 스피커로 즉시 재생됩니다.

## CLI 인자

서비스 모드 (동일 `tts` 바이너리):

- `--config` (기본값: `./config/config.sherpa.yaml`): 설정 파일 경로
- `--service`: HTTP 데몬 모드 실행

직접 CLI (`cmd/tts/main.go`):

- 모든 플래그는 더블 대시 형태 사용 (예: `--voice-list`)
- 인자 없이 `./bin/tts` 실행 시 자동으로 도움말 출력
- `--lang` (합성 시 선택): 모델 선택용 언어 버킷. 생략 시 `--voice` 선택자 또는 첫 설치 모델 기준으로 추론
- `--voice` (선택): 설치 모델 선택자 (`id`/`version`) 또는 숫자 스피커 ID(`sid`)
- `--format` (기본값: `wav`): `wav|pcm_s16le`
- `--out` (선택): 출력 파일 경로. 설정 시에만 파일 저장
- `--config` (기본값: `./config/config.sherpa.yaml`)
- `--rate` (기본값: `1.0`)
- `--sample-rate` (기본값: `0`, 선택 오버라이드)
- `--request-id` (선택): 상관관계/취소 ID
- `--no-play` (기본값: `false`): 즉시 스피커 재생 비활성화
- `--voice-list`: 설치 모델/보이스 목록 출력 (기본 전체 언어, `--lang` 으로 단일 언어). 보이스 이름이 없으면 `sid` 범위 표시
- `--remote-models`: sherpa-onnx 릴리스의 온라인 TTS 모델 패키지 목록
- `--install-remote-id`: 원격 목록에서 모델 다운로드+압축해제 (언어는 remote model ID에서 추론)
- `--menu`: 대화형 언어/모델/보이스 선택
- `--auto-install` (기본값: `true`): `--menu` 에서 선택 모델 미설치 시 자동 설치
- 위치 인자 `text...`: 합성 입력 텍스트

예시:

- `./bin/tts "Sentence test without explicit language"`
- `./bin/tts --voice kitten-nano-en-v0_1-fp16 "Sentence test"`
- `./bin/tts --voice-list --lang en`
- `./bin/tts --remote-models --lang en`
- `./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16`
- `./bin/tts --menu`

`./bin/` 접두사 없이 실행하려면:

- `sudo ln -sf "$(pwd)/bin/tts" /usr/local/bin/tts`
- 또는 `PATH` 에 `./bin` 추가

모델 선택 동작:

- `--voice` 가 설치 모델의 `id` 또는 `version` 과 일치하면 해당 모델 사용, 언어도 해당 모델 기준으로 추론
- 아니면 `--voice` 를 선택 모델의 숫자 스피커 ID(`sid`)로 처리
- 모델 미지정 시 선택 언어의 첫 설치 모델 사용, `--lang` 도 없으면 전체 언어 중 첫 설치 모델 사용

## 서비스 API

Base URL: `http://127.0.0.1:18741/v1`

인증 동작 (`internal/httpapi/server.go`):

- `bearer_token` 이 비어 있으면 인증 불필요
- `bearer_token` 이 설정되면 `/v1/models`, `/v1/models/install`, `/v1/models/{lang}/{version}`, `/v1/speak`, `/v1/stop`, `/v1/metrics` 는 `Authorization: Bearer <token>` 필요
- `/v1/health`, `/v1/capabilities`, `/` 는 인증 없이 접근 가능

엔드포인트:

- `GET /v1/health`
- `GET /v1/capabilities`
- `GET /v1/models`
- `POST /v1/models/install`
- `DELETE /v1/models/{lang}/{version}?force=true|false`
- `POST /v1/speak`
- `POST /v1/stop`
- `GET /v1/metrics`
- `GET /` (루트 점검 엔드포인트, `{status,time}` 반환)

주요 요청 필드:

- `/v1/models/install`: `lang` (필수), `url` (필수), `model_id`, `checksum`, `version`
- `/v1/speak`: `text`, `lang` (선택), `voice`, `rate`, `format`, `sample_rate`, `request_id`
- `/v1/stop`: `request_id`

`/v1/speak` 응답:

- `200 OK` + 바이너리 오디오 바디
- `Content-Type`: `audio/wav` 또는 `application/octet-stream`
- `X-Sample-Rate`: 출력 샘플레이트

## 다국어 장문 테스트

- `bash ./test.sh`
- 현재 동작: KO/ZH/JA/EN 테스트 모델 다운로드 후 스피커(`aplay`)로 재생하며 WAV 파일은 저장하지 않음

## Taskfile 워크플로

- `task dev:run`
- `task test:unit`
- `task service:install-user-unit`
- `task service:enable`
- `task release:build`
- `task release:package VERSION=v0.1.0`

## 레이아웃

- `cmd/tts`: CLI + 서비스 모드 단일 엔트리포인트
- `cmd/runtime-check`: 네이티브 런타임 가시성 체크 도우미
- `internal/httpapi`: REST 핸들러 및 오류 모델
- `internal/modelmgr`: 모델 설치/삭제 + 매니페스트
- `internal/synth`: 합성 큐/엔진 추상화 및 mock 오디오 생성
- `deploy/systemd`: `systemd --user` 유닛
- `cmd/tts`: 비서비스 합성용 직접 CLI

## 문서

- 전체 API 참조: `API_FULL.md`
