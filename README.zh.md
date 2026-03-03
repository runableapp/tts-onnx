# 支持我们的开源项目 📚

如果你觉得我们的工具有帮助，欢迎支持我们！

[**❤️ 通过 Polar.sh 支持**](https://buy.polar.sh/polar_cl_2RRA7HD1Pv9AFP7pAZRg8XDomwZc9WCLfHXaW0hdJZz)

---

# Linux TTS Daemon (Go)

这是一个面向 Ubuntu 的本地文本转语音守护进程，提供稳定的 REST API、模型生命周期管理，以及 `systemd --user` 运行方式。

## 当前运行状态

- 默认引擎为 `mock`，用于本地开发和 API 验证。
- 已集成真实 `sherpa-onnx` Go 运行时；使用 `config/config.sherpa.yaml` 可启用原生推理。
- KO/ZH/JA/EN 运行时使用本地 manifest 的已安装模型，并结合 sherpa release 的远程模型列表。

## 系统依赖

在 Ubuntu 上运行 `./bin/tts` 的运行时依赖：

- `libc6`
- `libstdc++6`
- `libgcc-s1`
- `ca-certificates`（用于下载远程模型）

可选运行时依赖：

- `alsa-utils`（用于 `aplay` 本地扬声器播放）

从源码构建时依赖：

- `golang`（Go toolchain）
- `gcc`（sherpa 运行时 cgo 链接步骤需要）
- `task`（仅在使用 `build.sh` / Taskfile 工作流时需要）

Ubuntu 安装示例：

```bash
sudo apt update
sudo apt install -y libc6 libstdc++6 libgcc-s1 ca-certificates alsa-utils golang gcc
```

## ONNX 与模型 URL

- Sherpa ONNX TTS 文档：
  - https://k2-fsa.github.io/sherpa/onnx/tts/index.html
- Sherpa ONNX TTS 预训练模型：
  - https://k2-fsa.github.io/sherpa/onnx/tts/pretrained_models/index.html
  - https://github.com/k2-fsa/sherpa-onnx/releases/tag/tts-models

- Sherpa ONNX 完整 TTS 目录：
  - https://k2-fsa.github.io/sherpa/onnx/tts/all/
- Sherpa ONNX Go 绑定：
  - https://github.com/k2-fsa/sherpa-onnx-go

本仓库固定使用的模型包 URL：

- 韩语（Mimic3 VITS）：
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-mimic3-ko_KO-kss_low.tar.bz2
- 中文（Piper）：
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-piper-zh_CN-huayan-medium.tar.bz2
- 日语（Kokoro int8 multi-lang）：
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-int8-multi-lang-v1_0.tar.bz2
- 英语（Kitten）：
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2

## Sherpa 运行时 `.so` 位置

`linux-tts-onnx` 当前使用动态链接构建（非完全静态）。

Sherpa 运行时共享库位于 Go 模块路径：

- `$(go env GOPATH)/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@<version>/lib/x86_64-unknown-linux-gnu/`

预期文件：

- `libsherpa-onnx-c-api.so`
- `libonnxruntime.so`

快速检查：

- 列出库文件：`ls "$(go env GOPATH)"/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@*/lib/x86_64-unknown-linux-gnu`
- 检查链接：`ldd ./bin/tts`

## 模型存储路径

下载模型保存到：

- `~/.local/share/tts-onnx/models`

当前语言模型路径：

- 英语：
  - `~/.local/share/tts-onnx/models/en/<version>/<model-id>/`
- 韩语：
  - `~/.local/share/tts-onnx/models/ko/<version>/`
- 中文：
  - `~/.local/share/tts-onnx/models/zh/<version>/`
- 日语：
  - `~/.local/share/tts-onnx/models/ja/<version>/<model-id>/`

各模型目录中常见资产文件：

- `voices.bin`
- `tokens.txt`
- `model.onnx` 或 `model.fp16.onnx`
- `espeak-ng-data/`

已安装模型状态记录在：

- `~/.local/share/tts-onnx/models/manifest.json`

快速检查：

- 通过服务查看已安装模型：`curl -fsS http://127.0.0.1:18741/v1/models`

## 快速开始

1. 构建并运行：
   - `go build -o ./bin/tts ./cmd/tts`
   - `./bin/tts --service --config ./config/config.example.yaml`
2. 健康检查：
   - `curl http://127.0.0.1:18741/v1/health`
3. 语音合成测试：
   - `curl -X POST http://127.0.0.1:18741/v1/speak -H 'content-type: application/json' -d '{"text":"hello world","lang":"en","format":"wav"}' --output out.wav`
4. 扬声器播放（服务）：
   - 在 `config/config.sherpa.yaml` 中设置 `play_on_speak: true`，调用 `/v1/speak` 时将立即在主机扬声器播放。

## CLI 参数

服务模式（同一个 `tts` 二进制）：

- `--config`（默认：`./config/config.sherpa.yaml`）：配置文件路径
- `--service`：启动 HTTP 守护进程模式

直接 CLI（`cmd/tts/main.go`）：

- 所有参数均为双短横线形式（例如：`--voice-list`）
- 直接运行 `./bin/tts` 且无参数时会自动输出帮助
- `--lang`（合成时可选）：模型选择语言桶；省略时从 `--voice` 或首个已安装模型推断
- `--voice`（可选）：已安装模型选择器（`id`/`version`）或数字说话人 ID（`sid`）
- `--format`（默认：`wav`）：`wav|pcm_s16le`
- `--out`（可选）：输出路径；仅设置时写文件
- `--config`（默认：`./config/config.sherpa.yaml`）
- `--rate`（默认：`1.0`）
- `--sample-rate`（默认：`0`，可选覆盖）
- `--request-id`（可选）：关联/取消 ID
- `--no-play`（默认：`false`）：禁用即时扬声器播放
- `--voice-list`：列出已安装模型与声音（默认全部语言，或通过 `--lang` 指定单语言）；若无名字则显示 sid 范围
- `--remote-models`：列出 sherpa-onnx release 上的在线 TTS 模型
- `--install-remote-id`：按远程模型 ID 下载并解压（语言从远程模型 ID 推断）
- `--menu`：交互式语言/模型/声音选择
- `--auto-install`（默认：`true`）：在 `--menu` 中选中未安装模型时自动安装
- 位置参数 `text...`：待合成文本

示例：

- `./bin/tts "Sentence test without explicit language"`
- `./bin/tts --voice kitten-nano-en-v0_1-fp16 "Sentence test"`
- `./bin/tts --voice-list --lang en`
- `./bin/tts --remote-models --lang en`
- `./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16`
- `./bin/tts --menu`

若希望不带 `./bin/` 前缀运行 `tts`：

- `sudo ln -sf "$(pwd)/bin/tts" /usr/local/bin/tts`
- 或将 `./bin` 加入 `PATH`

模型选择行为：

- 若 `--voice` 匹配已安装模型 `id` 或 `version`，则使用该模型并从模型推断语言
- 否则将 `--voice` 作为当前模型的数字说话人 ID（`sid`）
- 未指定模型时，`tts` 使用所选语言的首个已安装模型；若 `--lang` 也省略，则使用所有语言中的首个已安装模型

## 服务 API

Base URL：`http://127.0.0.1:18741/v1`

认证行为（`internal/httpapi/server.go`）：

- `bearer_token` 为空时不需要认证
- 设置 `bearer_token` 后，`/v1/models`、`/v1/models/install`、`/v1/models/{lang}/{version}`、`/v1/speak`、`/v1/stop`、`/v1/metrics` 需要 `Authorization: Bearer <token>`
- `/v1/health`、`/v1/capabilities`、`/` 无需认证

端点：

- `GET /v1/health`
- `GET /v1/capabilities`
- `GET /v1/models`
- `POST /v1/models/install`
- `DELETE /v1/models/{lang}/{version}?force=true|false`
- `POST /v1/speak`
- `POST /v1/stop`
- `GET /v1/metrics`
- `GET /`（根路径检查端点，返回 `{status,time}`）

常见请求字段：

- `/v1/models/install`：`lang`（必填）、`url`（必填）、`model_id`、`checksum`、`version`
- `/v1/speak`：`text`、`lang`（可选）、`voice`、`rate`、`format`、`sample_rate`、`request_id`
- `/v1/stop`：`request_id`

`/v1/speak` 响应：

- `200 OK` + 二进制音频
- `Content-Type`：`audio/wav` 或 `application/octet-stream`
- `X-Sample-Rate`：输出采样率

## 多语言长句测试

- `bash ./test.sh`
- 当前行为：下载 KO/ZH/JA/EN 测试模型，并通过扬声器（`aplay`）播放，不写入 WAV 文件

## Taskfile 工作流

- `task dev:run`
- `task test:unit`
- `task service:install-user-unit`
- `task service:enable`
- `task release:build`
- `task release:package VERSION=v0.1.0`

## 项目结构

- `cmd/tts`：CLI + 服务模式统一入口
- `cmd/runtime-check`：原生运行时可见性检查工具
- `internal/httpapi`：REST 处理器与错误模型
- `internal/modelmgr`：模型安装/删除 + manifest
- `internal/synth`：合成队列/引擎抽象与 mock 音频生成
- `deploy/systemd`：`systemd --user` 单元
- `cmd/tts`：非服务模式直接 CLI

## 文档

- 完整 API 参考：`API_FULL.md`
