# オープンソースプロジェクトを応援してください 📚

このツールが役に立った場合は、ぜひご支援ください。

[**❤️ Polar.sh でサポート**](https://buy.polar.sh/polar_cl_2RRA7HD1Pv9AFP7pAZRg8XDomwZc9WCLfHXaW0hdJZz)

---

# Linux TTS Daemon (Go)

安定した REST API、モデルのライフサイクル管理、`systemd --user` 運用を備えた Ubuntu 優先のローカル TTS デーモンです。

## 現在のランタイム状況

- 既定エンジンはローカル開発と API 検証向けの `mock` です。
- 実際の `sherpa-onnx` Go ランタイムを統合済みで、ネイティブ推論には `config/config.sherpa.yaml` を使用します。
- KO/ZH/JA/EN ランタイムは、ローカル manifest のインストール済みモデルと sherpa リリースのリモート一覧を利用します。

## システム依存関係

Ubuntu で `./bin/tts` を実行するためのランタイム依存:

- `libc6`
- `libstdc++6`
- `libgcc-s1`
- `ca-certificates`（リモートモデルのダウンロード用）

任意のランタイム依存:

- `alsa-utils`（`aplay` によるローカルスピーカー再生）

ソースからビルドする場合の依存:

- `golang`（Go toolchain）
- `gcc`（sherpa ランタイムの cgo リンクに必要）
- `task`（`build.sh` / Taskfile ワークフローを使う場合）

Ubuntu インストール例:

```bash
sudo apt update
sudo apt install -y libc6 libstdc++6 libgcc-s1 ca-certificates alsa-utils golang gcc
```

## ONNX とモデル URL

- Sherpa ONNX TTS ドキュメント:
  - https://k2-fsa.github.io/sherpa/onnx/tts/index.html
- Sherpa ONNX TTS 学習済みモデル:
  - https://k2-fsa.github.io/sherpa/onnx/tts/pretrained_models/index.html
  - https://github.com/k2-fsa/sherpa-onnx/releases/tag/tts-models

- Sherpa ONNX 全 TTS カタログ:
  - https://k2-fsa.github.io/sherpa/onnx/tts/all/
- Sherpa ONNX Go バインディング:
  - https://github.com/k2-fsa/sherpa-onnx-go

このリポジトリで固定利用しているモデル URL:

- 韓国語（Mimic3 VITS）:
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-mimic3-ko_KO-kss_low.tar.bz2
- 中国語（Piper）:
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-piper-zh_CN-huayan-medium.tar.bz2
- 日本語（Kokoro int8 multi-lang）:
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-int8-multi-lang-v1_0.tar.bz2
- 英語（Kitten）:
  - https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2

## Sherpa ランタイム `.so` の場所

`linux-tts-onnx` は現在、動的リンク（完全静的ではない）でビルドされます。

Sherpa ランタイム共有ライブラリの場所:

- `$(go env GOPATH)/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@<version>/lib/x86_64-unknown-linux-gnu/`

想定ファイル:

- `libsherpa-onnx-c-api.so`
- `libonnxruntime.so`

クイックチェック:

- ライブラリ一覧: `ls "$(go env GOPATH)"/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@*/lib/x86_64-unknown-linux-gnu`
- 依存確認: `ldd ./bin/tts`

## モデル保存パス

ダウンロード済みモデルの保存先:

- `~/.local/share/tts-onnx/models`

言語ごとのモデルパス:

- 英語:
  - `~/.local/share/tts-onnx/models/en/<version>/<model-id>/`
- 韓国語:
  - `~/.local/share/tts-onnx/models/ko/<version>/`
- 中国語:
  - `~/.local/share/tts-onnx/models/zh/<version>/`
- 日本語:
  - `~/.local/share/tts-onnx/models/ja/<version>/<model-id>/`

各モデルディレクトリ内の主なアセット:

- `voices.bin`
- `tokens.txt`
- `model.onnx` または `model.fp16.onnx`
- `espeak-ng-data/`

インストール状態のトラッキング:

- `~/.local/share/tts-onnx/models/manifest.json`

クイックチェック:

- サービスからインストール済みモデル確認: `curl -fsS http://127.0.0.1:18741/v1/models`

## クイックスタート

1. ビルドと実行:
   - `go build -o ./bin/tts ./cmd/tts`
   - `./bin/tts --service --config ./config/config.example.yaml`
2. ヘルスチェック:
   - `curl http://127.0.0.1:18741/v1/health`
3. 音声合成テスト:
   - `curl -X POST http://127.0.0.1:18741/v1/speak -H 'content-type: application/json' -d '{"text":"hello world","lang":"en","format":"wav"}' --output out.wav`
4. スピーカー再生（サービス）:
   - `config/config.sherpa.yaml` で `play_on_speak: true` を設定すると、`/v1/speak` 呼び出し時にホストスピーカーへ即時再生します。

## CLI 引数

サービスモード（同一 `tts` バイナリ）:

- `--config`（既定: `./config/config.sherpa.yaml`）: 設定ファイルパス
- `--service`: HTTP デーモンモード

直接 CLI（`cmd/tts/main.go`）:

- すべてのフラグは `--` 形式（例: `--voice-list`）
- 引数なしで `./bin/tts` 実行時はヘルプ表示
- `--lang`（合成時は任意）: モデル選択用言語バケット。省略時は `--voice` または最初のインストールモデルから推論
- `--voice`（任意）: インストールモデル選択子（`id`/`version`）または数値スピーカー ID（`sid`）
- `--format`（既定: `wav`）: `wav|pcm_s16le`
- `--out`（任意）: 出力ファイルパス。指定時のみ保存
- `--config`（既定: `./config/config.sherpa.yaml`）
- `--rate`（既定: `1.0`）
- `--sample-rate`（既定: `0`, 任意上書き）
- `--request-id`（任意）: 相関/キャンセル ID
- `--no-play`（既定: `false`）: 即時スピーカー再生を無効化
- `--voice-list`: インストール済みモデル/ボイス一覧（既定は全言語、`--lang` で単一言語）。名前が無い場合は sid 範囲を表示
- `--remote-models`: sherpa-onnx リリースのオンライン TTS モデル一覧
- `--install-remote-id`: リモート一覧からダウンロード+展開（言語は remote model ID から推論）
- `--menu`: 対話型の言語/モデル/ボイス選択
- `--auto-install`（既定: `true`）: `--menu` で未インストールモデル選択時の自動インストール
- 位置引数 `text...`: 合成入力テキスト

例:

- `./bin/tts "Sentence test without explicit language"`
- `./bin/tts --voice kitten-nano-en-v0_1-fp16 "Sentence test"`
- `./bin/tts --voice-list --lang en`
- `./bin/tts --remote-models --lang en`
- `./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16`
- `./bin/tts --menu`

`./bin/` なしで `tts` を使う場合:

- `sudo ln -sf "$(pwd)/bin/tts" /usr/local/bin/tts`
- または `PATH` に `./bin` を追加

モデル選択の挙動:

- `--voice` がインストール済み `id` または `version` と一致すれば、そのモデルで合成し言語もそのモデルから推論
- それ以外は `--voice` を数値 speaker id（`sid`）として扱う
- モデル未指定時は選択言語の最初のインストールモデル、`--lang` 省略時は全言語中の最初のインストールモデル

## サービス API

Base URL: `http://127.0.0.1:18741/v1`

認証挙動（`internal/httpapi/server.go`）:

- `bearer_token` が空なら認証不要
- `bearer_token` が設定されている場合、`/v1/models`, `/v1/models/install`, `/v1/models/{lang}/{version}`, `/v1/speak`, `/v1/stop`, `/v1/metrics` は `Authorization: Bearer <token>` が必要
- `/v1/health`, `/v1/capabilities`, `/` は無認証でアクセス可能

エンドポイント:

- `GET /v1/health`
- `GET /v1/capabilities`
- `GET /v1/models`
- `POST /v1/models/install`
- `DELETE /v1/models/{lang}/{version}?force=true|false`
- `POST /v1/speak`
- `POST /v1/stop`
- `GET /v1/metrics`
- `GET /`（ルート確認エンドポイント、`{status,time}` を返す）

主なリクエストフィールド:

- `/v1/models/install`: `lang`（必須）, `url`（必須）, `model_id`, `checksum`, `version`
- `/v1/speak`: `text`, `lang`（任意）, `voice`, `rate`, `format`, `sample_rate`, `request_id`
- `/v1/stop`: `request_id`

`/v1/speak` 応答:

- `200 OK` + バイナリ音声ボディ
- `Content-Type`: `audio/wav` または `application/octet-stream`
- `X-Sample-Rate`: 出力サンプルレート

## 多言語ロングセンテンステスト

- `bash ./test.sh`
- 現在の挙動: KO/ZH/JA/EN テストモデルをダウンロードし、WAV 保存せずにスピーカー（`aplay`）再生を実行

## Taskfile ワークフロー

- `task dev:run`
- `task test:unit`
- `task service:install-user-unit`
- `task service:enable`
- `task release:build`
- `task release:package VERSION=v0.1.0`

## レイアウト

- `cmd/tts`: CLI + サービスモードの単一エントリポイント
- `cmd/runtime-check`: ネイティブランタイム可視性チェック
- `internal/httpapi`: REST ハンドラとエラーモデル
- `internal/modelmgr`: モデルインストール/削除 + manifest
- `internal/synth`: 合成キュー/エンジン抽象化と mock 音声生成
- `deploy/systemd`: `systemd --user` ユニット
- `cmd/tts`: 非サービス合成向け直接 CLI

## ドキュメント

- API 完全版: `API_FULL.md`
