---
title: CQOps 用户手册
description: CQOps 设置及固定台站、野外通联记录实用指南
---

# CQOps 用户手册

CQOps 是一款通过键盘操作的业余无线电日志软件，适用于家庭台站、便携操作、俱乐部台站和休闲竞赛。QSO 始终先保存在本机；互联网服务均为可选项。可以先使用手动记录，再按需添加电台控制和在线服务。

本手册保留英文界面的菜单及字段名称，方便对照查找。快捷键随当前页面而变化，请查看底部帮助栏或按 **?**。

## 目录

1. [安装](#installation)
2. [首次设置](#setup)
3. [记录第一个 QSO](#first-qso)
4. [页面与状态](#screens)
5. [日常记录](#logging)
6. [台站配置](#profiles)
7. [日志簿与备份](#logbook)
8. [电台与数字模式](#radio)
9. [在线服务](#online)
10. [GPS 与 APRS](#position)
11. [野外操作](#portable)
12. [竞赛](#contests)
13. [CQOps Live](#dashboard)
14. [快捷键参考](#keys)
15. [故障排查与帮助](#help)

<a id="installation"></a>

## 安装

从[发行页面](https://github.com/szporwolik/cqops/releases)下载 CQOps。终端窗口至少需要 75 × 24 个字符；80 × 43 或更大更便于操作。

| 系统 | 安装方式 |
|---|---|
| Windows | 下载 `cqops-setup.exe`，或解压 `cqops-windows-portable.zip` 免安装使用。推荐 Windows Terminal。 |
| Debian、Ubuntu、Linux Mint、Pop!_OS | 下载对应的 `.deb`：大多数 Intel/AMD 电脑选 `amd64`，64 位 ARM 选 `arm64`，32 位 Raspberry Pi OS 选 `armhf`。使用软件包安装器打开。 |
| Fedora、RHEL、Rocky、AlmaLinux | 使用下方软件源安装命令。 |
| Arch、Manjaro、CachyOS | 使用 `paru -S cqops-bin` 或 `yay -S cqops-bin` 安装 AUR 软件包。 |
| 其他 Linux 系统 | 从发行页面下载适合处理器的 Linux `.tar.gz` 并解压。 |
| macOS | Apple Silicon 下载 `cqops-darwin-arm64`，Intel 下载 `cqops-darwin-amd64`，再执行下方命令。 |

Debian 系统也可通过软件源安装：

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

Fedora 系统：

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

macOS 请在下载目录执行以下命令，将 `FILE` 替换为实际下载的完整文件名：

```bash
chmod +x FILE
sudo mv FILE /usr/local/bin/cqops
```

运行 `cqops` 或已解压的便携版程序。`cqops --offline` 以离线模式启动，`cqops --version` 显示版本，`cqops --help` 显示启动选项。更新前请导出日志。

<a id="setup"></a>

## 首次设置

首次启动向导要求填写日志簿名称、台站呼号、Maidenhead 网格定位码和洲。**台站呼号**是发射时使用的呼号；**操作员配置**标识实际操作该台站的人。

**Ctrl+A** 可显示选填的台站参考编号及 CQ/ITU 分区。自己的 SOTA/POTA/WWFF 编号应填写在台站／日志簿设置中；QSO 表单中的编号属于对方台站。IARU 区域可稍后在 **F9 → Logbooks** 设置。

创建电台配置，填写名称、天线和功率。手动输入频率及模式时选择 **None**，也可选择 **flrig** 或 **Hamlib**。建议先确认基本记录正常，再设置可选连接。

**Tab / Shift+Tab** 切换字段，**Space** 更改选项，**Save & Next** 进入下一步。**Esc** 返回，**F10** 退出。核对摘要后保存。CQOps 会识别电脑时区，但 QSO 日期和时间均使用 UTC。操作前请检查电脑时钟。

<a id="first-qso"></a>

## 记录第一个 QSO

1. 按 **F1**，确认日志簿、台站呼号、操作员、电台和竞赛设置正确。
2. 输入对方呼号。如已配置查询服务，按 **Ins** 查询。
3. 检查 UTC 日期／时间、以 MHz 为单位的频率、波段、模式及收发信号报告。
4. 按需填写姓名、QTH、网格、参考编号或备注。
5. 按 **Enter** 保存。通联会出现在 Recent QSOs 中。

如果 **DUPE!** 要求确认，再按 **Enter** 仍然保存，按 **Esc** 取消确认。重复提示意味着需要核对，并不意味着一定要丢弃该通联。

<a id="screens"></a>

## 页面与状态

| 按键 | 页面 | 用途 |
|---|---|---|
| F1 | QSO | 输入通联、查看最近 QSO |
| F2 | Partner | 对方资料、地图、统计、照片 |
| F3 | APRS | 附近台站 |
| F4 | DX Cluster | 频点通告与筛选 |
| F5 | PSK Reporter | 数字模式接收报告 |
| F6 | References | SOTA、POTA、WWFF、IOTA 查询 |
| F7 | Band Plan | 频率与预设 |
| F8 | Logbook | 编辑、导入、导出、同步 |
| F9 | Configuration | 台站与服务设置 |
| F10 | Quit | 退出 CQOps |

顶部显示当前台站配置、本地时间（**L**）及 UTC（**Z**）。连接标签通常以白色表示正常、黄色表示禁用／连接中／等待、红色表示错误。WSJT 发射时会高亮。**WL!** 表示使用了不支持的旧 Wavelog 密钥。

<a id="logging"></a>

## 日常记录

用 **Tab / Shift+Tab** 切换字段，**PgUp / PgDn** 切换波段、模式或子模式。**Shift+Backspace** 清空当前字段，**Del** 清空整个表单。异频操作时请核对 **Freq RX**。

**Keep** 在保存后保留备注。**Retain** 保留整个表单，因此保存下一个通联前要检查呼号、时间、报告和参考编号。竞赛交换字段仅在竞赛启用时出现。其他特殊兴趣组信息可填入 **SIG / SIG Info**。

已知双方网格时，CQOps 会显示距离和方位。呼号查询结果可能是家庭地址，而非当前便携位置，请核实。新呼号、新 DXCC 和重复标记可辅助判断。

**F6** 按名称或编号搜索参考编号，并可填写对方的编号。**F7** 浏览频率规划，也可调整已连接电台的频率。条目仅供操作参考，并不代表发射许可；请确认操作权限和当地频率规划。

三个共用收藏位可保存频率、模式和波段：

| 位置 | 调用 | 保存当前值 |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

<a id="profiles"></a>

## 台站配置

在 **F9** 的对应菜单创建日志簿、操作员、电台和竞赛，**Ins** 添加条目。QSO 页面可使用：

| 快捷键 | 切换内容 |
|---|---|
| Ctrl+L | 日志簿 |
| Ctrl+O | 操作员 |
| Ctrl+R | 电台 |
| Ctrl+C | 竞赛 |

各日志簿分别保存台站资料和 Wavelog/APRS 设置。操作员配置标识实际操作者，其呼号记录在 ADIF `OPERATOR` 中。电台配置保存设备、功率、电台／旋转器控制和 WSJT-X 设置。每次切换后都应检查状态栏，尤其是在自动记录数字通联时。

其他 **F9** 菜单涵盖显示、单位、时区、呼号服务、集成和提示音。

<a id="logbook"></a>

## 日志簿与备份

在 **F8** 选择 QSO，按 **Enter** 或 **e** 编辑，再按 **Enter** 保存并确认。**Delete** 删除选中的通联。批量修改前先备份；**Ctrl+P** 是删除所有 QSO 的危险操作，不是搜索。

| F8 快捷键 | 操作 |
|---|---|
| Ctrl+I | 导入 ADIF、验证记录并跳过重复项 |
| Ctrl+E | 导出全部通联或按竞赛筛选的通联 |
| Ctrl+W | 将未发送的通联上传到 Wavelog |
| Alt+W | 从 Wavelog 下载 |

检查导入摘要和导出范围。导入的通联之后也可能上传到 Wavelog。CQOps 支持 ADIF 3.1.7，并保留竞赛 ID 和交换信息。每个日志簿都应单独备份，最好存到另一台设备。ADIF 备份的是通联，不包含所有程序设置或凭据。

Linux/macOS 配置位于 `~/.config/cqops/config.yaml`，Windows 位于 `%APPDATA%\cqops\config.yaml`。凭据单独存放在 `secrets.enc`；更换电脑后需要重新输入。不要把删除配置文件作为排障的第一步。

<a id="radio"></a>

## 电台与数字模式

### 电台控制

在 **F9 → Rigs** 选择 flrig 或 Hamlib，并匹配连接参数。先启动 flrig 或 `rigctld`。flrig 通常使用 `localhost:12345`。频率、模式、异频和功率能否读取取决于电台。选择 **None** 时手动输入。

### WSJT-X

使用 WSJT-X 2.6 或更新版本。将 **Settings → Reporting → UDP Server** 与 CQOps 当前电台配置的 UDP 参数对应。在 WSJT-X 中记录一次已完成的测试通联，并确认 CQOps 能收到。

收到的 QSO 使用当前日志簿和竞赛；重复项会跳过。开始前检查操作员和 WSJT 指示。操作员不一致时 CQOps 会警告。配置后还可继续上传至 Wavelog。正确选择 Mode/Submode；FT8 导出为 FT8，FT4/FT2 导出为 MFSK 加相应子模式。

### 天线旋转器控制

通过 Hamlib `rotctld` 控制属于实验性功能。使用前核对方向和机械限位，并准备可靠的停止方式。错误设置可能损坏天线、旋转器或馈线。

| 快捷键 | 操作 |
|---|---|
| Alt+, / Alt+. | 方位角 −5° / +5° |
| Alt+' / Alt+; | 仰角 −5° / +5° |
| Alt+\ | 转向计算出的方位 |
| Alt+/ | 停止转动 |

<a id="online"></a>

## 在线服务

### 呼号查询

在 **F9 → Callbook** 配置服务和优先顺序，再在 QSO 表单按 **Ins**。CQOps 按顺序尝试已启用服务。基本呼号回退查询可能去掉便携前后缀；请核对返回的位置。

| 服务 | 要求 |
|---|---|
| QRZ.com | XML 订阅及登录凭据 |
| HamQTH | 免费账号 |
| QRZ.RU | 独立于网站账号的 API 登录 |
| Callook.info | 美国呼号；无需账号 |

**F2** 显示对方资料。照片取决于服务和终端；General 中的实验性 **Kitty Graphics** 需要 Kitty、Ghostty、WezTerm 等兼容终端。

### Wavelog

每个日志簿分别设置 URL、API v2 令牌（`wl2_…`）和台站配置。不支持旧 v1 密钥。选择 Wavelog 台站后可能自动填写本地资料，保存前请检查呼号、网格和参考编号。

QSO 先保存在本机。上传失败可用 **F8 → Ctrl+W** 重试，**Alt+W** 下载通联。打开关联的 QSO 编辑时，CQOps 可能从 Wavelog 刷新内容。在线编辑和删除也会影响远程副本。请阅读确认提示，尤其在离线时；不要假定仅本地的修改已经同步到 Wavelog。


俱乐部电台：请使用所有者的 `wl2_` API 密钥，并在日志表单中启用 **共享俱乐部电台** 选项。已同步的通联将变为只读——编辑和删除请在 Wavelog 端完成，CQOps 不会为这些通联发送 PATCH 或 DELETE 请求。新通联照常上传，并归入当前操作员（未选择操作员时归入电台呼号）。 API 密钥经加密存储；启用此选项后，保存后将不再显示——留空密钥字段可继续使用原密钥，输入新密钥则可替换。
### DX Cluster 与传播

在 Integrations 设置 DX Cluster，然后打开 **F4**。**b / c / m / t** 按波段、发布者所在洲、模式及时间筛选，**Backspace** 清除筛选。**Enter** 填写 QSO、调谐已连接电台并返回 F1；**Space** 只调谐而不离开集群页面。

F1 上 **Ctrl+S** 打开发送通告窗口，**Ctrl+P** 取用显示的最近通告呼号。发送前请核实。**F5** 显示 PSK Reporter 接收报告，并不保证当前传播条件。Solar 显示 HamQSL 数据；缓存值可能已过时。

<a id="position"></a>

## GPS 与 APRS

### GPS

在 Integrations 配置串口 GPS 或 GPSD，并在台站／日志簿设置启用 **Grid from GPS**，用于 QSO、方位、APRS 和仪表板。GPS 红色表示错误，黄色表示未定位，白色表示已定位。操作前检查网格。可选 6、8、10 字符；字符更多并不保证接收机定位更准确。

### APRS

| 服务 | 连接 |
|---|---|
| APRS-IS | 互联网 APRS 服务器 |
| KISS | 串口硬件 TNC 和电台 |
| KISS Server | Dire Wolf 等 TCP TNC，可本地运行 |

在 **F9 → Integrations → APRS** 选择服务，在 **F9 → Logbooks → [active logbook] → APRS** 设置呼号／SSID、符号、备注、范围和信标间隔。只有打算发射时才启用 **APRS TX** 和 **Send beacons**。仅接收时显示 **APRS-RX**。位置信标会公开所在地，请先检查位置并考虑接收对象。

自动信标最短间隔为五分钟。**F3** 显示最近收到的台站：方向键选择，**Enter** 填入 QSO，**d / t / s** 筛选距离／时间／类型，**Backspace** 清除筛选，**b** 立即发送已配置的信标。使用 GPS 的信标需要 **Grid from GPS** 和有效位置。

<a id="portable"></a>

## 野外操作

出发前选好便携日志簿，核对呼号、网格、激活编号、电台、天线和功率。测试完整台站，并联网运行 CQOps 更新缓存的参考编号和呼号前缀。确认 **F6** 能找到所需编号，然后导出备份。

没有互联网仍可本地记录。`cqops --offline` 跳过网络功能，请勿依赖实时查询或同步。局域网设备也应在出发前用计划采用的启动模式测试。缓存资料可能过时。

结束后检查 QSO 数量和参考编号，导出 ADIF 并备份；如使用 Wavelog，再上传待发送通联。确认各奖项计划要求的提交格式，必要时转换导出文件。

<a id="contests"></a>

## 竞赛

CQOps 提供休闲竞赛记录、交换信息、流水号和 QSO 速率，但并非完整的计分或提交系统。复杂竞赛操作应使用专用软件。

在 **F9 → Contests** 按 **Ins**，设置名称、日期、ADIF 竞赛 ID、起始流水号及收发交换模板。

| 标记 | 内容 |
|---|---|
| `@rst` | 发送或接收的信号报告 |
| `@serial` | 流水号 |
| `@cqz` / `@mycqz` | 对方／自己的 CQ 分区 |
| `@itu` / `@myitu` | 对方／自己的 ITU 分区 |
| `@grid` / `@mygrid` | 对方／自己的网格 |

在 **F1** 用 **Ctrl+C** 切换竞赛。发射前核对交换信息和下一个流水号。状态栏显示数量、下个编号和时间；较宽窗口可显示更多速率统计。结束后切回无竞赛状态。

导出时打开 **F8**，用 **Ctrl+C** 选择竞赛筛选，再按 **Ctrl+E** 核对范围。输出为 ADIF，不是 Cabrillo。请遵守主办方的格式和提交规则。

<a id="dashboard"></a>

## CQOps Live

启用 **F9 → Integrations → HTTP Server**，按 **Ctrl+S** 保存。在 CQOps 电脑打开 `http://localhost:8073`。

默认地址 `0.0.0.0` 允许局域网访问，仍受防火墙限制。其他设备应使用 CQOps 电脑的 IP 和端口 `8073`。设置为 `127.0.0.1` 可限制为本机访问。仅在可信网络使用，不要将端口映射到互联网。

仪表板自动更新当前通联、QSO 地图、最近通联、速率、操作员、APRS 及可用的传播／天气信息。依赖互联网的图层可能离线不可用。Header 1、Header 2、Logo URL 和 Event Start 可定制活动展示；开始日期会筛选显示的统计和 QSO 列表。

<a id="keys"></a>

## 快捷键参考

| 页面 | 按键 | 操作 |
|---|---|---|
| 通用 | ? / Esc / F10 | 帮助／返回／退出 |
| QSO | Tab / Shift+Tab | 下一／上一字段 |
| QSO | Enter / Ins | 保存／查询 |
| QSO | Shift+Backspace / Del | 清空字段／表单 |
| QSO | Ctrl+L / Ctrl+O / Ctrl+R / Ctrl+C | 切换日志簿／操作员／电台／竞赛 |
| 日志簿 | ↑ / ↓, PgUp / PgDn, Home / End | 选择、翻页、首行／末行 |
| 日志簿 | Enter 或 e / Delete | 编辑／删除选中 QSO |
| 日志簿 | Ctrl+I / Ctrl+E | 导入／导出 ADIF |
| 日志簿 | Ctrl+W / Alt+W | 上传／下载 Wavelog |
| 日志簿 | Ctrl+C / Backspace | 竞赛筛选／清除搜索 |

快捷键依页面而定：**Ctrl+C 不是退出命令**。笔记本的功能键可能需要同时按 **Fn**。如果终端拦截快捷键，请检查终端键盘设置和 CQOps 帮助栏。

<a id="help"></a>

## 故障排查与帮助

| 问题 | 首先检查 |
|---|---|
| 无法启动或显示不完整 | 终端尺寸，Windows 使用 Windows Terminal，尝试 `cqops --offline` |
| 电台未连接 | 当前配置、flrig/rigctld 是否运行、型号、串口、速率、主机／端口、是否被其他程序占用 |
| 缺少 WSJT-X 通联 | UDP 设置、WSJT 指示、是否已在 WSJT-X 实际保存完成的 QSO、当前日志簿 |
| Wavelog 错误 | URL、`wl2_` 令牌、台站配置、互联网；本地 QSO 仍会保留 |
| GPS 无位置 | 串口／速率或 GPSD 地址、天空视野、有效定位、Grid from GPS |
| APRS 不发信标 | 日志簿、APRS TX 和 Send beacons、呼号／SSID、TNC／电台或网络 |
| 仪表板无法访问 | 服务是否启用、IP／端口、防火墙；localhost 指浏览器所在设备 |

设置问题请先通过 **F9** 检查，再考虑改动文件。如果提示凭据存储异常，或更换了电脑，请重新输入凭据。

必要时启用 **F9 → General → Debug**，安全地复现问题并收集相关诊断日志，完成后关闭调试。

| 系统 | 诊断日志 |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

通过 [GitHub Issues](https://github.com/szporwolik/cqops/issues) 报告问题，附上 CQOps 版本、系统、终端、复现步骤及相关日志。分享前删除密码、API 令牌和私人信息。
