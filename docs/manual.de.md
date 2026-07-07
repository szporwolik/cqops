---
title: CQOps Benutzerhandbuch
description: Anleitung zur Installation, Konfiguration und Nutzung von CQOps — einem schnellen, terminalbasierten Amateurfunk-Logger
---

> **Übersetzungshinweis:** Diese Übersetzung wurde mit einem LLM-Modell erstellt. Korrekturen sind willkommen — bitte als Pull Request gegen den Branch `dev` einreichen. Namen von Bildschirmen, Feldern, Befehlen und Tastenkürzeln bleiben teilweise absichtlich auf Englisch, damit sie zur CQOps-Oberfläche passen.

# CQOps Benutzerhandbuch

CQOps ist ein schneller, terminalbasierter Amateurfunk-Logger für Operatoren, die zuverlässiges Tastatur-Logging mit geringer Systemlast wollen. CQOps ist für den Shack, portable Einsätze, Clubstationen, Field Days und Rechner wie Raspberry-Pi-Systeme oder ältere Laptops gedacht.

CQOps speichert QSOs immer zuerst lokal. Internetbasierte Integrationen sind optional.

## Inhalt

1. [Was CQOps ist](#was-cqops-ist)
2. [Download und Installation](#download-und-installation)
3. [Erster Start](#erster-start)
4. [Erstes QSO loggen](#erstes-qso-loggen)
5. [Hauptbildschirm](#hauptbildschirm)
6. [Typische Workflows](#typische-workflows)
7. [QSO-Logging](#qso-logging)
8. [Logbook Editor und ADIF](#logbook-editor-und-adif)
9. [Contests](#contests)
10. [Favorites, Referenzen und Bandpläne](#favorites-referenzen-und-bandpläne)
11. [Integrationen](#integrationen)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Konfiguration](#konfiguration)
14. [Tastenkürzel](#tastenkürzel)
15. [Fehlerbehebung](#fehlerbehebung)
16. [Fehler melden](#fehler-melden)

---

## Was CQOps ist

CQOps ist auf schnelle QSO-Eingabe, lokales Logging und praktischen Feldeinsatz ausgelegt.

### Grundideen

- **Terminal-first** — für Tastaturbedienung optimiert.
- **Offline-first** — lokales QSO-Logging funktioniert ohne Internetzugang.
- **Geringe Systemlast** — geeignet für Raspberry-Pi-Klasse, ältere Laptops und gemeinsam genutzte Stations-PCs.
- **Portables Design** — Auslieferung als einzelne Go-Binary.
- **Mehrere Logbücher** — nützlich für Privat-, Portable-, Contest- und Club-Logs.
- **Mehrere Operatoren** — nützlich für Hot-Seat- und Clubstationsbetrieb.
- **Mehrere Rigs** — jedes Rig-Preset kann eigenes Backend und eigene WSJT-X-Einstellungen haben.
- **Optionale Integrationen** — QRZ.com, Wavelog, DX Cluster, PSK Reporter, APRS, GPS-Empfänger, Rig-Steuerung, Rotor-Steuerung, Solardaten und CQOps Live im Browser.

Lokales Logging benötigt keinen Internetzugang. Netzwerkfunktionen werden im Modus `--offline` übersprungen.

### Für wen CQOps geeignet ist

CQOps passt gut für:

- portable Operatoren,
- SOTA- und POTA-Aktivierer,
- Clubstationen,
- Field-Day-Teams,
- Operatoren, die einen Terminal-Workflow bevorzugen,
- Stationen, die schnell zwischen Operatoren, Logbüchern oder Rigs wechseln müssen.

CQOps soll nicht jede Funktion eines vollständigen Desktop-Loggers oder einer webbasierten Logbuchplattform ersetzen. Der Fokus liegt auf schnellem Terminal-Logging, Feldeinsatz, Offline-Betrieb und gemeinsam genutzten Stationen.

---

## Download und Installation

Alle Releases:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Paket | Link | Hinweise |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Empfohlen für die meisten Nutzer. Fügt CQOps zum Startmenü und PATH hinzu. |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Entpacken und ohne Installation starten. |

Verwenden Sie **Windows Terminal** statt der alten Konsole.

### Linux — Debian / Ubuntu

| Architektur | Link | Einsatz |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Die meisten Intel/AMD-PCs |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-Bit-ARM-Systeme |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-Bit Raspberry Pi OS |

Paket installieren:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — portables Tarball

| Architektur | Link | Einsatz |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Die meisten Intel/AMD-PCs |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-Bit-ARM-Systeme |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-Bit Raspberry Pi OS |

### macOS

| Architektur | Link | Einsatz |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | M1/M2/M3 Macs |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Intel Macs |

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

Quellcode-Builds benötigen Go 1.26 oder neuer.

### Terminal-Anforderungen

| Anforderung | Wert |
|---|---|
| Minimale Terminalgröße | 80×24 Zeichen |
| Empfohlene Terminalgröße | 80×43 Zeichen oder größer |
| Empfohlenes Windows-Terminal | Windows Terminal |
| Kitty-Grafik Terminal | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) oder [WezTerm](https://wezfurlong.org/wezterm/) |

### Grundbefehle

```bash
cqops              # TUI starten
cqops --offline    # Ohne Netzwerkaktivität starten
cqops --version    # Version ausgeben und beenden
cqops --help       # Hilfe anzeigen
```

---

## Erster Start

Beim ersten Start öffnet CQOps den Setup Wizard. Für lokales Logging sind nur die wichtigsten Stationsdaten erforderlich. Netzwerkintegrationen können übersprungen und später konfiguriert werden.

### Wizard-Seiten

| Seite | Konfiguriert |
|---|---|
| Station & Logbook | Erstes Logbuch, Stationsrufzeichen, Operator, Grid Locator, optionale Referenzen und Zonen, Wavelog URL/API/station profile ID |
| Rig | Rig-Preset, Modell, Antenne, Leistung, Backend, optionaler Rotor, optionale WSJT-X-UDP-Einstellungen |
| Integrations | QRZ.com lookup settings |
| General | IANA-Zeitzone |
| Summary | Prüfen und speichern |

Unterstützte Rig-Backends:

- None,
- flrig,
- Hamlib `rigctld`.

### Wizard-Navigation

| Taste | Aktion |
|---|---|
| Ctrl+S | Validieren und fortfahren; auf Summary speichern und CQOps starten |
| Esc | Zurück |
| F10 | Beenden |
| Tab / Shift+Tab | Zwischen Feldern wechseln |
| Space | Checkbox umschalten |

Wizard-Einstellungen können später mit **F9** geändert werden.

---

## Erstes QSO loggen

1. CQOps starten:

   ```bash
   cqops
   ```

2. Den Setup Wizard mindestens mit Rufzeichen und Grid Locator abschließen.
3. Das QSO form mit **F1** öffnen.
4. Das Rufzeichen des Kontakts eingeben. CQOps schreibt Rufzeichen automatisch groß.
5. Weitere Felder ausfüllen. Wenn das aktive Rig über flrig oder Hamlib verbunden ist, kann CQOps Frequenz, Band, Mode und Submode automatisch eintragen.
6. **Enter** oder **Ctrl+S** drücken, um zu speichern.
7. Wenn **DUPE!** erscheint, erneut **Enter** drücken, um trotzdem zu speichern, oder **Esc** zum Abbrechen.

Das gespeicherte QSO erscheint sofort in der Tabelle Recent QSOs.

---

## Hauptbildschirm

CQOps verwendet ein festes Terminal-Layout:

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

Die Status bar zeigt:

- CQOps-Version,
- aktives Logbuch,
- aktives Rig,
- Stationsrufzeichen,
- aktiven Operator,
- Integrationsstatus-Labels,
- lokale Zeit als `L`,
- UTC-Zeit als `Z`.

Typische Labels sind **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** und **GPS**. Die GPS-Anzeige folgt der gleichen Farbkonvention — rot bei getrennt, gelb bei verbunden ohne Fix, weiß wenn eine Position erfasst wurde.

| Farbe | Bedeutung |
|---|---|
| Weiß/Standard | Verbunden oder aktiv |
| Gelb | Deaktiviert, verbindet oder erwarteter Offline-Zustand |
| Rot | Fehler oder getrennt |
| Akzent + fett | WSJT-X sendet |

### Haupt-Tabs

| Taste | Tab | Bildschirm |
|---|---|---|
| F1 | QSO | QSO form und Recent QSOs |
| F2 | QRZ | Partner view: Callbook-Daten, Karte, Statistik, Foto |
| F4 | DXC | DX Cluster Spots und Filter |
| F5 | HRD | PSK Reporter Spots und Ausbreitungskarte |
| F6 | REF | SOTA/POTA/WWFF/IOTA-Referenzsuche |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, Wavelog Sync |
| F9 | CFG | Konfigurationsmenüs |

Die Help bar zeigt passende Tastenkürzel für den aktiven Bildschirm. **?** öffnet die vollständige Hilfe.

---

## Typische Workflows

### Portable-, SOTA- oder POTA-Betrieb

Vor dem Losfahren:

1. CQOps einmal mit Internetzugang starten.
2. CQOps Cache-Daten herunterladen oder aktualisieren lassen, z. B. Solardaten, REF-Daten und DXCC-Präfixe.
3. Prüfen, ob das Solar panel Daten zeigt.
4. Prüfen, ob REF search auf **F6** Ergebnisse liefert.

Im Feld:

1. CQOps offline starten:

   ```bash
   cqops --offline
   ```

2. Normal loggen. QSOs werden lokal gespeichert.
3. Wieder online **F8** öffnen und **w** drücken, um nicht gesendete QSOs zu Wavelog hochzuladen.

### Gemeinsame Clubstation und Hot-Seat-Logging

1. **F9 → Operators** öffnen.
2. **Ins** drücken, um Operator-Profile hinzuzufügen.
3. Im QSO form **Ctrl+O** drücken, um den aktiven Operator zu wechseln.
4. Vor dem Speichern den aktiven Operator in der Status bar prüfen.
5. **Retain** verwenden, wenn mehrere Operatoren ähnliche Kontakte loggen, ohne das ganze Formular erneut auszufüllen.

Der aktive Operator wird im ADIF-Feld `OPERATOR` gespeichert.

### Persönliche und Club-Logbücher

1. **F9 → Logbooks** öffnen.
2. **Ins** drücken, um Logbücher anzulegen.
3. Im QSO form **Ctrl+L** drücken, um das aktive Logbuch zu wechseln.
4. Vor dem Speichern das aktive Logbuch in der Status bar prüfen.

Jedes Logbuch kann eigene Stationsdaten, Wavelog-Einstellungen, Contest-Einstellungen und Operatoren behalten.

### Mehrere Rigs

1. **F9 → Rigs** öffnen.
2. **Ins** drücken, um Rig-Presets anzulegen.
3. Backend auswählen: None, flrig oder Hamlib.
4. Im QSO form **Ctrl+R** drücken, um das aktive Rig zu wechseln.

Ein Rig-Preset kann Backend, Modell, Antenne, Leistung, Rotor-Einstellungen und WSJT-X-UDP-Einstellungen enthalten.

### WSJT-X-Digitalbetrieb

Wenn die WSJT-X-UDP-Integration aktiviert ist, kann CQOps ADIF-Nachrichten von WSJT-X empfangen und abgeschlossene digitale QSOs automatisch loggen.

Automatisch geloggte QSOs:

- werden im aktiven Logbuch gespeichert,
- erscheinen sofort in Recent QSOs,
- überspringen Duplikate,
- übernehmen die aktive contest ID,
- können automatisch zu Wavelog hochgeladen werden, wenn Wavelog konfiguriert und erreichbar ist.

Wenn der von WSJT-X gemeldete Operator nicht zum aktiven Operator in CQOps passt, zeigt CQOps eine Warnung.

Vor längeren Digital-Sessions prüfen:

- aktives Logbuch,
- aktiven Operator,
- aktiven Contest,
- WSJT-X-Statuslabel.

### Wavelog Sync

CQOps speichert QSOs zuerst lokal. Wavelog Sync ist optional.

| Aktion | Wo | Tastenkürzel | Hinweise |
|---|---|---|---|
| Nicht gesendete QSOs hochladen | Logbook Editor | `w` | Upload in Batches von 50 |
| Von Wavelog herunterladen | Logbook Editor | `Ctrl+W` | Inkrementeller Download über `last_fetched_id` |

Upload-Status wird pro QSO verfolgt:

- not sent,
- sent,
- error.

Wenn der Upload fehlschlägt, bleibt das QSO lokal gespeichert und kann später erneut versucht werden. Das Purgen eines Logbuchs setzt die fetch ID auf `0`, wodurch ein vollständiger erneuter Download möglich wird.

---

## QSO-Logging

Das QSO form ist der Hauptbildschirm zum Loggen. Es wird mit **F1** geöffnet.

CQOps kann Felder aus diesen Quellen füllen:

| Quelle | Felder |
|---|---|
| flrig / Hamlib | Frequency, Freq RX bei Split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| REF database | SOTA-, POTA-, WWFF-, IOTA-Referenzen |
| Wavelog lookup | Worked/confirmed status, wenn konfiguriert |
| DXCC/prefix data | Präfix- und Länderinformationen |

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

Exchange-Felder erscheinen nur, wenn ein Contest aktiv ist.

Die untere Zeile enthält:

- **Comment**,
- **Keep** — behält das Comment-Feld zwischen QSOs,
- **Retain** — behält das ganze Formular nach dem Speichern.

Felder wie Band, Mode und Submode können mit **PgUp/PgDn** gewechselt werden.

### Pfad, Peilung und Badges

Wenn beide Grid Locators bekannt sind, zeigt CQOps Entfernung und Azimut.

Das QSO form kann außerdem Badges anzeigen:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Speichern

| Taste | Aktion |
|---|---|
| Enter | QSO speichern |
| Ctrl+S | QSO aus jedem Feld speichern |
| Esc | Duplikatbestätigung abbrechen |
| Enter bei DUPE-Bestätigung | Duplikat trotzdem speichern |

---

## Logbook Editor und ADIF

Den Logbook Editor mit **F8** öffnen.

Er dient für:

- QSO-Prüfung,
- Inline-Bearbeitung,
- Löschen von QSOs,
- ADIF-Import,
- ADIF-Export,
- Wavelog-Upload,
- Wavelog-Download,
- contestbezogene Operationen.

### QSOs bearbeiten

1. Zeile mit **↑/↓** auswählen.
2. **Enter** oder **e** drücken.
3. QSO bearbeiten.
4. Mit **Ctrl+S** speichern.

Änderungen erscheinen sofort in Recent QSOs.

### ADIF-Import und -Export

CQOps unterstützt ADIF 3.1.7 Import und Export.

| Aktion | Tastenkürzel |
|---|---|
| ADIF importieren | Ctrl+I |
| ADIF exportieren | Ctrl+E |

Der Import validiert Datensätze, überspringt Duplikate und zeigt eine Zusammenfassung. Importierte QSOs werden für Wavelog-Upload markiert, wenn Wavelog Sync konfiguriert ist.

Der Export kann alle QSOs oder contestgefilterte QSOs enthalten. `CONTEST_ID` bleibt erhalten.

### Behandlung digitaler Modi

Mode- und Submode-Behandlung folgt ADIF 3.1.7 wie in diesem Handbuch beschrieben:

- FT8 wird als eigenständiger Mode exportiert.
- FT4 und FT2 werden als MFSK mit passendem Submode exportiert.
- Importierte ältere MFSK + FT8-Datensätze werden zu eigenständigem FT8 normalisiert.

Das QSO form hat separate Felder **Mode** und **Submode**. Beide können mit **PgUp/PgDn** gewechselt werden.

---

## Contests

Contests ergänzen das QSO form um Exchange-Felder und Seriennummern.

Ein Contest wird im Logbook Editor mit **Ins** erstellt oder konfiguriert.

Contest-Konfiguration enthält:

- Contest-Name,
- Datum,
- ADIF contest ID,
- Exchange-Templates.

### Template-Marker

| Marker | Ersetzt durch |
|---|---|
| `@rst` | gesendetes oder empfangenes RST |
| `@serial` | automatisch erhöhte Seriennummer |
| `@call` | eigenes Rufzeichen |
| `@grid` | eigener Grid Locator |
| `@name` | Operator-Name aus dem Operator-Profil |

**Ctrl+C** wechselt den aktiven Contest.

Wenn ein Contest aktiv ist:

- zeigt das QSO form Exchange-Felder,
- erhöhen sich Seriennummern automatisch,
- können Recent QSOs nach Contest-QSOs filtern,
- bewahrt der ADIF-Export `CONTEST_ID`.

---

## Favorites, Referenzen und Bandpläne

### Favorites

Favorites speichern Frequenz-, Mode- und Band-Presets in 10 Slots.

| Tastenkürzel | Aktion |
|---|---|
| Alt+0–9 | Favorite laden |
| Alt+Shift+0–9 | aktuelle frequency, mode und band als Favorite speichern |

Favorites werden in der Konfiguration gespeichert und zwischen Logbüchern geteilt.

Beispiel:

1. `145.55` eingeben.
2. Mode auf `FM` setzen.
3. Band auf `2m` setzen.
4. **Alt+Shift+1** drücken.
5. Später **Alt+1** drücken, um das Preset zu laden.

### REF Lookup

REF Lookup mit **F6** öffnen.

Es sucht:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Suche nach Präfix, Name oder Referenzbezeichnung ist möglich. Gewählte Referenzen können das QSO form füllen.

### Band Plan Browser

Band Plan Browser mit **F7** öffnen.

Er bietet schnellen Zugriff auf:

- Amateurbänder,
- VHF/UHF-Bereiche,
- CB,
- PMR446,
- Broadcast-Presets.

Eine ausgewählte Frequenz kann zum Tunen des aktiven Rigs genutzt werden. Bandplan-Daten können auch als Markdown exportiert werden.

---

## Integrationen

Alle Integrationen sind optional. Lokales Logging funktioniert ohne sie.

### QRZ.com

QRZ.com lookup benötigt Internetzugang und ein QRZ-XML-Abo.

Im QSO form **Ins** drücken, um Callbook-Felder zu füllen, z. B.:

- name,
- QTH,
- grid,
- country,
- CQ/ITU zones,
- DXCC,
- continent.

Die Partner view auf **F2** kann das Operator-Foto anzeigen, wenn verfügbar.

> ⚠️ **Experimentell.** Die Fotoanzeige kann das Kitty-Terminal-Grafikprotokoll
> nutzen und erfordert ein kompatibles Terminal: Kitty, Ghostty oder WezTerm.
> Aktiviere in **F9 → General → Kitty Graphics**. Standard-Terminals und
> SSH-Sitzungen ohne Grafikweiterleitung zeigen stattdessen ein Glyph-Foto.

### Wavelog

Wavelog-Integration unterstützt:

- Upload,
- inkrementellen Download,
- worked/confirmed lookup.

Wavelog wird pro aktivem Logbuch konfiguriert mit:

- URL,
- API key,
- station profile ID.

CQOps speichert QSOs immer zuerst lokal. Ein Wavelog-Uploadfehler löscht keine lokalen Daten.

### flrig

flrig-Integration nutzt XML-RPC über HTTP.

Standard-Endpunkt:

```text
localhost:12345
```

CQOps kann lesen:

- frequency,
- mode,
- power.

Split-Betrieb mappt VFO A auf Frequency und VFO B auf Freq RX.

### Hamlib / rigctld

Hamlib-Rig-Steuerung nutzt den TCP-Daemon `rigctld`.

Je nach Radio und Backend kann CQOps abfragen:

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps behandelt fehlende VFO-Namensunterstützung nach Möglichkeit robust.

### Hamlib Rotor / rotctld

> ⚠️ **Experimentell.** Rotorsteuerung ist experimentell. Überprüfe immer
> die physischen Grenzen deiner Antenne vor der Nutzung. Sei bereit, die
> Bewegung sofort mit **Alt+/** zu stoppen. Mit Vorsicht verwenden — falsche
> Konfiguration kann deinen Rotor oder deine Antenne beschädigen.

Rotorsteuerung nutzt Hamlib `rotctld`.

CQOps unterstützt:

- azimuth,
- elevation,
- stop commands.

| Tastenkürzel | Aktion |
|---|---|
| Alt+, | Azimuth −5° |
| Alt+. | Azimuth +5° |
| Alt+; | Elevation +5° |
| Alt+' | Elevation −5° |
| Alt+\ | Rotor auf berechnete path bearing ausrichten |
| Alt+/ | Rotor stoppen |

### WSJT-X

WSJT-X-Integration nutzt UDP-Nachrichten von WSJT-X. CQOps parst ADIF-Nachrichten und kann abgeschlossene QSOs automatisch loggen.

Das Rig-Label wird akzentfarben, während WSJT-X sendet. Wenn der von WSJT-X gemeldete Operator nicht zum aktiven Operator passt, zeigt CQOps eine Warnung.

### GPS

CQOps kann die Position von einem GPS-Empfänger lesen und als Stations-Grid-Locator
verwenden — ideal für Portabel-, Mobil- oder Feldeinsätze.

Zwei Backends werden unterstützt:

- **Serial** — verbindet direkt mit einem GPS-Empfänger über eine serielle
  Schnittstelle (USB-zu-Seriell, integrierter COM-Port oder `/dev/ttyUSB0`).
- **GPSD** — verbindet mit einem [gpsd](https://gpsd.io/)-Server über TCP
  (Standard `127.0.0.1:2947`). Nützlich, wenn das GPS mit anderen Anwendungen
  geteilt oder über das Netzwerk genutzt wird.

Die GPS-Statusanzeige in der Statusleiste zeigt:

| Farbe | Bedeutung |
|--------|---------|
| Rot `GPS` | Getrennt / Fehler |
| Gelb `GPS` | Verbunden, noch kein Fix |
| Weiß `GPS` | Fix erfasst, Position gesperrt |

Wenn ein Fix erfasst wird, wird der Stations-Grid-Locator durch die
GPS-Position ersetzt und mit `(GPS)` in der Statuszeile markiert:

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Aktivieren Sie **Grid from GPS** in den Station & Logbook-Einstellungen,
um das GPS-Grid für QSO-Logging, APRS-Baken, die Dashboard-Karte und
Entfernungsberechnungen zu verwenden.

**Grid-Genauigkeit** — konfigurierbar im Integration-Menü (10, 8 oder 6
Zeichen). Standard ist 10 Zeichen (~25 m Genauigkeit).

### DX Cluster

DX Cluster-Integration nutzt Telnet und erfordert Internetzugang.

Standardserver:

```text
dxspots.com:7300
```

Filter umfassen:

- band,
- continent,
- mode,
- age/time.

| Taste | Aktion |
|---|---|
| Enter | QSO form füllen, Rig tunen und zu QSO zurückkehren |
| Space | Rig tunen und auf DX Cluster bleiben |
| Backspace | Filter löschen |

### PSK Reporter

PSK Reporter benötigt Internetzugang.

Er bietet:

- Ausbreitungs-Spots,
- band/time/mode-Filter,
- ASCII world map auf **F5**.

### APRS

CQOps unterstützt drei APRS-Diensttypen – wähle den passenden für dein
Station-Setup:

| Dienst | Verbindung | Internet nötig |
|---|---|---|
| **APRS-IS** | TCP zu einem APRS-IS-Server | Ja |
| **KISS** | Serielle Schnittstelle zu einem Hardware-KISS-TNC | Nein |
| **KISS Server** | TCP zu einem KISS-TNC-Server (z.B. Dire Wolf) | Nein (lokales Netzwerk) |

Wähle den Diensttyp im Integrationsmenü:

```text
F9 → Integrations → APRS → Service (Leertaste zum Wechseln)
```

Alle drei Dienste unterstützen den Empfang von APRS-Positionsberichten
nahegelegener Stationen und deren Anzeige auf der CQOps-Live-Karte mit:

- standard APRS-Symbolen,
- Callsign-Popups,
- auto-fit view,
- konfigurierbarem range circle.

Alle Dienste unterstützen auch **periodische Positions-Beacons**. CQOps
sendet deinen Grid-Locator im konfigurierten Intervall. Wenn GPS aktiv ist
und **Grid from GPS** aktiviert ist, verwendet das Beacon automatisch die
GPS-Position – ideal für portablen und mobilen Betrieb.

#### APRS-IS

Verbindet sich mit dem globalen APRS-IS-Netzwerk über das Internet.
Benötigt:

- ein gültiges Amateurfunk-Rufzeichen,
- einen APRS-IS-Passcode (aus dem Rufzeichen generiert),
- eine Internetverbindung.

Standardserver:

```text
euro.aprs2.net:14580
```

APRS-IS wird global unter **F9 → Integrations → APRS** konfiguriert.
Rufzeichen, SSID, Symbol, Kommentar, Beacon-Intervall und Reichweitenfilter
pro Logbuch unter **F9 → Logbooks → [aktives Logbuch] → APRS**.

#### KISS (seriell)

> ⚠️ **Experimentell.** KISS-TNC-Unterstützung ist experimentell. Gründlich
> testen, bevor du dich im Betrieb darauf verlässt.

Verbindet sich direkt mit einem Hardware-KISS-TNC über eine serielle
Schnittstelle. Keine Internetverbindung nötig – APRS-Frames werden über
dein Funkgerät gesendet und empfangen.

Konfiguriere Port, Baudrate, Datenbits, Parität, Stoppbits und DTR/RTS
im Integrationsmenü:

```text
F9 → Integrations → APRS → Service: KISS
```

Wenn KISS ausgewählt ist, werden die seriellen Felder (Port, Baud,
Datenbits, Parität, Stoppbits, DTR, RTS) sichtbar.

Der **Test**-Button öffnet die serielle Schnittstelle, um zu prüfen,
ob der TNC erreichbar ist.

#### KISS Server (TCP)

> ⚠️ **Experimentell.** KISS-Server-Unterstützung ist experimentell. Gründlich
> testen, bevor du dich im Betrieb darauf verlässt.

Verbindet sich mit einem KISS-TNC, der über TCP erreichbar ist – z.B.
eine [Dire Wolf](https://github.com/wb2osz/direwolf)-Instanz auf demselben
Rechner oder im lokalen Netzwerk. Keine Internetverbindung nötig.

Gib die Host-Adresse und den Port im Integrationsmenü ein:

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Standard: `127.0.0.1:8001`

#### Beaconing

Beacons werden im pro Logbuch konfigurierten Intervall gesendet. Das
minimale Intervall beträgt 1 Minute. Das Beacon enthält:

- Stationsrufzeichen mit SSID,
- Grid-Locator (GPS-basiert, wenn verfügbar),
- APRS-Symbol,
- optionalen Kommentar.

Wenn **GPS** aktiv ist und **Grid from GPS** in den Station-Einstellungen
aktiviert ist, verwendet das Beacon automatisch den GPS-Grid-Locator –
kein manuelles Grid-Update während der Fahrt nötig.

Beacon-Intervall und andere Einstellungen pro Logbuch:

```text
F9 → Logbooks → [aktives Logbuch] → APRS
```

#### Empfang

Empfangene APRS-Positionsberichte werden lokal gecached und auf der
CQOps-Live-Dashboard-Karte angezeigt. Stationen werden mit ihren
APRS-Symbolen dargestellt und können für Details angeklickt werden.
Die Anzeige passt sich automatisch an, um alle sichtbaren Stationen
innerhalb der konfigurierten Reichweite zu zeigen.

APRS-Empfang ist unabhängig vom Beacon-Senden – du kannst empfangen,
ohne ein Beacon zu senden, und umgekehrt. Aktiviere einfach APRS im
Integrationsmenü und wähle den Diensttyp.

### Solar Data

Solar data kommt von hamqsl.com und enthält:

- SFI,
- sunspot number,
- A/K indices,
- band-by-band conditions.

Live-Updates benötigen Internetzugang. Gecachte Daten bleiben nach erfolgreichem Abruf offline verfügbar.

---

## CQOps Live Dashboard

CQOps Live ist ein eingebautes Browser-Dashboard für Stationsaktivität in Echtzeit.

Nützlich für:

- öffentliche Field-Day-Anzeigen,
- Clubstations-Bildschirme,
- Contest-Monitoring,
- Beobachtung der Station aus einem anderen Raum,
- Event- oder Messestände.

### Dashboard aktivieren

1. **F9** drücken.
2. **Integrations** öffnen.
3. Zu **HTTP Server** gehen.
4. **HTTP server** aktivieren.
5. Optional Adresse und Port setzen.
6. Mit **Ctrl+S** speichern.
7. Dashboard im Browser öffnen.

Standardwerte:

| Einstellung | Standard |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

Der Server startet direkt nach dem Speichern.

### Anzeigemodi

CQOps Live hat zwei Anzeigemodi.

#### Overview mode

Wird gezeigt, wenn kein aktives Rufzeichen gearbeitet wird.

Zeigt:

- live Leaflet map,
- heutige QSO-Marker,
- great-circle paths,
- recent QSOs table,
- Stationsinformationen,
- Statistiken,
- 5-Minuten-, 15-Minuten- und 1-Stunden-Rate,
- top operators,
- QSOs mit größter Distanz.

#### Active / Now Working mode

Wird gezeigt, wenn gerade ein Rufzeichen gearbeitet wird.

Zeigt:

- großes callsign,
- submode indicator,
- QRZ-Foto, falls verfügbar,
- band- und mode-Badges,
- DUPE / NEW CALL / NEW DXCC-Indikatoren,
- distance und bearing,
- hervorgehobenen gestrichelten Kartenpfad vom Stationsgrid zum Partnergrid.

### Info box

Die Info box über der local map wechselt alle 5 Sekunden durch Module:

- band conditions,
- solar activity,
- geomagnetic field,
- letzter DX Cluster Spot,
- PSK Reporter Report-Zahlen pro Band.

Band conditions werden immer über die volle Breite gerendert.

### Weather row

Die weather row zeigt aktuelle Open-Meteo-Bedingungen für den Grid Locator der Station:

- temperature,
- wind,
- humidity,
- icon.

Wetterdaten werden browserseitig abgerufen und degradieren offline sauber.

### Local map

Die rechte local map kann anzeigen:

- APRS-Stationen,
- Standard-APRS-Symbole,
- range circle,
- callsign popups,
- optional day/night terminator overlay,
- optional RainViewer weather radar overlay.

### Echtzeit-Updates und Performance

CQOps Live aktualisiert über Server-Sent Events (SSE). Ein Seitenrefresh ist nicht nötig.

Das Dashboard ist für leistungsschwache Hardware ausgelegt:

- Browser rendert Karten,
- Browser berechnet Distanzen,
- Browser berechnet Statistiken,
- CQOps sendet leichte JSON-Updates,
- wenn der HTTP server deaktiviert ist, wird kein Port geöffnet und es laufen keine Dashboard-Goroutines.

### Dashboard-Anpassung

Im HTTP-Server-Integrationsformular können konfiguriert werden:

| Feld | Beschreibung |
|---|---|
| Header 1 | Haupttitel im Page Header und Hero-Bereich. Fällt auf „CQOps Live“ zurück. |
| Header 2 | Untertitel unter dem Titel. Fällt auf „Fast, portable ham radio logger“ zurück. |
| Logo URL | Öffentlich erreichbare Bild-URL oben links. Fällt auf das CQOps-Logo zurück. |
| Event Start | Datum im Format `YYYY-MM-DD`. Filtert Statistiken und QSO-Listen ab diesem Datum. |

---

## Konfiguration

Konfiguration mit **F9** öffnen.

### Konfigurationsdateien

| Plattform | Config-Pfad |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Sensitive Zugangsdaten werden separat in `secrets.enc` im gleichen Konfigurationsverzeichnis gespeichert.

Secrets werden mit einem maschinengebundenen Schlüssel verschlüsselt. Beim Umzug auf einen anderen Rechner müssen Zugangsdaten neu eingegeben werden.

### Konfigurationsmenüs

| Menü | Konfiguriert |
|---|---|
| Station | Callsign, grid, CQ/ITU zone, IARU region, references |
| Rig | Rig presets, model, antenna, power, backend, rotor, WSJT-X |
| Wavelog | URL, API key, station profile ID |
| QRZ | Username und password |
| DX Cluster | Host, port, login |
| Operators | Operator-Profile |
| Logbooks | Station-, Wavelog-, contest-, operator- und APRS-Einstellungen pro Logbuch |
| Integrations | APRS-Diensttyp (APRS-IS, KISS, KISS Server), GPS, HTTP-Server, DXC, QRZ |
| Notifications | QSO saved alerts, Wavelog status, dupe beep, error sounds |
| General | Timezone, distance units, map, debug mode |

### Multi-logbook

Mehrere Logbücher sind sinnvoll für Heimstation, portable Einsätze, Contests und Clubbetrieb.

**Ctrl+L** wechselt das aktive Logbuch.

Jedes Logbuch behält eigene:

- station details,
- Wavelog settings,
- contest settings,
- operator settings.

### Multi-operator

Operator-Profile enthalten:

- Operator-Rufzeichen,
- Operator-Name.

**Ctrl+O** wechselt den aktiven Operator.

Der aktive Operator wird im ADIF-Feld `OPERATOR` gespeichert und bei Wavelog-Uploads übernommen.

### Multi-rig

Rig-Presets speichern:

- backend,
- model,
- antenna,
- power,
- rotor settings,
- WSJT-X settings.

**Ctrl+R** wechselt das aktive Rig.

### Verschlüsselte Secrets

Seit v0.8.7 werden Zugangsdaten verschlüsselt gespeichert.

| Element | Wert |
|---|---|
| Secrets file | `secrets.enc` |
| Location | Gleiches Verzeichnis wie `config.yaml` |
| Unix permissions | `0600`, wo unterstützt |
| Encryption | AES-256-GCM mit maschinengebundenem Schlüssel |
| Protected data | QRZ password, DX Cluster login, Wavelog API keys |

Plaintext secrets aus älteren Konfigurationen werden beim ersten Start migriert.

Wenn `secrets.enc` beschädigt ist, startet CQOps mit einer Warnung und fordert zur erneuten Eingabe der Zugangsdaten auf.

---

## Tastenkürzel

### Global

| Taste | Aktion |
|---|---|
| F1 | QSO form und Recent QSOs |
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
| Ctrl+L | Aktives Logbuch wechseln |
| Ctrl+R | Aktives Rig wechseln |
| Ctrl+C | Aktiven Contest wechseln |
| Ctrl+O | Aktiven Operator wechseln |
| Esc | Zurück zum vorherigen Bildschirm |

### QSO form

| Taste | Aktion |
|---|---|
| Tab | Nächstes Feld |
| Shift+Tab | Vorheriges Feld |
| ↑ / ↓ | In Spalte bewegen |
| Enter | QSO speichern, ggf. mit Duplikatbestätigung |
| Ctrl+S | QSO aus jedem Feld speichern |
| Del | Alle Formularfelder löschen |
| Ins | Lookup: QRZ, Wavelog, DXCC und duplicate check |
| PgUp / PgDn | Band, mode oder submode wechseln |
| Ctrl+D | Spot dialog öffnen |
| Ctrl+T | Keep Comment umschalten |
| Alt+, | Rotor-Azimuth −5° |
| Alt+. | Rotor-Azimuth +5° |
| Alt+; | Rotor-Elevation +5° |
| Alt+' | Rotor-Elevation −5° |
| Alt+\ | Rotor auf Bearing vom eigenen Grid zum Partnergrid ausrichten |
| Alt+/ | Rotor stoppen |
| Alt+0–9 | Favorite laden |
| Alt+Shift+0–9 | aktuelle frequency, mode und band als Favorite speichern |

### Logbook Editor

| Taste | Aktion |
|---|---|
| ↑ / ↓ | Durch Zeilen navigieren |
| PgUp / PgDn | Vorherige oder nächste Seite |
| Home / End | Erste oder letzte Zeile |
| Enter / e | Ausgewähltes QSO bearbeiten |
| Delete | Ausgewähltes QSO löschen |
| p | Alle QSOs purgen |
| Ctrl+C | Contest-Filter wechseln |
| Ctrl+E | ADIF exportieren |
| Ctrl+I / Tab | ADIF importieren |
| w | Nicht gesendete QSOs zu Wavelog hochladen |
| Ctrl+W | Kontakte von Wavelog herunterladen |
| Esc / F6 | Editor schließen und zum QSO form zurückkehren |

### DX Cluster

| Taste | Aktion |
|---|---|
| ↑ / ↓ | Durch Spots navigieren |
| Enter | QSO form füllen, Rig tunen und zu QSO zurückkehren |
| Space | Rig auf ausgewählten Spot tunen und auf DX Cluster bleiben |
| Home | Bandfilter vorwärts wechseln |
| End | Bandfilter rückwärts wechseln |
| `\` | Kontinentfilter wechseln |
| Ins | Modefilter vorwärts wechseln |
| Del | Modefilter rückwärts wechseln |
| PgUp | Zeitfilter vorwärts wechseln |
| PgDn | Zeitfilter rückwärts wechseln |
| Backspace | Alle Filter löschen |
| Esc / F4 | Zum QSO form zurückkehren |

### Partner view

| Taste | Aktion |
|---|---|
| F2 | Partner view → Photo → Back wechseln |
| Esc / F1 | Zum QSO form zurückkehren |

---

## Fehlerbehebung

### CQOps startet nicht

Prüfen:

- Terminalgröße mindestens 80×24,
- Windows-Nutzer verwenden Windows Terminal,
- Netzwerkstart blockiert nicht; testen mit:

  ```bash
  cqops --offline
  ```

Logs prüfen:

| Plattform | Log-Pfad |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### Rig verbindet sich nicht

Für flrig:

- prüfen, ob flrig läuft,
- Port im aktiven Rig-Preset prüfen,
- Standardport ist `12345`.

Für Hamlib:

- prüfen, ob `rigctld` läuft,
- Host und Port prüfen,
- prüfen, ob Radio/Backend die angefragten Daten unterstützt.

Statuslabels helfen bei der Diagnose:

| Farbe | Bedeutung |
|---|---|
| Weiß/Standard | Verbunden |
| Gelb | Deaktiviert oder verbindet |
| Rot | Fehlgeschlagen |

Reconnect-Toasts können unterdrückt sein. CQOps kann still erneut versuchen.

### WSJT-X loggt nicht automatisch

Prüfen:

- WSJT-X **Settings → Reporting → UDP Server**,
- UDP-Host und Port passen zum aktiven Rig-Preset in CQOps,
- WSJT-X 2.6 oder neuer wird verwendet,
- WSJT-Statuslabel ist aktiv,
- aktives Logbuch ist korrekt,
- aktiver Operator ist korrekt.

### Wavelog-Upload schlägt fehl

Prüfen:

- Wavelog URL,
- API key,
- station profile ID,
- **WL**-Statuslabel.

Uploadfehler werden als Toasts angezeigt. QSOs bleiben lokal gespeichert, auch wenn der Upload fehlschlägt. Fehler einzelner QSOs blockieren nicht den Rest des Batches.

### Probleme mit config file

Config file:

| Plattform | Pfad |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets file:

```text
secrets.enc
```

Die Secrets-Datei liegt im gleichen Verzeichnis wie `config.yaml`.

Wenn die Konfiguration beschädigt ist, Datei verschieben oder löschen und CQOps neu starten. Der Setup Wizard erstellt eine frische Konfiguration.

Das Feld `last_fetched_id` erscheint erst nach einem erfolgreichen Wavelog-Download.

### Performance-Probleme

Versuchen:

- map rendering in General settings deaktivieren,
- Solar panel deaktivieren, wenn nicht benötigt,
- netzwerklastige Bildschirme wie DX Cluster und PSK Reporter offline meiden,
- `cqops --offline` verwenden, wenn das Netzwerk unzuverlässig ist.

---

## Fehler melden

Vor dem Melden eines Fehlers:

1. **Debug mode** in **F9 → General → Debug** aktivieren oder setzen:

   ```yaml
   debug: true
   ```

   in `config.yaml`.

2. Problem reproduzieren.
3. Relevantes Log anhängen.

Fehler auf GitHub melden:

<https://github.com/szporwolik/cqops/issues>

Bitte beifügen:

- CQOps-Version aus `cqops --version`,
- Betriebssystem,
- Terminal-Emulator,
- Schritte zur Reproduktion,
- relevantes Debug-Log.
