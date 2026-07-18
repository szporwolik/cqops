---
title: Podręcznik użytkownika CQOps
description: Instrukcja instalowania, konfigurowania i używania CQOps — szybkiego, terminalowego loggera krótkofalarskiego
---

# Podręcznik użytkownika CQOps

CQOps to szybki, terminalowy logger krótkofalarski przeznaczony dla operatorów, którzy oczekują niezawodnego logowania z klawiatury i małego obciążenia systemu. Został zaprojektowany do pracy w shacku, podczas aktywacji terenowych, na stacjach klubowych, podczas field day oraz na urządzeniach klasy Raspberry Pi i starszych laptopach.

CQOps zawsze najpierw zapisuje QSO lokalnie. Integracje internetowe są opcjonalne.

## Spis treści

1. [Czym jest CQOps](#what-cqops-is)
2. [Pobieranie i instalacja](#download-and-installation)
3. [Pierwsze uruchomienie](#first-launch)
4. [Zalogowanie pierwszego QSO](#log-your-first-qso)
5. [Ekran główny](#main-screen)
6. [Typowe scenariusze pracy](#common-workflows)
7. [Logowanie QSO](#qso-logging)
8. [Edytor logu i ADIF](#logbook-editor-and-adif)
9. [Zawody](#contests)
    - [Konfigurowanie zawodów](#setting-up-a-contest)
    - [Dolny pasek stanu](#bottom-status-bar)
    - [Panel statystyk zawodów](#contest-statistics-panel)
    - [Eksport ADIF zawodów](#contest-adif-export)
    - [Działanie trybu zawodów](#contest-mode-behavior)
10. [Ulubione, referencje i band plany](#favorites-references-and-band-plans)
11. [Integracje](#integrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Konfiguracja](#configuration)
14. [Skróty klawiaturowe](#keyboard-shortcuts)
15. [Rozwiązywanie problemów](#troubleshooting)
16. [Zgłaszanie błędów](#reporting-bugs)

---

<a id="what-cqops-is"></a>
## Czym jest CQOps

CQOps opiera się na szybkim wprowadzaniu QSO, lokalnym zapisie danych i praktycznej pracy terenowej.

### Główne założenia

- **Praca terminalowa** — zoptymalizowana do obsługi z klawiatury.
- **Logowanie offline-first** — lokalne logowanie QSO działa bez dostępu do internetu. Zawiera wbudowaną mapę świata dla dashboardu, która działa w pełni offline.
- **Małe obciążenie** — odpowiednie dla systemów klasy Raspberry Pi, starszych laptopów i współdzielonych komputerów stacyjnych.
- **Przenośna konstrukcja** — program jest rozpowszechniany jako pojedynczy plik binarny Go.
- **Wiele logów** — przydatne dla logów osobistych, terenowych, zawodów i stacji klubowych.
- **Wielu operatorów** — rozwiązanie przydatne przy zmianowej pracy operatorów i na współdzielonych stacjach klubowych.
- **Wiele radiostacji** — każdy preset radiostacji może przechowywać własny backend oraz ustawienia WSJT-X.
- **Opcjonalne integracje** — Multi-provider callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info), Wavelog, DX Cluster, PSK Reporter, GPS, APRS, sterowanie radiostacją, sterowanie rotorem, dane solarne oraz przeglądarkowy CQOps Live dashboard.

Lokalne logowanie nie wymaga internetu. Funkcje sieciowe są pomijane w trybie `--offline`.

### Dla kogo jest CQOps

CQOps dobrze sprawdza się dla:

- operatorów terenowych,
- aktywatorów SOTA i POTA,
- stacji klubowych,
- zespołów field day,
- operatorów preferujących pracę w terminalu,
- stacji wymagających szybkiego przełączania operatorów, logów lub radiostacji.

CQOps nie ma zastępować każdej funkcji pełnego loggera desktopowego ani internetowej platformy logowej. Koncentruje się na szybkim logowaniu terminalowym, pracy terenowej, działaniu offline i współdzielonych stanowiskach operatorskich.

### Praca klubowa i stacje współdzielone

CQOps powstał z myślą o środowisku klubów krótkofalarskich. Aktywny operator jest zawsze widoczny na pasku stanu — **jeden rzut oka** wystarcza, aby sprawdzić, kto aktualnie obsługuje stację. Zmiana operatora wymaga jednego skrótu (`Ctrl+O`) i działa natychmiast: znak wywoławczy oraz nazwa operatora są zapisywane w każdym kolejnym QSO. Bez wylogowywania, pytania o hasło i przerywania pracy.

Logi, presety radiostacji i zawody zmienia się w ten sam sposób — `Ctrl+L`, `Ctrl+R`, `Ctrl+C`. Stacja klubowa ze zmieniającymi się operatorami, wieloma radiostacjami i kilkoma aktywnymi zawodami może przełączyć cały kontekst w czasie krótszym niż sekunda, bez używania myszy.

Podczas field day i imprez publicznych **CQOps Live dashboard** może wyświetlać na dużym ekranie mapę w czasie rzeczywistym, strumień QSO i statystyki. Goście oraz członkowie klubu mogą obserwować pracę stacji bez tłoczenia się przy terminalu operatora. Wystarczy włączyć integrację **HTTP Server** i otworzyć stronę na dowolnym urządzeniu z przeglądarką internetową.

---

<a id="download-and-installation"></a>
## Pobieranie i instalacja

Lista wszystkich wydań:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Pakiet | Link | Uwagi |
|---|---|---|
| Instalator | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Zalecany dla większości użytkowników. Dodaje CQOps do menu Start i zmiennej PATH. |
| Przenośny ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Rozpakuj i uruchom bez instalowania. |

### Linux — Debian / Ubuntu / Pop!_OS / Linux Mint

Dodaj repozytorium Cloudsmith APT, a następnie zainstaluj:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

Lub pobierz pakiet `.deb` bezpośrednio:

| Architektura | Link | Zastosowanie |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Większość komputerów Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-bitowe systemy ARM |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-bitowy Raspberry Pi OS |

Zainstaluj pobrany pakiet:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Fedora / RHEL / Rocky / AlmaLinux

Dodaj repozytorium Cloudsmith RPM, a następnie zainstaluj:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

### Linux — Arch / Manjaro / CachyOS

Zainstaluj z AUR:

```bash
# CachyOS (domyślnie używa paru)
paru -S cqops-bin

# Arch / Manjaro
yay -S cqops-bin
```

Dostępne również przez `pacaur`, `aura` lub ręczne `makepkg`. PKGBUILD na [aur.archlinux.org/packages/cqops-bin](https://aur.archlinux.org/packages/cqops-bin).

### Linux — przenośne archiwum Tarball

| Architektura | Link | Zastosowanie |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Większość komputerów Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-bitowe systemy ARM |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-bitowy Raspberry Pi OS |

### macOS

| Architektura | Link | Zastosowanie |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Komputery Mac M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Komputery Mac z procesorem Intel |

Instalacja ręczna:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Kompilowanie ze źródeł

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

Kompilowanie ze źródeł wymaga Go 1.26 lub nowszego.

### Wymagania terminala

| Wymaganie | Wartość |
|---|---|
| Minimalny rozmiar terminala | 80×24 znaki |
| Zalecany rozmiar terminala | 80×43 znaki lub więcej |
| Zalecany terminal w Windows | Windows Terminal |
| Terminal obsługujący Kitty graphics | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) lub [WezTerm](https://wezfurlong.org/wezterm/) |

### Podstawowe polecenia

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

<a id="first-launch"></a>
## Pierwsze uruchomienie

Przy pierwszym uruchomieniu CQOps otwiera kreator konfiguracji. Do lokalnego logowania wymagane są wyłącznie podstawowe informacje o stacji. Integracje sieciowe można pominąć i skonfigurować później.

### Strony kreatora

| Page | Co konfiguruje |
|---|---|
| Station & Logbook | Początkowy log, znak stacji, operator, lokator, opcjonalne referencje i strefy oraz URL/API/station profile ID Wavelog |
| Rig | Preset radiostacji, model, antena, moc, backend, opcjonalny rotor i opcjonalne ustawienia UDP WSJT-X |
| Integrations | Ustawienia wyszukiwania callbooka (QRZ.com, HamQTH, QRZ.RU, Callook.info) |
| General | Strefa czasowa IANA |
| Summary | Przegląd i zapis ustawień |

Obsługiwane backendy radiostacji:

- None,
- flrig,
- Hamlib `rigctld`.

### Nawigacja w kreatorze

| Key | Action |
|---|---|
| Ctrl+S | Sprawdź dane i przejdź dalej; na stronie **Summary** zapisz ustawienia i uruchom CQOps |
| Esc | Wróć |
| F10 | Zakończ |
| Tab / Shift+Tab | Przejdź między polami |
| Space | Przełącz pole wyboru |

Ustawienia kreatora można później zmienić za pomocą **F9**.

---

<a id="log-your-first-qso"></a>
## Zalogowanie pierwszego QSO

1. Uruchom CQOps:

   ```bash
   cqops
   ```

2. Ukończ kreator konfiguracji, podając co najmniej swój znak wywoławczy i lokator.

3. Otwórz **QSO form** klawiszem **F1**.

4. Wprowadź znak korespondenta. CQOps automatycznie zamienia znaki wywoławcze na wielkie litery.

5. Uzupełnij pozostałe pola. Jeżeli aktywna radiostacja jest połączona przez flrig lub Hamlib, CQOps może automatycznie wypełnić częstotliwość, pasmo, tryb i submode.

6. Naciśnij **Enter**, aby zapisać.

7. Jeżeli pojawi się ostrzeżenie **DUPE!**, ponownie naciśnij **Enter**, aby mimo to zapisać QSO, albo **Esc**, aby anulować.

Zapisane QSO natychmiast pojawi się w tabeli **Recent QSOs**.

---

<a id="main-screen"></a>
## Ekran główny

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

**Status bar** pokazuje:

- wersję CQOps,
- aktywny log,
- aktywną radiostację,
- znak stacji,
- aktywnego operatora,
- etykiety stanu integracji,
- czas lokalny oznaczony `L`,
- czas UTC oznaczony `Z`.

Typowe etykiety to **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** i **GPS**. Etykieta **GPS** korzysta z tej samej konwencji kolorów: czerwony oznacza brak połączenia, żółty połączenie bez fixa, a biały uzyskaną pozycję.

| Kolor | Znaczenie |
|---|---|
| Biały/domyślny | Połączone lub aktywne |
| Żółty | Wyłączone, łączenie lub oczekiwany brak połączenia |
| Czerwony | Błąd lub rozłączenie |
| Kolor akcentu + pogrubienie | WSJT-X nadaje |

### Główne zakładki

| Key | Tab | Screen |
|---|---|---|
| F1 | QSO | **QSO form** i **Recent QSOs** |
| F2 | QRZ | **Partner view**: dane callbooka, mapa, statystyki, zdjęcie |
| F4 | DXC | Spoty **DX Cluster** i filtry |
| F5 | HRD | Spoty **PSK Reporter** i mapa propagacji |
| F6 | REF | Wyszukiwanie referencji SOTA/POTA/WWFF/IOTA |
| F7 | BPL | **Band Plan Browser** |
| F8 | LOG | **Logbook Editor**, ADIF, synchronizacja Wavelog |
| F9 | CFG | Menu konfiguracji |

**Help bar** pokazuje skróty właściwe dla aktywnego ekranu. Naciśnij **?**, aby otworzyć pełną nakładkę **Help**.

---

<a id="common-workflows"></a>
## Typowe scenariusze pracy

### Praca portable, SOTA lub POTA

Przed wyjazdem:

1. Uruchom CQOps co najmniej raz z dostępem do internetu.
2. Pozwól CQOps pobrać lub odświeżyć dane w cache, takie jak dane solarne, dane REF i prefiksy DXCC.
3. Sprawdź, czy panel **Solar** pokazuje dane.
4. Sprawdź, czy wyszukiwanie **REF** pod **F6** zwraca wyniki.

W terenie:

1. Uruchom CQOps w trybie offline:

   ```bash
   cqops --offline
   ```

2. Loguj normalnie. QSO są zapisywane lokalnie.
3. Po odzyskaniu dostępu do internetu otwórz **F8** i naciśnij **w**, aby wysłać niewysłane QSO do Wavelog.

### Współdzielona stacja klubowa i praca hot-seat

1. Otwórz **F9 → Operators**.
2. Naciśnij **Ins**, aby dodać profile operatorów.
3. W **QSO form** naciśnij **Ctrl+O**, aby zmienić aktywnego operatora.
4. Przed zapisaniem sprawdź aktywnego operatora na pasku stanu.
5. Użyj **Retain**, gdy wielu operatorów musi logować podobne łączności bez ponownego wpisywania całego formularza.

Aktywny operator jest zapisywany w polu ADIF `OPERATOR`.

### Logi osobiste i klubowe

1. Otwórz **F9 → Logbooks**.
2. Naciśnij **Ins**, aby utworzyć każdy log.
3. W **QSO form** naciśnij **Ctrl+L**, aby zmienić aktywny log.
4. Przed zapisaniem sprawdź aktywny log na pasku stanu.

Każdy log może przechowywać własne dane stacji, ustawienia Wavelog, ustawienia zawodów i operatorów.

### Wiele radiostacji

1. Otwórz **F9 → Rigs**.
2. Naciśnij **Ins**, aby utworzyć presety radiostacji.
3. Wybierz backend: None, flrig lub Hamlib.
4. W **QSO form** naciśnij **Ctrl+R**, aby zmienić aktywną radiostację.

Preset radiostacji może obejmować backend, model, antenę, moc, ustawienia rotora i ustawienia UDP WSJT-X.

### Praca emisjami cyfrowymi z WSJT-X

Gdy integracja UDP WSJT-X jest włączona, CQOps może odbierać komunikaty ADIF z WSJT-X i automatycznie logować zakończone cyfrowe QSO.

QSO logowane automatycznie:

- są zapisywane w aktywnym logu,
- natychmiast pojawiają się w **Recent QSOs**,
- pomijają duplikaty,
- dziedziczą aktywne contest ID,
- mogą być automatycznie wysyłane do Wavelog, gdy Wavelog jest skonfigurowany i osiągalny.

Jeżeli operator zgłoszony przez WSJT-X nie odpowiada aktywnemu operatorowi w CQOps, program pokazuje ostrzeżenie.

Przed dłuższą sesją cyfrową sprawdź:

- aktywny log,
- aktywnego operatora,
- aktywne zawody,
- etykietę stanu **WSJT**.

### Synchronizacja Wavelog

CQOps zawsze najpierw zapisuje QSO lokalnie. Synchronizacja Wavelog jest opcjonalna.

| Action | Where | Shortcut | Notes |
|---|---|---|---|
| Upload unsent QSOs | **Logbook Editor** | `w` | Wysyłanie w partiach po 50 |
| Download from Wavelog | **Logbook Editor** | `Ctrl+W` | Pobieranie przyrostowe z użyciem `last_fetched_id` |

Stan wysyłania jest śledzony dla każdego QSO:

- not sent,
- sent,
- error.

Jeżeli wysyłanie się nie powiedzie, QSO pozostaje w lokalnym logu i można ponowić próbę później. Operacja purge logu resetuje fetch ID do `0`, umożliwiając ponowne pobranie całego logu.

---

<a id="qso-logging"></a>
## Logowanie QSO

**QSO form** jest głównym ekranem logowania. Otwórz go klawiszem **F1**.

CQOps może uzupełniać pola z następujących źródeł:

| Źródło | Pola |
|---|---|
| flrig / Hamlib | Frequency, Freq RX przy pracy split, Mode, Submode |
| Callbook (QRZ.com / HamQTH / QRZ.RU / Callook.info) | Name, QTH, Grid, Country, CQ zone, ITU zone, DXCC, Continent, zdjęcie |
| Baza REF | Referencje SOTA, POTA, WWFF, IOTA |
| Wavelog lookup | Status worked/confirmed, jeżeli skonfigurowano |
| Dane DXCC/prefiksów | Prefiks i dane kraju |

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

Pola **Exch sent** i **Exch rcvd** pojawiają się tylko wtedy, gdy aktywne są zawody.

Dolny wiersz zawiera:

- **Comment**,
- **Keep** — zachowuje pole **Comment** między kolejnymi QSO,
- **Retain** — zachowuje cały formularz po zapisaniu.

Pola takie jak **Band**, **Mode** i **Submode** można zmieniać klawiszami **PgUp/PgDn**.

### Trasa, azymut i oznaczenia

Gdy znane są oba lokatory, CQOps pokazuje odległość i azymut.

**QSO form** może również wyświetlać oznaczenia:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Zapisywanie

| Key | Action |
|---|---|
| Enter | Zapisz QSO |
| Ctrl+S | Wyślij spot DX na podstawie wypełnionego formularza |
| Esc | Anuluj potwierdzenie duplikatu |
| Enter w potwierdzeniu DUPE | Zapisz duplikat mimo ostrzeżenia |

---

<a id="logbook-editor-and-adif"></a>
## Edytor logu i ADIF

Otwórz **Logbook Editor** klawiszem **F8**.

Służy on do:

- przeglądania QSO,
- edycji wierszy,
- usuwania QSO,
- importu ADIF,
- eksportu ADIF,
- wysyłania do Wavelog,
- pobierania z Wavelog,
- operacji związanych z zawodami.

### Edytowanie QSO

1. Wybierz wiersz klawiszami **↑/↓**.
2. Naciśnij **Enter** lub **e**.
3. Edytuj QSO.
4. Zapisz za pomocą **Ctrl+S**.

Zmiany natychmiast pojawią się w **Recent QSOs**.

### Import i eksport ADIF

CQOps obsługuje import i eksport ADIF 3.1.7.

| Action | Shortcut |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

Import sprawdza rekordy, pomija duplikaty i pokazuje podsumowanie. Zaimportowane QSO są oznaczane do wysłania do Wavelog, jeżeli synchronizacja Wavelog jest skonfigurowana.

Eksport może obejmować wszystkie QSO lub QSO przefiltrowane według zawodów. Pole `CONTEST_ID` jest zachowywane.

### Obsługa emisji cyfrowych

Obsługa trybu i submode jest zgodna z ADIF 3.1.7 w sposób opisany w tym podręczniku:

- FT8 jest eksportowane jako samodzielny mode.
- FT4 i FT2 są eksportowane jako MFSK z odpowiednim submode.
- Importowane starsze rekordy MFSK + FT8 są normalizowane do samodzielnego FT8.

**QSO form** ma oddzielne pola **Mode** i **Submode**. Oba można zmieniać za pomocą **PgUp/PgDn**.

---

<a id="contests"></a>
## Zawody

CQOps zawiera lekki panel logowania zawodów przeznaczony do **okazjonalnego udziału w zawodach**. Nie zastępuje wyspecjalizowanych loggerów zawodów, takich jak N1MM, Win-Test czy TR4W. Przy poważnej pracy multi-op, multi-radio lub w kategorii assisted należy używać loggera stworzonego specjalnie do zawodów. CQOps sprawdzi się, gdy chcesz rozdać kilka punktów, rekreacyjnie śledzić tempo albo zalogować kilka zawodowych QSO podczas aktywacji SOTA/POTA bez opuszczania codziennego loggera.

<a id="setting-up-a-contest"></a>
### Konfigurowanie zawodów

Utwórz lub skonfiguruj zawody w **Logbook Editor** za pomocą **Ins**.

Konfiguracja zawodów obejmuje:

- nazwę zawodów,
- datę,
- ADIF contest ID,
- szablony wymiany.

#### Znaczniki szablonu

| Marker | Replaced with |
|---|---|
| `@rst` | Wysłany lub odebrany RST |
| `@serial` | Automatycznie zwiększany numer seryjny |
| `@cqz` | Strefa CQ stacji DX |
| `@mycqz` | Twoja strefa CQ |
| `@itu` | Strefa ITU stacji DX |
| `@myitu` | Twoja strefa ITU |
| `@grid` | Lokator stacji DX |
| `@mygrid` | Twój lokator |

Naciśnij **Ctrl+C**, aby przełączać aktywne zawody, albo wybierz je z menu **Contest** (**F7**). Pola wymiany pojawiają się automatycznie w **QSO form**, a numery seryjne są zwiększane automatycznie.

<a id="bottom-status-bar"></a>
### Dolny pasek stanu

Gdy zawody są aktywne, dolny pasek pokazuje bieżące podsumowanie:

```
 IARU-HF · IARU HF   45 QSOs   Started 16:13   Last 14:04 ago   Next #45   On 2:41
```

| Pole | Znaczenie |
|-------|---------|
| `IARU-HF` | ADIF ID zawodów, czyli identyfikator maszynowy |
| `· IARU HF` | Wyświetlana nazwa zawodów — pokazywana, gdy różni się od ID |
| `45 QSOs` | Łączna liczba QSO zalogowanych w tej sesji zawodów |
| `Started 16:13` | Godzina pierwszego QSO w zawodach w danym dniu |
| `Last 14:04 ago` | Czas od ostatniego QSO zawodów |
| `Next #45` | Numer seryjny wysyłany w następnym QSO |
| `On 2:41` | Łączny czas pracy — suma odstępów między QSO krótszych niż 30 minut |

Pole `Started` jest ukrywane w terminalach węższych niż 120 kolumn. Nazwa zawodów i czas `On` są ukrywane poniżej 100 kolumn.

<a id="contest-statistics-panel"></a>
### Panel statystyk zawodów

Gdy zawody są aktywne i terminal jest odpowiednio szeroki, po prawej stronie **QSO form** pojawia się kompaktowy panel statystyk z żółtą ramką:

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

| Wiersz | Pole | Znaczenie |
|-----|-------|---------|
| **Rate** | `2/h` | Tempo z ostatnich **10 QSO** — krótkoterminowa szybkość serii |
| | `--/h` | Tempo z ostatnich **100 QSO** — pokazuje `--` do czasu zalogowania 100 QSO |
| **Count** | `60m 0` | QSO zalogowane w ciągu ostatnich 60 minut |
| | `hr 0` | QSO zalogowane w bieżącej godzinie zegarowej, od `:00` |
| **Peak** | `1m120` | Najlepsze tempo jednominutowe: 120/h oznacza 2 QSO w tej minucie |
| | `10m 54` | Najlepsze 10-minutowe okno przesuwne: średnio 54/h |
| | `60m 29` | Najlepsze 60-minutowe okno przesuwne: średnio 29/h |
| **Avg** | `8/h` | Średnia sesji — liczba QSO podzielona przez liczbę godzin od pierwszego QSO |
| | `Sess 5:36` | Łączny czas sesji od pierwszego do ostatniego QSO, jako H:MM lub same minuty |
| **Chart** | `max 1` | W najbardziej aktywnej minucie wykonano 1 QSO. Słupki pokazują QSO na minutę |
| | `-60m…now` | Lewa krawędź oznacza 60 minut temu, prawa — bieżącą chwilę |

Wykres używa znaków blokowych Unicode (`█`) skalowanych do czterech wierszy pionowych słupków. Wartości **Peak** nie mają sufiksu `/h`, ponieważ sama nazwa oznacza już „na godzinę”. Czasy trwania nie pokazują sekund — przy odświeżaniu co minutę byłyby one tylko szumem.

<a id="contest-adif-export"></a>
### Eksport ADIF zawodów

Aby wysłać log zawodów, otwórz **Logbook Editor** (`Ctrl+E`), gdy zawody są aktywne. Po zastosowaniu filtra zawodów okno eksportu ADIF oferuje eksport **wyłącznie QSO należących do aktywnych zawodów**. Powstaje zgodny ze standardem plik ADIF 3.1.7 z polami wymiany zawodów, numerami seryjnymi i zachowanym ADIF ID zawodów, gotowy do wysłania do robota organizatora lub systemu sprawdzania logów.

<a id="contest-mode-behavior"></a>
### Działanie trybu zawodów

Gdy zawody są aktywne:

- **QSO form** pokazuje pola wymiany,
- numery seryjne są automatycznie zwiększane,
- **Recent QSOs** można filtrować do QSO zawodów,
- eksport ADIF zachowuje `CONTEST_ID`,
- **QSO form**, panel zawodów i panel **Solar** otrzymują żółtą ramkę dla łatwego rozróżnienia,
- spoty DXC są sprawdzane pod kątem duplikatów względem wszystkich QSO zawodów, a nie tylko dzisiejszych.

---

<a id="favorites-references-and-band-plans"></a>
## Ulubione, referencje i band plany

### Favorites

**Favorites** przechowuje presety częstotliwości, trybu i pasma w trzech slotach — wystarczająco dla najczęściej używanych częstotliwości wywoławczych. Skróty używają klawisza `Alt`, aby nie kolidować ze standardową edycją terminalową i działać niezawodnie w różnych terminalach.

| Shortcut | Action |
|---|---|
| Alt+Ins / Alt+Home / Alt+PgUp | Przywołaj favorite ze slotu 1, 2 lub 3 |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Zapisz bieżące frequency, mode i band w slocie 1, 2 lub 3 |

**Favorites** są przechowywane w konfiguracji i współdzielone między logami.

Przykład:

1. Wprowadź `145.55`.
2. Ustaw **Mode** na `FM`.
3. Ustaw **Band** na `2m`.
4. Naciśnij **Alt+Shift+Ins**, aby zapisać preset w slocie 1.
5. Później naciśnij **Alt+Ins**, aby przywołać preset.

### REF Lookup

Otwórz **REF Lookup** klawiszem **F6**.

Wyszukuje on:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Można wyszukiwać według prefiksu, nazwy lub oznaczenia referencji. Wybrane referencje mogą wypełniać **QSO form**.

### Band Plan Browser

Otwórz **Band Plan Browser** klawiszem **F7**.

Zapewnia szybki dostęp do:

- Amateur bands,
- VHF/UHF ranges,
- CB,
- PMR446,
- Broadcast presets,
- Portable — często używane częstotliwości pracy terenowej, w tym SOTA, POTA i kanały wywoławcze.

Wybranej częstotliwości można użyć do dostrojenia aktywnej radiostacji. Dane band planu można również wyeksportować jako Markdown.

---

<a id="integrations"></a>
## Integracje

Wszystkie integracje są opcjonalne. Lokalne logowanie działa bez nich.

### Callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info)

CQOps obsługuje wielu dostawców callbooka z kaskadowaniem według priorytetu.
Po naciśnięciu **Ins** w formularzu QSO, dostawcy są odpytywani w kolejności,
aż jeden z nich zwróci wynik:

1. **QRZ.com** — wymaga internetu i subskrypcji QRZ XML. Najbardziej kompletne dane.
2. **HamQTH** — darmowy serwis globalny. Dobry zasięg, wymaga darmowego konta.
3. **QRZ.RU** — darmowy serwis skupiony na Rosji i okolicznych krajach. Wymaga loginu API. Zwraca imię, QTH, grid, lat/lon, klasę, status LoTW/eQSL i zdjęcie.
4. **Callook.info** — darmowy serwis dla USA. Nie wymaga konta, szybkie wyszukiwanie FCC.

Jeśli dostawca o wyższym priorytecie zawiedzie lub jest wyłączony, następny jest
próbowany. Gdy **Base call fallback** jest włączone (domyślnie: tak), CQOps
próbuje również bazowego znaku (bez prefiksu lub sufiksu), jeśli pełny znak
nie zwróci wyniku.

Włącz i skonfiguruj dostawców w **F9 → Callbook**.

W **QSO form** naciśnij **Ins**, aby uzupełnić pola callbooka, takie jak:

- Name,
- QTH,
- Grid,
- Country,
- CQ/ITU zones,
- DXCC,
- Continent.

**Partner view** pod **F2** może pokazywać zdjęcie operatora, jeżeli jest dostępne.

> ⚠️ **Experimental.** Wyświetlanie zdjęć może korzystać z protokołu
> Kitty terminal graphics i wymaga zgodnego terminala: Kitty, Ghostty lub
> WezTerm. Włącz tę funkcję w **F9 → General → Kitty Graphics**. Standardowe
> terminale oraz sesje SSH bez przekazywania grafiki użyją zastępczego obrazu
> z glifów.

### Wavelog

Integracja Wavelog obsługuje:

- wysyłanie,
- pobieranie przyrostowe,
- wyszukiwanie statusu worked/confirmed.

Wavelog jest konfigurowany osobno dla aktywnego logu za pomocą:

- URL,
- API key,
- station profile ID.

CQOps zawsze najpierw zapisuje QSO lokalnie. Błąd wysyłania do Wavelog nie usuwa lokalnych danych.

### flrig

Integracja flrig używa XML-RPC przez HTTP.

Domyślny endpoint:

```text
localhost:12345
```

CQOps może odczytywać:

- frequency,
- mode,
- power.

Przy pracy split VFO A jest mapowane na **Frequency**, a VFO B na **Freq RX**.

### Hamlib / rigctld

Sterowanie radiostacją przez Hamlib korzysta z demona TCP `rigctld`.

W zależności od radiostacji i obsługi backendu CQOps może odczytywać:

- frequency,
- mode,
- VFO,
- split,
- power.

Gdy to możliwe, CQOps poprawnie obsługuje brak wsparcia nazw VFO.

### Hamlib Rotator / rotctld

> ⚠️ **Experimental.** Sterowanie rotorem jest funkcją eksperymentalną. Przed
> uruchomieniem zawsze sprawdź fizyczne ograniczenia anteny. Przygotuj się do
> natychmiastowego zatrzymania ruchu za pomocą **Alt+/**. Zachowaj ostrożność —
> błędna konfiguracja może uszkodzić rotor lub antenę.

Sterowanie rotorem korzysta z Hamlib `rotctld`.

CQOps obsługuje:

- azimuth,
- elevation,
- polecenia stop.

| Shortcut | Action |
|---|---|
| Alt+, | Zmień azimuth o −5° |
| Alt+. | Zmień azimuth o +5° |
| Alt+; | Zmień elevation o +5° |
| Alt+' | Zmień elevation o −5° |
| Alt+\ | Skieruj rotor na obliczony azymut trasy |
| Alt+/ | Zatrzymaj rotor |

### WSJT-X

Integracja WSJT-X używa komunikatów UDP z WSJT-X. CQOps analizuje komunikaty ADIF i może automatycznie logować zakończone QSO.

Etykieta radiostacji otrzymuje kolor akcentu, gdy WSJT-X nadaje. Jeżeli operator zgłoszony przez WSJT-X nie odpowiada aktywnemu operatorowi, CQOps pokazuje ostrzeżenie.

### GPS

CQOps może odczytywać pozycję z odbiornika GPS i używać jej jako lokatora stacji — jest to idealne rozwiązanie do pracy portable, mobile i field.

Obsługiwane są dwa backendy:

- **Serial** — bezpośrednie połączenie z odbiornikiem GPS przez port szeregowy, np. USB-to-serial, wbudowany COM lub `/dev/ttyUSB0`.
- **GPSD** — połączenie z serwerem [gpsd](https://gpsd.io/) przez TCP, domyślnie `127.0.0.1:2947`. Przydatne, gdy GPS jest współdzielony z innymi aplikacjami albo dostępny przez sieć.

Wskaźnik **GPS** na pasku stanu pokazuje:

| Kolor | Znaczenie |
|--------|---------|
| Czerwony `GPS` | Rozłączony / błąd |
| Żółty `GPS` | Połączony, ale jeszcze bez fixa |
| Biały `GPS` | Fix uzyskany, pozycja ustalona |

Po uzyskaniu fixa lokator stacji zostaje zastąpiony pozycją obliczoną z GPS i oznaczony `(GPS)` w wierszu stanu:

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Włącz **Grid from GPS** w ustawieniach **Station & Logbook**, aby używać lokatora GPS podczas logowania QSO, nadawania beaconów APRS, wyświetlania mapy dashboardu i obliczania odległości.

**Grid precision** konfiguruje się w menu **Integration** jako 10, 8 lub 6 znaków. Domyślnie używane jest 10 znaków, czyli około 25 m dokładności. Lokator jest zawsze obliczany wewnętrznie z pełną precyzją, a skracany dopiero w miejscu użycia.

### DX Cluster

Integracja **DX Cluster** używa telnetu i wymaga dostępu do internetu.

Domyślny serwer:

```text
dxspots.com:7300
```

Filtry obejmują:

- band,
- spotter continent,
- mode,
- age/time.

| Key | Action |
|---|---|
| Enter | Wypełnij **QSO form**, dostrój radiostację i wróć do **QSO** |
| Space | Dostrój radiostację i pozostań w **DX Cluster** |
| Backspace | Wyczyść filtry |

Gdy **DX Cluster** jest połączony, **QSO form** zyskuje dwie dodatkowe funkcje:

- **Send a spot** — po wypełnieniu formularza naciśnij **Ctrl+S**, aby otworzyć okno spotu i wysłać spot DX do klastra.
- **Nearest spots** — po dostrojeniu częstotliwości bezpośrednio w **QSO form** pojawia się maksymalnie trzy pobliskie spoty, dzięki czemu można sprawdzić aktywność bez opuszczania ekranu logowania. Naciśnij **Ctrl+P**, aby wypełnić znak na podstawie najbliższego spotu.

### PSK Reporter

Integracja **PSK Reporter** wymaga dostępu do internetu. Pozwala szybko sprawdzić rzeczywistą propagację — zobaczyć, kto odbiera Twój sygnał albo kogo odbierasz na danym paśmie w tej chwili.

Udostępnia:

- spoty propagacyjne,
- filtry band/time/mode,
- tekstową mapę świata pod **F5**.

### APRS

CQOps obsługuje trzy typy usług APRS. Wybierz typ odpowiadający konfiguracji stacji:

| Service | Connection | Internet required |
|---|---|---|
| **APRS-IS** | TCP do serwera APRS-IS | Tak |
| **KISS** | Port szeregowy do sprzętowego TNC KISS | Nie |
| **KISS Server** | TCP do serwera TNC KISS, np. Dire Wolf | Nie, wystarczy sieć lokalna |

Wybierz typ usługi w menu **Integrations**:

```text
F9 → Integrations → APRS → Service (Space to cycle)
```

Wszystkie trzy usługi obsługują odbieranie raportów pozycji APRS pobliskich stacji i wyświetlanie ich na lokalnej mapie **CQOps Live** z:

- standardowymi symbolami APRS,
- oknami callsign,
- automatycznym dopasowaniem widoku,
- konfigurowalnym okręgiem zasięgu.

Wszystkie usługi obsługują również **periodic position beaconing**. CQOps nadaje lokator stacji w skonfigurowanym interwale. Gdy GPS jest aktywny i włączono **Grid from GPS**, beacon automatycznie używa pozycji z GPS — idealnie do pracy portable i mobile.

#### APRS-IS

Łączy się z globalną siecią APRS-IS przez internet. Wymaga:

- poprawnego krótkofalarskiego znaku wywoławczego,
- APRS-IS passcode wygenerowanego ze znaku,
- połączenia z internetem.

Domyślny serwer:

```text
euro.aprs2.net:14580
```

APRS-IS konfiguruje się globalnie pod **F9 → Integrations → APRS**. Callsign, SSID, symbol, comment, beacon interval i range filter dla konkretnego logu ustawia się pod **F9 → Logbooks → [active logbook] → APRS**.

#### KISS (serial)

Łączy się bezpośrednio ze sprzętowym TNC KISS przez port szeregowy. Internet nie jest wymagany — ramki APRS są wysyłane i odbierane przez radiostację.

W menu **Integrations** skonfiguruj serial port, baud rate, data bits, parity, stop bits oraz DTR/RTS:

```text
F9 → Integrations → APRS → Service: KISS
```

Po wybraniu **KISS** widoczne stają się pola dotyczące portu szeregowego: **Port**, **Baud**, **Data bits**, **Parity**, **Stop bits**, **DTR**, **RTS**.

Przycisk **Test** otwiera port szeregowy, aby sprawdzić dostępność TNC.

#### KISS Server (TCP)

Łączy się z TNC KISS dostępnym przez TCP, na przykład z instancją [Dire Wolf](https://github.com/wb2osz/direwolf) uruchomioną na tym samym komputerze albo w sieci lokalnej. Internet nie jest wymagany.

Wprowadź host i port w menu **Integrations**:

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Wartości domyślne: `127.0.0.1:8001`

#### Beaconing

Beacony są wysyłane z interwałem skonfigurowanym dla danego logu. Minimalny interwał wynosi 1 minutę. Beacon zawiera:

- callsign stacji z SSID,
- lokator, obliczony z GPS, jeśli jest dostępny,
- symbol APRS,
- opcjonalny comment.

Gdy **GPS** jest aktywny i w ustawieniach **Station** włączono **Grid from GPS**, beacon automatycznie używa lokatora obliczonego z GPS. Podczas przemieszczania nie trzeba ręcznie aktualizować lokatora.

Beacon interval i inne ustawienia konkretnego logu konfiguruje się pod:

```text
F9 → Logbooks → [active logbook] → APRS
```

#### Receiving

Odebrane raporty pozycji APRS są zapisywane lokalnie w cache i wyświetlane na mapie **CQOps Live dashboard**. Stacje są przedstawiane za pomocą symboli APRS i można je kliknąć, aby zobaczyć szczegóły. Widok automatycznie dopasowuje się tak, aby pokazać wszystkie widoczne stacje w skonfigurowanym zasięgu.

Odbiór APRS jest niezależny od nadawania beaconów — można odbierać bez nadawania i odwrotnie. Wystarczy włączyć APRS w menu **Integrations** i ustawić typ usługi.

### Solar Data

Dane solarne pochodzą z hamqsl.com i obejmują:

- SFI,
- sunspot number,
- A/K indices,
- warunki dla poszczególnych pasm.

Aktualizacje na żywo wymagają internetu. Po poprawnym pobraniu dane z cache pozostają dostępne offline.

---

<a id="cqops-live-dashboard"></a>
## CQOps Live Dashboard

CQOps Live to wbudowany dashboard przeglądarkowy pokazujący aktywność stacji w czasie rzeczywistym.

Przydaje się do:

- publicznych ekranów podczas field day,
- monitorów na stacji klubowej,
- monitorowania zawodów,
- obserwowania stacji z innego pomieszczenia,
- stoisk na imprezach i targach.

### Włączanie dashboardu

1. Naciśnij **F9**.
2. Otwórz **Integrations**.
3. Przejdź do **HTTP Server**.
4. Włącz **HTTP server**.
5. Opcjonalnie ustaw address i port.
6. Naciśnij **Ctrl+S**, aby zapisać.
7. Otwórz dashboard w przeglądarce.

Ustawienia domyślne:

| Setting | Default |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

Serwer uruchamia się natychmiast po zapisaniu ustawień.

> **Address binding:** Domyślne `0.0.0.0` udostępnia dashboard wszystkim urządzeniom w sieci lokalnej. Jest to przydatne przy ekranach field day, na stacjach klubowych i podczas sprawdzania stacji z innego pomieszczenia. Ustaw address na `127.0.0.1`, aby ograniczyć dostęp wyłącznie do lokalnego komputera.

### Tryby wyświetlania

CQOps Live ma dwa tryby wyświetlania.

#### Overview mode

Jest wyświetlany, gdy nie trwa praca z aktywnym callsign.

Pokazuje:

- **live maps** — markery dzisiejszych QSO z trasami po ortodromie od lokatora własnej stacji do każdego korespondenta oraz lokalną mapę APRS pokazującą pobliskie stacje,
- tabelę recent QSOs,
- informacje o stacji,
- statystyki,
- tempo z ostatnich 5 minut, 15 minut i 1 godziny,
- najlepszych operatorów,
- QSO na największą odległość.

#### Active / Now Working mode

Jest wyświetlany podczas pracy z konkretnym callsign.

Pokazuje:

- duży callsign,
- wskaźnik submode,
- zdjęcie QRZ, jeżeli jest dostępne,
- oznaczenia band i mode,
- wskaźniki **DUPE / NEW CALL / NEW DXCC**,
- odległość i azymut,
- wyróżnioną przerywaną trasę na mapie od lokatora stacji do lokatora korespondenta.

### Info box

**Info box** nad lokalną mapą co 5 sekund przełącza następujące moduły:

- warunki na pasmach,
- aktywność słoneczną,
- pole geomagnetyczne,
- najnowszy spot DX Cluster,
- liczbę raportów PSK Reporter dla poszczególnych pasm.

### Weather row

**Weather row** pokazuje bieżące warunki Open-Meteo dla lokatora stacji:

- temperaturę,
- wiatr,
- wilgotność,
- ikonę.

Dane pogodowe są pobierane po stronie przeglądarki i pozostają bezpiecznie pomijane w trybie offline.

### Local map

Prawa **local map** jest przeznaczona do **monitorowania otoczenia APRS**, aby pokazywać stacje APRS w pobliżu. Może wyświetlać:

- pobliskie stacje APRS ze standardowymi symbolami,
- pop-upy callsign po najechaniu lub kliknięciu,
- konfigurowalny okrąg zasięgu,
- opcjonalną warstwę terminatora dnia i nocy,
- opcjonalną warstwę radaru pogodowego RainViewer.

### Aktualizacje w czasie rzeczywistym i wydajność

CQOps Live aktualizuje dane przez Server-Sent Events (SSE). Odświeżanie strony nie jest potrzebne.

Dashboard zaprojektowano z myślą o sprzęcie o małej mocy:

- przeglądarka renderuje mapy,
- przeglądarka oblicza odległości,
- przeglądarka oblicza statystyki,
- CQOps przesyła lekkie aktualizacje JSON,
- gdy **HTTP server** jest wyłączony, żaden port nie jest otwierany i goroutines dashboardu nie działają.

### Dostosowywanie dashboardu

W formularzu integracji **HTTP Server** można skonfigurować:

| Field | Description |
|---|---|
| Header 1 | Główny tytuł widoczny w nagłówku strony i sekcji hero. Domyślnie „CQOps Live”. |
| Header 2 | Podtytuł pod tytułem. Domyślnie „Fast, portable ham radio logger”. |
| Logo URL | Publicznie dostępny URL obrazu w lewym górnym rogu. Domyślnie logo CQOps. |
| Event Start | Data w formacie `YYYY-MM-DD`. Filtruje statystyki i listy QSO od tej daty. |

---

<a id="configuration"></a>
## Konfiguracja

Otwórz konfigurację klawiszem **F9**.

### Pliki konfiguracyjne

| Platforma | Ścieżka konfiguracji |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Poufne dane uwierzytelniające są przechowywane osobno w pliku `secrets.enc` w tym samym katalogu konfiguracji.

Sekrety są szyfrowane kluczem powiązanym z komputerem. Po przeniesieniu konfiguracji na inny komputer dane uwierzytelniające trzeba wprowadzić ponownie.

### Menu konfiguracji

Naciśnij **F9**, aby otworzyć menu główne, a następnie wybierz:

| Menu | Configures |
|---|---|
| General | Units, timezone, partner map/picture, solar panel, źródła SCP/REF, Kitty Graphics, Debug mode |
| Logbooks | Station callsign, grid, references, CQ/ITU zones, IARU region, GPS grid; ustawienia Wavelog dla logu (URL, API key, station profile); ustawienia APRS dla logu (callsign, symbol, beacon, range) |
| Operators | Profile operator callsign i operator name dla stacji wielooperatorskich |
| Rigs | Presety radiostacji: model, antenna, power, backend (None/flrig/Hamlib), rotor, WSJT-X UDP |
| Contests | Profile zawodów: name, date, ADIF contest ID, exchange templates, starting serial number |
| Integration | DX Cluster (host, port, login), HTTP Server dashboardu (address, port, branding), GPS service (serial/GPSD, grid precision) |
| Callbook | Dostawcy QRZ.com, HamQTH, QRZ.RU, Callook.info; kolejność priorytetów, base-call fallback, Wavelog lookup |
| Notifications | QSO saved alerts, Wavelog QSO sent status, dupe beep, error sounds |

### Multi-logbook

Użyj wielu logów do pracy home, portable, contest i club.

Naciśnij **Ctrl+L**, aby przełączać aktywny log.

Każdy log przechowuje własne:

- dane stacji,
- ustawienia Wavelog,
- ustawienia zawodów,
- ustawienia operatorów.

### Multi-operator

Profile operatorów zawierają:

- operator callsign,
- operator name.

Naciśnij **Ctrl+O**, aby przełączać aktywnego operatora.

Aktywny operator jest zapisywany w polu ADIF `OPERATOR` i przekazywany podczas wysyłania do Wavelog.

### Multi-rig

Presety radiostacji przechowują:

- backend,
- model,
- antenna,
- power,
- ustawienia rotora,
- ustawienia WSJT-X.

Naciśnij **Ctrl+R**, aby przełączać aktywną radiostację.

### Szyfrowane sekrety

Od wersji v0.8.7 dane uwierzytelniające są przechowywane w postaci zaszyfrowanej.

| Element | Wartość |
|---|---|
| Plik sekretów | `secrets.enc` |
| Lokalizacja | Ten sam katalog co `config.yaml` |
| Uprawnienia Unix | `0600`, jeżeli obsługiwane |
| Szyfrowanie | AES-256-GCM z kluczem powiązanym z komputerem |
| Chronione dane | QRZ password, DX Cluster login, Wavelog API keys |

Sekrety zapisane jawnym tekstem w starszych konfiguracjach są migrowane przy pierwszym uruchomieniu.

Jeżeli `secrets.enc` jest uszkodzony, CQOps uruchamia się z ostrzeżeniem i prosi o ponowne wprowadzenie danych uwierzytelniających.

---

<a id="keyboard-shortcuts"></a>
## Skróty klawiaturowe

### Global

| Key | Action |
|---|---|
| F1 | **QSO form** i **Recent QSOs** |
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
## Rozwiązywanie problemów

### CQOps nie uruchamia się

Sprawdź:

- czy terminal ma co najmniej 80×24 znaki,
- czy w Windows używasz Windows Terminal,
- czy start nie jest blokowany przez sieć — spróbuj:

  ```bash
  cqops --offline
  ```

Sprawdź logi:

| Platforma | Ścieżka logów |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### Radiostacja nie łączy się

Dla flrig:

- sprawdź, czy flrig działa,
- sprawdź port w aktywnym presecie radiostacji,
- domyślny port to `12345`.

Dla Hamlib:

- sprawdź, czy `rigctld` działa,
- sprawdź host i port,
- sprawdź, czy radiostacja/backend obsługuje wymagane dane.

Etykiety stanu pomagają zdiagnozować problem:

| Kolor | Znaczenie |
|---|---|
| Biały/domyślny | Połączone |
| Żółty | Wyłączone lub łączenie |
| Czerwony | Błąd |

Powiadomienia toast o ponownym połączeniu mogą być wyciszone. CQOps może ponawiać próbę bez wyświetlania komunikatów.

### WSJT-X nie loguje automatycznie

Sprawdź:

- **WSJT-X Settings → Reporting → UDP Server**,
- czy UDP host i port odpowiadają ustawieniom aktywnego presetu radiostacji w CQOps,
- czy używany jest WSJT-X 2.6 lub nowszy,
- czy etykieta **WSJT** jest aktywna,
- czy aktywny log jest poprawny,
- czy aktywny operator jest poprawny.

### Wysyłanie do Wavelog nie działa

Sprawdź:

- Wavelog URL,
- API key,
- station profile ID,
- etykietę stanu **WL**.

Błędy wysyłania są pokazywane jako toasty. QSO pozostają zapisane lokalnie także wtedy, gdy wysłanie się nie powiedzie. Błąd pojedynczego QSO nie blokuje pozostałej części partii.

### Problemy z plikiem konfiguracji

Plik konfiguracji:

| Platforma | Ścieżka |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Plik sekretów:

```text
secrets.enc
```

Plik sekretów jest przechowywany w tym samym katalogu co `config.yaml`.

Jeżeli konfiguracja jest uszkodzona, przenieś lub usuń plik i uruchom CQOps ponownie. Kreator konfiguracji utworzy nową konfigurację.

Pole `last_fetched_id` pojawia się dopiero po poprawnym pobraniu danych z Wavelog.

### Problemy z wydajnością

Spróbuj:

- wyłączyć renderowanie map w ustawieniach **General**,
- wyłączyć panel **Solar**, jeżeli nie jest potrzebny,
- unikać ekranów intensywnie korzystających z sieci, takich jak **DX Cluster** i **PSK Reporter**, gdy pracujesz offline,
- używać `cqops --offline`, gdy połączenie sieciowe jest niestabilne.

---

<a id="reporting-bugs"></a>
## Zgłaszanie błędów

Przed zgłoszeniem błędu:

1. Włącz **Debug mode** w **F9 → General → Debug** albo ustaw:

   ```yaml
   debug: true
   ```

   w pliku `config.yaml`.

2. Odtwórz problem.
3. Dołącz odpowiedni log.

Błędy zgłaszaj w GitHub:

<https://github.com/szporwolik/cqops/issues>

Dołącz:

- wersję CQOps z `cqops --version`,
- system operacyjny,
- emulator terminala,
- kroki pozwalające odtworzyć problem,
- odpowiedni debug log.
