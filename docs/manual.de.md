---
title: CQOps-Benutzerhandbuch
description: Anleitung zur Installation, Konfiguration und Verwendung von CQOps — einem schnellen, terminalorientierten Amateurfunk-Logger
---

# CQOps-Benutzerhandbuch

CQOps ist ein schneller, terminalorientierter Amateurfunk-Logger für Funkamateure, die zuverlässig per Tastatur loggen möchten und dabei nur geringe Systemressourcen benötigen. Er ist für den Einsatz im Shack, im Portabelbetrieb, an Clubstationen, bei Field Days sowie auf Geräten der Raspberry-Pi-Klasse oder älteren Laptops ausgelegt.

CQOps speichert QSOs immer zuerst lokal. Internetbasierte Integrationen sind optional.

## Inhalt

1. [Was CQOps ist](#what-cqops-is)
2. [Download und Installation](#download-and-installation)
3. [Erster Start](#first-launch)
4. [Das erste QSO loggen](#log-your-first-qso)
5. [Hauptbildschirm](#main-screen)
6. [Häufige Arbeitsabläufe](#common-workflows)
7. [QSO-Logging](#qso-logging)
8. [Logbuch-Editor und ADIF](#logbook-editor-and-adif)
9. [Contests](#contests)
    - [Einen Contest einrichten](#setting-up-a-contest)
    - [Untere Statusleiste](#bottom-status-bar)
    - [Contest-Statistikpanel](#contest-statistics-panel)
    - [Contest-ADIF-Export](#contest-adif-export)
    - [Verhalten im Contest-Modus](#contest-mode-behavior)
10. [Favoriten, Referenzen und Bandpläne](#favorites-references-and-band-plans)
11. [Integrationen](#integrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Konfiguration](#configuration)
14. [Tastenkürzel](#keyboard-shortcuts)
15. [Fehlerbehebung](#troubleshooting)
16. [Fehler melden](#reporting-bugs)

---

<a id="what-cqops-is"></a>
## Was CQOps ist

CQOps ist auf schnelle QSO-Eingabe, lokale Datenspeicherung und praktischen Feldbetrieb ausgerichtet.

### Grundideen

- **Terminalorientierte Bedienung** — für die Nutzung mit der Tastatur optimiert.
- **Offline-first-Logging** — lokales QSO-Logging funktioniert ohne Internetzugang. Enthält eine eingebettete Weltkarte für das Dashboard, die vollständig offline funktioniert.
- **Geringer Ressourcenbedarf** — geeignet für Systeme der Raspberry-Pi-Klasse, ältere Laptops und gemeinsam genutzte Stations-PCs.
- **Portables Design** — wird als einzelne Go-Binärdatei verteilt.
- **Mehrere Logbücher** — nützlich für persönliche, portable, Contest- und Club-Logs.
- **Mehrere Operatoren** — geeignet für Hot-Seat-Betrieb und gemeinsam genutzte Clubstationen.
- **Mehrere Funkgeräte** — jedes Rig-Preset kann eigene Backend- und WSJT-X-Einstellungen speichern.
- **Optionale Integrationen** — Multi-provider Callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info), Wavelog, DX Cluster, PSK Reporter, GPS, APRS, Rig-Steuerung, Rotorsteuerung, Solardaten und das browserbasierte CQOps Live dashboard.

Lokales Logging benötigt keinen Internetzugang. Netzwerkfunktionen werden im Modus `--offline` übersprungen.

### Für wen CQOps gedacht ist

CQOps eignet sich besonders für:

- Portabeloperatoren,
- SOTA- und POTA-Aktivierer,
- Clubstationen,
- Field-Day-Teams,
- Operatoren, die einen Terminal-Workflow bevorzugen,
- Stationen, die schnell zwischen Operatoren, Logbüchern oder Funkgeräten wechseln müssen.

CQOps soll nicht jede Funktion eines vollständigen Desktop-Loggers oder einer webbasierten Logbuchplattform ersetzen. Der Schwerpunkt liegt auf schnellem Terminal-Logging, Feldbetrieb, Offline-Nutzung und gemeinsam genutzten Stationen.

### Club- und Mehrbenutzerbetrieb

CQOps wurde mit Blick auf Amateurfunkclubs entwickelt. Der aktive Operator ist immer in der Statusleiste sichtbar — **ein Blick** genügt, um zu sehen, wer gerade angemeldet ist. Der Operatorwechsel erfolgt mit einer einzigen Tastenkombination (`Ctrl+O`) und wird sofort wirksam; Rufzeichen und Name des Operators werden in jedes nachfolgende QSO geschrieben. Kein Abmelden, keine Passwortabfrage, keine Unterbrechung.

Logbücher, Rig-Presets und Contests werden auf dieselbe Weise gewechselt — `Ctrl+L`, `Ctrl+R`, `Ctrl+C`. Eine Clubstation mit wechselnden Operatoren, mehreren Funkgeräten und mehreren aktiven Contests kann den gesamten Kontext in weniger als einer Sekunde wechseln, ohne die Maus zu berühren.

Bei Field Days und öffentlichen Veranstaltungen kann das **CQOps Live dashboard** eine Echtzeitkarte, den QSO-Feed und Statistiken auf einem großen Bildschirm anzeigen. Besucher und Clubmitglieder können den Stationsbetrieb verfolgen, ohne sich um das Terminal des Operators zu drängen. Aktivieren Sie einfach die Integration **HTTP Server** und öffnen Sie die Seite auf einem beliebigen Gerät mit Webbrowser.

---

<a id="download-and-installation"></a>
## Download und Installation

Alle Releases anzeigen:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Paket | Link | Hinweise |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Für die meisten Benutzer empfohlen. Fügt CQOps dem Startmenü und PATH hinzu. |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Entpacken und ohne Installation starten. |

### Linux — Debian / Ubuntu

| Architektur | Link | Geeignet für |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Die meisten Intel-/AMD-PCs |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-Bit-ARM-Systeme |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-Bit-Raspberry-Pi-OS |

Installieren Sie das heruntergeladene Paket:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — portables Tarball

| Architektur | Link | Geeignet für |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Die meisten Intel-/AMD-PCs |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-Bit-ARM-Systeme |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-Bit-Raspberry-Pi-OS |

### macOS

| Architektur | Link | Geeignet für |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | M1/M2/M3-Macs |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Intel-Macs |

Manuelle Installation:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Aus dem Quellcode bauen

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

Zum Bauen aus dem Quellcode ist Go 1.26 oder neuer erforderlich.

### Terminalanforderungen

| Anforderung | Wert |
|---|---|
| Minimale Terminalgröße | 80×24 Zeichen |
| Empfohlene Terminalgröße | 80×43 Zeichen oder größer |
| Empfohlenes Windows-Terminal | Windows Terminal |
| Terminal mit Kitty graphics | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) oder [WezTerm](https://wezfurlong.org/wezterm/) |

### Grundlegende Befehle

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

<a id="first-launch"></a>
## Erster Start

Beim ersten Start öffnet CQOps den Einrichtungsassistenten. Für lokales Logging sind nur die wesentlichen Stationsdaten erforderlich. Netzwerkintegrationen können übersprungen und später konfiguriert werden.

### Seiten des Assistenten

| Page | Konfiguriert |
|---|---|
| Station & Logbook | Erstes Logbuch, Stationsrufzeichen, Operator, Grid locator, optionale Referenzen und Zonen sowie Wavelog URL/API/station profile ID |
| Rig | Rig-Preset, Modell, Antenne, Leistung, Backend, optionaler Rotor und optionale WSJT-X-UDP-Einstellungen |
| Integrations | Callbook-Lookup-Einstellungen (QRZ.com, HamQTH, QRZ.RU, Callook.info) |
| General | IANA-Zeitzone |
| Summary | Einstellungen prüfen und speichern |

Unterstützte Rig-Backends:

- None,
- flrig,
- Hamlib `rigctld`.

### Navigation im Assistenten

| Key | Action |
|---|---|
| Ctrl+S | Validieren und fortfahren; auf **Summary** speichern und CQOps starten |
| Esc | Zurück |
| F10 | Beenden |
| Tab / Shift+Tab | Zwischen Feldern wechseln |
| Space | Kontrollkästchen umschalten |

Die Einstellungen des Assistenten können später mit **F9** geändert werden.

---

<a id="log-your-first-qso"></a>
## Das erste QSO loggen

1. Starten Sie CQOps:

   ```bash
   cqops
   ```

2. Schließen Sie den Einrichtungsassistenten mindestens mit Ihrem Rufzeichen und Grid locator ab.

3. Öffnen Sie **QSO form** mit **F1**.

4. Geben Sie das Rufzeichen der Gegenstation ein. CQOps wandelt Rufzeichen automatisch in Großbuchstaben um.

5. Füllen Sie die übrigen Felder aus. Wenn das aktive Funkgerät über flrig oder Hamlib verbunden ist, kann CQOps frequency, band, mode und submode automatisch eintragen.

6. Drücken Sie **Enter**, um zu speichern.

7. Wenn eine Warnung **DUPE!** erscheint, drücken Sie erneut **Enter**, um trotzdem zu speichern, oder **Esc**, um abzubrechen.

Das gespeicherte QSO erscheint sofort in der Tabelle **Recent QSOs**.

---

<a id="main-screen"></a>
## Hauptbildschirm

CQOps verwendet ein festes Terminallayout:

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

Die **Status bar** zeigt:

- CQOps-Version,
- aktives Logbuch,
- aktives Rig,
- Stationsrufzeichen,
- aktiven Operator,
- Statusbezeichnungen der Integrationen,
- lokale Zeit mit `L`,
- UTC-Zeit mit `Z`.

Häufige Bezeichnungen sind **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** und **GPS**. Die Bezeichnung **GPS** verwendet dieselbe Farbkonvention: Rot bei getrennter Verbindung, Gelb bei Verbindung ohne Fix und Weiß bei vorhandenem Positionsfix.

| Farbe | Bedeutung |
|---|---|
| Weiß/Standard | Verbunden oder aktiv |
| Gelb | Deaktiviert, Verbindung wird hergestellt oder erwartungsgemäß offline |
| Rot | Fehler oder getrennt |
| Akzentfarbe + fett | WSJT-X sendet |

### Haupt-Tabs

| Key | Tab | Screen |
|---|---|---|
| F1 | QSO | **QSO form** und **Recent QSOs** |
| F2 | QRZ | **Partner view**: Callbook-Daten, Karte, Statistiken, Foto |
| F4 | DXC | **DX Cluster**-Spots und Filter |
| F5 | HRD | **PSK Reporter**-Spots und Ausbreitungskarte |
| F6 | REF | Suche nach SOTA/POTA/WWFF/IOTA-Referenzen |
| F7 | BPL | **Band Plan Browser** |
| F8 | LOG | **Logbook Editor**, ADIF, Wavelog-Synchronisierung |
| F9 | CFG | Konfigurationsmenüs |

Die **Help bar** zeigt die für den aktiven Bildschirm relevanten Tastenkürzel. Drücken Sie **?**, um das vollständige **Help overlay** zu öffnen.

---

<a id="common-workflows"></a>
## Häufige Arbeitsabläufe

### Portabel-, SOTA- oder POTA-Betrieb

Vor dem Aufbruch:

1. Starten Sie CQOps einmal mit Internetzugang.
2. Lassen Sie CQOps zwischengespeicherte Daten wie Solardaten, REF-Daten und DXCC-Präfixe herunterladen oder aktualisieren.
3. Prüfen Sie, ob das Panel **Solar** Daten anzeigt.
4. Prüfen Sie, ob die **REF**-Suche unter **F6** Ergebnisse liefert.

Im Feld:

1. Starten Sie CQOps im Offline-Modus:

   ```bash
   cqops --offline
   ```

2. Loggen Sie wie gewohnt. QSOs werden lokal gespeichert.
3. Sobald wieder eine Internetverbindung besteht, öffnen Sie **F8** und drücken **w**, um nicht gesendete QSOs zu Wavelog hochzuladen.

### Gemeinsam genutzte Clubstation und Hot-Seat-Logging

1. Öffnen Sie **F9 → Operators**.
2. Drücken Sie **Ins**, um Operatorprofile hinzuzufügen.
3. Drücken Sie in **QSO form** **Ctrl+O**, um den aktiven Operator zu wechseln.
4. Prüfen Sie vor dem Speichern den aktiven Operator in der Statusleiste.
5. Verwenden Sie **Retain**, wenn mehrere Operatoren ähnliche Verbindungen loggen und nicht das gesamte Formular erneut eingeben sollen.

Der aktive Operator wird im ADIF-Feld `OPERATOR` gespeichert.

### Persönliche und Club-Logbücher

1. Öffnen Sie **F9 → Logbooks**.
2. Drücken Sie **Ins**, um jedes Logbuch anzulegen.
3. Drücken Sie in **QSO form** **Ctrl+L**, um das aktive Logbuch zu wechseln.
4. Prüfen Sie vor dem Speichern das aktive Logbuch in der Statusleiste.

Jedes Logbuch kann eigene Stationsdaten, Wavelog-Einstellungen, Contest-Einstellungen und Operatoren besitzen.

### Mehrere Funkgeräte

1. Öffnen Sie **F9 → Rigs**.
2. Drücken Sie **Ins**, um Rig-Presets anzulegen.
3. Wählen Sie das Backend: None, flrig oder Hamlib.
4. Drücken Sie in **QSO form** **Ctrl+R**, um das aktive Rig zu wechseln.

Ein Rig-Preset kann Backend, Modell, Antenne, Leistung, Rotoreinstellungen und WSJT-X-UDP-Einstellungen enthalten.

### Digitalbetrieb mit WSJT-X

Wenn die WSJT-X-UDP-Integration aktiviert ist, kann CQOps ADIF-Nachrichten von WSJT-X empfangen und abgeschlossene digitale QSOs automatisch loggen.

Automatisch geloggte QSOs:

- werden im aktiven Logbuch gespeichert,
- erscheinen sofort in **Recent QSOs**,
- überspringen Duplikate,
- übernehmen die aktive contest ID,
- können automatisch zu Wavelog hochgeladen werden, wenn Wavelog konfiguriert und erreichbar ist.

Wenn der von WSJT-X gemeldete Operator nicht mit dem aktiven Operator in CQOps übereinstimmt, zeigt CQOps eine Warnung an.

Prüfen Sie vor längeren Digitalsitzungen:

- aktives Logbuch,
- aktiven Operator,
- aktiven Contest,
- Statusbezeichnung **WSJT**.

### Wavelog-Synchronisierung

CQOps speichert QSOs immer zuerst lokal. Die Wavelog-Synchronisierung ist optional.

| Action | Where | Shortcut | Notes |
|---|---|---|---|
| Upload unsent QSOs | **Logbook Editor** | `w` | Upload in Gruppen von 50 |
| Download from Wavelog | **Logbook Editor** | `Ctrl+W` | Inkrementeller Download mit `last_fetched_id` |

Der Uploadstatus wird pro QSO verfolgt:

- not sent,
- sent,
- error.

Schlägt der Upload fehl, bleibt das QSO im lokalen Logbuch und kann später erneut gesendet werden. Das Purge eines Logbuchs setzt die fetch ID auf `0` zurück und ermöglicht dadurch einen vollständigen erneuten Download.

---

<a id="qso-logging"></a>
## QSO-Logging

**QSO form** ist der zentrale Logging-Bildschirm. Öffnen Sie ihn mit **F1**.

CQOps kann Felder aus folgenden Quellen ausfüllen:

| Quelle | Felder |
|---|---|
| flrig / Hamlib | Frequency, Freq RX bei Split, Mode, Submode |
| Callbook (QRZ.com / HamQTH / QRZ.RU / Callook.info) | Name, QTH, Grid, Country, CQ zone, ITU zone, DXCC, Continent, Foto |
| REF-Datenbank | SOTA-, POTA-, WWFF- und IOTA-Referenzen |
| Wavelog lookup | Worked/confirmed-Status, wenn konfiguriert |
| DXCC-/Präfixdaten | Präfix- und Länderinformationen |

### Formularlayout

| Linke Spalte | Mittlere Spalte | Rechte Spalte |
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

Die Felder **Exch sent** und **Exch rcvd** erscheinen nur, wenn ein Contest aktiv ist.

Die untere Zeile enthält:

- **Comment**,
- **Keep** — behält das Feld **Comment** zwischen QSOs bei,
- **Retain** — behält nach dem Speichern das gesamte Formular bei.

Felder wie **Band**, **Mode** und **Submode** können mit **PgUp/PgDn** durchgeschaltet werden.

### Pfad, Peilung und Kennzeichnungen

Wenn beide Grid locator bekannt sind, zeigt CQOps Entfernung und Azimut an.

**QSO form** kann außerdem Kennzeichnungen wie diese anzeigen:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Speichern

| Key | Action |
|---|---|
| Enter | QSO speichern |
| Ctrl+S | DX-Spot aus dem ausgefüllten Formular senden |
| Esc | Duplikatbestätigung abbrechen |
| Enter bei DUPE-Bestätigung | Duplikat trotzdem speichern |

---

<a id="logbook-editor-and-adif"></a>
## Logbuch-Editor und ADIF

Öffnen Sie **Logbook Editor** mit **F8**.

Er dient zum:

- Prüfen von QSOs,
- direkten Bearbeiten,
- Löschen von QSOs,
- ADIF-Import,
- ADIF-Export,
- Wavelog-Upload,
- Wavelog-Download,
- Ausführen contestbezogener Vorgänge.

### QSOs bearbeiten

1. Wählen Sie mit **↑/↓** eine Zeile aus.
2. Drücken Sie **Enter** oder **e**.
3. Bearbeiten Sie das QSO.
4. Speichern Sie mit **Ctrl+S**.

Änderungen erscheinen sofort in **Recent QSOs**.

### ADIF-Import und -Export

CQOps unterstützt den Import und Export von ADIF 3.1.7.

| Action | Shortcut |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

Beim Import werden Datensätze validiert, Duplikate übersprungen und eine Zusammenfassung angezeigt. Importierte QSOs werden für den Wavelog-Upload markiert, wenn die Wavelog-Synchronisierung konfiguriert ist.

Der Export kann alle QSOs oder nach Contest gefilterte QSOs enthalten. `CONTEST_ID` bleibt erhalten.

### Behandlung digitaler Betriebsarten

Die Behandlung von mode und submode entspricht ADIF 3.1.7, wie in diesem Handbuch beschrieben:

- FT8 wird als eigenständiger mode exportiert.
- FT4 und FT2 werden als MFSK mit passendem submode exportiert.
- Importierte ältere MFSK-+‑FT8-Datensätze werden zu eigenständigem FT8 normalisiert.

**QSO form** besitzt getrennte Felder **Mode** und **Submode**. Beide können mit **PgUp/PgDn** durchgeschaltet werden.

---

<a id="contests"></a>
## Contests

CQOps enthält ein leichtgewichtiges Contest-Logging-Panel für die **gelegentliche Teilnahme an Contests**. Es ersetzt keine spezialisierten Contest-Logger wie N1MM, Win-Test oder TR4W. Für ernsthaften Multi-Op-, Multi-Radio- oder Assisted-Betrieb sollten Sie einen dafür vorgesehenen Contest-Logger einsetzen. CQOps eignet sich, wenn Sie ein paar Punkte verteilen, Ihre Rate aus Spaß verfolgen oder während einer SOTA-/POTA-Aktivierung einige Contest-QSOs loggen möchten, ohne den gewohnten Logger zu verlassen.

<a id="setting-up-a-contest"></a>
### Einen Contest einrichten

Erstellen oder konfigurieren Sie einen Contest in **Logbook Editor** mit **Ins**.

Die Contest-Konfiguration umfasst:

- Contest-Name,
- Datum,
- ADIF contest ID,
- Austauschvorlagen.

#### Vorlagenmarker

| Marker | Replaced with |
|---|---|
| `@rst` | Gesendeter oder empfangener RST |
| `@serial` | Automatisch hochzählende Seriennummer |
| `@cqz` | CQ-Zone der DX-Station |
| `@mycqz` | Ihre CQ-Zone |
| `@itu` | ITU-Zone der DX-Station |
| `@myitu` | Ihre ITU-Zone |
| `@grid` | Grid der DX-Station |
| `@mygrid` | Ihr Grid |

Drücken Sie **Ctrl+C**, um den aktiven Contest durchzuschalten, oder wählen Sie ihn im Menü **Contest** (**F7**) aus. Die Austauschfelder erscheinen automatisch in **QSO form**, und Seriennummern werden automatisch erhöht.

<a id="bottom-status-bar"></a>
### Untere Statusleiste

Wenn ein Contest aktiv ist, zeigt die untere Leiste eine Live-Zusammenfassung:

```
 IARU-HF · IARU HF   45 QSOs   Started 16:13   Last 14:04 ago   Next #45   On 2:41
```

| Feld | Bedeutung |
|-------|---------|
| `IARU-HF` | ADIF ID des Contests, also die maschinenlesbare Kennung |
| `· IARU HF` | Anzeigename des Contests — sichtbar, wenn er sich von der ID unterscheidet |
| `45 QSOs` | Gesamtzahl der in dieser Contest-Sitzung geloggten QSOs |
| `Started 16:13` | Uhrzeit des ersten Contest-QSOs des Tages |
| `Last 14:04 ago` | Zeit seit dem letzten Contest-QSO |
| `Next #45` | Seriennummer, die beim nächsten QSO gesendet wird |
| `On 2:41` | Gesamte Betriebszeit — Summe der Abstände zwischen QSOs, die kürzer als 30 Minuten sind |

Das Feld `Started` wird bei Terminals mit weniger als 120 Spalten ausgeblendet. Contest-Name und `On`-Zeit werden unter 100 Spalten ausgeblendet.

<a id="contest-statistics-panel"></a>
### Contest-Statistikpanel

Wenn ein Contest aktiv und das Terminal breit genug ist, erscheint rechts neben **QSO form** ein kompaktes Statistikpanel mit gelbem Rahmen:

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

| Zeile | Feld | Bedeutung |
|-----|-------|---------|
| **Rate** | `2/h` | Rate über die letzten **10 QSOs** — kurzfristiges Serientempo |
| | `--/h` | Rate über die letzten **100 QSOs** — zeigt `--`, bis 100 QSOs geloggt wurden |
| **Count** | `60m 0` | In den letzten 60 Minuten geloggte QSOs |
| | `hr 0` | In der aktuellen vollen Stunde seit `:00` geloggte QSOs |
| **Peak** | `1m120` | Beste Ein-Minuten-Rate: 120/h entspricht 2 QSOs in dieser Minute |
| | `10m 54` | Bestes gleitendes 10-Minuten-Fenster: durchschnittlich 54/h |
| | `60m 29` | Bestes gleitendes 60-Minuten-Fenster: durchschnittlich 29/h |
| **Avg** | `8/h` | Sitzungsdurchschnitt — Gesamtzahl der QSOs geteilt durch die Stunden seit dem ersten QSO |
| | `Sess 5:36` | Gesamte Sitzungsdauer vom ersten bis zum letzten QSO, als H:MM oder nur Minuten |
| **Chart** | `max 1` | In der aktivsten Minute wurde 1 QSO geloggt. Balken zeigen QSOs pro Minute |
| | `-60m…now` | Linker Rand = vor 60 Minuten, rechter Rand = jetzt |

Das Diagramm verwendet Unicode-Blockzeichen (`█`), die auf vier Zeilen vertikaler Balken skaliert sind. Peak-Raten lassen das Suffix `/h` weg, da **Peak** bereits „pro Stunde“ impliziert. Bei Zeitangaben werden Sekunden weggelassen, weil sie bei minütlicher Aktualisierung nur unnötige Unruhe erzeugen würden.

<a id="contest-adif-export"></a>
### Contest-ADIF-Export

Um ein Contest-Log einzureichen, öffnen Sie bei aktivem Contest **Logbook Editor** (`Ctrl+E`). Wenn ein Contest-Filter aktiv ist, bietet der ADIF-Exportdialog an, **nur die QSOs des aktiven Contests** zu exportieren. Dadurch entsteht eine standardkonforme ADIF-3.1.7-Datei mit Austauschfeldern, Seriennummern und erhaltener Contest-ADIF-ID, die direkt zum Robot des Veranstalters oder zu einem Logprüfsystem hochgeladen werden kann.

<a id="contest-mode-behavior"></a>
### Verhalten im Contest-Modus

Wenn ein Contest aktiv ist:

- zeigt **QSO form** Austauschfelder,
- werden Seriennummern automatisch erhöht,
- kann **Recent QSOs** auf Contest-QSOs gefiltert werden,
- bleibt `CONTEST_ID` beim ADIF-Export erhalten,
- erhalten **QSO form**, Contest-Panel und **Solar**-Panel zur deutlichen Kennzeichnung einen gelben Rahmen,
- werden DXC-Spots auf Duplikate gegenüber allen Contest-QSOs geprüft, nicht nur gegenüber den heutigen.

---

<a id="favorites-references-and-band-plans"></a>
## Favoriten, Referenzen und Bandpläne

### Favorites

**Favorites** speichert Presets für frequency, mode und band in drei Slots — ausreichend für die am häufigsten verwendeten Anruffrequenzen. Die Tastenkürzel verwenden `Alt`, um Konflikte mit üblichen Terminal-Bearbeitungstasten zu vermeiden und terminalübergreifend zuverlässig zu funktionieren.

| Shortcut | Action |
|---|---|
| Alt+Ins / Alt+Home / Alt+PgUp | Favorite aus Slot 1, 2 oder 3 abrufen |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Aktuelle frequency, mode und band in Slot 1, 2 oder 3 speichern |

**Favorites** werden in der Konfiguration gespeichert und von allen Logbüchern gemeinsam genutzt.

Beispiel:

1. Geben Sie `145.55` ein.
2. Setzen Sie **Mode** auf `FM`.
3. Setzen Sie **Band** auf `2m`.
4. Drücken Sie **Alt+Shift+Ins**, um das Preset in Slot 1 zu speichern.
5. Drücken Sie später **Alt+Ins**, um das Preset abzurufen.

### REF Lookup

Öffnen Sie **REF Lookup** mit **F6**.

Die Suche umfasst:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Sie können nach Präfix, Name oder Referenzkennung suchen. Ausgewählte Referenzen können **QSO form** ausfüllen.

### Band Plan Browser

Öffnen Sie **Band Plan Browser** mit **F7**.

Er bietet schnellen Zugriff auf:

- Amateur bands,
- VHF/UHF ranges,
- CB,
- PMR446,
- Broadcast presets,
- Portable — häufige Frequenzen für Portabel- und Feldbetrieb, darunter SOTA, POTA und Anrufkanäle.

Eine ausgewählte Frequenz kann zum Abstimmen des aktiven Rigs verwendet werden. Bandplandaten lassen sich außerdem als Markdown exportieren.

---

<a id="integrations"></a>
## Integrationen

Alle Integrationen sind optional. Lokales Logging funktioniert ohne sie.

### Callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info)

CQOps unterstützt mehrere Callbook-Anbieter mit prioritätsbasierter Kaskadierung.
Wenn Sie im QSO-Formular **Ins** drücken, werden die Anbieter der Reihe nach
abgefragt, bis einer ein Ergebnis liefert:

1. **QRZ.com** — benötigt Internet und ein QRZ-XML-Abonnement. Umfassendste Daten.
2. **HamQTH** — kostenloser globaler Dienst. Gute Abdeckung, erfordert ein kostenloses Konto.
3. **QRZ.RU** — kostenloser Dienst mit Fokus auf Russland und Nachbarländer. Erfordert API-Login. Liefert Name, QTH, Grid, Lat/Lon, Klasse, LoTW/eQSL und Foto.
4. **Callook.info** — kostenloser US-fokussierter Dienst. Kein Konto erforderlich, schnelle FCC-Abfragen.

Wenn ein Anbieter mit höherer Priorität ausfällt oder deaktiviert ist, wird der
nächste versucht. Wenn **Base call fallback** aktiviert ist (Standard: ein),
versucht CQOps auch das Basisrufzeichen (ohne Präfix oder Suffix), falls das
vollständige Rufzeichen kein Ergebnis liefert.

Aktivieren und konfigurieren Sie die Anbieter unter **F9 → Callbook**.

Drücken Sie in **QSO form** **Ins**, um Callbook-Felder wie diese auszufüllen:

- Name,
- QTH,
- Grid,
- Country,
- CQ/ITU zones,
- DXCC,
- Continent.

**Partner view** unter **F2** kann das Foto des Operators anzeigen, sofern verfügbar.

> ⚠️ **Experimental.** Die Fotoanzeige kann das Kitty-terminal-graphics-
> Protokoll verwenden und erfordert ein kompatibles Terminal: Kitty, Ghostty
> oder WezTerm. Aktivieren Sie dies unter **F9 → General → Kitty Graphics**.
> Standardterminals und SSH-Sitzungen ohne Grafikweiterleitung verwenden als
> Fallback ein aus Glyphen erzeugtes Bild.

### Wavelog

Die Wavelog-Integration unterstützt:

- Upload,
- inkrementellen Download,
- Worked/confirmed-Lookup.

Wavelog wird pro aktivem Logbuch konfiguriert mit:

- URL,
- API key,
- station profile ID.

CQOps speichert QSOs immer zuerst lokal. Ein Fehler beim Wavelog-Upload löscht keine lokalen Daten.

### flrig

Die flrig-Integration verwendet XML-RPC über HTTP.

Standard-Endpoint:

```text
localhost:12345
```

CQOps kann folgende Werte lesen:

- frequency,
- mode,
- power.

Bei Split-Betrieb wird VFO A **Frequency** und VFO B **Freq RX** zugeordnet.

### Hamlib / rigctld

Die Hamlib-Rig-Steuerung verwendet den TCP-Daemon `rigctld`.

Je nach Funkgerät und Backend-Unterstützung kann CQOps Folgendes abfragen:

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps behandelt fehlende Unterstützung für VFO-Namen nach Möglichkeit ohne Fehler.

### Hamlib Rotator / rotctld

> ⚠️ **Experimental.** Die Rotorsteuerung ist experimentell. Prüfen Sie vor
> dem Betrieb immer die mechanischen Grenzen Ihrer Antenne. Halten Sie sich
> bereit, die Bewegung sofort mit **Alt+/** zu stoppen. Verwenden Sie die
> Funktion vorsichtig — eine falsche Konfiguration kann Rotor oder Antenne
> beschädigen.

Die Rotorsteuerung verwendet Hamlib `rotctld`.

CQOps unterstützt:

- azimuth,
- elevation,
- stop commands.

| Shortcut | Action |
|---|---|
| Alt+, | Azimuth um −5° ändern |
| Alt+. | Azimuth um +5° ändern |
| Alt+; | Elevation um +5° ändern |
| Alt+' | Elevation um −5° ändern |
| Alt+\ | Rotor auf die berechnete Pfadpeilung ausrichten |
| Alt+/ | Rotor stoppen |

### WSJT-X

Die WSJT-X-Integration verwendet UDP-Nachrichten von WSJT-X. CQOps verarbeitet ADIF-Nachrichten und kann abgeschlossene QSOs automatisch loggen.

Die Rig-Bezeichnung erhält die Akzentfarbe, während WSJT-X sendet. Wenn der von WSJT-X gemeldete Operator nicht dem aktiven Operator entspricht, zeigt CQOps eine Warnung an.

### GPS

CQOps kann eine Position von einem GPS-Empfänger lesen und als Grid locator der Station verwenden — ideal für Portabel-, Mobil- und Feldbetrieb.

Zwei Backends werden unterstützt:

- **Serial** — direkte Verbindung zu einem GPS-Empfänger über eine serielle Schnittstelle, etwa USB-to-serial, einen integrierten COM-Port oder `/dev/ttyUSB0`.
- **GPSD** — Verbindung zu einem [gpsd](https://gpsd.io/)-Server über TCP, standardmäßig `127.0.0.1:2947`. Nützlich, wenn das GPS von mehreren Anwendungen gemeinsam verwendet oder über das Netzwerk erreicht wird.

Die **GPS**-Statusanzeige in der Statusleiste zeigt:

| Farbe | Bedeutung |
|--------|---------|
| Rot `GPS` | Getrennt / Fehler |
| Gelb `GPS` | Verbunden, noch kein Fix |
| Weiß `GPS` | Fix vorhanden, Position bestimmt |

Sobald ein Fix vorliegt, wird der Grid locator der Station durch die GPS-basierte Position ersetzt und in der Statuszeile mit `(GPS)` markiert:

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Aktivieren Sie **Grid from GPS** in den Einstellungen **Station & Logbook**, um das GPS-Grid für QSO-Logging, APRS-Beacons, die Dashboard-Karte und Entfernungsberechnungen zu verwenden.

**Grid precision** wird im Menü **Integration** mit 10, 8 oder 6 Zeichen konfiguriert. Standard sind 10 Zeichen, entsprechend ungefähr 25 m Genauigkeit. Intern wird das Grid immer mit voller Genauigkeit berechnet und erst an der Verwendungsstelle gekürzt.

### DX Cluster

Die Integration **DX Cluster** verwendet Telnet und benötigt Internetzugang.

Standardserver:

```text
dxspots.com:7300
```

Filter umfassen:

- band,
- spotter continent,
- mode,
- age/time.

| Key | Action |
|---|---|
| Enter | **QSO form** ausfüllen, Rig abstimmen und zu **QSO** zurückkehren |
| Space | Rig abstimmen und in **DX Cluster** bleiben |
| Backspace | Alle Filter löschen |

Wenn **DX Cluster** verbunden ist, erhält **QSO form** zwei zusätzliche Funktionen:

- **Send a spot** — drücken Sie bei ausgefülltem Formular **Ctrl+S**, um den Spotdialog zu öffnen und einen DX-Spot an den Cluster zu senden.
- **Nearest spots** — wenn eine Frequenz eingestellt ist, erscheinen bis zu drei nahe Spots direkt in **QSO form**, sodass Sie die Bandbelegung sehen können, ohne den Logging-Bildschirm zu verlassen. Drücken Sie **Ctrl+P**, um das Rufzeichen aus dem nächstgelegenen Spot zu übernehmen.

### PSK Reporter

Die Integration **PSK Reporter** benötigt Internetzugang. Sie eignet sich hervorragend, um reale Ausbreitungsbedingungen schnell zu prüfen: Wer empfängt Ihr Signal oder wen empfangen Sie gerade auf einem bestimmten Band?

Sie bietet:

- Ausbreitungsspots,
- band/time/mode-Filter,
- ASCII-Weltkarte unter **F5**.

### APRS

CQOps unterstützt drei APRS-Diensttypen. Wählen Sie den Typ, der zu Ihrer Stationskonfiguration passt:

| Service | Connection | Internet required |
|---|---|---|
| **APRS-IS** | TCP zu einem APRS-IS-Server | Ja |
| **KISS** | Serielle Schnittstelle zu einem Hardware-KISS-TNC | Nein |
| **KISS Server** | TCP zu einem KISS-TNC-Server, z. B. Dire Wolf | Nein, nur lokales Netzwerk |

Wählen Sie den Diensttyp im Menü **Integrations**:

```text
F9 → Integrations → APRS → Service (Space to cycle)
```

Alle drei Dienste unterstützen den Empfang von APRS-Positionsmeldungen benachbarter Stationen und deren Anzeige auf der lokalen Karte von **CQOps Live** mit:

- standardmäßigen APRS-Symbolen,
- Callsign-Popups,
- automatischer Anpassung des Kartenausschnitts,
- konfigurierbarem Reichweitenkreis.

Alle Dienste unterstützen außerdem **periodic position beaconing**. CQOps sendet den Grid locator der Station im konfigurierten Intervall. Wenn GPS aktiv und **Grid from GPS** eingeschaltet ist, verwendet der Beacon automatisch die GPS-basierte Position — ideal für Portabel- und Mobilbetrieb.

#### APRS-IS

Verbindet sich über das Internet mit dem weltweiten APRS-IS-Netz. Erforderlich sind:

- ein gültiges Amateurfunkrufzeichen,
- ein aus dem Rufzeichen erzeugter APRS-IS passcode,
- eine Internetverbindung.

Standardserver:

```text
euro.aprs2.net:14580
```

APRS-IS wird global unter **F9 → Integrations → APRS** konfiguriert. Callsign, SSID, symbol, comment, beacon interval und range filter pro Logbuch werden unter **F9 → Logbooks → [active logbook] → APRS** eingestellt.

#### KISS (serial)

Verbindet sich direkt über eine serielle Schnittstelle mit einem Hardware-KISS-TNC. Eine Internetverbindung ist nicht erforderlich; APRS-Frames werden über das Funkgerät gesendet und empfangen.

Konfigurieren Sie serial port, baud rate, data bits, parity, stop bits sowie DTR/RTS im Menü **Integrations**:

```text
F9 → Integrations → APRS → Service: KISS
```

Wenn **KISS** gewählt ist, werden die seriellen Felder **Port**, **Baud**, **Data bits**, **Parity**, **Stop bits**, **DTR** und **RTS** sichtbar.

Die Schaltfläche **Test** öffnet die serielle Schnittstelle, um zu prüfen, ob der TNC erreichbar ist.

#### KISS Server (TCP)

Verbindet sich mit einem über TCP erreichbaren KISS-TNC, zum Beispiel mit einer [Dire Wolf](https://github.com/wb2osz/direwolf)-Instanz auf demselben Rechner oder im lokalen Netzwerk. Eine Internetverbindung ist nicht erforderlich.

Geben Sie Host und Port im Menü **Integrations** ein:

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Standardwerte: `127.0.0.1:8001`

#### Beaconing

Beacons werden im pro Logbuch konfigurierten Intervall gesendet. Das Mindestintervall beträgt 1 Minute. Der Beacon enthält:

- Stations-callsign mit SSID,
- Grid locator, wenn möglich GPS-basiert,
- APRS-symbol,
- optionalen comment.

Wenn **GPS** aktiv und **Grid from GPS** in den Einstellungen **Station** eingeschaltet ist, verwendet der Beacon automatisch den GPS-basierten Grid locator. Bei Bewegung ist keine manuelle Grid-Aktualisierung erforderlich.

Beacon interval und weitere Einstellungen pro Logbuch werden hier konfiguriert:

```text
F9 → Logbooks → [active logbook] → APRS
```

#### Receiving

Empfangene APRS-Positionsmeldungen werden lokal zwischengespeichert und auf der Karte des **CQOps Live dashboard** angezeigt. Stationen erscheinen mit ihren APRS-Symbolen und können für Details angeklickt werden. Die Ansicht passt sich automatisch an, sodass alle sichtbaren Stationen innerhalb der konfigurierten Reichweite dargestellt werden.

APRS-Empfang und Beacon-Senden sind voneinander unabhängig — Sie können empfangen, ohne zu senden, und umgekehrt. Aktivieren Sie APRS einfach im Menü **Integrations** und wählen Sie den Diensttyp.

### Solar Data

Solardaten stammen von hamqsl.com und umfassen:

- SFI,
- sunspot number,
- A/K indices,
- Bedingungen für die einzelnen Bänder.

Live-Aktualisierungen benötigen Internetzugang. Nach einem erfolgreichen Abruf bleiben zwischengespeicherte Daten offline verfügbar.

---

<a id="cqops-live-dashboard"></a>
## CQOps Live Dashboard

CQOps Live ist ein integriertes Browser-Dashboard für die Stationsaktivität in Echtzeit.

Es eignet sich für:

- öffentliche Anzeigen bei Field Days,
- Bildschirme an Clubstationen,
- Contest-Überwachung,
- Beobachtung der Station aus einem anderen Raum,
- Veranstaltungs- oder Messestände.

### Dashboard aktivieren

1. Drücken Sie **F9**.
2. Öffnen Sie **Integrations**.
3. Gehen Sie zu **HTTP Server**.
4. Aktivieren Sie **HTTP server**.
5. Legen Sie optional address und port fest.
6. Drücken Sie **Ctrl+S**, um zu speichern.
7. Öffnen Sie das Dashboard in einem Browser.

Standardeinstellungen:

| Setting | Default |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

Der Server startet unmittelbar nach dem Speichern.

> **Address binding:** Die Voreinstellung `0.0.0.0` macht das Dashboard von jedem Gerät im lokalen Netzwerk erreichbar. Das ist für Field-Day-Anzeigen, Clubstationsbildschirme oder die Kontrolle aus einem anderen Raum nützlich. Setzen Sie address auf `127.0.0.1`, um den Zugriff auf den lokalen Rechner zu beschränken.

### Anzeigemodi

CQOps Live besitzt zwei Anzeigemodi.

#### Overview mode

Wird angezeigt, wenn gerade kein aktives callsign gearbeitet wird.

Enthält:

- **live maps** — Marker der heutigen QSOs mit Großkreisverbindungen vom eigenen Stations-Grid zu jeder Gegenstation sowie eine lokale APRS-Karte mit Stationen in der Umgebung,
- Tabelle der recent QSOs,
- Stationsinformationen,
- Statistiken,
- Rate-Tracking über 5 Minuten, 15 Minuten und 1 Stunde,
- Top-Operatoren,
- QSOs mit der größten Entfernung.

#### Active / Now Working mode

Wird angezeigt, wenn gerade ein callsign gearbeitet wird.

Enthält:

- großes callsign,
- submode-Anzeige,
- QRZ-Foto, sofern verfügbar,
- band- und mode-Kennzeichnungen,
- **DUPE / NEW CALL / NEW DXCC**-Indikatoren,
- Entfernung und Peilung,
- hervorgehobenen gestrichelten Kartenpfad vom Stations-Grid zum Grid der Gegenstation.

### Info box

Die **Info box** oberhalb der lokalen Karte wechselt alle 5 Sekunden zwischen diesen Modulen:

- Bandbedingungen,
- Sonnenaktivität,
- geomagnetisches Feld,
- neuester DX-Cluster-Spot,
- Anzahl der PSK-Reporter-Meldungen pro Band.

### Weather row

Die **Weather row** zeigt aktuelle Open-Meteo-Bedingungen für den Grid locator der Station:

- Temperatur,
- Wind,
- Luftfeuchtigkeit,
- Symbol.

Wetterdaten werden im Browser abgerufen und bei Offline-Betrieb ohne Fehler ausgelassen.

### Local map

Die rechte **local map** ist für die **Überwachung der APRS-Nachbarschaft** bestimmt. Sie kann Folgendes anzeigen:

- nahe APRS-Stationen mit Standardsymbolen,
- Callsign-Popups bei Hover oder Klick,
- konfigurierbaren Reichweitenkreis,
- optionales Tag-/Nacht-Terminator-Overlay,
- optionales RainViewer-Wetterradar-Overlay.

### Echtzeitaktualisierung und Leistung

CQOps Live aktualisiert sich über Server-Sent Events (SSE). Ein Neuladen der Seite ist nicht erforderlich.

Das Dashboard ist für leistungsschwache Hardware ausgelegt:

- der Browser rendert die Karten,
- der Browser berechnet Entfernungen,
- der Browser berechnet Statistiken,
- CQOps überträgt kompakte JSON-Aktualisierungen,
- wenn **HTTP server** deaktiviert ist, wird kein Port geöffnet und es laufen keine Dashboard-goroutines.

### Dashboard anpassen

Im Integrationsformular **HTTP Server** können Sie Folgendes konfigurieren:

| Field | Description |
|---|---|
| Header 1 | Haupttitel im Seitenkopf und im Hero-Bereich. Fällt auf „CQOps Live“ zurück. |
| Header 2 | Untertitel unter dem Titel. Fällt auf „Fast, portable ham radio logger“ zurück. |
| Logo URL | Öffentlich erreichbare Bild-URL oben links. Fällt auf das CQOps-Logo zurück. |
| Event Start | Datum im Format `YYYY-MM-DD`. Filtert Statistiken und QSO-Listen ab diesem Datum. |

---

<a id="configuration"></a>
## Konfiguration

Öffnen Sie die Konfiguration mit **F9**.

### Konfigurationsdateien

| Plattform | Konfigurationspfad |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Vertrauliche Zugangsdaten werden separat in `secrets.enc` im selben Konfigurationsverzeichnis gespeichert.

Secrets werden mit einem an den Rechner gebundenen Schlüssel verschlüsselt. Beim Übertragen der Konfiguration auf einen anderen Rechner müssen die Zugangsdaten erneut eingegeben werden.

### Konfigurationsmenüs

Drücken Sie **F9**, um das Hauptmenü zu öffnen, und wählen Sie dann:

| Menu | Configures |
|---|---|
| General | Units, timezone, partner map/picture, solar panel, SCP/REF-Datenquellen, Kitty Graphics, Debug mode |
| Logbooks | Station callsign, grid, references, CQ/ITU zones, IARU region, GPS grid; Wavelog pro Logbuch (URL, API key, station profile); APRS pro Logbuch (callsign, symbol, beacon, range) |
| Operators | Operator-callsign- und operator-name-Profile für Multi-Operator-Stationen |
| Rigs | Rig-Presets: model, antenna, power, backend (None/flrig/Hamlib), rotor, WSJT-X UDP |
| Contests | Contest-Profile: name, date, ADIF contest ID, exchange templates, starting serial number |
| Integration | DX Cluster (host, port, login), HTTP Server für das Dashboard (address, port, branding), GPS service (serial/GPSD, grid precision) |
| Callbook | QRZ.com, HamQTH, QRZ.RU, Callook.info Anbieter; Prioritätsreihenfolge, Base-call-Fallback, Wavelog-Lookup |
| Notifications | QSO saved alerts, Wavelog QSO sent status, dupe beep, error sounds |

### Multi-logbook

Verwenden Sie mehrere Logbücher für home, portable, contest und club.

Drücken Sie **Ctrl+L**, um das aktive Logbuch durchzuschalten.

Jedes Logbuch speichert eigene:

- Stationsdaten,
- Wavelog-Einstellungen,
- Contest-Einstellungen,
- Operator-Einstellungen.

### Multi-operator

Operatorprofile enthalten:

- operator callsign,
- operator name.

Drücken Sie **Ctrl+O**, um den aktiven Operator durchzuschalten.

Der aktive Operator wird im ADIF-Feld `OPERATOR` gespeichert und bei Wavelog-Uploads übernommen.

### Multi-rig

Rig-Presets speichern:

- backend,
- model,
- antenna,
- power,
- Rotoreinstellungen,
- WSJT-X-Einstellungen.

Drücken Sie **Ctrl+R**, um das aktive Rig durchzuschalten.

### Verschlüsselte Secrets

Seit v0.8.7 werden Zugangsdaten verschlüsselt gespeichert.

| Element | Wert |
|---|---|
| Secrets-Datei | `secrets.enc` |
| Speicherort | Dasselbe Verzeichnis wie `config.yaml` |
| Unix-Berechtigungen | `0600`, sofern unterstützt |
| Verschlüsselung | AES-256-GCM mit einem an den Rechner gebundenen Schlüssel |
| Geschützte Daten | QRZ password, DX Cluster login, Wavelog API keys |

Klartext-Secrets aus älteren Konfigurationen werden beim ersten Start migriert.

Wenn `secrets.enc` beschädigt ist, startet CQOps mit einer Warnung und fordert Sie zur erneuten Eingabe der Zugangsdaten auf.

---

<a id="keyboard-shortcuts"></a>
## Tastenkürzel

### Global

| Key | Action |
|---|---|
| F1 | **QSO form** und **Recent QSOs** |
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
## Fehlerbehebung

### CQOps startet nicht

Prüfen Sie:

- ob das Terminal mindestens 80×24 Zeichen groß ist,
- ob Windows-Benutzer Windows Terminal verwenden,
- ob der Netzwerkstart blockiert, indem Sie Folgendes versuchen:

  ```bash
  cqops --offline
  ```

Prüfen Sie die Logs:

| Plattform | Logpfad |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### Rig verbindet sich nicht

Für flrig:

- prüfen Sie, ob flrig läuft,
- prüfen Sie den Port im aktiven Rig-Preset,
- der Standardport ist `12345`.

Für Hamlib:

- prüfen Sie, ob `rigctld` läuft,
- prüfen Sie Host und Port,
- prüfen Sie, ob Funkgerät und Backend die angeforderten Daten unterstützen.

Statusbezeichnungen helfen bei der Diagnose:

| Farbe | Bedeutung |
|---|---|
| Weiß/Standard | Verbunden |
| Gelb | Deaktiviert oder Verbindung wird hergestellt |
| Rot | Fehlgeschlagen |

Reconnect-Toasts können unterdrückt sein. CQOps kann Wiederholungsversuche still im Hintergrund durchführen.

### WSJT-X loggt nicht automatisch

Prüfen Sie:

- **WSJT-X Settings → Reporting → UDP Server**,
- ob UDP host und port mit dem aktiven Rig-Preset in CQOps übereinstimmen,
- ob WSJT-X 2.6 oder neuer verwendet wird,
- ob die Statusbezeichnung **WSJT** aktiv ist,
- ob das aktive Logbuch korrekt ist,
- ob der aktive Operator korrekt ist.

### Wavelog-Upload schlägt fehl

Prüfen Sie:

- Wavelog URL,
- API key,
- station profile ID,
- Statusbezeichnung **WL**.

Uploadfehler werden als Toasts angezeigt. QSOs bleiben lokal gespeichert, auch wenn der Upload fehlschlägt. Fehler einzelner QSOs blockieren den Rest der Uploadgruppe nicht.

### Probleme mit der Konfigurationsdatei

Konfigurationsdatei:

| Plattform | Pfad |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets-Datei:

```text
secrets.enc
```

Die Secrets-Datei befindet sich im selben Verzeichnis wie `config.yaml`.

Wenn die Konfiguration beschädigt ist, verschieben oder löschen Sie die Datei und starten CQOps neu. Der Einrichtungsassistent erstellt eine neue Konfiguration.

Das Feld `last_fetched_id` erscheint erst nach einem erfolgreichen Wavelog-Download.

### Leistungsprobleme

Versuchen Sie:

- die Kartenanzeige in **General** zu deaktivieren,
- das Panel **Solar** zu deaktivieren, wenn es nicht benötigt wird,
- netzwerkintensive Bildschirme wie **DX Cluster** und **PSK Reporter** im Offline-Betrieb zu vermeiden,
- `cqops --offline` zu verwenden, wenn die Netzwerkverbindung unzuverlässig ist.

---

<a id="reporting-bugs"></a>
## Fehler melden

Vor dem Melden eines Fehlers:

1. Aktivieren Sie **Debug mode** unter **F9 → General → Debug**, oder setzen Sie:

   ```yaml
   debug: true
   ```

   in `config.yaml`.

2. Reproduzieren Sie das Problem.
3. Fügen Sie das relevante Log bei.

Melden Sie Fehler auf GitHub:

<https://github.com/szporwolik/cqops/issues>

Geben Sie Folgendes an:

- CQOps-Version aus `cqops --version`,
- Betriebssystem,
- Terminalemulator,
- Schritte zum Reproduzieren,
- relevantes debug log.
