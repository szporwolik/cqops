---
title: CQOps 用户手册
description: CQOps 的安装、配置与使用指南——一款快速、以终端为核心的业余无线电日志软件
---

# CQOps 用户手册

CQOps 是一款快速、以终端为核心的业余无线电日志软件，面向希望通过键盘可靠记录通联、同时保持较低系统开销的操作员。它适用于固定台站、便携操作、俱乐部台站、外场日，以及树莓派级设备或较旧的笔记本电脑。

CQOps 始终先将 QSO 保存在本地。基于互联网的集成功能均为可选项。

> **界面语言说明：** CQOps 应用程序目前仅提供英文界面。本手册使用简体中文进行说明，但保留屏幕名称、菜单项、字段、按钮和状态标签的英文原文，以便您能够在应用程序中准确找到它们。

<a id="contents"></a>
## 目录

1. [CQOps 是什么](#what-cqops-is)
2. [下载与安装](#download-and-installation)
3. [首次启动](#first-launch)
4. [记录您的第一个 QSO](#log-your-first-qso)
5. [主屏幕](#main-screen)
6. [常用工作流程](#common-workflows)
7. [QSO 记录](#qso-logging)
8. [日志簿编辑器与 ADIF](#logbook-editor-and-adif)
9. [竞赛](#contests)
    - [设置竞赛](#setting-up-a-contest)
    - [底部状态栏](#bottom-status-bar)
    - [竞赛统计面板](#contest-statistics-panel)
    - [竞赛 ADIF 导出](#contest-adif-export)
    - [竞赛模式行为](#contest-mode-behavior)
10. [收藏、参考编号与频率规划](#favorites-references-and-band-plans)
11. [集成功能](#integrations)
12. [CQOps Live 仪表板](#cqops-live-dashboard)
13. [配置](#configuration)
14. [键盘快捷键](#keyboard-shortcuts)
15. [故障排除](#troubleshooting)
16. [报告错误](#reporting-bugs)

---

<a id="what-cqops-is"></a>
## CQOps 是什么

CQOps 围绕快速 QSO 输入、本地优先日志记录和实用的外场操作而设计。

### 核心理念

- **以终端为核心的操作** — 针对键盘操作进行了优化。
- **离线优先日志记录** — 无需互联网连接即可在本地记录 QSO。内置用于仪表板的世界地图，可完全离线工作。
- **低系统开销** — 适用于树莓派级系统、较旧的笔记本电脑和共享台站电脑。
- **便携式设计** — 以单个 Go 二进制文件发布。
- **多个日志簿** — 适用于个人、便携、竞赛和俱乐部日志。
- **多个操作员** — 适用于轮换操作和共享俱乐部台站的工作流程。
- **多部电台** — 每个电台预设都可以保存独立的后端和 WSJT-X 设置。
- **可选集成功能** — 多提供商呼号数据库（QRZ.com、HamQTH、Callook.info）、Wavelog、DX Cluster、PSK Reporter、GPS、APRS、电台控制、天线旋转器控制、太阳活动数据和 CQOps Live 浏览器仪表板。

本地日志记录不需要互联网连接。在 `--offline` 模式下，网络功能会被跳过。

### CQOps 适合哪些用户

CQOps 特别适合：

- 便携操作员，
- SOTA 和 POTA 激活者，
- 俱乐部台站，
- 外场日团队，
- 偏好终端工作流程的操作员，
- 需要在操作员、日志簿或电台之间快速切换的台站。

CQOps 并不打算替代功能全面的桌面日志软件或基于 Web 的日志平台。它专注于快速终端日志记录、外场操作、离线使用和共享台站工作流程。

### 俱乐部与共享台站使用

CQOps 从设计之初就考虑了业余无线电俱乐部环境。当前操作员始终显示在状态栏中——**只需看一眼**，就能知道当前由谁操作。按一次 `Ctrl+O` 即可切换操作员并立即生效；之后的每个 QSO 都会写入该操作员的呼号和姓名。无需注销、无需密码提示，也不会打断操作。

日志簿、电台预设和竞赛也能以相同方式循环切换——`Ctrl+L`、`Ctrl+R`、`Ctrl+C`。拥有轮换操作员、多部电台和多个活动竞赛的俱乐部台站，可以在不到一秒内切换工作上下文，全程无需使用鼠标。

对于外场日和公众活动，**CQOps Live 仪表板**可以将实时地图、QSO 动态和统计信息投射到大屏幕上。访客和俱乐部成员无需围在操作员终端旁，就能观看台站工作。只需启用 HTTP server 集成，然后使用任何带有 Web 浏览器的设备访问即可。

---

<a id="download-and-installation"></a>
## 下载与安装

浏览所有发布版本：

<https://github.com/szporwolik/cqops/releases>

### Windows

| 软件包 | 链接 | 说明 |
|---|---|---|
| 安装程序 | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | 推荐大多数用户使用。将 CQOps 添加到 Start Menu 和 PATH。 |
| 便携 ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | 解压后直接运行，无需安装。 |

### Linux — Debian / Ubuntu / Pop!_OS / Linux Mint

添加 Cloudsmith APT 仓库并安装：

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

或直接下载 `.deb`：

| 架构 | 链接 | 适用于 |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | 大多数 Intel/AMD 电脑 |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64 位 ARM 系统 |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32 位 Raspberry Pi OS |

安装已下载的软件包：

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Fedora / RHEL / Rocky / AlmaLinux

添加 Cloudsmith RPM 仓库并安装：

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

### Linux — Arch / Manjaro / CachyOS

使用任意 AUR 助手从 AUR 安装：

```bash
yay -S cqops-bin
```

也可通过 `paru`、`pacaur`、`aura` 或手动 `makepkg` 安装。PKGBUILD 位于 [aur.archlinux.org/packages/cqops-bin](https://aur.archlinux.org/packages/cqops-bin)。

### Linux — 便携 Tarball

| 架构 | 链接 | 适用于 |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | 大多数 Intel/AMD 电脑 |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64 位 ARM 系统 |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32 位 Raspberry Pi OS |

### macOS

| 架构 | 链接 | 适用于 |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | M1/M2/M3 Mac |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Intel Mac |

手动安装：

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### 从源代码构建

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

从源代码构建需要 Go 1.26 或更高版本。

### 终端要求

| 要求 | 值 |
|---|---|
| 最小终端尺寸 | 80×24 个字符 |
| 推荐终端尺寸 | 80×43 个字符或更大 |
| 推荐的 Windows 终端 | Windows Terminal |
| 支持 Kitty 图形协议的终端 | [Kitty](https://sw.kovidgoyal.net/kitty/)、[Ghostty](https://ghostty.org/) 或 [WezTerm](https://wezfurlong.org/wezterm/) |

### 基本命令

```bash
cqops              # 启动 TUI
cqops --offline    # 启动但不进行网络活动
cqops --version    # 显示版本并退出
cqops --help       # 显示帮助
```

---

<a id="first-launch"></a>
## 首次启动

首次启动时，CQOps 会打开设置向导。进行本地日志记录只需要填写基本台站信息。网络集成功能可以跳过，并在以后配置。

### 向导页面

| 页面 | 配置内容 |
|---|---|
| `Station & Logbook` | 初始日志簿、台站呼号、操作员、网格定位符、可选参考编号和分区，以及 Wavelog URL/API/station profile ID |
| `Rig` | 电台预设、型号、天线、功率、后端、可选旋转器和可选 WSJT-X UDP 设置 |
| `Integrations` | 呼号查询设置（QRZ.com、HamQTH、Callook.info） |
| `General` | IANA 时区 |
| `Summary` | 检查并保存 |

支持的电台后端：

- `None`，
- flrig，
- Hamlib `rigctld`。

### 向导导航

| 按键 | 操作 |
|---|---|
| Ctrl+S | 验证并继续；在 `Summary` 页面保存并启动 CQOps |
| Esc | 返回上一页 |
| F10 | 退出 |
| Tab / Shift+Tab | 在字段之间移动 |
| Space | 切换复选框 |

以后可以使用 **F9** 更改向导中的设置。

---

<a id="log-your-first-qso"></a>
## 记录您的第一个 QSO

1. 启动 CQOps：

   ```bash
   cqops
   ```

2. 完成设置向导，至少填写您的呼号和网格定位符。

3. 按 **F1** 打开 QSO 表单。

4. 输入对方呼号。CQOps 会自动将呼号转换为大写。

5. 填写其余字段。如果当前电台通过 flrig 或 Hamlib 连接，CQOps 可以自动填写频率、波段、模式和子模式。

6. 按 **Enter** 保存。

7. 如果出现 **DUPE!** 警告，再次按 **Enter** 可强制保存，按 **Esc** 则取消。

保存后的 QSO 会立即出现在 `Recent QSOs` 表格中。

---

<a id="main-screen"></a>
## 主屏幕

CQOps 使用固定的终端布局：

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

### 状态栏

状态栏显示：

- CQOps 版本，
- 当前日志簿，
- 当前电台，
- 台站呼号，
- 当前操作员，
- 集成状态标签，
- 以 `L` 标记的本地时间，
- 以 `Z` 标记的 UTC 时间。

常见标签包括 **Net**、**WSJT**、**Rig**、**Flrig**、**Hamlib**、**Rotator**、**DXC**、**WL** 和 **GPS**。GPS 标签遵循相同的颜色约定：断开时为红色；已连接但尚无定位时为黄色；获得位置定位后为白色。

| 颜色 | 含义 |
|---|---|
| 白色/默认 | 已连接或处于活动状态 |
| 黄色 | 已禁用、正在连接或预期处于离线状态 |
| 红色 | 错误或已断开 |
| 强调色 + 粗体 | WSJT-X 正在发射 |

### 主标签页

| 按键 | 标签 | 屏幕 |
|---|---|---|
| F1 | QSO | QSO 表单和 `Recent QSOs` |
| F2 | QRZ | 对方信息视图：呼号数据库数据、地图、统计和照片 |
| F4 | DXC | DX Cluster 通报和过滤器 |
| F5 | HRD | PSK Reporter 通报和传播地图 |
| F6 | REF | SOTA/POTA/WWFF/IOTA 参考编号搜索 |
| F7 | BPL | `Band Plan Browser` |
| F8 | LOG | 日志簿编辑器、ADIF 和 Wavelog 同步 |
| F9 | CFG | 配置菜单 |

帮助栏会显示与当前屏幕相关的快捷键。按 **?** 打开完整帮助覆盖层。

---

<a id="common-workflows"></a>
## 常用工作流程

### 便携、SOTA 或 POTA 操作

离家前：

1. 在有互联网连接的情况下运行一次 CQOps。
2. 让 CQOps 下载或刷新缓存数据，例如太阳活动数据、REF 数据和 DXCC 前缀。
3. 检查 `Solar` 面板是否显示数据。
4. 检查 **F6** 上的 REF 搜索是否能够返回结果。

在外场：

1. 以离线模式启动 CQOps：

   ```bash
   cqops --offline
   ```

2. 正常记录。QSO 会保存在本地。
3. 恢复联网后，打开 **F8** 并按 **w**，将尚未发送的 QSO 上传到 Wavelog。

### 共享俱乐部台站与轮换操作

1. 打开 **F9 → Operators**。
2. 按 **Ins** 添加操作员配置文件。
3. 在 QSO 表单中按 **Ctrl+O** 切换当前操作员。
4. 保存前检查状态栏中的当前操作员。
5. 当多个操作员需要记录相似的通联而不希望重新输入整个表单时，使用 **Retain**。

当前操作员会保存到 ADIF 的 `OPERATOR` 字段中。

### 个人与俱乐部日志簿

1. 打开 **F9 → Logbooks**。
2. 按 **Ins** 创建各个日志簿。
3. 在 QSO 表单中按 **Ctrl+L** 切换当前日志簿。
4. 保存前检查状态栏中的当前日志簿。

每个日志簿都可以保存独立的台站信息、Wavelog 设置、竞赛设置和操作员。

### 多部电台

1. 打开 **F9 → Rigs**。
2. 按 **Ins** 创建电台预设。
3. 选择后端：`None`、flrig 或 Hamlib。
4. 在 QSO 表单中按 **Ctrl+R** 切换当前电台。

电台预设可以包含后端、型号、天线、功率、旋转器设置和 WSJT-X UDP 设置。

### WSJT-X 数字模式操作

启用 WSJT-X UDP 集成后，CQOps 可以接收来自 WSJT-X 的 ADIF 消息，并自动记录已完成的数字模式 QSO。

自动记录的 QSO：

- 保存到当前日志簿，
- 立即显示在 `Recent QSOs` 中，
- 跳过重复通联，
- 继承当前竞赛 ID，
- 当 Wavelog 已配置且可访问时，可以自动上传到 Wavelog。

如果 WSJT-X 报告的操作员与 CQOps 中的当前操作员不匹配，CQOps 会显示警告。

在较长的数字模式操作前，请检查：

- 当前日志簿，
- 当前操作员，
- 当前竞赛，
- WSJT-X 状态标签。

### Wavelog 同步

CQOps 始终先将 QSO 保存在本地。Wavelog 同步是可选功能。

| 操作 | 位置 | 快捷键 | 说明 |
|---|---|---|---|
| 上传尚未发送的 QSO | `Logbook Editor` | `w` | 每批上传 50 条 |
| 从 Wavelog 下载 | `Logbook Editor` | `Ctrl+W` | 使用 `last_fetched_id` 进行增量下载 |

每个 QSO 都会跟踪上传状态：

- 未发送，
- 已发送，
- 错误。

如果上传失败，QSO 仍保留在本地日志簿中，稍后可以重试。清空日志簿会将获取 ID 重置为 `0`，从而允许重新完整下载。

---

<a id="qso-logging"></a>
## QSO 记录

QSO 表单是主要的日志记录屏幕。按 **F1** 打开。

CQOps 可以从以下来源填充字段：

| 来源 | 字段 |
|---|---|
| flrig / Hamlib | 频率、分频工作时的接收频率、模式、子模式 |
| 呼号数据库（QRZ.com / HamQTH / Callook.info） | 姓名、QTH、网格、国家、CQ 区、ITU 区、DXCC、大洲 |
| REF 数据库 | SOTA、POTA、WWFF、IOTA 参考编号 |
| Wavelog 查询 | 配置后可显示已通联/已确认状态 |
| DXCC/前缀数据 | 前缀及国家相关数据 |

### 表单布局

| 左列 | 中列 | 右列 |
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

只有在竞赛处于活动状态时，交换字段才会显示。

底部一行包含：

- **Comment**，
- **Keep** — 在各个 QSO 之间保留 `Comment` 字段，
- **Retain** — 保存后保留整个表单内容。

`Band`、`Mode` 和 `Submode` 等字段可以使用 **PgUp/PgDn** 循环切换。

### 路径、方位和标记

当双方的网格定位符都已知时，CQOps 会显示距离和方位角。

QSO 表单还可以显示以下标记：

- **DUPE!**
- **New Call!**
- **New DXCC!**

### 保存

| 按键 | 操作 |
|---|---|
| Enter | 保存 QSO |
| Ctrl+S | 根据已填写的表单发送 DX spot |
| Esc | 取消重复通联确认 |
| 在 DUPE 确认时按 Enter | 仍然保存重复通联 |

---
<a id="logbook-editor-and-adif"></a>
## 日志簿编辑器与 ADIF

按 **F8** 打开 `Logbook Editor`。

可用于：

- 查看 QSO，
- 行内编辑，
- 删除 QSO，
- 导入 ADIF，
- 导出 ADIF，
- 上传到 Wavelog，
- 从 Wavelog 下载，
- 与竞赛相关的操作。

### 编辑 QSO

1. 使用 **↑/↓** 选择一行。
2. 按 **Enter** 或 **e**。
3. 编辑 QSO。
4. 使用 **Ctrl+S** 保存。

更改会立即显示在 `Recent QSOs` 中。

### ADIF 导入与导出

CQOps 支持 ADIF 3.1.7 的导入和导出。

| 操作 | 快捷键 |
|---|---|
| 导入 ADIF | Ctrl+I |
| 导出 ADIF | Ctrl+E |

导入时会验证记录、跳过重复项并显示摘要。当已配置 Wavelog 同步时，导入的 QSO 会被标记为待上传到 Wavelog。

导出可以包含所有 QSO，也可以只包含按竞赛筛选的 QSO。`CONTEST_ID` 会被保留。

### 数字模式处理

本手册所述的模式和子模式处理遵循 ADIF 3.1.7：

- FT8 作为独立模式导出。
- FT4 和 FT2 作为 MFSK 导出，并使用相应的子模式。
- 导入的旧式 MFSK + FT8 记录会规范化为独立的 FT8。

QSO 表单中有独立的 **Mode** 和 **Submode** 字段。两者都可以使用 **PgUp/PgDn** 循环切换。

---

<a id="contests"></a>
## 竞赛

CQOps 包含一个轻量级竞赛日志面板，专为**休闲参与竞赛**而设计——它不能替代 N1MM、Win-Test 或 TR4W 等专用竞赛日志软件。如果您要参加严肃的多人操作、多电台或辅助类别竞赛，请使用专门的竞赛日志软件。CQOps 的定位是：当您只想贡献一些分数、轻松查看速率，或在 SOTA/POTA 激活期间记录少量竞赛 QSO，同时又不想离开日常日志软件时，提供便捷支持。

<a id="setting-up-a-contest"></a>
### 设置竞赛

在 `Logbook Editor` 中按 **Ins** 创建或配置竞赛。

竞赛配置包括：

- 竞赛名称，
- 日期，
- ADIF contest ID，
- 交换模板。

#### 模板标记

| 标记 | 替换内容 |
|---|---|
| `@rst` | 发送或接收的 RST |
| `@serial` | 自动递增的序列号 |
| `@cqz` | 对方台站的 CQ 区 |
| `@mycqz` | 您自己的 CQ 区 |
| `@itu` | 对方台站的 ITU 区 |
| `@myitu` | 您自己的 ITU 区 |
| `@grid` | 对方台站的网格定位符 |
| `@mygrid` | 您自己的网格定位符 |

按 **Ctrl+C** 循环切换当前竞赛，或者从 `Contest` 菜单（**F7**）中选择。交换字段会自动显示在 QSO 表单中，序列号会自动递增。

<a id="bottom-status-bar"></a>
### 底部状态栏

当竞赛处于活动状态时，底部栏会显示实时摘要：

```text
 IARU-HF · IARU HF   45 QSOs   Started 16:13   Last 14:04 ago   Next #45   On 2:41
```

| 字段 | 含义 |
|---|---|
| `IARU-HF` | 竞赛 ADIF ID（机器可读的竞赛标识符） |
| `· IARU HF` | 竞赛显示名称——仅在与 ID 不同时显示 |
| `45 QSOs` | 本次竞赛会话中记录的 QSO 总数 |
| `Started 16:13` | 今天该竞赛第一个 QSO 的时间 |
| `Last 14:04 ago` | 距离最近一次竞赛 QSO 的时间 |
| `Next #45` | 下一个 QSO 将发送的序列号 |
| `On 2:41` | 总在台时间——所有小于 30 分钟的 QSO 间隔之和 |

当终端较窄（少于 120 列）时，`Started` 字段会隐藏。少于 100 列时，竞赛名称和在台时间也会隐藏。

<a id="contest-statistics-panel"></a>
### 竞赛统计面板

当竞赛处于活动状态且终端足够宽时，QSO 表单右侧会显示带黄色边框的紧凑统计面板：

```text
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

| 行 | 字段 | 含义 |
|---|---|---|
| **Rate** | `2/h` | 最近 **10 个 QSO** 的速率——短时爆发速度 |
| | `--/h` | 最近 **100 个 QSO** 的速率——在记录满 100 个 QSO 前显示 `--` |
| **Count** | `60m 0` | 最近 60 分钟内记录的 QSO 数 |
| | `hr 0` | 当前整点小时内（从 `:00` 开始）记录的 QSO 数 |
| **Peak** | `1m120` | 最佳 1 分钟速率：120/h 表示该分钟内完成了 2 个 QSO |
| | `10m 54` | 最佳 10 分钟滑动窗口：平均 54/h |
| | `60m 29` | 最佳 60 分钟滑动窗口：平均 29/h |
| **Avg** | `8/h` | 会话平均速率——QSO 总数 ÷ 从第一个 QSO 起经过的小时数 |
| | `Sess 5:36` | 从第一个 QSO 到最后一个 QSO 的会话总时长（H:MM 或仅分钟） |
| **Chart** | `max 1` | 最繁忙的一分钟有 1 个 QSO。柱形图显示每分钟 QSO 数 |
| | `-60m…now` | 左边缘为 60 分钟前，右边缘为当前时间 |

图表使用 Unicode 方块字符（`█`），按 4 行垂直柱形缩放。峰值速率省略 `/h` 后缀，因为 `Peak` 已表示“每小时”。所有持续时间都不显示秒，因为每分钟刷新时秒数只会造成视觉噪声。

<a id="contest-adif-export"></a>
### 竞赛 ADIF 导出

要提交竞赛日志，请在竞赛处于活动状态时打开 **Logbook Editor**（`Ctrl+E`）。应用竞赛过滤器后，ADIF 导出对话框会提供只导出**属于当前竞赛的 QSO**的选项。由此生成符合 ADIF 3.1.7 标准的文件，其中保留竞赛交换字段、序列号和竞赛 ADIF ID，可直接上传到竞赛主办方的自动接收系统或日志检查系统。

<a id="contest-mode-behavior"></a>
### 竞赛模式行为

当竞赛处于活动状态时：

- QSO 表单显示交换字段，
- 序列号自动递增，
- `Recent QSOs` 可以筛选竞赛 QSO，
- ADIF 导出保留 `CONTEST_ID`，
- QSO 表单、竞赛面板和太阳活动面板会显示黄色边框，以便视觉区分，
- DXC 通报会针对该竞赛中的所有 QSO（不仅仅是当天的 QSO）检查重复通联标记。

---

<a id="favorites-references-and-band-plans"></a>
## 收藏、参考编号与频率规划

### 收藏

收藏功能使用 3 个槽位保存频率、模式和波段预设，足以存放最常用的呼叫频率。快捷键使用 `Alt`，以避免与标准终端编辑键冲突，并确保在各种终端中可靠工作。

| 快捷键 | 操作 |
|---|---|
| Alt+Ins / Alt+Home / Alt+PgUp | 调用槽位 1、2 或 3 中的收藏 |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | 将当前频率、模式和波段保存到槽位 1、2 或 3 |

收藏保存在配置中，并在所有日志簿之间共享。

示例：

1. 输入 `145.55`。
2. 将模式设为 `FM`。
3. 将波段设为 `2m`。
4. 按 **Alt+Shift+Ins** 保存到槽位 1。
5. 以后按 **Alt+Ins** 即可调用该预设。

### REF 查询

按 **F6** 打开 `REF Lookup`。

它可以搜索：

- SOTA，
- POTA，
- WWFF，
- IOTA。

可以按前缀、名称或参考编号标识符进行搜索。选定的参考编号可以填入 QSO 表单。

### 频率规划浏览器

按 **F7** 打开 `Band Plan Browser`。

它提供以下内容的快速访问：

- 业余无线电波段，
- VHF/UHF 频率范围，
- CB，
- PMR446，
- 广播频率预设，
- `Portable`——常见便携/外场操作频率（SOTA、POTA、呼叫信道）。

选中的频率可用于调谐当前电台。频率规划数据也可以导出为 Markdown。

---
<a id="integrations"></a>
## 集成功能

所有集成功能均为可选项。即使不使用它们，本地日志记录仍可正常工作。

### 呼号数据库（QRZ.com、HamQTH、Callook.info）

CQOps 支持多个呼号数据库提供商，并按优先级级联查询。
在 QSO 表单中按 **Ins** 时，系统会按顺序查询各提供商，直到其中一个返回结果：

1. **QRZ.com** — 需要互联网连接和 QRZ XML 订阅。数据最全面。
2. **HamQTH** — 免费的全球服务。覆盖范围良好，需要免费账户。
3. **Callook.info** — 面向美国呼号的免费服务。无需账户，FCC 查询速度快。

如果优先级较高的提供商失败或已禁用，系统会尝试下一个提供商。
启用 **Base call fallback**（默认开启）后，如果完整呼号没有结果，CQOps 还会尝试不带前缀或后缀的基础呼号。

在 **F9 → Callbook** 中启用和配置提供商。

在 QSO 表单中按 **Ins**，可以填充以下呼号数据库字段：

- 姓名，
- QTH，
- 网格定位符，
- 国家，
- CQ/ITU 分区，
- DXCC，
- 大洲。

**F2** 上的 `Partner view` 可以在有照片时显示操作员照片。

> ⚠️ **实验性功能。** 照片显示可以使用 Kitty 终端图形协议，并需要兼容终端：Kitty、Ghostty 或 WezTerm。
> 在 **F9 → General → Kitty Graphics** 中启用。普通终端以及未启用图形转发的 SSH 会话会回退到字符图像。

### Wavelog

Wavelog 集成支持：

- 上传，
- 增量下载，
- 已通联/已确认查询。

Wavelog 按当前日志簿单独配置，包括：

- URL，
- API key，
- station profile ID。

CQOps 始终先将 QSO 保存在本地。Wavelog 上传失败不会删除本地数据。

### flrig

flrig 集成通过 HTTP 使用 XML-RPC。

默认端点：

```text
localhost:12345
```

CQOps 可以读取：

- 频率，
- 模式，
- 功率。

分频工作时，VFO A 映射为 Frequency，VFO B 映射为 Freq RX。

### Hamlib / rigctld

Hamlib 电台控制使用 `rigctld` TCP 守护进程。

根据电台和后端的支持情况，CQOps 可以查询：

- 频率，
- 模式，
- VFO，
- 分频状态，
- 功率。

在可能的情况下，CQOps 会妥善处理不支持 VFO 名称的后端。

### Hamlib 旋转器 / rotctld

> ⚠️ **实验性功能。** 旋转器控制仍属实验性功能。操作前务必确认天线的机械转动范围，并随时准备使用 **Alt+/** 立即停止运动。请谨慎使用——错误配置可能损坏旋转器或天线。

旋转器控制使用 Hamlib `rotctld`。

CQOps 支持：

- 方位角，
- 仰角，
- 停止命令。

| 快捷键 | 操作 |
|---|---|
| Alt+, | 方位角 −5° |
| Alt+. | 方位角 +5° |
| Alt+; | 仰角 +5° |
| Alt+' | 仰角 −5° |
| Alt+\ | 将旋转器指向计算出的路径方位 |
| Alt+/ | 停止旋转器 |

### WSJT-X

WSJT-X 集成使用来自 WSJT-X 的 UDP 消息。CQOps 会解析 ADIF 消息，并可自动记录已完成的 QSO。

当 WSJT-X 正在发射时，电台标签会变为强调色。如果 WSJT-X 报告的操作员与当前操作员不匹配，CQOps 会显示警告。

### GPS

CQOps 可以从 GPS 接收机读取位置，并将其用作台站网格定位符，非常适合便携、移动或外场操作。

支持两种后端：

- **Serial** — 通过串口直接连接 GPS 接收机（USB 转串口、内置 COM 端口或 `/dev/ttyUSB0`）。
- **GPSD** — 通过 TCP 连接到 [gpsd](https://gpsd.io/) 服务器（默认 `127.0.0.1:2947`）。当 GPS 需要与其他应用程序共享，或通过网络访问时非常有用。

状态栏中的 GPS 状态指示器含义如下：

| 颜色 | 含义 |
|---|---|
| 红色 `GPS` | 已断开/发生错误 |
| 黄色 `GPS` | 已连接，但尚未获得定位 |
| 白色 `GPS` | 已获得定位，位置已锁定 |

获得定位后，台站网格定位符会替换为 GPS 计算出的定位结果，并在状态行中标记为 `(GPS)`：

```text
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

在 `Station & Logbook` 设置中启用 **Grid from GPS**，可将 GPS 网格用于 QSO 日志记录、APRS 信标、仪表板地图和距离计算。

**Grid precision** — 可在 `Integration` 菜单中配置为 10、8 或 6 个字符。默认值为 10 字符（精度约 25 米）。系统始终在内部以完整精度计算网格，然后在实际使用时截断为配置的长度。

### DX Cluster

DX Cluster 集成通过 telnet 工作，并需要互联网连接。

默认服务器：

```text
dxspots.com:7300
```

过滤器包括：

- 波段，
- spotter 所在大洲，
- 模式，
- 年龄/时间。

| 按键 | 操作 |
|---|---|
| Enter | 填充 QSO 表单、调谐电台并返回 QSO 屏幕 |
| Space | 调谐到所选通报，并留在 DX Cluster 屏幕 |
| Backspace | 清除过滤器 |

当 DX Cluster 已连接时，QSO 表单会获得两项附加功能：

- **发送通报** — 填好表单后按 **Ctrl+S**，打开通报对话框并向集群发送 DX spot。
- **附近通报** — 调谐到某一频率后，QSO 表单中会直接显示最多三个邻近通报，因此无需离开日志屏幕即可查看当前波段活动。按 **Ctrl+P** 可从最近的通报填充呼号。

### PSK Reporter

PSK Reporter 集成需要互联网连接。它是快速检查实际传播情况的优秀工具——可以立即查看在任意波段上谁能收到您的信号，或您能收到谁的信号。

它提供：

- 传播通报，
- 波段/时间/模式过滤器，
- **F5** 上的 ASCII 世界地图。

### APRS

CQOps 支持三种 APRS 服务类型，请选择与您的台站设置相匹配的类型：

| 服务 | 连接方式 | 是否需要互联网 |
|---|---|---|
| **APRS-IS** | 通过 TCP 连接 APRS-IS 服务器 | 是 |
| **KISS** | 通过串口连接硬件 KISS TNC | 否 |
| **KISS Server** | 通过 TCP 连接 KISS TNC 服务器（例如 Dire Wolf） | 否（本地网络） |

在 `Integrations` 菜单中选择服务类型：

```text
F9 → Integrations → APRS → Service (Space to cycle)
```

三种服务都支持接收附近台站的 APRS 位置报告，并将它们显示在 CQOps Live 本地地图上，包括：

- 标准 APRS 符号，
- 呼号弹出信息，
- 自动调整视图范围，
- 可配置的范围圆。

所有服务也都支持**周期性位置发信标**。CQOps 会按照配置的时间间隔发送台站网格定位符。当 GPS 已启用且 **Grid from GPS** 开启时，信标会自动使用 GPS 位置，非常适合便携和移动操作。

#### APRS-IS

通过互联网连接到全球 APRS-IS 网络。需要：

- 有效的业余无线电呼号，
- APRS-IS passcode（根据呼号生成），
- 互联网连接。

默认服务器：

```text
euro.aprs2.net:14580
```

APRS-IS 在 **F9 → Integrations → APRS** 下全局配置。
每个日志簿的呼号、SSID、符号、注释、信标间隔和范围过滤器在 **F9 → Logbooks → [active logbook] → APRS** 下设置。

#### KISS（串口）

通过串口直接连接硬件 KISS TNC。无需互联网连接——APRS 帧通过您的电台发送和接收。

在 `Integrations` 菜单中配置串口、波特率、数据位、校验位、停止位以及 DTR/RTS：

```text
F9 → Integrations → APRS → Service: KISS
```

选择 KISS 后，会显示串口专用字段（Port、Baud、Data bits、Parity、Stop bits、DTR、RTS）。

**Test** 按钮会尝试打开串口，以验证 TNC 是否可访问。

#### KISS Server（TCP）

连接到可通过 TCP 访问的 KISS TNC，例如运行在本机或局域网内的 [Dire Wolf](https://github.com/wb2osz/direwolf) 实例。无需互联网连接。

在 `Integrations` 菜单中输入主机和端口：

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

默认值：`127.0.0.1:8001`

#### 发信标

信标会按照每个日志簿配置的间隔发送。最小间隔为 1 分钟。信标包含：

- 带 SSID 的台站呼号，
- 网格定位符（在可用时使用 GPS 位置），
- APRS 符号，
- 可选注释。

当 **GPS** 处于活动状态，并且在 `Station` 设置中启用了 **Grid from GPS** 时，信标会自动使用 GPS 计算的网格定位符；移动过程中无需手动更新网格。

信标间隔和其他每日志簿设置位于：

```text
F9 → Logbooks → [active logbook] → APRS
```

#### 接收

接收到的 APRS 位置报告会缓存在本地，并显示在 CQOps Live 仪表板地图上。台站使用各自的 APRS 符号显示，点击后可查看详情。显示范围会自动调整，以显示配置距离范围内的所有可见台站。

APRS 接收与信标发送互相独立——可以只接收而不发送，也可以只发送而不接收。只需在 `Integrations` 菜单中启用 APRS，并设置服务类型。

### 太阳活动数据

太阳活动数据来自 hamqsl.com，包括：

- SFI，
- 太阳黑子数，
- A/K 指数，
- 各波段传播条件。

实时更新需要互联网连接。成功获取一次后，缓存数据可在离线状态下继续使用。

---

<a id="cqops-live-dashboard"></a>
## CQOps Live 仪表板

CQOps Live 是一个内置的浏览器仪表板，用于实时显示台站活动。

适用于：

- 外场日公众展示，
- 俱乐部台站屏幕，
- 竞赛监控，
- 从另一个房间查看台站，
- 活动或展会展台。

### 启用仪表板

1. 按 **F9**。
2. 打开 **Integrations**。
3. 进入 **HTTP Server**。
4. 启用 **HTTP server**。
5. 可选择设置地址和端口。
6. 按 **Ctrl+S** 保存。
7. 在浏览器中打开仪表板。

默认设置：

| 设置 | 默认值 |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

保存后，服务器会立即启动。

> **地址绑定：** 默认的 `0.0.0.0` 会使仪表板可由局域网中的任何设备访问，适合外场日展示、俱乐部台站屏幕或从另一个房间查看台站。将地址设为 `127.0.0.1` 可限制为仅本机访问。

### 显示模式

CQOps Live 有两种显示模式。

#### Overview 模式

当当前没有正在操作的呼号时显示。

内容包括：

- **实时地图** — 显示当天的 QSO 标记，以及从本台网格到每个对方台站的大圆路径；还包含用于显示附近 APRS 台站的本地 APRS 地图，
- 最近 QSO 表格，
- 台站信息，
- 统计信息，
- 5 分钟、15 分钟和 1 小时速率跟踪，
- 排名靠前的操作员，
- 距离最远的 QSO。

#### Active / Now Working 模式

当正在操作某个呼号时显示。

内容包括：

- 大号呼号，
- 子模式指示，
- 可用时显示 QRZ 照片，
- 波段和模式标记，
- DUPE / NEW CALL / NEW DXCC 指示，
- 距离和方位，
- 从本台网格到对方网格的高亮虚线路径。

### 信息框

本地地图上方的信息框每 5 秒轮换显示以下模块：

- 波段条件，
- 太阳活动，
- 地磁场，
- 最新 DX Cluster 通报，
- PSK Reporter 各波段报告数量。

### 天气行

天气行会根据台站网格定位符显示 Open-Meteo 当前天气：

- 温度，
- 风，
- 湿度，
- 图标。

天气数据由浏览器端获取；离线时会平稳降级，不影响其他功能。

### 本地地图

右侧本地地图专用于 **APRS 邻近区域监控**，可查看台站附近哪些用户正在使用 APRS。它可以显示：

- 使用标准 APRS 符号显示的附近台站，
- 鼠标悬停/点击时显示呼号弹出信息，
- 可配置的范围圆，
- 可选的昼夜分界线覆盖层，
- 可选的 RainViewer 天气雷达覆盖层。

### 实时更新与性能

CQOps Live 通过 Server-Sent Events（SSE）更新，无需刷新页面。

该仪表板针对低功耗硬件进行了设计：

- 浏览器负责地图渲染，
- 浏览器负责距离计算，
- 浏览器负责统计计算，
- CQOps 仅推送轻量级 JSON 更新，
- 当 HTTP server 被禁用时，不会打开端口，也不会运行仪表板 goroutine。

### 仪表板自定义

在 `HTTP Server` 集成表单中，可以配置：

| 字段 | 说明 |
|---|---|
| Header 1 | 页面标题和主视觉区域中显示的主标题。未设置时使用 “CQOps Live”。 |
| Header 2 | 标题下方的副标题。未设置时使用 “Fast, portable ham radio logger”。 |
| Logo URL | 显示在左上角的公开可访问图片 URL。未设置时使用 CQOps 标志。 |
| Event Start | `YYYY-MM-DD` 格式的日期。统计和 QSO 列表仅显示从该日期开始的数据。 |

---
<a id="configuration"></a>
## 配置

按 **F9** 打开配置。

### 配置文件

| 平台 | 配置路径 |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

敏感凭据会单独保存在同一配置目录中的 `secrets.enc` 文件里。

这些机密信息使用与当前机器绑定的密钥加密。将配置移动到另一台机器时，必须重新输入凭据。

### 配置菜单

按 **F9** 打开主菜单，然后选择：

| 菜单 | 配置内容 |
|---|---|
| General | 单位、时区、对方地图/图片、太阳活动面板、SCP/REF 数据源、Kitty 图形、调试模式 |
| Logbooks | 台站呼号、网格、参考编号、CQ/ITU 区、IARU 区域、GPS 网格；每日志簿 Wavelog（URL、API key、station profile）；每日志簿 APRS（呼号、符号、信标、范围） |
| Operators | 多操作员台站的操作员呼号和姓名配置 |
| Rigs | 电台预设：型号、天线、功率、后端（None/flrig/Hamlib）、旋转器、WSJT-X UDP |
| Contests | 竞赛配置：名称、日期、ADIF contest ID、交换模板、起始序列号 |
| Integration | DX Cluster、仪表板 HTTP server、GPS、APRS、Solar、PSK Reporter |
| Callbook | QRZ.com、HamQTH、Callook.info 提供商；优先级排序、基础呼号回退、Wavelog 查询 |
| Notifications | QSO 保存提示、Wavelog QSO 发送状态、重复通联蜂鸣、错误声音 |

### 多日志簿

使用多个日志簿分别管理固定台、便携、竞赛和俱乐部操作。

按 **Ctrl+L** 循环切换当前日志簿。

每个日志簿都保存独立的：

- 台站信息，
- Wavelog 设置，
- 竞赛设置，
- 操作员设置。

### 多操作员

操作员配置包含：

- 操作员呼号，
- 操作员姓名。

按 **Ctrl+O** 循环切换当前操作员。

当前操作员会保存到 ADIF 的 `OPERATOR` 字段，并随 Wavelog 上传一同发送。

### 多部电台

电台预设保存：

- 后端，
- 型号，
- 天线，
- 功率，
- 旋转器设置，
- WSJT-X 设置。

按 **Ctrl+R** 循环切换当前电台。

### 加密机密信息

从 v0.8.7 开始，凭据会加密保存。

| 项目 | 值 |
|---|---|
| 机密文件 | `secrets.enc` |
| 位置 | 与 `config.yaml` 相同的目录 |
| Unix 权限 | 在支持的平台上为 `0600` |
| 加密方式 | 使用机器绑定密钥的 AES-256-GCM |
| 受保护数据 | QRZ 密码、DX Cluster 登录信息、Wavelog API keys |

旧配置中的明文机密信息会在首次运行时迁移。

如果 `secrets.enc` 已损坏，CQOps 会在启动时显示警告，并要求重新输入凭据。

---

<a id="keyboard-shortcuts"></a>
## 键盘快捷键

### 全局

| 按键 | 操作 |
|---|---|
| F1 | QSO 表单和 `Recent QSOs` |
| F2 | `Partner view` |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | `REF Lookup` |
| F7 | `Band Plan Browser` |
| F8 | `Logbook Editor` |
| F9 | 配置/主菜单 |
| F10 | 退出 |
| Ctrl+F9 | 日志查看器 |
| ? | 帮助覆盖层 |
| Ctrl+L | 循环切换当前日志簿 |
| Ctrl+R | 循环切换当前电台 |
| Ctrl+C | 循环切换当前竞赛 |
| Ctrl+O | 循环切换当前操作员 |
| Esc | 返回上一屏幕 |

### QSO 表单

| 按键 | 操作 |
|---|---|
| Tab | 下一个字段 |
| Shift+Tab | 上一个字段 |
| ↑ / ↓ | 在当前列中移动 |
| Enter | 保存 QSO；如有需要会进行重复通联确认 |
| Del | 清除所有表单字段 |
| Ins | 查询 Callbook、Wavelog、DXCC，并检查重复通联 |
| PgUp / PgDn | 循环切换波段、模式或子模式 |
| Ctrl+S | 根据已填写的表单发送 DX spot |
| Ctrl+P | 从最近的 DXC spot 填充呼号 |
| Ctrl+C | 循环切换当前竞赛 |
| Alt+, | 旋转器方位角 −5° |
| Alt+. | 旋转器方位角 +5° |
| Alt+; | 旋转器仰角 +5° |
| Alt+' | 旋转器仰角 −5° |
| Alt+\ | 将旋转器指向从本台网格到对方网格的方位 |
| Alt+/ | 停止旋转器 |
| Alt+Ins / Alt+Home / Alt+PgUp | 调用收藏（槽位 1/2/3） |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | 将频率、模式和波段保存到收藏 |

### Logbook Editor

| 按键 | 操作 |
|---|---|
| ↑ / ↓ | 在各行之间导航 |
| PgUp / PgDn | 上一页或下一页 |
| Home / End | 第一行或最后一行 |
| Enter / e | 编辑所选 QSO |
| Delete | 删除所选 QSO |
| p | 清空所有 QSO |
| Ctrl+C | 循环切换竞赛过滤器 |
| Ctrl+E | 导出 ADIF |
| Ctrl+I / Tab | 导入 ADIF |
| w | 将尚未发送的 QSO 上传到 Wavelog |
| Ctrl+W | 从 Wavelog 下载通联记录 |
| Esc / F6 | 关闭编辑器并返回 QSO 表单 |

### DX Cluster

| 按键 | 操作 |
|---|---|
| ↑ / ↓ | 在通报之间导航 |
| Enter | 填充 QSO 表单、调谐电台并返回 QSO 屏幕 |
| Space | 调谐到所选通报并留在 DX Cluster 屏幕 |
| Home | 向前循环切换波段过滤器 |
| End | 向后循环切换波段过滤器 |
| `\` | 循环切换 spotter 大洲过滤器 |
| Ins | 向前循环切换模式过滤器 |
| Del | 向后循环切换模式过滤器 |
| PgUp | 向前循环切换时间过滤器 |
| PgDn | 向后循环切换时间过滤器 |
| Backspace | 清除所有过滤器 |
| Esc / F4 | 返回 QSO 表单 |

### Partner view

| 按键 | 操作 |
|---|---|
| F2 | 循环切换 Partner view → Photo → Back |
| Esc / F1 | 返回 QSO 表单 |

---

<a id="troubleshooting"></a>
## 故障排除

### CQOps 无法启动

请检查：

- 终端尺寸至少为 80×24，
- Windows 用户是否正在使用 Windows Terminal，
- 是否因网络启动操作造成阻塞；可尝试：

  ```bash
  cqops --offline
  ```

检查日志：

| 平台 | 日志路径 |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### 电台无法连接

对于 flrig：

- 确认 flrig 正在运行，
- 确认当前电台预设中的端口，
- 默认端口为 `12345`。

对于 Hamlib：

- 确认 `rigctld` 正在运行，
- 确认主机和端口，
- 检查您的电台/后端是否支持所请求的数据。

状态标签可以帮助诊断问题：

| 颜色 | 含义 |
|---|---|
| 白色/默认 | 已连接 |
| 黄色 | 已禁用或正在连接 |
| 红色 | 连接失败 |

重新连接提示可能会被抑制。CQOps 可以在不显示提示的情况下静默重试。

### WSJT-X 无法自动记录

请检查：

- WSJT-X **Settings → Reporting → UDP Server**，
- UDP 主机和端口是否与 CQOps 当前电台预设一致，
- 是否使用 WSJT-X 2.6 或更高版本，
- WSJT 状态标签是否处于活动状态，
- 当前日志簿是否正确，
- 当前操作员是否正确。

### Wavelog 上传失败

请检查：

- Wavelog URL，
- API key，
- station profile ID，
- **WL** 状态标签。

上传错误会以提示消息显示。即使上传失败，QSO 仍会保存在本地。单个 QSO 的失败不会阻止同一批次中其余 QSO 的上传。

### 配置文件问题

配置文件：

| 平台 | 路径 |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

机密文件：

```text
secrets.enc
```

机密文件与 `config.yaml` 保存在同一目录中。

如果配置已损坏，请移动或删除该文件，然后重新启动 CQOps。设置向导会创建新的配置。

只有在成功从 Wavelog 下载后，才会出现 `last_fetched_id` 字段。

### 性能问题

可以尝试：

- 在 `General` 设置中禁用地图渲染，
- 不需要时禁用 `Solar` 面板，
- 离线时避免使用 DX Cluster 和 PSK Reporter 等网络负载较高的屏幕，
- 网络不稳定时使用 `cqops --offline`。

---

<a id="reporting-bugs"></a>
## 报告错误

报告错误前：

1. 在 **F9 → General → Debug** 中启用 **Debug mode**，或者在 `config.yaml` 中设置：

   ```yaml
   debug: true
   ```

2. 重现问题。
3. 附上相关日志。

在 GitHub 上报告问题：

<https://github.com/szporwolik/cqops/issues>

请包含：

- `cqops --version` 显示的 CQOps 版本，
- 操作系统，
- 终端模拟器，
- 重现步骤，
- 相关调试日志。
