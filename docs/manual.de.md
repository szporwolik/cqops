---
title: CQOps-Benutzerhandbuch
description: Vollständige Anleitung zur Installation, Konfiguration und Nutzung von CQOps — einem schnellen, terminalorientierten Amateurfunk-Logger
---

> **Hinweis:** Diese Übersetzung wurde mit einem LLM-Modell erstellt. Korrekturen sind willkommen — bitte als Pull Request gegen den `dev`-Branch einreichen.

# CQOps-Benutzerhandbuch

## Inhaltsverzeichnis

1. [Über CQOps](#über-cqops)
2. [Download & Installation](#download--installation)
3. [Erster Start — Einrichtungsassistent](#erster-start--einrichtungsassistent)
4. [Schnellstart: Erstes QSO loggen](#schnellstart-erstes-qso-loggen)
5. [Hauptbildschirm-Übersicht](#hauptbildschirm-übersicht)
6. [Typische Arbeitsabläufe](#typische-arbeitsabläufe)
7. [Kernfunktionen](#kernfunktionen)
8. [Integrationen](#integrationen)
9. [Konfigurationsreferenz](#konfigurationsreferenz)
10. [Tastenkürzel](#tastenkürzel)
11. [Fehlerbehebung](#fehlerbehebung)

---

## Über CQOps

CQOps ist ein schneller, terminalorientierter Amateurfunk-Logger für Operator, die Geschwindigkeit, Zuverlässigkeit und geringe Systemlast benötigen — im Shack, auf dem Gipfel, beim Field Day oder an einer gemeinsam genutzten Clubstation.

**Offline-first.** Lokales QSO-Logging benötigt keinen Internetzugang. Zwischengespeicherte Referenzdaten, Solardaten und DXCC-Präfixe bleiben nach dem ersten Download verfügbar. Netzwerkintegrationen wie Wavelog, QRZ.com, DX Cluster und PSK Reporter benötigen eine Verbindung und werden im Modus `--offline` übersprungen.

**Für den Feldeinsatz gebaut.** CQOps ist QRP-tauglich, SOTA/POTA-freundlich und läuft problemlos auf Rechnern der Raspberry-Pi-Klasse, alten Laptops und Systemen ohne Desktop-Umgebung.

**Bereit für Clubstationen.** CQOps unterstützt mehrere Logbücher, Operator-Profile und Rig-Presets. Aktives Logbuch, aktiver Operator oder aktives Rig lassen sich mit einem einzigen Tastendruck wechseln.

**Portabel von Grund auf.** CQOps ist ein einzelnes in Go geschriebenes Binary. Es hat keine CGO-Abhängigkeit und benötigt keine Systemdienste.

**Plattformübergreifend.** Windows, Linux und macOS werden auf amd64 und arm64 unterstützt.

### Für wen CQOps geeignet ist

- Portable Operator, die schnelles Tastatur-Logging auf Hardware mit geringer Leistungsaufnahme benötigen.
- SOTA- und POTA-Aktivierer, die offline loggen und später hochladen.
- Clubstationen mit mehreren Operatoren an derselben Station.
- Field-Day-Teams mit gemeinsam genutzten Rechnern oder Raspberry Pi-Hardware.
- Operator, die einen Terminal-Workflow einer Desktop-GUI vorziehen.

CQOps ist nicht als Ersatz für vollwertige Desktop-Logger oder webbasierte Logbuch-Plattformen gedacht. Der Fokus liegt auf schnellem Terminal-Logging, Feldeinsatz, Offline-Betrieb und Shared-Station-Workflows.

---

## Download & Installation

> [Alle Releases durchsuchen →](https://github.com/szporwolik/cqops/releases)

### Windows

| Paket | Link | Hinweise |
|---------|------|-------|
| **Installer** | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Empfohlen für die meisten Nutzer. Fügt CQOps zum Startmenü und PATH hinzu. |
| Portables ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Entpacken und ohne Installation ausführen. |

### Linux — Debian / Ubuntu

| Architektur | Link | Für |
|-------------|------|---------|
| **amd64** | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Die meisten Intel/AMD-PCs |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-Bit-ARM-Systeme |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-Bit Raspberry Pi OS |

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — portables Tar-Archiv

| Architektur | Link | Für |
|-------------|------|---------|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Die meisten Intel/AMD-PCs |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-Bit-ARM-Systeme |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-Bit Raspberry Pi OS |

### macOS

| Architektur | Link | Für |
|-------------|------|---------|
| **Apple Silicon** | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | M1/M2/M3 Macs |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Intel Macs |

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Aus dem Quellcode

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build        # Nur bauen; Binary wird in build/ geschrieben
make install      # Bauen und systemweit installieren
```

Quellcode-Builds benötigen Go 1.26 oder neuer.

### Voraussetzungen

- Terminalgröße: mindestens 80×24 Zeichen.
- Empfohlene Terminalgröße: 80×43 oder größer.
- Ein moderner Terminal-Emulator wird empfohlen. Unter Windows Windows Terminal anstelle der Legacy-Konsole verwenden.

### Kommandozeilenoptionen

```bash
cqops              # TUI starten
cqops --offline    # Ohne Netzwerkaktivität starten
cqops --version    # Version anzeigen und beenden
cqops --help       # Hilfe anzeigen
```

---

## Erster Start — Einrichtungsassistent

Beim ersten Start öffnet CQOps einen Setup-Wizard für die grundlegenden Stationseinstellungen. Netzwerkintegrationen können übersprungen werden; lokales Logging funktioniert ohne sie.

1. **Station & Logbuch**   
   Konfigurieren Sie das erste Logbuch, Stationsrufzeichen, Operator und Grid Locator. Optionale Felder umfassen SOTA/POTA/WWFF-Referenzen, IARU-Region, CQ/ITU-Zone, DXCC und SIG/SIG Info. Die Wavelog-Einrichtung ist hier ebenfalls verfügbar: URL, API-Key, Stationsprofil-ID, Update und Test.

2. **Rig**   
   Konfigurieren Sie ein Rig-Preset: Name, Modell, Antenne, Leistung und Rig-Backend. Unterstützte Backends sind None, flrig und Hamlib rigctld. Optionale Einstellungen umfassen Hamlib-Rotorsteuerung und WSJT-X UDP-Integration.

3. **Integrationen**   
   Konfigurieren Sie QRZ.com-Callbook-Lookup: Aktivierungsoption, Benutzername, maskiertes Passwort und Test.

4. **Allgemein**   
   Wählen Sie die IANA-Zeitzone. CQOps erkennt standardmäßig die Systemzeitzone und bietet eine scrollbare Liste.

5. **Zusammenfassung**   
   Überprüfen Sie die Konfiguration. Drücken Sie **Ctrl+S** zum Speichern und Starten von CQOps.

**Assistent-Navigation:** **Ctrl+S** geht nach Validierung weiter. **Esc** geht zurück. **F10** beendet. Die Leertaste schaltet Kontrollkästchen um. Tab und Shift+Tab bewegen zwischen Feldern.

Alle Assistent-Einstellungen können später über das Konfigurationsmenü mit **F9** geändert werden.

---

## Schnellstart: Erstes QSO loggen

1. **CQOps installieren und starten.**   
   Laden Sie das Paket für Ihre Plattform herunter, starten Sie `cqops` und durchlaufen Sie den Setup-Wizard mit mindestens Rufzeichen und Grid Locator.

2. **QSO Form verwenden.**   
   Das QSO Form öffnet sich mit **F1**. Geben Sie ein Rufzeichen ein; CQOps wandelt es automatisch in Großbuchstaben um. Wenn das aktive Rig über flrig oder Hamlib verbunden ist, werden Frequenz, Band, Mode und Submode automatisch ausgefüllt. Datum und Uhrzeit werden auf aktuelle UTC gesetzt.

3. **Durch Felder bewegen.**   
   Verwenden Sie **Tab**, **Shift+Tab** und **↑/↓**.

4. **QSO speichern.**   
   Drücken Sie **Enter** oder **Ctrl+S**. Wenn eine **DUPE!**-Warnung erscheint, drücken Sie **Enter** erneut zum trotzdem Speichern oder **Esc** zum Abbrechen.

Das neue QSO erscheint sofort in der Recent QSOs-Tabelle unter dem Formular.

---

## Hauptbildschirm-Übersicht

```text
┌─ Status Bar ──────────────────────────────────────────────────────────────────┐
│  CQOps v0.8.8  Log Portable  Rig FTDx10  Call SP9MOA/P                          │
│  Net WSJT Hamlib DXC WL                                            23:00L 2100Z │
├─ Tab Bar ─────────────────────────────────────────────────────────────────────┤
│ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮         │
│ │F1 QSO│ │F2 QRZ│ │F4 DXC│ │F5 HRD│ │F6 REF│ │F7 BPL│ │F8 LOG│ │F9 CFG│         │
├─ Main Content Areasbereich ────────────────────────────────────────────────────────────┤
│                                                                                  │
│  QSO Form, Tabelle, Karte, Editor oder aktiver Bildschirminhalt              │
│                                                                                  │
├─ Help Bar ────────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Status Bar

Die Status Bar zeigt die CQOps-Version, das aktive Logbuch, das aktive Rig, das Stationsrufzeichen und den aktiven Operator. Rechts werden Integrationsstatus-Labels und Uhrzeit in lokal (`L`) und UTC (`Z`) angezeigt.

**Label-Farben:**

| Farbe | Bedeutung |
|-------|-----------|
| Weiß/Standard | Verbunden oder aktiv |
| Gelb | Deaktiviert, Verbindungsaufbau oder erwartet offline |
| Rot | Fehler oder getrennt |
| Akzent + fett | WSJT-X sendet |

Mögliche Labels sind **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC** und **WL**.

### Tab Bar

| Taste | Tab | Bildschirm |
|-----|-----|--------|
| F1 | QSO | QSO Form und letzte QSOs |
| F2 | QRZ | Partner View: Callbook-Daten, Karte, Statistik, Foto |
| F4 | DXC | DX Cluster Spots und Filter |
| F5 | HRD | PSK Reporter Spots und Ausbreitungskarte |
| F6 | REF | SOTA/POTA/WWFF/IOTA-Referenzsuche |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, Wavelog-Sync |
| F9 | CFG | Konfigurationsmenüs |

### Help Bar

Die untere Zeile zeigt die wichtigsten Tastenkürzel für den aktiven Bildschirm. Drücken Sie **?** für die vollständige Hilfe-Overlay.

---

## Typische Arbeitsabläufe

### Portabelbetrieb / SOTA / POTA

1. **Vor dem Verlassen des Hauses** CQOps einmal mit Internetzugang starten. Dadurch kann CQOps Caches wie Solardaten, REF-Daten und DXCC-Präfixe füllen.
2. **Cache vor dem Offline-Gehen überprüfen.** Prüfen Sie, ob das Solar Panel Daten anzeigt und die REF-Suche unter **F6** Ergebnisse liefert.
3. **Im Feld** CQOps mit `cqops --offline` starten. Netzwerkaktivität wird übersprungen, was Verzögerungen durch nicht erreichbare Dienste vermeidet.
4. **Normal loggen.** Lokales Logging funktioniert ohne Internet.
5. **Später hochladen.** Wenn Sie wieder online sind, öffnen Sie den Logbook Editor mit **F8** und drücken Sie **w**, um nicht gesendete QSOs an Wavelog hochzuladen.

### Gemeinsame Clubstation & Hot-Seat-Betrieb

1. **Operator-Profile hinzufügen:** öffnen Sie **F9 → Operators**, dann drücken Sie **Ins** für jeden Operator. Geben Sie Rufzeichen und Name ein.
2. **Aktiven Operator wechseln:** im QSO Form **Ctrl+O** drücken. Der aktive Operator wird in der Status Bar angezeigt und im `OPERATOR`-Feld gespeicherter QSOs geschrieben.
3. **Hot-Seat-Logging verwenden:** Operator A loggt ein QSO, Operator B drückt **Ctrl+O**, loggt dann unter eigenem Operator-Profil.
4. **Retain bei Bedarf verwenden:** aktivieren Sie **Retain**, wenn mehrere Operatoren denselben Kontakt loggen müssen, ohne das Formular neu auszufüllen.

Vor dem Speichern an einer gemeinsam genutzten Station das aktive Logbuch und den aktiven Operator in der Status Bar prüfen.

### Privates + Club-Logbuch

Viele Funkamateure führen ein persönliches Logbuch und ein oder mehrere Club-Logbücher.

1. **Logbücher erstellen:** öffnen Sie **F9 → Logbooks**, dann drücken Sie **Ins** für jedes Logbuch.
2. **Aktives Logbuch wechseln:** **Ctrl+L** im QSO Form drücken. Die Status Bar zeigt das aktive Logbuch.
3. **Stationsdaten getrennt halten:** jedes Logbuch kann eigenes Stationsrufzeichen, Wavelog-Einstellungen, Contest-Einstellungen und Operatoren haben.
4. **Schnelles Dual-Logging:** **Retain** aktivieren, QSO in einem Logbuch speichern, **Ctrl+L** drücken, dann im anderen Logbuch erneut speichern, falls angemessen.

### Mehrere Rigs

1. **Rig-Presets erstellen:** öffnen Sie **F9 → Rigs**, dann drücken Sie **Ins** für jedes Rig.
2. **Backend einstellen:** flrig oder Hamlib für CAT-gesteuerte Rigs verwenden. None für manuell abgestimmte Rigs.
3. **Aktives Rig wechseln:** **Ctrl+R** im QSO Form drücken.
4. **Gemischte Stationen betreiben:** z.B. ein CAT-gesteuertes KW-Rig und ein manuelles VHF/UHF-Rig in derselben Sitzung.
5. **WSJT-X pro Rig konfigurieren:** jedes Rig-Preset kann eigene WSJT-X UDP-Einstellungen haben.

Wenn das aktive Rig CAT-Steuerung hat, kann CQOps Frequenz, Band, Mode und Submode automatisch ausfüllen. Für manuelle Rigs geben Sie diese selbst ein.

### FT8 / WSJT-X Auto-Logging

Wenn WSJT-X über UDP verbunden ist, kann CQOps digitale QSOs automatisch aus WSJT-X ADIF-Nachrichten loggen.

- Automatisch geloggte QSOs werden im aktiven Logbuch gespeichert.
- Doppelte automatisch geloggte QSOs werden übersprungen.
- Automatisch geloggte QSOs erben die aktive Contest-ID.
- QSOs erscheinen sofort in den letzten QSOs.
- Wenn Wavelog konfiguriert und erreichbar ist, können automatisch geloggte QSOs automatisch hochgeladen werden.
- Wenn der WSJT-X-Operator nicht mit dem aktiven Operator übereinstimmt, zeigt CQOps eine Warnung.

Prüfen Sie das aktive Logbuch, den aktiven Operator und den aktiven Contest vor langen Digitalsitzungen.

### Wavelog-Sync

Wavelog-Sync ist optional. CQOps speichert QSOs immer zuerst lokal.

**Upload:** **w** im Logbook Editor (**F8**) drücken. CQOps lädt nicht gesendete QSOs in Chargen von 50 hoch und verfolgt den Status pro QSO: nicht gesendet, gesendet oder Fehler.

**Download:** **Ctrl+W** im Logbook Editor drücken. Downloads sind inkrementell. CQOps holt QSOs, die neuer als die gespeicherte `last_fetched_id` für das aktive Logbuch sind. Doppelte werden übersprungen.

Wenn ein Wavelog-Upload fehlschlägt, bleibt das QSO im lokalen Logbuch und kann später erneut versucht werden. Das Leeren eines Logbuchs setzt die Fetch-ID auf `0` zurück, was einen vollständigen Neu-Download ermöglicht.

---

## Kernfunktionen

### QSO-Logging

Das QSO Form (**F1**) ist der primäre Logging-Bildschirm. Es verwendet ein dreispaltiges Layout und kann Felder aus Rig-Steuerung, QRZ.com, Wavelog-Lookup, DXCC/Präfix-Daten und REF-Datenbanken automatisch ausfüllen.

**Formularfelder:**

| Linke Spalte | Mittlere Spalte | Rechte Spalte |
|-------------|---------------|--------------|
| Date UTC | Mode **(▼)** | Power W |
| Time UTC | Submode **(▼)** | Freq RX |
| Call | Name | SOTA Ref |
| RST sent | QTH | POTA Ref |
| RST rcvd | Grid | WWFF Ref |
| Frequency MHz | Country | IOTA |
| Band **(▼)** | SIG | SIG Info |
| Exch sent ⚠️ | | |
| Exch rcvd ⚠️ | | |

⚠️ Austauschfelder erscheinen nur, wenn ein Contest aktiv ist. Mit **(▼)** markierte Felder werden mit **PgUp/PgDn** durchgeschaltet.

Die untere Zeile enthält:

- **Comment** (Kommentar)
- **Keep** — bewahrt das Comment-Feld zwischen QSOs; Umschalten mit **Ctrl+T**
- **Retain** — bewahrt das gesamte Formular nach dem Speichern

Die Pfad/Peilung-Zeile zeigt Entfernung und Azimut, wenn beide Grid Locator bekannt sind. Sie kann auch Badges wie **DUPE!**, **New Call!** und **New DXCC!** anzeigen.

### Auto-Fill-Quellen

| Quelle | Felder |
|--------|--------|
| flrig / Hamlib | Frequency, Freq RX bei Split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| REF-Datenbank | SOTA, POTA, WWFF, IOTA-Referenzen |
| Wavelog-Lookup | Worked/Confirmed-Status wenn konfiguriert |

### Contest-Logging

Contests fügen Austauschfelder und Seriennummern-Behandlung zum QSO Form hinzu.

Erstellen oder konfigurieren Sie einen Contest im Logbook Editor (**F8**) mit **Ins**. Legen Sie Contest-Name, Datum, ADIF-Contest-ID und Austauschvorlagen fest.

Unterstützte Vorlagen-Marker:

| Marker | Ersetzt durch |
|--------|---------------|
| `@rst` | RST gesendet oder empfangen |
| `@serial` | Automatisch inkrementierende Seriennummer |
| `@call` | Ihr Rufzeichen |
| `@grid` | Ihr Grid Locator |
| `@name` | Operator-Name aus dem Operator-Profil |

Drücken Sie **Ctrl+C**, um den aktiven Contest zu wechseln. Wenn ein Contest aktiv ist:

- zeigt das QSO Form Austauschfelder,
- Seriennummern werden automatisch inkrementiert,
- Letzte QSOs können auf Contest-QSOs gefiltert werden,
- ADIF-Export bewahrt `CONTEST_ID`.

### Logbook Editor

Der Logbook Editor (**F8**) dient der QSO-Verwaltung, ADIF-Import/Export, Wavelog-Sync und contestbezogenen Operationen.

**Inline-Bearbeitung:** Zeile mit **↑/↓** auswählen, **Enter** oder **e** drücken, QSO bearbeiten, dann mit **Ctrl+S** speichern. Änderungen werden sofort in den letzten QSOs reflektiert.

### ADIF-Import & -Export

CQOps unterstützt ADIF 3.1.7 Import und Export.

- **Ctrl+I** importiert eine ADIF-Datei, validiert Datensätze, überspringt Duplikate und zeigt eine Zusammenfassung.
- **Ctrl+E** exportiert QSOs. Export kann alle QSOs oder contest-gefilterte QSOs umfassen.
- Importierte QSOs werden für Wavelog-Upload markiert, wenn Wavelog-Sync konfiguriert ist.

### Favoriten

Favoriten speichern Frequenz-, Mode- und Band-Presets in 10 Slots.

| Tastenkürzel | Aktion |
|----------|--------|
| Alt+0–9 | Favoriten-Slot abrufen |
| Alt+Shift+0–9 | Aktuelle Frequenz/Mode/Band in Slot speichern |

Favoriten werden in der Konfiguration gespeichert und sind logbuchübergreifend nutzbar.

Beispiel: Für ein polnisches SOTA-FM-Anrufsetup geben Sie `145.55` ein, setzen Mode `FM`, setzen Band `2m`, dann drücken Sie **Alt+Shift+1**. Später drücken Sie **Alt+1** zum Abrufen.

### REF-Suche

Der REF-Bildschirm (**F6**) durchsucht SOTA-, POTA-, WWFF- und IOTA-Referenzen. Suche nach Präfix, Name oder Referenzbezeichner. Ausgewählte Referenzen können das QSO Form füllen.

### Band Plan Browser

Der Band Plan Browser (**F7**) bietet schnellen Zugriff auf Amateurbänder, VHF/UHF-Bereiche, CB, PMR446 und Broadcast-Presets. Eine ausgewählte Frequenz kann zum Abstimmen des aktiven Rigs verwendet werden. Bandplandaten können auch als Markdown exportiert werden.

---

## Integrationen

### QRZ.com

QRZ.com-Lookup erfordert Internetzugang und ein QRZ XML-Abonnement.

Drücken Sie **Ins** im QSO Form, um Callbook-Felder wie Name, QTH, Grid, Land, CQ/ITU-Zonen, DXCC und Kontinent zu füllen. Die Partner View (**F2**) kann das Operator-Foto anzeigen, wenn verfügbar.

### Wavelog

Wavelog-Integration erfordert Internetzugang. Sie unterstützt Upload, inkrementellen Download und Worked/Confirmed-Lookup.

Wavelog wird pro aktivem Logbuch mit URL, API-Key und Stationsprofil-ID konfiguriert. CQOps speichert QSOs immer zuerst lokal; Wavelog-Upload-Fehler führen nicht zu Datenverlust.

Siehe [Wavelog-Sync](#wavelog-sync).

### flrig

flrig-Integration verwendet XML-RPC über HTTP. Der Standard-Endpunkt ist `localhost:12345`.

CQOps kann Frequenz, Mode und Leistung von flrig auslesen. Split-Betrieb wird als VFO A auf Frequency und VFO B auf Freq RX abgebildet.

### Hamlib / rigctld

Hamlib-Rig-Steuerung verwendet den `rigctld` TCP-Daemon. CQOps kann Frequenz, Mode, VFO, Split und Leistung je nach Funkgerät-Unterstützung abfragen.

Einige Funkgeräte oder Hamlib-Backends unterstützen nicht jede Abfrage. CQOps behandelt fehlende VFO-Namensunterstützung nach Möglichkeit.

### Hamlib Rotor / rotctld

Rotorsteuerung verwendet Hamlib `rotctld`. CQOps unterstützt Azimut, Elevation und Stop-Befehle.

Nützliche Tastenkürzel:

| Tastenkürzel | Aktion |
|----------|--------|
| Ctrl+←/→ | Azimut um 5° anpassen |
| Ctrl+↑/↓ | Elevation um 5° anpassen |
| Ctrl+A | Rotor auf berechnete Pfadpeilung ausrichten |
| Ctrl+F1 | Rotor stoppen |

### WSJT-X

WSJT-X-Integration verwendet UDP-Nachrichten von WSJT-X. CQOps parst ADIF-Nachrichten und kann abgeschlossene QSOs automatisch loggen.

Das Rig-Label wird akzentfarben, während WSJT-X sendet. Wenn der von WSJT-X gemeldete Operator nicht mit dem aktiven Operator übereinstimmt, zeigt CQOps eine Warnung.

Siehe [FT8 / WSJT-X Auto-Logging](#ft8--wsjt-x-auto-logging).

### DX Cluster

DX Cluster-Integration verwendet eine Telnet-Verbindung und erfordert Internetzugang. Der Standard-Server ist `dxspots.com:7300`.

Filter umfassen Band, Kontinent, Mode und Alter/Zeit. Drücken Sie **Enter** auf einem Spot, um das QSO Form zu füllen, das aktive Rig abzustimmen und zum QSO-Bildschirm zurückzukehren. Drücken Sie **Space** zum Abstimmen ohne das Formular zu füllen. Drücken Sie **Backspace**, um Filter zu löschen.

### PSK Reporter

PSK Reporter-Integration erfordert Internetzugang. Sie bietet Ausbreitungs-Spots, Band/Zeit/Mode-Filter und eine ASCII-Weltkarte auf **F5**.

### Solardaten

Solardaten umfassen SFI, Sonnenfleckenzahl, A/K-Indizes und bandweise Bedingungen von hamqsl.com. Live-Updates erfordern Internetzugang. Zwischengespeicherte Daten bleiben nach erfolgreichem Abruf offline verfügbar.

### CQOps Live — Browser-Dashboard

CQOps Live ist ein eingebautes Web-Dashboard, das Ihre Stationsaktivität in Echtzeit in jedem Browser anzeigt — perfekt für Field-Day-Präsentationen, Clubstationsbildschirme, Contest-Überwachung oder um die Station aus einem anderen Raum im Auge zu behalten.

**Aktivierung**

1. Drücken Sie **F9**, um das Hauptmenü zu öffnen, und wählen Sie **Integrations**.
2. Scrollen Sie zum Bereich **HTTP Server** und aktivieren Sie **Enable HTTP server**.
3. Optional: Legen Sie die Adresse (Standard `0.0.0.0`) und den Port (Standard `8073`) fest.
4. Drücken Sie **Ctrl+S** zum Speichern. Der Server startet sofort.
5. Öffnen Sie `http://localhost:8073` (oder die konfigurierte Adresse) in einem beliebigen Browser.

**Was das Dashboard anzeigt**

Das Dashboard hat zwei Modi, die automatisch wechseln:

- **Übersichtsmodus** (kein aktives Rufzeichen): eine Live-Leaflet-Karte mit heutigen QSO-Markern und Großkreis-Pfaden, eine Tabelle der letzten QSOs, Stationsinfo, Statistiken, Top-Operatoren und QSOs mit der größten Entfernung.
- **Aktiv / Now-Working-Modus** (Rufzeichen in Bearbeitung): eine prominente Rufzeichenanzeige, QRZ-Foto (falls verfügbar), Band-/Mode-Badges, DUPE/NEW-CALL/NEW-DXCC-Indikatoren, Entfernung und Richtung sowie eine hervorgehobene gestrichelte Linie auf der Karte von Ihrer Station zum Partnerstandort.

Alle Panels werden in Echtzeit über Server-Sent Events (SSE) aktualisiert — kein Seiten-Reload nötig.

**Anpassung**

Im HTTP-Server-Integrationsformular können Sie konfigurieren:

| Feld | Beschreibung |
|-------|-------------|
| Header 1 | Haupttitel in der Kopfzeile und im Hero-Bereich. Fällt auf "CQOps Live" zurück. |
| Header 2 | Untertitel unter dem Titel. Fällt auf "Fast, portable ham radio logger" zurück. |
| Logo URL | Eine öffentlich zugängliche Bild-URL, die oben links angezeigt wird. Fällt auf das CQOps-Logo zurück. |
| Event Start | Ein Datum im Format `YYYY-MM-DD`. Wenn gesetzt, werden Statistiken und QSO-Listen ab diesem Datum gefiltert — nützlich für mehrtägige Veranstaltungen. |

**Leistung**

Das Dashboard ist für stromsparende Hardware ausgelegt. Der Browser übernimmt alle Kartenberechnungen, Entfernungen und Statistiken. Die CQOps-Terminal-App sendet nur schlanke JSON-Updates via SSE. Wenn der HTTP-Server deaktiviert ist, entsteht kein Overhead.

**Typische Anwendungsfälle**

- **Field Day / Contest Public Display**: großer Bildschirm oder Projektor für Live-Karte und letzte QSOs.
- **Clubstation-Infoscreen**: eigener Monitor mit Stationsaktivität für Besucher.
- **Fernüberwachung**: Dashboard auf Tablet oder Handy, um die Station aus einem anderen Raum zu beobachten.
- **Messe-/Eventstand**: Header 1/2 und Club-Logo für eine professionelle Darstellung konfigurieren.

---

## Konfigurationsreferenz

Die CQOps-Konfiguration wird gespeichert in:

| Plattform | Konfigurationspfad |
|----------|-------------|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Vertrauliche Zugangsdaten werden separat in `secrets.enc` im selben Konfigurationsverzeichnis gespeichert. Zugangsdaten sind mit einem maschinengebundenen Schlüssel verschlüsselt, daher müssen Zugangsdaten bei Übertragung auf einen anderen Rechner neu eingegeben werden.

Konfiguration mit **F9** öffnen.

| Menü | Konfiguriert |
|------|------------|
| Station | Rufzeichen, Grid, CQ/ITU-Zone, IARU-Region, Referenzen |
| Rig | Rig-Presets, Modell, Antenne, Leistung, Backend, Rotor, WSJT-X |
| Wavelog | URL, API-Key, Stationsprofil-ID |
| QRZ | Benutzername und Passwort |
| DX Cluster | Host, Port, Login |
| Operators | Operator-Profile: Rufzeichen und Name |
| Logbooks | Stations-, Wavelog-, Contest- und Operator-Einstellungen pro Logbuch |
| Notifications | Toast- und Benachrichtigungsverhalten |
| General | Zeitzone, Entfernungseinheiten, Karte, Debug-Modus |

### Multi-Logbuch

Verwenden Sie mehrere Logbücher für Heim-, Portabel-, Contest- und Clubbetrieb. Drücken Sie **Ctrl+L**, um das aktive Logbuch zu wechseln. Jedes Logbuch hat eigene Stationsdaten, Wavelog-Einstellungen, Contest-Einstellungen und Operator-Einstellungen.

### Multi-Operator

Operator-Profile enthalten Operator-Rufzeichen und Name. Drücken Sie **Ctrl+O**, um den aktiven Operator zu wechseln. Der aktive Operator wird im ADIF-`OPERATOR`-Feld gespeichert und bei Wavelog-Uploads verwendet.

### Multi-Rig

Rig-Presets speichern Backend, Modell, Antenne, Leistung, Rotor und WSJT-X-Einstellungen. Drücken Sie **Ctrl+R**, um das aktive Rig zu wechseln.

### Verschlüsselte Zugangsdaten

Seit v0.8.7 werden Zugangsdaten verschlüsselt gespeichert.

- **Zugangsdaten-Datei:** `secrets.enc`
- **Speicherort:** selbes Verzeichnis wie `config.yaml`
- **Unix-Berechtigungen:** `0600` wo unterstützt
- **Verschlüsselung:** AES-256-GCM mit maschinengebundenem Schlüssel
- **Geschützte Daten:** QRZ-Passwort, DX Cluster-Login, Wavelog API-Keys
- **Migration:** Klartext-Zugangsdaten aus älteren Konfigurationen werden beim ersten Start migriert
- **Wiederherstellung:** wenn `secrets.enc` beschädigt ist, startet CQOps mit einer Warnung und fordert zur Neueingabe der Zugangsdaten auf

---

## Tastenkürzel

### Global

| Taste | Aktion |
|-----|--------|
| F1 | QSO Form und letzte QSOs |
| F2 | Partner View |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | REF-Suche |
| F7 | Band Plan Browser |
| F8 | Logbook Editor |
| F9 | Konfiguration / Hauptmenü |
| F10 | Beenden |
| Ctrl+F9 | Log-Viewer |
| ? | Hilfe-Overlay |
| Ctrl+L | Aktives Logbuch wechseln |
| Ctrl+R | Aktives Rig wechseln |
| Ctrl+C | Aktiven Contest wechseln |
| Ctrl+O | Aktiven Operator wechseln |
| Esc | Zurück zum vorherigen Bildschirm |

### QSO Form — F1

| Taste | Aktion |
|-----|--------|
| Tab | Nächstes Feld |
| Shift+Tab | Vorheriges Feld |
| ↑ / ↓ | Innerhalb der Spalte bewegen |
| Enter | QSO speichern, mit Dupe-Bestätigung falls nötig |
| Ctrl+S | QSO von beliebigem Feld aus speichern |
| Del | Alle Formularfelder löschen |
| Ins | Lookup: QRZ, Wavelog, DXCC und Dupe-Prüfung |
| PgUp / PgDn | Band, Mode oder Submode durchschalten |
| Ctrl+D | Spot-Dialog öffnen |
| Ctrl+T | Keep Comment umschalten |
| Ctrl+←/→ | Rotor-Azimut um 5° anpassen |
| Ctrl+↑/↓ | Rotor-Elevation um 5° anpassen |
| Ctrl+A | Rotor auf Peilung von eigenem Grid zum Partner-Grid ausrichten |
| Ctrl+F1 | Rotor stoppen |
| Alt+0–9 | Favoriten-Slot abrufen |
| Alt+Shift+0–9 | Aktuelle Frequenz/Mode/Band in Favoriten-Slot speichern |

### Logbook Editor — F8

| Taste | Aktion |
|-----|--------|
| ↑ / ↓ | Zeilen navigieren |
| PgUp / PgDn | Vorherige oder nächste Seite |
| Home / End | Erste oder letzte Zeile |
| Enter / e | Ausgewähltes QSO bearbeiten |
| Delete | Ausgewähltes QSO löschen |
| p | Alle QSOs löschen |
| Ctrl+C | Contest-Filter umschalten |
| Ctrl+E | ADIF exportieren |
| Ctrl+I / Tab | ADIF importieren |
| w | Nicht gesendete QSOs an Wavelog hochladen |
| Ctrl+W | Kontakte von Wavelog herunterladen |
| Esc / F6 | Editor schließen, zurück zu QSO |

### DX Cluster — F4

| Taste | Aktion |
|-----|--------|
| ↑ / ↓ | Spots navigieren |
| Enter | Formular füllen + Rig abstimmen + zu QSO gehen |
| Space | Rig auf Spot abstimmen (auf DXC bleiben) |
| Home | Bandfilter vorwärts |
| End | Bandfilter rückwärts |
| \\ | Kontinentfilter |
| Ins | Modefilter vorwärts |
| Del | Modefilter rückwärts |
| PgUp | Zeitfilter vorwärts |
| PgDn | Zeitfilter rückwärts |
| Backspace | Alle Filter löschen |
| Esc / F4 | Zurück zum QSO Form |

### Partner View — F2

| Taste | Aktion |
|-----|--------|
| F2 | Zyklus: Partner View → Foto → Zurück |
| Esc / F1 | Zurück zum QSO Form |

---

## Fehlerbehebung

### Die App startet nicht

- Terminal muss mindestens 80×24 Zeichen haben.
- Unter Windows Windows Terminal verwenden, nicht die legacy `cmd.exe`-Konsole.
- `cqops --offline` versuchen, um Netzwerkprobleme auszuschließen.
- Logs prüfen: `~/.local/share/cqops/logs/` (Linux), `~/Library/Application Support/cqops/logs/` (macOS) oder `%APPDATA%\cqops\logs\` (Windows).

### Rig verbindet sich nicht

- **flrig:** prüfen, ob flrig läuft und der Port übereinstimmt (Standard `12345`).
- **Hamlib:** prüfen, ob rigctld läuft und der TCP-Port korrekt ist.
- Statuslabel-Farbe: weiß = verbunden, gelb = Verbindungsaufbau/deaktiviert, rot = Fehler.
- Unterdrückte Reconnect-Toasts sind normal — CQOps wiederholt im Hintergrund.

### WSJT-X loggt nicht automatisch

- WSJT-X UDP-Einstellungen prüfen: Settings → Reporting → UDP Server.
- WSJT-X muss Version 2.6 oder neuer sein.
- Statuslabel sollte weiß (Standard) sein, wenn WSJT-X läuft.

### Wavelog-Upload schlägt fehl

- URL, API-Key und Stationsprofil-ID in der Konfiguration prüfen.
- Statuslabel: weiß = erreichbar, gelb = deaktiviert/kein Internet, rot = Fehler.
- Upload-Fehler werden als Toasts angezeigt; QSOs bleiben lokal gespeichert.
- Einzelne QSO-Fehler blockieren nicht den Rest der Charge.

### Probleme mit der Konfigurationsdatei

- Konfiguration: `~/.config/cqops/config.yaml` (Linux/macOS) oder `%APPDATA%\cqops\config.yaml` (Windows).
- Zugangsdaten: `secrets.enc` im selben Verzeichnis.
- Wenn die Konfiguration beschädigt ist, löschen und neu starten — der Assistent erstellt eine neue.
- Das Feld `last_fetched_id` erscheint nur nach einem erfolgreichen Wavelog-Download.

### Performance

- Karten-Rendering und Solar Panel in den General-Einstellungen deaktivieren.
- Nicht verwendete Tabs schließen (DXC, PSK).
- Mit `--offline` starten, wenn das Netzwerk unzuverlässig ist.

### Fehler melden

**Debug-Modus** vor dem Reproduzieren eines Problems aktivieren — F9 → General → Debug oder `debug: true` in der Konfiguration setzen. Vollständige Logs werden im plattformspezifischen Log-Verzeichnis geschrieben.

Melden Sie Probleme auf [GitHub Issues](https://github.com/szporwolik/cqops/issues) mit:
- CQOps-Version (`cqops --version`)
- Betriebssystem und Terminal-Emulator
- Schritte zur Reproduktion
- Debug-Log
