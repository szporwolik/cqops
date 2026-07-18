---
title: Manuel utilisateur de CQOps
description: Guide d’installation, de configuration et d’utilisation de CQOps — un logger radioamateur rapide, conçu en priorité pour le terminal
---

# Manuel utilisateur de CQOps

CQOps est un logger radioamateur rapide, conçu en priorité pour le terminal, destiné aux opérateurs qui souhaitent saisir leurs contacts de manière fiable au clavier tout en conservant une faible charge système. Il convient à l’utilisation en station, aux opérations portables, aux stations de club, aux field days ainsi qu’aux machines de classe Raspberry Pi ou aux anciens ordinateurs portables.

CQOps enregistre toujours les QSO localement en premier. Les intégrations utilisant Internet sont facultatives.

## Sommaire

1. [Présentation de CQOps](#what-cqops-is)
2. [Téléchargement et installation](#download-and-installation)
3. [Premier démarrage](#first-launch)
4. [Enregistrer votre premier QSO](#log-your-first-qso)
5. [Écran principal](#main-screen)
6. [Flux de travail courants](#common-workflows)
7. [Enregistrement des QSO](#qso-logging)
8. [Éditeur de log et ADIF](#logbook-editor-and-adif)
9. [Concours](#contests)
    - [Configurer un concours](#setting-up-a-contest)
    - [Barre d’état inférieure](#bottom-status-bar)
    - [Panneau de statistiques du concours](#contest-statistics-panel)
    - [Export ADIF du concours](#contest-adif-export)
    - [Comportement du mode concours](#contest-mode-behavior)
10. [Favoris, références et plans de bandes](#favorites-references-and-band-plans)
11. [Intégrations](#integrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Configuration](#configuration)
14. [Raccourcis clavier](#keyboard-shortcuts)
15. [Dépannage](#troubleshooting)
16. [Signaler des bogues](#reporting-bugs)

---

<a id="what-cqops-is"></a>
## Présentation de CQOps

CQOps est centré sur la saisie rapide des QSO, l’enregistrement local des données et une utilisation pratique sur le terrain.

### Principes essentiels

- **Utilisation orientée terminal** — optimisée pour le clavier.
- **Enregistrement offline-first** — l’enregistrement local des QSO fonctionne sans accès à Internet.
- **Faible consommation de ressources** — adapté aux systèmes de classe Raspberry Pi, aux anciens ordinateurs portables et aux PC de station partagés.
- **Conception portable** — distribué sous la forme d’un unique binaire Go.
- **Plusieurs logs** — utile pour les logs personnels, portables, de concours et de club.
- **Plusieurs opérateurs** — adapté aux flux hot-seat et aux stations de club partagées.
- **Plusieurs équipements** — chaque preset d’équipement peut conserver ses propres réglages de backend et de WSJT-X.
- **Intégrations facultatives** — Callbook multi-fournisseur (QRZ.com, HamQTH, QRZ.RU, Callook.info), Wavelog, DX Cluster, PSK Reporter, GPS, APRS, contrôle du poste, contrôle du rotor, données solaires et CQOps Live dashboard dans le navigateur.

L’enregistrement local ne nécessite pas d’accès à Internet. Les fonctions réseau sont ignorées en mode `--offline`.

### À qui s’adresse CQOps

CQOps convient notamment aux utilisateurs suivants :

- opérateurs portables,
- activateurs SOTA et POTA,
- stations de club,
- équipes de field day,
- opérateurs qui préfèrent travailler dans un terminal,
- stations nécessitant un passage rapide entre opérateurs, logs ou équipements.

CQOps n’a pas vocation à remplacer toutes les fonctions d’un logger de bureau complet ou d’une plateforme web de log. Il se concentre sur la saisie rapide dans le terminal, l’utilisation sur le terrain, le fonctionnement hors ligne et les stations partagées.

### Utilisation en club et station partagée

CQOps a été conçu en tenant compte des besoins des clubs radioamateurs. L’opérateur actif est toujours visible dans la barre d’état — **un seul coup d’œil** suffit pour savoir qui utilise actuellement la station. Le changement d’opérateur nécessite une seule combinaison (`Ctrl+O`) et prend effet immédiatement : l’indicatif et le nom de l’opérateur sont inscrits dans chaque QSO suivant. Aucun logout, aucune demande de mot de passe, aucune interruption.

Les logs, presets d’équipement et concours sont parcourus de la même manière — `Ctrl+L`, `Ctrl+R`, `Ctrl+C`. Une station de club avec des opérateurs tournants, plusieurs équipements et plusieurs concours actifs peut changer de contexte en moins d’une seconde, sans toucher à la souris.

Lors des field days et des événements publics, le **CQOps Live dashboard** peut projeter une carte en temps réel, le flux des QSO et des statistiques sur un grand écran. Les visiteurs et les membres du club peuvent suivre l’activité de la station sans se regrouper autour du terminal de l’opérateur. Il suffit d’activer l’intégration **HTTP Server** et d’y accéder depuis n’importe quel appareil équipé d’un navigateur web.

---

<a id="download-and-installation"></a>
## Téléchargement et installation

Parcourir toutes les versions :

<https://github.com/szporwolik/cqops/releases>

### Windows

| Paquet | Lien | Remarques |
|---|---|---|
| Programme d’installation | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recommandé pour la plupart des utilisateurs. Ajoute CQOps au menu Démarrer et au PATH. |
| ZIP portable | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extraire et exécuter sans installation. |

### Linux — Debian / Ubuntu / Pop!_OS / Linux Mint

Ajoutez le dépôt Cloudsmith APT, puis installez :

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

Ou téléchargez le `.deb` directement :

| Architecture | Lien | Usage |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La plupart des PC Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Systèmes ARM 64 bits |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS 32 bits |

Installez le paquet téléchargé :

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Fedora / RHEL / Rocky / AlmaLinux

Ajoutez le dépôt Cloudsmith RPM, puis installez :

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

### Linux — Arch / Manjaro / CachyOS

Installez depuis l'AUR avec n'importe quel helper AUR :

```bash
yay -S cqops-bin
```

Également disponible via `paru`, `pacaur`, `aura` ou `makepkg` manuel. PKGBUILD sur [aur.archlinux.org/packages/cqops-bin](https://aur.archlinux.org/packages/cqops-bin).

### Linux — archive Tarball portable

| Architecture | Lien | Usage |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La plupart des PC Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Systèmes ARM 64 bits |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS 32 bits |

### macOS

| Architecture | Lien | Usage |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Mac M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Mac Intel |

Installation manuelle :

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Compiler depuis les sources

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

La compilation depuis les sources nécessite Go 1.26 ou une version ultérieure.

### Exigences relatives au terminal

| Exigence | Valeur |
|---|---|
| Taille minimale du terminal | 80×24 caractères |
| Taille recommandée | 80×43 caractères ou plus |
| Terminal Windows recommandé | Windows Terminal |
| Terminal prenant en charge Kitty graphics | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) ou [WezTerm](https://wezfurlong.org/wezterm/) |

### Commandes de base

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

<a id="first-launch"></a>
## Premier démarrage

Au premier démarrage, CQOps ouvre l’assistant de configuration. Seules les informations essentielles de la station sont nécessaires pour l’enregistrement local. Les intégrations réseau peuvent être ignorées puis configurées ultérieurement.

### Pages de l’assistant

| Page | Configuration effectuée |
|---|---|
| Station & Logbook | Log initial, indicatif de station, opérateur, grid locator, références et zones facultatives, Wavelog URL/API/station profile ID |
| Rig | Preset d’équipement, modèle, antenne, puissance, backend, rotor facultatif et réglages UDP WSJT-X facultatifs |
| Integrations | Réglages de recherche du callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info) |
| General | Fuseau horaire IANA |
| Summary | Vérification et enregistrement |

Backends d’équipement pris en charge :

- None,
- flrig,
- Hamlib `rigctld`.

### Navigation dans l’assistant

| Key | Action |
|---|---|
| Ctrl+S | Valider et continuer ; sur **Summary**, enregistrer et démarrer CQOps |
| Esc | Revenir en arrière |
| F10 | Quitter |
| Tab / Shift+Tab | Passer d’un champ à l’autre |
| Space | Basculer les cases à cocher |

Les réglages de l’assistant peuvent être modifiés ultérieurement avec **F9**.

---

<a id="log-your-first-qso"></a>
## Enregistrer votre premier QSO

1. Démarrez CQOps :

   ```bash
   cqops
   ```

2. Terminez l’assistant de configuration en indiquant au minimum votre indicatif et votre grid locator.

3. Ouvrez **QSO form** avec **F1**.

4. Saisissez l’indicatif du correspondant. CQOps convertit automatiquement les indicatifs en majuscules.

5. Complétez les autres champs. Si l’équipement actif est connecté par flrig ou Hamlib, CQOps peut remplir automatiquement frequency, band, mode et submode.

6. Appuyez sur **Enter** pour enregistrer.

7. Si un avertissement **DUPE!** apparaît, appuyez à nouveau sur **Enter** pour enregistrer malgré tout, ou sur **Esc** pour annuler.

Le QSO enregistré apparaît immédiatement dans le tableau **Recent QSOs**.

---

<a id="main-screen"></a>
## Écran principal

CQOps utilise une disposition fixe dans le terminal :

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

La **Status bar** affiche :

- la version de CQOps,
- le log actif,
- l’équipement actif,
- l’indicatif de la station,
- l’opérateur actif,
- les libellés d’état des intégrations,
- l’heure locale marquée `L`,
- l’heure UTC marquée `Z`.

Les libellés courants comprennent **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** et **GPS**. Le libellé **GPS** suit la même convention de couleurs : rouge lorsque la connexion est absente, jaune lorsque le GPS est connecté sans fix et blanc lorsqu’une position est acquise.

| Couleur | Signification |
|---|---|
| Blanc/par défaut | Connecté ou actif |
| Jaune | Désactivé, en cours de connexion ou normalement hors ligne |
| Rouge | Erreur ou déconnecté |
| Couleur d’accent + gras | WSJT-X émet |

### Onglets principaux

| Key | Tab | Screen |
|---|---|---|
| F1 | QSO | **QSO form** et **Recent QSOs** |
| F2 | QRZ | **Partner view** : données du callbook, carte, statistiques, photo |
| F4 | DXC | Spots et filtres de **DX Cluster** |
| F5 | HRD | Spots de **PSK Reporter** et carte de propagation |
| F6 | REF | Recherche de références SOTA/POTA/WWFF/IOTA |
| F7 | BPL | **Band Plan Browser** |
| F8 | LOG | **Logbook Editor**, ADIF, synchronisation Wavelog |
| F9 | CFG | Menus de configuration |

La **Help bar** affiche les raccourcis correspondant à l’écran actif. Appuyez sur **?** pour ouvrir l’intégralité du **Help overlay**.

---

<a id="common-workflows"></a>
## Flux de travail courants

### Opération portable, SOTA ou POTA

Avant le départ :

1. Exécutez CQOps une fois avec un accès à Internet.
2. Laissez CQOps télécharger ou actualiser les données en cache telles que les données solaires, les données REF et les préfixes DXCC.
3. Vérifiez que le panneau **Solar** affiche des données.
4. Vérifiez que la recherche **REF** sous **F6** renvoie des résultats.

Sur le terrain :

1. Démarrez CQOps en mode hors ligne :

   ```bash
   cqops --offline
   ```

2. Loggez normalement. Les QSO sont enregistrés localement.
3. Une fois la connexion rétablie, ouvrez **F8** et appuyez sur **w** pour envoyer à Wavelog les QSO non transmis.

### Station de club partagée et enregistrement hot-seat

1. Ouvrez **F9 → Operators**.
2. Appuyez sur **Ins** pour ajouter des profils d’opérateur.
3. Dans **QSO form**, appuyez sur **Ctrl+O** pour changer d’opérateur actif.
4. Vérifiez l’opérateur actif dans la barre d’état avant l’enregistrement.
5. Utilisez **Retain** lorsque plusieurs opérateurs doivent enregistrer des contacts similaires sans ressaisir l’ensemble du formulaire.

L’opérateur actif est enregistré dans le champ ADIF `OPERATOR`.

### Logs personnels et de club

1. Ouvrez **F9 → Logbooks**.
2. Appuyez sur **Ins** pour créer chaque log.
3. Dans **QSO form**, appuyez sur **Ctrl+L** pour changer de log actif.
4. Vérifiez le log actif dans la barre d’état avant l’enregistrement.

Chaque log peut conserver ses propres informations de station, réglages Wavelog, réglages de concours et opérateurs.

### Plusieurs équipements

1. Ouvrez **F9 → Rigs**.
2. Appuyez sur **Ins** pour créer des presets d’équipement.
3. Sélectionnez le backend : None, flrig ou Hamlib.
4. Dans **QSO form**, appuyez sur **Ctrl+R** pour changer d’équipement actif.

Un preset d’équipement peut contenir le backend, le modèle, l’antenne, la puissance, les réglages du rotor et les réglages UDP de WSJT-X.

### Opération numérique avec WSJT-X

Lorsque l’intégration UDP de WSJT-X est activée, CQOps peut recevoir les messages ADIF de WSJT-X et enregistrer automatiquement les QSO numériques terminés.

Les QSO enregistrés automatiquement :

- sont sauvegardés dans le log actif,
- apparaissent immédiatement dans **Recent QSOs**,
- ignorent les doublons,
- héritent de la contest ID active,
- peuvent être envoyés automatiquement à Wavelog lorsque celui-ci est configuré et accessible.

Si l’opérateur indiqué par WSJT-X ne correspond pas à l’opérateur actif dans CQOps, un avertissement est affiché.

Avant une longue session numérique, vérifiez :

- le log actif,
- l’opérateur actif,
- le concours actif,
- le libellé d’état **WSJT**.

### Synchronisation Wavelog

CQOps enregistre toujours les QSO localement en premier. La synchronisation Wavelog est facultative.

| Action | Where | Shortcut | Notes |
|---|---|---|---|
| Upload unsent QSOs | **Logbook Editor** | `w` | Envoi par lots de 50 |
| Download from Wavelog | **Logbook Editor** | `Ctrl+W` | Téléchargement incrémental avec `last_fetched_id` |

L’état d’envoi est suivi pour chaque QSO :

- not sent,
- sent,
- error.

En cas d’échec de l’envoi, le QSO reste dans le log local et l’opération peut être réessayée ultérieurement. Le purge d’un log réinitialise la fetch ID à `0`, ce qui permet un téléchargement complet.

---

<a id="qso-logging"></a>
## Enregistrement des QSO

**QSO form** est l’écran principal d’enregistrement. Ouvrez-le avec **F1**.

CQOps peut remplir les champs à partir des sources suivantes :

| Source | Champs |
|---|---|
| flrig / Hamlib | Frequency, Freq RX en split, Mode, Submode |
| Callbook (QRZ.com / HamQTH / QRZ.RU / Callook.info) | Name, QTH, Grid, Country, CQ zone, ITU zone, DXCC, Continent, photo |
| Base REF | Références SOTA, POTA, WWFF et IOTA |
| Wavelog lookup | État worked/confirmed si configuré |
| Données DXCC/préfixes | Données liées au préfixe et au pays |

### Disposition du formulaire

| Colonne gauche | Colonne centrale | Colonne droite |
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

Les champs **Exch sent** et **Exch rcvd** n’apparaissent que lorsqu’un concours est actif.

La ligne inférieure contient :

- **Comment**,
- **Keep** — conserve le champ **Comment** entre les QSO,
- **Retain** — conserve l’ensemble du formulaire après l’enregistrement.

Les champs tels que **Band**, **Mode** et **Submode** peuvent être parcourus avec **PgUp/PgDn**.

### Trajet, azimut et indicateurs

Lorsque les deux grid locator sont connus, CQOps affiche la distance et l’azimut.

**QSO form** peut également afficher des indicateurs comme :

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Enregistrement

| Key | Action |
|---|---|
| Enter | Enregistrer le QSO |
| Ctrl+S | Envoyer un spot DX depuis le formulaire rempli |
| Esc | Annuler la confirmation de doublon |
| Enter dans la confirmation DUPE | Enregistrer malgré tout le doublon |

---

<a id="logbook-editor-and-adif"></a>
## Éditeur de log et ADIF

Ouvrez **Logbook Editor** avec **F8**.

Il permet de :

- consulter les QSO,
- effectuer des modifications directes,
- supprimer des QSO,
- importer des fichiers ADIF,
- exporter des fichiers ADIF,
- envoyer vers Wavelog,
- télécharger depuis Wavelog,
- exécuter les opérations liées aux concours.

### Modifier les QSO

1. Sélectionnez une ligne avec **↑/↓**.
2. Appuyez sur **Enter** ou **e**.
3. Modifiez le QSO.
4. Enregistrez avec **Ctrl+S**.

Les modifications apparaissent immédiatement dans **Recent QSOs**.

### Import et export ADIF

CQOps prend en charge l’import et l’export ADIF 3.1.7.

| Action | Shortcut |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

L’import valide les enregistrements, ignore les doublons et affiche un résumé. Les QSO importés sont marqués pour l’envoi à Wavelog si la synchronisation Wavelog est configurée.

L’export peut contenir tous les QSO ou uniquement ceux filtrés par concours. `CONTEST_ID` est conservé.

### Traitement des modes numériques

Le traitement de mode et submode suit ADIF 3.1.7 comme décrit dans ce manuel :

- FT8 est exporté comme mode autonome.
- FT4 et FT2 sont exportés comme MFSK avec le submode approprié.
- Les anciens enregistrements MFSK + FT8 importés sont normalisés en FT8 autonome.

**QSO form** comporte des champs séparés **Mode** et **Submode**. Les deux peuvent être parcourus avec **PgUp/PgDn**.

---

<a id="contests"></a>
## Concours

CQOps comprend un panneau léger de logging de concours destiné à une **participation occasionnelle aux concours**. Il ne remplace pas les loggers spécialisés comme N1MM, Win-Test ou TR4W. Pour une participation sérieuse en multi-op, multi-radio ou catégorie assisted, utilisez un logger de concours dédié. CQOps convient lorsque vous souhaitez donner quelques points, suivre votre rate pour le plaisir ou enregistrer quelques QSO de concours pendant une activation SOTA/POTA sans quitter votre logger habituel.

<a id="setting-up-a-contest"></a>
### Configurer un concours

Créez ou configurez un concours dans **Logbook Editor** avec **Ins**.

La configuration du concours comprend :

- le nom du concours,
- la date,
- l’ADIF contest ID,
- les modèles d’échange.

#### Marqueurs de modèle

| Marker | Replaced with |
|---|---|
| `@rst` | RST envoyé ou reçu |
| `@serial` | Numéro de série incrémenté automatiquement |
| `@cqz` | Zone CQ de la station DX |
| `@mycqz` | Votre zone CQ |
| `@itu` | Zone ITU de la station DX |
| `@myitu` | Votre zone ITU |
| `@grid` | Grid de la station DX |
| `@mygrid` | Votre grid |

Appuyez sur **Ctrl+C** pour parcourir les concours actifs ou sélectionnez-en un dans le menu **Contest** (**F7**). Les champs d’échange apparaissent automatiquement dans **QSO form** et les numéros de série s’incrémentent automatiquement.

<a id="bottom-status-bar"></a>
### Barre d’état inférieure

Lorsqu’un concours est actif, la barre inférieure affiche un résumé en temps réel :

```
 IARU-HF · IARU HF   45 QSOs   Started 16:13   Last 14:04 ago   Next #45   On 2:41
```

| Champ | Signification |
|-------|---------|
| `IARU-HF` | ADIF ID du concours, c’est-à-dire son identifiant exploitable par machine |
| `· IARU HF` | Nom affiché du concours, visible lorsqu’il diffère de l’ID |
| `45 QSOs` | Nombre total de QSO enregistrés pendant cette session de concours |
| `Started 16:13` | Heure du premier QSO de concours de la journée |
| `Last 14:04 ago` | Temps écoulé depuis le dernier QSO de concours |
| `Next #45` | Numéro de série qui sera envoyé lors du prochain QSO |
| `On 2:41` | Temps total d’activité — somme des intervalles entre QSO inférieurs à 30 minutes |

Le champ `Started` est masqué sur les terminaux de moins de 120 colonnes. Le nom du concours et le temps `On` sont masqués en dessous de 100 colonnes.

<a id="contest-statistics-panel"></a>
### Panneau de statistiques du concours

Lorsqu’un concours est actif et que le terminal est suffisamment large, un panneau compact bordé de jaune apparaît à droite de **QSO form** :

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

| Ligne | Champ | Signification |
|-----|-------|---------|
| **Rate** | `2/h` | Rate sur les **10 derniers QSO** — vitesse de courte rafale |
| | `--/h` | Rate sur les **100 derniers QSO** — affiche `--` jusqu’à 100 QSO |
| **Count** | `60m 0` | QSO enregistrés pendant les 60 dernières minutes |
| | `hr 0` | QSO enregistrés pendant l’heure en cours depuis `:00` |
| **Peak** | `1m120` | Meilleur rate sur 1 minute : 120/h correspond à 2 QSO dans cette minute |
| | `10m 54` | Meilleure fenêtre glissante de 10 minutes : moyenne de 54/h |
| | `60m 29` | Meilleure fenêtre glissante de 60 minutes : moyenne de 29/h |
| **Avg** | `8/h` | Moyenne de session — total des QSO divisé par le nombre d’heures depuis le premier QSO |
| | `Sess 5:36` | Durée totale de la session du premier au dernier QSO, en H:MM ou en minutes |
| **Chart** | `max 1` | La minute la plus chargée comportait 1 QSO. Les barres affichent les QSO par minute |
| | `-60m…now` | Bord gauche = il y a 60 minutes ; bord droit = maintenant |

Le graphique utilise des blocs Unicode (`█`) répartis sur quatre lignes de barres verticales. Les rates **Peak** omettent le suffixe `/h`, puisque **Peak** implique déjà « par heure ». Les durées omettent les secondes, qui ne feraient qu’ajouter du bruit avec une actualisation à la minute.

<a id="contest-adif-export"></a>
### Export ADIF du concours

Pour soumettre votre log de concours, ouvrez **Logbook Editor** (`Ctrl+E`) lorsqu’un concours est actif. Lorsqu’un filtre de concours est appliqué, la boîte d’export ADIF propose d’exporter **uniquement les QSO appartenant au concours actif**. Le résultat est un fichier ADIF 3.1.7 conforme, avec les champs d’échange, les numéros de série et l’ADIF ID du concours conservés, prêt à être envoyé au robot de l’organisateur ou à son système de contrôle des logs.

<a id="contest-mode-behavior"></a>
### Comportement du mode concours

Lorsqu’un concours est actif :

- **QSO form** affiche les champs d’échange,
- les numéros de série s’incrémentent automatiquement,
- **Recent QSOs** peut être filtré sur les QSO du concours,
- l’export ADIF conserve `CONTEST_ID`,
- **QSO form**, le panneau du concours et le panneau **Solar** reçoivent une bordure jaune pour être facilement identifiables,
- les spots DXC sont vérifiés par rapport à tous les QSO du concours, et pas seulement ceux du jour, afin de signaler les doublons.

---

<a id="favorites-references-and-band-plans"></a>
## Favoris, références et plans de bandes

### Favorites

**Favorites** enregistre des presets de frequency, mode et band dans trois slots, ce qui suffit pour les fréquences d’appel les plus utilisées. Les raccourcis utilisent `Alt` pour éviter les conflits avec les touches d’édition habituelles du terminal et fonctionner de manière fiable dans différents terminaux.

| Shortcut | Action |
|---|---|
| Alt+Ins / Alt+Home / Alt+PgUp | Rappeler un favorite du slot 1, 2 ou 3 |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Enregistrer les frequency, mode et band actuels dans le slot 1, 2 ou 3 |

**Favorites** est enregistré dans la configuration et partagé entre les logs.

Exemple :

1. Saisissez `145.55`.
2. Réglez **Mode** sur `FM`.
3. Réglez **Band** sur `2m`.
4. Appuyez sur **Alt+Shift+Ins** pour enregistrer le preset dans le slot 1.
5. Plus tard, appuyez sur **Alt+Ins** pour le rappeler.

### REF Lookup

Ouvrez **REF Lookup** avec **F6**.

La recherche porte sur :

- SOTA,
- POTA,
- WWFF,
- IOTA.

Vous pouvez rechercher par préfixe, nom ou identifiant de référence. Les références sélectionnées peuvent remplir **QSO form**.

### Band Plan Browser

Ouvrez **Band Plan Browser** avec **F7**.

Il offre un accès rapide à :

- Amateur bands,
- VHF/UHF ranges,
- CB,
- PMR446,
- Broadcast presets,
- Portable — fréquences courantes pour les opérations portables et sur le terrain, notamment SOTA, POTA et les canaux d’appel.

Une fréquence sélectionnée peut être utilisée pour régler l’équipement actif. Les données du plan de bandes peuvent également être exportées au format Markdown.

---

<a id="integrations"></a>
## Intégrations

Toutes les intégrations sont facultatives. L’enregistrement local fonctionne sans elles.

### Callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info)

CQOps prend en charge plusieurs fournisseurs de callbook avec cascade par priorité.
Lorsque vous appuyez sur **Ins** dans le formulaire QSO, les fournisseurs sont
interrogés dans l'ordre jusqu'à ce que l'un d'eux renvoie un résultat :

1. **QRZ.com** — nécessite Internet et un abonnement QRZ XML. Données les plus complètes.
2. **HamQTH** — service mondial gratuit. Bonne couverture, nécessite un compte gratuit.
3. **QRZ.RU** — service gratuit axé sur la Russie et les pays voisins. Nécessite un login API. Fournit nom, QTH, grid, lat/lon, classe, statut LoTW/eQSL et photo.
4. **Callook.info** — service gratuit axé sur les États-Unis. Aucun compte requis, recherches FCC rapides.

Si un fournisseur de priorité supérieure échoue ou est désactivé, le suivant est
essayé. Lorsque **Base call fallback** est activé (par défaut : oui), CQOps
essaie également l'indicatif de base (sans préfixe ni suffixe) si l'indicatif
complet ne donne aucun résultat.

Activez et configurez les fournisseurs dans **F9 → Callbook**.

Dans **QSO form**, appuyez sur **Ins** pour remplir des champs du callbook tels que :

- Name,
- QTH,
- Grid,
- Country,
- CQ/ITU zones,
- DXCC,
- Continent.

**Partner view** sous **F2** peut afficher la photo de l’opérateur lorsqu’elle est disponible.

> ⚠️ **Experimental.** L’affichage des photos peut utiliser le protocole Kitty
> terminal graphics et nécessite un terminal compatible : Kitty, Ghostty ou
> WezTerm. Activez-le sous **F9 → General → Kitty Graphics**. Les terminaux
> standard et les sessions SSH sans transmission graphique utiliseront une
> image de remplacement composée de glyphes.

### Wavelog

L’intégration Wavelog prend en charge :

- l’envoi,
- le téléchargement incrémental,
- la recherche de l’état worked/confirmed.

Wavelog se configure par log actif avec :

- URL,
- API key,
- station profile ID.

CQOps enregistre toujours les QSO localement en premier. Un échec de l’envoi Wavelog ne supprime pas les données locales.

### flrig

L’intégration flrig utilise XML-RPC sur HTTP.

Endpoint par défaut :

```text
localhost:12345
```

CQOps peut lire :

- frequency,
- mode,
- power.

En fonctionnement split, VFO A correspond à **Frequency** et VFO B à **Freq RX**.

### Hamlib / rigctld

Le contrôle du poste par Hamlib utilise le daemon TCP `rigctld`.

Selon le poste et les possibilités du backend, CQOps peut interroger :

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps gère proprement l’absence de prise en charge des noms de VFO lorsque cela est possible.

### Hamlib Rotator / rotctld

> ⚠️ **Experimental.** Le contrôle du rotor est expérimental. Vérifiez toujours
> les limites physiques de votre antenne avant toute utilisation. Soyez prêt à
> arrêter immédiatement le mouvement avec **Alt+/**. Utilisez cette fonction
> avec prudence : une mauvaise configuration peut endommager le rotor ou
> l’antenne.

Le contrôle du rotor utilise Hamlib `rotctld`.

CQOps prend en charge :

- azimuth,
- elevation,
- stop commands.

| Shortcut | Action |
|---|---|
| Alt+, | Modifier azimuth de −5° |
| Alt+. | Modifier azimuth de +5° |
| Alt+; | Modifier elevation de +5° |
| Alt+' | Modifier elevation de −5° |
| Alt+\ | Orienter le rotor vers le cap de trajet calculé |
| Alt+/ | Arrêter le rotor |

### WSJT-X

L’intégration WSJT-X utilise les messages UDP de WSJT-X. CQOps analyse les messages ADIF et peut enregistrer automatiquement les QSO terminés.

Le libellé de l’équipement prend la couleur d’accent lorsque WSJT-X émet. Si l’opérateur indiqué par WSJT-X ne correspond pas à l’opérateur actif, CQOps affiche un avertissement.

### GPS

CQOps peut lire la position d’un récepteur GPS et l’utiliser comme grid locator de la station — idéal pour les opérations portables, mobiles ou sur le terrain.

Deux backends sont pris en charge :

- **Serial** — connexion directe à un récepteur GPS par port série, par exemple USB-to-serial, port COM intégré ou `/dev/ttyUSB0`.
- **GPSD** — connexion à un serveur [gpsd](https://gpsd.io/) par TCP, par défaut `127.0.0.1:2947`. Utile lorsque le GPS est partagé avec d’autres applications ou accessible par le réseau.

L’indicateur **GPS** dans la barre d’état affiche :

| Couleur | Signification |
|--------|---------|
| Rouge `GPS` | Déconnecté / erreur |
| Jaune `GPS` | Connecté, pas encore de fix |
| Blanc `GPS` | Fix acquis, position verrouillée |

Lorsqu’un fix est obtenu, le grid locator de la station est remplacé par la position calculée à partir du GPS et marqué `(GPS)` dans la ligne d’état :

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Activez **Grid from GPS** dans les réglages **Station & Logbook** pour utiliser le grid GPS dans l’enregistrement des QSO, les balises APRS, la carte du dashboard et les calculs de distance.

**Grid precision** se configure dans le menu **Integration** avec 10, 8 ou 6 caractères. La valeur par défaut est 10 caractères, soit environ 25 m de précision. Le grid est toujours calculé en interne à pleine précision puis tronqué au niveau de son utilisation.

### DX Cluster

L’intégration **DX Cluster** utilise telnet et nécessite un accès à Internet.

Serveur par défaut :

```text
dxspots.com:7300
```

Les filtres comprennent :

- band,
- spotter continent,
- mode,
- age/time.

| Key | Action |
|---|---|
| Enter | Remplir **QSO form**, régler l’équipement et revenir à **QSO** |
| Space | Régler l’équipement et rester dans **DX Cluster** |
| Backspace | Effacer tous les filtres |

Lorsque **DX Cluster** est connecté, **QSO form** bénéficie de deux fonctions supplémentaires :

- **Send a spot** — lorsque le formulaire est rempli, appuyez sur **Ctrl+S** pour ouvrir la boîte de spot et envoyer un spot DX au cluster.
- **Nearest spots** — lorsqu’une fréquence est réglée, jusqu’à trois spots proches apparaissent directement dans **QSO form**, ce qui permet de voir l’activité de la bande sans quitter l’écran d’enregistrement. Appuyez sur **Ctrl+P** pour remplir l’indicatif à partir du spot le plus proche.

### PSK Reporter

L’intégration **PSK Reporter** nécessite un accès à Internet. C’est un excellent outil pour vérifier rapidement la propagation réelle : qui reçoit votre signal, ou qui recevez-vous actuellement sur une bande donnée ?

Elle fournit :

- des spots de propagation,
- des filtres band/time/mode,
- une carte du monde ASCII sous **F5**.

### APRS

CQOps prend en charge trois types de service APRS. Choisissez celui qui correspond à la configuration de votre station :

| Service | Connection | Internet required |
|---|---|---|
| **APRS-IS** | TCP vers un serveur APRS-IS | Oui |
| **KISS** | Port série vers un TNC KISS matériel | Non |
| **KISS Server** | TCP vers un serveur TNC KISS, par exemple Dire Wolf | Non, le réseau local suffit |

Sélectionnez le type de service dans le menu **Integrations** :

```text
F9 → Integrations → APRS → Service (Space to cycle)
```

Les trois services prennent en charge la réception des rapports de position APRS des stations proches et leur affichage sur la carte locale de **CQOps Live** avec :

- les symboles APRS standard,
- des popups de callsign,
- l’ajustement automatique de la vue,
- un cercle de portée configurable.

Tous les services prennent également en charge le **periodic position beaconing**. CQOps transmet le grid locator de la station à l’intervalle configuré. Lorsque GPS est actif et que **Grid from GPS** est activé, la balise utilise automatiquement la position issue du GPS — idéal en portable ou en mobile.

#### APRS-IS

Se connecte au réseau mondial APRS-IS par Internet. Nécessite :

- un indicatif radioamateur valide,
- un APRS-IS passcode généré à partir de l’indicatif,
- une connexion Internet.

Serveur par défaut :

```text
euro.aprs2.net:14580
```

APRS-IS est configuré globalement sous **F9 → Integrations → APRS**. Callsign, SSID, symbol, comment, beacon interval et range filter de chaque log sont configurés sous **F9 → Logbooks → [active logbook] → APRS**.

#### KISS (serial)

Se connecte directement à un TNC KISS matériel par port série. Aucune connexion Internet n’est nécessaire : les trames APRS sont émises et reçues par la radio.

Configurez serial port, baud rate, data bits, parity, stop bits et DTR/RTS dans le menu **Integrations** :

```text
F9 → Integrations → APRS → Service: KISS
```

Lorsque **KISS** est sélectionné, les champs série **Port**, **Baud**, **Data bits**, **Parity**, **Stop bits**, **DTR** et **RTS** deviennent visibles.

Le bouton **Test** ouvre le port série afin de vérifier que le TNC est accessible.

#### KISS Server (TCP)

Se connecte à un TNC KISS accessible en TCP, par exemple une instance [Dire Wolf](https://github.com/wb2osz/direwolf) exécutée sur la même machine ou sur le réseau local. Aucune connexion Internet n’est nécessaire.

Saisissez l’hôte et le port dans le menu **Integrations** :

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Valeurs par défaut : `127.0.0.1:8001`

#### Beaconing

Les balises sont envoyées selon l’intervalle configuré pour chaque log. L’intervalle minimal est de 1 minute. La balise contient :

- le callsign de la station avec SSID,
- le grid locator issu du GPS lorsqu’il est disponible,
- le symbol APRS,
- un comment facultatif.

Lorsque **GPS** est actif et que **Grid from GPS** est activé dans les réglages **Station**, la balise utilise automatiquement le grid locator calculé à partir du GPS. Aucune mise à jour manuelle du grid n’est nécessaire pendant un déplacement.

Beacon interval et les autres réglages propres au log sont configurés sous :

```text
F9 → Logbooks → [active logbook] → APRS
```

#### Receiving

Les rapports de position APRS reçus sont conservés en cache local et affichés sur la carte du **CQOps Live dashboard**. Les stations sont représentées par leurs symboles APRS et peuvent être sélectionnées pour afficher les détails. La vue s’ajuste automatiquement afin de montrer toutes les stations visibles dans la portée configurée.

La réception APRS est indépendante de l’émission des balises : vous pouvez recevoir sans émettre, et inversement. Activez simplement APRS dans **Integrations** et choisissez le type de service.

### Solar Data

Les données solaires proviennent de hamqsl.com et comprennent :

- SFI,
- sunspot number,
- A/K indices,
- les conditions bande par bande.

Les mises à jour en direct nécessitent un accès à Internet. Les données en cache restent disponibles hors ligne après un téléchargement réussi.

---

<a id="cqops-live-dashboard"></a>
## CQOps Live Dashboard

CQOps Live est un dashboard intégré au navigateur qui affiche en temps réel l’activité de la station.

Il est utile pour :

- les affichages publics pendant les field days,
- les écrans de station de club,
- la surveillance d’un concours,
- l’observation de la station depuis une autre pièce,
- les stands lors d’événements ou de salons.

### Activer le dashboard

1. Appuyez sur **F9**.
2. Ouvrez **Integrations**.
3. Accédez à **HTTP Server**.
4. Activez **HTTP server**.
5. Définissez éventuellement address et port.
6. Appuyez sur **Ctrl+S** pour enregistrer.
7. Ouvrez le dashboard dans un navigateur.

Réglages par défaut :

| Setting | Default |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

Le serveur démarre immédiatement après l’enregistrement.

> **Address binding:** La valeur par défaut `0.0.0.0` rend le dashboard accessible depuis tout appareil du réseau local. Cela est utile pour les écrans de field day, les stations de club ou la surveillance depuis une autre pièce. Réglez address sur `127.0.0.1` pour limiter l’accès à la machine locale.

### Modes d’affichage

CQOps Live possède deux modes d’affichage.

#### Overview mode

Affiché lorsqu’aucun callsign actif n’est en cours de contact.

Il présente :

- **live maps** — les marqueurs des QSO du jour avec des trajets orthodromiques depuis le grid de la station vers chaque correspondant, ainsi qu’une carte APRS locale montrant les stations proches,
- le tableau des recent QSOs,
- les informations de station,
- les statistiques,
- le suivi du rate sur 5 minutes, 15 minutes et 1 heure,
- les meilleurs opérateurs,
- les QSO les plus lointains.

#### Active / Now Working mode

Affiché lorsqu’un callsign est en cours de contact.

Il présente :

- un callsign en grand,
- l’indicateur submode,
- la photo QRZ lorsqu’elle est disponible,
- les indicateurs band et mode,
- les indicateurs **DUPE / NEW CALL / NEW DXCC**,
- la distance et l’azimut,
- un trajet en pointillés mis en évidence sur la carte entre le grid de la station et celui du correspondant.

### Info box

L’**Info box** au-dessus de la carte locale change de module toutes les 5 secondes :

- conditions des bandes,
- activité solaire,
- champ géomagnétique,
- dernier spot DX Cluster,
- nombre de rapports PSK Reporter par bande.

### Weather row

La **Weather row** affiche les conditions Open-Meteo actuelles pour le grid locator de la station :

- température,
- vent,
- humidité,
- icône.

Les données météo sont récupérées dans le navigateur et sont simplement omises lorsque la connexion est indisponible.

### Local map

La **local map** de droite est consacrée à la **surveillance du voisinage APRS**. Elle peut afficher :

- les stations APRS proches avec les symboles standard,
- des popups de callsign au survol ou au clic,
- un cercle de portée configurable,
- un overlay facultatif du terminateur jour/nuit,
- un overlay facultatif du radar météo RainViewer.

### Mises à jour en temps réel et performances

CQOps Live se met à jour par Server-Sent Events (SSE). Aucun rechargement de page n’est nécessaire.

Le dashboard est conçu pour le matériel peu puissant :

- le navigateur effectue le rendu des cartes,
- le navigateur calcule les distances,
- le navigateur calcule les statistiques,
- CQOps envoie des mises à jour JSON légères,
- lorsque **HTTP server** est désactivé, aucun port n’est ouvert et aucune goroutine du dashboard ne s’exécute.

### Personnaliser le dashboard

Dans le formulaire d’intégration **HTTP Server**, vous pouvez configurer :

| Field | Description |
|---|---|
| Header 1 | Titre principal de l’en-tête et de la zone hero. Utilise « CQOps Live » par défaut. |
| Header 2 | Sous-titre sous le titre. Utilise « Fast, portable ham radio logger » par défaut. |
| Logo URL | URL publique de l’image affichée en haut à gauche. Utilise le logo CQOps par défaut. |
| Event Start | Date au format `YYYY-MM-DD`. Filtre les statistiques et listes de QSO à partir de cette date. |

---

<a id="configuration"></a>
## Configuration

Ouvrez la configuration avec **F9**.

### Fichiers de configuration

| Plateforme | Chemin de configuration |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Les identifiants sensibles sont enregistrés séparément dans `secrets.enc`, dans le même répertoire de configuration.

Les secrets sont chiffrés avec une clé liée à la machine. Lors du transfert de la configuration vers une autre machine, les identifiants doivent être saisis à nouveau.

### Menus de configuration

Appuyez sur **F9** pour ouvrir le menu principal, puis sélectionnez :

| Menu | Configures |
|---|---|
| General | Units, timezone, partner map/picture, solar panel, sources SCP/REF, Kitty Graphics, Debug mode |
| Logbooks | Station callsign, grid, references, CQ/ITU zones, IARU region, GPS grid ; Wavelog par log (URL, API key, station profile) ; APRS par log (callsign, symbol, beacon, range) |
| Operators | Profils operator callsign et operator name pour les stations multi-opérateurs |
| Rigs | Presets d’équipement : model, antenna, power, backend (None/flrig/Hamlib), rotor, WSJT-X UDP |
| Contests | Profils de concours : name, date, ADIF contest ID, exchange templates, starting serial number |
| Integration | DX Cluster (host, port, login), HTTP Server du dashboard (address, port, branding), GPS service (serial/GPSD, grid precision) |
| Callbook | Fournisseurs QRZ.com, HamQTH, QRZ.RU, Callook.info ; ordre de priorité, base-call fallback, Wavelog lookup |
| Notifications | QSO saved alerts, Wavelog QSO sent status, dupe beep, error sounds |

### Multi-logbook

Utilisez plusieurs logs pour home, portable, contest et club.

Appuyez sur **Ctrl+L** pour parcourir les logs actifs.

Chaque log conserve ses propres :

- informations de station,
- réglages Wavelog,
- réglages de concours,
- réglages d’opérateur.

### Multi-operator

Les profils opérateur contiennent :

- operator callsign,
- operator name.

Appuyez sur **Ctrl+O** pour parcourir les opérateurs actifs.

L’opérateur actif est enregistré dans le champ ADIF `OPERATOR` et transmis lors des envois vers Wavelog.

### Multi-rig

Les presets d’équipement enregistrent :

- backend,
- model,
- antenna,
- power,
- réglages du rotor,
- réglages WSJT-X.

Appuyez sur **Ctrl+R** pour parcourir les équipements actifs.

### Secrets chiffrés

Depuis la version v0.8.7, les identifiants sont enregistrés sous forme chiffrée.

| Élément | Valeur |
|---|---|
| Fichier des secrets | `secrets.enc` |
| Emplacement | Même répertoire que `config.yaml` |
| Permissions Unix | `0600` lorsque pris en charge |
| Chiffrement | AES-256-GCM avec une clé liée à la machine |
| Données protégées | QRZ password, DX Cluster login, Wavelog API keys |

Les secrets en clair provenant d’anciennes configurations sont migrés au premier démarrage.

Si `secrets.enc` est endommagé, CQOps démarre avec un avertissement et demande de saisir à nouveau les identifiants.

---

<a id="keyboard-shortcuts"></a>
## Raccourcis clavier

### Global

| Key | Action |
|---|---|
| F1 | **QSO form** et **Recent QSOs** |
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
## Dépannage

### CQOps ne démarre pas

Vérifiez :

- que le terminal mesure au moins 80×24 caractères,
- que les utilisateurs de Windows utilisent Windows Terminal,
- que le démarrage réseau ne bloque pas, en essayant :

  ```bash
  cqops --offline
  ```

Consultez les logs :

| Plateforme | Chemin des logs |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### L’équipement ne se connecte pas

Pour flrig :

- vérifiez que flrig est en cours d’exécution,
- vérifiez le port dans le preset actif,
- le port par défaut est `12345`.

Pour Hamlib :

- vérifiez que `rigctld` est en cours d’exécution,
- vérifiez host et port,
- vérifiez que l’équipement/backend prend en charge les données demandées.

Les libellés d’état facilitent le diagnostic :

| Couleur | Signification |
|---|---|
| Blanc/par défaut | Connecté |
| Jaune | Désactivé ou en cours de connexion |
| Rouge | Échec |

Les toasts de reconnexion peuvent être masqués. CQOps peut effectuer silencieusement de nouvelles tentatives.

### WSJT-X n’enregistre pas automatiquement

Vérifiez :

- **WSJT-X Settings → Reporting → UDP Server**,
- que UDP host et port correspondent au preset actif dans CQOps,
- que WSJT-X 2.6 ou une version ultérieure est utilisé,
- que le libellé d’état **WSJT** est actif,
- que le log actif est correct,
- que l’opérateur actif est correct.

### L’envoi Wavelog échoue

Vérifiez :

- Wavelog URL,
- API key,
- station profile ID,
- le libellé d’état **WL**.

Les erreurs d’envoi sont affichées sous forme de toasts. Les QSO restent enregistrés localement même si l’envoi échoue. L’échec d’un QSO individuel ne bloque pas le reste du lot.

### Problèmes liés au fichier de configuration

Fichier de configuration :

| Plateforme | Chemin |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Fichier des secrets :

```text
secrets.enc
```

Le fichier des secrets est enregistré dans le même répertoire que `config.yaml`.

Si la configuration est endommagée, déplacez ou supprimez le fichier puis redémarrez CQOps. L’assistant de configuration créera une nouvelle configuration.

Le champ `last_fetched_id` n’apparaît qu’après un téléchargement Wavelog réussi.

### Problèmes de performances

Essayez de :

- désactiver le rendu de la carte dans **General**,
- désactiver le panneau **Solar** s’il n’est pas nécessaire,
- éviter les écrans utilisant beaucoup le réseau, comme **DX Cluster** et **PSK Reporter**, lorsque vous travaillez hors ligne,
- utiliser `cqops --offline` lorsque le réseau n’est pas fiable.

---

<a id="reporting-bugs"></a>
## Signaler des bogues

Avant de signaler un bogue :

1. Activez **Debug mode** dans **F9 → General → Debug**, ou définissez :

   ```yaml
   debug: true
   ```

   dans `config.yaml`.

2. Reproduisez le problème.
3. Joignez le log pertinent.

Signalez les problèmes sur GitHub :

<https://github.com/szporwolik/cqops/issues>

Indiquez :

- la version de CQOps obtenue avec `cqops --version`,
- le système d’exploitation,
- l’émulateur de terminal,
- les étapes permettant de reproduire le problème,
- le debug log pertinent.
