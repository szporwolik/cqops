---
title: CQOps ユーザーマニュアル
description: 高速でターミナル中心のアマチュア無線ロガー CQOps のインストール、設定、使用方法
---

# CQOps ユーザーマニュアル

CQOps は、キーボードによる信頼性の高いログ入力と低いシステム負荷を求めるオペレーター向けの、高速でターミナル中心のアマチュア無線ロガーです。シャックでの運用、ポータブル運用、クラブ局、field day、Raspberry Pi クラスの機器や旧型ノート PC での使用を想定しています。

CQOps は、QSO を必ず最初にローカルへ保存します。インターネットを使用する連携機能は任意です。

## 目次

1. [CQOps とは](#what-cqops-is)
2. [ダウンロードとインストール](#download-and-installation)
3. [初回起動](#first-launch)
4. [最初の QSO を記録する](#log-your-first-qso)
5. [メイン画面](#main-screen)
6. [一般的な運用手順](#common-workflows)
7. [QSO の記録](#qso-logging)
8. [ログブックエディターと ADIF](#logbook-editor-and-adif)
9. [コンテスト](#contests)
    - [コンテストを設定する](#setting-up-a-contest)
    - [下部ステータスバー](#bottom-status-bar)
    - [コンテスト統計パネル](#contest-statistics-panel)
    - [コンテスト ADIF のエクスポート](#contest-adif-export)
    - [コンテストモードの動作](#contest-mode-behavior)
10. [お気に入り、リファレンス、バンドプラン](#favorites-references-and-band-plans)
11. [連携機能](#integrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [設定](#configuration)
14. [キーボードショートカット](#keyboard-shortcuts)
15. [トラブルシューティング](#troubleshooting)
16. [バグの報告](#reporting-bugs)

---

<a id="what-cqops-is"></a>
## CQOps とは

CQOps は、高速な QSO 入力、ローカル優先のログ保存、実用的なフィールド運用を中心に設計されています。

### 主な考え方

- **ターミナル中心の操作** — キーボード操作に最適化されています。
- **Offline-first logging** — インターネット接続がなくてもローカルに QSO を記録できます。完全オフラインで動作するダッシュボード用の埋め込み世界地図を含みます。
- **低負荷** — Raspberry Pi クラスのシステム、旧型ノート PC、共有の局用 PC に適しています。
- **ポータブルな設計** — 単一の Go バイナリとして配布されます。
- **複数のログブック** — 個人、ポータブル、コンテスト、クラブ局のログに利用できます。
- **複数のオペレーター** — hot-seat 運用や共有クラブ局での運用に適しています。
- **複数の無線機** — 各 rig preset に固有の backend と WSJT-X 設定を保存できます。
- **任意の連携機能** — マルチプロバイダー callbook（QRZ.com、HamQTH、QRZ.RU、Callook.info）、Wavelog、DX Cluster、PSK Reporter、GPS、APRS、無線機制御、ローテーター制御、太陽活動データ、ブラウザー用 CQOps Live dashboard。

ローカルでのログ記録にインターネット接続は不要です。`--offline` モードではネットワーク機能が実行されません。

### CQOps の対象ユーザー

CQOps は、次のような利用者に適しています。

- ポータブル運用者、
- SOTA および POTA アクティベーター、
- クラブ局、
- field day チーム、
- ターミナルでの操作を好むオペレーター、
- オペレーター、ログブック、無線機を素早く切り替える必要がある局。

CQOps は、完全なデスクトップロガーや Web ベースのログブックプラットフォームのすべての機能を置き換えることを目的としていません。高速なターミナルログ、フィールド運用、オフライン利用、共有局での運用に重点を置いています。

### クラブ局および共有局での利用

CQOps は、アマチュア無線クラブでの利用を考慮して設計されています。アクティブなオペレーターは常にステータスバーに表示されるため、**ひと目見るだけで**現在誰が運用しているか確認できます。オペレーターの切り替えは `Ctrl+O` だけで行え、直ちに反映されます。その後のすべての QSO にオペレーターの indicativo と name が記録されます。ログアウト、パスワード入力、作業中断は不要です。

ログブック、rig preset、コンテストも同様に `Ctrl+L`、`Ctrl+R`、`Ctrl+C` で切り替えられます。交代制のオペレーター、複数の無線機、複数のアクティブなコンテストを運用するクラブ局でも、マウスを使わず 1 秒未満でコンテキストを切り替えられます。

field day や公開イベントでは、**CQOps Live dashboard** を使用して、リアルタイムマップ、QSO フィード、統計を大型画面に表示できます。来場者やクラブメンバーは、オペレーターのターミナル周辺に集まることなく運用状況を確認できます。**HTTP Server** 連携を有効にし、Web ブラウザーを備えた任意の端末からアクセスしてください。

---

<a id="download-and-installation"></a>
## ダウンロードとインストール

すべてのリリースを表示:

<https://github.com/szporwolik/cqops/releases>

### Windows

| パッケージ | リンク | 備考 |
|---|---|---|
| インストーラー | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | ほとんどのユーザーに推奨。CQOps を Start Menu と PATH に追加します。 |
| ポータブル ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | 展開して、インストールせずに実行します。 |

### Linux — Debian / Ubuntu / Pop!_OS / Linux Mint

Cloudsmith APTリポジトリを追加してインストール:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

または `.deb` を直接ダウンロード:

| アーキテクチャ | リンク | 対象 |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | ほとんどの Intel/AMD PC |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64 ビット ARM システム |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32 ビット Raspberry Pi OS |

ダウンロードしたパッケージをインストールします。

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Fedora / RHEL / Rocky / AlmaLinux

Cloudsmith RPMリポジトリを追加してインストール:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

### Linux — Arch / Manjaro / CachyOS

AURからインストール:

```bash
# CachyOS（デフォルトでparuを使用）
paru -S cqops-bin

# Arch / Manjaro
yay -S cqops-bin
```

`pacaur`、`aura`、または手動 `makepkg` でも利用可能。PKGBUILD: [aur.archlinux.org/packages/cqops-bin](https://aur.archlinux.org/packages/cqops-bin)。

### Linux — ポータブル Tarball

| アーキテクチャ | リンク | 対象 |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | ほとんどの Intel/AMD PC |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64 ビット ARM システム |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32 ビット Raspberry Pi OS |

### macOS

| アーキテクチャ | リンク | 対象 |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | M1/M2/M3 Mac |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Intel Mac |

手動でインストールします。

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### ソースからビルドする

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

ソースからのビルドには Go 1.26 以降が必要です。

### ターミナル要件

| 要件 | 値 |
|---|---|
| 最小ターミナルサイズ | 80×24 文字 |
| 推奨ターミナルサイズ | 80×43 文字以上 |
| 推奨 Windows ターミナル | Windows Terminal |
| Kitty graphics 対応ターミナル | [Kitty](https://sw.kovidgoyal.net/kitty/)、[Ghostty](https://ghostty.org/)、[WezTerm](https://wezfurlong.org/wezterm/) |

### 基本コマンド

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

<a id="first-launch"></a>
## 初回起動

初回起動時に CQOps はセットアップウィザードを開きます。ローカルでのログ記録には、局の基本情報のみが必要です。ネットワーク連携はスキップして後から設定できます。

### ウィザードのページ

| Page | 設定内容 |
|---|---|
| Station & Logbook | 初期ログブック、局の indicativo、operator、grid locator、任意の reference と zone、Wavelog URL/API/station profile ID |
| Rig | rig preset、model、antenna、power、backend、任意の rotor、任意の WSJT-X UDP 設定 |
| Integrations | callbook lookup の設定（QRZ.com、HamQTH、QRZ.RU、Callook.info） |
| General | IANA timezone |
| Summary | 内容の確認と保存 |

対応する rig backend:

- None、
- flrig、
- Hamlib `rigctld`。

### ウィザードの操作

| Key | Action |
|---|---|
| Ctrl+S | 入力を検証して次へ進む。**Summary** では保存して CQOps を起動する |
| Esc | 戻る |
| F10 | 終了 |
| Tab / Shift+Tab | フィールド間を移動する |
| Space | チェックボックスを切り替える |

ウィザードで設定した内容は、後から **F9** で変更できます。

---

<a id="log-your-first-qso"></a>
## 最初の QSO を記録する

1. CQOps を起動します。

   ```bash
   cqops
   ```

2. セットアップウィザードで、少なくとも自局の indicativo と grid locator を入力します。

3. **F1** を押して **QSO form** を開きます。

4. 相手局の indicativo を入力します。CQOps は indicativo を自動的に大文字へ変換します。

5. 残りのフィールドを入力します。アクティブな無線機が flrig または Hamlib で接続されている場合、CQOps は frequency、band、mode、submode を自動入力できます。

6. **Enter** を押して保存します。

7. **DUPE!** の警告が表示された場合は、もう一度 **Enter** を押すとそのまま保存でき、**Esc** を押すとキャンセルできます。

保存した QSO は、すぐに **Recent QSOs** テーブルへ表示されます。

---

<a id="main-screen"></a>
## メイン画面

CQOps は固定されたターミナルレイアウトを使用します。

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

**Status bar** には次の情報が表示されます。

- CQOps のバージョン、
- アクティブなログブック、
- アクティブな rig、
- 局の indicativo、
- アクティブな operator、
- 連携機能の状態ラベル、
- `L` が付いたローカル時刻、
- `Z` が付いた UTC 時刻。

一般的なラベルには **Net**、**WSJT**、**Rig**、**Flrig**、**Hamlib**、**Rotator**、**DXC**、**WL**、**GPS** があります。**GPS** ラベルも同じ色の規則を使用します。赤は未接続、黄は接続済みだが fix なし、白は位置 fix 取得済みを示します。

| 色 | 意味 |
|---|---|
| 白/デフォルト | 接続済みまたはアクティブ |
| 黄 | 無効、接続中、または想定どおりオフライン |
| 赤 | エラーまたは切断 |
| アクセント色 + 太字 | WSJT-X が送信中 |

### メインタブ

| Key | Tab | Screen |
|---|---|---|
| F1 | QSO | **QSO form** と **Recent QSOs** |
| F2 | QRZ | **Partner view**: callbook データ、マップ、統計、写真 |
| F4 | DXC | **DX Cluster** の spot と filter |
| F5 | HRD | **PSK Reporter** の spot と伝搬マップ |
| F6 | REF | SOTA/POTA/WWFF/IOTA reference 検索 |
| F7 | BPL | **Band Plan Browser** |
| F8 | LOG | **Logbook Editor**、ADIF、Wavelog 同期 |
| F9 | CFG | 設定メニュー |

**Help bar** には、現在の画面で使用できるショートカットが表示されます。**?** を押すと完全な **Help overlay** が開きます。

---

<a id="common-workflows"></a>
## 一般的な運用手順

### ポータブル、SOTA、POTA 運用

出発前:

1. インターネット接続がある状態で CQOps を一度実行します。
2. 太陽活動データ、REF データ、DXCC prefix などのキャッシュデータを CQOps にダウンロードまたは更新させます。
3. **Solar** パネルにデータが表示されていることを確認します。
4. **F6** の **REF** 検索で結果が表示されることを確認します。

フィールドでは:

1. CQOps をオフラインモードで起動します。

   ```bash
   cqops --offline
   ```

2. 通常どおりログを入力します。QSO はローカルに保存されます。
3. オンラインに戻ったら **F8** を開き、**w** を押して未送信の QSO を Wavelog にアップロードします。

### 共有クラブ局と hot-seat ロギング

1. **F9 → Operators** を開きます。
2. **Ins** を押して operator profile を追加します。
3. **QSO form** で **Ctrl+O** を押し、アクティブな operator を切り替えます。
4. 保存前にステータスバーでアクティブな operator を確認します。
5. 複数のオペレーターが似た内容の交信を入力し、フォーム全体を再入力したくない場合は **Retain** を使用します。

アクティブな operator は ADIF の `OPERATOR` フィールドに保存されます。

### 個人用とクラブ用のログブック

1. **F9 → Logbooks** を開きます。
2. **Ins** を押して各ログブックを作成します。
3. **QSO form** で **Ctrl+L** を押し、アクティブなログブックを切り替えます。
4. 保存前にステータスバーでアクティブなログブックを確認します。

各ログブックは、固有の局情報、Wavelog 設定、コンテスト設定、operator を保持できます。

### 複数の無線機

1. **F9 → Rigs** を開きます。
2. **Ins** を押して rig preset を作成します。
3. backend として None、flrig、Hamlib のいずれかを選択します。
4. **QSO form** で **Ctrl+R** を押し、アクティブな rig を切り替えます。

rig preset には backend、model、antenna、power、rotor 設定、WSJT-X UDP 設定を含められます。

### WSJT-X によるデジタル運用

WSJT-X UDP 連携が有効な場合、CQOps は WSJT-X から ADIF メッセージを受信し、完了したデジタル QSO を自動的に記録できます。

自動記録された QSO は:

- アクティブなログブックに保存される、
- 直ちに **Recent QSOs** に表示される、
- duplicate をスキップする、
- アクティブな contest ID を継承する、
- Wavelog が設定済みで接続可能な場合、自動的にアップロードできる。

WSJT-X が通知した operator と CQOps のアクティブな operator が一致しない場合、CQOps は警告を表示します。

長時間のデジタル運用前に、次を確認してください。

- アクティブなログブック、
- アクティブな operator、
- アクティブな contest、
- **WSJT** status label。

### Wavelog の同期

CQOps は、QSO を必ず最初にローカルへ保存します。Wavelog との同期は任意です。

| Action | Where | Shortcut | Notes |
|---|---|---|---|
| Upload unsent QSOs | **Logbook Editor** | `w` | 50 件単位の batch でアップロード |
| Download from Wavelog | **Logbook Editor** | `Ctrl+W` | `last_fetched_id` を使用した増分ダウンロード |

アップロード状態は QSO ごとに管理されます。

- not sent、
- sent、
- error。

アップロードが失敗しても、QSO はローカルログブックに残り、後から再試行できます。ログブックを purge すると fetch ID が `0` に戻り、完全な再ダウンロードが可能になります。

---

<a id="qso-logging"></a>
## QSO の記録

**QSO form** は、メインのログ入力画面です。**F1** で開きます。

CQOps は、次の情報源からフィールドを入力できます。

| 情報源 | フィールド |
|---|---|
| flrig / Hamlib | Frequency、split 時の Freq RX、Mode、Submode |
| Callbook（QRZ.com / HamQTH / QRZ.RU / Callook.info） | Name、QTH、Grid、Country、CQ zone、ITU zone、DXCC、Continent、写真 |
| REF database | SOTA、POTA、WWFF、IOTA reference |
| Wavelog lookup | 設定時の worked/confirmed status |
| DXCC/prefix data | prefix と country 関連データ |

### フォームのレイアウト

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

**Exch sent** と **Exch rcvd** は、コンテストがアクティブな場合のみ表示されます。

最下段には次の項目があります。

- **Comment**、
- **Keep** — QSO 間で **Comment** フィールドを保持する、
- **Retain** — 保存後もフォーム全体を保持する。

**Band**、**Mode**、**Submode** などのフィールドは **PgUp/PgDn** で切り替えられます。

### 経路、方位、バッジ

両方の grid locator が分かっている場合、CQOps は距離と方位を表示します。

**QSO form** には次のようなバッジも表示されます。

- **DUPE!**
- **New Call!**
- **New DXCC!**

### 保存

| Key | Action |
|---|---|
| Enter | QSO を保存する |
| Ctrl+S | 入力済みフォームから DX spot を送信する |
| Esc | duplicate の確認をキャンセルする |
| DUPE 確認画面で Enter | duplicate をそのまま保存する |

---

<a id="logbook-editor-and-adif"></a>
## ログブックエディターと ADIF

**F8** を押して **Logbook Editor** を開きます。

次の操作に使用できます。

- QSO の確認、
- インライン編集、
- QSO の削除、
- ADIF import、
- ADIF export、
- Wavelog upload、
- Wavelog download、
- コンテスト関連の操作。

### QSO を編集する

1. **↑/↓** で行を選択します。
2. **Enter** または **e** を押します。
3. QSO を編集します。
4. **Ctrl+S** で保存します。

変更内容は直ちに **Recent QSOs** に反映されます。

### ADIF の import と export

CQOps は ADIF 3.1.7 の import と export に対応しています。

| Action | Shortcut |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

import 時には record を検証し、duplicate をスキップして summary を表示します。Wavelog 同期が設定されている場合、import された QSO は Wavelog upload 対象としてマークされます。

export には、すべての QSO または contest filter を適用した QSO を含められます。`CONTEST_ID` は保持されます。

### デジタルモードの処理

mode と submode の処理は、このマニュアルで説明する ADIF 3.1.7 に従います。

- FT8 は独立した mode として export されます。
- FT4 と FT2 は、適切な submode を持つ MFSK として export されます。
- import された旧形式の MFSK + FT8 record は、独立した FT8 に正規化されます。

**QSO form** には、独立した **Mode** と **Submode** フィールドがあります。どちらも **PgUp/PgDn** で切り替えられます。

---

<a id="contests"></a>
## コンテスト

CQOps には、**気軽なコンテスト参加**向けの軽量なコンテストログパネルがあります。N1MM、Win-Test、TR4W などの専用コンテストロガーの代わりになるものではありません。本格的な multi-op、multi-radio、assisted category で参加する場合は、専用のコンテストロガーを使用してください。CQOps は、数局にポイントを提供したい場合、rate を楽しみながら確認したい場合、または SOTA/POTA アクティベーション中に普段のロガーを離れず数件のコンテスト QSO を記録したい場合に適しています。

<a id="setting-up-a-contest"></a>
### コンテストを設定する

**Logbook Editor** で **Ins** を押し、コンテストを作成または設定します。

コンテスト設定には次の項目があります。

- contest name、
- date、
- ADIF contest ID、
- exchange template。

#### テンプレートマーカー

| Marker | Replaced with |
|---|---|
| `@rst` | 送信または受信した RST |
| `@serial` | 自動加算される serial number |
| `@cqz` | DX局の CQ zone |
| `@mycqz` | 自局の CQ zone |
| `@itu` | DX局の ITU zone |
| `@myitu` | 自局の ITU zone |
| `@grid` | DX局の grid square |
| `@mygrid` | 自局の grid square |

**Ctrl+C** でアクティブなコンテストを切り替えるか、**Contest** メニュー（**F7**）から選択します。exchange field は **QSO form** に自動表示され、serial number は自動的に増加します。

<a id="bottom-status-bar"></a>
### 下部ステータスバー

コンテストがアクティブな場合、下部バーにリアルタイムの概要が表示されます。

```
 IARU-HF · IARU HF   45 QSOs   Started 16:13   Last 14:04 ago   Next #45   On 2:41
```

| フィールド | 意味 |
|-------|---------|
| `IARU-HF` | コンテストの ADIF ID。機械可読の識別子 |
| `· IARU HF` | コンテストの表示名。ID と異なる場合に表示 |
| `45 QSOs` | このコンテストセッションで記録した QSO の総数 |
| `Started 16:13` | 当日の最初のコンテスト QSO の時刻 |
| `Last 14:04 ago` | 最後のコンテスト QSO からの経過時間 |
| `Next #45` | 次の QSO で送信する serial number |
| `On 2:41` | 総 on-air time。30 分未満の QSO 間隔を合計した時間 |

`Started` フィールドは、120 列未満のターミナルでは非表示になります。コンテスト名と `On` 時間は、100 列未満では非表示になります。

<a id="contest-statistics-panel"></a>
### コンテスト統計パネル

コンテストがアクティブでターミナル幅が十分な場合、**QSO form** の右側に黄色い枠のコンパクトな統計パネルが表示されます。

```
╭──────────────────────────────────╮
│  Rate     2/h   --/h             │
│  Count 60m   0  hr   0           │
│  Peak  1m120 10m 54 60m 29       │
│  Avg      8/h  Sess 5:36         │
│  QSO/min  last 60m  max 1        │
│                                  │
│                                  │
│                                  │
│                                  │
│  -60m                       now  │
╰──────────────────────────────────╯
```

| 行 | フィールド | 意味 |
|-----|-------|---------|
| **Rate** | `2/h` | 直近 **10 QSO** の rate。短時間の burst speed |
| | `--/h` | 直近 **100 QSO** の rate。100 QSO に達するまでは `--` |
| **Count** | `60m 0` | 過去 60 分間に記録した QSO 数 |
| | `hr 0` | 現在の clock hour の `:00` 以降に記録した QSO 数 |
| **Peak** | `1m120` | 1 分間の最高 rate。120/h はその 1 分に 2 QSO を意味する |
| | `10m 54` | 10 分間の sliding window の最高値。平均 54/h |
| | `60m 29` | 60 分間の sliding window の最高値。平均 29/h |
| **Avg** | `8/h` | セッション平均。QSO 総数を最初の QSO からの経過時間で割った値 |
| | `Sess 5:36` | 最初の QSO から最後の QSO までのセッション時間。H:MM または分のみ |
| **Chart** | `max 1` | 最も多い 1 分間に 1 QSO。バーは 1 分あたりの QSO 数を示す |
| | `-60m…now` | 左端 = 60 分前、右端 = 現在 |

グラフは Unicode ブロック文字（`█`）を使用し、縦棒 4 行にスケーリングされます。**Peak** はすでに「1 時間あたり」を意味するため、peak rate では `/h` を省略します。1 分ごとの更新では不要なため、時間表示から秒を省略します。

<a id="contest-adif-export"></a>
### コンテスト ADIF のエクスポート

コンテストログを提出するには、コンテストがアクティブな状態で **Logbook Editor** を開き、`Ctrl+E` を押します。contest filter が適用されている場合、ADIF export dialog から **アクティブなコンテストに属する QSO のみ**を export できます。contest exchange field、serial number、contest ADIF ID を保持した ADIF 3.1.7 準拠ファイルが生成され、主催者の robot または log-checking system にアップロードできます。

<a id="contest-mode-behavior"></a>
### コンテストモードの動作

コンテストがアクティブな場合:

- **QSO form** に exchange field が表示される、
- serial number が自動加算される、
- **Recent QSOs** を contest QSO のみに filter できる、
- ADIF export で `CONTEST_ID` が保持される、
- **QSO form**、contest panel、**Solar** panel が黄色い枠で表示される、
- DXC spot は当日分だけでなく、すべての contest QSO と照合されて dupe が判定される。

---

<a id="favorites-references-and-band-plans"></a>
## お気に入り、リファレンス、バンドプラン

### Favorites

**Favorites** は frequency、mode、band の preset を 3 つの slot に保存できます。よく使う呼出周波数を保存するのに十分です。ショートカットは `Alt` を使用するため、標準的なターミナル編集キーと競合せず、さまざまなターミナルで安定して動作します。

| Shortcut | Action |
|---|---|
| Alt+Ins / Alt+Home / Alt+PgUp | slot 1、2、3 の favorite を呼び出す |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | 現在の frequency、mode、band を slot 1、2、3 に保存する |

**Favorites** は設定に保存され、すべてのログブックで共有されます。

例:

1. `145.55` を入力します。
2. **Mode** を `FM` に設定します。
3. **Band** を `2m` に設定します。
4. **Alt+Shift+Ins** を押して slot 1 に preset を保存します。
5. 後から **Alt+Ins** を押して preset を呼び出します。

### REF Lookup

**F6** で **REF Lookup** を開きます。

次のサービスを検索します。

- SOTA、
- POTA、
- WWFF、
- IOTA。

prefix、name、reference designator で検索できます。選択した reference は **QSO form** に入力できます。

### Band Plan Browser

**F7** で **Band Plan Browser** を開きます。

次の項目へすぐにアクセスできます。

- Amateur bands、
- VHF/UHF ranges、
- CB、
- PMR446、
- Broadcast presets、
- Portable — SOTA、POTA、calling channel など、ポータブル/フィールド運用で一般的な周波数。

選択した周波数でアクティブな rig を tune できます。band plan data は Markdown 形式でも export できます。

---

<a id="integrations"></a>
## 連携機能

すべての連携機能は任意です。連携を使用しなくてもローカルログは動作します。

### Callbook（QRZ.com、HamQTH、QRZ.RU、Callook.info）

CQOps は優先度ベースのカスケード方式で複数の callbook プロバイダーに対応しています。
QSO フォームで **Ins** を押すと、結果が返るまでプロバイダーが順に照会されます。

1. **QRZ.com** — インターネット接続と QRZ XML サブスクリプションが必要です。最も包括的なデータ。
2. **HamQTH** — 無料のグローバルサービス。カバレッジが良好で、無料アカウントが必要です。
3. **QRZ.RU** — ロシアと周辺国向けの無料サービス。API ログインが必要。名前、QTH、グリッド、緯度/経度、クラス、LoTW/eQSL、写真を返します。
4. **Callook.info** — 無料の米国向けサービス。アカウント不要、FCC の高速検索。

優先度の高いプロバイダーが失敗または無効の場合、次のプロバイダーが試行されます。
**Base call fallback** が有効（デフォルト：オン）の場合、フルコールが結果を返さない
ときに、CQOps はベースコールサイン（プレフィックスやサフィックスなし）も試行します。

プロバイダーの有効化と設定は **F9 → Callbook** で行います。

**QSO form** で **Ins** を押すと、次のような callbook field を入力できます。

- Name、
- QTH、
- Grid、
- Country、
- CQ/ITU zones、
- DXCC、
- Continent。

**F2** の **Partner view** では、利用可能な場合にオペレーターの写真を表示できます。

> ⚠️ **Experimental.** 写真表示には Kitty terminal graphics protocol を
> 使用でき、対応ターミナルである Kitty、Ghostty、WezTerm が必要です。
> **F9 → General → Kitty Graphics** で有効にしてください。標準ターミナルや
> graphics passthrough のない SSH セッションでは、glyph photo が代わりに
> 使用されます。

### Wavelog

Wavelog 連携は次の機能に対応しています。

- upload、
- incremental download、
- worked/confirmed lookup。

Wavelog はアクティブなログブックごとに、次の項目で設定します。

- URL、
- API key、
- station profile ID。

CQOps は QSO を必ず最初にローカルへ保存します。Wavelog upload に失敗しても、ローカルデータは削除されません。

### flrig

flrig 連携は HTTP 上の XML-RPC を使用します。

デフォルト endpoint:

```text
localhost:12345
```

CQOps は次の情報を読み取れます。

- frequency、
- mode、
- power。

split 運用では、VFO A が **Frequency**、VFO B が **Freq RX** に対応します。

### Hamlib / rigctld

Hamlib による rig control は、TCP daemon `rigctld` を使用します。

無線機と backend の対応状況に応じて、CQOps は次の情報を取得できます。

- frequency、
- mode、
- VFO、
- split、
- power。

可能な場合、CQOps は VFO name 非対応でも安全に処理します。

### Hamlib Rotator / rotctld

> ⚠️ **Experimental.** rotor control は実験的機能です。運用前に必ず
> アンテナの物理的な可動範囲を確認してください。**Alt+/** ですぐに
> 動作を停止できるよう準備してください。誤った設定は rotor または
> antenna を損傷する可能性があるため、慎重に使用してください。

rotor control は Hamlib `rotctld` を使用します。

CQOps は次の操作に対応しています。

- azimuth、
- elevation、
- stop command。

| Shortcut | Action |
|---|---|
| Alt+, | azimuth を −5° 調整する |
| Alt+. | azimuth を +5° 調整する |
| Alt+; | elevation を +5° 調整する |
| Alt+' | elevation を −5° 調整する |
| Alt+\ | 計算した path bearing に rotor を向ける |
| Alt+/ | rotor を停止する |

### WSJT-X

WSJT-X 連携は WSJT-X からの UDP メッセージを使用します。CQOps は ADIF メッセージを解析し、完了した QSO を自動的に記録できます。

WSJT-X が送信中は rig label がアクセント色になります。WSJT-X が通知した operator とアクティブな operator が一致しない場合、CQOps は警告を表示します。

### GPS

CQOps は GPS receiver から位置を読み取り、局の grid locator として使用できます。ポータブル、モービル、フィールド運用に適しています。

2 種類の backend に対応しています。

- **Serial** — USB-to-serial、内蔵 COM port、`/dev/ttyUSB0` などの serial port から GPS receiver へ直接接続します。
- **GPSD** — TCP で [gpsd](https://gpsd.io/) server に接続します。デフォルトは `127.0.0.1:2947` です。GPS を他のアプリと共有する場合や、ネットワーク経由で利用する場合に便利です。

ステータスバーの **GPS** indicator は次の状態を示します。

| 色 | 意味 |
|--------|---------|
| 赤 `GPS` | 未接続 / エラー |
| 黄 `GPS` | 接続済み、まだ fix なし |
| 白 `GPS` | fix 取得済み、位置確定 |

fix を取得すると、局の grid locator は GPS から算出した位置へ置き換えられ、status line に `(GPS)` と表示されます。

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

**Station & Logbook** 設定で **Grid from GPS** を有効にすると、QSO logging、APRS beacon、dashboard map、distance calculation に GPS grid を使用できます。

**Grid precision** は **Integration** メニューで 10、8、6 文字から選択します。デフォルトは 10 文字で、約 25 m の精度です。内部では常に完全な精度で grid を計算し、使用時に指定した長さへ切り詰めます。

### DX Cluster

**DX Cluster** 連携は telnet を使用し、インターネット接続が必要です。

デフォルト server:

```text
dxspots.com:7300
```

filter には次の項目があります。

- band、
- spotter continent、
- mode、
- age/time。

| Key | Action |
|---|---|
| Enter | **QSO form** に入力し、rig を tune して **QSO** へ戻る |
| Space | rig を tune し、**DX Cluster** に留まる |
| Backspace | すべての filter を消去する |

**DX Cluster** が接続されている場合、**QSO form** で次の追加機能を使用できます。

- **Send a spot** — フォーム入力後に **Ctrl+S** を押し、spot dialog を開いて DX spot を cluster へ送信します。
- **Nearest spots** — 周波数を tune すると、最大 3 件の近接 spot が **QSO form** に直接表示されます。ログ画面を離れずにバンド上の局を確認できます。**Ctrl+P** を押すと、最も近い spot から indicativo を入力できます。

### PSK Reporter

**PSK Reporter** 連携にはインターネット接続が必要です。現在の実際の伝搬状態を素早く確認でき、自局の信号を誰が受信しているか、または自局が誰を受信しているかをバンドごとに確認できます。

次の機能があります。

- propagation spot、
- band/time/mode filter、
- **F5** の ASCII world map。

### APRS

CQOps は 3 種類の APRS service type に対応しています。局の構成に合うものを選択してください。

| Service | Connection | Internet required |
|---|---|---|
| **APRS-IS** | APRS-IS server への TCP 接続 | 必要 |
| **KISS** | hardware KISS TNC への serial port 接続 | 不要 |
| **KISS Server** | Dire Wolf などの KISS TNC server への TCP 接続 | 不要。local network のみで可 |

**Integrations** メニューで service type を選択します。

```text
F9 → Integrations → APRS → Service (Space to cycle)
```

3 種類すべてで、近隣局の APRS position report を受信し、**CQOps Live** の local map に次の形式で表示できます。

- 標準 APRS symbol、
- callsign popup、
- 自動的な表示範囲調整、
- 設定可能な range circle。

すべての service は **periodic position beaconing** にも対応します。CQOps は設定した interval で局の grid locator を送信します。GPS が有効で **Grid from GPS** がオンの場合、beacon は GPS から取得した位置を自動的に使用します。ポータブルやモービル運用に適しています。

#### APRS-IS

インターネット経由で世界規模の APRS-IS network に接続します。次の情報が必要です。

- 有効なアマチュア無線の indicativo、
- indicativo から生成した APRS-IS passcode、
- インターネット接続。

デフォルト server:

```text
euro.aprs2.net:14580
```

APRS-IS は **F9 → Integrations → APRS** で全体設定します。ログブックごとの callsign、SSID、symbol、comment、beacon interval、range filter は **F9 → Logbooks → [active logbook] → APRS** で設定します。

#### KISS (serial)

serial port で hardware KISS TNC に直接接続します。インターネット接続は不要で、APRS frame は無線機を通して送受信されます。

**Integrations** メニューで serial port、baud rate、data bits、parity、stop bits、DTR/RTS を設定します。

```text
F9 → Integrations → APRS → Service: KISS
```

**KISS** を選択すると、serial 関連の **Port**、**Baud**、**Data bits**、**Parity**、**Stop bits**、**DTR**、**RTS** が表示されます。

**Test** ボタンは serial port を開き、TNC に接続できるか確認します。

#### KISS Server (TCP)

TCP で接続できる KISS TNC に接続します。たとえば同じマシンまたは local network 上で動作する [Dire Wolf](https://github.com/wb2osz/direwolf) instance を使用できます。インターネット接続は不要です。

**Integrations** メニューで host と port を入力します。

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

デフォルト: `127.0.0.1:8001`

#### Beaconing

beacon はログブックごとに設定した interval で送信されます。最小 interval は 1 分です。beacon には次の情報が含まれます。

- SSID 付き station callsign、
- 利用可能な場合は GPS から算出した grid locator、
- APRS symbol、
- 任意の comment。

**GPS** がアクティブで、**Station** 設定の **Grid from GPS** が有効な場合、beacon は GPS から算出した grid locator を自動的に使用します。移動中に手動で grid を更新する必要はありません。

Beacon interval およびその他のログブック別設定は次の場所で行います。

```text
F9 → Logbooks → [active logbook] → APRS
```

#### Receiving

受信した APRS position report はローカルにキャッシュされ、**CQOps Live dashboard** の map に表示されます。局は APRS symbol で表示され、クリックすると詳細を確認できます。表示範囲は、設定した range 内にあるすべての局が見えるよう自動調整されます。

APRS receive と beacon transmit は独立しています。送信せずに受信のみ、または受信せず送信のみの運用も可能です。**Integrations** メニューで APRS を有効にし、service type を設定してください。

### Solar Data

太陽活動データは hamqsl.com から取得し、次の情報を含みます。

- SFI、
- sunspot number、
- A/K indices、
- バンドごとのコンディション。

リアルタイム更新にはインターネット接続が必要です。一度正常に取得すれば、キャッシュデータはオフラインでも利用できます。

---

<a id="cqops-live-dashboard"></a>
## CQOps Live Dashboard

CQOps Live は、局の活動をリアルタイムで表示する内蔵ブラウザー dashboard です。

次の用途に適しています。

- field day の公開ディスプレイ、
- クラブ局の画面、
- コンテスト監視、
- 別室からの局の確認、
- イベントや展示会のブース。

### dashboard を有効にする

1. **F9** を押します。
2. **Integrations** を開きます。
3. **HTTP Server** を選択します。
4. **HTTP server** を有効にします。
5. 必要に応じて address と port を設定します。
6. **Ctrl+S** を押して保存します。
7. ブラウザーで dashboard を開きます。

デフォルト設定:

| Setting | Default |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

保存後、server は直ちに起動します。

> **Address binding:** デフォルトの `0.0.0.0` では、local network 上のどの端末からでも dashboard にアクセスできます。field day の表示、クラブ局の画面、別室からの監視に便利です。ローカルマシンだけに制限する場合は address を `127.0.0.1` に設定します。

### 表示モード

CQOps Live には 2 種類の表示モードがあります。

#### Overview mode

現在交信中の callsign がない場合に表示されます。

次の情報を表示します。

- **live maps** — 当日の QSO marker、自局 grid から各相手局までの great-circle path、近隣 APRS 局を表示する local APRS map、
- recent QSOs table、
- 局情報、
- 統計、
- 5 分、15 分、1 時間の rate tracking、
- top operator、
- 最長距離 QSO。

#### Active / Now Working mode

現在 callsign と交信中の場合に表示されます。

次の情報を表示します。

- 大きな callsign、
- submode indicator、
- 利用可能な場合は QRZ photo、
- band と mode badge、
- **DUPE / NEW CALL / NEW DXCC** indicator、
- distance と bearing、
- 自局 grid から相手局 grid までの強調された dashed map path。

### Info box

local map 上部の **Info box** は、5 秒ごとに次の module を切り替えます。

- band condition、
- solar activity、
- geomagnetic field、
- 最新の DX Cluster spot、
- band ごとの PSK Reporter report 数。

### Weather row

**Weather row** は、局の grid locator に対応する現在の Open-Meteo 情報を表示します。

- temperature、
- wind、
- humidity、
- icon。

天候データはブラウザー側で取得され、オフライン時には問題なく省略されます。

### Local map

右側の **local map** は、**近隣 APRS 局の監視**専用です。次の情報を表示できます。

- 標準 APRS symbol を使用した近隣局、
- hover/click で表示される callsign popup、
- 設定可能な range circle、
- 任意の day/night terminator overlay、
- 任意の RainViewer weather radar overlay。

### リアルタイム更新と性能

CQOps Live は Server-Sent Events（SSE）で更新されます。ページの再読み込みは不要です。

dashboard は低性能なハードウェアでも動作するよう設計されています。

- browser が map を描画する、
- browser が distance を計算する、
- browser が statistic を計算する、
- CQOps は軽量な JSON update を送信する、
- **HTTP server** が無効な場合は port を開かず、dashboard goroutine も実行されない。

### dashboard のカスタマイズ

**HTTP Server** 連携フォームでは、次の項目を設定できます。

| Field | Description |
|---|---|
| Header 1 | ページ header と hero area に表示する main title。未設定時は「CQOps Live」。 |
| Header 2 | title の下に表示する subtitle。未設定時は「Less clicking. More radio.」。 |
| Logo URL | 左上に表示する公開 image URL。未設定時は CQOps logo。 |
| Event Start | `YYYY-MM-DD` 形式の日付。その日以降の statistic と QSO list のみに filter する。 |

---

<a id="configuration"></a>
## 設定

**F9** で設定を開きます。

### 設定ファイル

| プラットフォーム | 設定パス |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

機密性の高い認証情報は、同じ設定ディレクトリ内の `secrets.enc` に分けて保存されます。

secrets はマシンに紐づく key で暗号化されます。設定を別のマシンへ移動した場合、認証情報を再入力する必要があります。

### 設定メニュー

**F9** で main menu を開き、次の項目を選択します。

| Menu | Configures |
|---|---|
| General | Units、timezone、partner map/picture、solar panel、SCP/REF data source、Kitty Graphics、Debug mode |
| Logbooks | Station callsign、grid、references、CQ/ITU zones、IARU region、GPS grid。ログブック別 Wavelog（URL、API key、station profile）。ログブック別 APRS（callsign、symbol、beacon、range） |
| Operators | multi-operator station 用の operator callsign と operator name profile |
| Rigs | rig preset: model、antenna、power、backend（None/flrig/Hamlib）、rotor、WSJT-X UDP |
| Contests | contest profile: name、date、ADIF contest ID、exchange templates、starting serial number |
| Integration | DX Cluster（host、port、login）、dashboard 用 HTTP Server（address、port、branding）、GPS service（serial/GPSD、grid precision） |
| Callbook | QRZ.com、HamQTH、QRZ.RU、Callook.info プロバイダー; 優先順位、base-call fallback、Wavelog lookup |
| Notifications | QSO saved alerts、Wavelog QSO sent status、dupe beep、error sounds |

### Multi-logbook

home、portable、contest、club 用に複数のログブックを使用できます。

**Ctrl+L** でアクティブなログブックを切り替えます。

各ログブックには固有の次の設定があります。

- 局情報、
- Wavelog 設定、
- contest 設定、
- operator 設定。

### Multi-operator

operator profile には次の情報があります。

- operator callsign、
- operator name。

**Ctrl+O** でアクティブな operator を切り替えます。

アクティブな operator は ADIF の `OPERATOR` field に保存され、Wavelog upload にも反映されます。

### Multi-rig

rig preset には次の情報が保存されます。

- backend、
- model、
- antenna、
- power、
- rotor 設定、
- WSJT-X 設定。

**Ctrl+R** でアクティブな rig を切り替えます。

### 暗号化された secrets

v0.8.7 以降、認証情報は暗号化して保存されます。

| 項目 | 値 |
|---|---|
| Secrets file | `secrets.enc` |
| 保存場所 | `config.yaml` と同じディレクトリ |
| Unix permission | 対応環境では `0600` |
| Encryption | マシンに紐づく key を使用した AES-256-GCM |
| 保護対象 | QRZ password、DX Cluster login、Wavelog API keys |

旧設定に含まれる平文の secrets は、初回起動時に移行されます。

`secrets.enc` が破損している場合、CQOps は警告付きで起動し、認証情報の再入力を求めます。

---

<a id="keyboard-shortcuts"></a>
## キーボードショートカット

### Global

| Key | Action |
|---|---|
| F1 | **QSO form** and **Recent QSOs** |
| F2 | **Partner view** |
| F4 | **DX Cluster** |
| F5 | **PSK Reporter** |
| F6 | **REF Lookup** |
| F7 | **Band Plan Browser** |
| F8 | **Logbook Editor** |
| F9 | **Configuration / main menu** |
| F10 | Quit |
| Ctrl+F9 | **Log viewer** |
| ? | **Help overlay** |
| Ctrl+L | Cycle active logbook |
| Ctrl+R | Cycle active rig |
| Ctrl+C | Cycle active contest |
| Ctrl+O | Cycle active operator |
| Esc | Back to previous screen |

### QSO form

| Key | Action |
|---|---|
| Tab | Next field |
| Shift+Tab | Previous field |
| ↑ / ↓ | Move within column |
| Enter | Save QSO, with duplicate confirmation if needed |
| Del | Clear all form fields |
| Ins | Lookup: Callbook, Wavelog, DXCC, and duplicate check |
| PgUp / PgDn | Cycle band, mode, or submode |
| Ctrl+S | Send DX spot from filled form |
| Ctrl+P | Fill call from nearest DXC spot |
| Ctrl+C | Cycle active contest |
| Alt+, | Adjust rotor azimuth −5° |
| Alt+. | Adjust rotor azimuth +5° |
| Alt+; | Adjust rotor elevation +5° |
| Alt+' | Adjust rotor elevation −5° |
| Alt+\ | Point rotor to bearing from own grid to partner grid |
| Alt+/ | Stop rotor |
| Alt+Ins / Alt+Home / Alt+PgUp | Recall favorite (slot 1/2/3) |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Save frequency, mode, band to favorite |

### Logbook Editor

| Key | Action |
|---|---|
| ↑ / ↓ | Navigate rows |
| PgUp / PgDn | Previous or next page |
| Home / End | First or last row |
| Enter / e | Edit selected QSO |
| Delete | Delete selected QSO |
| p | Purge all QSOs |
| Ctrl+C | Cycle contest filter |
| Ctrl+E | Export ADIF |
| Ctrl+I / Tab | Import ADIF |
| w | Upload unsent QSOs to Wavelog |
| Ctrl+W | Download contacts from Wavelog |
| Esc / F6 | Close editor and return to QSO form |

### DX Cluster

| Key | Action |
|---|---|
| ↑ / ↓ | Navigate spots |
| Enter | Fill QSO form, tune rig, and return to QSO |
| Space | Tune rig to selected spot and stay on DX Cluster |
| Home | Cycle band filter forward |
| End | Cycle band filter backward |
| `\` | Cycle spotter continent filter |
| Ins | Cycle mode filter forward |
| Del | Cycle mode filter backward |
| PgUp | Cycle time filter forward |
| PgDn | Cycle time filter backward |
| Backspace | Clear all filters |
| Esc / F4 | Return to QSO form |

### Partner view

| Key | Action |
|---|---|
| F2 | Cycle Partner view → Photo → Back |
| Esc / F1 | Return to QSO form |

---

<a id="troubleshooting"></a>
## トラブルシューティング

### CQOps が起動しない

次の項目を確認してください。

- ターミナルサイズが 80×24 文字以上であること、
- Windows ユーザーは Windows Terminal を使用していること、
- ネットワーク起動処理が停止原因でないか、次のコマンドで確認すること。

  ```bash
  cqops --offline
  ```

ログを確認します。

| プラットフォーム | ログのパス |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### rig が接続されない

flrig の場合:

- flrig が起動していることを確認する、
- アクティブな rig preset の port を確認する、
- デフォルト port は `12345`。

Hamlib の場合:

- `rigctld` が起動していることを確認する、
- host と port を確認する、
- 無線機/backend が要求する data に対応していることを確認する。

状態ラベルは問題の診断に役立ちます。

| 色 | 意味 |
|---|---|
| 白/デフォルト | 接続済み |
| 黄 | 無効または接続中 |
| 赤 | 失敗 |

reconnect toast が非表示になっている場合があります。CQOps は通知せず再試行できます。

### WSJT-X が自動ログしない

次を確認してください。

- **WSJT-X Settings → Reporting → UDP Server**、
- UDP host と port が CQOps のアクティブな rig preset と一致していること、
- WSJT-X 2.6 以降を使用していること、
- **WSJT** status label がアクティブであること、
- アクティブなログブックが正しいこと、
- アクティブな operator が正しいこと。

### Wavelog upload が失敗する

次を確認してください。

- Wavelog URL、
- API key、
- station profile ID、
- **WL** status label。

upload error は toast として表示されます。upload に失敗しても QSO はローカルに保存されたままです。個々の QSO の失敗によって、batch 内の残りの QSO が停止することはありません。

### 設定ファイルの問題

設定ファイル:

| プラットフォーム | パス |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets file:

```text
secrets.enc
```

secrets file は `config.yaml` と同じディレクトリに保存されます。

設定が破損している場合はファイルを移動または削除し、CQOps を再起動してください。セットアップウィザードが新しい設定を作成します。

`last_fetched_id` field は、Wavelog download が正常に完了した後にのみ表示されます。

### 性能上の問題

次を試してください。

- **General** 設定で map rendering を無効にする、
- 不要な場合は **Solar** panel を無効にする、
- オフライン時は **DX Cluster** や **PSK Reporter** などネットワーク負荷の高い画面を避ける、
- ネットワークが不安定な場合は `cqops --offline` を使用する。

---

<a id="reporting-bugs"></a>
## バグの報告

バグを報告する前に:

1. **F9 → General → Debug** で **Debug mode** を有効にするか、`config.yaml` に次を設定します。

   ```yaml
   debug: true
   ```

2. 問題を再現します。
3. 関連するログを添付します。

GitHub で issue を報告してください。

<https://github.com/szporwolik/cqops/issues>

次の情報を含めてください。

- `cqops --version` で表示される CQOps version、
- operating system、
- terminal emulator、
- 問題を再現する手順、
- 関連する debug log。
