# TTS ONNX 手册（使用与配置）

本指南用于在运行时使用和配置 `linux-tts-onnx`。  
不包含源码编译/构建内容。

---

### 1) 本手册前提

- 你已经有可运行的 `tts` 二进制（例如 `./bin/tts` 或 `/usr/local/bin/tts`）。
- 你在 Linux 环境下（示例使用 Ubuntu 风格命令）。

### 2) 运行时依赖

安装运行时包：

```bash
sudo apt update
sudo apt install -y libc6 libstdc++6 libgcc-s1 ca-certificates alsa-utils
```

说明：

- `ca-certificates`：用于下载远程模型。
- `alsa-utils`：提供扬声器播放所需的 `aplay`。

### 3) 默认运行时路径

`tts-onnx` 数据默认存储在：

- Models: `~/.local/share/tts-onnx/models`
- Manifest: `~/.local/share/tts-onnx/models/manifest.json`
- State: `~/.local/state/tts-onnx`
- Cache: `~/.cache/tts-onnx`

### 4) 首次设置（CLI 模式）

如果你希望从干净状态开始：

```bash
rm -rf ~/.local/share/tts-onnx
```

安装推荐模型：

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
./bin/tts --install-remote-id vits-mimic3-ko_KO-kss_low
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

模型下载 URL：

- 全部发布列表：`https://github.com/k2-fsa/sherpa-onnx/releases/tag/tts-models`
- 英语 (`kitten-nano-en-v0_1-fp16`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2`
- 韩语 (`vits-mimic3-ko_KO-kss_low`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-mimic3-ko_KO-kss_low.tar.bz2`
- 日语多语种 (`kokoro-multi-lang-v1_0`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-multi-lang-v1_0.tar.bz2`

通过 URL 手动安装（不使用 `--install-remote-id`）：

1. 启动服务：

```bash
./bin/tts --service --config ./config/config.sherpa.yaml
```

2. 通过 API 使用显式 URL 安装模型：

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

3. 验证：

```bash
curl -fsS http://127.0.0.1:18741/v1/models
```

### 4.1) 远程模型发现（`--remote-models`）

列出所有远程模型：

```bash
./bin/tts --remote-models
```

按单一语言过滤：

```bash
./bin/tts --remote-models --lang en
./bin/tts --remote-models --lang ko
./bin/tts --remote-models --lang ja
```

这里 `--lang` 是可选的。省略可显示所有远程模型。

输出字段：

- `lang`：推断语言标签（或 `unknown`）
- `id`：模型包 ID（配合 `--install-remote-id`）
- `version`：推断版本/标签
- `url`：直接下载 URL

若 `lang=unknown`，请显式安装：

```bash
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

### 4.2) 模型下载/安装方式

方式 A（推荐）：按模型 ID 安装

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
./bin/tts --install-remote-id vits-mimic3-ko_KO-kss_low
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

方式 B：显式 URL 安装（服务 API）

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

### 4.3) 下载失败与重试

如果出现临时网络/CDN 错误，例如：

- `download failed with status 503`

请稍后重试相同命令：

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
```

安装后也可通过以下命令确认模型文件是否可用：

```bash
./bin/tts --voice-list --lang en
./bin/tts --voice-list --lang ko
./bin/tts --voice-list --lang ja
```

为什么某些安装仍需要 `--lang`：

- 某些远程模型 ID 是多语言或显示 `lang=unknown`。
- 传入 `--lang <language>` 可指定安装目标语言桶。

### 5) 验证已安装语音/模型

```bash
./bin/tts --voice-list
./bin/tts --voice-list --lang en
./bin/tts --voice-list --lang ko
./bin/tts --voice-list --lang ja
```

### 6) 语音合成（扬声器播放）

基础播放（不写输出文件）：

```bash
./bin/tts "Hello, this is a test."
./bin/tts --lang ko "안녕하세요. 테스트입니다."
./bin/tts --lang ja "こんにちは。テストです。"
```

合成时 `--lang` 可选。省略时会根据 `--voice` 或首个已安装模型自动推断。

固定指定模型：

```bash
./bin/tts --voice kitten-nano-en-v0_1-fp16 "English test"
./bin/tts --voice vits-mimic3-ko_KO-kss_low "한국어 테스트"
./bin/tts --voice v1_0 "日本語テスト"
```

模型支持多说话人时可用数字 speaker ID (`sid`)：

```bash
./bin/tts --lang ko --voice 6 "speaker id six test"
```

### 7) 可选输出文件

仅在需要时写文件：

```bash
./bin/tts --out ./out.wav "save this audio"
```

### 8) 服务设置（不含构建步骤）

准备用户配置：

```bash
mkdir -p ~/.config/tts-onnx
cp ./config/config.sherpa.yaml ~/.config/tts-onnx/config.yaml
```

安装并启动用户服务：

```bash
bash ./scripts/install-user-unit.sh
bash ./scripts/enable-user-service.sh
```

健康检查：

```bash
curl -fsS http://127.0.0.1:18741/v1/health
```

查看日志：

```bash
journalctl --user -u tts-onnx.service -f
```

停止/禁用服务：

```bash
bash ./scripts/disable-user-service.sh
```

### 9) 服务 API 快速使用

语音请求：

```bash
curl -X POST http://127.0.0.1:18741/v1/speak \
  -H 'content-type: application/json' \
  -d '{"text":"hello world","lang":"en","format":"wav"}' \
  --output out.wav
```

列出已安装模型：

```bash
curl -fsS http://127.0.0.1:18741/v1/models
```

### 10) 故障排查

- **`aplay not found in PATH`**
  - 安装 `alsa-utils`。
- **`cannot infer language for remote model ...`**
  - 在安装命令中添加 `--lang <language>`。
- **`no supported model file found in ...`**
  - 通过 `--install-remote-id` 重新安装模型。
- **扬声器没有声音**
  - 检查 Linux 音频输出设备与 `aplay` 设备。
