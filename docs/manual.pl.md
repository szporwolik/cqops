---
title: CQOps — Podręcznik Użytkownika
description: Kompletny przewodnik instalacji, konfiguracji i obsługi CQOps
---

# CQOps — Podręcznik Użytkownika

## Spis Treści

1. [O Programie](#o-programie)
2. [Instalacja](#instalacja)
3. [Kreator Pierwszego Uruchomienia](#kreator-pierwszego-uruchomienia)
4. [Układ Głównego Ekranu](#układ-głównego-ekranu)
5. [Logowanie QSO](#logowanie-qso)
6. [Logowanie Contestowe](#logowanie-contestowe)
7. [Edytor Logbooka](#edytor-logbooka)
8. [Konfiguracja](#konfiguracja)
9. [Integracje](#integracje)
10. [Skróty Klawiszowe](#skróty-klawiszowe)
11. [Rozwiązywanie Problemów](#rozwiązywanie-problemów)

---

## O Programie

CQOps to szybki, terminalowy dziennik łączności dla krótkofalowców. Działa offline i został zaprojektowany z myślą o pracy terenowej i przenośnej — uruchomi się na Raspberry Pi, starym laptopie, czy dowolnym komputerze bez GUI.

**Główne założenia:**
- Szybkie logowanie z klawiatury
- Działa na słabym sprzęcie (Raspberry Pi)
- Offline-first z opcjonalną synchronizacją w chmurze (Wavelog)
- Czyste Go, pojedynczy plik binarny, bez CGO / bez zależności systemowych
- Działa przez SSH, VNC, RDP lub bezpośrednio w terminalu

---

## Pobieranie

> [Wszystkie wydania →](https://github.com/szporwolik/cqops/releases)

### Windows

| Pakiet | Link |
|--------|------|
| **Instalator** (zalecany) | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) |
| Przenośny (ZIP) | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) |

Instalator dodaje CQOps do Menu Start i PATH.

### Linux — Debian / Ubuntu

| Architektura | Link |
|-------------|------|
| **amd64** (większość PC) | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) |
| armhf (Raspberry Pi) | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) |

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Archiwum Przenośne

| Architektura | Link |
|-------------|------|
| **amd64** | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) |
| armhf (Pi) | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) |

### macOS

| Architektura | Link |
|-------------|------|
| **Apple Silicon** (M1/M2/M3) | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) |
| Intel (x86_64) | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) |

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Z Kodu Źródłowego

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
```

### Wymagania

- Terminal o minimalnym rozmiarze 80×24 znaków (zalecane 80×43)
- Go 1.26+ (tylko przy budowaniu ze źródeł)
- WSJT-X 2.6+ (opcjonalnie, do automatycznego logowania emisji cyfrowych)

### Opcje Wiersza Poleceń

```bash
cqops              # Normalne uruchomienie (TUI)
cqops --offline    # Uruchomienie bez aktywności sieciowej
cqops --version    # Wyświetlenie wersji i zakończenie
cqops --help       # Wyświetlenie pomocy
```

---

## Kreator Pierwszego Uruchomienia

Przy pierwszym uruchomieniu CQOps otwiera kreator konfiguracji:

1. **Stacja** — znak wywoławczy, lokator Maidenhead, region IARU, strefa CQ/ITU, kontynent
2. **Radio** — model, antena, moc, backend (flrig lub Hamlib rigctld)
3. **QRZ.com** — nazwa użytkownika i hasło API (opcjonalnie)
4. **Wavelog** — URL, klucz API, ID profilu stacji (opcjonalnie)
5. **DX Cluster** — host, port, login (opcjonalnie)
6. **Strefa czasowa** — lokalna strefa IANA (np. `Europe/Warsaw`)

Każdą integrację można pominąć — wszystko działa offline. Ustawienia można zmienić później w ekranach konfiguracji.

**Nawigacja w kreatorze:** Tab / Shift+Tab między polami, Enter aby przejść dalej, Esc aby pominąć lub wrócić.

---

## Układ Głównego Ekranu

```
┌─ Pasek Statusu ─────────────────────────────────────────────────────┐
│ SP9SPM │ DEBUG │ DIGI (Yaesu FTDx10) │ OP: SP9SPM │ 14:32 UTC │
├────────────────────────────────────────────────────────────────────┤
│ Net ● WSJT ● Rig ● WL ● QRZ ● DXC ●                                 │
├─ Pasek Zakładek ────────────────────────────────────────────────────┤
│ [QSO] [Ostatnie] [Partner] [DX Cluster] [PSK] [Solar] [Log] [BPL]  │
├─ Główny Obszar ─────────────────────────────────────────────────────┤
│                                                                      │
│  (Formularz QSO, tabela, mapa itp. — zmienia się z aktywną zakładką)│
│                                                                      │
├─ Pasek Pomocy ──────────────────────────────────────────────────────┤
│ F1 QSO  F2 Ostatnie  F3 Partner  F4 DXC  F5 PSK  F6 Solar  F7 BPL  │
└──────────────────────────────────────────────────────────────────────┘
```

### Pasek Statusu

Pokazuje znak wywoławczy, nazwę aktywnego logbooka, nazwę radia (z modelem), aktywnego operatora i czas UTC oraz lokalny.

### Kropki Statusu Integracji

| Kropka   | Znaczenie                                  |
|----------|--------------------------------------------|
| ● Zielona  | Połączony / online                       |
| ● Czerwona | Rozłączony / błąd                         |
| ● Bursztynowa | Łączenie / stan przejściowy           |
| ○ Szara    | Wyłączony / nieskonfigurowany            |

- **Net** — łączność z internetem
- **WSJT** — status połączenia UDP z WSJT-X
- **Rig** — połączenie z flrig lub Hamlib rigctld
- **WL** — dostępność API Wavelog
- **QRZ** — status subskrypcji QRZ.com
- **DXC** — połączenie telnet z DX Clusterem

### Pasek Zakładek

Naciśnij **F1–F8** aby przełączać między ekranami:

| Klawisz | Ekran         | Przeznaczenie                                  |
|---------|---------------|------------------------------------------------|
| F1      | QSO           | Formularz wprowadzania QSO                     |
| F2      | Ostatnie      | Tabela ostatnich QSO z edycją inline           |
| F3      | Partner       | Dane z callbooka, mapa, statystyki             |
| F4      | DX Cluster    | Spoty DX na żywo z filtrami                    |
| F5      | PSK Reporter  | Tabela spotów PSK i mapa propagacji            |
| F6      | Solar         | Indeks słoneczny, indeksy A/K, warunki         |
| F7      | Band Plan     | Plany pasm (HAM, VHF/UHF, CB, broadcast)       |
| F8      | Edytor        | Zarządzanie QSO, import/export ADIF, Wavelog   |

### Pasek Pomocy

Dolny wiersz pokazuje najważniejsze skróty dla bieżącego ekranu. Naciśnij **?** aby zobaczyć wszystkie dostępne skróty.

---

## Logowanie QSO

Formularz QSO (F1) to główny interfejs logowania.

### Nawigacja w Formularzu

| Klawisz       | Akcja                                    |
|---------------|------------------------------------------|
| Tab           | Następne pole (uwzględnia kolumny)       |
| Shift+Tab     | Poprzednie pole                          |
| ↑ / ↓         | Ruch w górę/dół w obrębie kolumny        |
| Ctrl+S        | Zapisz QSO (z dowolnego pola)            |
| Enter         | Zapisz QSO (z ostatniego pola)           |
| Esc           | Wyczyść formularz / anuluj               |
| Ins           | Pobierz dane z QRZ (callbook lookup)     |
| Spacja        | Przełącz zachowanie komentarza / operatora|
| Ctrl+O        | Przełączanie między operatorami          |

### Pola Formularza

| Kolumna 1 (Znak)  | Kolumna 2 (Raport) | Kolumna 3 (Szczegóły) |
|--------------------|---------------------|------------------------|
| Znak               | RST Nadane          | Imię                   |
| Pasmo              | RST Odebrane        | QTH                    |
| Częstotliwość (MHz)| Emisja              | Lokator                |
| Freq RX (MHz)      | Subemisja           | Komentarz              |
|                    |                     | Uwagi                  |
|                    |                     | SOTA / POTA / WWFF     |
|                    |                     | IOTA                   |

### Logowanie Łączności

1. Wpisz znak wywoławczy (automatycznie zmieniany na wielkie litery).
2. Pasmo, częstotliwość i emisja są automatycznie pobierane z radia (flrig/Hamlib).
3. Data i czas są ustawiane automatycznie na bieżące UTC.
4. Naciskaj **Tab** aby przechodzić między polami lub **↑/↓** do nawigacji.
5. Naciśnij **Enter** aby zapisać. Formularz sprawdza poprawność danych i ostrzega o duplikatach.

### Wykrywanie Duplikatów

Jeśli kombinacja znaku, pasma i emisji już istnieje w logbooku dla dzisiejszego dnia, CQOps pokazuje ostrzeżenie **DUPE!**. Naciśnij **Enter** ponownie aby potwierdzić zapis, lub **Esc** aby anulować.

### Oznaczenia

Po zapisaniu QSO formularz może pokazać oznaczenia przy polu znaku:
- **New Call** — pierwsza łączność z tym znakiem w tym logbooku
- **New DXCC** — pierwsza łączność z tym krajem DXCC w tym logbooku

### Automatyczne Wypełnianie z Radia

Gdy radio jest podłączone (flrig lub Hamlib), te pola są automatycznie wypełniane:
- Częstotliwość
- Częstotliwość RX (jeśli wykryto split)
- Emisja / Subemisja

### Automatyczne Wypełnianie z QRZ

Naciśnij **Ins** z wypełnionym polem znaku aby pobrać dane z QRZ:
- Imię, QTH, Lokator, Kraj, Strefa CQ, Strefa ITU, DXCC, Kontynent

### Zachowanie Komentarza

Naciśnij **Spację** na przełączniku w obszarze statusu. Gdy włączone, pole Komentarz zachowuje wartość między QSO — przydatne podczas contestów.

### Automatyczne Logowanie z WSJT-X

Gdy WSJT-X jest podłączony przez UDP, QSO są automatycznie zapisywane:
- Rekord ADIF z WSJT-X jest parsowany i zapisywany.
- Duplikaty są wykrywane i pomijane.
- Emisja, subemisja, częstotliwość i operator są pobierane z WSJT-X.
- Automatycznie zalogowane QSO pojawiają się natychmiast w Ostatnich QSO.

---

## Logowanie Contestowe

CQOps wspiera logowanie contestowe z polami wymiany i numeracją.

### Konfiguracja Contestu

1. Przejdź do **Edytora Logbooka** (F8).
2. Naciśnij **Ins** aby utworzyć nowy contest.
3. Wypełnij:
   - **Nazwa** — twoja etykieta dla tego contestu
   - **Data contestu** — data zawodów
   - **Contest ID** — identyfikator ADIF (np. `AADX-CW`, `CQ-WW-SSB`)
   - **Wymiana nadana** — szablon (np. `@rst @serial`)
   - **Wymiana odebrana** — szablon (np. `@rst @serial`)
   - **Prefill wymiany** — automatyczne wypełnianie oczekiwanych wartości
4. Aktywuj contest.

### Znaczniki Wymiany

| Znacznik    | Zastępowany przez                           |
|-------------|---------------------------------------------|
| `@rst`      | Wartość RST Nadane / RST Odebrane           |
| `@serial`   | Automatycznie zwiększany numer              |
| `@call`     | Twój znak wywoławczy                        |
| `@grid`     | Twój lokator                                |
| `@name`     | Twoje imię (z profilu operatora)            |

### Podczas Contestu

- Formularz QSO zyskuje pola **STX** (wymiana nadana) i **SRX** (wymiana odebrana).
- Numery automatycznie się zwiększają po każdym QSO.
- STX_STRING i SRX_STRING są tworzone z szablonów wymiany.
- Tabela Ostatnich QSO filtruje tylko QSO contestowe gdy contest jest aktywny.
- Menu contestu pokazuje aktualny numer i liczbę QSO.

### Po Contescie

- Eksport QSO do ADIF z wszystkimi polami contestowymi.
- QSO contestowe mają pole `CONTEST_ID` w eksporcie ADIF.
- Upload do Wavelog zachowuje metadane contestu.

---

## Edytor Logbooka

Edytor Logbooka (F8) to centralne miejsce do zarządzania QSO, operacjami ADIF i synchronizacją Wavelog.

### Funkcje Edytora

| Akcja               | Klawisz / Menu                               |
|---------------------|----------------------------------------------|
| Przeglądanie QSO    | Tabela widoczna po otwarciu                  |
| Edycja QSO          | **Enter** na wierszu → formularz edycji      |
| Usunięcie QSO       | **Del** na wybranym wierszu (z potwierdzeniem)|
| Filtrowanie QSO     | Ctrl+F → wpisz znak/pasmo/emisję             |
| Import ADIF         | Menu Plik → Import ADIF                      |
| Eksport ADIF        | Menu Plik → Eksport ADIF                     |
| Pobieranie Wavelog  | Menu Wavelog → Pobierz kontakty              |
| Wysyłanie Wavelog   | Menu Wavelog → Wyślij niewysłane QSO         |

### Edycja Inline

1. Wybierz wiersz QSO klawiszami **↑/↓**.
2. Naciśnij **Enter** aby otworzyć formularz edycji.
3. Edytuj pola — Tab/Shift+Tab do nawigacji.
4. Naciśnij **Ctrl+S** lub **Enter** na ostatnim polu aby zapisać.
5. Zmiany są natychmiast widoczne w Ostatnich QSO.

### Import ADIF

- Obsługa formatu ADIF 3.1.7.
- Walidacja rekordów przed importem.
- Wykrywanie i pomijanie duplikatów.
- Podsumowanie: liczba rekordów, zaimportowanych, duplikatów, błędów.

### Eksport ADIF

- Eksport do formatu ADIF 3.1.7.
- Wszystkie standardowe pola ADIF plus pola contestowe.
- Opcja eksportu przefiltrowanych wyników lub całego logbooka.

### Pobieranie z Wavelog

Pobiera QSO z serwera Wavelog do lokalnego logbooka.

- **Przyrostowe**: Pobiera tylko QSO nowsze niż ostatnie pobranie. Pole `last_fetched_id` jest zapisywane dla każdego logbooka.
- **Bezpieczne dla duplikatów**: Już zaimportowane QSO są wykrywane i pomijane.
- **Wznawialne**: Przy przerwaniu ID nie jest aktualizowane — następne pobranie kontynuuje od tego samego miejsca.
- **Świadome czyszczenia**: Wyczyszczenie logbooka resetuje ID pobierania do 0.

### Wysyłanie do Wavelog

Wysyła lokalnie zapisane QSO do Wavelog.

- **Wysyłka wsadowa**: 50 QSO na żądanie HTTP.
- **Śledzenie statusu**: Każde QSO pokazuje status Wavelog.
- **Obsługa duplikatów**: Duplikaty po stronie serwera są wykrywane.
- **Odświeżanie bez wyścigu**: Tabela Ostatnich QSO aktualizuje się natychmiast po uploadzie.

---

## Konfiguracja

Wszystkie ustawienia w `~/.config/cqops/config.yaml`.

### Obsługa Wielu Logbooków

CQOps wspiera wiele logbooków — przydatne do oddzielenia stacji klubowej, terenowej i domowej.

- Przełączanie aktywnego logbooka w Edytorze (F8).
- Każdy logbook ma własne ustawienia stacji, Wavelog, contestów i operatorów.

### Obsługa Wielu Operatorów

- Profile operatorów w konfiguracji (znak + imię).
- Aktywny operator wybierany przez **Ctrl+O**.
- Zalogowane pole OPERATOR używa aktywnego operatora.
- Upload do Wavelog używa znaku aktywnego operatora.

### Obsługa Wielu Radiostacji

- Konfiguracja wielu radiostacji z różnymi backendami.
- Dla każdego: model, antena, moc, konfiguracja rotora.
- WSJT-X można włączyć dla każdej radiostacji osobno.

### Szyfrowane Sekrety

Od wersji 0.8.7, wrażliwe dane są przechowywane szyfrowane:

- **Plik sekretów**: `~/.config/cqops/secrets.enc` (uprawnienia 0600)
- **Szyfrowanie**: AES-256-GCM z kluczem powiązanym z maszyną
- **Chronione dane**: hasło QRZ, login DXC, klucze API Wavelog
- **Auto-migracja**: Tekstowe sekrety ze starszych konfiguracji są migrowane przy pierwszym uruchomieniu
- **Odzyskiwanie**: Jeśli plik sekretów jest uszkodzony, aplikacja uruchamia się normalnie z ostrzeżeniem

### Ekrany Konfiguracji

Dostęp do ekranów konfiguracji z menu Edytora:

- **Stacja** — znak, lokator, strefa CQ/ITU, region IARU, referencje
- **Radio** — model, antena, moc, backend, domyślna częstotliwość/emisja
- **Wavelog** — URL, klucz API, ID profilu stacji
- **QRZ** — nazwa użytkownika, hasło
- **DX Cluster** — host, port, login
- **Powiadomienia** — włączanie/wyłączanie toastów
- **Ogólne** — strefa czasowa, jednostki odległości, mapa, debug

---

## Integracje

### Wavelog

Integracja z platformą logowania w chmurze:
- **Upload**: Wysyłanie lokalnych QSO do Wavelog (partie po 50).
- **Download**: Pobieranie QSO z Wavelog (przyrostowe).
- **Private lookup**: Sprawdzanie statusu przepracowania przed zalogowaniem.

### QRZ.com

Usługa callbook lookup. Wymaga subskrypcji QRZ XML.

- **Ins** na formularzu QSO uruchamia pobieranie danych.
- Wypełnia: imię, QTH, lokator, kraj, strefę CQ/ITU, DXCC, kontynent, IOTA.
- Zdjęcie wyświetlane w widoku Partnera (F3).

### flrig

Sterowanie radiem przez interfejs XML-RPC flrig (HTTP).

- Automatyczne wypełnianie częstotliwości, emisji i mocy.
- Wykrywanie pracy split (VFO A → Freq, VFO B → Freq RX).

### Hamlib Rigctld

Sterowanie radiem przez demona TCP Hamlib.

- Obsługa częstotliwości, emisji, VFO, split i mocy.
- Sterowanie rotorem przez Hamlib rotctld (azymut, elewacja, stop).

### WSJT-X

Automatyczne logowanie QSO z emisji cyfrowych przez UDP.

- Nasłuchuje komunikaty UDP WSJT-X.
- Parsuje ADIF z wiadomości `QsoLogged` i zapisuje QSO.
- Wskaźnik TX (zielona kropka podczas nadawania).
- Synchronizacja częstotliwości i emisji z WSJT-X do formularza QSO.

### DX Cluster

Spoty DX na żywo przez telnet.

- Połączenie z dowolnym węzłem DX Cluster.
- Filtry: stacji nasłuchującej, pasma, emisji, czasu, kontynentu.
- Dialog spota umożliwia dostrojenie radia lub pre-fill formularza QSO.

### PSK Reporter

Raportowanie i wizualizacja propagacji.

- Tabela spotów z znakiem, częstotliwością, SNR, lokatorem, odległością.
- Filtry pasma, czasu i emisji.
- Mapa propagacji w ASCII.

### Solar

Dane słoneczno-ziemskie z hamqsl.com.

- Indeks strumienia słonecznego (SFI), liczba plam (SN).
- Indeks A i K.
- Podsumowanie warunków propagacji.

### Baza REF

Baza referencji SOTA, POTA, WWFF i IOTA.

- Wyszukiwanie po prefiksie, nazwie lub oznaczeniu.
- Automatyczne wypełnianie pól SOTA/POTA/WWFF/IOTA w formularzu.

### Band Plan

Przeglądarka planów pasm amatorskich i broadcast.

- **Pasma KF**: 160m do 23cm.
- **VHF/UHF**: 2m, 70cm i wyższe.
- **CB / PMR446**: kanały CB i PMR446.
- **Broadcast**: stacje AM, FM i SW (BBC, VOA itp.).
- **Eksport** planu pasm do Markdown.

---

## Skróty Klawiszowe

### Globalne

| Klawisz      | Akcja                                  |
|--------------|----------------------------------------|
| F1           | Formularz QSO                          |
| F2           | Ostatnie QSO                           |
| F3           | Widok Partnera                         |
| F4           | DX Cluster                             |
| F5           | PSK Reporter                           |
| F6           | Solar                                  |
| F7           | Band Plan                              |
| F8           | Edytor Logbooka                        |
| ?            | Wszystkie skróty (nakładka pomocy)     |
| Ctrl+C / Esc | Wyjście (z głównego ekranu Esc = menu) |

### Formularz QSO (F1)

| Klawisz    | Akcja                              |
|------------|------------------------------------|
| Tab        | Następne pole                      |
| Shift+Tab  | Poprzednie pole                    |
| ↑ / ↓      | Ruch w kolumnie                    |
| Enter      | Zapisz QSO                         |
| Ctrl+S     | Zapisz QSO (z dowolnego pola)      |
| Esc        | Wyczyść / anuluj                   |
| Ins        | Pobieranie danych QRZ              |
| Spacja     | Przełącznik komentarza / operatora |
| Ctrl+O     | Zmiana operatora                   |

### Ostatnie QSO (F2)

| Klawisz    | Akcja                              |
|------------|------------------------------------|
| ↑ / ↓      | Nawigacja po wierszach             |
| Home/End   | Pierwszy/ostatni wiersz            |
| Enter      | Edycja inline                      |
| Del        | Usuń QSO                           |

### Edytor Logbooka (F8)

| Klawisz    | Akcja                              |
|------------|------------------------------------|
| ↑ / ↓      | Nawigacja po wierszach             |
| Enter      | Edycja QSO                         |
| Del        | Usuń QSO                           |
| Ins        | Nowy (contest/operator)            |
| Ctrl+F     | Filtrowanie QSO                    |
| Esc        | Powrót do menu                     |

### DX Cluster (F4)

| Klawisz    | Akcja                              |
|------------|------------------------------------|
| ↑ / ↓      | Nawigacja po spotach               |
| Enter      | Dialog spota                       |
| S          | Filtr stacji nasłuchującej         |
| B          | Filtr pasma                        |
| M          | Filtr emisji                       |
| T          | Filtr czasu                        |
| C          | Filtr kontynentu                   |

---

## Rozwiązywanie Problemów

### Aplikacja nie uruchamia się

- Sprawdź czy terminal ma minimum 80×24 znaków.
- Na Windows uruchamiaj z Windows Terminal, nie z `cmd.exe`.
- Spróbuj `cqops --offline` aby wykluczyć problemy sieciowe.
- Sprawdź logi w `~/.config/cqops/cqops.log`.

### Radio nie łączy się

- **flrig**: Sprawdź czy flrig działa i port się zgadza (domyślnie `12345`).
- **Hamlib**: Sprawdź czy rigctld działa i port TCP jest poprawny.
- Bursztynowa kropka = łączenie, czerwona = błąd.

### WSJT-X nie loguje automatycznie

- Sprawdź czy WSJT-X wysyła UDP na właściwy host/port.
- W WSJT-X: Ustawienia → Raportowanie → Serwer UDP.
- Zielona kropka WSJT oznacza poprawne połączenie.
- WSJT-X musi być w wersji 2.6 lub nowszej.

### Wysyłka do Wavelog nie działa

- Sprawdź URL, klucz API i ID profilu stacji w konfiguracji.
- Zielona kropka WL oznacza dostępność.
- Błędy są pokazywane jako toasty; QSO pozostaje zapisane lokalnie.

### Problemy z plikiem konfiguracyjnym

- Konfiguracja: `~/.config/cqops/config.yaml` (Linux/macOS) lub `%APPDATA%\cqops\config.yaml` (Windows).
- Sekrety w `secrets.enc` w tym samym katalogu.
- Jeśli konfiguracja jest uszkodzona, usuń ją i uruchom ponownie.
- Pole `last_fetched_id` pojawia się dopiero po udanym pobraniu z Wavelog.

### Problemy z wydajnością

- Wyłącz renderowanie mapy w ustawieniach ogólnych.
- Wyłącz panel solarny na ekranie QSO.
- Zamknij nieużywane zakładki (DXC, PSK).
- Uruchom z `--offline` przy niestabilnym internecie.

### Zgłaszanie Błędów

Zgłaszaj problemy na [GitHub Issues](https://github.com/szporwolik/cqops/issues) podając:
- Wersję CQOps (`cqops --version`)
- System operacyjny i emulator terminala
- Kroki do odtworzenia
- Logi z `~/.config/cqops/cqops.log`
