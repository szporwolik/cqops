---
title: Manuale utente di CQOps
description: Guida pratica alla configurazione di CQOps e alla registrazione dei collegamenti in stazione e in portatile
---

# Manuale utente di CQOps

CQOps è un logger radioamatoriale gestito da tastiera, adatto alla stazione fissa, al portatile, alle stazioni di club e ai contest occasionali. I QSO vengono salvati prima sul computer; i servizi Internet sono facoltativi. Inizia dalla registrazione manuale e aggiungi controllo radio e servizi online quando servono.

I nomi di menu e campi seguono l’interfaccia inglese. Le scorciatoie dipendono dalla schermata attiva: consulta la barra di aiuto o **?**.

## Indice

1. [Installazione](#installation)
2. [Prima configurazione](#setup)
3. [Primo QSO](#first-qso)
4. [Schermate e stato](#screens)
5. [Registrazione quotidiana](#logging)
6. [Profili di stazione](#profiles)
7. [Log e backup](#logbook)
8. [Radio e modi digitali](#radio)
9. [Servizi online](#online)
10. [GPS e APRS](#position)
11. [Attività in portatile](#portable)
12. [Contest](#contests)
13. [CQOps Live](#dashboard)
14. [Scorciatoie da tastiera](#keys)
15. [Risoluzione dei problemi e assistenza](#help)

<a id="installation"></a>

## Installazione

Scarica CQOps dalla [pagina delle versioni](https://github.com/szporwolik/cqops/releases). Il terminale deve misurare almeno 75 × 24 caratteri; 80 × 43 o più è preferibile.

| Sistema | Installazione |
|---|---|
| Windows | Scarica `cqops-setup.exe` oppure estrai `cqops-windows-portable.zip` per usarlo senza installazione. Si consiglia Windows Terminal. |
| Debian, Ubuntu, Linux Mint, Pop!_OS | Scarica il `.deb` corretto: `amd64` per la maggior parte dei PC Intel/AMD, `arm64` per ARM a 64 bit, `armhf` per Raspberry Pi OS a 32 bit. Aprilo con l’installatore di pacchetti. |
| Fedora, RHEL, Rocky, AlmaLinux | Usa i comandi del repository riportati sotto. |
| Arch, Manjaro, CachyOS | Installa il pacchetto AUR con `paru -S cqops-bin` o `yay -S cqops-bin`. |
| Altri sistemi Linux | Scarica ed estrai l’archivio Linux `.tar.gz` adatto al processore. |
| macOS | Scarica `cqops-darwin-arm64` per Apple Silicon o `cqops-darwin-amd64` per Intel. Usa i comandi seguenti. |

Sui sistemi Debian puoi anche installare dal repository:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

Sui sistemi Fedora:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

Su macOS esegui questi comandi nella cartella di download, sostituendo `FILE` con il nome esatto del file:

```bash
chmod +x FILE
sudo mv FILE /usr/local/bin/cqops
```

Avvia `cqops` o il programma portatile estratto. `cqops --offline` avvia senza rete, `cqops --version` mostra la versione e `cqops --help` le opzioni di avvio. Esporta i log prima di aggiornare.

<a id="setup"></a>

## Prima configurazione

La procedura iniziale richiede nome del log, nominativo di stazione, locatore Maidenhead e continente. Il **nominativo di stazione** è quello usato in trasmissione; il **profilo operatore** identifica la persona ai comandi.

**Ctrl+A** mostra referenze di stazione e zone CQ/ITU facoltative. La tua referenza SOTA/POTA/WWFF va nelle impostazioni di stazione/log; quelle del modulo QSO riguardano il corrispondente. Imposta successivamente la regione IARU in **F9 → Logbooks**.

Crea un profilo radio con nome, antenna e potenza. Scegli **None** per inserire manualmente frequenza e modo, **flrig** oppure **Hamlib**. Configura le connessioni facoltative dopo aver verificato la registrazione di base.

**Tab / Shift+Tab** cambia campo, **Space** modifica le opzioni e **Save & Next** prosegue. **Esc** torna indietro; **F10** esce. Controlla il riepilogo e salva. CQOps rileva il fuso orario del computer; date e orari QSO sono in UTC. Controlla l’orologio prima di operare.

<a id="first-qso"></a>

## Primo QSO

1. Premi **F1**. Verifica log, nominativo, operatore, radio e contest attivi.
2. Inserisci il nominativo del corrispondente. **Ins** avvia la ricerca se configurata.
3. Controlla data/ora UTC, frequenza in MHz, banda, modo e rapporti inviato/ricevuto.
4. Aggiungi nome, QTH, locatore, referenza o commento se utili.
5. Premi **Enter**. Il contatto appare in Recent QSOs.

Se **DUPE!** richiede conferma, premi ancora **Enter** per salvare comunque oppure **Esc** per annullare la conferma. L’avviso invita a controllare il contatto, non dimostra che debba essere scartato.

<a id="screens"></a>

## Schermate e stato

| Tasto | Schermata | Funzione |
|---|---|---|
| F1 | QSO | Inserimento contatti e QSO recenti |
| F2 | Partner | Dati del corrispondente, mappa, statistiche, foto |
| F3 | APRS | Stazioni vicine |
| F4 | DX Cluster | Spot e filtri |
| F5 | PSK Reporter | Rapporti di ricezione digitale |
| F6 | References | Ricerca SOTA, POTA, WWFF, IOTA |
| F7 | Band Plan | Frequenze e preselezioni |
| F8 | Logbook | Modifica, importazione, esportazione, sincronizzazione |
| F9 | Configuration | Impostazioni di stazione e servizi |
| F10 | Quit | Uscita da CQOps |

La barra superiore mostra configurazione attiva, ora locale (**L**) e UTC (**Z**). Il bianco indica normalmente attività, il giallo disattivazione/connessione/attesa, il rosso un errore. WSJT è evidenziato durante la trasmissione. **WL!** segnala una vecchia chiave Wavelog non supportata.

<a id="logging"></a>

## Registrazione quotidiana

Usa **Tab / Shift+Tab** tra i campi e **PgUp / PgDn** per cambiare banda, modo o sottomodo. **Shift+Backspace** cancella il campo corrente; **Del** l’intero modulo. In split verifica **Freq RX**.

**Keep** conserva il commento dopo il salvataggio. **Retain** conserva tutto il modulo: controlla nominativo, ora, rapporti e referenze prima del contatto successivo. I campi di scambio compaiono solo con un contest attivo. **SIG / SIG Info** serve per altri dati relativi a gruppi d’interesse.

Con entrambi i locatori noti, CQOps mostra distanza e azimut. Il callbook può indicare il domicilio invece della posizione portatile attuale: verificalo. Gli indicatori di nuovo nominativo, nuovo DXCC e duplicato aiutano a valutare il contatto.

**F6** cerca referenze per nome o identificativo e può compilare quella del corrispondente. **F7** mostra i piani di banda e può sintonizzare la radio collegata. Sono aiuti operativi, non autorizzazioni a trasmettere: verifica le tue abilitazioni e il piano locale.

Tre preferiti condivisi memorizzano frequenza, modo e banda:

| Posizione | Richiama | Salva valori attuali |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

<a id="profiles"></a>

## Profili di stazione

Crea log, operatori, radio e contest nei rispettivi menu **F9**; **Ins** aggiunge una voce. Dalla schermata QSO:

| Scorciatoia | Cambia |
|---|---|
| Ctrl+L | Log |
| Ctrl+O | Operatore |
| Ctrl+R | Radio |
| Ctrl+C | Contest |

I log mantengono dati di stazione e impostazioni Wavelog/APRS separati. Il profilo operatore identifica la persona; il nominativo viene registrato nel campo ADIF `OPERATOR`. I profili radio conservano apparati, potenza, controllo radio/rotore e WSJT-X. Verifica la barra di stato dopo ogni cambio, soprattutto con la registrazione digitale automatica.

Gli altri menu **F9** riguardano visualizzazione, unità, fuso orario, callbook, integrazioni e notifiche sonore.

<a id="logbook"></a>

## Log e backup

In **F8**, seleziona un QSO e premi **Enter** o **e** per modificarlo. Salva con **Enter** e conferma. **Delete** elimina il contatto selezionato. Esegui un backup prima di modifiche massive; **Ctrl+P** cancella tutti i QSO, non è una ricerca.

| Scorciatoia F8 | Azione |
|---|---|
| Ctrl+I | Importa ADIF, verifica i record e salta i duplicati |
| Ctrl+E | Esporta tutti i contatti o una selezione per contest |
| Ctrl+W | Invia i contatti non ancora caricati su Wavelog |
| Alt+W | Scarica da Wavelog |

Controlla riepilogo d’importazione e selezione d’esportazione. I contatti importati possono essere caricati successivamente su Wavelog. CQOps supporta ADIF 3.1.7 e conserva ID dei contest e scambi. Mantieni backup separati per ogni log, preferibilmente su un altro dispositivo. ADIF salva i contatti, non tutte le impostazioni o credenziali.

La configurazione è in `~/.config/cqops/config.yaml` su Linux/macOS e `%APPDATA%\cqops\config.yaml` su Windows. Le credenziali sono separate in `secrets.enc`; reinseriscile quando cambi computer. Non cancellare la configurazione come primo tentativo di risoluzione.

<a id="radio"></a>

## Radio e modi digitali

### Controllo radio

In **F9 → Rigs** scegli flrig o Hamlib e fai corrispondere le impostazioni di connessione. Avvia prima flrig o `rigctld`. flrig normalmente usa `localhost:12345`. Le letture disponibili di frequenza, modo, split e potenza dipendono dalla radio. Con **None**, inseriscile manualmente.

### WSJT-X

Usa WSJT-X 2.6 o successivo. Fai corrispondere **Settings → Reporting → UDP Server** ai parametri UDP del profilo radio CQOps attivo. Registra un QSO di prova completato in WSJT-X e verifica che compaia in CQOps.

I QSO ricevuti usano log e contest attivi; i duplicati vengono ignorati. Controlla operatore e indicatore WSJT prima della sessione. CQOps avvisa se gli operatori non coincidono. Se configurato, può seguire il caricamento Wavelog. Scegli Mode/Submode appropriati: FT8 viene esportato come FT8, FT4/FT2 come MFSK con il relativo sottomodo.

### Controllo rotore

Il controllo Hamlib `rotctld` è sperimentale. Verifica direzione e limiti fisici. Predisponi un arresto sicuro: impostazioni errate possono danneggiare antenna, rotore o linea di alimentazione.

| Scorciatoia | Azione |
|---|---|
| Alt+, / Alt+. | Azimut −5° / +5° |
| Alt+' / Alt+; | Elevazione −5° / +5° |
| Alt+\ | Punta verso l’azimut calcolato |
| Alt+/ | Ferma il movimento |

<a id="online"></a>

## Servizi online

### Callbook

Imposta fornitori e priorità in **F9 → Callbook**, poi premi **Ins** nel modulo QSO. CQOps prova i fornitori abilitati in ordine. La ricerca del nominativo base può omettere prefissi/suffissi portatili; verifica la posizione restituita.

| Fornitore | Accesso |
|---|---|
| QRZ.com | Abbonamento XML e credenziali |
| HamQTH | Account gratuito |
| QRZ.RU | Accesso API distinto da quello del sito |
| Callook.info | Nominativi USA; nessun account |

**F2** mostra il corrispondente. Le foto dipendono da fornitore e terminale; **Kitty Graphics**, sperimentale in General, richiede un terminale compatibile come Kitty, Ghostty o WezTerm.

### Wavelog

Configura URL, token API v2 (`wl2_…`) e profilo di stazione per ogni log. Le vecchie chiavi v1 non sono accettate. Selezionare una stazione Wavelog può compilare i dati locali: controlla nominativo, locatore e referenze prima di salvare.

I QSO vengono salvati prima localmente. Riprova gli invii falliti con **F8 → Ctrl+W**; **Alt+W** scarica i contatti. Aprendo un QSO collegato per modificarlo, CQOps può aggiornarlo da Wavelog. Modifiche ed eliminazioni online riguardano anche la copia remota. Leggi la conferma, specialmente offline; una modifica solo locale non è necessariamente arrivata a Wavelog.


Stazioni di club: usate la chiave `wl2_` del proprietario insieme all'opzione **Stazione di club condivisa** nel modulo del registro. I contatti sincronizzati diventano di sola lettura — modifiche ed eliminazioni vanno fatte lato Wavelog e CQOps non invia mai PATCH o DELETE per essi. I nuovi contatti vengono caricati normalmente e attribuiti all'operatore attivo (in mancanza, al nominativo della stazione). La chiave API è salvata cifrata e, con questa opzione attiva, non viene mai più mostrata dopo il salvataggio — lasciate vuoto il campo per mantenerla, digitatene una nuova per sostituirla.
### DX Cluster e propagazione

Configura DX Cluster in Integrations e apri **F4**. **b / c / m / t** filtrano banda, continente dello spotter, modo ed età. **Backspace** cancella i filtri. **Enter** compila QSO, sintonizza la radio collegata e torna a F1; **Space** sintonizza senza uscire dal cluster.

Su F1, **Ctrl+S** apre la finestra dello spot e **Ctrl+P** prende il nominativo dallo spot visualizzato più vicino. Verificalo prima dell’invio. **F5** mostra rapporti PSK Reporter, non garantisce la propagazione attuale. Solar mostra condizioni HamQSL; i valori memorizzati possono essere vecchi. **F5 è disattivato per impostazione predefinita: attivate PSK Reporter in Integrazioni.**

<a id="position"></a>

## GPS e APRS

### GPS

Configura GPS seriale o GPSD in Integrations. Abilita **Grid from GPS** nelle impostazioni stazione/log per usare il locatore in QSO, azimut, APRS e dashboard. GPS rosso indica errore, giallo nessuna posizione, bianco posizione acquisita. Controlla il locatore prima di operare. Scegli 6, 8 o 10 caratteri; più caratteri non garantiscono maggiore precisione del ricevitore.

### APRS

| Servizio | Connessione |
|---|---|
| APRS-IS | Server APRS Internet |
| KISS | TNC hardware seriale e radio |
| KISS Server | TNC TCP come Dire Wolf; può funzionare localmente |

Scegli il servizio in **F9 → Integrations → APRS**. Imposta nominativo/SSID, simbolo, commento, raggio e intervallo in **F9 → Logbooks → [active logbook] → APRS**. Abilita **APRS TX** e **Send beacons** solo se vuoi trasmettere. La sola ricezione indica **APRS-RX**. I beacon rivelano la posizione: controllala e considera chi potrà riceverla.

L’intervallo automatico minimo è cinque minuti. **F3** mostra le stazioni ascoltate di recente: frecce per selezionare, **Enter** per compilare QSO, **d / t / s** per distanza/età/tipo, **Backspace** per cancellare i filtri, **b** per inviare subito un beacon configurato. I beacon GPS richiedono **Grid from GPS** e una posizione valida.

<a id="portable"></a>

## Attività in portatile

Prima di partire seleziona il log portatile; controlla nominativo, locatore, referenza, radio, antenna e potenza. Prova l’intera stazione e avvia CQOps online per aggiornare referenze e prefissi. Verifica che **F6** trovi le referenze necessarie. Esporta un backup.

Il log locale funziona senza Internet. `cqops --offline` salta le funzioni di rete: non fare affidamento su ricerche in diretta o sincronizzazione. Prova gli apparati in rete locale con la modalità di avvio scelta prima della partenza. I dati in cache possono essere obsoleti.

Dopo l’attività controlla numero di QSO e referenze, esporta ADIF, conserva una copia e invia i contatti in attesa a Wavelog se utilizzato. Verifica il formato richiesto da ciascun diploma e converti l’esportazione se necessario.

<a id="contests"></a>

## Contest

CQOps supporta contest occasionali, scambi, progressivi e ritmo dei QSO. Non è un sistema completo di punteggio o invio dei log. Per attività avanzata usa un logger dedicato.

In **F9 → Contests**, premi **Ins** e imposta nome, data, ID ADIF, progressivo iniziale e modelli di scambio inviato/ricevuto.

| Segnaposto | Valore |
|---|---|
| `@rst` | Rapporto inviato o ricevuto |
| `@serial` | Numero progressivo |
| `@cqz` / `@mycqz` | Zona CQ del corrispondente / propria |
| `@itu` / `@myitu` | Zona ITU del corrispondente / propria |
| `@grid` / `@mygrid` | Locatore del corrispondente / proprio |

Su **F1**, **Ctrl+C** cambia contest. Verifica scambio e prossimo numero prima di trasmettere. La barra mostra totale, prossimo numero e tempi; finestre larghe mostrano più statistiche di ritmo. Al termine torna alla modalità senza contest attivo.

Per esportare apri **F8**, scegli il filtro contest con **Ctrl+C**, poi **Ctrl+E** e verifica la selezione. Il formato è ADIF, non Cabrillo. Rispetta formato e regole d’invio dell’organizzatore.

<a id="dashboard"></a>

## CQOps Live

Abilita **F9 → Integrations → HTTP Server** e salva con **Ctrl+S**. Sul computer CQOps apri `http://localhost:8073`.

L’indirizzo predefinito `0.0.0.0` consente l’accesso dalla rete locale, se permesso dal firewall. Su un altro dispositivo usa l’IP del computer CQOps e la porta `8073`. `127.0.0.1` limita l’accesso al solo computer CQOps. Usa una rete fidata; non inoltrare la porta verso Internet.

La dashboard aggiorna automaticamente contatto corrente, mappe QSO, contatti recenti, ritmo, operatori, APRS e dati disponibili di propagazione/meteo. I livelli Internet possono mancare offline. Header 1, Header 2, Logo URL ed Event Start personalizzano l’evento; la data iniziale filtra statistiche ed elenchi QSO.

<a id="keys"></a>

## Scorciatoie da tastiera

| Schermata | Tasti | Azione |
|---|---|---|
| Generale | ? / Esc / F10 | Aiuto / indietro / esci |
| QSO | Tab / Shift+Tab | Campo successivo / precedente |
| QSO | Enter / Ins | Salva / ricerca |
| QSO | Shift+Backspace / Del | Cancella campo / modulo |
| QSO | Ctrl+L / Ctrl+O / Ctrl+R / Ctrl+C | Cambia log / operatore / radio / contest |
| Log | ↑ / ↓, PgUp / PgDn, Home / End | Selezione, pagina, prima / ultima riga |
| Log | Enter o e / Delete | Modifica / elimina QSO selezionato |
| Log | Ctrl+I / Ctrl+E | Importa / esporta ADIF |
| Log | Ctrl+W / Alt+W | Carica / scarica Wavelog |
| Log | Ctrl+C / Backspace | Filtro contest / cancella ricerca |

Le scorciatoie dipendono dalla schermata: **Ctrl+C non chiude CQOps**. Sui portatili può servire **Fn** per i tasti funzione. Se il terminale intercetta un tasto, controlla le sue impostazioni e la barra d’aiuto CQOps.

<a id="help"></a>

## Risoluzione dei problemi e assistenza

| Problema | Prime verifiche |
|---|---|
| Avvio o schermata incompleta | Dimensioni terminale, Windows Terminal su Windows, prova `cqops --offline` |
| Radio scollegata | Profilo attivo, flrig/rigctld avviato, modello, porta seriale, velocità, host/porta, seriale occupata da altro programma |
| QSO WSJT-X assente | UDP corrispondente, indicatore WSJT, QSO realmente registrato in WSJT-X, log attivo |
| Errore Wavelog | URL, token `wl2_`, profilo, Internet; i QSO locali restano salvati |
| Nessuna posizione GPS | Porta/velocità o indirizzo GPSD, cielo libero, posizione valida, Grid from GPS |
| APRS senza beacon | Log, APRS TX e Send beacons, nominativo/SSID, TNC/radio o Internet |
| Dashboard irraggiungibile | Server attivo, IP/porta corretti, firewall; localhost indica il dispositivo che esegue il browser |

Usa **F9** prima di modificare i file di configurazione. Reinserisci le credenziali se CQOps segnala problemi con i segreti salvati o dopo un cambio di computer.

Se necessario abilita **F9 → General → Debug**, riproduci il problema in sicurezza e raccogli il log diagnostico; disabilita poi il debug.

| Sistema | Log diagnostici |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

Segnala i problemi su [GitHub Issues](https://github.com/szporwolik/cqops/issues), includendo versione CQOps, sistema, terminale, passaggi e log pertinente. Rimuovi password, token API e informazioni private prima di condividere.
