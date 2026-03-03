# TTS ONNX マニュアル（利用と設定）

このガイドは `linux-tts-onnx` をランタイムで利用・設定するためのものです。  
ソースコードのコンパイル/ビルド手順は含みません。

---

### 1) このマニュアルの前提

- 実行可能な `tts` バイナリがあること（例: `./bin/tts` または `/usr/local/bin/tts`）。
- Linux 環境であること（コマンド例は Ubuntu 形式）。

### 2) ランタイム依存関係

ランタイムパッケージをインストール:

```bash
sudo apt update
sudo apt install -y libc6 libstdc++6 libgcc-s1 ca-certificates alsa-utils
```

補足:

- `ca-certificates` はリモートモデルのダウンロードに必要です。
- `alsa-utils` はスピーカー再生の `aplay` を提供します。

### 3) 既定のランタイムパス

`tts-onnx` データは以下に保存されます:

- Models: `~/.local/share/tts-onnx/models`
- Manifest: `~/.local/share/tts-onnx/models/manifest.json`
- State: `~/.local/state/tts-onnx`
- Cache: `~/.cache/tts-onnx`

### 4) 初期セットアップ（CLI モード）

クリーンな状態から始める場合:

```bash
rm -rf ~/.local/share/tts-onnx
```

推奨モデルをインストール:

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
./bin/tts --install-remote-id vits-mimic3-ko_KO-kss_low
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

モデルダウンロード URL:

- リリース一覧: `https://github.com/k2-fsa/sherpa-onnx/releases/tag/tts-models`
- 英語 (`kitten-nano-en-v0_1-fp16`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2`
- 韓国語 (`vits-mimic3-ko_KO-kss_low`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-mimic3-ko_KO-kss_low.tar.bz2`
- 日本語マルチ言語 (`kokoro-multi-lang-v1_0`):
  - `https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-multi-lang-v1_0.tar.bz2`

URL 指定の手動インストール（`--install-remote-id` なし）:

1. サービス起動:

```bash
./bin/tts --service --config ./config/config.sherpa.yaml
```

2. API で URL を指定してモデル導入:

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

3. 確認:

```bash
curl -fsS http://127.0.0.1:18741/v1/models
```

### 4.1) リモートモデル探索（`--remote-models`）

すべてのリモートモデルを表示:

```bash
./bin/tts --remote-models
```

言語ごとに表示:

```bash
./bin/tts --remote-models --lang en
./bin/tts --remote-models --lang ko
./bin/tts --remote-models --lang ja
```

ここで `--lang` は任意です。省略すると全モデルを表示します。

出力フィールド:

- `lang`: 推定言語タグ（または `unknown`）
- `id`: モデルパッケージ ID（`--install-remote-id` で使用）
- `version`: 推定バージョン/タグ
- `url`: 直接ダウンロード URL

`lang=unknown` の場合は `--lang` を指定して導入:

```bash
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

### 4.2) モデルのダウンロード/導入方法

方法 A（推奨）: モデル ID で導入

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
./bin/tts --install-remote-id vits-mimic3-ko_KO-kss_low
./bin/tts --lang ja --install-remote-id kokoro-multi-lang-v1_0
```

方法 B: URL を直接指定（サービス API）

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

### 4.3) ダウンロード失敗と再試行

以下のような一時的なネットワーク/CDN エラーが出た場合:

- `download failed with status 503`

少し待って同じコマンドを再実行:

```bash
./bin/tts --install-remote-id kitten-nano-en-v0_1-fp16
```

導入後は次で確認できます:

```bash
./bin/tts --voice-list --lang en
./bin/tts --voice-list --lang ko
./bin/tts --voice-list --lang ja
```

一部の導入で `--lang` が必要な理由:

- 一部のモデル ID は多言語または `lang=unknown` です。
- `--lang <language>` を渡すと導入先言語バケットを指定できます。

### 5) インストール済みモデル/ボイス確認

```bash
./bin/tts --voice-list
./bin/tts --voice-list --lang en
./bin/tts --voice-list --lang ko
./bin/tts --voice-list --lang ja
```

### 6) 音声合成（スピーカー再生）

基本再生（出力ファイルなし）:

```bash
./bin/tts "Hello, this is a test."
./bin/tts --lang ko "안녕하세요. 테스트입니다."
./bin/tts --lang ja "こんにちは。テストです。"
```

合成時の `--lang` は任意です。省略すると `--voice` または最初のインストールモデルから推定されます。

特定モデルを指定:

```bash
./bin/tts --voice kitten-nano-en-v0_1-fp16 "English test"
./bin/tts --voice vits-mimic3-ko_KO-kss_low "한국어 테스트"
./bin/tts --voice v1_0 "日本語テスト"
```

マルチスピーカーモデルで数値 `sid` を使用:

```bash
./bin/tts --lang ko --voice 6 "speaker id six test"
```

### 7) 任意の出力ファイル

必要な場合のみ保存:

```bash
./bin/tts --out ./out.wav "save this audio"
```

### 8) サービス設定（ビルド手順なし）

ユーザー設定の準備:

```bash
mkdir -p ~/.config/tts-onnx
cp ./config/config.sherpa.yaml ~/.config/tts-onnx/config.yaml
```

ユーザーサービスを導入・起動:

```bash
bash ./scripts/install-user-unit.sh
bash ./scripts/enable-user-service.sh
```

ヘルスチェック:

```bash
curl -fsS http://127.0.0.1:18741/v1/health
```

ログ確認:

```bash
journalctl --user -u tts-onnx.service -f
```

停止/無効化:

```bash
bash ./scripts/disable-user-service.sh
```

### 9) サービス API のクイック利用

音声合成リクエスト:

```bash
curl -X POST http://127.0.0.1:18741/v1/speak \
  -H 'content-type: application/json' \
  -d '{"text":"hello world","lang":"en","format":"wav"}' \
  --output out.wav
```

インストール済みモデル一覧:

```bash
curl -fsS http://127.0.0.1:18741/v1/models
```

### 10) トラブルシューティング

- **`aplay not found in PATH`**
  - `alsa-utils` をインストール
- **`cannot infer language for remote model ...`**
  - 導入コマンドに `--lang <language>` を追加
- **`no supported model file found in ...`**
  - `--install-remote-id` で再導入
- **スピーカーから音が出ない**
  - Linux の出力デバイス設定と `aplay` デバイスを確認
