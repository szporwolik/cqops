---
title: Podręcznik użytkownika CQOps
description: Praktyczna instrukcja konfiguracji CQOps, logowania łączności i pracy ze stacji domowej lub w terenie
---

# Podręcznik użytkownika CQOps

CQOps to obsługiwany klawiaturą logger krótkofalarski do pracy domowej, terenowej, klubowej i okazjonalnego udziału w zawodach. QSO są zapisywane najpierw na komputerze; usługi internetowe są opcjonalne. Zacznij od ręcznego logowania, a sterowanie radiem i usługi online dodaj wtedy, gdy będą potrzebne.

Nazwy menu i pól w tej instrukcji odpowiadają angielskiemu interfejsowi. Skróty zależą od aktywnego ekranu: sprawdzisz je na dolnym pasku pomocy lub pod **?**.

## Spis treści

1. [Instalacja](#installation)
2. [Pierwsza konfiguracja](#setup)
3. [Pierwsze QSO](#first-qso)
4. [Ekrany i status](#screens)
5. [Codzienne logowanie](#logging)
6. [Profile stacji](#profiles)
7. [Dziennik i kopie zapasowe](#logbook)
8. [Radio i emisje cyfrowe](#radio)
9. [Usługi internetowe](#online)
10. [GPS i APRS](#position)
11. [Praca w terenie](#portable)
12. [Zawody](#contests)
13. [CQOps Live](#dashboard)
14. [Skróty klawiaturowe](#keys)
15. [Rozwiązywanie problemów i pomoc](#help)

<a id="installation"></a>

## Instalacja

Pobierz CQOps ze [strony wydań](https://github.com/szporwolik/cqops/releases). Okno terminala musi mieć co najmniej 75 × 24 znaki; wygodniej pracuje się przy 80 × 43 lub więcej.

| System | Instalacja |
|---|---|
| Windows | Pobierz `cqops-setup.exe` lub rozpakuj `cqops-windows-portable.zip`, aby korzystać bez instalacji. Zalecany jest Windows Terminal. |
| Debian, Ubuntu, Linux Mint, Pop!_OS | Pobierz odpowiedni `.deb`: `amd64` dla większości komputerów Intel/AMD, `arm64` dla 64-bitowego ARM lub `armhf` dla 32-bitowego Raspberry Pi OS. Otwórz go w instalatorze pakietów. |
| Fedora, RHEL, Rocky, AlmaLinux | Użyj podanych niżej poleceń instalacji z repozytorium. |
| Arch, Manjaro, CachyOS | Zainstaluj pakiet AUR przez `paru -S cqops-bin` lub `yay -S cqops-bin`. |
| Inne systemy Linux | Pobierz ze strony wydań archiwum Linux `.tar.gz` dla swojego procesora i rozpakuj je. |
| macOS | Pobierz `cqops-darwin-arm64` dla Apple Silicon lub `cqops-darwin-amd64` dla Intela. Użyj poleceń poniżej. |

W systemach opartych na Debianie możesz zamiast tego użyć repozytorium:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

W systemach opartych na Fedorze:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

W macOS wykonaj te polecenia w katalogu pobierania, zastępując `FILE` dokładną nazwą pobranego pliku:

```bash
chmod +x FILE
sudo mv FILE /usr/local/bin/cqops
```

Uruchom `cqops` lub rozpakowany program przenośny. `cqops --offline` uruchamia tryb offline, `cqops --version` wyświetla wersję, a `cqops --help` pokazuje opcje uruchamiania. Przed aktualizacją wyeksportuj dzienniki.

<a id="setup"></a>

## Pierwsza konfiguracja

Kreator pierwszego uruchomienia pyta o nazwę dziennika, znak stacji, lokator Maidenhead i kontynent. **Znak stacji** to znak używany podczas nadawania; **profil operatora** określa osobę obsługującą stację.

**Ctrl+A** odsłania opcjonalne referencje stacji i strefy CQ/ITU. Własną referencję SOTA/POTA/WWFF wpisz w ustawieniach stacji/dziennika; referencje w formularzu QSO dotyczą korespondenta. Region IARU ustaw później w **F9 → Logbooks**.

Utwórz profil radia, podając nazwę, antenę i moc. Wybierz **None** dla ręcznego wpisywania częstotliwości i emisji albo **flrig** lub **Hamlib**. Opcjonalne połączenia skonfiguruj po sprawdzeniu podstawowego logowania.

**Tab / Shift+Tab** przenosi między polami, **Space** zmienia opcje, a przycisk **Save & Next** przechodzi dalej. **Esc** cofa, **F10** kończy program. Sprawdź podsumowanie i zapisz. CQOps wykrywa strefę czasową komputera; daty i godziny QSO są w UTC. Przed pracą sprawdź zegar komputera.

<a id="first-qso"></a>

## Pierwsze QSO

1. Naciśnij **F1**. Sprawdź dziennik, znak stacji, operatora, radio i zawody.
2. Wpisz znak korespondenta. **Ins** uruchamia wyszukiwanie, jeśli je skonfigurowano. Priorytet oznacza zaufanie do danych: domyślnie QRZ.com (100) > HamQTH (90) > Callook.info (80) > QRZ.RU (70) > lokalny dziennik (60) > Wavelog (10), CTY.DAT zawsze na końcu; dostawcy o niższym priorytecie uzupełniają tylko puste pola.
3. Sprawdź datę i czas UTC, częstotliwość w MHz, pasmo, emisję oraz raporty nadany i odebrany.
4. Uzupełnij imię, QTH, lokator, referencję lub komentarz, jeśli są przydatne.
5. Naciśnij **Enter**. Łączność pojawi się w Recent QSOs.

Jeśli **DUPE!** wymaga potwierdzenia, ponownie naciśnij **Enter**, aby zapisać mimo ostrzeżenia, albo **Esc**, aby anulować potwierdzenie. Ostrzeżenie oznacza, że warto sprawdzić łączność, nie że trzeba ją odrzucić.

<a id="screens"></a>

## Ekrany i status

| Klawisz | Ekran | Przeznaczenie |
|---|---|---|
| F1 | QSO | Wpisywanie łączności i ostatnie QSO |
| F2 | Partner | Dane korespondenta, mapa, statystyki, zdjęcie |
| F3 | APRS | Stacje w pobliżu |
| F4 | DX Cluster | Spoty i filtry |
| F5 | PSK Reporter | Raporty odbioru emisji cyfrowych |
| F6 | References | Wyszukiwanie SOTA, POTA, WWFF, IOTA |
| F7 | Band Plan | Częstotliwości i gotowe ustawienia |
| F8 | Logbook | Edycja, import, eksport, synchronizacja |
| F9 | Configuration | Ustawienia stacji i usług |
| F10 | Quit | Zakończenie CQOps |

Górny pasek pokazuje aktywną konfigurację stacji, czas lokalny (**L**) i UTC (**Z**). Biały wskaźnik zwykle oznacza aktywne połączenie, żółty — wyłączenie/łączenie/oczekiwanie, a czerwony — błąd. WSJT jest wyróżniony podczas nadawania. **WL!** ostrzega o nieobsługiwanym starym kluczu Wavelog.

<a id="logging"></a>

## Codzienne logowanie

**Tab / Shift+Tab** przechodzi między polami, a **PgUp / PgDn** zmienia pasmo, emisję lub podemisję. **Shift+Backspace** czyści bieżące pole; **Del** czyści formularz. Przy pracy split sprawdź **Freq RX**.

**Keep** zachowuje komentarz po zapisie. **Retain** zachowuje cały formularz: przed kolejnym zapisem sprawdź znak, czas, raporty i referencje. Pola wymiany zawodniczej są widoczne tylko przy aktywnych zawodach. **SIG / SIG Info** służą do innych informacji o grupach zainteresowań.

Gdy oba lokatory są znane, CQOps pokazuje odległość i azymut. Lokalizacja z callbooka może dotyczyć stacji domowej, nie aktualnego miejsca pracy terenowej — zweryfikuj ją. Oznaczenia nowego znaku, nowego DXCC i duplikatu pomagają ocenić łączność.

**F6** wyszukuje referencje według nazwy lub oznaczenia i może uzupełnić referencję korespondenta. **F7** wyświetla band plany i może przestrajać podłączone radio. To pomoc operatorska, a nie zgoda na nadawanie: sprawdź swoje uprawnienia i lokalny band plan.

Trzy wspólne ulubione zapisują częstotliwość, emisję i pasmo:

| Pozycja | Przywołanie | Zapis bieżących wartości |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

<a id="profiles"></a>

## Profile stacji

Twórz dzienniki, operatorów, radia i zawody w odpowiednich menu **F9**; **Ins** dodaje wpis. Na ekranie QSO:

| Skrót | Przełącza |
|---|---|
| Ctrl+L | Dziennik |
| Ctrl+O | Operatora |
| Ctrl+R | Radio |
| Ctrl+C | Zawody |

Dzienniki mają osobne dane stacji i ustawienia Wavelog/APRS. Profile operatorów określają osobę przy radiu; znak trafia do pola ADIF `OPERATOR`. Profile radia przechowują sprzęt, moc, sterowanie radiem, rotorem i ustawienia WSJT-X. Po każdym przełączeniu sprawdzaj pasek statusu, zwłaszcza przy automatycznym logowaniu emisji cyfrowych.

Pozostałe menu **F9** obejmują wygląd, jednostki, strefę czasową, callbooki, integracje i dźwięki powiadomień.

<a id="logbook"></a>

## Dziennik i kopie zapasowe

Na **F8** wybierz QSO i naciśnij **Enter** lub **e**, aby je edytować. Zapisz przez **Enter** i potwierdź. **Delete** usuwa wybraną łączność. Przed zmianami zbiorczymi zrób kopię; **Ctrl+P** to destrukcyjne usunięcie wszystkich QSO, nie wyszukiwanie.

| Skrót na F8 | Działanie |
|---|---|
| Ctrl+I | Import ADIF, sprawdzenie rekordów i pominięcie duplikatów |
| Ctrl+E | Eksport wszystkich łączności lub wyboru według zawodów |
| Ctrl+W | Wysłanie niewysłanych łączności do Wavelog |
| Alt+W | Pobranie z Wavelog |

Sprawdź podsumowanie importu i zakres eksportu. Zaimportowane łączności mogą następnie trafić do Wavelog. CQOps obsługuje ADIF 3.1.7 i zachowuje identyfikatory zawodów oraz wymiany. Przechowuj osobne kopie każdego dziennika, najlepiej na innym urządzeniu. ADIF zabezpiecza łączności, ale nie wszystkie ustawienia i dane logowania.

Konfiguracja znajduje się w `~/.config/cqops/config.yaml` na Linux/macOS i `%APPDATA%\cqops\config.yaml` na Windows. Dane dostępowe są osobno w `secrets.enc`; po przeniesieniu na inny komputer wpisz je ponownie. Nie zaczynaj rozwiązywania problemów od usuwania konfiguracji.

<a id="radio"></a>

## Radio i emisje cyfrowe

### Sterowanie radiem

W **F9 → Rigs** wybierz flrig lub Hamlib i dopasuj ustawienia połączenia. Najpierw uruchom flrig lub `rigctld`. flrig zwykle używa `localhost:12345`. Dostępne odczyty częstotliwości, emisji, splitu i mocy zależą od radia. Przy **None** wpisuj je ręcznie.

### WSJT-X

Używaj WSJT-X 2.6 lub nowszego. Dopasuj **Settings → Reporting → UDP Server** w WSJT-X do ustawień UDP aktywnego profilu radia CQOps. Zapisz ukończone próbne QSO w WSJT-X i sprawdź, czy pojawia się w CQOps.

Odebrane QSO trafiają do aktywnego dziennika i zawodów; duplikaty są pomijane. Przed sesją sprawdź operatora i wskaźnik WSJT. CQOps ostrzega o niezgodności operatora. Po skonfigurowaniu możliwe jest wysłanie do Wavelog. Wybierz właściwe Mode/Submode; FT8 jest eksportowane jako FT8, a FT4/FT2 jako MFSK z odpowiednią podemisją.

### Sterowanie rotorem

Sterowanie przez Hamlib `rotctld` jest eksperymentalne. Sprawdź kierunek i ograniczenia mechaniczne. Zapewnij bezpieczny sposób zatrzymania: błędne ustawienia mogą uszkodzić antenę, rotor lub przewód antenowy.

| Skrót | Działanie |
|---|---|
| Alt+, / Alt+. | Azymut −5° / +5° |
| Alt+' / Alt+; | Elewacja −5° / +5° |
| Alt+\ | Obrót na obliczony azymut |
| Alt+/ | Zatrzymanie ruchu |

<a id="online"></a>

## Usługi internetowe

### Callbooki

Ustaw dostawców i kolejność w **F9 → Callbook**, a następnie naciśnij **Ins** w formularzu QSO. CQOps odpytuje włączonych dostawców po kolei. Wyszukiwanie znaku bazowego może pominąć prefiks/sufiks pracy terenowej; sprawdź zwróconą lokalizację.

| Dostawca | Dostęp |
|---|---|
| QRZ.com | Subskrypcja XML i dane logowania |
| HamQTH | Bezpłatne konto |
| QRZ.RU | Login API, osobny od danych strony WWW |
| Callook.info | Znaki USA; bez konta |

**F2** pokazuje dane korespondenta. Zdjęcia zależą od dostawcy i terminala; eksperymentalne **Kitty Graphics** w General wymaga zgodnego terminala, np. Kitty, Ghostty lub WezTerm.

### Wavelog

Dla każdego dziennika ustaw URL, token API v2 (`wl2_…`) i profil stacji. Stare klucze v1 nie są obsługiwane. Wybór stacji Wavelog może uzupełnić lokalne dane: przed zapisem sprawdź znak, lokator i referencje.

QSO są najpierw zapisywane lokalnie. Nieudane wysyłanie ponów przez **F8 → Ctrl+W**; **Alt+W** pobiera łączności. Otwarcie powiązanego QSO do edycji może odświeżyć je z Wavelog. Edycja i usuwanie online wpływają też na kopię zdalną. Czytaj potwierdzenia, szczególnie offline; nie zakładaj, że zmiana wyłącznie lokalna dotarła do Wavelog.


Stacje klubowe: użyj klucza `wl2_` właściciela razem z opcją **Wspólna stacja klubowa** w formularzu dziennika. Zsynchronizowane łączności stają się tylko do odczytu — edycje i usunięcia wykonuje się po stronie Wavelog, a CQOps nigdy nie wysyła dla nich PATCH ani DELETE. Nowe łączności są wysyłane normalnie i przypisywane aktywnemu operatorowi (gdy go brak — znakowi stacji). Klucz API jest zapisywany szyfrowanie i przy włączonej opcji po zapisie nigdy nie jest ponownie wyświetlany — pozostaw pole puste, aby go zachować, wpisz nowy, aby zastąpić.
### DX Cluster i propagacja

Skonfiguruj DX Cluster w Integrations i otwórz **F4**. **b / c / m / t** filtrują pasmo, kontynent spotującego, emisję i wiek spotu. **Backspace** czyści filtry. **Enter** wypełnia formularz QSO, przestraja podłączone radio i wraca do F1; **Space** przestraja bez opuszczania klastra.

Na F1 **Ctrl+S** otwiera okno wysyłania spotu, a **Ctrl+P** pobiera znak z najbliższego wyświetlanego spotu. Sprawdź go przed wysłaniem. **F5** pokazuje raporty odbioru PSK Reporter, nie gwarancję bieżącej propagacji. Panel Solar pokazuje warunki HamQSL; dane z pamięci podręcznej mogą być nieaktualne. **F5 jest domyślnie wyłączone — włącz PSK Reporter w Integrations.**

<a id="position"></a>

## GPS i APRS

### GPS

W Integrations skonfiguruj odbiornik GPS przez port szeregowy lub GPSD. Włącz **Grid from GPS** w ustawieniach stacji/dziennika, aby korzystać z lokatora w QSO, azymutach, APRS i panelu WWW. Czerwony GPS oznacza błąd, żółty — brak pozycji, biały — ustaloną pozycję. Sprawdź lokator przed pracą. Wybierz 6, 8 lub 10 znaków; dłuższy lokator nie gwarantuje większej dokładności odbiornika.

### APRS

| Usługa | Połączenie |
|---|---|
| APRS-IS | Serwer APRS w internecie |
| KISS | Sprzętowy TNC przez port szeregowy i radio |
| KISS Server | TNC TCP, np. Dire Wolf; może działać lokalnie |

Wybierz usługę w **F9 → Integrations → APRS**. Znak/SSID, symbol, komentarz, zasięg i odstęp beaconów ustaw w **F9 → Logbooks → [active logbook] → APRS**. **APRS TX** i **Send beacons** włącz tylko, jeśli chcesz nadawać. Sam odbiór sygnalizuje **APRS-RX**. Beacony ujawniają pozycję; najpierw sprawdź ją i odbiorców transmisji.

Automatyczne beacony są nadawane nie częściej niż co pięć minut. **F3** pokazuje ostatnio słyszane stacje: strzałki wybierają, **Enter** wypełnia QSO, **d / t / s** zmienia filtry odległości/wieku/typu, **Backspace** je czyści, a **b** od razu wysyła skonfigurowany beacon. Beacony z GPS wymagają **Grid from GPS** i prawidłowej pozycji.

<a id="portable"></a>

## Praca w terenie

Przed wyjazdem wybierz dziennik terenowy i sprawdź znak, lokator, referencję aktywacji, radio, antenę i moc. Przetestuj cały zestaw i uruchom CQOps online, aby odświeżyć referencje i prefiksy. Sprawdź, czy **F6** znajduje potrzebne referencje. Wyeksportuj kopię zapasową.

Bez internetu lokalne logowanie nadal działa. `cqops --offline` pomija funkcje sieciowe; nie polegaj na wyszukiwaniu online ani synchronizacji. Przed wyjazdem przetestuj sprzęt w sieci lokalnej w wybranym trybie uruchamiania. Dane z pamięci podręcznej mogą być stare.

Po powrocie sprawdź liczbę QSO i referencje, wyeksportuj ADIF, zachowaj kopię i wyślij niewysłane łączności do Wavelog, jeśli go używasz. Sprawdź format wymagany przez dany program dyplomowy; w razie potrzeby przekonwertuj eksport.

<a id="contests"></a>

## Zawody

CQOps obsługuje okazjonalne logowanie zawodów, wymiany, numery kolejne i tempo pracy. Nie jest pełnym systemem punktacji i wysyłania zgłoszeń. Do zaawansowanej pracy używaj loggera zawodniczego.

W **F9 → Contests** naciśnij **Ins** i ustaw nazwę, datę, identyfikator ADIF zawodów, numer początkowy i szablony wymiany nadawanej/odbieranej.

| Znacznik | Wartość |
|---|---|
| `@rst` | Raport nadany lub odebrany |
| `@serial` | Numer kolejny |
| `@cqz` / `@mycqz` | Strefa CQ korespondenta / własna |
| `@itu` / `@myitu` | Strefa ITU korespondenta / własna |
| `@grid` / `@mygrid` | Lokator korespondenta / własny |

Na **F1** skrót **Ctrl+C** przełącza zawody. Przed nadawaniem sprawdź wymianę i następny numer. Pasek statusu pokazuje liczbę QSO, następny numer i czasy; szersze okna pokazują więcej statystyk tempa. Po zakończeniu wyłącz aktywne zawody.

Aby wyeksportować, otwórz **F8**, wybierz filtr zawodów przez **Ctrl+C**, potem **Ctrl+E** i sprawdź zakres eksportu. Wynik to ADIF, nie Cabrillo. Przestrzegaj wymagań organizatora dotyczących formatu i zgłoszenia.

<a id="dashboard"></a>

## CQOps Live

Włącz **F9 → Integrations → HTTP Server** i zapisz przez **Ctrl+S**. Na komputerze z CQOps otwórz `http://localhost:8073`.

Domyślne `0.0.0.0` pozwala na dostęp z sieci lokalnej, o ile zezwala zapora. Na innym urządzeniu użyj adresu IP komputera z CQOps i portu `8073`. `127.0.0.1` ogranicza dostęp do samego komputera z CQOps. Korzystaj z zaufanej sieci; nie przekierowuj portu do internetu.

Panel automatycznie pokazuje bieżącą łączność, mapy QSO, ostatnie kontakty, tempo, operatorów, APRS i dostępne dane propagacyjne/pogodowe. Warstwy internetowe mogą nie działać offline. Header 1, Header 2, Logo URL i Event Start dostosowują ekran wydarzenia; data początkowa filtruje wyświetlane statystyki i listy QSO.

<a id="keys"></a>

## Skróty klawiaturowe

| Ekran | Klawisze | Działanie |
|---|---|---|
| Ogólne | ? / Esc / F10 | Pomoc / powrót / wyjście |
| QSO | Tab / Shift+Tab | Następne / poprzednie pole |
| QSO | Enter / Ins | Zapis / wyszukiwanie |
| QSO | Shift+Backspace / Del | Wyczyszczenie pola / formularza |
| QSO | Ctrl+L / Ctrl+O / Ctrl+R / Ctrl+C | Zmiana dziennika / operatora / radia / zawodów |
| Dziennik | ↑ / ↓, PgUp / PgDn, Home / End | Wybór, zmiana strony, pierwszy / ostatni wiersz |
| Dziennik | Enter lub e / Delete | Edycja / usunięcie wybranego QSO |
| Dziennik | Ctrl+I / Ctrl+E | Import / eksport ADIF |
| Dziennik | Ctrl+W / Alt+W | Wysyłanie / pobieranie Wavelog |
| Dziennik | Ctrl+C / Backspace | Filtr zawodów / wyczyszczenie wyszukiwania |

Skróty zależą od ekranu: **Ctrl+C nie zamyka programu**. Na laptopie klawisze funkcyjne mogą wymagać **Fn**. Jeśli terminal przechwytuje skrót, sprawdź jego ustawienia klawiatury i pasek pomocy CQOps.

<a id="help"></a>

## Rozwiązywanie problemów i pomoc

| Problem | Co sprawdzić najpierw |
|---|---|
| Uruchamianie lub niepełny ekran | Rozmiar terminala; Windows Terminal na Windows; próba `cqops --offline` |
| Brak połączenia z radiem | Aktywny profil, uruchomione flrig/rigctld, model, port szeregowy, prędkość, host/port, zajęcie portu przez inny program |
| Brak QSO z WSJT-X | Zgodność UDP, wskaźnik WSJT, faktyczny zapis zakończonego QSO w WSJT-X, aktywny dziennik |
| Błąd Wavelog | URL, token `wl2_`, profil stacji, internet; lokalne QSO pozostają bezpieczne |
| Brak pozycji GPS | Port/prędkość lub adres GPSD, widok nieba, prawidłowa pozycja, włączone Grid from GPS |
| APRS nie nadaje beaconów | Dziennik, APRS TX i Send beacons, znak/SSID, TNC/radio lub internet |
| Panel niedostępny | Włączony serwer, właściwy IP i port, zapora; localhost oznacza urządzenie, na którym działa przeglądarka |

Przy problemach z ustawieniami użyj **F9**, zanim zaczniesz zmieniać pliki. Wpisz ponownie dane dostępowe, jeśli CQOps zgłasza problem z zapisanymi sekretami lub został przeniesiony na inny komputer.

W razie potrzeby włącz **F9 → General → Debug**, bezpiecznie odtwórz problem i zbierz odpowiedni log diagnostyczny; potem wyłącz debugowanie.

| System | Logi diagnostyczne |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

Zgłaszaj problemy w [GitHub Issues](https://github.com/szporwolik/cqops/issues). Podaj wersję CQOps, system, terminal, kroki odtworzenia i odpowiedni log. Usuń hasła, tokeny API i prywatne informacje przed udostępnieniem.
