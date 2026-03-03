# Support Our Open Source Project 📚

If you find our tools helpful, please consider supporting us!

[**❤️ Support via Polar.sh**](https://buy.polar.sh/polar_cl_2RRA7HD1Pv9AFP7pAZRg8XDomwZc9WCLfHXaW0hdJZz)

---

# Linux TTS Daemon (Go)

Local Ubuntu-first text-to-speech daemon with a stable REST API, model lifecycle management, and `systemd --user` operation.

## Current Runtime Status

- Default engine is `mock` for local development and API validation.
- Real `sherpa-onnx` Go runtime is integrated; use `config/config.sherpa.yaml` to run with native inference.
- KO/ZH/JA/EN runtime uses installed models from local manifest plus remote list from sherpa release.

## System Dependencies

Runtime dependencies for `./bin/tts` on Ubuntu:

- `libc6`
- `libstdc++6`
- `libgcc-s1`
- `ca-certificates` (for downloading remote models)

Optional runtime dependencies:

- `alsa-utils` (for local speaker playback via `aplay`)

Build-time dependencies (if building from source):

- `golang` (Go toolchain)
- `gcc` (required for cgo link step with sherpa runtime)
- `task` (Taskfile runner, only if using `build.sh` / Taskfile workflows)

Ubuntu install example:

```bash
sudo apt update
sudo apt install -y libc6 libstdc++6 libgcc-s1 ca-certificates alsa-utils golang gcc
```

## ONNX And Model URLs

- Sherpa ONNX TTS docs:
  - https://k2-fsa.github.io/sherpa/onnx/tts/index.html
- Sherpa ONNX TTS pretrained models:
  - https://k2-fsa.github.io/sherpa/onnx/tts/pretrained_models/index.html
  - https://github.com/k2-fsa/sherpa-onnx/releases/tag/tts-models

- Sherpa ONNX full TTS catalog:
  - https://k2-fsa.github.io/sherpa/onnx/tts/all/
- Sherpa ONNX Go bindings:
  - https://github.com/k2-fsa/sherpa-onnx-go

Pinned model artifact URLs used in this repo:

- Korean (Mimic3 VITS):
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-mimic3-ko_KO-kss_low.tar.bz2
- Chinese (Piper):
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-piper-zh_CN-huayan-medium.tar.bz2
- Japanese (Kokoro int8 multi-lang):
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-int8-multi-lang-v1_0.tar.bz2
- English (Kitten):
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2

## Sherpa Runtime `.so` Location

`linux-tts-onnx` currently builds with dynamic linking (not fully static).

Sherpa runtime shared libraries are provided by the Go module at:

- `$(go env GOPATH)/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@<version>/lib/x86_64-unknown-linux-gnu/`

Expected files:

- `libsherpa-onnx-c-api.so`
- `libonnxruntime.so`

Quick checks:

- list libs: `ls "$(go env GOPATH)"/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@*/lib/x86_64-unknown-linux-gnu`
- inspect linkage: `ldd ./bin/tts`

## Model Storage Paths

Downloaded models are stored under:

- `~/.local/share/tts-onnx/models`

Current language model paths:

- English:
  - `~/.local/share/tts-onnx/models/en/<version>/<model-id>/`
- Korean:
  - `~/.local/share/tts-onnx/models/ko/<version>/`
- Chinese:
  - `~/.local/share/tts-onnx/models/zh/<version>/`
- Japanese:
  - `~/.local/share/tts-onnx/models/ja/<version>/<model-id>/`

Voice and model asset files are inside each model directory, e.g.:

- `voices.bin`
- `tokens.txt`
- `model.onnx` or `model.fp16.onnx`
- `espeak-ng-data/`

Installed model state is tracked in:

- `~/.local/share/tts-onnx/models/manifest.json`

Quick checks:

- Installed models from service: `curl -fsS http://127.0.0.1:18741/v1/models`

## Quick Start

1. Build and run:
   - `go build -o ./bin/tts ./cmd/tts`
   - `./bin/tts --service --config ./config/config.example.yaml`
2. Check health:
   - `curl http://127.0.0.1:18741/v1/health`
3. Speak test:
   - `curl -X POST http://127.0.0.1:18741/v1/speak -H 'content-type: application/json' -d '{"text":"hello world","lang":"en","format":"wav"}' --output out.wav`
4. Speaker playback (service):
   - In `config/config.sherpa.yaml`, `play_on_speak: true` plays audio on the host speaker immediately when `/v1/speak` is called.

## CLI Arguments

Service mode (same `tts` binary):

- `--config` (default: `./config/config.sherpa.yaml`): config file path.
- `--service`: run HTTP daemon mode

Direct CLI (`cmd/tts/main.go`):

- All flags use double-dash form (example: `--voice-list`).
- Running `./bin/tts` with no arguments prints help automatically.
- `--lang` (optional for synthesis): explicit language bucket for model selection; if omitted, `tts` infers from `--voice` selector or falls back to first installed model
- `--voice` (optional): installed model selector (`id`/`version`) or numeric speaker id (`sid`)
- `--format` (default: `wav`): `wav|pcm_s16le`
- `--out` (optional): output path; file is written only when this is set
- `--config` (default: `./config/config.sherpa.yaml`)
- `--rate` (default: `1.0`)
- `--sample-rate` (default: `0`, optional override)
- `--request-id` (optional): correlation/cancel id
- `--no-play` (default: `false`): disable immediate speaker playback
- `--voice-list`: list installed models and voices (all languages by default, or one language with `--lang`); when voice names are unavailable, it shows numeric sid range
- `--remote-models`: list online TTS model packages from sherpa-onnx release
- `--install-remote-id`: download+extract model from remote list (language inferred from remote model ID)
- `--menu`: interactive language/model/voice selector
- `--auto-install` (default: `true`): used with `--menu` when selected model is not installed
- positional `text...`: synthesis input text

Examples:

- `./bin/tts "Sentence test without explicit language"`
- `./bin/tts --voice kitten-nano-en-v0_1-fp16 "Sentence test"`
- `./bin/tts --voice-list --lang en`
- `./bin/tts --remote-models --lang en`
- `./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16`
- `./bin/tts --menu`

If you want to run `tts` without `./bin/` prefix:

- `sudo ln -sf "$(pwd)/bin/tts" /usr/local/bin/tts`
- or add `./bin` to your `PATH`

Model selection behavior:

- If `--voice` matches an installed model `id` or `version`, that model is used for synthesis and language is inferred from that model.
- Otherwise, `--voice` is treated as numeric speaker id (`sid`) for the selected model.
- If no model is specified, `tts` uses the first installed model for selected language; when `--lang` is omitted, it uses the first installed model across all languages.

## Service API

Base URL: `http://127.0.0.1:18741/v1`

Auth behavior (`internal/httpapi/server.go`):

- If `bearer_token` is empty, no auth is required.
- If `bearer_token` is set, `/v1/models`, `/v1/models/install`, `/v1/models/{lang}/{version}`, `/v1/speak`, `/v1/stop`, and `/v1/metrics` require `Authorization: Bearer <token>`.
- `/v1/health`, `/v1/capabilities`, and `/` remain accessible without auth.

Endpoints:

- `GET /v1/health`
- `GET /v1/capabilities`
- `GET /v1/models`
- `POST /v1/models/install`
- `DELETE /v1/models/{lang}/{version}?force=true|false`
- `POST /v1/speak`
- `POST /v1/stop`
- `GET /v1/metrics`
- `GET /` (root sanity endpoint returning `{status,time}`)

Common request fields:

- `/v1/models/install`: `lang` (required), `url` (required), `model_id`, `checksum`, `version`
- `/v1/speak`: `text`, `lang` (optional), `voice`, `rate`, `format`, `sample_rate`, `request_id`
- `/v1/stop`: `request_id`

`/v1/speak` response:

- `200 OK` with binary audio body
- `Content-Type`: `audio/wav` or `application/octet-stream`
- `X-Sample-Rate`: output sample rate

## Multi-Language Long Sentence Test

- `bash ./test.sh`
- Current behavior: downloads KO/ZH/JA/EN test models and plays samples through speaker (`aplay`) without writing WAV files.

## Taskfile Workflows

- `task dev:run`
- `task test:unit`
- `task service:install-user-unit`
- `task service:enable`
- `task release:build`
- `task release:package VERSION=v0.1.0`

## Layout

- `cmd/tts`: single entrypoint for CLI + service mode
- `cmd/runtime-check`: native runtime visibility check helper
- `internal/httpapi`: REST handlers and error model
- `internal/modelmgr`: model install/delete + manifest
- `internal/synth`: synthesis queue/engine abstraction and mock audio generation
- `deploy/systemd`: `systemd --user` unit
- `cmd/tts`: direct CLI for non-service synthesis

## Docs

- Full API reference: `API_FULL.md`
