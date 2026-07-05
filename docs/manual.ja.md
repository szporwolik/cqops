---
title: CQOps ユーザーマニュアル
description: CQOps のインストール、設定、利用方法 — 高速なターミナルファーストのアマチュア無線ロガー
---

> **翻訳メモ:** この翻訳は LLM モデルで生成されています。修正は `dev` ブランチへの Pull Request として歓迎します。CQOps の画面と一致させるため、一部の画面名、フィールド名、コマンド、ショートカットは意図的に英語のまま残しています。

# CQOps ユーザーマニュアル

CQOps は、低いシステム負荷で信頼できるキーボードログを行いたいオペレーター向けの、高速なターミナルファーストのアマチュア無線ロガーです。シャック、ポータブル運用、クラブ局、Field Day、Raspberry Pi クラスの機器や古いノート PC での使用を想定しています。

CQOps は常に QSO を最初にローカルへ保存します。インターネットを使う連携機能は任意です。

## 目次

1. [CQOps とは](#cqops-とは)
2. [ダウンロードとインストール](#ダウンロードとインストール)
3. [初回起動](#初回起動)
4. [最初の QSO を記録する](#最初の-qso-を記録する)
5. [メイン画面](#メイン画面)
6. [よく使うワークフロー](#よく使うワークフロー)
7. [QSO ログ入力](#qso-ログ入力)
8. [Logbook Editor と ADIF](#logbook-editor-と-adif)
9. [コンテスト](#コンテスト)
10. [Favorites、リファレンス、Band Plan](#favoritesリファレンスband-plan)
11. [連携機能](#連携機能)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [設定](#設定)
14. [キーボードショートカット](#キーボードショートカット)
15. [トラブルシューティング](#トラブルシューティング)
16. [バグ報告](#バグ報告)

---

## CQOps とは

CQOps は、QSO の高速入力、ローカルファーストのログ保存、実用的なフィールド運用を中心に設計されています。

### 主な考え方

- **Terminal-first** — キーボード操作に最適化。
- **Offline-first** — ローカル QSO ログはインターネットなしで動作。
- **低負荷** — Raspberry Pi クラス、古いノート PC、共有ステーション PC に適合。
- **ポータブル設計** — 単一の Go バイナリとして配布。
- **複数 logbook** — 個人、ポータブル、コンテスト、クラブ用ログに便利。
- **複数 operator** — hot-seat やクラブ局の共有運用に便利。
- **複数 rig** — 各 rig preset は独自の backend と WSJT-X 設定を保持可能。
- **任意の連携** — QRZ.com、Wavelog、DX Cluster、PSK Reporter、APRS、GPS受信機、rig control、rotor control、solar data、CQOps Live ブラウザ dashboard。

ローカルログにはインターネット接続は不要です。ネットワーク機能は `--offline` モードではスキップされます。

### CQOps が向いているユーザー

CQOps は次の用途に適しています。

- ポータブル運用者。
- SOTA / POTA アクティベーター。
- クラブ局。
- Field Day チーム。
- ターミナル操作を好むオペレーター。
- operator、logbook、rig を素早く切り替える必要がある局。

CQOps は、フル機能のデスクトップロガーや Web ベースの logbook プラットフォームを完全に置き換えるものではありません。高速なターミナルログ、フィールド運用、オフライン利用、共有ステーション運用に集中しています。

---

## ダウンロードとインストール

すべてのリリース:

<https://github.com/szporwolik/cqops/releases>

### Windows

| パッケージ | リンク | 注記 |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | ほとんどのユーザーに推奨。CQOps を Start Menu と PATH に追加します。 |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | 展開してインストールなしで実行します。 |

古いコンソールではなく **Windows Terminal** を使用してください。

### Linux — Debian / Ubuntu

| アーキテクチャ | リンク | 用途 |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | ほとんどの Intel/AMD PC |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-bit ARM システム |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-bit Raspberry Pi OS |

ダウンロードしたパッケージをインストールします。

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — ポータブル tarball

| アーキテクチャ | リンク | 用途 |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | ほとんどの Intel/AMD PC |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-bit ARM システム |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-bit Raspberry Pi OS |

### macOS

| アーキテクチャ | リンク | 用途 |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | M1/M2/M3 Mac |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Intel Mac |

手動インストール:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### ソースからビルド

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

ソースビルドには Go 1.26 以降が必要です。

### ターミナル要件

| 要件 | 値 |
|---|---|
| 最小ターミナルサイズ | 80×24 文字 |
| 推奨ターミナルサイズ | 80×43 文字以上 |
| 推奨 Windows ターミナル | Windows Terminal |

### 基本コマンド

```bash
cqops              # TUI を起動
cqops --offline    # ネットワーク活動なしで起動
cqops --version    # バージョンを表示して終了
cqops --help       # ヘルプを表示
```

---

## 初回起動

初回起動時、CQOps は setup wizard を開きます。ローカルログに必要なのは基本的な局情報だけです。ネットワーク連携はスキップして後で設定できます。

### wizard ページ

| ページ | 設定内容 |
|---|---|
| Station & Logbook | 初期 logbook、局コールサイン、operator、grid locator、任意の references と zones、Wavelog URL/API/station profile ID |
| Rig | rig preset、model、antenna、power、backend、任意の rotor、任意の WSJT-X UDP 設定 |
| Integrations | QRZ.com lookup settings |
| General | IANA timezone |
| Summary | 確認と保存 |

対応する rig backend:

- None,
- flrig,
- Hamlib `rigctld`.

### wizard の操作

| キー | 操作 |
|---|---|
| Ctrl+S | 検証して次へ進む。Summary では保存して CQOps を開始 |
| Esc | 戻る |
| F10 | 終了 |
| Tab / Shift+Tab | フィールド間を移動 |
| Space | checkbox を切り替え |

wizard の設定は後で **F9** から変更できます。

---

## 最初の QSO を記録する

1. CQOps を起動します。

   ```bash
   cqops
   ```

2. setup wizard で少なくとも自局のコールサインと grid locator を入力します。
3. **F1** で QSO form を開きます。
4. 相手局のコールサインを入力します。CQOps はコールサインを自動的に大文字化します。
5. 残りの項目を入力します。active rig が flrig または Hamlib で接続されている場合、CQOps は frequency、band、mode、submode を自動入力できます。
6. **Enter** または **Ctrl+S** で保存します。
7. **DUPE!** 警告が出た場合、もう一度 **Enter** でそれでも保存、または **Esc** でキャンセルします。

保存された QSO はすぐに Recent QSOs テーブルに表示されます。

---

## メイン画面

CQOps は固定のターミナルレイアウトを使います。

```text
┌─ Status Bar ───────────────────────────────────────────────────────────────────┐
│  CQOps v0.8.9  Log Portable  Rig FTDx10  Call SP9MOA/P                         │
│  Net WSJT Hamlib DXC WL                                           23:00L 2100Z │
├─ Tab Bar ──────────────────────────────────────────────────────────────────────┤
│  F1 QSO   F2 QRZ   F4 DXC   F5 HRD   F6 REF   F7 BPL   F8 LOG   F9 CFG         │
├─ Main Content Area ────────────────────────────────────────────────────────────┤
│  QSO form, partner view, map, editor, dashboard data, or active screen content  │
├─ Help Bar ─────────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                              │
└────────────────────────────────────────────────────────────────────────────────┘
```

### Status bar

Status bar には以下が表示されます。

- CQOps バージョン。
- active logbook。
- active rig。
- station callsign。
- active operator。
- integration status labels。
- `L` で示されるローカル時刻。
- `Z` で示される UTC 時刻。

よく使われるラベルには **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL**, **GPS** があります。GPSラベルは同じ色規則に従います — 切断時は赤、接続時でフィックスなしは黄色、位置取得時は白です。

| 色 | 意味 |
|---|---|
| 白/デフォルト | 接続済みまたは有効 |
| 黄 | 無効、接続中、または想定された offline |
| 赤 | エラーまたは未接続 |
| アクセント + 太字 | WSJT-X が送信中 |

### メインタブ

| キー | タブ | 画面 |
|---|---|---|
| F1 | QSO | QSO form と Recent QSOs |
| F2 | QRZ | Partner view: callbook data、map、stats、photo |
| F4 | DXC | DX Cluster spots と filters |
| F5 | HRD | PSK Reporter spots と propagation map |
| F6 | REF | SOTA/POTA/WWFF/IOTA reference search |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor、ADIF、Wavelog sync |
| F9 | CFG | Configuration menus |

Help bar はアクティブ画面に関連するショートカットを表示します。**?** で全体ヘルプを開きます。

---

## よく使うワークフロー

### Portable、SOTA、POTA 運用

出発前:

1. インターネット接続ありで CQOps を一度起動します。
2. solar data、REF data、DXCC prefixes などのキャッシュを CQOps に取得または更新させます。
3. Solar panel にデータが表示されることを確認します。
4. **F6** の REF search が結果を返すことを確認します。

現地で:

1. CQOps を offline mode で起動します。

   ```bash
   cqops --offline
   ```

2. 通常どおりログします。QSOs はローカルに保存されます。
3. オンラインに戻ったら **F8** を開き、**w** を押して未送信 QSOs を Wavelog にアップロードします。

### 共有クラブ局と hot-seat logging

1. **F9 → Operators** を開きます。
2. **Ins** で operator profiles を追加します。
3. QSO form で **Ctrl+O** を押して active operator を切り替えます。
4. 保存前に Status bar で active operator を確認します。
5. 複数 operator が似た内容の QSO を入力する場合は、全フォームを再入力しないよう **Retain** を使います。

active operator は ADIF の `OPERATOR` フィールドに保存されます。

### 個人 logbook とクラブ logbook

1. **F9 → Logbooks** を開きます。
2. **Ins** で各 logbook を作成します。
3. QSO form で **Ctrl+L** を押して active logbook を切り替えます。
4. 保存前に Status bar で active logbook を確認します。

各 logbook は独自の station details、Wavelog settings、contest settings、operators を保持できます。

### 複数 rig

1. **F9 → Rigs** を開きます。
2. **Ins** で rig presets を作成します。
3. backend を選択します: None、flrig、Hamlib。
4. QSO form で **Ctrl+R** を押して active rig を切り替えます。

rig preset には backend、model、antenna、power、rotor settings、WSJT-X UDP settings を含められます。

### WSJT-X デジタル運用

WSJT-X UDP integration が有効な場合、CQOps は WSJT-X から ADIF メッセージを受信し、完了したデジタル QSO を自動ログできます。

自動ログされた QSOs は:

- active logbook に保存されます。
- すぐに Recent QSOs に表示されます。
- 重複をスキップします。
- active contest ID を継承します。
- Wavelog が設定済みで到達可能な場合、自動アップロードできます。

WSJT-X が報告する operator が CQOps の active operator と一致しない場合、CQOps は警告を表示します。

長時間のデジタルセッション前に確認するもの:

- active logbook。
- active operator。
- active contest。
- WSJT-X status label。

### Wavelog sync

CQOps は QSO をまずローカルに保存します。Wavelog sync は任意です。

| 操作 | 場所 | ショートカット | 注記 |
|---|---|---|---|
| 未送信 QSOs をアップロード | Logbook Editor | `w` | 50 件単位でアップロード |
| Wavelog からダウンロード | Logbook Editor | `Ctrl+W` | `last_fetched_id` による差分ダウンロード |

アップロード状態は QSO ごとに追跡されます。

- not sent,
- sent,
- error.

アップロードに失敗しても QSO はローカル logbook に残り、後で再試行できます。logbook を purge すると fetch ID が `0` に戻り、完全再ダウンロードが可能になります。

---

## QSO ログ入力

QSO form はメインのログ入力画面です。**F1** で開きます。

CQOps は次の情報源からフィールドを入力できます。

| 情報源 | フィールド |
|---|---|
| flrig / Hamlib | Frequency、split 時の Freq RX、mode、submode |
| QRZ.com | Name、QTH、grid、country、CQ zone、ITU zone、DXCC、continent |
| REF database | SOTA、POTA、WWFF、IOTA references |
| Wavelog lookup | 設定時の worked/confirmed status |
| DXCC/prefix data | prefix と country 関連データ |

### フォーム配置

| 左列 | 中央列 | 右列 |
|---|---|---|
| Date UTC | Mode | Power W |
| Time UTC | Submode | Freq RX |
| Call | Name | SOTA Ref |
| RST sent | QTH | POTA Ref |
| RST rcvd | Grid | WWFF Ref |
| Frequency MHz | Country | IOTA |
| Band | SIG | SIG Info |
| Exch sent |  |  |
| Exch rcvd |  |  |

exchange フィールドは contest が active の場合だけ表示されます。

下段には以下があります。

- **Comment**。
- **Keep** — QSO 間で Comment フィールドを保持します。
- **Retain** — 保存後もフォーム全体を保持します。

Band、Mode、Submode などのフィールドは **PgUp/PgDn** で切り替えられます。

### 経路、方位、バッジ

両方の grid locator が分かる場合、CQOps は距離と azimuth を表示します。

QSO form は次のようなバッジも表示できます。

- **DUPE!**
- **New Call!**
- **New DXCC!**

### 保存

| キー | 操作 |
|---|---|
| Enter | QSO を保存 |
| Ctrl+S | 任意のフィールドから QSO を保存 |
| Esc | 重複確認をキャンセル |
| DUPE 確認で Enter | 重複をそれでも保存 |

---

## Logbook Editor と ADIF

Logbook Editor は **F8** で開きます。

用途:

- QSO 確認。
- inline editing。
- QSO 削除。
- ADIF import。
- ADIF export。
- Wavelog upload。
- Wavelog download。
- contest 関連操作。

### QSO の編集

1. **↑/↓** で行を選択します。
2. **Enter** または **e** を押します。
3. QSO を編集します。
4. **Ctrl+S** で保存します。

変更はすぐに Recent QSOs に反映されます。

### ADIF import と export

CQOps は ADIF 3.1.7 の import と export をサポートします。

| 操作 | ショートカット |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

import はレコードを検証し、重複をスキップし、サマリーを表示します。Wavelog sync が設定されている場合、import された QSOs は Wavelog upload 用にマークされます。

export はすべての QSOs または contest-filtered QSOs を含められます。`CONTEST_ID` は保持されます。

### デジタルモードの扱い

Mode と submode の扱いは、このマニュアルで説明する ADIF 3.1.7 に従います。

- FT8 は standalone mode として export されます。
- FT4 と FT2 は適切な submode を持つ MFSK として export されます。
- import された legacy MFSK + FT8 レコードは standalone FT8 に正規化されます。

QSO form には **Mode** と **Submode** の別フィールドがあります。どちらも **PgUp/PgDn** で切り替えられます。

---

## コンテスト

コンテストは QSO form に exchange フィールドと serial 処理を追加します。

Logbook Editor で **Ins** を押して contest を作成または設定します。

contest 設定には以下が含まれます。

- contest name。
- date。
- ADIF contest ID。
- exchange templates。

### template markers

| Marker | 置き換え内容 |
|---|---|
| `@rst` | 送信または受信 RST |
| `@serial` | 自動増加する serial number |
| `@call` | 自局コールサイン |
| `@grid` | 自局 grid locator |
| `@name` | operator profile の operator name |

**Ctrl+C** で active contest を切り替えます。

contest が active の場合:

- QSO form は exchange フィールドを表示します。
- serial numbers は自動増加します。
- Recent QSOs は contest QSOs をフィルタできます。
- ADIF export は `CONTEST_ID` を保持します。

---

## Favorites、リファレンス、Band Plan

### Favorites

Favorites は frequency、mode、band の presets を 10 slots に保存します。

| ショートカット | 操作 |
|---|---|
| Alt+0–9 | favorite を呼び出す |
| Alt+Shift+0–9 | 現在の frequency、mode、band を favorite に保存 |

Favorites は設定に保存され、logbooks 間で共有されます。

例:

1. `145.55` を入力します。
2. mode を `FM` にします。
3. band を `2m` にします。
4. **Alt+Shift+1** を押します。
5. 後で **Alt+1** を押して preset を呼び出します。

### REF Lookup

**F6** で REF Lookup を開きます。

検索対象:

- SOTA,
- POTA,
- WWFF,
- IOTA.

prefix、name、reference designator で検索できます。選択した references は QSO form に入力できます。

### Band Plan Browser

**F7** で Band Plan Browser を開きます。

すばやくアクセスできるもの:

- amateur bands。
- VHF/UHF ranges。
- CB。
- PMR446。
- broadcast presets。

選択した frequency は active rig のチューニングに使えます。band plan data は Markdown として export することもできます。

---

## 連携機能

すべての連携機能は任意です。ローカルログはそれらなしで動作します。

### QRZ.com

QRZ.com lookup にはインターネット接続と QRZ XML subscription が必要です。

QSO form で **Ins** を押すと、次のような callbook fields を入力できます。

- name,
- QTH,
- grid,
- country,
- CQ/ITU zones,
- DXCC,
- continent.

**F2** の Partner view は、利用可能な場合 operator photo を表示できます。

### Wavelog

Wavelog integration は以下をサポートします。

- upload。
- incremental download。
- worked/confirmed lookup。

Wavelog は active logbook ごとに設定します。

- URL。
- API key。
- station profile ID。

CQOps は常に QSOs を先にローカルへ保存します。Wavelog upload failure によってローカルデータが削除されることはありません。

### flrig

flrig integration は HTTP 上の XML-RPC を使用します。

デフォルト endpoint:

```text
localhost:12345
```

CQOps が読み取れるもの:

- frequency,
- mode,
- power.

split operation では VFO A を Frequency に、VFO B を Freq RX に対応させます。

### Hamlib / rigctld

Hamlib rig control は TCP daemon `rigctld` を使います。

radio と backend のサポートにより、CQOps は以下を取得できます。

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps は、可能な範囲で VFO-name support の不足を安全に扱います。

### Hamlib Rotor / rotctld

Rotor control は Hamlib `rotctld` を使います。

CQOps は以下をサポートします。

- azimuth,
- elevation,
- stop commands.

| ショートカット | 操作 |
|---|---|
| Ctrl+←/→ | azimuth を 5° 調整 |
| Ctrl+↑/↓ | elevation を 5° 調整 |
| Ctrl+A | rotor を計算された path bearing へ向ける |
| Ctrl+F1 | rotor を停止 |

### WSJT-X

WSJT-X integration は WSJT-X からの UDP messages を使います。CQOps は ADIF messages を解析し、完了した QSOs を自動ログできます。

WSJT-X が送信中の間、rig label はアクセント色になります。WSJT-X が報告する operator が active operator と一致しない場合、CQOps は警告を表示します。

### GPS

CQOpsはGPSレシーバーから位置を読み取り、ステーショングリッドロケーター
として使用できます — ポータブル、モバイル、フィールド運用に最適です。

2つのバックエンドがサポートされています：

- **Serial** — シリアルポート経由でGPSレシーバーに直接接続します
  （USB-シリアル、内蔵COMポート、または `/dev/ttyUSB0`）。
- **GPSD** — TCP経由で [gpsd](https://gpsd.io/) サーバーに接続します
  （デフォルト `127.0.0.1:2947`）。GPSを他のアプリケーションと共有する
  場合やネットワーク経由でアクセスする場合に便利です。

ステータスバーのGPSインジケーター：

| 色 | 意味 |
|--------|---------|
| 赤 `GPS` | 切断 / エラー |
| 黄 `GPS` | 接続済み、フィックスなし |
| 白 `GPS` | フィックス取得、位置確定 |

フィックスが取得されると、ステーショングリッドロケーターがGPS位置に
置き換えられ、ステータス行に `(GPS)` と表示されます：

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Station & Logbook設定で **Grid from GPS** を有効にすると、QSOログ記録、
APRSビーコン、ダッシュボードマップ、距離計算にGPSグリッドが使用されます。

**グリッド精度** — 統合メニューで設定可能（10、8、または6文字）。
デフォルトは10文字（約25 mの精度）。

### DX Cluster

DX Cluster integration は telnet を使用し、インターネット接続が必要です。

デフォルト server:

```text
dxspots.com:7300
```

フィルター:

- band,
- continent,
- mode,
- age/time.

| キー | 操作 |
|---|---|
| Enter | QSO form を入力し、rig をチューニングして QSO に戻る |
| Space | rig をチューニングし、DX Cluster に留まる |
| Backspace | filters をクリア |

### PSK Reporter

PSK Reporter integration にはインターネット接続が必要です。

提供するもの:

- propagation spots。
- band/time/mode filters。
- **F5** の ASCII world map。

### APRS

CQOps は 3 種類の APRS サービスに対応しています — お使いの局構成に
合ったものを選んでください：

| サービス | 接続 | インターネット |
|---|---|---|
| **APRS-IS** | APRS-IS サーバーへの TCP 接続 | 必要 |
| **KISS** | ハードウェア KISS TNC へのシリアル接続 | 不要 |
| **KISS Server** | KISS TNC サーバー（Dire Wolf など）への TCP 接続 | 不要（ローカルネットワーク） |

サービス種別は Integrations メニューで選択します：

```text
F9 → Integrations → APRS → Service（Space で切替）
```

3つのサービスすべてで、近隣局からの APRS 位置情報の受信と
CQOps Live ローカルマップへの表示が可能です：

- 標準 APRS シンボル、
- コールサイン ポップアップ、
- 自動フィット表示、
- 設定可能な範囲円。

また、すべてのサービスで**定期的な位置ビーコン送信**に対応しています。
CQOps は設定された間隔で自局のグリッドロケーターを送信します。
GPS が有効で **Grid from GPS** がオンになっている場合、ビーコンは
自動的に GPS から取得した位置を使用します — ポータブルや移動運用に
最適です。

#### APRS-IS

インターネット経由でグローバルな APRS-IS ネットワークに接続します。
以下のものが必要です：

- 有効なアマチュア無線コールサイン、
- APRS-IS パスコード（コールサインから生成）、
- インターネット接続。

デフォルトサーバー：

```text
euro.aprs2.net:14580
```

APRS-IS は **F9 → Integrations → APRS** でグローバルに設定します。
コールサイン、SSID、シンボル、コメント、ビーコン間隔、範囲フィルター
はログブックごとに **F9 → Logbooks → [アクティブログブック] → APRS**
で設定します。

#### KISS（シリアル）

シリアルポート経由でハードウェア KISS TNC に直接接続します。
インターネット接続は不要です — APRS フレームは無線機を通じて
送受信されます。

シリアルポート、ボーレート、データビット、パリティ、ストップビット、
DTR/RTS を Integrations メニューで設定します：

```text
F9 → Integrations → APRS → Service: KISS
```

KISS 選択時には、シリアル固有のフィールド（Port, Baud, Data bits,
Parity, Stop bits, DTR, RTS）が表示されます。

**Test** ボタンでシリアルポートを開き、TNC に到達可能かを確認します。

#### KISS Server（TCP）

TCP 経由でアクセス可能な KISS TNC に接続します — 例えば、同じマシン
またはローカルネットワーク上の
[Dire Wolf](https://github.com/wb2osz/direwolf) インスタンス。
インターネット接続は不要です。

Integrations メニューでサーバーアドレス（host:port）を入力します：

```text
F9 → Integrations → APRS → Service: KISS Server → Server
```

デフォルト：`localhost:8001`

#### ビーコン送信

ビーコンはログブックごとに設定された間隔で送信されます。
最小間隔は 1 分です。ビーコンには以下が含まれます：

- SSID 付きの局コールサイン、
- グリッドロケーター（GPS 利用可能時は GPS から取得）、
- APRS シンボル、
- オプションのコメント。

**GPS** が有効で **Grid from GPS** が Station 設定でオンになっている
場合、ビーコンは自動的に GPS から取得したグリッドロケーターを使用
します — 移動中に手動でグリッドを更新する必要はありません。

ビーコン間隔とその他のログブックごとの設定：

```text
F9 → Logbooks → [アクティブログブック] → APRS
```

#### 受信

受信した APRS 位置情報はローカルにキャッシュされ、CQOps Live ダッシュ
ボードのマップに表示されます。各局は APRS シンボル付きで表示され、
クリックで詳細を確認できます。設定された範囲内の全可視局を表示する
ように表示は自動調整されます。

APRS 受信はビーコン送信とは独立しています — ビーコンを送信せずに
受信することも、その逆も可能です。Integrations メニューで APRS を
有効にしてサービス種別を設定するだけです。

### Solar Data

Solar data は hamqsl.com から取得され、以下を含みます。

- SFI。
- sunspot number。
- A/K indices。
- band-by-band conditions。

live updates にはインターネット接続が必要です。取得成功後、キャッシュされたデータは offline でも利用できます。

---

## CQOps Live Dashboard

CQOps Live はリアルタイムの局活動を表示する内蔵ブラウザ dashboard です。

用途:

- Field Day の公開表示。
- クラブ局の情報画面。
- contest monitoring。
- 別室からの局監視。
- イベントや展示ブース。

### dashboard を有効化する

1. **F9** を押します。
2. **Integrations** を開きます。
3. **HTTP Server** へ移動します。
4. **HTTP server** を有効化します。
5. 必要に応じて address と port を設定します。
6. **Ctrl+S** で保存します。
7. ブラウザで dashboard を開きます。

デフォルト設定:

| 設定 | デフォルト |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

保存後、server はすぐに開始します。

### 表示モード

CQOps Live には 2 つの表示モードがあります。

#### Overview mode

active callsign を扱っていないときに表示されます。

表示内容:

- live Leaflet map。
- 今日の QSO markers。
- great-circle paths。
- recent QSOs table。
- station information。
- statistics。
- 5-minute、15-minute、1-hour rate tracking。
- top operators。
- longest-distance QSOs。

#### Active / Now Working mode

callsign を処理中のときに表示されます。

表示内容:

- 大きな callsign。
- submode indicator。
- 利用可能な場合の QRZ photo。
- band と mode badges。
- DUPE / NEW CALL / NEW DXCC indicators。
- distance と bearing。
- station grid から partner grid への強調された破線 map path。

### Info box

local map の上にある info box は、5 秒ごとに次の modules を切り替えます。

- band conditions。
- solar activity。
- geomagnetic field。
- latest DX Cluster spot。
- PSK Reporter per-band report counts。

Band conditions は常に全幅で表示されます。

### Weather row

weather row は station grid locator の現在の Open-Meteo conditions を表示します。

- temperature。
- wind。
- humidity。
- icon。

天気データはブラウザ側で取得され、offline では自然に degrade します。

### Local map

右側の local map は以下を表示できます。

- APRS stations。
- standard APRS symbols。
- range circle。
- callsign popups。
- optional day/night terminator overlay。
- optional RainViewer weather radar overlay。

### リアルタイム更新と性能

CQOps Live は Server-Sent Events (SSE) で更新されます。ページ再読み込みは不要です。

dashboard は低消費電力ハードウェア向けに設計されています。

- map rendering はブラウザが処理。
- distance calculations はブラウザが処理。
- statistics はブラウザが処理。
- CQOps は軽量な JSON updates を送信。
- HTTP server が無効な場合、port は開かれず dashboard goroutines も動作しません。

### dashboard のカスタマイズ

HTTP Server integration form で設定できます。

| フィールド | 説明 |
|---|---|
| Header 1 | page header と hero area に表示される主タイトル。空の場合 “CQOps Live”。 |
| Header 2 | タイトル下のサブタイトル。空の場合 “Fast, portable ham radio logger”。 |
| Logo URL | 左上に表示される公開画像 URL。空の場合 CQOps logo。 |
| Event Start | `YYYY-MM-DD` 形式の日付。この日以降の stats と QSO lists に絞り込みます。 |

---

## 設定

**F9** で設定を開きます。

### 設定ファイル

| プラットフォーム | config path |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

機密 credential は同じ設定ディレクトリの `secrets.enc` に別保存されます。

secrets は machine-tied key で暗号化されます。別のマシンへ設定を移す場合、credential は再入力が必要です。

### 設定メニュー

| Menu | 設定内容 |
|---|---|
| Station | Callsign, grid, CQ/ITU zone, IARU region, references |
| Rig | Rig presets, model, antenna, power, backend, rotor, WSJT-X |
| Wavelog | URL, API key, station profile ID |
| QRZ | Username and password |
| DX Cluster | Host, port, login |
| Operators | Operator profiles |
| Logbooks | Station, Wavelog, contest, operator, APRS settings per logbook |
| Integrations | APRS サービス種別（APRS-IS, KISS, KISS Server）, GPS, HTTP サーバー, DXC, QRZ |
| Notifications | QSO saved alerts, Wavelog status, dupe beep, error sounds |
| General | Timezone, distance units, map, debug mode |

### Multi-logbook

自宅、portable、contest、club operation には複数 logbook を使えます。

**Ctrl+L** で active logbook を切り替えます。

各 logbook は独自の以下を保持します。

- station details。
- Wavelog settings。
- contest settings。
- operator settings。

### Multi-operator

Operator profiles には以下が含まれます。

- operator callsign。
- operator name。

**Ctrl+O** で active operator を切り替えます。

active operator は ADIF `OPERATOR` フィールドに保存され、Wavelog uploads にも反映されます。

### Multi-rig

Rig presets は以下を保存します。

- backend。
- model。
- antenna。
- power。
- rotor settings。
- WSJT-X settings。

**Ctrl+R** で active rig を切り替えます。

### 暗号化 secrets

v0.8.7 以降、credentials は暗号化保存されます。

| 項目 | 値 |
|---|---|
| Secrets file | `secrets.enc` |
| Location | `config.yaml` と同じディレクトリ |
| Unix permissions | 対応環境では `0600` |
| Encryption | machine-tied key による AES-256-GCM |
| Protected data | QRZ password, DX Cluster login, Wavelog API keys |

古い設定の plaintext secrets は初回起動時に移行されます。

`secrets.enc` が破損している場合、CQOps は警告付きで起動し、credentials の再入力を求めます。

---

## キーボードショートカット

### Global

| キー | 操作 |
|---|---|
| F1 | QSO form と Recent QSOs |
| F2 | Partner view |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | REF Lookup |
| F7 | Band Plan Browser |
| F8 | Logbook Editor |
| F9 | Configuration / main menu |
| F10 | Quit |
| Ctrl+F9 | Log viewer |
| ? | Help overlay |
| Ctrl+L | active logbook を切り替え |
| Ctrl+R | active rig を切り替え |
| Ctrl+C | active contest を切り替え |
| Ctrl+O | active operator を切り替え |
| Esc | 前の画面へ戻る |

### QSO form

| キー | 操作 |
|---|---|
| Tab | 次のフィールド |
| Shift+Tab | 前のフィールド |
| ↑ / ↓ | 列内を移動 |
| Enter | 必要に応じて重複確認付きで QSO 保存 |
| Ctrl+S | 任意のフィールドから QSO 保存 |
| Del | フォーム全フィールドをクリア |
| Ins | Lookup: QRZ, Wavelog, DXCC, duplicate check |
| PgUp / PgDn | band, mode, submode を切り替え |
| Ctrl+D | spot dialog を開く |
| Ctrl+T | Keep Comment を切り替え |
| Ctrl+←/→ | rotor azimuth を 5° 調整 |
| Ctrl+↑/↓ | rotor elevation を 5° 調整 |
| Ctrl+A | 自局 grid から相手 grid への bearing に rotor を向ける |
| Ctrl+F1 | rotor を停止 |
| Alt+0–9 | favorite を呼び出す |
| Alt+Shift+0–9 | 現在の frequency, mode, band を favorite として保存 |

### Logbook Editor

| キー | 操作 |
|---|---|
| ↑ / ↓ | 行を移動 |
| PgUp / PgDn | 前ページまたは次ページ |
| Home / End | 最初または最後の行 |
| Enter / e | 選択 QSO を編集 |
| Delete | 選択 QSO を削除 |
| p | 全 QSOs を purge |
| Ctrl+C | contest filter を切り替え |
| Ctrl+E | Export ADIF |
| Ctrl+I / Tab | Import ADIF |
| w | 未送信 QSOs を Wavelog へ upload |
| Ctrl+W | Wavelog から contacts を download |
| Esc / F6 | editor を閉じて QSO form へ戻る |

### DX Cluster

| キー | 操作 |
|---|---|
| ↑ / ↓ | spots を移動 |
| Enter | QSO form を入力し、rig をチューニングして QSO へ戻る |
| Space | 選択 spot に rig をチューニングし、DX Cluster に留まる |
| Home | band filter を前へ進める |
| End | band filter を戻す |
| `\` | continent filter を切り替え |
| Ins | mode filter を前へ進める |
| Del | mode filter を戻す |
| PgUp | time filter を前へ進める |
| PgDn | time filter を戻す |
| Backspace | すべての filters をクリア |
| Esc / F4 | QSO form へ戻る |

### Partner view

| キー | 操作 |
|---|---|
| F2 | Partner view → Photo → Back を切り替え |
| Esc / F1 | QSO form へ戻る |

---

## トラブルシューティング

### CQOps が起動しない

確認事項:

- terminal size が 80×24 以上である。
- Windows では Windows Terminal を使っている。
- network startup がブロックしていないか次で確認する。

  ```bash
  cqops --offline
  ```

ログの場所:

| プラットフォーム | Logs path |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### Rig が接続しない

flrig の場合:

- flrig が動作していることを確認します。
- active rig preset の port を確認します。
- default port は `12345` です。

Hamlib の場合:

- `rigctld` が動作していることを確認します。
- host と port を確認します。
- radio/backend が要求データをサポートしていることを確認します。

Status labels は診断に役立ちます。

| 色 | 意味 |
|---|---|
| 白/デフォルト | 接続済み |
| 黄 | 無効または接続中 |
| 赤 | 失敗 |

Reconnect toasts は抑制される場合があります。CQOps は静かに再試行できます。

### WSJT-X が自動ログしない

確認事項:

- WSJT-X **Settings → Reporting → UDP Server**。
- UDP host と port が CQOps の active rig preset と一致している。
- WSJT-X 2.6 以降を使っている。
- WSJT status label が active。
- active logbook が正しい。
- active operator が正しい。

### Wavelog upload が失敗する

確認事項:

- Wavelog URL。
- API key。
- station profile ID。
- **WL** status label。

upload errors は toasts として表示されます。upload に失敗しても QSOs はローカルに保存されたままです。個別 QSO の失敗は batch の残りをブロックしません。

### config file の問題

Config file:

| プラットフォーム | パス |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets file:

```text
secrets.enc
```

secrets file は `config.yaml` と同じディレクトリに保存されます。

config が破損している場合、移動または削除して CQOps を再起動します。setup wizard が新しい config を作成します。

`last_fetched_id` フィールドは、Wavelog download が成功した後にだけ現れます。

### 性能問題

試すこと:

- General settings で map rendering を無効化する。
- 不要なら Solar panel を無効化する。
- offline 時は DX Cluster や PSK Reporter などネットワーク負荷の高い画面を避ける。
- ネットワークが不安定な場合は `cqops --offline` を使う。

---

## バグ報告

バグを報告する前に:

1. **F9 → General → Debug** で **Debug mode** を有効にする、または次を設定します。

   ```yaml
   debug: true
   ```

   `config.yaml` 内に設定します。

2. 問題を再現します。
3. 関連ログを添付します。

GitHub に issue を報告してください。

<https://github.com/szporwolik/cqops/issues>

含める情報:

- `cqops --version` の CQOps バージョン。
- operating system。
- terminal emulator。
- 再現手順。
- 関連 debug log。
