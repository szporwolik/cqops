---
title: Manuale utente CQOps
description: Guida all'installazione, configurazione e uso di CQOps — un logger radioamatoriale veloce, orientato al terminale
---

> **Nota sulla traduzione:** Questa traduzione è stata generata con un modello LLM. Le correzioni sono benvenute come Pull Request verso il branch `dev`. Alcuni nomi di schermate, campi, comandi e scorciatoie restano intenzionalmente in inglese per corrispondere all'interfaccia CQOps.

# Manuale utente CQOps

CQOps è un logger radioamatoriale veloce, orientato al terminale, per operatori che vogliono registrare QSO da tastiera con basso carico di sistema. È progettato per lo shack, l'operatività portatile, le stazioni di club, i field day e macchine come Raspberry Pi o vecchi laptop.

CQOps salva sempre i QSO prima in locale. Le integrazioni basate su internet sono opzionali.

## Indice

1. [Cos'è CQOps](#cosè-cqops)
2. [Download e installazione](#download-e-installazione)
3. [Primo avvio](#primo-avvio)
4. [Registrare il primo QSO](#registrare-il-primo-qso)
5. [Schermata principale](#schermata-principale)
6. [Flussi di lavoro comuni](#flussi-di-lavoro-comuni)
7. [Registrazione QSO](#registrazione-qso)
8. [Logbook Editor e ADIF](#logbook-editor-e-adif)
9. [Contest](#contest)
10. [Favorites, riferimenti e band plan](#favorites-riferimenti-e-band-plan)
11. [Integrazioni](#integrazioni)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Configurazione](#configurazione)
14. [Scorciatoie da tastiera](#scorciatoie-da-tastiera)
15. [Risoluzione dei problemi](#risoluzione-dei-problemi)
16. [Segnalazione bug](#segnalazione-bug)

---

## Cos'è CQOps

CQOps è costruito intorno a inserimento rapido dei QSO, logging local-first e operatività pratica sul campo.

### Idee principali

- **Terminal-first** — ottimizzato per l'uso da tastiera.
- **Offline-first** — il logging locale dei QSO funziona senza accesso a internet.
- **Basso overhead** — adatto a sistemi di classe Raspberry Pi, vecchi laptop e PC di stazioni condivise.
- **Design portatile** — distribuito come singolo binario Go.
- **Più logbook** — utile per log personali, portatili, contest e club.
- **Più operatori** — utile per hot-seat e stazioni di club condivise.
- **Più rig** — ogni preset di rig può avere il proprio backend e impostazioni WSJT-X.
- **Integrazioni opzionali** — QRZ.com, Wavelog, DX Cluster, PSK Reporter, APRS, ricevitore GPS, controllo rig, controllo rotore, dati solari e CQOps Live nel browser.

Il logging locale non richiede accesso a internet. Le funzioni di rete vengono saltate in modalità `--offline`.

### Per chi è CQOps

CQOps è adatto a:

- operatori portatili,
- attivatori SOTA e POTA,
- stazioni di club,
- team di field day,
- operatori che preferiscono un flusso di lavoro da terminale,
- stazioni che devono passare rapidamente tra operatori, logbook o rig.

CQOps non intende sostituire ogni funzione di un logger desktop completo o di una piattaforma web di logbook. Si concentra su logging veloce da terminale, operatività sul campo, uso offline e flussi di stazione condivisa.

---

## Download e installazione

Tutte le release:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Pacchetto | Link | Note |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Consigliato per la maggior parte degli utenti. Aggiunge CQOps al Start Menu e al PATH. |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Estrai ed esegui senza installare. |

Usa **Windows Terminal** invece della vecchia console.

### Linux — Debian / Ubuntu

| Architettura | Link | Uso |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La maggior parte dei PC Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Sistemi ARM a 64 bit |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS a 32 bit |

Installa il pacchetto scaricato:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — tarball portatile

| Architettura | Link | Uso |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La maggior parte dei PC Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Sistemi ARM a 64 bit |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS a 32 bit |

### macOS

| Architettura | Link | Uso |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Mac M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Mac Intel |

Installazione manuale:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Compilare dal sorgente

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

La compilazione dal sorgente richiede Go 1.26 o più recente.

### Requisiti del terminale

| Requisito | Valore |
|---|---|
| Dimensione minima terminale | 80×24 caratteri |
| Dimensione consigliata | 80×43 caratteri o più |
| Terminale consigliato su Windows | Windows Terminal |
| Terminale per grafica Kitty | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) o [WezTerm](https://wezfurlong.org/wezterm/) |

### Comandi base

```bash
cqops              # Avvia la TUI
cqops --offline    # Avvia senza attività di rete
cqops --version    # Mostra la versione ed esce
cqops --help       # Mostra l'aiuto
```

---

## Primo avvio

Al primo avvio CQOps apre il setup wizard. Per il logging locale servono solo le informazioni essenziali della stazione. Le integrazioni di rete possono essere saltate e configurate più tardi.

### Pagine del wizard

| Pagina | Cosa configura |
|---|---|
| Station & Logbook | Logbook iniziale, nominativo di stazione, operatore, grid locator, riferimenti e zone opzionali, Wavelog URL/API/station profile ID |
| Rig | Preset di rig, modello, antenna, potenza, backend, rotore opzionale, impostazioni UDP WSJT-X opzionali |
| Integrations | Impostazioni QRZ.com lookup |
| General | Fuso orario IANA |
| Summary | Revisione e salvataggio |

Backend rig supportati:

- None,
- flrig,
- Hamlib `rigctld`.

### Navigazione nel wizard

| Tasto | Azione |
|---|---|
| Ctrl+S | Valida e continua; su Summary salva e avvia CQOps |
| Esc | Torna indietro |
| F10 | Esci |
| Tab / Shift+Tab | Sposta tra i campi |
| Space | Commuta checkbox |

Le impostazioni del wizard possono essere modificate più tardi con **F9**.

---

## Registrare il primo QSO

1. Avvia CQOps:

   ```bash
   cqops
   ```

2. Completa il setup wizard almeno con nominativo e grid locator.
3. Apri il QSO form con **F1**.
4. Inserisci il nominativo del contatto. CQOps converte automaticamente i nominativi in maiuscolo.
5. Compila gli altri campi. Se il rig attivo è collegato tramite flrig o Hamlib, CQOps può compilare automaticamente frequenza, banda, mode e submode.
6. Premi **Enter** o **Ctrl+S** per salvare.
7. Se compare un avviso **DUPE!**, premi di nuovo **Enter** per salvare comunque, oppure **Esc** per annullare.

Il QSO salvato appare subito nella tabella Recent QSOs.

---

## Schermata principale

CQOps usa un layout fisso nel terminale:

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

La Status bar mostra:

- versione CQOps,
- logbook attivo,
- rig attivo,
- nominativo di stazione,
- operatore attivo,
- etichette di stato delle integrazioni,
- ora locale marcata come `L`,
- ora UTC marcata come `Z`.

Etichette comuni: **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** e **GPS**. L'etichetta GPS segue la stessa convenzione di colori — rosso quando disconnesso, giallo quando connesso senza fix, bianco quando una posizione è acquisita.

| Colore | Significato |
|---|---|
| Bianco/predefinito | Connesso o attivo |
| Giallo | Disabilitato, in connessione o offline previsto |
| Rosso | Errore o disconnesso |
| Accento + grassetto | WSJT-X sta trasmettendo |

### Schede principali

| Tasto | Scheda | Schermata |
|---|---|---|
| F1 | QSO | QSO form e Recent QSOs |
| F2 | QRZ | Partner view: dati callbook, mappa, statistiche, foto |
| F4 | DXC | Spot DX Cluster e filtri |
| F5 | HRD | Spot PSK Reporter e mappa di propagazione |
| F6 | REF | Ricerca riferimenti SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, sincronizzazione Wavelog |
| F9 | CFG | Menu di configurazione |

La Help bar mostra le scorciatoie rilevanti per la schermata attiva. **?** apre l'aiuto completo.

---

## Flussi di lavoro comuni

### Operatività portable, SOTA o POTA

Prima di uscire:

1. Avvia CQOps una volta con accesso a internet.
2. Lascia che CQOps scarichi o aggiorni dati in cache come dati solari, dati REF e prefissi DXCC.
3. Controlla che il Solar panel mostri dati.
4. Controlla che REF search su **F6** restituisca risultati.

Sul campo:

1. Avvia CQOps in modalità offline:

   ```bash
   cqops --offline
   ```

2. Registra normalmente. I QSO vengono salvati in locale.
3. Quando torni online, apri **F8** e premi **w** per caricare su Wavelog i QSO non inviati.

### Stazione di club condivisa e hot-seat logging

1. Apri **F9 → Operators**.
2. Premi **Ins** per aggiungere profili operatore.
3. Nel QSO form, premi **Ctrl+O** per cambiare l'operatore attivo.
4. Controlla l'operatore attivo nella Status bar prima di salvare.
5. Usa **Retain** quando più operatori devono registrare contatti simili senza riscrivere tutto il form.

L'operatore attivo viene salvato nel campo ADIF `OPERATOR`.

### Logbook personali e di club

1. Apri **F9 → Logbooks**.
2. Premi **Ins** per creare ogni logbook.
3. Nel QSO form, premi **Ctrl+L** per cambiare il logbook attivo.
4. Controlla il logbook attivo nella Status bar prima di salvare.

Ogni logbook può mantenere propri dettagli di stazione, impostazioni Wavelog, impostazioni contest e operatori.

### Più rig

1. Apri **F9 → Rigs**.
2. Premi **Ins** per creare preset di rig.
3. Seleziona backend: None, flrig o Hamlib.
4. Nel QSO form, premi **Ctrl+R** per cambiare il rig attivo.

Un preset di rig può includere backend, modello, antenna, potenza, impostazioni rotore e impostazioni UDP WSJT-X.

### Operatività digitale WSJT-X

Quando l'integrazione UDP WSJT-X è abilitata, CQOps può ricevere messaggi ADIF da WSJT-X e registrare automaticamente i QSO digitali completati.

I QSO auto-registrati:

- vengono salvati nel logbook attivo,
- appaiono subito in Recent QSOs,
- saltano i duplicati,
- ereditano il contest ID attivo,
- possono essere caricati automaticamente su Wavelog quando Wavelog è configurato e raggiungibile.

Se l'operatore riportato da WSJT-X non corrisponde all'operatore attivo in CQOps, CQOps mostra un avviso.

Prima di lunghe sessioni digitali, controlla:

- logbook attivo,
- operatore attivo,
- contest attivo,
- etichetta di stato WSJT-X.

### Sincronizzazione Wavelog

CQOps salva i QSO prima in locale. La sincronizzazione Wavelog è opzionale.

| Azione | Dove | Scorciatoia | Note |
|---|---|---|---|
| Caricare QSO non inviati | Logbook Editor | `w` | Carica in batch di 50 |
| Scaricare da Wavelog | Logbook Editor | `Ctrl+W` | Download incrementale tramite `last_fetched_id` |

Lo stato di upload è tracciato per QSO:

- not sent,
- sent,
- error.

Se l'upload fallisce, il QSO resta nel logbook locale e può essere riprovato più tardi. Il purging di un logbook reimposta il fetch ID a `0`, permettendo un nuovo download completo.

---

## Registrazione QSO

Il QSO form è la schermata principale di registrazione. Si apre con **F1**.

CQOps può compilare i campi da:

| Fonte | Campi |
|---|---|
| flrig / Hamlib | Frequency, Freq RX se split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| REF database | Riferimenti SOTA, POTA, WWFF, IOTA |
| Wavelog lookup | Worked/confirmed status quando configurato |
| DXCC/prefix data | Dati di prefisso e paese |

### Layout del form

| Colonna sinistra | Colonna centrale | Colonna destra |
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

I campi exchange compaiono solo quando un contest è attivo.

La riga inferiore contiene:

- **Comment**,
- **Keep** — conserva il campo Comment tra i QSO,
- **Retain** — conserva l'intero form dopo il salvataggio.

Campi come Band, Mode e Submode possono essere cambiati con **PgUp/PgDn**.

### Percorso, direzione e badge

Quando entrambi i grid locator sono noti, CQOps mostra distanza e azimut.

Il QSO form può anche mostrare badge come:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Salvataggio

| Tasto | Azione |
|---|---|
| Enter | Salva QSO |
| Ctrl+S | Salva QSO da qualsiasi campo |
| Esc | Annulla conferma duplicato |
| Enter su conferma DUPE | Salva comunque il duplicato |

---

## Logbook Editor e ADIF

Apri Logbook Editor con **F8**.

Serve per:

- revisione QSO,
- modifica inline,
- eliminazione QSO,
- import ADIF,
- export ADIF,
- upload Wavelog,
- download Wavelog,
- operazioni relative ai contest.

### Modificare QSO

1. Seleziona una riga con **↑/↓**.
2. Premi **Enter** o **e**.
3. Modifica il QSO.
4. Salva con **Ctrl+S**.

Le modifiche appaiono subito in Recent QSOs.

### Import ed export ADIF

CQOps supporta import ed export ADIF 3.1.7.

| Azione | Scorciatoia |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

L'import valida i record, salta i duplicati e mostra un riepilogo. I QSO importati vengono marcati per upload Wavelog quando la sincronizzazione Wavelog è configurata.

L'export può includere tutti i QSO o i QSO filtrati per contest. `CONTEST_ID` viene conservato.

### Gestione dei modi digitali

La gestione di mode e submode segue ADIF 3.1.7 come descritto in questo manuale:

- FT8 viene esportato come mode autonomo.
- FT4 e FT2 vengono esportati come MFSK con il submode appropriato.
- I record legacy MFSK + FT8 importati vengono normalizzati a FT8 autonomo.

Il QSO form ha campi separati **Mode** e **Submode**. Entrambi possono essere cambiati con **PgUp/PgDn**.

---

## Contest

I contest aggiungono campi exchange e gestione del seriale al QSO form.

Crea o configura un contest nel Logbook Editor con **Ins**.

La configurazione contest include:

- nome contest,
- data,
- ADIF contest ID,
- template exchange.

### Marker dei template

| Marker | Sostituito con |
|---|---|
| `@rst` | RST inviato o ricevuto |
| `@serial` | Numero seriale auto-incrementale |
| `@call` | Il tuo nominativo |
| `@grid` | Il tuo grid locator |
| `@name` | Nome operatore dal profilo operatore |

Premi **Ctrl+C** per cambiare il contest attivo.

Quando un contest è attivo:

- il QSO form mostra campi exchange,
- i seriali aumentano automaticamente,
- Recent QSOs può filtrare i QSO contest,
- l'export ADIF conserva `CONTEST_ID`.

---

## Favorites, riferimenti e band plan

### Favorites

Favorites memorizza preset di frequenza, mode e band in 10 slot.

| Scorciatoia | Azione |
|---|---|
| Alt+0–9 | Richiama un favorite |
| Alt+Shift+0–9 | Salva frequency, mode e band correnti come favorite |

Favorites è salvato nella configurazione ed è condiviso tra logbook.

Esempio:

1. Inserisci `145.55`.
2. Imposta mode su `FM`.
3. Imposta band su `2m`.
4. Premi **Alt+Shift+1**.
5. Più tardi, premi **Alt+1** per richiamare il preset.

### REF Lookup

Apri REF Lookup con **F6**.

Cerca:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Puoi cercare per prefisso, nome o designatore di riferimento. I riferimenti selezionati possono compilare il QSO form.

### Band Plan Browser

Apri Band Plan Browser con **F7**.

Fornisce accesso rapido a:

- bande radioamatoriali,
- range VHF/UHF,
- CB,
- PMR446,
- preset broadcast.

Una frequenza selezionata può essere usata per sintonizzare il rig attivo. I dati band plan possono anche essere esportati come Markdown.

---

## Integrazioni

Tutte le integrazioni sono opzionali. Il logging locale funziona senza di esse.

### QRZ.com

QRZ.com lookup richiede internet e un abbonamento QRZ XML.

Nel QSO form, premi **Ins** per compilare campi callbook come:

- name,
- QTH,
- grid,
- country,
- CQ/ITU zones,
- DXCC,
- continent.

La Partner view su **F2** può mostrare la foto dell'operatore quando disponibile.

> ⚠️ **Sperimentale.** La visualizzazione delle foto può usare il protocollo
> grafico Kitty e richiede un terminale compatibile: Kitty, Ghostty o WezTerm.
> Attiva in **F9 → General → Kitty Graphics**. I terminali standard e le
> sessioni SSH senza inoltro grafico mostreranno invece un glifo.

### Wavelog

L'integrazione Wavelog supporta:

- upload,
- download incrementale,
- worked/confirmed lookup.

Wavelog è configurato per logbook attivo con:

- URL,
- API key,
- station profile ID.

CQOps salva sempre i QSO prima in locale. Un errore di upload Wavelog non elimina i dati locali.

### flrig

L'integrazione flrig usa XML-RPC su HTTP.

Endpoint predefinito:

```text
localhost:12345
```

CQOps può leggere:

- frequency,
- mode,
- power.

In split, VFO A è mappato a Frequency e VFO B a Freq RX.

### Hamlib / rigctld

Il controllo rig Hamlib usa il daemon TCP `rigctld`.

A seconda di radio e backend, CQOps può interrogare:

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps gestisce dove possibile la mancanza del supporto ai nomi VFO.

### Hamlib Rotor / rotctld

> ⚠️ **Sperimentale.** Il controllo rotore è sperimentale. Verifica sempre i
> limiti fisici della tua antenna prima di operare. Sii pronto a fermare il
> movimento immediatamente con **Alt+/** . Usa con cautela — una configurazione
> errata può danneggiare il rotore o l'antenna.

Il controllo rotore usa Hamlib `rotctld`.

CQOps supporta:

- azimuth,
- elevation,
- stop commands.

| Scorciatoia | Azione |
|---|---|
| Alt+, | Regola azimuth −5° |
| Alt+. | Regola azimuth +5° |
| Alt+; | Regola elevation +5° |
| Alt+' | Regola elevation −5° |
| Alt+\ | Punta il rotore al path bearing calcolato |
| Alt+/ | Ferma il rotore |

### WSJT-X

L'integrazione WSJT-X usa messaggi UDP da WSJT-X. CQOps analizza messaggi ADIF e può registrare automaticamente QSO completati.

L'etichetta rig diventa del colore di accento mentre WSJT-X trasmette. Se l'operatore riportato da WSJT-X non coincide con l'operatore attivo, CQOps mostra un avviso.

### GPS

CQOps può leggere la posizione da un ricevitore GPS e usarla come grid
locator della stazione — ideale per operazioni portatili, mobili o in campo.

Sono supportati due backend:

- **Serial** — si connette direttamente a un ricevitore GPS tramite una
  porta seriale (USB-seriale, porta COM integrata o `/dev/ttyUSB0`).
- **GPSD** — si connette a un server [gpsd](https://gpsd.io/) via TCP
  (predefinito `127.0.0.1:2947`). Utile quando il GPS è condiviso con
  altre applicazioni o accessibile via rete.

L'indicatore GPS nella barra di stato mostra:

| Colore | Significato |
|--------|---------|
| Rosso `GPS` | Disconnesso / errore |
| Giallo `GPS` | Connesso, nessun fix ancora |
| Bianco `GPS` | Fix acquisito, posizione bloccata |

Quando viene acquisito un fix, il grid locator della stazione viene
sostituito con la posizione GPS e contrassegnato con `(GPS)` nella riga
di stato:

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Abilita **Grid from GPS** nelle impostazioni Station & Logbook per
utilizzare il grid GPS per il logging QSO, i beacon APRS, la mappa del
dashboard e i calcoli di distanza.

**Precisione del grid** — configurabile nel menu Integrazioni (10, 8 o 6
caratteri). Predefinito 10 caratteri (~25 m di precisione).

### DX Cluster

L'integrazione DX Cluster usa telnet e richiede internet.

Server predefinito:

```text
dxspots.com:7300
```

I filtri includono:

- band,
- continent,
- mode,
- age/time.

| Tasto | Azione |
|---|---|
| Enter | Compila QSO form, sintonizza il rig e torna a QSO |
| Space | Sintonizza il rig e resta su DX Cluster |
| Backspace | Cancella filtri |

### PSK Reporter

PSK Reporter richiede internet.

Fornisce:

- spot di propagazione,
- filtri band/time/mode,
- ASCII world map su **F5**.

### APRS

CQOps supporta tre tipi di servizio APRS — scegli quello adatto alla
configurazione della tua stazione:

| Servizio | Connessione | Internet richiesto |
|---|---|---|
| **APRS-IS** | TCP a un server APRS-IS | Sì |
| **KISS** | Porta seriale a un TNC KISS hardware | No |
| **KISS Server** | TCP a un server TNC KISS (es. Dire Wolf) | No (rete locale) |

Seleziona il tipo di servizio nel menu Integrations:

```text
F9 → Integrations → APRS → Service (Spazio per cambiare)
```

Tutti e tre i servizi supportano la ricezione di rapporti di posizione
APRS da stazioni vicine e la loro visualizzazione sulla mappa locale
CQOps Live con:

- simboli APRS standard,
- popup callsign,
- auto-fit view,
- range circle configurabile.

Tutti i servizi supportano anche il **beaconing periodico di posizione**.
CQOps trasmette il tuo grid locator all'intervallo configurato. Quando
il GPS è attivo e **Grid from GPS** è abilitato, il beacon usa
automaticamente la posizione derivata dal GPS — ideale per operazioni
portatili e mobili.

#### APRS-IS

Si connette alla rete globale APRS-IS tramite internet. Richiede:

- un nominativo radioamatoriale valido,
- un passcode APRS-IS (generato dal tuo nominativo),
- una connessione internet.

Server predefinito:

```text
euro.aprs2.net:14580
```

APRS-IS è configurato globalmente in **F9 → Integrations → APRS**.
Nominativo, SSID, simbolo, commento, intervallo beacon e filtro di
portata per logbook in **F9 → Logbooks → [logbook attivo] → APRS**.

#### KISS (seriale)

> ⚠️ **Sperimentale.** Il supporto KISS TNC è sperimentale. Testa
> accuratamente prima di farci affidamento per l'operatività.

Si connette direttamente a un TNC KISS hardware tramite porta seriale.
Nessuna connessione internet richiesta — i frame APRS sono inviati e
ricevuti attraverso la tua radio.

Configura la porta seriale, baud rate, data bits, parità, stop bits e
DTR/RTS nel menu Integrations:

```text
F9 → Integrations → APRS → Service: KISS
```

Quando KISS è selezionato, i campi seriali (Port, Baud, Data bits,
Parity, Stop bits, DTR, RTS) diventano visibili.

Il pulsante **Test** apre la porta seriale per verificare che il TNC
sia raggiungibile.

#### KISS Server (TCP)

> ⚠️ **Sperimentale.** Il supporto KISS Server è sperimentale. Testa
> accuratamente prima di farci affidamento per l'operatività.

Si connette a un TNC KISS accessibile via TCP — ad esempio, un'istanza
[Dire Wolf](https://github.com/wb2osz/direwolf) in esecuzione sulla
stessa macchina o sulla rete locale. Nessuna connessione internet
richiesta.

Inserisci host e porta nel menu Integrations:

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Predefinito: `127.0.0.1:8001`

#### Beaconing

I beacon vengono inviati all'intervallo configurato per logbook.
L'intervallo minimo è di 1 minuto. Il beacon include:

- nominativo di stazione con SSID,
- grid locator (basato sul GPS quando disponibile),
- simbolo APRS,
- commento opzionale.

Quando il **GPS** è attivo e **Grid from GPS** è abilitato nelle
impostazioni Station, il beacon usa automaticamente il grid locator
derivato dal GPS — nessun aggiornamento manuale del grid necessario
durante gli spostamenti.

Intervallo beacon e altre impostazioni per logbook:

```text
F9 → Logbooks → [logbook attivo] → APRS
```

#### Ricezione

I rapporti di posizione APRS ricevuti vengono memorizzati nella cache
locale e visualizzati sulla mappa del dashboard CQOps Live. Le stazioni
sono mostrate con i loro simboli APRS e possono essere cliccate per i
dettagli. La visualizzazione si auto-adatta per mostrare tutte le
stazioni visibili entro la portata configurata.

La ricezione APRS è indipendente dalla trasmissione beacon — puoi
ricevere senza inviare un beacon, e viceversa. Attiva semplicemente
APRS nel menu Integrations e imposta il tipo di servizio.

### Solar Data

Solar data proviene da hamqsl.com e include:

- SFI,
- sunspot number,
- A/K indices,
- band-by-band conditions.

Gli aggiornamenti live richiedono internet. I dati in cache restano disponibili offline dopo un fetch riuscito.

---

## CQOps Live Dashboard

CQOps Live è un dashboard integrato nel browser per attività di stazione in tempo reale.

È utile per:

- display pubblici di field day,
- schermate di stazione di club,
- monitoraggio contest,
- osservare la stazione da un'altra stanza,
- stand per eventi o fiere.

### Abilitare il dashboard

1. Premi **F9**.
2. Apri **Integrations**.
3. Vai a **HTTP Server**.
4. Abilita **HTTP server**.
5. Opzionalmente imposta address e port.
6. Premi **Ctrl+S** per salvare.
7. Apri il dashboard in un browser.

Impostazioni predefinite:

| Impostazione | Predefinito |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

Il server parte immediatamente dopo il salvataggio.

### Modalità di visualizzazione

CQOps Live ha due modalità.

#### Overview mode

Mostrata quando non c'è un callsign attivo in lavorazione.

Mostra:

- live Leaflet map,
- marker QSO di oggi,
- great-circle paths,
- tabella recent QSOs,
- informazioni stazione,
- statistiche,
- rate tracking a 5 minuti, 15 minuti e 1 ora,
- top operators,
- QSO più lunghi per distanza.

#### Active / Now Working mode

Mostrata quando si sta lavorando un callsign.

Mostra:

- callsign grande,
- submode indicator,
- foto QRZ se disponibile,
- badge band e mode,
- indicatori DUPE / NEW CALL / NEW DXCC,
- distance e bearing,
- percorso tratteggiato evidenziato sulla mappa dal grid stazione al grid partner.

### Info box

L'info box sopra la local map alterna moduli ogni 5 secondi:

- band conditions,
- solar activity,
- geomagnetic field,
- ultimo spot DX Cluster,
- conteggi report PSK Reporter per banda.

Band conditions viene sempre renderizzato a tutta larghezza.

### Weather row

La weather row mostra condizioni Open-Meteo attuali per il grid locator della stazione:

- temperature,
- wind,
- humidity,
- icon.

I dati meteo vengono recuperati lato browser e degradano correttamente offline.

### Local map

La local map a destra può mostrare:

- stazioni APRS,
- simboli APRS standard,
- range circle,
- popup callsign,
- day/night terminator overlay opzionale,
- RainViewer weather radar overlay opzionale.

### Aggiornamenti in tempo reale e performance

CQOps Live si aggiorna tramite Server-Sent Events (SSE). Non serve refresh della pagina.

Il dashboard è progettato per hardware poco potente:

- il browser renderizza la mappa,
- il browser calcola distanze,
- il browser calcola statistiche,
- CQOps invia aggiornamenti JSON leggeri,
- quando HTTP server è disabilitato, nessuna porta viene aperta e nessuna goroutine del dashboard gira.

### Personalizzazione dashboard

Nel form di integrazione HTTP Server puoi configurare:

| Campo | Descrizione |
|---|---|
| Header 1 | Titolo principale nel page header e hero area. Valore predefinito: “CQOps Live”. |
| Header 2 | Sottotitolo sotto il titolo. Valore predefinito: “Fast, portable ham radio logger”. |
| Logo URL | URL pubblico di immagine mostrata in alto a sinistra. Valore predefinito: logo CQOps. |
| Event Start | Data nel formato `YYYY-MM-DD`. Filtra statistiche e liste QSO da quella data. |

---

## Configurazione

Apri la configurazione con **F9**.

### File di configurazione

| Piattaforma | Percorso config |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Le credenziali sensibili sono salvate separatamente in `secrets.enc` nella stessa directory di configurazione.

I secrets sono cifrati con una chiave legata alla macchina. Spostando la configurazione su un'altra macchina, le credenziali devono essere reinserite.

### Menu di configurazione

| Menu | Configura |
|---|---|
| Station | Callsign, grid, CQ/ITU zone, IARU region, references |
| Rig | Rig presets, model, antenna, power, backend, rotor, WSJT-X |
| Wavelog | URL, API key, station profile ID |
| QRZ | Username e password |
| DX Cluster | Host, port, login |
| Operators | Profili operatore |
| Logbooks | Impostazioni station, Wavelog, contest, operator e APRS per logbook |
| Integrations | Tipo di servizio APRS (APRS-IS, KISS, KISS Server), GPS, server HTTP, DXC, QRZ |
| Notifications | QSO saved alerts, Wavelog status, dupe beep, error sounds |
| General | Timezone, distance units, map, debug mode |

### Multi-logbook

Usa più logbook per casa, portable, contest e club.

**Ctrl+L** cambia il logbook attivo.

Ogni logbook mantiene i propri:

- station details,
- Wavelog settings,
- contest settings,
- operator settings.

### Multi-operator

I profili operatore contengono:

- nominativo operatore,
- nome operatore.

**Ctrl+O** cambia l'operatore attivo.

L'operatore attivo è salvato nel campo ADIF `OPERATOR` e segue gli upload Wavelog.

### Multi-rig

I preset di rig memorizzano:

- backend,
- model,
- antenna,
- power,
- rotor settings,
- WSJT-X settings.

**Ctrl+R** cambia il rig attivo.

### Secrets cifrati

Da v0.8.7, le credenziali sono salvate cifrate.

| Elemento | Valore |
|---|---|
| Secrets file | `secrets.enc` |
| Location | Stessa directory di `config.yaml` |
| Unix permissions | `0600` dove supportato |
| Encryption | AES-256-GCM con chiave legata alla macchina |
| Protected data | QRZ password, DX Cluster login, Wavelog API keys |

I plaintext secrets delle vecchie configurazioni migrano al primo avvio.

Se `secrets.enc` è corrotto, CQOps parte con un avviso e chiede di reinserire le credenziali.

---

## Scorciatoie da tastiera

### Globali

| Tasto | Azione |
|---|---|
| F1 | QSO form e Recent QSOs |
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
| Ctrl+L | Cambia logbook attivo |
| Ctrl+R | Cambia rig attivo |
| Ctrl+C | Cambia contest attivo |
| Ctrl+O | Cambia operatore attivo |
| Esc | Torna alla schermata precedente |

### QSO form

| Tasto | Azione |
|---|---|
| Tab | Campo successivo |
| Shift+Tab | Campo precedente |
| ↑ / ↓ | Muovi nella colonna |
| Enter | Salva QSO, con conferma duplicato se necessaria |
| Ctrl+S | Salva QSO da qualsiasi campo |
| Del | Cancella tutti i campi del form |
| Ins | Lookup: QRZ, Wavelog, DXCC e duplicate check |
| PgUp / PgDn | Cambia band, mode o submode |
| Ctrl+D | Apri spot dialog |
| Ctrl+T | Commuta Keep Comment |
| Alt+, | Regola azimuth rotore −5° |
| Alt+. | Regola azimuth rotore +5° |
| Alt+; | Regola elevation rotore +5° |
| Alt+' | Regola elevation rotore −5° |
| Alt+\ | Punta rotore al bearing dal tuo grid al grid partner |
| Alt+/ | Ferma rotore |
| Alt+0–9 | Richiama favorite |
| Alt+Shift+0–9 | Salva frequency, mode e band correnti come favorite |

### Logbook Editor

| Tasto | Azione |
|---|---|
| ↑ / ↓ | Naviga righe |
| PgUp / PgDn | Pagina precedente o successiva |
| Home / End | Prima o ultima riga |
| Enter / e | Modifica QSO selezionato |
| Delete | Elimina QSO selezionato |
| p | Purge di tutti i QSO |
| Ctrl+C | Cambia filtro contest |
| Ctrl+E | Export ADIF |
| Ctrl+I / Tab | Import ADIF |
| w | Carica QSO non inviati su Wavelog |
| Ctrl+W | Scarica contatti da Wavelog |
| Esc / F6 | Chiudi editor e torna al QSO form |

### DX Cluster

| Tasto | Azione |
|---|---|
| ↑ / ↓ | Naviga spot |
| Enter | Compila QSO form, sintonizza rig e torna a QSO |
| Space | Sintonizza rig sullo spot selezionato e resta su DX Cluster |
| Home | Avanza filtro band |
| End | Indietro filtro band |
| `\` | Cambia filtro continent |
| Ins | Avanza filtro mode |
| Del | Indietro filtro mode |
| PgUp | Avanza filtro time |
| PgDn | Indietro filtro time |
| Backspace | Cancella tutti i filtri |
| Esc / F4 | Torna al QSO form |

### Partner view

| Tasto | Azione |
|---|---|
| F2 | Cicla Partner view → Photo → Back |
| Esc / F1 | Torna al QSO form |

---

## Risoluzione dei problemi

### CQOps non parte

Controlla:

- il terminale è almeno 80×24,
- gli utenti Windows usano Windows Terminal,
- l'avvio di rete non blocca, provando:

  ```bash
  cqops --offline
  ```

Controlla i log:

| Piattaforma | Percorso log |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### Il rig non si collega

Per flrig:

- verifica che flrig sia in esecuzione,
- verifica la porta nel preset rig attivo,
- la porta predefinita è `12345`.

Per Hamlib:

- verifica che `rigctld` sia in esecuzione,
- verifica host e porta,
- controlla che radio/backend supporti i dati richiesti.

Le status label aiutano la diagnosi:

| Colore | Significato |
|---|---|
| Bianco/predefinito | Connesso |
| Giallo | Disabilitato o in connessione |
| Rosso | Fallito |

I reconnect toasts possono essere soppressi. CQOps può riprovare in silenzio.

### WSJT-X non registra automaticamente

Controlla:

- WSJT-X **Settings → Reporting → UDP Server**,
- host e porta UDP corrispondono al preset rig attivo in CQOps,
- usi WSJT-X 2.6 o più recente,
- la status label WSJT è attiva,
- il logbook attivo è corretto,
- l'operatore attivo è corretto.

### Upload Wavelog fallisce

Controlla:

- Wavelog URL,
- API key,
- station profile ID,
- status label **WL**.

Gli errori di upload sono mostrati come toasts. I QSO restano salvati localmente anche se l'upload fallisce. Errori di singoli QSO non bloccano il resto del batch.

### Problemi con config file

Config file:

| Piattaforma | Percorso |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets file:

```text
secrets.enc
```

Il secrets file è nella stessa directory di `config.yaml`.

Se la configurazione è corrotta, spostala o cancellala e riavvia CQOps. Il setup wizard creerà una nuova configurazione.

Il campo `last_fetched_id` appare solo dopo un download Wavelog riuscito.

### Problemi di performance

Prova a:

- disabilitare map rendering in General settings,
- disabilitare il Solar panel se non serve,
- evitare schermate pesanti di rete come DX Cluster e PSK Reporter quando sei offline,
- usare `cqops --offline` quando la rete è inaffidabile.

---

## Segnalazione bug

Prima di segnalare un bug:

1. Abilita **Debug mode** in **F9 → General → Debug**, oppure imposta:

   ```yaml
   debug: true
   ```

   in `config.yaml`.

2. Riproduci il problema.
3. Allega il log rilevante.

Segnala problemi su GitHub:

<https://github.com/szporwolik/cqops/issues>

Includi:

- versione CQOps da `cqops --version`,
- sistema operativo,
- terminal emulator,
- passi per riprodurre,
- debug log rilevante.
