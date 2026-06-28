---
title: Podręcznik użytkownika CQOps
description: Kompletny przewodnik po instalacji, konfiguracji i obsłudze CQOps — szybkiego, terminalowego loggera dla krótkofalowców
---

> **Uwaga:** To tłumaczenie zostało wygenerowane przy użyciu modelu LLM. Poprawki są mile widziane — prosimy o zgłaszanie ich jako Pull Request do gałęzi `dev`.

# Podręcznik użytkownika CQOps

## Spis treści

1. [O CQOps](#o-cqops)
2. [Pobieranie i instalacja](#pobieranie-i-instalacja)
3. [Pierwsze uruchomienie — kreator konfiguracji](#pierwsze-uruchomienie--kreator-konfiguracji)
4. [Szybki start: zaloguj pierwszą łączność](#szybki-start-zaloguj-pierwszą-łączność)
5. [Przegląd głównego ekranu](#przegląd-głównego-ekranu)
6. [Typowe scenariusze pracy](#typowe-scenariusze-pracy)
7. [Główne funkcje](#główne-funkcje)
8. [Integracje](#integracje)
9. [Konfiguracja](#konfiguracja)
10. [Skróty klawiszowe](#skróty-klawiszowe)
11. [Rozwiązywanie problemów](#rozwiązywanie-problemów)

---

## O CQOps

CQOps to szybki, terminalowy logger dla krótkofalowców, którzy potrzebują szybkości, niezawodności i niskiego obciążenia systemu — w shacku, na szczycie, podczas field day albo przy współdzielonej stacji klubowej.

**Offline-first.** Lokalne logowanie QSO nie wymaga internetu. Dane referencyjne, dane solarne i prefiksy DXCC zapisane w pamięci podręcznej pozostają dostępne po jednorazowym pobraniu. Integracje sieciowe, takie jak Wavelog, QRZ.com, DX Cluster i PSK Reporter, wymagają łączności i są pomijane w trybie `--offline`.

**Zbudowany do pracy terenowej.** CQOps jest gotowy na QRP, przyjazny dla SOTA/POTA i działa komfortowo na komputerach klasy Raspberry Pi, starszych laptopach oraz systemach bez środowiska graficznego.

**Gotowy do pracy na stacji klubowej.** CQOps obsługuje wiele logbooków, profile operatorów i presety rigów. Aktywny logbook, operator albo rig można przełączyć jednym skrótem klawiszowym.

**Przenośny z założenia.** CQOps to pojedynczy plik binarny napisany w Go. Nie wymaga CGO ani żadnych obowiązkowych usług systemowych.

**Wieloplatformowy.** Windows, Linux i macOS są wspierane na amd64 i arm64.

### Dla kogo jest CQOps

- Dla operatorów terenowych, którzy potrzebują szybkiego logowania z klawiatury na sprzęcie o niskim poborze mocy.
- Dla aktywatorów SOTA i POTA, którzy logują offline i przesyłają log później.
- Dla stacji klubowych z wieloma operatorami dzielącymi tę samą stację.
- Dla zespołów field day używających współdzielonych maszyn lub sprzętu klasy Raspberry Pi.
- Dla operatorów, którzy wolą pracę w terminalu niż klasyczne GUI.

CQOps nie ma zastępować rozbudowanych loggerów desktopowych ani webowych platform logbookowych. Koncentruje się na szybkim logowaniu w terminalu, pracy terenowej, trybie offline i scenariuszach współdzielonych stacji.

---

## Pobieranie i instalacja

> [Przeglądaj wszystkie wydania →](https://github.com/szporwolik/cqops/releases)

### Windows

| Pakiet | Link | Uwagi |
|---------|------|-------|
| **Instalator** | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Zalecany dla większości użytkowników. Dodaje CQOps do Menu Start i PATH. |
| Przenośny ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Wypakuj i uruchom bez instalacji. |

### Linux — Debian / Ubuntu

| Architektura | Link | Dla |
|-------------|------|---------|
| **amd64** | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Większość PC Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-bitowe systemy ARM |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-bitowy Raspberry Pi OS |

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Archiwum Przenośne

| Architektura | Link | Dla |
|-------------|------|---------|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Większość PC Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-bitowe systemy ARM |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-bitowy Raspberry Pi OS |

### macOS

| Architektura | Link | Dla |
|-------------|------|---------|
| **Apple Silicon** | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Maki M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Maki Intel |

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Z kodu źródłowego

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build        # Tylko budowanie; plik binarny w build/
make install      # Budowanie i instalacja systemowa
```

Budowanie ze źródeł wymaga Go 1.26 lub nowszego.

### Wymagania

- Rozmiar terminala: minimum 80×24 znaków.
- Zalecany rozmiar terminala: 80×43 lub większy.
- Zalecany nowoczesny emulator terminala. Na Windows używaj Windows Terminal zamiast legacy console.

### Opcje wiersza poleceń

```bash
cqops              # Uruchom TUI
cqops --offline    # Uruchom bez aktywności sieciowej
cqops --version    # Wyświetl wersję i zakończ
cqops --help       # Pokaż pomoc
```

---

## Pierwsze uruchomienie — kreator konfiguracji

Przy pierwszym uruchomieniu CQOps otwiera kreator konfiguracji dla podstawowych ustawień stacji. Integracje sieciowe można pominąć; lokalne logowanie działa bez nich.

1. **Stacja i Logbook**   
   Skonfiguruj początkowy logbook, znak wywoławczy stacji, operatora i grid locator. Opcjonalne pola obejmują referencje SOTA/POTA/WWFF, region IARU, strefę CQ/ITU, DXCC i SIG/SIG Info. Tutaj dostępna jest również konfiguracja Wavelog: URL, klucz API, ID profilu stacji, Update i Test.

2. **Rig**   
   Skonfiguruj preset rigu: nazwa, model, antena, moc i backend radiowy. Obsługiwane backendy to None, flrig i Hamlib rigctld. Opcjonalne ustawienia obejmują sterowanie rotorem przez Hamlib i integrację UDP WSJT-X.

3. **Integracje**   
   Skonfiguruj callbook QRZ.com: opcja włączenia, nazwa użytkownika, maskowane hasło i Test.

4. **Ogólne**   
   Wybierz strefę czasową IANA. CQOps domyślnie wykrywa systemową strefę czasową i udostępnia przewijaną listę.

5. **Podsumowanie**   
   Przejrzyj konfigurację. Naciśnij **Ctrl+S**, aby zapisać i uruchomić CQOps.

**Nawigacja w kreatorze:** **Ctrl+S** przechodzi dalej po walidacji. **Esc** cofa. **F10** zamyka. Spacja przełącza pola wyboru. Tab i Shift+Tab przechodzą między polami.

Wszystkie ustawienia kreatora można później zmienić z menu konfiguracji pod **F9**.

---

## Szybki start: zaloguj pierwszą łączność

1. **Zainstaluj i uruchom CQOps.**   
   Pobierz pakiet dla swojej platformy, uruchom `cqops` i ukończ kreator konfiguracji z co najmniej znakiem wywoławczym i lokatorem.

2. **Użyj QSO Form.**   
   QSO Form otwiera się pod **F1**. Wpisz znak wywoławczy; CQOps automatycznie zamienia go na wielkie litery. Jeśli aktywna rig jest podłączona przez flrig lub Hamlib, częstotliwość, pasmo, mode i submode są wypełniane automatycznie. Data i czas są ustawiane na aktualny czas UTC.

3. **Poruszaj się po polach.**   
   Używaj **Tab**, **Shift+Tab** i **↑/↓**.

4. **Zapisz QSO.**   
   Naciśnij **Enter** lub **Ctrl+S**. Jeśli pojawi się ostrzeżenie **DUPE!**, naciśnij **Enter** ponownie aby zapisać mimo to, lub **Esc** aby anulować.

Nowe QSO pojawia się natychmiast w tabeli Recent QSOs pod formularzem.

---

## Przegląd głównego ekranu

```text
┌─ Status Bar ─────────────────────────────────────────────────────────────────┐
│  CQOps v0.8.8  Log Portable  Rig FTDx10  Call SP9MOA/P                          │
│  Net WSJT Hamlib DXC WL                                            23:00L 2100Z │
├─ Tab Bar ─────────────────────────────────────────────────────────────────┤
│ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮         │
│ │F1 QSO│ │F2 QRZ│ │F4 DXC│ │F5 HRD│ │F6 REF│ │F7 BPL│ │F8 LOG│ │F9 CFG│         │
├─ Main Content Area ──────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  QSO Form, tabela, mapa, edytor lub zawartość aktywnego ekranu              │
│                                                                                  │
├─ Help Bar ───────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Status Bar

Status Bar pokazuje wersję CQOps, aktywny logbook, aktywną radiostację, znak wywoławczy stacji i aktywnego operatora. Po prawej stronie wyświetlane są etykiety statusu integracji oraz czas lokalny (`L`) i UTC (`Z`).

**Kolory etykiet:**

| Kolor | Znaczenie |
|-------|-----------|
| Biały/domyślny | Połączony lub aktywny |
| Żółty | Wyłączony, łączenie lub oczekiwany offline |
| Czerwony | Błąd lub rozłączony |
| Akcent + pogrubienie | WSJT-X nadaje |

Etykiety, które mogą się pojawić to **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC** i **WL**.

### Tab Bar

| Klawisz | Tab | Ekran |
|-----|-----|--------|
| F1 | QSO | QSO Form i tabela Recent QSOs |
| F2 | QRZ | Partner View: dane callbooka, mapa, statystyki, zdjęcie |
| F4 | DXC | Spoty DX Cluster i filtry |
| F5 | HRD | Spoty PSK Reporter i mapa propagacji |
| F6 | REF | Wyszukiwanie referencji SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, synchronizacja Wavelog |
| F9 | CFG | Menu konfiguracji |

### Help Bar

Dolny wiersz pokazuje najważniejsze skróty dla aktywnego ekranu. Naciśnij **?** aby zobaczyć pełną nakładkę pomocy.

---

## Typowe scenariusze pracy

### Praca terenowa / SOTA / POTA

1. **Przed wyjściem z domu** uruchom CQOps raz z dostępem do internetu. Pozwoli to CQOps wypełnić pamięć podręczną — dane solarne, REF i prefiksy DXCC.
2. **Zweryfikuj pamięć podręczną** przed przejściem w tryb offline. Sprawdź, czy panel Solar pokazuje dane i czy wyszukiwanie REF na **F6** zwraca wyniki.
3. **W terenie** uruchom CQOps z `cqops --offline`. Aktywność sieciowa zostaje pominięta, co eliminuje opóźnienia od nieosiągalnych usług.
4. **Loguj normalnie.** Lokalne logowanie działa bez internetu.
5. **Prześlij później.** Po powrocie do sieci otwórz Logbook Editor pod **F8** i naciśnij **w** aby przesłać niewysłane QSO do Wavelog.

### Współdzielona stacja klubowa i hot-seat

1. **Dodaj profile operatorów:** otwórz **F9 → Operators**, następnie naciśnij **Ins** dla każdego operatora. Wpisz ich znaki wywoławcze i imię.
2. **Przełącz aktywnego operatora:** na QSO Form naciśnij **Ctrl+O**. Aktywny operator jest pokazany na Status Bar i zapisywany w polu `OPERATOR` dla zapisanych QSO.
3. **Używaj hot-seat:** operator A loguje QSO, operator B naciska **Ctrl+O**, następnie loguje ze swoim profilem operatora.
4. **Użyj Retain, gdy jest potrzebne:** włącz **Retain** jeśli wielu operatorów musi zalogować tę samą łączność bez ponownego wpisywania całego formularza.

Przed zapisaniem na współdzielonej stacji sprawdź aktywny logbook i aktywnego operatora na Status Bar.

### Prywatny + klubowy logbook

Wielu operatorów prowadzi osobisty logbook i jeden lub więcej logbooków klubowych.

1. **Utwórz logbooki:** otwórz **F9 → Logbooks**, następnie naciśnij **Ins** dla każdego logbooka.
2. **Przełącz aktywny logbook:** naciśnij **Ctrl+L** na QSO Form. Status Bar pokazuje aktywny logbook.
3. **Dane stacji są oddzielne:** każdy logbook może mieć własny znak wywoławczy stacji, ustawienia Wavelog, ustawienia contestów i operatorów.
4. **Szybkie podwójne logowanie:** włącz **Retain**, zapisz QSO w jednym logbooku, naciśnij **Ctrl+L**, następnie zapisz je ponownie w drugim logbooku jeśli to właściwe.

### Wiele rigu

1. **Utwórz presety rigu:** otwórz **F9 → Rigs**, następnie naciśnij **Ins** dla każdej rigu.
2. **Ustaw backend:** użyj flrig lub Hamlib dla rigu z CAT. Użyj None dla rigu strojonych ręcznie.
3. **Przełącz aktywną radiostację:** naciśnij **Ctrl+R** na QSO Form.
4. **Obsługuj mieszane stacje:** na przykład używaj rigu KF z CAT i ręcznej rigu VHF/UHF w tej samej sesji.
5. **Skonfiguruj WSJT-X na każdą radiostację:** każdy preset rigu może mieć własne ustawienia UDP WSJT-X.

Gdy aktywna rig ma CAT, CQOps może automatycznie wypełnić częstotliwość, pasmo, emisję i subemisję. Dla rigu ręcznych wprowadź je samodzielnie.

### FT8 / automatyczne logowanie WSJT-X

Gdy WSJT-X jest podłączony przez UDP, CQOps może automatycznie logować cyfrowe QSO z komunikatów ADIF WSJT-X.

- Automatycznie logowane QSO są zapisywane w aktywnym logbooku.
- Zduplikowane automatyczne QSO są pomijane.
- Automatycznie logowane QSO dziedziczą aktywny contest ID.
- QSO pojawiają się natychmiast w Recent QSOs.
- Jeśli Wavelog jest skonfigurowany i osiągalny, automatycznie logowane QSO mogą być automatycznie przesyłane.
- Jeśli operator WSJT-X nie pasuje do aktywnego operatora, CQOps pokazuje ostrzeżenie.

Przed długimi sesjami cyfrowymi sprawdź aktywny logbook, aktywnego operatora i aktywny contest.

### Synchronizacja Wavelog

Synchronizacja Wavelog jest opcjonalna. CQOps zawsze zapisuje QSO lokalnie w pierwszej kolejności.

**Upload:** naciśnij **w** w Logbook Editor (**F8**). CQOps przesyła niewysłane QSO w paczkach po 50 i śledzi status każdego QSO: niewysłane, wysłane lub błąd.

**Download:** naciśnij **Ctrl+W** w Edytorze Logbooka. Download jest przyrostowe. CQOps pobiera QSO nowsze niż zapisany `last_fetched_id` dla aktywnego logbooka. Duplikaty są pomijane.

Jeśli upload do Wavelog się nie powiedzie, QSO pozostaje w lokalnym logbooku i można spróbować ponownie później. Purge logbooka resetuje ID pobierania do `0`, umożliwiając pełne ponowne pobranie.

---

## Główne funkcje

### Logowanie QSO

QSO Form (**F1**) to główny ekran logowania. Używa układu trzykolumnowego i może automatycznie wypełniać pola z kontroli rigu, QRZ.com, lookupu Wavelog, danych DXCC/prefiksów i baz REF.

**Pola formularza:**

| Lewa Kolumna | Środkowa Kolumna | Prawa Kolumna |
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

⚠️ Pola wymiany pojawiają się tylko gdy aktywne są zawody. Pola oznaczone **(▼)** przełącza się **PgUp/PgDn**.

Dolny wiersz zawiera:

- **Comment** (Komentarz)
- **Keep** — zachowuje pole Comment między QSO; przełączaj **Ctrl+T**
- **Retain** — zachowuje cały formularz po zapisaniu

Linia ścieżki/namiaru pokazuje odległość i azymut gdy oba grid locatory są znane. Może też pokazywać oznaczenia takie jak **DUPE!**, **New Call!** i **New DXCC!**.

### Źródła autowypełniania

| Źródło | Pola |
|--------|--------|
| flrig / Hamlib | Frequency, Freq RX jeśli split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| Baza REF | Referencje SOTA, POTA, WWFF, IOTA |
| lookupu Wavelog | Status worked/confirmed gdy skonfigurowany |

### Logowanie contestowe

Zawodyy dodają pola wymiany i obsługę numeracji do QSO Form.

Utwórz lub skonfiguruj contest w Logbook Editor (**F8**) klawiszem **Ins**. Ustaw nazwę contestu, datę, ADIF contest ID i szablony wymiany.

Wspierane znaczniki szablonów:

| Znacznik | Zastępowany Przez |
|--------|---------------|
| `@rst` | RST nadane lub odebrane |
| `@serial` | Automatycznie zwiększany numer |
| `@call` | Twój znak wywoławczy |
| `@grid` | Twój lokator |
| `@name` | Imię operatora z profilu operatora |

Naciśnij **Ctrl+C**, aby przełączać aktywny contest. Gdy contest jest aktywny:

- QSO Form pokazuje pola wymiany,
- numery automatycznie się zwiększają,
- Recent QSOs mogą filtrować QSO z contestu,
- eksport ADIF zachowuje `CONTEST_ID`.

### Logbook Editor

Logbook Editor (**F8**) służy do zarządzania QSO, importu/eksportu ADIF, synchronizacji Wavelog i operacji contestowych.

**Edycja inline:** wybierz wiersz klawiszami **↑/↓**, naciśnij **Enter** lub **e**, edytuj QSO, następnie zapisz **Ctrl+S**. Zmiany są natychmiast odzwierciedlane w Recent QSOs.

### Import i eksport ADIF

CQOps wspiera import i eksport ADIF 3.1.7.

- **Ctrl+I** importuje plik ADIF, waliduje rekordy, pomija duplikaty i pokazuje podsumowanie.
- **Ctrl+E** eksportuje QSO. Eksport może obejmować wszystkie QSO albo QSO filtrowane po conteście.
- Zaimportowane QSO są oznaczane do uploadu Wavelog, jeśli synchronizacja Wavelog jest skonfigurowana.

### Ulubione

Ulubione przechowują presety częstotliwości, emisji i pasma w 10 slotach.

| Skrót | Akcja |
|----------|--------|
| Alt+0–9 | Przywołaj slot ulubionych |
| Alt+Shift+0–9 | Zapisz bieżącą częstotliwość/emisję/pasmo do slotu |

Ulubione są przechowywane w konfiguracji i współdzielone między logbookami.

Przykład: dla polskiego setupu wywoławczego SOTA FM, wprowadź `145.55`, ustaw emisję `FM`, ustaw pasmo `2m`, następnie naciśnij **Alt+Shift+1**. Później naciśnij **Alt+1** aby przywołać.

### Wyszukiwanie REF

Ekran REF (**F6**) wyszukuje referencje SOTA, POTA, WWFF i IOTA. Szukaj po prefiksie, nazwie lub oznaczeniu referencji. Wybrane referencje mogą wypełnić QSO Form.

### Band Plan Browser

Przeglądarka Planów Pasm (**F7**) zapewnia szybki dostęp do pasm amatorskich, zakresów VHF/UHF, CB, PMR446 i presetów broadcast. Wybrana częstotliwość może być użyta do dostrojenia aktywnej rigu. Dane planów pasm można również wyeksportować jako Markdown.

---

## Integracje

### QRZ.com

Wyszukiwanie QRZ.com wymaga dostępu do internetu i subskrypcji QRZ XML.

Naciśnij **Ins** na QSO Form aby wypełnić pola callbook takie jak imię, QTH, grid, kraj, strefy CQ/ITU, DXCC i kontynent. Partner View (**F2**) może pokazać zdjęcie operatora gdy dostępne.

### Wavelog

Integracja Wavelog wymaga dostępu do internetu. Obsługuje upload, przyrostowe pobieranie oraz lookup worked/confirmed.

Wavelog jest konfigurowany na aktywny logbook z URL, kluczem API i ID profilu stacji. CQOps zawsze zapisuje QSO lokalnie w pierwszej kolejności; błąd uploadu Wavelog nie powoduje utraty danych.

Zobacz [Synchronizacja Wavelog](#synchronizacja-wavelog).

### flrig

Integracja flrig używa XML-RPC przez HTTP. Domyślny endpoint to `localhost:12345`.

CQOps może odczytywać częstotliwość, emisję i moc z flrig. Praca split jest mapowana jako VFO A na Frequency i VFO B na Freq RX.

### Hamlib / rigctld

Kontrola rigu Hamlib używa demona TCP `rigctld`. CQOps może odpytywać o częstotliwość, emisję, VFO, split i moc w zależności od wsparcia rigu.

Niektóre radiostacje lub backendy Hamlib nie wspierają wszystkich zapytań. CQOps obsługuje brak wsparcia nazwy VFO w sposób bezpieczny gdzie to możliwe.

### Hamlib Rotor / rotctld

Kontrola rotora używa Hamlib `rotctld`. CQOps wspiera azymut, elewację i komendy stop.

Przydatne skróty:

| Skrót | Akcja |
|----------|--------|
| Ctrl+←/→ | Dostosuj azymut o 5° |
| Ctrl+↑/↓ | Dostosuj elewację o 5° |
| Ctrl+A | Skieruj rotor na obliczony namiar ścieżki |
| Ctrl+F1 | Zatrzymaj rotor |

### WSJT-X

Integracja WSJT-X używa komunikatów UDP z WSJT-X. CQOps parsuje komunikaty ADIF i może automatycznie logować ukończone QSO.

Etykieta rigu staje się koloru akcentu podczas nadawania WSJT-X. Jeśli operator zgłaszany przez WSJT-X nie pasuje do aktywnego operatora, CQOps pokazuje ostrzeżenie.

Zobacz [FT8 / automatyczne logowanie WSJT-X](#ft8--automatyczne-logowanie-wsjt-x).

### DX Cluster

Integracja DX Cluster używa połączenia telnet i wymaga dostępu do internetu. Domyślny serwer to `dxspots.com:7300`.

Filtry obejmują pasmo, kontynent, emisję i czas. Naciśnij **Enter** na spocie aby wypełnić QSO Form, dostroić aktywną radiostację i wrócić do ekranu QSO. Naciśnij **Space** aby dostroić bez wypełniania formularza. Naciśnij **Backspace** aby wyczyścić filtry.

### PSK Reporter

Integracja PSK Reporter wymaga dostępu do internetu. Zapewnia spoty propagacyjne, filtry pasma/czasu/emisji i mapę świata ASCII na **F5**.

### Dane solarne

Dane solarne obejmują SFI, liczbę plam, indeksy A/K i warunki pasmowe z hamqsl.com. Aktualizacje na żywo wymagają dostępu do internetu. Dane z pamięci podręcznej pozostają dostępne offline po udanym pobraniu.

---

## Konfiguracja

Konfiguracja CQOps jest przechowywana w:

| Platforma | Ścieżka konfiguracji |
|----------|-------------|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Wrażliwe dane logowania są przechowywane osobno w `secrets.enc` w tym samym katalogu konfiguracyjnym. dane poufne są szyfrowane kluczem powiązanym z maszyną, więc dane logowania muszą być wprowadzone ponownie przy przenoszeniu konfiguracji na inną maszynę.

Otwórz konfigurację klawiszem **F9**.

| Menu | Konfiguruje |
|------|------------|
| Station | Znak, grid, strefa CQ/ITU, region IARU, referencje |
| Rig | Presety rigu, model, antena, moc, backend, rotor, WSJT-X |
| Wavelog | URL, klucz API, ID profilu stacji |
| QRZ | Nazwa użytkownika i hasło |
| DX Cluster | Host, port, login |
| Operators | Profile operatorów: znak i imię |
| Logbooks | Ustawienia stacji, Wavelog, contestu i operatora dla każdego logbooka |
| Notifications | Zachowanie toastów i powiadomień |
| General | Strefa czasowa, jednostki odległości, mapa, tryb debug |

### Multi-Logbook

Używaj wielu logbooków do pracy domowej, terenowej, contestowej i klubowej. Naciśnij **Ctrl+L** aby przełączać aktywny logbook. Każdy logbook przechowuje własne dane stacji, ustawienia Wavelog, ustawienia contestów i operatorów.

### Multi-Operator

Profile operatorów zawierają znak wywoławczy i imię operatora. Naciśnij **Ctrl+O** aby przełączać aktywnego operatora. Aktywny operator jest zapisywany w polu ADIF `OPERATOR` i jest używany przy uploadzie Wavelog.

### Multi-Rig

Presety rigu przechowują backend, model, antenę, moc, rotor i ustawienia WSJT-X. Naciśnij **Ctrl+R** aby przełączać aktywną radiostację.

### Szyfrowane dane poufne

Od wersji 0.8.7 dane logowania są przechowywane szyfrowane.

- **Plik danych poufnych:** `secrets.enc`
- **Lokalizacja:** ten sam katalog co `config.yaml`
- **Uprawnienia Unix:** `0600` gdzie wspierane
- **Szyfrowanie:** AES-256-GCM z kluczem powiązanym z maszyną
- **Chronione dane:** hasło QRZ, login DX Cluster, klucze API Wavelog
- **Migracja:** tekstowe dane poufne ze starszych konfiguracji migrują przy pierwszym uruchomieniu
- **Odzyskiwanie:** jeśli `secrets.enc` jest uszkodzony, CQOps uruchamia się z ostrzeżeniem i prosi o ponowne wprowadzenie danych

---

## Skróty klawiszowe

### Globalne

| Klawisz | Akcja |
|-----|--------|
| F1 | QSO Form i Ostatnie QSO |
| F2 | Partner View |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | Wyszukiwanie REF |
| F7 | Przeglądarka Planów Pasm |
| F8 | Logbook Editor |
| F9 | Konfiguracja / menu główne |
| F10 | Zamknij |
| Ctrl+F9 | Przeglądarka logów |
| ? | Nakładka pomocy |
| Ctrl+L | Przełącz aktywny logbook |
| Ctrl+R | Przełącz aktywną radiostację |
| Ctrl+C | Przełącz aktywny contest |
| Ctrl+O | Przełącz aktywnego operatora |
| Esc | Powrót do poprzedniego ekranu |

### QSO Form — F1

| Klawisz | Akcja |
|-----|--------|
| Tab | Następne pole |
| Shift+Tab | Poprzednie pole |
| ↑ / ↓ | Ruch w kolumnie |
| Enter | Zapisz QSO, z potwierdzeniem duplikatu jeśli potrzeba |
| Ctrl+S | Zapisz QSO z dowolnego pola |
| Del | Wyczyść wszystkie pola formularza |
| Ins | Wyszukiwanie: QRZ, Wavelog, DXCC i sprawdzenie duplikatu |
| PgUp / PgDn | Przełączanie pasma, emisji lub subemisji |
| Ctrl+D | Otwórz dialog spota |
| Ctrl+T | Przełącz Keep Comment |
| Ctrl+←/→ | Dostosuj azymut rotora o 5° |
| Ctrl+↑/↓ | Dostosuj elewację rotora o 5° |
| Ctrl+A | Skieruj rotor na namiar od własnego gridu do gridu partnera |
| Ctrl+F1 | Zatrzymaj rotor |
| Alt+0–9 | Przywołaj slot ulubionych |
| Alt+Shift+0–9 | Zapisz bieżącą częstotliwość/emisję/pasmo do slotu |

### Logbook Editor — F8

| Klawisz | Akcja |
|-----|--------|
| ↑ / ↓ | Nawigacja po wierszach |
| PgUp / PgDn | Poprzednia lub następna strona |
| Home / End | Pierwszy lub ostatni wiersz |
| Enter / e | Edytuj wybrane QSO |
| Delete | Usuń wybrane QSO |
| p | Wyczyść wszystkie QSO |
| Ctrl+C | Przełącz filtr contestu |
| Ctrl+E | Eksport ADIF |
| Ctrl+I / Tab | Import ADIF |
| w | Wyślij niewysłane QSO do Wavelog |
| Ctrl+W | Pobierz kontakty z Wavelog |
| Esc / F6 | Zamknij edytor, wróć do QSO |

### DX Cluster — F4

| Klawisz | Akcja |
|-----|--------|
| ↑ / ↓ | Nawigacja po spotach |
| Enter | Wypełnij formularz + dostrój radiostację + przejdź do QSO |
| Space | Dostrój radiostację do spota (pozostań na DXC) |
| Home | Filtr pasma do przodu |
| End | Filtr pasma do tyłu |
| \\ | Filtr kontynentu |
| Ins | Filtr emisji do przodu |
| Del | Filtr emisji do tyłu |
| PgUp | Filtr czasu do przodu |
| PgDn | Filtr czasu do tyłu |
| Backspace | Wyczyść wszystkie filtry |
| Esc / F4 | Powrót do QSO Form |

### Partner View — F2

| Klawisz | Akcja |
|-----|--------|
| F2 | Cykl: Partner View → Zdjęcie → Powrót |
| Esc / F1 | Powrót do QSO Form |

---

## Rozwiązywanie problemów

### Aplikacja się nie uruchamia

- Terminal musi mieć co najmniej 80×24 znaków.
- Na Windows używaj Windows Terminal, nie legacy `cmd.exe`.
- Spróbuj `cqops --offline` aby wykluczyć problemy sieciowe.
- Sprawdź logi: `~/.local/share/cqops/logs/` (Linux), `~/Library/Application Support/cqops/logs/` (macOS) lub `%APPDATA%\cqops\logs\` (Windows).

### Radio się nie łączy

- **flrig:** sprawdź czy flrig działa i port się zgadza (domyślnie `12345`).
- **Hamlib:** sprawdź czy rigctld działa i port TCP jest poprawny.
- Kolor etykiety statusu: biały = połączony, żółty = łączenie/wyłączony, czerwony = błąd.
- Wyciszone toasty reconnect są normalne — CQOps ponawia próby w tle.

### WSJT-X nie loguje automatycznie

- Sprawdź ustawienia UDP WSJT-X: Settings → Reporting → UDP Server.
- WSJT-X musi być w wersji 2.6 lub nowszej.
- Etykieta statusu powinna być biała (domyślna) gdy WSJT-X działa.

### Upload do Wavelog kończy się błędem

- Sprawdź URL, klucz API i ID profilu stacji w konfiguracji.
- Etykieta statusu: biała = osiągalny, żółta = wyłączony/brak internetu, czerwona = błąd.
- Błędy uploadu są pokazywane jako toasty; QSO pozostają zapisane lokalnie.
- Pojedyncze niepowodzenia QSO nie blokują reszty partii.

### Problemy z plikiem konfiguracyjnym

- Konfiguracja: `~/.config/cqops/config.yaml` (Linux/macOS) lub `%APPDATA%\cqops\config.yaml` (Windows).
- dane poufne: `secrets.enc` w tym samym katalogu.
- Jeśli konfiguracja jest uszkodzona, usuń ją i uruchom ponownie — kreator utworzy nową.
- Pole `last_fetched_id` pojawia się tylko po udanym pobraniu z Wavelog.

### Wydajność

- Wyłącz renderowanie mapy i panel solarny w ustawieniach General.
- Zamknij nieużywane zakładki (DXC, PSK).
- Uruchom z `--offline` jeśli sieć jest zawodna.

### Zgłaszanie błędów

Włącz **tryb debug** przed odtworzeniem problemu — F9 → General → Debug lub ustaw `debug: true` w konfiguracji. Pełne logi są zapisywane w katalogu logów właściwym dla platformy.

Zgłaszaj problemy na [GitHub Issues](https://github.com/szporwolik/cqops/issues) podając:
- Wersję CQOps (`cqops --version`)
- System operacyjny i emulator terminala
- Kroki do odtworzenia
- Log debug
