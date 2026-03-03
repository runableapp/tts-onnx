# Linux TTS Daemon API (Full)

Version: `v1`  
Base URL: `http://127.0.0.1:18741/v1`

## Overview

- Local REST API for Linux TTS runtime.
- Supported output formats: `wav`, `pcm_s16le`.
- Language is model-driven (for example `ko`, `zh`, `ja`, `en` when installed).
- Errors return stable machine-readable JSON codes.
- Optional speaker playback on `/speak` when `play_on_speak: true` in config.

## Authentication

- Default: no token required.
- If `bearer_token` is set in config:
  - `/v1/models`, `/v1/models/install`, `/v1/models/{lang}/{version}`, `/v1/speak`, `/v1/stop`, `/v1/metrics` require `Authorization: Bearer <token>`
  - `/v1/health`, `/v1/capabilities`, and `/` remain accessible.

## Content Types

- Requests: `application/json`
- `/speak` success:
  - `audio/wav` for WAV
  - `application/octet-stream` for PCM (`pcm_s16le`)

## Endpoints

### 0) `GET /`

Root sanity endpoint.

Example response:

```json
{
  "status": "ok",
  "time": "2026-03-02T12:34:56Z"
}
```

---

### 1) `GET /health`

Service liveness and runtime identity.

Example response:

```json
{
  "status": "ok",
  "runtime": "sherpa-onnx",
  "uptime": "32.152s"
}
```

---

### 2) `GET /capabilities`

Runtime + installed model summary.

Example response:

```json
{
  "status": "ok",
  "runtime": "sherpa-onnx",
  "languages_supported": ["en", "ja", "ko", "zh"],
  "models_loaded": {
    "en": "v0_1-fp16",
    "ja": "v1_0"
  },
  "max_text_chars": 1200
}
```

Notes:

- `languages_supported` is derived from currently installed model buckets.
- `models_loaded` maps each installed language to the first installed version.

---

### 3) `GET /models`

Returns installed model manifest JSON as stored by the service.

Example response:

```json
{
  "updated_at": "2026-03-02T10:12:34Z",
  "installed": {
    "en": [
      {
        "id": "kitten-nano-en-v0_1-fp16",
        "version": "v0_1-fp16",
        "sha256": "eee6fbec2231ddf8ecce8c9cf0a03fd37e19cef08a19118090a73b21c2a9bf56",
        "license": "",
        "attribution": "",
        "installed_at": "2026-03-02T10:10:00Z",
        "path": "/home/user/.local/share/tts-onnx/models/en/v0_1-fp16"
      }
    ]
  }
}
```

---

### 4) `POST /models/install`

Download + verify + extract + register a model.

Request fields:

- `lang` (required): target language bucket (e.g. `ko`, `zh`, `ja`, `en`)
- `url` (required): direct model archive URL
- `model_id` (optional): metadata id stored in manifest
- `checksum` (optional): SHA256; if set and mismatched, install fails
- `version` (optional): explicit version label (auto-generated if omitted)

Example request:

```json
{
  "lang": "en",
  "model_id": "kitten-nano-en-v0_1-fp16",
  "url": "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2"
}
```

Success response:

- HTTP `201`
- response body is the installed model entry.

---

### 5) `DELETE /models/{lang}/{version}?force=true|false`

Delete model version from manifest and disk.

Example:

```bash
curl -X DELETE "http://127.0.0.1:18741/v1/models/en/v0_1-fp16?force=false"
```

Response:

```json
{"status":"ok"}
```

---

### 6) `POST /speak`

Generate speech audio from text.

Request fields:

- `text` (required)
- `lang` (optional): if omitted, service resolves from `voice` or first installed model
- `voice` (optional): model selector (`id`/`version`) or model speaker selector (`sid` etc.)
- `rate` (optional, default `1.0`)
- `format` (optional, default `wav`): `wav|pcm_s16le`
- `sample_rate` (optional)
- `request_id` (optional; used by `/stop`)

Example request:

```json
{
  "text": "Hello from Linux TTS daemon.",
  "lang": "en",
  "format": "wav",
  "rate": 1.0,
  "request_id": "req-001"
}
```

Success response:

- HTTP `200`
- binary audio body
- headers:
  - `Content-Type: audio/wav` or `application/octet-stream`
  - `X-Sample-Rate: <int>`

---

### 7) `POST /stop`

Best-effort cancellation by request ID.

Request:

```json
{
  "request_id": "req-001"
}
```

Response:

```json
{
  "stopped": true
}
```

---

### 8) `GET /metrics`

Prometheus-style counters.

Example response body:

```text
tts_requests_total 124
tts_requests_failed_total 2
```

## Error Model

All errors return JSON:

```json
{
  "code": "MODEL_MISSING",
  "message": "installed model required: no installed model",
  "retryable": false
}
```

Error codes in current API layer:

- `MODEL_MISSING`
- `INVALID_LANGUAGE`
- `SYNTH_TIMEOUT`
- `BAD_REQUEST`
- `INTERNAL_ERROR`
- `UNAUTHORIZED`

Possible HTTP mapping:

- `400`: invalid payload / method not allowed / missing required fields
- `401`: missing or invalid bearer token when token mode enabled
- `404`: model/version not found or model missing for synthesis
- `500`: runtime failure, download/extract failure, internal error
- `504`: synthesis timeout

## Example cURL Flows

### Install + speak (EN)

```bash
curl -X POST http://127.0.0.1:18741/v1/models/install \
  -H "content-type: application/json" \
  -d '{
    "lang":"en",
    "model_id":"kitten-nano-en-v0_1-fp16",
    "url":"https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2"
  }'

curl -X POST http://127.0.0.1:18741/v1/speak \
  -H "content-type: application/json" \
  -d '{"text":"Hello world","lang":"en","format":"wav"}' \
  --output en.wav
```

### Speak (KO, ZH, JA)

```bash
curl -X POST http://127.0.0.1:18741/v1/speak \
  -H "content-type: application/json" \
  -d '{"text":"안녕하세요. 한국어 테스트입니다.","lang":"ko","format":"wav"}' \
  --output ko.wav

curl -X POST http://127.0.0.1:18741/v1/speak \
  -H "content-type: application/json" \
  -d '{"text":"这是中文语音合成测试。","lang":"zh","format":"wav"}' \
  --output zh.wav

curl -X POST http://127.0.0.1:18741/v1/speak \
  -H "content-type: application/json" \
  -d '{"text":"こんにちは。日本語テストです。","lang":"ja","format":"wav"}' \
  --output ja.wav
```
