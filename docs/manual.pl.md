---
title: Instrukcja użytkownika CQOps
description: Przewodnik instalacji, konfiguracji i używania CQOps — szybkiego terminalowego loggera krótkofalarskiego
---

> **Uwaga o tłumaczeniu:** To tłumaczenie zostało wygenerowane z użyciem modelu LLM. Poprawki są mile widziane jako Pull Request do gałęzi `dev`. Nazwy ekranów, pól, komend i skrótów są miejscami celowo pozostawione po angielsku, aby były zgodne z interfejsem CQOps.

# Instrukcja użytkownika CQOps

CQOps to szybki, terminalowy logger krótkofalarski dla operatorów, którzy chcą sprawnego logowania z klawiatury przy małym obciążeniu systemu. Jest przeznaczony do pracy w shacku, w terenie, w klubie, podczas field day oraz na słabszych komputerach, takich jak Raspberry Pi lub starsze laptopy.

CQOps zawsze zapisuje QSO najpierw lokalnie. Integracje internetowe są opcjonalne.

## Spis treści

1. [Czym jest CQOps](#czym-jest-cqops)
2. [Pobieranie i instalacja](#pobieranie-i-instalacja)
3. [Pierwsze uruchomienie](#pierwsze-uruchomienie)
4. [Pierwsze QSO](#pierwsze-qso)
5. [Główny ekran](#główny-ekran)
6. [Typowe scenariusze pracy](#typowe-scenariusze-pracy)
7. [Logowanie QSO](#logowanie-qso)
8. [Logbook Editor i ADIF](#logbook-editor-i-adif)
9. [Zawody](#zawody)
10. [Favorites, REF i Band Plan](#favorites-ref-i-band-plan)
11. [Integracje](#integracje)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Konfiguracja](#konfiguracja)
14. [Skróty klawiaturowe](#skróty-klawiaturowe)
15. [Rozwiązywanie problemów](#rozwiązywanie-problemów)
16. [Zgłaszanie błędów](#zgłaszanie-błędów)

---

## Czym jest CQOps

CQOps koncentruje się na szybkim wprowadzaniu QSO, lokalnym logowaniu i praktycznej pracy w terenie.

### Główne założenia

- **Terminal-first** — praca zoptymalizowana pod klawiaturę.
- **Offline-first** — lokalne logowanie QSO działa bez internetu.
- **Niskie obciążenie** — dobre dla Raspberry Pi, starszych laptopów i komputerów w klubie.
- **Przenośna forma** — CQOps jest dystrybuowany jako pojedynczy binarny plik Go.
- **Wiele logbooków** — do pracy prywatnej, terenowej, zawodów i klubów.
- **Wielu operatorów** — do pracy hot-seat i współdzielonych stacji.
- **Wiele rigów** — każdy preset rigu może mieć własny backend i ustawienia WSJT-X.
- **Opcjonalne integracje** — QRZ.com, Wavelog, DX Cluster, PSK Reporter, APRS, sterowanie radiem, sterowanie rotorem, dane solarne oraz CQOps Live w przeglądarce.

Lokalne logowanie nie wymaga internetu. Funkcje sieciowe są pomijane w trybie `--offline`.

### Dla kogo jest CQOps

CQOps pasuje do:

- operatorów portable,
- aktywatorów SOTA i POTA,
- stacji klubowych,
- zespołów field day,
- operatorów preferujących terminal,
- stacji, gdzie trzeba szybko przełączać operatorów, logbooki lub rigi.

CQOps nie ma zastępować każdego elementu rozbudowanego loggera desktopowego ani platformy webowej. Skupia się na szybkim terminalowym logowaniu, pracy terenowej, pracy offline i scenariuszach współdzielonej stacji.

---

## Pobieranie i instalacja

Wszystkie wydania:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Pakiet | Link | Uwagi |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Zalecany dla większości użytkowników. Dodaje CQOps do Start Menu i PATH. |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Rozpakuj i uruchom bez instalacji. |

Używaj **Windows Terminal**, nie starej konsoli systemowej.

### Linux — Debian / Ubuntu

| Architektura | Link | Zastosowanie |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Większość komputerów Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-bitowe systemy ARM |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-bitowy Raspberry Pi OS |

Instalacja pobranego pakietu:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — przenośny tarball

| Architektura | Link | Zastosowanie |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Większość komputerów Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-bitowe systemy ARM |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-bitowy Raspberry Pi OS |

### macOS

| Architektura | Link | Zastosowanie |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Maki M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Maki Intel |

Instalacja ręczna:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Budowanie ze źródeł

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

Budowanie ze źródeł wymaga Go 1.26 lub nowszego.

### Wymagania terminala

| Wymaganie | Wartość |
|---|---|
| Minimalny rozmiar terminala | 80×24 znaki |
| Zalecany rozmiar terminala | 80×43 znaki lub więcej |
| Zalecany terminal w Windows | Windows Terminal |

### Podstawowe komendy

```bash
cqops              # Start TUI
cqops --offline    # Start bez aktywności sieciowej
cqops --version    # Pokaż wersję i zakończ
cqops --help       # Pokaż pomoc
```

---

## Pierwsze uruchomienie

Przy pierwszym starcie CQOps uruchamia setup wizard. Do lokalnego logowania wymagane są tylko podstawowe dane stacji. Integracje sieciowe można pominąć i ustawić później.

### Strony kreatora

| Strona | Co ustawia |
|---|---|
| Station & Logbook | Początkowy logbook, znak stacji, operator, grid locator, opcjonalne referencje i strefy, Wavelog URL/API/station profile ID |
| Rig | Preset rigu, model, antena, moc, backend, opcjonalny rotor, opcjonalne ustawienia UDP WSJT-X |
| Integrations | Ustawienia QRZ.com lookup |
| General | Strefa czasowa IANA |
| Summary | Przegląd i zapis |

Obsługiwane backendy rigu:

- None,
- flrig,
- Hamlib `rigctld`.

### Nawigacja w kreatorze

| Klawisz | Akcja |
|---|---|
| Ctrl+S | Waliduj i kontynuuj; na Summary zapisuje i startuje CQOps |
| Esc | Wróć |
| F10 | Zakończ |
| Tab / Shift+Tab | Przejście między polami |
| Space | Przełącz checkbox |

Ustawienia kreatora można później zmienić przez **F9**.

---

## Pierwsze QSO

1. Uruchom CQOps:

   ```bash
   cqops
   ```

2. Uzupełnij setup wizard co najmniej znakiem i grid locatorem.
3. Otwórz QSO form klawiszem **F1**.
4. Wpisz znak korespondenta. CQOps automatycznie zamienia znaki na wielkie litery.
5. Uzupełnij pozostałe pola. Jeśli aktywny rig jest połączony przez flrig lub Hamlib, CQOps może automatycznie wypełnić częstotliwość, pasmo, mode i submode.
6. Naciśnij **Enter** lub **Ctrl+S**, aby zapisać.
7. Jeśli pojawi się ostrzeżenie **DUPE!**, naciśnij ponownie **Enter**, aby mimo to zapisać, albo **Esc**, aby anulować.

Zapisane QSO pojawia się od razu w tabeli Recent QSOs.

---

## Główny ekran

CQOps używa stałego układu terminala:

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

Status bar pokazuje:

- wersję CQOps,
- aktywny logbook,
- aktywny rig,
- znak stacji,
- aktywnego operatora,
- statusy integracji,
- czas lokalny oznaczony jako `L`,
- czas UTC oznaczony jako `Z`.

Typowe etykiety: **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL**.

| Kolor | Znaczenie |
|---|---|
| Biały/domyślny | Połączone lub aktywne |
| Żółty | Wyłączone, łączy się lub oczekiwany tryb offline |
| Czerwony | Błąd lub rozłączone |
| Akcent + pogrubienie | WSJT-X nadaje |

### Główne zakładki

| Klawisz | Zakładka | Ekran |
|---|---|---|
| F1 | QSO | QSO form i Recent QSOs |
| F2 | QRZ | Partner view: dane callbook, mapa, statystyki, zdjęcie |
| F4 | DXC | Spoty DX Cluster i filtry |
| F5 | HRD | Spoty PSK Reporter i mapa propagacji |
| F6 | REF | Wyszukiwarka referencji SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, synchronizacja Wavelog |
| F9 | CFG | Menu konfiguracji |

Help bar pokazuje skróty właściwe dla aktywnego ekranu. **?** otwiera pełną pomoc.

---

## Typowe scenariusze pracy

### Praca portable, SOTA lub POTA

Przed wyjazdem:

1. Uruchom CQOps raz z dostępem do internetu.
2. Pozwól pobrać lub odświeżyć cache: dane solarne, dane REF i prefiksy DXCC.
3. Sprawdź, czy panel Solar pokazuje dane.
4. Sprawdź, czy wyszukiwanie REF na **F6** zwraca wyniki.

W terenie:

1. Uruchom CQOps w trybie offline:

   ```bash
   cqops --offline
   ```

2. Loguj normalnie. QSO są zapisywane lokalnie.
3. Po powrocie online otwórz **F8** i naciśnij **w**, aby wysłać niewysłane QSO do Wavelog.

### Współdzielona stacja klubowa i hot-seat

1. Otwórz **F9 → Operators**.
2. Naciśnij **Ins**, aby dodać profile operatorów.
3. Na QSO form naciśnij **Ctrl+O**, aby zmienić aktywnego operatora.
4. Przed zapisem sprawdź aktywnego operatora w status bar.
5. Używaj **Retain**, gdy kilku operatorów loguje podobne łączności bez przepisywania całego formularza.

Aktywny operator jest zapisywany w polu ADIF `OPERATOR`.

### Logbook prywatny i klubowy

1. Otwórz **F9 → Logbooks**.
2. Naciśnij **Ins**, aby utworzyć logbooki.
3. Na QSO form naciśnij **Ctrl+L**, aby zmienić aktywny logbook.
4. Przed zapisem sprawdź aktywny logbook w status bar.

Każdy logbook może mieć własne dane stacji, ustawienia Wavelog, ustawienia zawodów i operatorów.

### Wiele rigów

1. Otwórz **F9 → Rigs**.
2. Naciśnij **Ins**, aby utworzyć presety rigów.
3. Wybierz backend: None, flrig lub Hamlib.
4. Na QSO form naciśnij **Ctrl+R**, aby zmienić aktywny rig.

Preset rigu może zawierać backend, model, antenę, moc, ustawienia rotora i ustawienia UDP WSJT-X.

### Praca cyfrowa WSJT-X

Gdy integracja UDP WSJT-X jest włączona, CQOps może odbierać wiadomości ADIF z WSJT-X i automatycznie logować zakończone cyfrowe QSO.

Automatycznie logowane QSO:

- trafiają do aktywnego logbooka,
- od razu pojawiają się w Recent QSOs,
- pomijają duplikaty,
- dziedziczą aktywny contest ID,
- mogą być automatycznie wysłane do Wavelog, gdy Wavelog jest skonfigurowany i dostępny.

Jeśli operator raportowany przez WSJT-X nie zgadza się z aktywnym operatorem w CQOps, pojawia się ostrzeżenie.

Przed dłuższą sesją cyfrową sprawdź:

- aktywny logbook,
- aktywnego operatora,
- aktywne zawody,
- status WSJT-X.

### Synchronizacja Wavelog

CQOps zawsze zapisuje QSO najpierw lokalnie. Synchronizacja Wavelog jest opcjonalna.

| Akcja | Gdzie | Skrót | Uwagi |
|---|---|---|---|
| Upload niewysłanych QSO | Logbook Editor | `w` | Wysyłka w paczkach po 50 |
| Download z Wavelog | Logbook Editor | `Ctrl+W` | Pobieranie przyrostowe przez `last_fetched_id` |

Status uploadu jest śledzony per QSO:

- not sent,
- sent,
- error.

Jeśli upload się nie uda, QSO pozostaje w lokalnym logbooku i można spróbować ponownie. Purge logbooka resetuje fetch ID do `0`, co pozwala na pełne ponowne pobranie.

---

## Logowanie QSO

QSO form to główny ekran logowania. Otwiera się go klawiszem **F1**.

CQOps może wypełniać pola z:

| Źródło | Pola |
|---|---|
| flrig / Hamlib | Frequency, Freq RX przy split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| REF database | Referencje SOTA, POTA, WWFF, IOTA |
| Wavelog lookup | Worked/confirmed status, gdy skonfigurowane |
| DXCC/prefix data | Dane prefiksu i kraju |

### Układ formularza

| Lewa kolumna | Środkowa kolumna | Prawa kolumna |
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

Pola exchange pojawiają się tylko wtedy, gdy aktywne są zawody.

Dolny wiersz zawiera:

- **Comment**,
- **Keep** — zachowuje Comment między QSO,
- **Retain** — zachowuje cały formularz po zapisaniu.

Pola takie jak Band, Mode i Submode można przełączać przez **PgUp/PgDn**.

### Ścieżka, azymut i badge

Gdy oba grid locatory są znane, CQOps pokazuje odległość i azymut.

QSO form może także pokazywać badge:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Zapisywanie

| Klawisz | Akcja |
|---|---|
| Enter | Zapisz QSO |
| Ctrl+S | Zapisz QSO z dowolnego pola |
| Esc | Anuluj potwierdzenie duplikatu |
| Enter na potwierdzeniu DUPE | Zapisz duplikat mimo ostrzeżenia |

---

## Logbook Editor i ADIF

Logbook Editor otwiera się klawiszem **F8**.

Służy do:

- przeglądania QSO,
- edycji inline,
- usuwania QSO,
- importu ADIF,
- eksportu ADIF,
- uploadu do Wavelog,
- downloadu z Wavelog,
- operacji związanych z zawodami.

### Edycja QSO

1. Wybierz wiersz przez **↑/↓**.
2. Naciśnij **Enter** lub **e**.
3. Edytuj QSO.
4. Zapisz przez **Ctrl+S**.

Zmiany od razu pojawiają się w Recent QSOs.

### Import i eksport ADIF

CQOps obsługuje import i eksport ADIF 3.1.7.

| Akcja | Skrót |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

Import waliduje rekordy, pomija duplikaty i pokazuje podsumowanie. Importowane QSO są oznaczane do uploadu do Wavelog, jeśli synchronizacja Wavelog jest skonfigurowana.

Eksport może obejmować wszystkie QSO albo QSO filtrowane zawodami. `CONTEST_ID` jest zachowywany.

### Obsługa trybów cyfrowych

Obsługa mode i submode jest zgodna z ADIF 3.1.7 w zakresie opisanym w tej instrukcji:

- FT8 jest eksportowany jako osobny mode.
- FT4 i FT2 są eksportowane jako MFSK z odpowiednim submode.
- Importowane starsze rekordy MFSK + FT8 są normalizowane do samodzielnego FT8.

QSO form ma osobne pola **Mode** i **Submode**. Oba można przełączać przez **PgUp/PgDn**.

---

## Zawody

Zawody dodają pola exchange i obsługę numerów seryjnych do QSO form.

Zawody tworzy się lub konfiguruje w Logbook Editor przez **Ins**.

Konfiguracja zawodów obejmuje:

- nazwę zawodów,
- datę,
- ADIF contest ID,
- szablony exchange.

### Markery szablonu

| Marker | Zastępowany przez |
|---|---|
| `@rst` | RST wysłany lub odebrany |
| `@serial` | Automatycznie zwiększany numer seryjny |
| `@call` | Twój znak |
| `@grid` | Twój grid locator |
| `@name` | Nazwa operatora z profilu operatora |

**Ctrl+C** przełącza aktywne zawody.

Gdy zawody są aktywne:

- QSO form pokazuje pola exchange,
- numery seryjne zwiększają się automatycznie,
- Recent QSOs mogą filtrować QSO zawodowe,
- eksport ADIF zachowuje `CONTEST_ID`.

---

## Favorites, REF i Band Plan

### Favorites

Favorites przechowują presety frequency, mode i band w 10 slotach.

| Skrót | Akcja |
|---|---|
| Alt+0–9 | Przywołaj favorite |
| Alt+Shift+0–9 | Zapisz aktualną frequency, mode i band jako favorite |

Favorites są przechowywane w konfiguracji i współdzielone między logbookami.

Przykład:

1. Wpisz `145.55`.
2. Ustaw mode na `FM`.
3. Ustaw band na `2m`.
4. Naciśnij **Alt+Shift+1**.
5. Później naciśnij **Alt+1**, aby przywołać preset.

### REF Lookup

REF Lookup otwiera się przez **F6**.

Wyszukuje:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Możesz szukać po prefiksie, nazwie lub oznaczeniu referencji. Wybrane referencje mogą wypełnić QSO form.

### Band Plan Browser

Band Plan Browser otwiera się przez **F7**.

Daje szybki dostęp do:

- pasm amatorskich,
- zakresów VHF/UHF,
- CB,
- PMR446,
- presetów broadcast.

Wybrana częstotliwość może zostać użyta do strojenia aktywnego rigu. Dane band plan można też eksportować jako Markdown.

---

## Integracje

Wszystkie integracje są opcjonalne. Lokalne logowanie działa bez nich.

### QRZ.com

QRZ.com lookup wymaga internetu i subskrypcji QRZ XML.

Na QSO form naciśnij **Ins**, aby wypełnić pola callbook, takie jak:

- name,
- QTH,
- grid,
- country,
- CQ/ITU zones,
- DXCC,
- continent.

Partner view na **F2** może pokazać zdjęcie operatora, jeśli jest dostępne.

### Wavelog

Integracja Wavelog obsługuje:

- upload,
- incremental download,
- worked/confirmed lookup.

Wavelog jest konfigurowany per aktywny logbook:

- URL,
- API key,
- station profile ID.

CQOps zawsze zapisuje QSO najpierw lokalnie. Błąd uploadu do Wavelog nie usuwa danych lokalnych.

### flrig

Integracja flrig używa XML-RPC przez HTTP.

Domyślny endpoint:

```text
localhost:12345
```

CQOps może odczytać:

- frequency,
- mode,
- power.

Split mapuje VFO A na Frequency i VFO B na Freq RX.

### Hamlib / rigctld

Sterowanie rig przez Hamlib używa demona TCP `rigctld`.

W zależności od radia i backendu CQOps może odpytać:

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps obsługuje brak wsparcia dla nazw VFO tak łagodnie, jak to możliwe.

### Hamlib Rotor / rotctld

Sterowanie rotorem używa Hamlib `rotctld`.

CQOps obsługuje:

- azimuth,
- elevation,
- stop commands.

| Skrót | Akcja |
|---|---|
| Ctrl+←/→ | Zmień azimuth o 5° |
| Ctrl+↑/↓ | Zmień elevation o 5° |
| Ctrl+A | Ustaw rotor na obliczony bearing |
| Ctrl+F1 | Zatrzymaj rotor |

### WSJT-X

Integracja WSJT-X używa wiadomości UDP z WSJT-X. CQOps parsuje wiadomości ADIF i może automatycznie logować zakończone QSO.

Etykieta rigu przyjmuje kolor akcentu, gdy WSJT-X nadaje. Jeśli operator raportowany przez WSJT-X nie zgadza się z aktywnym operatorem, CQOps pokazuje ostrzeżenie.

### DX Cluster

Integracja DX Cluster używa telnetu i wymaga internetu.

Domyślny serwer:

```text
dxspots.com:7300
```

Filtry obejmują:

- band,
- continent,
- mode,
- age/time.

| Klawisz | Akcja |
|---|---|
| Enter | Wypełnij QSO form, dostrój rig i wróć do QSO |
| Space | Dostrój rig i pozostań na DX Cluster |
| Backspace | Wyczyść filtry |

### PSK Reporter

PSK Reporter wymaga internetu.

Dostarcza:

- spoty propagacyjne,
- filtry band/time/mode,
- ASCII world map na **F5**.

### APRS

APRS używa połączenia TCP do serwera APRS-IS i wymaga internetu.

Domyślny serwer:

```text
euro.aprs2.net:14580
```

CQOps może odbierać raporty pozycji od pobliskich stacji i pokazywać je na lokalnej mapie CQOps Live z:

- standardowymi symbolami,
- popupami callsign,
- auto-fit view,
- konfigurowalnym range circle.

CQOps może również wysyłać okresowy beacon z:

- znakiem stacji,
- SSID,
- grid locatorem,
- opcjonalnym komentarzem.

APRS konfiguruje się per logbook w:

```text
F9 → Logbooks → [active logbook] → APRS
```

### Solar Data

Solar data pochodzi z hamqsl.com i obejmuje:

- SFI,
- sunspot number,
- A/K indices,
- band-by-band conditions.

Aktualizacje live wymagają internetu. Dane w cache pozostają dostępne offline po udanym pobraniu.

---

## CQOps Live Dashboard

CQOps Live to wbudowany dashboard w przeglądarce pokazujący aktywność stacji w czasie rzeczywistym.

Przydaje się do:

- publicznych ekranów field day,
- ekranów stacji klubowej,
- monitoringu zawodów,
- obserwacji stacji z innego pokoju,
- stoisk eventowych lub targowych.

### Włączanie dashboardu

1. Naciśnij **F9**.
2. Otwórz **Integrations**.
3. Przejdź do **HTTP Server**.
4. Włącz **HTTP server**.
5. Opcjonalnie ustaw address i port.
6. Naciśnij **Ctrl+S**, aby zapisać.
7. Otwórz dashboard w przeglądarce.

Domyślne ustawienia:

| Ustawienie | Domyślnie |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

Serwer startuje od razu po zapisaniu.

### Tryby wyświetlania

CQOps Live ma dwa tryby.

#### Overview mode

Widoczny, gdy nie jest aktywnie robiony żaden znak.

Pokazuje:

- live Leaflet map,
- dzisiejsze markery QSO,
- great-circle paths,
- tabelę recent QSOs,
- informacje o stacji,
- statystyki,
- rate tracking 5 minut, 15 minut i 1 godzina,
- top operators,
- najdłuższe QSO.

#### Active / Now Working mode

Widoczny, gdy pracujesz z konkretnym znakiem.

Pokazuje:

- duży callsign,
- submode indicator,
- zdjęcie QRZ, jeśli dostępne,
- badge band i mode,
- wskaźniki DUPE / NEW CALL / NEW DXCC,
- distance i bearing,
- wyróżnioną przerywaną ścieżkę na mapie od gridu stacji do gridu korespondenta.

### Info box

Info box nad local map przełącza moduły co 5 sekund:

- band conditions,
- solar activity,
- geomagnetic field,
- ostatni spot DX Cluster,
- liczba raportów PSK Reporter per band.

Band conditions zawsze renderuje się na pełną szerokość.

### Weather row

Weather row pokazuje aktualne warunki Open-Meteo dla grid locatora stacji:

- temperature,
- wind,
- humidity,
- icon.

Dane pogodowe są pobierane po stronie przeglądarki i łagodnie degradują się offline.

### Local map

Prawa local map może pokazywać:

- stacje APRS,
- standardowe symbole APRS,
- range circle,
- popupy callsign,
- opcjonalny day/night terminator overlay,
- opcjonalny RainViewer weather radar overlay.

### Aktualizacje live i wydajność

CQOps Live aktualizuje się przez Server-Sent Events (SSE). Odświeżanie strony nie jest potrzebne.

Dashboard jest zaprojektowany pod słabszy sprzęt:

- przeglądarka renderuje mapę,
- przeglądarka liczy odległości,
- przeglądarka liczy statystyki,
- CQOps wysyła lekkie aktualizacje JSON,
- gdy HTTP server jest wyłączony, port nie jest otwierany i nie działają goroutines dashboardu.

### Personalizacja dashboardu

W formularzu integracji HTTP Server można ustawić:

| Pole | Opis |
|---|---|
| Header 1 | Główny tytuł w headerze i hero area. Domyślnie „CQOps Live”. |
| Header 2 | Podtytuł pod tytułem. Domyślnie „Fast, portable ham radio logger”. |
| Logo URL | Publicznie dostępny URL obrazka w lewym górnym rogu. Domyślnie logo CQOps. |
| Event Start | Data w formacie `YYYY-MM-DD`. Filtruje statystyki i listy QSO od tej daty. |

---

## Konfiguracja

Konfigurację otwiera się przez **F9**.

### Pliki konfiguracyjne

| Platforma | Ścieżka config |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Wrażliwe dane są przechowywane oddzielnie w `secrets.enc` w tym samym katalogu konfiguracji.

Secrets są szyfrowane kluczem powiązanym z maszyną. Przy przenoszeniu konfiguracji na inny komputer dane uwierzytelniające trzeba wpisać ponownie.

### Menu konfiguracji

| Menu | Konfiguruje |
|---|---|
| Station | Callsign, grid, CQ/ITU zone, IARU region, references |
| Rig | Rig presets, model, antenna, power, backend, rotor, WSJT-X |
| Wavelog | URL, API key, station profile ID |
| QRZ | Username i password |
| DX Cluster | Host, port, login |
| Operators | Profile operatorów |
| Logbooks | Ustawienia station, Wavelog, contest, operator i APRS per logbook |
| Notifications | QSO saved alerts, Wavelog status, dupe beep, error sounds |
| General | Timezone, distance units, map, debug mode |

### Multi-logbook

Używaj wielu logbooków do pracy domowej, portable, zawodów i klubu.

**Ctrl+L** przełącza aktywny logbook.

Każdy logbook ma własne:

- station details,
- Wavelog settings,
- contest settings,
- operator settings.

### Multi-operator

Profile operatorów zawierają:

- znak operatora,
- nazwę operatora.

**Ctrl+O** przełącza aktywnego operatora.

Aktywny operator jest zapisywany w polu ADIF `OPERATOR` i jest używany przy uploadach Wavelog.

### Multi-rig

Presety rigu przechowują:

- backend,
- model,
- antenna,
- power,
- rotor settings,
- WSJT-X settings.

**Ctrl+R** przełącza aktywny rig.

### Szyfrowane secrets

Od wersji v0.8.7 dane uwierzytelniające są przechowywane w postaci zaszyfrowanej.

| Element | Wartość |
|---|---|
| Secrets file | `secrets.enc` |
| Location | Ten sam katalog co `config.yaml` |
| Unix permissions | `0600`, gdzie obsługiwane |
| Encryption | AES-256-GCM z kluczem powiązanym z maszyną |
| Protected data | QRZ password, DX Cluster login, Wavelog API keys |

Plaintext secrets ze starszych konfiguracji są migrowane przy pierwszym starcie.

Jeśli `secrets.enc` jest uszkodzony, CQOps startuje z ostrzeżeniem i prosi o ponowne wpisanie danych.

---

## Skróty klawiaturowe

### Globalne

| Klawisz | Akcja |
|---|---|
| F1 | QSO form i Recent QSOs |
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
| Ctrl+L | Przełącz aktywny logbook |
| Ctrl+R | Przełącz aktywny rig |
| Ctrl+C | Przełącz aktywne zawody |
| Ctrl+O | Przełącz aktywnego operatora |
| Esc | Wróć do poprzedniego ekranu |

### QSO form

| Klawisz | Akcja |
|---|---|
| Tab | Następne pole |
| Shift+Tab | Poprzednie pole |
| ↑ / ↓ | Ruch w kolumnie |
| Enter | Zapisz QSO, z potwierdzeniem duplikatu jeśli potrzebne |
| Ctrl+S | Zapisz QSO z dowolnego pola |
| Del | Wyczyść wszystkie pola formularza |
| Ins | Lookup: QRZ, Wavelog, DXCC i duplicate check |
| PgUp / PgDn | Przełącz band, mode lub submode |
| Ctrl+D | Otwórz spot dialog |
| Ctrl+T | Przełącz Keep Comment |
| Ctrl+←/→ | Zmień azimuth rotora o 5° |
| Ctrl+↑/↓ | Zmień elevation rotora o 5° |
| Ctrl+A | Ustaw rotor na bearing z własnego gridu do gridu korespondenta |
| Ctrl+F1 | Zatrzymaj rotor |
| Alt+0–9 | Przywołaj favorite |
| Alt+Shift+0–9 | Zapisz aktualną frequency, mode i band jako favorite |

### Logbook Editor

| Klawisz | Akcja |
|---|---|
| ↑ / ↓ | Nawigacja po wierszach |
| PgUp / PgDn | Poprzednia lub następna strona |
| Home / End | Pierwszy lub ostatni wiersz |
| Enter / e | Edytuj wybrane QSO |
| Delete | Usuń wybrane QSO |
| p | Purge wszystkich QSO |
| Ctrl+C | Przełącz filtr zawodów |
| Ctrl+E | Export ADIF |
| Ctrl+I / Tab | Import ADIF |
| w | Upload niewysłanych QSO do Wavelog |
| Ctrl+W | Download kontaktów z Wavelog |
| Esc / F6 | Zamknij editor i wróć do QSO form |

### DX Cluster

| Klawisz | Akcja |
|---|---|
| ↑ / ↓ | Nawigacja po spotach |
| Enter | Wypełnij QSO form, dostrój rig i wróć do QSO |
| Space | Dostrój rig do wybranego spotu i pozostań na DX Cluster |
| Home | Następny filtr band |
| End | Poprzedni filtr band |
| `\` | Przełącz filtr continent |
| Ins | Następny filtr mode |
| Del | Poprzedni filtr mode |
| PgUp | Następny filtr time |
| PgDn | Poprzedni filtr time |
| Backspace | Wyczyść wszystkie filtry |
| Esc / F4 | Wróć do QSO form |

### Partner view

| Klawisz | Akcja |
|---|---|
| F2 | Przełącz Partner view → Photo → Back |
| Esc / F1 | Wróć do QSO form |

---

## Rozwiązywanie problemów

### CQOps nie startuje

Sprawdź:

- rozmiar terminala to co najmniej 80×24,
- w Windows używany jest Windows Terminal,
- czy start nie blokuje się na sieci:

  ```bash
  cqops --offline
  ```

Logi:

| Platforma | Ścieżka logów |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### Rig się nie łączy

Dla flrig:

- sprawdź, czy flrig działa,
- sprawdź port w aktywnym pressecie rigu,
- domyślny port to `12345`.

Dla Hamlib:

- sprawdź, czy działa `rigctld`,
- sprawdź host i port,
- sprawdź, czy radio/backend obsługuje wymagane dane.

Kolory status label pomagają w diagnozie:

| Kolor | Znaczenie |
|---|---|
| Biały/domyślny | Połączone |
| Żółty | Wyłączone lub łączy się |
| Czerwony | Błąd |

Reconnect toasts mogą być ukryte. CQOps może ponawiać połączenie po cichu.

### WSJT-X nie loguje automatycznie

Sprawdź:

- WSJT-X **Settings → Reporting → UDP Server**,
- UDP host i port zgodne z aktywnym presetem rigu w CQOps,
- używana jest wersja WSJT-X 2.6 lub nowsza,
- WSJT status label jest aktywny,
- aktywny logbook jest poprawny,
- aktywny operator jest poprawny.

### Upload Wavelog nie działa

Sprawdź:

- Wavelog URL,
- API key,
- station profile ID,
- status label **WL**.

Błędy uploadu są pokazywane jako toasts. QSO pozostają zapisane lokalnie nawet po nieudanym uploadzie. Błąd pojedynczego QSO nie blokuje reszty paczki.

### Problemy z config file

Config file:

| Platforma | Ścieżka |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets file:

```text
secrets.enc
```

Secrets file jest przechowywany w tym samym katalogu co `config.yaml`.

Jeśli config jest uszkodzony, przenieś go lub usuń i uruchom CQOps ponownie. Setup wizard utworzy świeżą konfigurację.

Pole `last_fetched_id` pojawia się dopiero po udanym pobraniu z Wavelog.

### Problemy z wydajnością

Spróbuj:

- wyłączyć map rendering w General settings,
- wyłączyć Solar panel, jeśli nie jest potrzebny,
- unikać ekranów obciążających sieć, takich jak DX Cluster i PSK Reporter, gdy pracujesz offline,
- użyć `cqops --offline`, gdy sieć jest niestabilna.

---

## Zgłaszanie błędów

Przed zgłoszeniem błędu:

1. Włącz **Debug mode** w **F9 → General → Debug** albo ustaw:

   ```yaml
   debug: true
   ```

   w `config.yaml`.

2. Odtwórz problem.
3. Dołącz właściwy log.

Zgłaszaj problemy na GitHub:

<https://github.com/szporwolik/cqops/issues>

Dołącz:

- wersję CQOps z `cqops --version`,
- system operacyjny,
- terminal emulator,
- kroki do odtworzenia,
- właściwy debug log.
