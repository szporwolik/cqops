---
title: CQOps Benutzerhandbuch
description: Praktische Anleitung zur Einrichtung von CQOps und zum Loggen im Shack und unterwegs
---

# CQOps Benutzerhandbuch

CQOps ist ein tastaturbedienter Amateurfunk-Logger für den Heim-, Portabel- und Clubbetrieb sowie gelegentliche Contests. QSOs werden zuerst auf dem eigenen Computer gespeichert; Internetdienste sind optional. Beginnen Sie mit manueller Eingabe und ergänzen Sie Funkgerätesteuerung oder Online-Dienste nach Bedarf.

Menü- und Feldnamen entsprechen hier der englischen Oberfläche. Tastenkürzel gelten je nach aktivem Bildschirm; die untere Hilfeleiste und **?** zeigen die verfügbaren Funktionen.

## Inhalt

1. [Installation](#installation)
2. [Erste Einrichtung](#setup)
3. [Das erste QSO](#first-qso)
4. [Bildschirme und Status](#screens)
5. [Tägliches Loggen](#logging)
6. [Stationsprofile](#profiles)
7. [Logbuch und Sicherungen](#logbook)
8. [Funkgerät und Digitalbetrieb](#radio)
9. [Online-Dienste](#online)
10. [GPS und APRS](#position)
11. [Portabelbetrieb](#portable)
12. [Contests](#contests)
13. [CQOps Live](#dashboard)
14. [Tastenkürzel](#keys)
15. [Fehlerbehebung und Hilfe](#help)

<a id="installation"></a>

## Installation

Laden Sie CQOps von der [Release-Seite](https://github.com/szporwolik/cqops/releases). Das Terminalfenster muss mindestens 75 × 24 Zeichen groß sein; ab 80 × 43 arbeitet es sich komfortabler.

| System | Installation |
|---|---|
| Windows | `cqops-setup.exe` herunterladen oder `cqops-windows-portable.zip` ohne Installation entpacken. Windows Terminal wird empfohlen. |
| Debian, Ubuntu, Linux Mint, Pop!_OS | Passendes `.deb` herunterladen: `amd64` für die meisten Intel/AMD-PCs, `arm64` für 64-Bit-ARM oder `armhf` für 32-Bit-Raspberry-Pi-OS. Mit dem Paketinstallationsprogramm öffnen. |
| Fedora, RHEL, Rocky, AlmaLinux | Die unten stehenden Repository-Befehle verwenden. |
| Arch, Manjaro, CachyOS | AUR-Paket mit `paru -S cqops-bin` oder `yay -S cqops-bin` installieren. |
| Andere Linux-Systeme | Passendes Linux-Archiv `.tar.gz` für den Prozessor von der Release-Seite laden und entpacken. |
| macOS | `cqops-darwin-arm64` für Apple Silicon oder `cqops-darwin-amd64` für Intel laden. Die Befehle unten verwenden. |

Auf Debian-basierten Systemen ist alternativ die Installation aus dem Repository möglich:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

Auf Fedora-basierten Systemen:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

Unter macOS im Downloadordner ausführen und `FILE` durch den genauen heruntergeladenen Dateinamen ersetzen:

```bash
chmod +x FILE
sudo mv FILE /usr/local/bin/cqops
```

Starten Sie mit `cqops` oder führen Sie das entpackte portable Programm aus. `cqops --offline` startet offline, `cqops --version` zeigt die Version und `cqops --help` die Startoptionen. Exportieren Sie Ihre Logbücher vor einem Update.

<a id="setup"></a>

## Erste Einrichtung

Der Assistent fragt nach Logbuchname, Stationsrufzeichen, Maidenhead-Locator und Kontinent. Das **Stationsrufzeichen** wird beim Senden verwendet; ein **Operatorprofil** bezeichnet die Person an der Station.

**Ctrl+A** zeigt optionale Stationsreferenzen und CQ-/ITU-Zonen. Die eigene SOTA-/POTA-/WWFF-Referenz gehört in die Stations-/Logbucheinstellungen; Referenzen im QSO-Formular gehören zur Gegenstation. Die IARU-Region wird später unter **F9 → Logbooks** eingestellt.

Legen Sie ein Funkgeräteprofil mit Name, Antenne und Leistung an. Wählen Sie **None** für manuelle Frequenz- und Betriebsarteneingabe, **flrig** oder **Hamlib**. Richten Sie optionale Verbindungen erst ein, wenn das grundlegende Loggen funktioniert.

**Tab / Shift+Tab** wechselt Felder, **Space** ändert Auswahloptionen und **Save & Next** führt weiter. **Esc** geht zurück, **F10** beendet das Programm. Prüfen und speichern Sie die Zusammenfassung. CQOps erkennt die Zeitzone des Computers; QSO-Datum und -Zeit werden in UTC geführt. Prüfen Sie vor dem Betrieb die Rechneruhr.

<a id="first-qso"></a>

## Das erste QSO

1. **F1** drücken und Logbuch, Stationsrufzeichen, Operator, Funkgerät und Contest prüfen.
2. Rufzeichen der Gegenstation eingeben; **Ins** startet eine konfigurierte Abfrage.
3. UTC-Datum/-Zeit, Frequenz in MHz, Band, Betriebsart sowie gesendeten und empfangenen Rapport prüfen.
4. Bei Bedarf Name, QTH, Locator, Referenz oder Kommentar ergänzen.
5. **Enter** drücken. Die Verbindung erscheint in Recent QSOs.

Fordert **DUPE!** eine Bestätigung, speichert ein weiteres **Enter** trotzdem; **Esc** bricht die Bestätigung ab. Die Warnung fordert zum Prüfen auf und bedeutet nicht automatisch, dass das QSO verworfen werden muss.

<a id="screens"></a>

## Bildschirme und Status

| Taste | Bildschirm | Zweck |
|---|---|---|
| F1 | QSO | Kontakte eingeben und letzte QSOs ansehen |
| F2 | Partner | Rufzeichendaten, Karte, Statistik, Foto |
| F3 | APRS | Stationen in der Nähe |
| F4 | DX Cluster | Spots und Filter |
| F5 | PSK Reporter | Empfangsberichte digitaler Betriebsarten |
| F6 | References | SOTA-, POTA-, WWFF-, IOTA-Suche |
| F7 | Band Plan | Frequenzen und Betriebsvorgaben |
| F8 | Logbook | Bearbeiten, importieren, exportieren, synchronisieren |
| F9 | Configuration | Stations- und Diensteinstellungen |
| F10 | Quit | CQOps beenden |

Oben stehen die aktive Stationseinrichtung, Ortszeit (**L**) und UTC (**Z**). Weiß bedeutet normalerweise aktiv, Gelb deaktiviert/Verbindungsaufbau/Warten und Rot einen Fehler. WSJT ist während des Sendens hervorgehoben. **WL!** warnt vor einem nicht unterstützten alten Wavelog-Schlüssel.

<a id="logging"></a>

## Tägliches Loggen

Mit **Tab / Shift+Tab** wechseln Sie Felder, mit **PgUp / PgDn** Band, Betriebsart oder Unterbetriebsart. **Shift+Backspace** leert das aktuelle Feld, **Del** das Formular. Prüfen Sie bei Splitbetrieb **Freq RX**.

**Keep** behält den Kommentar nach dem Speichern. **Retain** behält das gesamte Formular: vor dem nächsten Speichern Rufzeichen, Zeit, Rapporte und Referenzen prüfen. Contest-Austauschfelder erscheinen nur bei aktivem Contest. **SIG / SIG Info** ist für weitere Angaben zu Interessengruppen vorgesehen.

Sind beide Locator bekannt, zeigt CQOps Entfernung und Peilung. Callbook-Daten können den Heimatstandort statt des aktuellen Portabelstandorts angeben; prüfen Sie sie. Kennzeichnungen für neues Rufzeichen, neues DXCC und Duplikat helfen bei der Einordnung.

**F6** sucht Referenzen nach Name oder Kennung und kann die Referenz der Gegenstation übernehmen. **F7** zeigt Bandpläne und kann ein verbundenes Funkgerät abstimmen. Die Einträge sind Betriebshilfen, keine Sendegenehmigung: eigene Berechtigungen und örtlichen Bandplan beachten.

Drei gemeinsame Favoriten speichern Frequenz, Betriebsart und Band:

| Platz | Abrufen | Aktuelle Werte speichern |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

<a id="profiles"></a>

## Stationsprofile

Logbücher, Operatoren, Funkgeräte und Contests werden in den jeweiligen **F9**-Menüs angelegt; **Ins** fügt einen Eintrag hinzu. Im QSO-Bildschirm:

| Kürzel | Wechseln |
|---|---|
| Ctrl+L | Logbuch |
| Ctrl+O | Operator |
| Ctrl+R | Funkgerät |
| Ctrl+C | Contest |

Logbücher haben getrennte Stationsdaten und Wavelog-/APRS-Einstellungen. Operatorprofile bezeichnen die bedienende Person; das Rufzeichen wird in ADIF `OPERATOR` gespeichert. Funkgeräteprofile enthalten Ausrüstung, Leistung, Funkgeräte-/Rotorsteuerung und WSJT-X-Einstellungen. Prüfen Sie nach jedem Wechsel die Statusleiste, besonders beim automatischen digitalen Loggen.

Weitere **F9**-Menüs betreffen Darstellung, Einheiten, Zeitzone, Callbooks, Integrationen und Hinweistöne.

<a id="logbook"></a>

## Logbuch und Sicherungen

In **F8** ein QSO auswählen und mit **Enter** oder **e** bearbeiten. Mit **Enter** speichern und bestätigen. **Delete** löscht den ausgewählten Kontakt. Vor Massenänderungen sichern; **Ctrl+P** löscht destruktiv alle QSOs und ist keine Suchfunktion.

| Kürzel in F8 | Aktion |
|---|---|
| Ctrl+I | ADIF importieren, Datensätze prüfen und Duplikate überspringen |
| Ctrl+E | Alle Kontakte oder nach Contest gefilterte Auswahl exportieren |
| Ctrl+W | Noch nicht übertragene Kontakte zu Wavelog senden |
| Alt+W | Von Wavelog herunterladen |

Importzusammenfassung und Exportauswahl prüfen. Importierte Kontakte können anschließend zu Wavelog übertragen werden. CQOps unterstützt ADIF 3.1.7 und erhält Contest-IDs und Austauschdaten. Sichern Sie jedes Logbuch separat, möglichst auf einem anderen Gerät. ADIF sichert Kontakte, nicht sämtliche Einstellungen oder Zugangsdaten.

Die Konfiguration liegt unter Linux/macOS in `~/.config/cqops/config.yaml`, unter Windows in `%APPDATA%\cqops\config.yaml`. Zugangsdaten stehen separat in `secrets.enc`; beim Rechnerwechsel müssen sie neu eingegeben werden. Löschen Sie Konfigurationsdateien nicht als ersten Schritt der Fehlersuche.

<a id="radio"></a>

## Funkgerät und Digitalbetrieb

### Funkgerätesteuerung

Unter **F9 → Rigs** flrig oder Hamlib wählen und die Verbindungseinstellungen abgleichen. Zuerst flrig oder `rigctld` starten. flrig verwendet üblicherweise `localhost:12345`. Welche Frequenz-, Betriebsart-, Split- und Leistungsdaten verfügbar sind, hängt vom Funkgerät ab. Bei **None** erfolgt die Eingabe manuell.

### WSJT-X

Verwenden Sie WSJT-X ab 2.6. Gleichen Sie **Settings → Reporting → UDP Server** mit den UDP-Einstellungen des aktiven CQOps-Funkgeräteprofils ab. Loggen Sie ein abgeschlossenes Test-QSO in WSJT-X und prüfen Sie, ob es in CQOps erscheint.

Empfangene QSOs verwenden das aktive Logbuch und den aktiven Contest; Duplikate werden übersprungen. Vor einer Sitzung Operator und WSJT-Anzeige prüfen. CQOps warnt bei abweichendem Operator. Bei entsprechender Einrichtung kann der Wavelog-Upload folgen. Mode/Submode passend wählen; FT8 wird als FT8 exportiert, FT4/FT2 als MFSK mit passender Unterbetriebsart.

### Rotorsteuerung

Die Steuerung über Hamlib `rotctld` ist experimentell. Prüfen Sie Richtung und mechanische Grenzen vor der Verwendung. Halten Sie eine sichere Stoppmöglichkeit bereit: falsche Einstellungen können Antenne, Rotor oder Speiseleitung beschädigen.

| Kürzel | Aktion |
|---|---|
| Alt+, / Alt+. | Azimut −5° / +5° |
| Alt+' / Alt+; | Elevation −5° / +5° |
| Alt+\ | Auf die berechnete Peilung drehen |
| Alt+/ | Bewegung stoppen |

<a id="online"></a>

## Online-Dienste

### Callbooks

Anbieter und Reihenfolge unter **F9 → Callbook** festlegen, dann **Ins** im QSO-Formular drücken. Aktivierte Anbieter werden nacheinander abgefragt. Die Basisrufzeichen-Suche kann Portabelpräfixe/-suffixe weglassen; den gefundenen Standort prüfen.

| Anbieter | Zugang |
|---|---|
| QRZ.com | XML-Abonnement und Zugangsdaten |
| HamQTH | Kostenloses Konto |
| QRZ.RU | API-Zugang getrennt vom Website-Zugang |
| Callook.info | US-Rufzeichen; kein Konto |

**F2** zeigt die Gegenstation. Fotos hängen vom Anbieter und Terminal ab; das experimentelle **Kitty Graphics** unter General benötigt beispielsweise Kitty, Ghostty oder WezTerm.

### Wavelog

URL, API-v2-Token (`wl2_…`) und Stationsprofil pro Logbuch einstellen. Alte v1-Schlüssel werden nicht akzeptiert. Die Auswahl einer Wavelog-Station kann lokale Stationsdaten ausfüllen: vor dem Speichern Rufzeichen, Locator und Referenzen prüfen.

QSOs werden zuerst lokal gespeichert. Fehlgeschlagene Uploads mit **F8 → Ctrl+W** wiederholen; **Alt+W** lädt Kontakte herunter. Beim Öffnen eines verknüpften QSOs zur Bearbeitung kann CQOps die Daten aus Wavelog aktualisieren. Online-Änderungen und Löschungen betreffen auch die entfernte Kopie. Lesen Sie die Bestätigung, besonders offline; eine rein lokale Änderung ist nicht automatisch in Wavelog angekommen.


Clubstationen: Verwenden Sie den `wl2_`-Schlüssel des Besitzers zusammen mit der Option **Gemeinsame Clubstation** im Logbuchformular. Synchronisierte Kontakte sind dann schreibgeschützt — Änderungen und Löschungen erfolgen auf der Wavelog-Seite; CQOps sendet für sie keine PATCH- oder DELETE-Anfragen. Neue Kontakte werden normal hochgeladen und dem aktiven Operator zugeordnet (fehlt dieser, dem Stationsrufzeichen). Der API-Schlüssel wird verschlüsselt gespeichert und bei aktivierter Option nach dem Speichern nie wieder angezeigt — Feld leer lassen, um ihn zu behalten, neuen Schlüssel eingeben, um ihn zu ersetzen.
### DX Cluster und Ausbreitung

DX Cluster unter Integrations einrichten und **F4** öffnen. **b / c / m / t** filtern Band, Kontinent des Spotters, Betriebsart und Alter. **Backspace** setzt Filter zurück. **Enter** übernimmt den Spot, stimmt das verbundene Funkgerät ab und kehrt zu F1 zurück; **Space** stimmt ab, ohne den Cluster zu verlassen.

Auf F1 öffnet **Ctrl+S** den Spot-Dialog und **Ctrl+P** übernimmt das Rufzeichen des nächstgelegenen angezeigten Spots. Vor dem Senden prüfen. **F5** zeigt PSK-Reporter-Empfangsberichte, keine Garantie aktueller Ausbreitung. Solar zeigt HamQSL-Bedingungen; zwischengespeicherte Werte können veraltet sein.

<a id="position"></a>

## GPS und APRS

### GPS

Unter Integrations einen seriellen GPS-Empfänger oder GPSD einrichten. **Grid from GPS** in den Stations-/Logbucheinstellungen aktivieren, um den Locator für QSOs, Peilungen, APRS und Dashboard zu nutzen. Rot bedeutet Fehler, Gelb keine Positionsbestimmung, Weiß gültige Position. Vor dem Betrieb den Locator prüfen. 6, 8 oder 10 Zeichen wählen; mehr Zeichen garantieren keine höhere Empfängergenauigkeit.

### APRS

| Dienst | Verbindung |
|---|---|
| APRS-IS | APRS-Server im Internet |
| KISS | Serieller Hardware-TNC und Funkgerät |
| KISS Server | TCP-TNC wie Dire Wolf; lokal nutzbar |

Dienst unter **F9 → Integrations → APRS** wählen. Rufzeichen/SSID, Symbol, Kommentar, Reichweite und Bakenintervall unter **F9 → Logbooks → [active logbook] → APRS** setzen. **APRS TX** und **Send beacons** nur aktivieren, wenn gesendet werden soll. Reiner Empfang zeigt **APRS-RX**. Positionsbaken veröffentlichen Ihren Standort; Position und gewünschten Empfängerkreis vorher prüfen.

Automatische Bakenintervalle betragen mindestens fünf Minuten. **F3** zeigt kürzlich gehörte Stationen: Pfeile wählen, **Enter** übernimmt ins QSO-Formular, **d / t / s** ändern Entfernung/Alter/Typ, **Backspace** löscht Filter und **b** sendet eine konfigurierte Bake sofort. GPS-Baken benötigen **Grid from GPS** und eine gültige Position.

<a id="portable"></a>

## Portabelbetrieb

Vor der Abfahrt das Portabellogbuch wählen; Rufzeichen, Locator, Aktivierungsreferenz, Funkgerät, Antenne und Leistung prüfen. Die gesamte Station testen und CQOps online starten, um Referenz- und Präfixdaten zu aktualisieren. Prüfen, ob **F6** die benötigten Referenzen findet. Sicherung exportieren.

Lokales Loggen funktioniert ohne Internet. `cqops --offline` überspringt Netzwerkfunktionen; Live-Abfragen und Synchronisierung sind dann nicht verlässlich verfügbar. Lokale Netzwerkgeräte vor der Abfahrt im gewählten Startmodus testen. Zwischengespeicherte Daten können veraltet sein.

Danach QSO-Anzahl und Referenzen prüfen, ADIF exportieren, sichern und gegebenenfalls nicht übertragene Kontakte zu Wavelog senden. Das benötigte Einreichungsformat jedes Diplomprogramms prüfen und bei Bedarf konvertieren.

<a id="contests"></a>

## Contests

CQOps unterstützt gelegentliches Contest-Loggen, Austauschdaten, laufende Nummern und QSO-Raten. Es ist kein vollständiges Auswertungs- oder Einreichungssystem. Für anspruchsvollen Contestbetrieb einen spezialisierten Logger verwenden.

Unter **F9 → Contests** mit **Ins** Name, Datum, ADIF-Contest-ID, Startnummer und Vorlagen für gesendeten/empfangenen Austausch festlegen.

| Platzhalter | Wert |
|---|---|
| `@rst` | Gesendeter oder empfangener Rapport |
| `@serial` | Laufende Nummer |
| `@cqz` / `@mycqz` | CQ-Zone der Gegenstation / eigene |
| `@itu` / `@myitu` | ITU-Zone der Gegenstation / eigene |
| `@grid` / `@mygrid` | Locator der Gegenstation / eigener |

Auf **F1** wechselt **Ctrl+C** den Contest. Austausch und nächste Nummer vor dem Senden prüfen. Die Statusleiste zeigt Anzahl, nächste Nummer und Zeiten; breitere Fenster zeigen weitere Ratenstatistiken. Nach Abschluss wieder ohne aktiven Contest arbeiten.

Zum Export **F8** öffnen, mit **Ctrl+C** den Contestfilter wählen, dann **Ctrl+E** und die Auswahl prüfen. Ausgabeformat ist ADIF, nicht Cabrillo. Format- und Einreichungsregeln des Veranstalters beachten.

<a id="dashboard"></a>

## CQOps Live

**F9 → Integrations → HTTP Server** aktivieren und mit **Ctrl+S** speichern. Auf dem CQOps-Computer `http://localhost:8073` öffnen.

Die Standardadresse `0.0.0.0` erlaubt Zugriff aus dem lokalen Netz, soweit die Firewall ihn zulässt. Auf anderen Geräten die IP-Adresse des CQOps-Computers mit Port `8073` verwenden. `127.0.0.1` beschränkt den Zugriff auf diesen Computer. Nur in einem vertrauenswürdigen Netz verwenden; den Port nicht ins Internet weiterleiten.

Das Dashboard aktualisiert Kontakt, QSO-Karten, letzte Verbindungen, Raten, Operatoren, APRS sowie verfügbare Ausbreitungs-/Wetterdaten automatisch. Internetabhängige Ebenen können offline fehlen. Header 1, Header 2, Logo URL und Event Start passen die Veranstaltungsanzeige an; das Startdatum filtert Statistiken und QSO-Listen.

<a id="keys"></a>

## Tastenkürzel

| Bildschirm | Tasten | Aktion |
|---|---|---|
| Allgemein | ? / Esc / F10 | Hilfe / zurück / beenden |
| QSO | Tab / Shift+Tab | Nächstes / vorheriges Feld |
| QSO | Enter / Ins | Speichern / abfragen |
| QSO | Shift+Backspace / Del | Feld / Formular leeren |
| QSO | Ctrl+L / Ctrl+O / Ctrl+R / Ctrl+C | Logbuch / Operator / Funkgerät / Contest wechseln |
| Logbuch | ↑ / ↓, PgUp / PgDn, Home / End | Auswahl, Seite, erste / letzte Zeile |
| Logbuch | Enter oder e / Delete | Gewähltes QSO bearbeiten / löschen |
| Logbuch | Ctrl+I / Ctrl+E | ADIF importieren / exportieren |
| Logbuch | Ctrl+W / Alt+W | Wavelog hochladen / herunterladen |
| Logbuch | Ctrl+C / Backspace | Contestfilter / Suche leeren |

Kürzel sind bildschirmabhängig: **Ctrl+C beendet CQOps nicht**. Auf Laptops kann für Funktionstasten **Fn** nötig sein. Fängt das Terminal eine Taste ab, dessen Tastatureinstellungen und die CQOps-Hilfe prüfen.

<a id="help"></a>

## Fehlerbehebung und Hilfe

| Problem | Zuerst prüfen |
|---|---|
| Start oder unvollständiger Bildschirm | Terminalgröße, Windows Terminal unter Windows, `cqops --offline` testen |
| Funkgerät nicht verbunden | Aktives Profil, flrig/rigctld gestartet, Modell, serielle Schnittstelle, Geschwindigkeit, Host/Port, Schnittstelle durch anderes Programm belegt |
| WSJT-X-QSO fehlt | UDP-Einstellungen, WSJT-Anzeige, QSO tatsächlich in WSJT-X geloggt, aktives Logbuch |
| Wavelog-Fehler | URL, `wl2_`-Token, Stationsprofil, Internet; lokale QSOs bleiben erhalten |
| Keine GPS-Position | Schnittstelle/Geschwindigkeit oder GPSD-Adresse, freie Himmelssicht, gültige Position, Grid from GPS |
| Keine APRS-Baken | Logbuch, APRS TX und Send beacons, Rufzeichen/SSID, TNC/Funkgerät oder Internet |
| Dashboard nicht erreichbar | Server aktiviert, richtige IP und Port, Firewall; localhost bezeichnet das Gerät mit dem Browser |

Bei Einstellungsproblemen zuerst **F9** nutzen, bevor Dateien geändert werden. Zugangsdaten neu eingeben, wenn CQOps ein Problem mit gespeicherten Geheimnissen meldet oder der Computer gewechselt wurde.

Bei Bedarf **F9 → General → Debug** aktivieren, den Fehler sicher reproduzieren und das Diagnoseprotokoll sichern; danach Debug wieder deaktivieren.

| System | Diagnoseprotokolle |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

Probleme unter [GitHub Issues](https://github.com/szporwolik/cqops/issues) melden. CQOps-Version, Betriebssystem, Terminal, Schritte und relevantes Protokoll angeben. Passwörter, API-Token und private Informationen vor dem Teilen entfernen.
