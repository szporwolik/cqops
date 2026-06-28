---
title: Manuale Utente CQOps
description: Guida completa per installare, configurare e usare CQOps — un veloce logger radioamatoriale da terminale
---

> **Nota:** Questa traduzione è stata generata con un modello LLM. Le correzioni sono benvenute — inviale come Pull Request verso il branch `dev`.

# Manuale Utente CQOps

## Indice

1. [Informazioni su CQOps](#informazioni-su-cqops)
2. [Download e Installazione](#download-e-installazione)
3. [Primo Avvio — Configurazione Guidata](#primo-avvio--configurazione-guidata)
4. [Avvio Rapido: Registra il Tuo Primo QSO](#avvio-rapido-registra-il-tuo-primo-qso)
5. [Panoramica della Schermata Principale](#panoramica-della-schermata-principale)
6. [Flussi di Lavoro Comuni](#flussi-di-lavoro-comuni)
7. [Funzionalità Principali](#funzionalità-principali)
8. [Integrazioni](#integrazioni)
9. [Riferimento di Configurazione](#riferimento-di-configurazione)
10. [Scorciatoie da Tastiera](#scorciatoie-da-tastiera)
11. [Risoluzione dei Problemi](#risoluzione-dei-problemi)

---

## Informazioni su CQOps

CQOps è un veloce logger radioamatoriale da terminale, progettato per operatori che necessitano di velocità, affidabilità e basso carico di sistema — in shack, su una vetta, a un field day o in una stazione di club.

**Offline-first.** La registrazione locale dei QSO non richiede accesso a internet. I dati di riferimento, i dati solari e i prefissi DXCC memorizzati nella cache rimangono disponibili dopo essere stati scaricati una volta. Le integrazioni di rete come Wavelog, QRZ.com, DX Cluster e PSK Reporter richiedono connettività e vengono saltate in modalità `--offline`.

**Costruito per l'operatività in campo.** CQOps è pronto per il QRP, amichevole per SOTA/POTA e funziona comodamente su macchine di classe Raspberry Pi, vecchi laptop e sistemi senza ambiente desktop.

**Pronto per stazioni di club.** CQOps supporta più logbook, profili operatore e preset di rig. Cambia il logbook attivo, l'operatore attivo o il rig attivo con un solo tasto.

**Portatile per progettazione.** CQOps è un singolo binario scritto in Go. Non ha dipendenze CGO né servizi di sistema richiesti.

**Multi-piattaforma.** Windows, Linux e macOS sono supportati su amd64 e arm64.

### A Chi È Rivolto CQOps

- Operatori portatili che necessitano di registrazione veloce da tastiera su hardware a basso consumo.
- Attivatori SOTA e POTA che registrano offline e caricano dopo.
- Stazioni di club con più operatori che condividono la stessa stazione.
- Team di field day che usano macchine condivise o hardware di classe Raspberry Pi.
- Operatori che preferiscono un flusso di lavoro da terminale rispetto a una GUI desktop.

CQOps non intende sostituire logger desktop completi o piattaforme di logbook basate su web. Si concentra sulla registrazione veloce da terminale, l'operatività in campo, l'uso offline e i flussi di lavoro in stazioni condivise.

---

## Download e Installazione

> [Sfoglia tutte le release →](https://github.com/szporwolik/cqops/releases)

### Windows

| Pacchetto | Link | Note |
|---------|------|-------|
| **Installatore** | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Raccomandato per la maggior parte degli utenti. Aggiunge CQOps al Menu Start e al PATH. |
| ZIP Portatile | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Estrai ed esegui senza installare. |

### Linux — Debian / Ubuntu

| Architettura | Link | Per |
|-------------|------|---------|
| **amd64** | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La maggior parte dei PC Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Sistemi ARM a 64 bit |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS a 32 bit |

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Tarball Portatile

| Architettura | Link | Per |
|-------------|------|---------|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La maggior parte dei PC Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Sistemi ARM a 64 bit |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS a 32 bit |

### macOS

| Architettura | Link | Per |
|-------------|------|---------|
| **Apple Silicon** | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Mac M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Mac Intel |

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Dal Codice Sorgente

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build        # Solo compilazione; il binario viene scritto in build/
make install      # Compilazione e installazione di sistema
```

La compilazione da sorgente richiede Go 1.26 o superiore.

### Requisiti

- Dimensione del terminale: minimo 80×24 caratteri.
- Dimensione consigliata: 80×43 o maggiore.
- Si raccomanda un emulatore di terminale moderno. Su Windows, usa Windows Terminal invece della console legacy.

### Opzioni della Riga di Comando

```bash
cqops              # Avvia la TUI
cqops --offline    # Avvia senza attività di rete
cqops --version    # Mostra la versione ed esci
cqops --help       # Mostra l'aiuto
```

---

## Primo Avvio — Configurazione Guidata

Al primo avvio, CQOps apre una configurazione guidata per le impostazioni essenziali della stazione. Le integrazioni di rete possono essere saltate; la registrazione locale funziona senza di esse.

1. **Station & Logbook** — Configura il logbook iniziale, il nominativo di stazione, l'operatore e il grid locator. Campi opzionali: riferimenti SOTA/POTA/WWFF, regione IARU, zona CQ/ITU, DXCC e SIG/SIG Info. La configurazione Wavelog è disponibile qui: URL, chiave API, ID profilo stazione, Update e Test.

2. **Rig** — Configura un preset di rig: nome, modello, antenna, potenza e backend radio. Backend supportati: None, flrig e Hamlib rigctld. Impostazioni opzionali: controllo rotore Hamlib e integrazione UDP WSJT-X.

3. **Integrations** — Configura la ricerca callbook QRZ.com: flag di abilitazione, nome utente, password mascherata e Test.

4. **General** — Seleziona il fuso orario IANA. CQOps rileva il fuso orario di sistema per impostazione predefinita e fornisce un elenco scorrevole.

5. **Summary** — Rivedi la configurazione. Premi **Ctrl+S** per salvare e avviare CQOps.

**Navigazione guidata:** **Ctrl+S** avanza dopo la convalida. **Esc** torna indietro. **F10** esce. Spazio commuta le caselle di controllo. Tab e Shift+Tab si spostano tra i campi.

Tutte le impostazioni della configurazione guidata possono essere modificate successivamente dal menu di configurazione con **F9**.

---

## Avvio Rapido: Registra il Tuo Primo QSO

1. **Installa ed esegui CQOps.** Scarica il pacchetto per la tua piattaforma, avvia `cqops` e completa la configurazione guidata con almeno il tuo nominativo e grid locator.

2. **Usa il QSO form.** Il QSO form si apre su **F1**. Inserisci un nominativo; CQOps lo converte automaticamente in maiuscolo. Se il rig attivo è connesso tramite flrig o Hamlib, frequenza, banda, modo e submodo vengono compilati automaticamente. Data e ora sono impostate sull'UTC corrente.

3. **Spostati tra i campi.** Usa **Tab**, **Shift+Tab** e **↑/↓**.

4. **Salva il QSO.** Premi **Enter** o **Ctrl+S**. Se appare un avviso **DUPE!**, premi di nuovo **Enter** per salvare comunque, o **Esc** per annullare.

Il nuovo QSO appare immediatamente nella tabella Recent QSOs sotto il form.

---

## Panoramica della Schermata Principale

```text
┌─ Barra di Stato ────────────────────────────────────────────────────────────────┐
│  CQOps v0.8.8  Log Portable  Rig FTDx10  Call SP9MOA/P                          │
│  Net WSJT Hamlib DXC WL                                            23:00L 2100Z │
├─ Barra delle Schede ─────────────────────────────────────────────────────────────┤
│ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮         │
│ │F1 QSO│ │F2 QRZ│ │F4 DXC│ │F5 HRD│ │F6 REF│ │F7 BPL│ │F8 LOG│ │F9 CFG│         │
├─ Area di Contenuto Principale ───────────────────────────────────────────────────┤
│                                                                                  │
│  QSO form, tabella, mappa, editor o contenuto della schermata attiva              │
│                                                                                  │
├─ Barra di Aiuto ─────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Barra di Stato

La barra di stato mostra la versione di CQOps, il logbook attivo, il rig attivo, il nominativo di stazione e l'operatore attivo. A destra sono mostrate le etichette di stato delle integrazioni e l'ora locale (`L`) e UTC (`Z`).

**Colori delle etichette:**

| Colore | Significato |
|-------|---------|
| Bianco/predefinito | Connesso o attivo |
| Giallo | Disabilitato, in connessione o offline previsto |
| Rosso | Errore o disconnesso |
| Accento + grassetto | WSJT-X sta trasmettendo |

Etichette che possono apparire: **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC** e **WL**.

### Barra delle Schede

| Tasto | Scheda | Schermata |
|-----|-----|--------|
| F1 | QSO | QSO form e tabella Recent QSOs |
| F2 | QRZ | Partner view: dati callbook, mappa, statistiche, foto |
| F4 | DXC | Spot DX Cluster e filtri |
| F5 | HRD | Spot PSK Reporter e mappa di propagazione |
| F6 | REF | Ricerca riferimenti SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Browser piani di banda |
| F8 | LOG | Logbook Editor, ADIF, sincronizzazione Wavelog |
| F9 | CFG | Menu di configurazione |

### Barra di Aiuto

La riga inferiore mostra i tasti più rilevanti per la schermata attiva. Premi **?** per la sovrapposizione completa dell'aiuto.

---

## Flussi di Lavoro Comuni

### Operatività Portatile / SOTA / POTA

1. **Prima di uscire di casa**, esegui CQOps una volta con accesso a internet per popolare le cache: dati solari, dati REF e prefissi DXCC.
2. **Verifica la cache** prima di andare offline. Controlla che il pannello Solar mostri dati e che la ricerca REF su **F6** restituisca risultati.
3. **In campo**, avvia CQOps con `cqops --offline`. L'attività di rete viene saltata, evitando ritardi da servizi non raggiungibili.
4. **Registra normalmente.** La registrazione locale funziona senza internet.
5. **Carica dopo.** Quando torni online, apri il Logbook Editor con **F8** e premi **w** per caricare i QSO non inviati su Wavelog.

### Stazione di Club Condivisa e Hot-Seat

1. **Aggiungi profili operatore:** apri **F9 → Operators**, poi premi **Ins** per ogni operatore. Inserisci nominativo e nome.
2. **Cambia l'operatore attivo:** sul QSO form, premi **Ctrl+O**. L'operatore attivo è mostrato nella barra di stato e scritto nel campo `OPERATOR` dei QSO salvati.
3. **Usa il logging hot-seat:** l'operatore A registra un QSO, l'operatore B preme **Ctrl+O**, poi registra sotto il proprio profilo operatore.
4. **Usa Retain quando necessario:** attiva **Retain** se più operatori devono registrare lo stesso contatto senza ricompilare l'intero form.

Prima di salvare in una stazione condivisa, controlla il logbook attivo e l'operatore attivo nella barra di stato.

### Logbook Privato + Club

Molti operatori mantengono un logbook personale e uno o più logbook di club.

1. **Crea logbook:** apri **F9 → Logbooks**, poi premi **Ins** per ogni logbook.
2. **Cambia il logbook attivo:** premi **Ctrl+L** sul QSO form. La barra di stato mostra il logbook attivo.
3. **Mantieni separati i dati di stazione:** ogni logbook può avere il proprio nominativo, impostazioni Wavelog, impostazioni contest e operatori.
4. **Doppia registrazione rapida:** attiva **Retain**, salva il QSO in un logbook, premi **Ctrl+L**, poi salvalo di nuovo nell'altro logbook se appropriato.

### Rig Multipli

1. **Crea preset di rig:** apri **F9 → Rigs**, poi premi **Ins** per ogni rig.
2. **Imposta il backend:** usa flrig o Hamlib per rig controllati da CAT. Usa None per rig sintonizzati manualmente.
3. **Cambia il rig attivo:** premi **Ctrl+R** sul QSO form.
4. **Opera stazioni miste:** ad esempio, un rig HF controllato da CAT e un rig VHF/UHF manuale nella stessa sessione.
5. **Configura WSJT-X per rig:** ogni preset di rig può avere le proprie impostazioni UDP WSJT-X.

Quando il rig attivo ha il controllo CAT, CQOps può compilare automaticamente frequenza, banda, modo e submodo. Per i rig manuali, inseriscili tu stesso.

### FT8 / Auto-Registrazione WSJT-X

Quando WSJT-X è connesso via UDP, CQOps può registrare automaticamente i QSO digitali dai messaggi ADIF di WSJT-X.

- I QSO auto-registrati vengono salvati nel logbook attivo.
- I QSO auto-registrati duplicati vengono saltati.
- I QSO auto-registrati ereditano l'ID contest attivo.
- I QSO appaiono immediatamente in Recent QSOs.
- Se Wavelog è configurato e raggiungibile, i QSO auto-registrati possono essere caricati automaticamente.
- Se l'operatore WSJT-X non corrisponde all'operatore attivo, CQOps mostra un avviso.

Controlla il logbook attivo, l'operatore attivo e il contest attivo prima di lunghe sessioni digitali.

### Sincronizzazione Wavelog

La sincronizzazione Wavelog è opzionale. CQOps salva sempre i QSO prima in locale.

**Caricamento:** premi **w** nel Logbook Editor (**F8**). CQOps carica i QSO non inviati in lotti da 50 e traccia lo stato per QSO: non inviato, inviato o errore.

**Download:** premi **Ctrl+W** nel Logbook Editor. I download sono incrementali. CQOps recupera i QSO più recenti del `last_fetched_id` salvato per il logbook attivo. I duplicati vengono saltati.

Se un caricamento Wavelog fallisce, il QSO rimane nel logbook locale e può essere ritentato successivamente. Svuotare un logbook reimposta l'ID di recupero a `0`, consentendo un re-download completo.

---

## Funzionalità Principali

### Registrazione QSO

Il QSO form (**F1**) è la schermata principale di registrazione. Utilizza un layout a tre colonne e può compilare automaticamente i campi dal controllo rig, QRZ.com, ricerca Wavelog, dati DXCC/prefissi e database REF.

**Campi del form:**

| Colonna Sinistra | Colonna Centrale | Colonna Destra |
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

⚠️ I campi di scambio appaiono solo quando un contest è attivo. I campi contrassegnati con **(▼)** si scorrono con **PgUp/PgDn**.

La riga inferiore contiene:

- **Comment** (Commento)
- **Keep** — preserva il campo Comment tra i QSO; commuta con **Ctrl+T**
- **Retain** — preserva l'intero form dopo il salvataggio

La linea di percorso/rilevamento mostra distanza e azimut quando entrambi i grid locator sono noti. Può anche mostrare badge come **DUPE!**, **New Call!** e **New DXCC!**.

### Fonti di Compilazione Automatica

| Fonte | Campi |
|--------|--------|
| flrig / Hamlib | Frequency, Freq RX se split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| Database REF | Riferimenti SOTA, POTA, WWFF, IOTA |
| Ricerca Wavelog | Stato worked/confirmed se configurato |

### Registrazione Contest

I contest aggiungono campi di scambio e gestione dei seriali al QSO form.

Crea o configura un contest nel Logbook Editor (**F8**) con **Ins**. Imposta nome contest, data, ID contest ADIF e modelli di scambio.

Marcatori di modello supportati:

| Marcatore | Sostituito Con |
|--------|---------------|
| `@rst` | RST inviato o ricevuto |
| `@serial` | Numero seriale auto-incrementante |
| `@call` | Il tuo nominativo |
| `@grid` | Il tuo grid locator |
| `@name` | Nome operatore dal profilo operatore |

Premi **Ctrl+C** per scorrere il contest attivo. Quando un contest è attivo:

- il QSO form mostra i campi di scambio,
- i numeri seriali si auto-incrementano,
- Recent QSOs può filtrare sui QSO di contest,
- l'esportazione ADIF preserva `CONTEST_ID`.

### Logbook Editor

Il Logbook Editor (**F8**) serve per la gestione QSO, importazione/esportazione ADIF, sincronizzazione Wavelog e operazioni relative ai contest.

**Modifica in linea:** seleziona una riga con **↑/↓**, premi **Enter** o **e**, modifica il QSO, poi salva con **Ctrl+S**. Le modifiche si riflettono immediatamente in Recent QSOs.

### Importazione ed Esportazione ADIF

CQOps supporta l'importazione e l'esportazione ADIF 3.1.7.

- **Ctrl+I** importa un file ADIF, convalida i record, salta i duplicati e mostra un riepilogo.
- **Ctrl+E** esporta i QSO. L'esportazione può includere tutti i QSO o i QSO filtrati per contest.
- I QSO importati vengono contrassegnati per il caricamento Wavelog se la sincronizzazione Wavelog è configurata.

### Preferiti

I preferiti memorizzano preset di frequenza, modo e banda in 10 slot.

| Scorciatoia | Azione |
|----------|--------|
| Alt+0–9 | Richiama lo slot preferito |
| Alt+Shift+0–9 | Salva frequenza/modo/banda correnti nello slot |

I preferiti sono memorizzati nella configurazione e condivisi tra i logbook.

Esempio: per una configurazione di chiamata SOTA FM polacca, inserisci `145.55`, imposta modo `FM`, banda `2m`, poi premi **Alt+Shift+1**. Dopo, premi **Alt+1** per richiamarlo.

### Ricerca REF

La schermata REF (**F6**) cerca riferimenti SOTA, POTA, WWFF e IOTA. Cerca per prefisso, nome o designatore di riferimento. I riferimenti selezionati possono compilare il QSO form.

### Browser Piani di Banda

Il Browser Piani di Banda (**F7**) fornisce accesso rapido a bande radioamatoriali, gamme VHF/UHF, CB, PMR446 e preset broadcast. Una frequenza selezionata può essere usata per sintonizzare il rig attivo. I dati dei piani di banda possono anche essere esportati come Markdown.

---

## Integrazioni

### QRZ.com

La ricerca QRZ.com richiede accesso a internet e un abbonamento QRZ XML.

Premi **Ins** sul QSO form per compilare i campi callbook come nome, QTH, grid, paese, zone CQ/ITU, DXCC e continente. La Partner view (**F2**) può mostrare la foto dell'operatore quando disponibile.

### Wavelog

L'integrazione Wavelog richiede accesso a internet. Supporta caricamento, download incrementale e ricerca worked/confirmed.

Wavelog è configurato per logbook attivo con URL, chiave API e ID profilo stazione. CQOps salva sempre i QSO prima in locale; il fallimento del caricamento Wavelog non causa perdita di dati.

Vedi [Sincronizzazione Wavelog](#sincronizzazione-wavelog).

### flrig

L'integrazione flrig utilizza XML-RPC su HTTP. L'endpoint predefinito è `localhost:12345`.

CQOps può leggere frequenza, modo e potenza da flrig. L'operazione split è mappata come VFO A → Frequency e VFO B → Freq RX.

### Hamlib / rigctld

Il controllo rig Hamlib utilizza il demone TCP `rigctld`. CQOps può interrogare frequenza, modo, VFO, split e potenza in base al supporto del rig.

Alcuni rig o backend Hamlib non supportano tutte le interrogazioni. CQOps gestisce la mancanza di supporto del nome VFO in modo sicuro quando possibile.

### Hamlib Rotor / rotctld

Il controllo rotore utilizza Hamlib `rotctld`. CQOps supporta comandi di azimut, elevazione e stop.

Scorciatoie utili:

| Scorciatoia | Azione |
|----------|--------|
| Ctrl+←/→ | Regola l'azimut di 5° |
| Ctrl+↑/↓ | Regola l'elevazione di 5° |
| Ctrl+A | Punta il rotore sul rilevamento di percorso calcolato |
| Ctrl+F1 | Ferma il rotore |

### WSJT-X

L'integrazione WSJT-X utilizza messaggi UDP da WSJT-X. CQOps analizza i messaggi ADIF e può auto-registrare i QSO completati.

L'etichetta del rig diventa color accento mentre WSJT-X trasmette. Se l'operatore riportato da WSJT-X non corrisponde all'operatore attivo, CQOps mostra un avviso.

Vedi [FT8 / Auto-Registrazione WSJT-X](#ft8--auto-registrazione-wsjt-x).

### DX Cluster

L'integrazione DX Cluster utilizza una connessione telnet e richiede accesso a internet. Il server predefinito è `dxspots.com:7300`.

I filtri includono banda, continente, modo ed età/tempo. Premi **Enter** su uno spot per compilare il QSO form, sintonizzare il rig attivo e tornare alla schermata QSO. Premi **Space** per sintonizzare senza compilare il form. Premi **Backspace** per cancellare i filtri.

### PSK Reporter

L'integrazione PSK Reporter richiede accesso a internet. Fornisce spot di propagazione, filtri banda/tempo/modo e una mappa mondiale ASCII su **F5**.

### Dati Solari

I dati solari includono SFI, numero di macchie solari, indici A/K e condizioni banda per banda da hamqsl.com. Gli aggiornamenti in tempo reale richiedono accesso a internet. I dati nella cache rimangono disponibili offline dopo un recupero riuscito.

---

## Riferimento di Configurazione

La configurazione di CQOps è memorizzata in:

| Piattaforma | Percorso configurazione |
|----------|-------------|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Le credenziali sensibili sono memorizzate separatamente in `secrets.enc` nella stessa directory di configurazione. I segreti sono cifrati con una chiave legata alla macchina, quindi le credenziali devono essere reinserite quando si sposta una configurazione su un'altra macchina.

Apri la configurazione con **F9**.

| Menu | Configura |
|------|------------|
| Station | Nominativo, grid, zona CQ/ITU, regione IARU, riferimenti |
| Rig | Preset di rig, modello, antenna, potenza, backend, rotore, WSJT-X |
| Wavelog | URL, chiave API, ID profilo stazione |
| QRZ | Nome utente e password |
| DX Cluster | Host, porta, login |
| Operators | Profili operatore: nominativo e nome |
| Logbooks | Impostazioni stazione, Wavelog, contest e operatore per logbook |
| Notifications | Comportamento toast e notifiche |
| General | Fuso orario, unità di distanza, mappa, modalità debug |

### Multi-Logbook

Usa più logbook per operatività domestica, portatile, contest e club. Premi **Ctrl+L** per scorrere il logbook attivo. Ogni logbook mantiene i propri dettagli di stazione, impostazioni Wavelog, impostazioni contest e operatori.

### Multi-Operatore

I profili operatore contengono nominativo e nome dell'operatore. Premi **Ctrl+O** per scorrere l'operatore attivo. L'operatore attivo viene salvato nel campo ADIF `OPERATOR` e usato nei caricamenti Wavelog.

### Multi-Rig

I preset di rig memorizzano backend, modello, antenna, potenza, rotore e impostazioni WSJT-X. Premi **Ctrl+R** per scorrere il rig attivo.

### Segreti Cifrati

Dalla v0.8.7, le credenziali sono memorizzate cifrate.

- **File dei segreti:** `secrets.enc`
- **Posizione:** stessa directory di `config.yaml`
- **Permessi Unix:** `0600` dove supportato
- **Cifratura:** AES-256-GCM con chiave legata alla macchina
- **Dati protetti:** password QRZ, login DX Cluster, chiavi API Wavelog
- **Migrazione:** i segreti in chiaro da configurazioni precedenti migrano al primo avvio
- **Ripristino:** se `secrets.enc` è corrotto, CQOps si avvia con un avviso e chiede di reinserire le credenziali

---

## Scorciatoie da Tastiera

### Globali

| Tasto | Azione |
|-----|--------|
| F1 | QSO form e Recent QSOs |
| F2 | Partner view |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | Ricerca REF |
| F7 | Browser Piani di Banda |
| F8 | Logbook Editor |
| F9 | Configurazione / menu principale |
| F10 | Esci |
| Ctrl+F9 | Visualizzatore log |
| ? | Sovrapposizione aiuto |
| Ctrl+L | Scorri logbook attivo |
| Ctrl+R | Scorri rig attivo |
| Ctrl+C | Scorri contest attivo |
| Ctrl+O | Scorri operatore attivo |
| Esc | Torna alla schermata precedente |

### QSO Form — F1

| Tasto | Azione |
|-----|--------|
| Tab | Campo successivo |
| Shift+Tab | Campo precedente |
| ↑ / ↓ | Spostati nella colonna |
| Enter | Salva QSO, con conferma duplicato se necessario |
| Ctrl+S | Salva QSO da qualsiasi campo |
| Del | Cancella tutti i campi del form |
| Ins | Ricerca: QRZ, Wavelog, DXCC e controllo duplicato |
| PgUp / PgDn | Scorri banda, modo o submodo |
| Ctrl+D | Apri finestra spot |
| Ctrl+T | Commuta Keep Comment |
| Ctrl+←/→ | Regola azimut rotore di 5° |
| Ctrl+↑/↓ | Regola elevazione rotore di 5° |
| Ctrl+A | Punta rotore sul rilevamento dal proprio grid al grid partner |
| Ctrl+F1 | Ferma rotore |
| Alt+0–9 | Richiama slot preferito |
| Alt+Shift+0–9 | Salva frequenza/modo/banda correnti nello slot preferito |

### Logbook Editor — F8

| Tasto | Azione |
|-----|--------|
| ↑ / ↓ | Naviga tra le righe |
| PgUp / PgDn | Pagina precedente o successiva |
| Home / End | Prima o ultima riga |
| Enter / e | Modifica QSO selezionato |
| Delete | Elimina QSO selezionato |
| p | Svuota tutti i QSO |
| Ctrl+C | Commuta filtro contest |
| Ctrl+E | Esporta ADIF |
| Ctrl+I / Tab | Importa ADIF |
| w | Carica QSO non inviati su Wavelog |
| Ctrl+W | Scarica contatti da Wavelog |
| Esc / F6 | Chiudi editor, torna a QSO |

### DX Cluster — F4

| Tasto | Azione |
|-----|--------|
| ↑ / ↓ | Naviga tra gli spot |
| Enter | Compila form + sintonizza rig + vai a QSO |
| Space | Sintonizza rig sullo spot (rimani su DXC) |
| Home | Filtro banda avanti |
| End | Filtro banda indietro |
| \\ | Filtro continente |
| Ins | Filtro modo avanti |
| Del | Filtro modo indietro |
| PgUp | Filtro tempo avanti |
| PgDn | Filtro tempo indietro |
| Backspace | Cancella tutti i filtri |
| Esc / F4 | Torna al QSO form |

### Partner View — F2

| Tasto | Azione |
|-----|--------|
| F2 | Ciclo: Partner view → Foto → Indietro |
| Esc / F1 | Torna al QSO form |

---

## Risoluzione dei Problemi

### L'app non si avvia

- Il terminale deve avere almeno 80×24 caratteri.
- Su Windows, usa Windows Terminal, non la console legacy `cmd.exe`.
- Prova `cqops --offline` per escludere problemi di rete.
- Controlla i log: `~/.local/share/cqops/logs/` (Linux), `~/Library/Application Support/cqops/logs/` (macOS) o `%APPDATA%\cqops\logs\` (Windows).

### Il rig non si connette

- **flrig:** verifica che flrig sia in esecuzione e la porta corrisponda (predefinita `12345`).
- **Hamlib:** verifica che rigctld sia in esecuzione e la porta TCP sia corretta.
- Colore etichetta di stato: bianco = connesso, giallo = in connessione/disabilitato, rosso = errore.
- I toast di riconnessione soppressi sono normali — CQOps riprova in background.

### WSJT-X non auto-registra

- Verifica le impostazioni UDP di WSJT-X: Settings → Reporting → UDP Server.
- WSJT-X deve essere versione 2.6 o superiore.
- L'etichetta di stato dovrebbe essere bianca (predefinita) quando WSJT-X è in esecuzione.

### Il caricamento Wavelog fallisce

- Verifica URL, chiave API e ID profilo stazione nella configurazione.
- Etichetta di stato: bianco = raggiungibile, giallo = disabilitato/nessuna internet, rosso = errore.
- Gli errori di caricamento sono mostrati come toast; i QSO rimangono salvati localmente.
- I fallimenti di singoli QSO non bloccano il resto del lotto.

### Problemi con il file di configurazione

- Configurazione: `~/.config/cqops/config.yaml` (Linux/macOS) o `%APPDATA%\cqops\config.yaml` (Windows).
- Segreti: `secrets.enc` nella stessa directory.
- Se la configurazione è corrotta, eliminala e riavvia — la configurazione guidata ne creerà una nuova.
- Il campo `last_fetched_id` appare solo dopo un download Wavelog riuscito.

### Prestazioni

- Disabilita il rendering della mappa e il pannello solare nelle impostazioni General.
- Chiudi le schede non utilizzate (DXC, PSK).
- Esegui con `--offline` se la rete non è affidabile.

### Segnalazione Bug

Abilita la **modalità Debug** prima di riprodurre un problema — F9 → General → Debug, o imposta `debug: true` nella configurazione. I log completi vengono scritti nella directory dei log specifica della piattaforma.

Segnala i problemi su [GitHub Issues](https://github.com/szporwolik/cqops/issues) con:
- Versione CQOps (`cqops --version`)
- Sistema operativo ed emulatore di terminale
- Passi per riprodurre
- Log di debug
