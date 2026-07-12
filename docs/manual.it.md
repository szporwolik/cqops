---
title: Manuale utente di CQOps
description: Guida all’installazione, alla configurazione e all’uso di CQOps — un logger radioamatoriale rapido e orientato al terminale
---

# Manuale utente di CQOps

CQOps è un logger radioamatoriale rapido e orientato al terminale, pensato per gli operatori che desiderano registrare i collegamenti in modo affidabile tramite tastiera e con un basso carico di sistema. È progettato per l’uso in stazione, le attività portatili, le stazioni di club, i field day e i sistemi della classe Raspberry Pi o i portatili meno recenti.

CQOps salva sempre i QSO prima in locale. Le integrazioni basate su Internet sono facoltative.

## Contenuti

1. [Che cos’è CQOps](#what-cqops-is)
2. [Download e installazione](#download-and-installation)
3. [Primo avvio](#first-launch)
4. [Registrare il primo QSO](#log-your-first-qso)
5. [Schermata principale](#main-screen)
6. [Flussi di lavoro comuni](#common-workflows)
7. [Registrazione dei QSO](#qso-logging)
8. [Editor del log e ADIF](#logbook-editor-and-adif)
9. [Contest](#contests)
    - [Configurare un contest](#setting-up-a-contest)
    - [Barra di stato inferiore](#bottom-status-bar)
    - [Pannello delle statistiche del contest](#contest-statistics-panel)
    - [Esportazione ADIF del contest](#contest-adif-export)
    - [Comportamento della modalità contest](#contest-mode-behavior)
10. [Preferiti, referenze e band plan](#favorites-references-and-band-plans)
11. [Integrazioni](#integrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Configurazione](#configuration)
14. [Scorciatoie da tastiera](#keyboard-shortcuts)
15. [Risoluzione dei problemi](#troubleshooting)
16. [Segnalazione dei bug](#reporting-bugs)

---

<a id="what-cqops-is"></a>
## Che cos’è CQOps

CQOps è basato sull’inserimento rapido dei QSO, sul salvataggio locale dei dati e su un utilizzo pratico sul campo.

### Idee principali

- **Uso orientato al terminale** — ottimizzato per la tastiera.
- **Registrazione offline-first** — il salvataggio locale dei QSO funziona senza accesso a Internet.
- **Basso utilizzo di risorse** — adatto a sistemi della classe Raspberry Pi, portatili meno recenti e PC di stazione condivisi.
- **Design portatile** — distribuito come singolo binario Go.
- **Più logbook** — utile per log personali, portatili, contest e di club.
- **Più operatori** — utile per flussi hot-seat e stazioni di club condivise.
- **Più apparati** — ogni preset dell’apparato può conservare impostazioni proprie per backend e WSJT-X.
- **Integrazioni facoltative** — QRZ.com, Wavelog, DX Cluster, PSK Reporter, GPS, APRS, controllo dell’apparato, controllo del rotore, dati solari e CQOps Live dashboard nel browser.

La registrazione locale non richiede Internet. Le funzioni di rete vengono ignorate in modalità `--offline`.

### A chi è destinato CQOps

CQOps è particolarmente indicato per:

- operatori portatili,
- attivatori SOTA e POTA,
- stazioni di club,
- team field day,
- operatori che preferiscono lavorare da terminale,
- stazioni che richiedono un rapido passaggio tra operatori, logbook o apparati.

CQOps non intende sostituire ogni funzione di un logger desktop completo o di una piattaforma web per i log. Si concentra sulla registrazione rapida da terminale, sull’attività sul campo, sull’uso offline e sui flussi di lavoro delle stazioni condivise.

### Uso nei club e nelle stazioni condivise

CQOps è stato sviluppato pensando agli ambienti dei club radioamatoriali. L’operatore attivo è sempre visibile nella barra di stato: **basta uno sguardo** per sapere chi sta utilizzando la stazione. Il cambio operatore richiede una sola combinazione (`Ctrl+O`) e ha effetto immediato; l’indicativo e il nome dell’operatore vengono scritti in ogni QSO successivo. Nessun logout, nessuna richiesta di password, nessuna interruzione.

Logbook, preset degli apparati e contest si scorrono allo stesso modo: `Ctrl+L`, `Ctrl+R`, `Ctrl+C`. Una stazione di club con operatori a rotazione, più apparati e diversi contest attivi può cambiare contesto in meno di un secondo senza usare il mouse.

Durante field day ed eventi pubblici, **CQOps Live dashboard** può proiettare su un grande schermo una mappa in tempo reale, il flusso dei QSO e le statistiche. Visitatori e soci del club possono seguire l’attività senza affollare il terminale dell’operatore. È sufficiente abilitare l’integrazione **HTTP Server** e accedere da qualsiasi dispositivo dotato di browser web.

---

<a id="download-and-installation"></a>
## Download e installazione

Visualizza tutte le release:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Pacchetto | Link | Note |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Consigliato per la maggior parte degli utenti. Aggiunge CQOps al menu Start e al PATH. |
| ZIP portatile | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Estrarre ed eseguire senza installazione. |

### Linux — Debian / Ubuntu

| Architettura | Link | Utilizzo |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La maggior parte dei PC Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Sistemi ARM a 64 bit |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS a 32 bit |

Installa il pacchetto scaricato:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Tarball portatile

| Architettura | Link | Utilizzo |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La maggior parte dei PC Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Sistemi ARM a 64 bit |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS a 32 bit |

### macOS

| Architettura | Link | Utilizzo |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Mac M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Mac Intel |

Installazione manuale:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Compilazione dai sorgenti

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

La compilazione dai sorgenti richiede Go 1.26 o successivo.

### Requisiti del terminale

| Requisito | Valore |
|---|---|
| Dimensione minima del terminale | 80×24 caratteri |
| Dimensione consigliata | 80×43 caratteri o superiore |
| Terminale Windows consigliato | Windows Terminal |
| Terminale con Kitty graphics | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) o [WezTerm](https://wezfurlong.org/wezterm/) |

### Comandi di base

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

<a id="first-launch"></a>
## Primo avvio

Al primo avvio, CQOps apre la procedura guidata di configurazione. Per la registrazione locale sono necessarie solo le informazioni essenziali della stazione. Le integrazioni di rete possono essere saltate e configurate in seguito.

### Pagine della procedura guidata

| Page | Cosa configura |
|---|---|
| Station & Logbook | Logbook iniziale, indicativo di stazione, operatore, grid locator, referenze e zone facoltative, Wavelog URL/API/station profile ID |
| Rig | Preset dell’apparato, modello, antenna, potenza, backend, rotore facoltativo e impostazioni UDP WSJT-X facoltative |
| Integrations | Impostazioni di ricerca QRZ.com |
| General | Fuso orario IANA |
| Summary | Revisione e salvataggio |

I backend supportati sono:

- None,
- flrig,
- Hamlib `rigctld`.

### Navigazione nella procedura guidata

| Key | Action |
|---|---|
| Ctrl+S | Convalidare e continuare; in **Summary**, salvare e avviare CQOps |
| Esc | Tornare indietro |
| F10 | Uscire |
| Tab / Shift+Tab | Spostarsi tra i campi |
| Space | Attivare o disattivare le caselle |

Le impostazioni della procedura guidata possono essere modificate in seguito con **F9**.

---

<a id="log-your-first-qso"></a>
## Registrare il primo QSO

1. Avvia CQOps:

   ```bash
   cqops
   ```

2. Completa la procedura guidata indicando almeno il tuo indicativo e il grid locator.

3. Apri **QSO form** con **F1**.

4. Inserisci l’indicativo del corrispondente. CQOps converte automaticamente gli indicativi in maiuscolo.

5. Compila i campi rimanenti. Se l’apparato attivo è connesso tramite flrig o Hamlib, CQOps può compilare automaticamente frequency, band, mode e submode.

6. Premi **Enter** per salvare.

7. Se compare l’avviso **DUPE!**, premi nuovamente **Enter** per salvare comunque oppure **Esc** per annullare.

Il QSO salvato appare immediatamente nella tabella **Recent QSOs**.

---

<a id="main-screen"></a>
## Schermata principale

CQOps utilizza un layout fisso nel terminale:

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

La **Status bar** mostra:

- la versione di CQOps,
- il logbook attivo,
- l’apparato attivo,
- l’indicativo della stazione,
- l’operatore attivo,
- le etichette di stato delle integrazioni,
- l’ora locale contrassegnata con `L`,
- l’ora UTC contrassegnata con `Z`.

Le etichette comuni includono **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** e **GPS**. L’etichetta **GPS** segue la stessa convenzione cromatica: rosso quando è disconnesso, giallo quando è connesso ma senza fix e bianco quando è stata acquisita la posizione.

| Colore | Significato |
|---|---|
| Bianco/predefinito | Connesso o attivo |
| Giallo | Disabilitato, in connessione o previsto offline |
| Rosso | Errore o disconnesso |
| Colore di accento + grassetto | WSJT-X sta trasmettendo |

### Tab principali

| Key | Tab | Screen |
|---|---|---|
| F1 | QSO | **QSO form** e **Recent QSOs** |
| F2 | QRZ | **Partner view**: dati del callbook, mappa, statistiche, foto |
| F4 | DXC | Spot e filtri di **DX Cluster** |
| F5 | HRD | Spot di **PSK Reporter** e mappa di propagazione |
| F6 | REF | Ricerca di referenze SOTA/POTA/WWFF/IOTA |
| F7 | BPL | **Band Plan Browser** |
| F8 | LOG | **Logbook Editor**, ADIF, sincronizzazione Wavelog |
| F9 | CFG | Menu di configurazione |

La **Help bar** mostra le scorciatoie relative alla schermata attiva. Premi **?** per aprire il **Help overlay** completo.

---

<a id="common-workflows"></a>
## Flussi di lavoro comuni

### Attività portable, SOTA o POTA

Prima di partire:

1. Avvia CQOps almeno una volta con accesso a Internet.
2. Consenti a CQOps di scaricare o aggiornare i dati in cache, come dati solari, dati REF e prefissi DXCC.
3. Verifica che il pannello **Solar** mostri i dati.
4. Verifica che la ricerca **REF** in **F6** restituisca risultati.

Sul campo:

1. Avvia CQOps in modalità offline:

   ```bash
   cqops --offline
   ```

2. Registra normalmente. I QSO vengono salvati in locale.
3. Quando torni online, apri **F8** e premi **w** per caricare su Wavelog i QSO non inviati.

### Stazione di club condivisa e registrazione hot-seat

1. Apri **F9 → Operators**.
2. Premi **Ins** per aggiungere i profili degli operatori.
3. In **QSO form**, premi **Ctrl+O** per cambiare l’operatore attivo.
4. Controlla l’operatore attivo nella barra di stato prima di salvare.
5. Usa **Retain** quando più operatori devono registrare collegamenti simili senza reinserire l’intero modulo.

L’operatore attivo viene salvato nel campo ADIF `OPERATOR`.

### Logbook personali e di club

1. Apri **F9 → Logbooks**.
2. Premi **Ins** per creare ogni logbook.
3. In **QSO form**, premi **Ctrl+L** per cambiare il logbook attivo.
4. Controlla il logbook attivo nella barra di stato prima di salvare.

Ogni logbook può conservare i propri dati della stazione, impostazioni Wavelog, impostazioni del contest e operatori.

### Più apparati

1. Apri **F9 → Rigs**.
2. Premi **Ins** per creare preset degli apparati.
3. Seleziona il backend: None, flrig o Hamlib.
4. In **QSO form**, premi **Ctrl+R** per cambiare l’apparato attivo.

Un preset può includere backend, modello, antenna, potenza, impostazioni del rotore e impostazioni UDP di WSJT-X.

### Attività digitale con WSJT-X

Quando l’integrazione UDP di WSJT-X è abilitata, CQOps può ricevere messaggi ADIF da WSJT-X e registrare automaticamente i QSO digitali completati.

I QSO registrati automaticamente:

- vengono salvati nel logbook attivo,
- appaiono immediatamente in **Recent QSOs**,
- ignorano i duplicati,
- ereditano il contest ID attivo,
- possono essere caricati automaticamente su Wavelog quando è configurato e raggiungibile.

Se l’operatore indicato da WSJT-X non corrisponde all’operatore attivo in CQOps, viene mostrato un avviso.

Prima di una lunga sessione digitale, controlla:

- il logbook attivo,
- l’operatore attivo,
- il contest attivo,
- l’etichetta di stato **WSJT**.

### Sincronizzazione Wavelog

CQOps salva sempre i QSO prima in locale. La sincronizzazione con Wavelog è facoltativa.

| Action | Where | Shortcut | Notes |
|---|---|---|---|
| Upload unsent QSOs | **Logbook Editor** | `w` | Caricamento in batch da 50 |
| Download from Wavelog | **Logbook Editor** | `Ctrl+W` | Download incrementale tramite `last_fetched_id` |

Lo stato del caricamento viene tracciato per ogni QSO:

- not sent,
- sent,
- error.

Se il caricamento fallisce, il QSO rimane nel logbook locale e può essere ritentato in seguito. Il purge di un logbook reimposta la fetch ID a `0`, consentendo un nuovo download completo.

---

<a id="qso-logging"></a>
## Registrazione dei QSO

**QSO form** è la schermata principale di registrazione. Aprila con **F1**.

CQOps può compilare i campi dalle seguenti fonti:

| Fonte | Campi |
|---|---|
| flrig / Hamlib | Frequency, Freq RX in split, Mode, Submode |
| QRZ.com | Name, QTH, Grid, Country, CQ zone, ITU zone, DXCC, Continent |
| Database REF | Referenze SOTA, POTA, WWFF e IOTA |
| Wavelog lookup | Stato worked/confirmed quando configurato |
| Dati DXCC/prefissi | Informazioni relative al prefisso e al Paese |

### Layout del modulo

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

I campi **Exch sent** e **Exch rcvd** compaiono solo quando è attivo un contest.

La riga inferiore contiene:

- **Comment**,
- **Keep** — conserva il campo **Comment** tra un QSO e l’altro,
- **Retain** — conserva l’intero modulo dopo il salvataggio.

I campi come **Band**, **Mode** e **Submode** possono essere scorsi con **PgUp/PgDn**.

### Percorso, puntamento e indicatori

Quando sono noti entrambi i grid locator, CQOps mostra distanza e azimut.

**QSO form** può inoltre mostrare indicatori come:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Salvataggio

| Key | Action |
|---|---|
| Enter | Salvare il QSO |
| Ctrl+S | Inviare uno spot DX dal modulo compilato |
| Esc | Annullare la conferma del duplicato |
| Enter nella conferma DUPE | Salvare comunque il duplicato |

---

<a id="logbook-editor-and-adif"></a>
## Editor del log e ADIF

Apri **Logbook Editor** con **F8**.

Può essere utilizzato per:

- rivedere i QSO,
- modificarli direttamente,
- eliminare QSO,
- importare ADIF,
- esportare ADIF,
- caricare su Wavelog,
- scaricare da Wavelog,
- eseguire operazioni relative ai contest.

### Modificare i QSO

1. Seleziona una riga con **↑/↓**.
2. Premi **Enter** o **e**.
3. Modifica il QSO.
4. Salva con **Ctrl+S**.

Le modifiche appaiono immediatamente in **Recent QSOs**.

### Importazione ed esportazione ADIF

CQOps supporta importazione ed esportazione ADIF 3.1.7.

| Action | Shortcut |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

L’importazione convalida i record, ignora i duplicati e mostra un riepilogo. I QSO importati vengono contrassegnati per il caricamento su Wavelog quando la sincronizzazione è configurata.

L’esportazione può includere tutti i QSO o quelli filtrati per contest. `CONTEST_ID` viene mantenuto.

### Gestione dei modi digitali

La gestione di mode e submode segue ADIF 3.1.7 come descritto in questo manuale:

- FT8 viene esportato come mode autonomo.
- FT4 e FT2 vengono esportati come MFSK con il submode appropriato.
- I record legacy MFSK + FT8 importati vengono normalizzati in FT8 autonomo.

**QSO form** dispone di campi separati **Mode** e **Submode**. Entrambi possono essere scorsi con **PgUp/PgDn**.

---

<a id="contests"></a>
## Contest

CQOps include un pannello leggero per la registrazione dei contest, pensato per una **partecipazione occasionale**. Non sostituisce logger specifici come N1MM, Win-Test o TR4W. Per attività serie multi-op, multi-radio o in categoria assisted, utilizza un logger dedicato. CQOps è adatto quando vuoi distribuire qualche punto, controllare il tuo rate per divertimento o registrare alcuni QSO di contest durante un’attivazione SOTA/POTA senza lasciare il logger abituale.

<a id="setting-up-a-contest"></a>
### Configurare un contest

Crea o configura un contest in **Logbook Editor** con **Ins**.

La configurazione include:

- nome del contest,
- data,
- ADIF contest ID,
- modelli di scambio.

#### Marker dei modelli

| Marker | Replaced with |
|---|---|
| `@rst` | RST inviato o ricevuto |
| `@serial` | Numero seriale incrementato automaticamente |
| `@call` | Il tuo indicativo |
| `@grid` | Il tuo grid locator |
| `@name` | Operator name dal profilo operatore |

Premi **Ctrl+C** per scorrere i contest attivi oppure selezionane uno dal menu **Contest** (**F7**). I campi di scambio compaiono automaticamente in **QSO form** e i numeri seriali vengono incrementati in automatico.

<a id="bottom-status-bar"></a>
### Barra di stato inferiore

Quando è attivo un contest, la barra inferiore mostra un riepilogo in tempo reale:

```
 IARU-HF · IARU HF   45 QSOs   Started 16:13   Last 14:04 ago   Next #45   On 2:41
```

| Campo | Significato |
|-------|---------|
| `IARU-HF` | ADIF ID del contest, cioè il suo identificatore leggibile dalla macchina |
| `· IARU HF` | Nome visualizzato del contest, mostrato quando differisce dall’ID |
| `45 QSOs` | Numero totale di QSO registrati in questa sessione di contest |
| `Started 16:13` | Ora del primo QSO del contest nella giornata |
| `Last 14:04 ago` | Tempo trascorso dall’ultimo QSO del contest |
| `Next #45` | Numero seriale che sarà inviato nel prossimo QSO |
| `On 2:41` | Tempo totale on-air: somma degli intervalli tra QSO inferiori a 30 minuti |

Il campo `Started` viene nascosto nei terminali con meno di 120 colonne. Il nome del contest e il tempo `On` vengono nascosti sotto le 100 colonne.

<a id="contest-statistics-panel"></a>
### Pannello delle statistiche del contest

Quando è attivo un contest e il terminale è sufficientemente largo, a destra di **QSO form** compare un pannello compatto con bordo giallo:

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

| Riga | Campo | Significato |
|-----|-------|---------|
| **Rate** | `2/h` | Rate degli ultimi **10 QSO**, velocità della breve sequenza |
| | `--/h` | Rate degli ultimi **100 QSO**; mostra `--` finché non vengono registrati 100 QSO |
| **Count** | `60m 0` | QSO registrati negli ultimi 60 minuti |
| | `hr 0` | QSO registrati nell’ora corrente a partire da `:00` |
| **Peak** | `1m120` | Miglior rate di 1 minuto: 120/h equivale a 2 QSO in quel minuto |
| | `10m 54` | Migliore finestra mobile di 10 minuti: media 54/h |
| | `60m 29` | Migliore finestra mobile di 60 minuti: media 29/h |
| **Avg** | `8/h` | Media della sessione: QSO totali divisi per le ore dal primo QSO |
| | `Sess 5:36` | Durata totale della sessione dal primo all’ultimo QSO, in H:MM o solo minuti |
| **Chart** | `max 1` | Il minuto più intenso ha avuto 1 QSO. Le barre mostrano i QSO al minuto |
| | `-60m…now` | Bordo sinistro = 60 minuti fa; bordo destro = adesso |

Il grafico utilizza caratteri a blocchi Unicode (`█`) scalati su quattro righe di barre verticali. I rate **Peak** omettono il suffisso `/h`, perché **Peak** implica già «all’ora». Le durate omettono i secondi, che sarebbero solo rumore con un aggiornamento ogni minuto.

<a id="contest-adif-export"></a>
### Esportazione ADIF del contest

Per inviare il log del contest, apri **Logbook Editor** (`Ctrl+E`) mentre il contest è attivo. Se viene applicato un filtro del contest, la finestra di esportazione ADIF consente di esportare **solo i QSO appartenenti al contest attivo**. Il risultato è un file ADIF 3.1.7 conforme allo standard, con campi di scambio, numeri seriali e ADIF ID del contest preservati, pronto per il robot dell’organizzatore o per il sistema di controllo dei log.

<a id="contest-mode-behavior"></a>
### Comportamento della modalità contest

Quando è attivo un contest:

- **QSO form** mostra i campi di scambio,
- i numeri seriali aumentano automaticamente,
- **Recent QSOs** può essere filtrato sui QSO del contest,
- l’esportazione ADIF conserva `CONTEST_ID`,
- **QSO form**, il pannello contest e il pannello **Solar** ricevono un bordo giallo per essere distinti visivamente,
- gli spot DXC vengono confrontati con tutti i QSO del contest, non solo quelli odierni, per segnalare i duplicati.

---

<a id="favorites-references-and-band-plans"></a>
## Preferiti, referenze e band plan

### Favorites

**Favorites** memorizza preset di frequency, mode e band in tre slot, sufficienti per le frequenze di chiamata più utilizzate. Le scorciatoie usano `Alt` per evitare conflitti con i normali comandi di modifica del terminale e funzionare in modo affidabile su terminali diversi.

| Shortcut | Action |
|---|---|
| Alt+Ins / Alt+Home / Alt+PgUp | Richiamare il favorite dallo slot 1, 2 o 3 |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Salvare frequency, mode e band correnti nello slot 1, 2 o 3 |

**Favorites** viene memorizzato nella configurazione ed è condiviso tra i logbook.

Esempio:

1. Inserisci `145.55`.
2. Imposta **Mode** su `FM`.
3. Imposta **Band** su `2m`.
4. Premi **Alt+Shift+Ins** per salvare il preset nello slot 1.
5. In seguito, premi **Alt+Ins** per richiamarlo.

### REF Lookup

Apri **REF Lookup** con **F6**.

Cerca in:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Puoi cercare per prefisso, nome o designatore della referenza. Le referenze selezionate possono compilare **QSO form**.

### Band Plan Browser

Apri **Band Plan Browser** con **F7**.

Fornisce un accesso rapido a:

- Amateur bands,
- VHF/UHF ranges,
- CB,
- PMR446,
- Broadcast presets,
- Portable — frequenze comuni per attività portatili e sul campo, incluse SOTA, POTA e canali di chiamata.

La frequenza selezionata può essere usata per sintonizzare l’apparato attivo. I dati del band plan possono inoltre essere esportati in Markdown.

---

<a id="integrations"></a>
## Integrazioni

Tutte le integrazioni sono facoltative. La registrazione locale funziona senza di esse.

### QRZ.com

La ricerca QRZ.com richiede accesso a Internet e un abbonamento QRZ XML.

In **QSO form**, premi **Ins** per compilare campi del callbook come:

- Name,
- QTH,
- Grid,
- Country,
- CQ/ITU zones,
- DXCC,
- Continent.

**Partner view** in **F2** può mostrare la foto dell’operatore quando disponibile.

> ⚠️ **Experimental.** La visualizzazione delle foto può utilizzare il
> protocollo Kitty terminal graphics e richiede un terminale compatibile:
> Kitty, Ghostty o WezTerm. Abilitala in **F9 → General → Kitty Graphics**.
> I terminali standard e le sessioni SSH senza inoltro grafico useranno
> un’immagine di glifi come alternativa.

### Wavelog

L’integrazione Wavelog supporta:

- caricamento,
- download incrementale,
- ricerca dello stato worked/confirmed.

Wavelog viene configurato per ogni logbook attivo con:

- URL,
- API key,
- station profile ID.

CQOps salva sempre i QSO prima in locale. Un errore di caricamento su Wavelog non elimina i dati locali.

### flrig

L’integrazione flrig utilizza XML-RPC su HTTP.

Endpoint predefinito:

```text
localhost:12345
```

CQOps può leggere:

- frequency,
- mode,
- power.

Nel funzionamento split, VFO A viene associato a **Frequency** e VFO B a **Freq RX**.

### Hamlib / rigctld

Il controllo dell’apparato tramite Hamlib utilizza il daemon TCP `rigctld`.

A seconda dell’apparato e del supporto del backend, CQOps può interrogare:

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps gestisce in modo sicuro l’assenza di supporto per i nomi VFO quando possibile.

### Hamlib Rotator / rotctld

> ⚠️ **Experimental.** Il controllo del rotore è sperimentale. Verifica sempre
> i limiti fisici dell’antenna prima dell’utilizzo. Preparati a fermare
> immediatamente il movimento con **Alt+/**. Usa questa funzione con cautela:
> una configurazione errata può danneggiare il rotore o l’antenna.

Il controllo del rotore utilizza Hamlib `rotctld`.

CQOps supporta:

- azimuth,
- elevation,
- stop commands.

| Shortcut | Action |
|---|---|
| Alt+, | Regolare azimuth di −5° |
| Alt+. | Regolare azimuth di +5° |
| Alt+; | Regolare elevation di +5° |
| Alt+' | Regolare elevation di −5° |
| Alt+\ | Puntare il rotore verso il rilevamento calcolato |
| Alt+/ | Arrestare il rotore |

### WSJT-X

L’integrazione WSJT-X utilizza messaggi UDP provenienti da WSJT-X. CQOps analizza i messaggi ADIF e può registrare automaticamente i QSO completati.

L’etichetta dell’apparato assume il colore di accento mentre WSJT-X sta trasmettendo. Se l’operatore indicato da WSJT-X non corrisponde all’operatore attivo, CQOps mostra un avviso.

### GPS

CQOps può leggere la posizione da un ricevitore GPS e utilizzarla come grid locator della stazione, soluzione ideale per attività portatili, mobili o sul campo.

Sono supportati due backend:

- **Serial** — collegamento diretto a un ricevitore GPS tramite porta seriale, ad esempio USB-to-serial, porta COM integrata o `/dev/ttyUSB0`.
- **GPSD** — collegamento a un server [gpsd](https://gpsd.io/) tramite TCP, per impostazione predefinita `127.0.0.1:2947`. Utile quando il GPS è condiviso con altre applicazioni o raggiunto tramite rete.

L’indicatore **GPS** nella barra di stato mostra:

| Colore | Significato |
|--------|---------|
| Rosso `GPS` | Disconnesso / errore |
| Giallo `GPS` | Connesso, senza fix |
| Bianco `GPS` | Fix acquisito, posizione stabilita |

Quando viene acquisito un fix, il grid locator della stazione viene sostituito con la posizione derivata dal GPS e contrassegnato con `(GPS)` nella riga di stato:

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Abilita **Grid from GPS** nelle impostazioni **Station & Logbook** per usare il grid GPS nella registrazione dei QSO, nei beacon APRS, nella mappa del dashboard e nei calcoli della distanza.

**Grid precision** si configura nel menu **Integration** con 10, 8 o 6 caratteri. Il valore predefinito è 10 caratteri, circa 25 m di precisione. Il grid viene sempre calcolato internamente alla massima precisione e troncato nel punto di utilizzo.

### DX Cluster

L’integrazione **DX Cluster** utilizza telnet e richiede l’accesso a Internet.

Server predefinito:

```text
dxspots.com:7300
```

I filtri includono:

- band,
- spotter continent,
- mode,
- age/time.

| Key | Action |
|---|---|
| Enter | Compilare **QSO form**, sintonizzare l’apparato e tornare a **QSO** |
| Space | Sintonizzare l’apparato e restare in **DX Cluster** |
| Backspace | Cancellare tutti i filtri |

Quando **DX Cluster** è connesso, **QSO form** ottiene due funzioni aggiuntive:

- **Send a spot** — con il modulo compilato, premi **Ctrl+S** per aprire la finestra dello spot e inviare uno spot DX al cluster.
- **Nearest spots** — quando è sintonizzata una frequenza, fino a tre spot vicini compaiono direttamente in **QSO form**, così puoi vedere l’attività sulla banda senza lasciare la schermata di registrazione. Premi **Ctrl+P** per compilare l’indicativo dallo spot più vicino.

### PSK Reporter

L’integrazione **PSK Reporter** richiede accesso a Internet. È uno strumento eccellente per verificare rapidamente la propagazione reale: chi sta ricevendo il tuo segnale o chi stai ricevendo su una determinata banda in quel momento.

Fornisce:

- spot di propagazione,
- filtri band/time/mode,
- mappa mondiale ASCII in **F5**.

### APRS

CQOps supporta tre tipi di servizio APRS. Scegli quello adatto alla configurazione della tua stazione:

| Service | Connection | Internet required |
|---|---|---|
| **APRS-IS** | TCP verso un server APRS-IS | Sì |
| **KISS** | Porta seriale verso un TNC KISS hardware | No |
| **KISS Server** | TCP verso un server TNC KISS, ad esempio Dire Wolf | No, è sufficiente la rete locale |

Seleziona il tipo di servizio nel menu **Integrations**:

```text
F9 → Integrations → APRS → Service (Space to cycle)
```

Tutti e tre i servizi supportano la ricezione dei rapporti di posizione APRS dalle stazioni vicine e la loro visualizzazione sulla mappa locale di **CQOps Live** con:

- simboli APRS standard,
- popup del callsign,
- adattamento automatico della vista,
- cerchio di portata configurabile.

Tutti supportano anche il **periodic position beaconing**. CQOps trasmette il grid locator della stazione all’intervallo configurato. Quando GPS è attivo e **Grid from GPS** è abilitato, il beacon usa automaticamente la posizione derivata dal GPS, ideale per attività portatili e mobili.

#### APRS-IS

Si collega alla rete globale APRS-IS tramite Internet. Richiede:

- un indicativo radioamatoriale valido,
- un APRS-IS passcode generato dall’indicativo,
- una connessione Internet.

Server predefinito:

```text
euro.aprs2.net:14580
```

APRS-IS viene configurato globalmente in **F9 → Integrations → APRS**. Callsign, SSID, symbol, comment, beacon interval e range filter per ogni logbook si impostano in **F9 → Logbooks → [active logbook] → APRS**.

#### KISS (serial)

Si collega direttamente a un TNC KISS hardware tramite porta seriale. Non è necessaria una connessione Internet: i frame APRS vengono inviati e ricevuti attraverso la radio.

Configura serial port, baud rate, data bits, parity, stop bits e DTR/RTS nel menu **Integrations**:

```text
F9 → Integrations → APRS → Service: KISS
```

Quando viene selezionato **KISS**, diventano visibili i campi seriali **Port**, **Baud**, **Data bits**, **Parity**, **Stop bits**, **DTR** e **RTS**.

Il pulsante **Test** apre la porta seriale per verificare che il TNC sia raggiungibile.

#### KISS Server (TCP)

Si collega a un TNC KISS raggiungibile tramite TCP, ad esempio un’istanza [Dire Wolf](https://github.com/wb2osz/direwolf) in esecuzione sullo stesso computer o nella rete locale. Non è necessaria una connessione Internet.

Inserisci host e porta nel menu **Integrations**:

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Valori predefiniti: `127.0.0.1:8001`

#### Beaconing

I beacon vengono inviati all’intervallo configurato per ciascun logbook. L’intervallo minimo è 1 minuto. Il beacon include:

- callsign della stazione con SSID,
- grid locator derivato dal GPS quando disponibile,
- symbol APRS,
- comment facoltativo.

Quando **GPS** è attivo e **Grid from GPS** è abilitato nelle impostazioni **Station**, il beacon usa automaticamente il grid locator derivato dal GPS. Non è necessario aggiornare manualmente il grid durante il movimento.

Beacon interval e le altre impostazioni per logbook si configurano in:

```text
F9 → Logbooks → [active logbook] → APRS
```

#### Receiving

I rapporti di posizione APRS ricevuti vengono memorizzati nella cache locale e visualizzati sulla mappa di **CQOps Live dashboard**. Le stazioni sono rappresentate con i rispettivi simboli APRS e possono essere selezionate per visualizzare i dettagli. La vista si adatta automaticamente per mostrare tutte le stazioni visibili entro la portata configurata.

La ricezione APRS è indipendente dalla trasmissione dei beacon: puoi ricevere senza trasmettere e viceversa. È sufficiente abilitare APRS in **Integrations** e impostare il tipo di servizio.

### Solar Data

I dati solari provengono da hamqsl.com e includono:

- SFI,
- sunspot number,
- A/K indices,
- condizioni per ciascuna banda.

Gli aggiornamenti in tempo reale richiedono Internet. I dati in cache restano disponibili offline dopo un download riuscito.

---

<a id="cqops-live-dashboard"></a>
## CQOps Live Dashboard

CQOps Live è un dashboard integrato nel browser per visualizzare in tempo reale l’attività della stazione.

È utile per:

- display pubblici durante i field day,
- schermi delle stazioni di club,
- monitoraggio dei contest,
- osservazione della stazione da un’altra stanza,
- stand durante eventi o fiere.

### Abilitare il dashboard

1. Premi **F9**.
2. Apri **Integrations**.
3. Vai a **HTTP Server**.
4. Abilita **HTTP server**.
5. Imposta facoltativamente address e port.
6. Premi **Ctrl+S** per salvare.
7. Apri il dashboard in un browser.

Impostazioni predefinite:

| Setting | Default |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

Il server si avvia immediatamente dopo il salvataggio.

> **Address binding:** Il valore predefinito `0.0.0.0` rende il dashboard accessibile da qualsiasi dispositivo nella rete locale. È utile per display field day, schermi di stazioni di club o per controllare la stazione da un’altra stanza. Imposta address su `127.0.0.1` per limitare l’accesso al computer locale.

### Modalità di visualizzazione

CQOps Live dispone di due modalità di visualizzazione.

#### Overview mode

Viene mostrata quando non si sta lavorando alcun callsign attivo.

Visualizza:

- **live maps** — marker dei QSO odierni con percorsi ortodromici dal grid della stazione a ciascun corrispondente e una mappa APRS locale con le stazioni vicine,
- tabella dei recent QSOs,
- informazioni sulla stazione,
- statistiche,
- tracciamento del rate su 5 minuti, 15 minuti e 1 ora,
- migliori operatori,
- QSO a maggiore distanza.

#### Active / Now Working mode

Viene mostrata quando si sta lavorando un callsign.

Visualizza:

- callsign grande,
- indicatore submode,
- foto QRZ quando disponibile,
- indicatori band e mode,
- indicatori **DUPE / NEW CALL / NEW DXCC**,
- distanza e puntamento,
- percorso tratteggiato evidenziato sulla mappa dal grid della stazione a quello del corrispondente.

### Info box

La **Info box** sopra la mappa locale cambia modulo ogni 5 secondi:

- condizioni delle bande,
- attività solare,
- campo geomagnetico,
- ultimo spot DX Cluster,
- quantità di rapporti PSK Reporter per banda.

### Weather row

La **Weather row** mostra le condizioni Open-Meteo correnti per il grid locator della stazione:

- temperatura,
- vento,
- umidità,
- icona.

I dati meteo vengono recuperati dal browser e vengono semplicemente omessi quando si è offline.

### Local map

La **local map** a destra è dedicata al **monitoraggio delle stazioni APRS vicine**. Può mostrare:

- stazioni APRS vicine con simboli standard,
- popup del callsign al passaggio o al clic,
- cerchio di portata configurabile,
- overlay facoltativo del terminatore giorno/notte,
- overlay facoltativo del radar meteo RainViewer.

### Aggiornamenti in tempo reale e prestazioni

CQOps Live si aggiorna tramite Server-Sent Events (SSE). Non è necessario ricaricare la pagina.

Il dashboard è progettato per hardware poco potente:

- il browser esegue il rendering delle mappe,
- il browser calcola le distanze,
- il browser calcola le statistiche,
- CQOps invia aggiornamenti JSON leggeri,
- quando **HTTP server** è disabilitato, non viene aperta alcuna porta e non vengono eseguite goroutine del dashboard.

### Personalizzazione del dashboard

Nel modulo di integrazione **HTTP Server** puoi configurare:

| Field | Description |
|---|---|
| Header 1 | Titolo principale nell’intestazione della pagina e nell’area hero. Usa «CQOps Live» come valore di riserva. |
| Header 2 | Sottotitolo sotto il titolo. Usa «Fast, portable ham radio logger» come valore di riserva. |
| Logo URL | URL pubblico dell’immagine mostrata in alto a sinistra. Usa il logo CQOps come valore di riserva. |
| Event Start | Data in formato `YYYY-MM-DD`. Filtra statistiche ed elenchi QSO a partire da quella data. |

---

<a id="configuration"></a>
## Configurazione

Apri la configurazione con **F9**.

### File di configurazione

| Piattaforma | Percorso della configurazione |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Le credenziali sensibili vengono salvate separatamente in `secrets.enc`, nella stessa directory della configurazione.

I secrets vengono cifrati con una chiave legata alla macchina. Quando la configurazione viene spostata su un altro computer, le credenziali devono essere inserite nuovamente.

### Menu di configurazione

Premi **F9** per aprire il menu principale, quindi seleziona:

| Menu | Configures |
|---|---|
| General | Units, timezone, partner map/picture, solar panel, sorgenti CTY.DAT/SCP/REF, Kitty Graphics, Debug mode |
| Logbooks | Station callsign, grid, references, CQ/ITU zones, IARU region, GPS grid; Wavelog per logbook (URL, API key, station profile); APRS per logbook (callsign, symbol, beacon, range) |
| Operators | Profili operator callsign e operator name per stazioni multi-operatore |
| Rigs | Preset degli apparati: model, antenna, power, backend (None/flrig/Hamlib), rotor, WSJT-X UDP |
| Contests | Profili contest: name, date, ADIF contest ID, exchange templates, starting serial number |
| Integration | DX Cluster (host, port, login), QRZ.com (username, password), HTTP Server del dashboard (address, port, branding), GPS service (serial/GPSD, grid precision) |
| Notifications | QSO saved alerts, Wavelog upload status, dupe beep, error sounds |

### Multi-logbook

Usa più logbook per home, portable, contest e club.

Premi **Ctrl+L** per scorrere i logbook attivi.

Ogni logbook conserva i propri:

- dati della stazione,
- impostazioni Wavelog,
- impostazioni del contest,
- impostazioni degli operatori.

### Multi-operator

I profili operatore contengono:

- operator callsign,
- operator name.

Premi **Ctrl+O** per scorrere gli operatori attivi.

L’operatore attivo viene salvato nel campo ADIF `OPERATOR` e incluso nei caricamenti Wavelog.

### Multi-rig

I preset degli apparati memorizzano:

- backend,
- model,
- antenna,
- power,
- impostazioni del rotore,
- impostazioni WSJT-X.

Premi **Ctrl+R** per scorrere gli apparati attivi.

### Secrets cifrati

Dalla versione v0.8.7 le credenziali vengono salvate in forma cifrata.

| Elemento | Valore |
|---|---|
| File dei secrets | `secrets.enc` |
| Posizione | Stessa directory di `config.yaml` |
| Permessi Unix | `0600` dove supportato |
| Cifratura | AES-256-GCM con una chiave legata alla macchina |
| Dati protetti | QRZ password, DX Cluster login, Wavelog API keys |

I secrets in chiaro delle configurazioni precedenti vengono migrati al primo avvio.

Se `secrets.enc` è danneggiato, CQOps si avvia con un avviso e richiede di inserire nuovamente le credenziali.

---

<a id="keyboard-shortcuts"></a>
## Scorciatoie da tastiera

### Global

| Key | Action |
|---|---|
| F1 | **QSO form** e **Recent QSOs** |
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
| Ins | Lookup: QRZ, Wavelog, DXCC, and duplicate check |
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
## Risoluzione dei problemi

### CQOps non si avvia

Controlla:

- che il terminale abbia almeno 80×24 caratteri,
- che gli utenti Windows utilizzino Windows Terminal,
- che l’avvio di rete non stia bloccando, provando:

  ```bash
  cqops --offline
  ```

Controlla i log:

| Piattaforma | Percorso dei log |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### L’apparato non si connette

Per flrig:

- verifica che flrig sia in esecuzione,
- verifica la porta nel preset attivo,
- la porta predefinita è `12345`.

Per Hamlib:

- verifica che `rigctld` sia in esecuzione,
- controlla host e port,
- verifica che l’apparato/backend supporti i dati richiesti.

Le etichette di stato aiutano a diagnosticare il problema:

| Colore | Significato |
|---|---|
| Bianco/predefinito | Connesso |
| Giallo | Disabilitato o in connessione |
| Rosso | Errore |

I toast di riconnessione possono essere nascosti. CQOps può riprovare silenziosamente.

### WSJT-X non registra automaticamente

Controlla:

- **WSJT-X Settings → Reporting → UDP Server**,
- che UDP host e port corrispondano al preset attivo in CQOps,
- che sia in uso WSJT-X 2.6 o successivo,
- che l’etichetta di stato **WSJT** sia attiva,
- che il logbook attivo sia corretto,
- che l’operatore attivo sia corretto.

### Il caricamento Wavelog non riesce

Controlla:

- Wavelog URL,
- API key,
- station profile ID,
- l’etichetta di stato **WL**.

Gli errori di caricamento vengono mostrati come toast. I QSO restano salvati in locale anche in caso di errore. Il fallimento di un singolo QSO non blocca il resto del batch.

### Problemi con il file di configurazione

File di configurazione:

| Piattaforma | Percorso |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

File dei secrets:

```text
secrets.enc
```

Il file dei secrets è nella stessa directory di `config.yaml`.

Se la configurazione è danneggiata, sposta o elimina il file e riavvia CQOps. La procedura guidata creerà una nuova configurazione.

Il campo `last_fetched_id` compare solo dopo un download Wavelog riuscito.

### Problemi di prestazioni

Prova a:

- disabilitare il rendering della mappa in **General**,
- disabilitare il pannello **Solar** se non necessario,
- evitare schermate con traffico di rete elevato, come **DX Cluster** e **PSK Reporter**, quando lavori offline,
- usare `cqops --offline` quando la rete non è affidabile.

---

<a id="reporting-bugs"></a>
## Segnalazione dei bug

Prima di segnalare un bug:

1. Abilita **Debug mode** in **F9 → General → Debug**, oppure imposta:

   ```yaml
   debug: true
   ```

   in `config.yaml`.

2. Riproduci il problema.
3. Allega il log pertinente.

Segnala i problemi su GitHub:

<https://github.com/szporwolik/cqops/issues>

Includi:

- versione di CQOps da `cqops --version`,
- sistema operativo,
- emulatore di terminale,
- passaggi per riprodurre il problema,
- debug log pertinente.
