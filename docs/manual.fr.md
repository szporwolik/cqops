---
title: Manuel utilisateur CQOps
description: Guide d'installation, de configuration et d'utilisation de CQOps — un logger radioamateur rapide, orienté terminal
---

> **Note de traduction :** Cette traduction a été générée avec un modèle LLM. Les corrections sont bienvenues sous forme de Pull Request vers la branche `dev`. Certains noms d'écrans, de champs, de commandes et de raccourcis restent volontairement en anglais afin de correspondre à l'interface CQOps.

# Manuel utilisateur CQOps

CQOps est un logger radioamateur rapide, orienté terminal, pour les opérateurs qui veulent une journalisation fiable au clavier avec une faible charge système. Il est conçu pour le shack, l'opération portable, les stations de club, les field days et les machines comme les Raspberry Pi ou les anciens ordinateurs portables.

CQOps enregistre toujours les QSOs localement en premier. Les intégrations basées sur internet sont optionnelles.

## Sommaire

1. [Qu'est-ce que CQOps](#quest-ce-que-cqops)
2. [Téléchargement et installation](#téléchargement-et-installation)
3. [Premier lancement](#premier-lancement)
4. [Enregistrer votre premier QSO](#enregistrer-votre-premier-qso)
5. [Écran principal](#écran-principal)
6. [Flux de travail courants](#flux-de-travail-courants)
7. [Journalisation QSO](#journalisation-qso)
8. [Logbook Editor et ADIF](#logbook-editor-et-adif)
9. [Concours](#concours)
10. [Favorites, références et plans de bande](#favorites-références-et-plans-de-bande)
11. [Intégrations](#intégrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Configuration](#configuration)
14. [Raccourcis clavier](#raccourcis-clavier)
15. [Dépannage](#dépannage)
16. [Signaler des bugs](#signaler-des-bugs)

---

## Qu'est-ce que CQOps

CQOps est construit autour de la saisie rapide des QSOs, de la journalisation locale et de l'opération pratique sur le terrain.

### Idées principales

- **Terminal-first** — optimisé pour l'utilisation au clavier.
- **Offline-first** — la journalisation locale des QSOs fonctionne sans accès internet.
- **Faible charge** — adapté aux systèmes de classe Raspberry Pi, aux anciens portables et aux PCs de station partagée.
- **Conception portable** — distribué comme un seul binaire Go.
- **Plusieurs logbooks** — utile pour les logs personnels, portables, concours et club.
- **Plusieurs opérateurs** — utile pour les flux hot-seat et les stations de club partagées.
- **Plusieurs rigs** — chaque preset de rig peut garder son propre backend et ses paramètres WSJT-X.
- **Intégrations optionnelles** — QRZ.com, Wavelog, DX Cluster, PSK Reporter, APRS, récepteur GPS, contrôle de rig, contrôle de rotor, données solaires et CQOps Live dans le navigateur.

La journalisation locale ne nécessite pas d'accès internet. Les fonctions réseau sont ignorées en mode `--offline`.

### À qui s'adresse CQOps

CQOps convient bien aux :

- opérateurs portables,
- activateurs SOTA et POTA,
- stations de club,
- équipes de field day,
- opérateurs qui préfèrent un flux de travail en terminal,
- stations qui doivent changer rapidement d'opérateur, de logbook ou de rig.

CQOps n'a pas vocation à remplacer toutes les fonctions d'un logger de bureau complet ou d'une plateforme web de logbook. Il se concentre sur la journalisation rapide en terminal, l'opération sur le terrain, l'utilisation hors-ligne et les stations partagées.

---

## Téléchargement et installation

Toutes les versions :

<https://github.com/szporwolik/cqops/releases>

### Windows

| Paquet | Lien | Notes |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recommandé pour la plupart des utilisateurs. Ajoute CQOps au Start Menu et au PATH. |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extraire et exécuter sans installation. |

Utilisez **Windows Terminal** plutôt que l'ancienne console.

### Linux — Debian / Ubuntu

| Architecture | Lien | Utilisation |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La plupart des PCs Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Systèmes ARM 64 bits |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS 32 bits |

Installer le paquet téléchargé :

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — tarball portable

| Architecture | Lien | Utilisation |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La plupart des PCs Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Systèmes ARM 64 bits |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS 32 bits |

### macOS

| Architecture | Lien | Utilisation |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Macs M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Macs Intel |

Installation manuelle :

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Construire depuis les sources

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

La construction depuis les sources nécessite Go 1.26 ou plus récent.

### Exigences du terminal

| Exigence | Valeur |
|---|---|
| Taille minimale du terminal | 80×24 caractères |
| Taille recommandée | 80×43 caractères ou plus |
| Terminal recommandé sous Windows | Windows Terminal |
| Terminal pour graphiques Kitty | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) ou [WezTerm](https://wezfurlong.org/wezterm/) |

### Commandes de base

```bash
cqops              # Lancer la TUI
cqops --offline    # Lancer sans activité réseau
cqops --version    # Afficher la version et quitter
cqops --help       # Afficher l'aide
```

---

## Premier lancement

Au premier lancement, CQOps ouvre le setup wizard. Seules les informations essentielles de station sont nécessaires pour la journalisation locale. Les intégrations réseau peuvent être ignorées et configurées plus tard.

### Pages du wizard

| Page | Ce qu'elle configure |
|---|---|
| Station & Logbook | Logbook initial, indicatif de station, opérateur, grid locator, références et zones optionnelles, Wavelog URL/API/station profile ID |
| Rig | Preset de rig, modèle, antenne, puissance, backend, rotor optionnel, paramètres UDP WSJT-X optionnels |
| Integrations | Paramètres QRZ.com lookup |
| General | Fuseau horaire IANA |
| Summary | Vérification et sauvegarde |

Backends de rig supportés :

- None,
- flrig,
- Hamlib `rigctld`.

### Navigation dans le wizard

| Touche | Action |
|---|---|
| Ctrl+S | Valider et continuer ; sur Summary, sauvegarder et lancer CQOps |
| Esc | Revenir en arrière |
| F10 | Quitter |
| Tab / Shift+Tab | Passer d'un champ à l'autre |
| Space | Basculer une case à cocher |

Les paramètres du wizard peuvent être modifiés plus tard avec **F9**.

---

## Enregistrer votre premier QSO

1. Lancez CQOps :

   ```bash
   cqops
   ```

2. Terminez le setup wizard avec au moins votre indicatif et votre grid locator.
3. Ouvrez le QSO form avec **F1**.
4. Saisissez l'indicatif du contact. CQOps met automatiquement les indicatifs en majuscules.
5. Remplissez les autres champs. Si le rig actif est connecté via flrig ou Hamlib, CQOps peut remplir automatiquement la fréquence, la bande, le mode et le submode.
6. Appuyez sur **Enter** ou **Ctrl+S** pour sauvegarder.
7. Si l'avertissement **DUPE!** apparaît, appuyez de nouveau sur **Enter** pour enregistrer quand même, ou **Esc** pour annuler.

Le QSO sauvegardé apparaît immédiatement dans la table Recent QSOs.

---

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

La Status bar affiche :

- version de CQOps,
- logbook actif,
- rig actif,
- indicatif de station,
- opérateur actif,
- étiquettes de statut des intégrations,
- heure locale marquée `L`,
- heure UTC marquée `Z`.

Les étiquettes courantes incluent **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** et **GPS**. L'étiquette GPS suit la même convention de couleur — rouge quand déconnecté, jaune quand connecté sans fix, blanc quand une position est acquise.

| Couleur | Signification |
|---|---|
| Blanc/défaut | Connecté ou actif |
| Jaune | Désactivé, connexion en cours ou offline attendu |
| Rouge | Erreur ou déconnecté |
| Accent + gras | WSJT-X transmet |

### Onglets principaux

| Touche | Onglet | Écran |
|---|---|---|
| F1 | QSO | QSO form et Recent QSOs |
| F2 | QRZ | Partner view : données callbook, carte, stats, photo |
| F4 | DXC | Spots DX Cluster et filtres |
| F5 | HRD | Spots PSK Reporter et carte de propagation |
| F6 | REF | Recherche de références SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, synchronisation Wavelog |
| F9 | CFG | Menus de configuration |

La Help bar affiche les raccourcis adaptés à l'écran actif. **?** ouvre l'aide complète.

---

## Flux de travail courants

### Opération portable, SOTA ou POTA

Avant de partir :

1. Lancez CQOps une fois avec un accès internet.
2. Laissez CQOps télécharger ou rafraîchir les données en cache : données solaires, données REF et préfixes DXCC.
3. Vérifiez que le Solar panel affiche des données.
4. Vérifiez que REF search sur **F6** renvoie des résultats.

Sur le terrain :

1. Lancez CQOps en mode offline :

   ```bash
   cqops --offline
   ```

2. Journalisez normalement. Les QSOs sont sauvegardés localement.
3. Une fois de retour en ligne, ouvrez **F8** et appuyez sur **w** pour téléverser les QSOs non envoyés vers Wavelog.

### Station de club partagée et hot-seat logging

1. Ouvrez **F9 → Operators**.
2. Appuyez sur **Ins** pour ajouter des profils d'opérateur.
3. Dans le QSO form, appuyez sur **Ctrl+O** pour changer l'opérateur actif.
4. Vérifiez l'opérateur actif dans la Status bar avant de sauvegarder.
5. Utilisez **Retain** quand plusieurs opérateurs doivent journaliser des contacts similaires sans ressaisir tout le formulaire.

L'opérateur actif est sauvegardé dans le champ ADIF `OPERATOR`.

### Logbooks personnels et de club

1. Ouvrez **F9 → Logbooks**.
2. Appuyez sur **Ins** pour créer chaque logbook.
3. Dans le QSO form, appuyez sur **Ctrl+L** pour changer le logbook actif.
4. Vérifiez le logbook actif dans la Status bar avant de sauvegarder.

Chaque logbook peut conserver ses propres détails de station, paramètres Wavelog, paramètres de concours et opérateurs.

### Plusieurs rigs

1. Ouvrez **F9 → Rigs**.
2. Appuyez sur **Ins** pour créer des presets de rig.
3. Sélectionnez le backend : None, flrig ou Hamlib.
4. Dans le QSO form, appuyez sur **Ctrl+R** pour changer le rig actif.

Un preset de rig peut inclure backend, modèle, antenne, puissance, paramètres de rotor et paramètres UDP WSJT-X.

### Opération numérique WSJT-X

Quand l'intégration UDP WSJT-X est activée, CQOps peut recevoir des messages ADIF depuis WSJT-X et journaliser automatiquement les QSOs numériques terminés.

Les QSOs auto-journalisés :

- sont sauvegardés dans le logbook actif,
- apparaissent immédiatement dans Recent QSOs,
- ignorent les doublons,
- héritent du contest ID actif,
- peuvent être envoyés automatiquement vers Wavelog quand Wavelog est configuré et joignable.

Si l'opérateur indiqué par WSJT-X ne correspond pas à l'opérateur actif dans CQOps, CQOps affiche un avertissement.

Avant de longues sessions numériques, vérifiez :

- logbook actif,
- opérateur actif,
- concours actif,
- étiquette de statut WSJT-X.

### Synchronisation Wavelog

CQOps sauvegarde d'abord les QSOs localement. La synchronisation Wavelog est optionnelle.

| Action | Où | Raccourci | Notes |
|---|---|---|---|
| Envoyer les QSOs non envoyés | Logbook Editor | `w` | Envoi par lots de 50 |
| Télécharger depuis Wavelog | Logbook Editor | `Ctrl+W` | Téléchargement incrémental via `last_fetched_id` |

L'état d'envoi est suivi par QSO :

- not sent,
- sent,
- error.

Si l'envoi échoue, le QSO reste dans le logbook local et peut être réessayé plus tard. Purging un logbook remet le fetch ID à `0`, ce qui permet un téléchargement complet.

---

## Journalisation QSO

Le QSO form est l'écran principal de journalisation. Ouvrez-le avec **F1**.

CQOps peut remplir les champs depuis :

| Source | Champs |
|---|---|
| flrig / Hamlib | Frequency, Freq RX en split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| REF database | Références SOTA, POTA, WWFF, IOTA |
| Wavelog lookup | Worked/confirmed status quand configuré |
| DXCC/prefix data | Données de préfixe et de pays |

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

Les champs exchange apparaissent uniquement lorsqu'un concours est actif.

La ligne du bas contient :

- **Comment**,
- **Keep** — conserve le champ Comment entre les QSOs,
- **Retain** — conserve tout le formulaire après sauvegarde.

Les champs comme Band, Mode et Submode peuvent être parcourus avec **PgUp/PgDn**.

### Chemin, azimut et badges

Quand les deux grid locators sont connus, CQOps affiche la distance et l'azimut.

Le QSO form peut aussi afficher des badges comme :

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Sauvegarde

| Touche | Action |
|---|---|
| Enter | Sauvegarder le QSO |
| Ctrl+S | Sauvegarder le QSO depuis n'importe quel champ |
| Esc | Annuler la confirmation de doublon |
| Enter sur la confirmation DUPE | Sauvegarder le doublon quand même |

---

## Logbook Editor et ADIF

Ouvrez le Logbook Editor avec **F8**.

Il sert à :

- vérifier les QSOs,
- éditer inline,
- supprimer des QSOs,
- importer ADIF,
- exporter ADIF,
- téléverser vers Wavelog,
- télécharger depuis Wavelog,
- effectuer des opérations liées aux concours.

### Modifier les QSOs

1. Sélectionnez une ligne avec **↑/↓**.
2. Appuyez sur **Enter** ou **e**.
3. Modifiez le QSO.
4. Sauvegardez avec **Ctrl+S**.

Les changements apparaissent immédiatement dans Recent QSOs.

### Import et export ADIF

CQOps prend en charge l'import et l'export ADIF 3.1.7.

| Action | Raccourci |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

L'import valide les enregistrements, ignore les doublons et affiche un résumé. Les QSOs importés sont marqués pour l'envoi vers Wavelog lorsque la synchronisation Wavelog est configurée.

L'export peut inclure tous les QSOs ou les QSOs filtrés par concours. `CONTEST_ID` est conservé.

### Gestion des modes numériques

La gestion de mode et submode suit ADIF 3.1.7 comme décrit dans ce manuel :

- FT8 est exporté comme mode autonome.
- FT4 et FT2 sont exportés comme MFSK avec le submode approprié.
- Les anciens enregistrements MFSK + FT8 importés sont normalisés en FT8 autonome.

Le QSO form possède des champs séparés **Mode** et **Submode**. Les deux peuvent être parcourus avec **PgUp/PgDn**.

---

## Concours

Les concours ajoutent des champs exchange et la gestion du numéro de série au QSO form.

Créez ou configurez un concours dans le Logbook Editor avec **Ins**.

La configuration d'un concours comprend :

- nom du concours,
- date,
- ADIF contest ID,
- templates exchange.

### Marqueurs de template

| Marqueur | Remplacé par |
|---|---|
| `@rst` | RST envoyé ou reçu |
| `@serial` | Numéro de série auto-incrémenté |
| `@call` | Votre indicatif |
| `@grid` | Votre grid locator |
| `@name` | Nom d'opérateur depuis le profil opérateur |

Appuyez sur **Ctrl+C** pour changer le concours actif.

Quand un concours est actif :

- le QSO form affiche les champs exchange,
- les numéros de série s'incrémentent automatiquement,
- Recent QSOs peut filtrer les QSOs de concours,
- l'export ADIF conserve `CONTEST_ID`.

---

## Favorites, références et plans de bande

### Favorites

Favorites stocke les presets de fréquence, mode et band dans 10 emplacements.

| Raccourci | Action |
|---|---|
| Alt+0–9 | Rappeler un favorite |
| Alt+Shift+0–9 | Sauvegarder les frequency, mode et band actuels comme favorite |

Favorites est stocké dans la configuration et partagé entre logbooks.

Exemple :

1. Saisissez `145.55`.
2. Réglez mode sur `FM`.
3. Réglez band sur `2m`.
4. Appuyez sur **Alt+Shift+1**.
5. Plus tard, appuyez sur **Alt+1** pour rappeler le preset.

### REF Lookup

Ouvrez REF Lookup avec **F6**.

Il recherche :

- SOTA,
- POTA,
- WWFF,
- IOTA.

Vous pouvez chercher par préfixe, nom ou désignateur de référence. Les références sélectionnées peuvent remplir le QSO form.

### Band Plan Browser

Ouvrez le Band Plan Browser avec **F7**.

Il donne un accès rapide à :

- bandes radioamateur,
- plages VHF/UHF,
- CB,
- PMR446,
- presets broadcast.

Une fréquence sélectionnée peut servir à régler le rig actif. Les données de band plan peuvent aussi être exportées en Markdown.

---

## Intégrations

Toutes les intégrations sont optionnelles. La journalisation locale fonctionne sans elles.

### QRZ.com

QRZ.com lookup nécessite un accès internet et un abonnement QRZ XML.

Dans le QSO form, appuyez sur **Ins** pour remplir des champs callbook comme :

- name,
- QTH,
- grid,
- country,
- CQ/ITU zones,
- DXCC,
- continent.

La Partner view sur **F2** peut afficher la photo de l'opérateur lorsqu'elle est disponible.

> ⚠️ **Expérimental.** L'affichage des photos peut utiliser le protocole
> graphique Kitty et nécessite un terminal compatible : Kitty, Ghostty ou
> WezTerm. Activez dans **F9 → General → Kitty Graphics**. Les terminaux
> standard et les sessions SSH sans relais graphique afficheront un glyphe.

### Wavelog

L'intégration Wavelog supporte :

- envoi,
- téléchargement incrémental,
- worked/confirmed lookup.

Wavelog est configuré par logbook actif avec :

- URL,
- API key,
- station profile ID.

CQOps sauvegarde toujours les QSOs localement en premier. Un échec d'envoi Wavelog ne supprime pas les données locales.

### flrig

L'intégration flrig utilise XML-RPC sur HTTP.

Endpoint par défaut :

```text
localhost:12345
```

CQOps peut lire :

- frequency,
- mode,
- power.

En split, VFO A est mappé à Frequency et VFO B à Freq RX.

### Hamlib / rigctld

Le contrôle de rig Hamlib utilise le daemon TCP `rigctld`.

Selon le support radio/backend, CQOps peut interroger :

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps gère autant que possible l'absence de support des noms VFO.

### Hamlib Rotor / rotctld

> ⚠️ **Expérimental.** Le contrôle de rotor est expérimental. Vérifiez toujours
> les limites physiques de votre antenne avant d'opérer. Soyez prêt à arrêter
> le mouvement immédiatement avec **Alt+/** . À utiliser avec précaution — une
> configuration incorrecte peut endommager votre rotor ou votre antenne.

Le contrôle de rotor utilise Hamlib `rotctld`.

CQOps supporte :

- azimuth,
- elevation,
- stop commands.

| Raccourci | Action |
|---|---|
| Alt+, | Ajuster l'azimuth −5° |
| Alt+. | Ajuster l'azimuth +5° |
| Alt+; | Ajuster l'elevation +5° |
| Alt+' | Ajuster l'elevation −5° |
| Alt+\ | Pointer le rotor vers le path bearing calculé |
| Alt+/ | Arrêter le rotor |

### WSJT-X

L'intégration WSJT-X utilise les messages UDP de WSJT-X. CQOps analyse les messages ADIF et peut journaliser automatiquement les QSOs terminés.

L'étiquette rig prend la couleur d'accent pendant que WSJT-X transmet. Si l'opérateur rapporté par WSJT-X ne correspond pas à l'opérateur actif, CQOps affiche un avertissement.

### GPS

CQOps peut lire la position d'un récepteur GPS et l'utiliser comme grid
locator de la station — idéal pour les opérations portables, mobiles ou
en extérieur.

Deux backends sont pris en charge :

- **Serial** — se connecte directement à un récepteur GPS via un port
  série (USB-série, port COM intégré ou `/dev/ttyUSB0`).
- **GPSD** — se connecte à un serveur [gpsd](https://gpsd.io/) via TCP
  (par défaut `127.0.0.1:2947`). Utile lorsque le GPS est partagé avec
  d'autres applications ou accessible via le réseau.

L'indicateur GPS dans la barre d'état affiche :

| Couleur | Signification |
|--------|---------|
| Rouge `GPS` | Déconnecté / erreur |
| Jaune `GPS` | Connecté, pas encore de fix |
| Blanc `GPS` | Fix acquis, position verrouillée |

Lorsqu'un fix est acquis, le grid locator de la station est remplacé par
la position GPS et marqué `(GPS)` dans la ligne d'état :

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Activez **Grid from GPS** dans les paramètres Station & Logbook pour
utiliser le grid GPS pour la journalisation des QSO, les balises APRS,
la carte du tableau de bord et les calculs de distance.

**Précision du grid** — configurable dans le menu Intégrations (10, 8 ou
6 caractères). Par défaut 10 caractères (~25 m de précision).

### DX Cluster

L'intégration DX Cluster utilise telnet et nécessite internet.

Serveur par défaut :

```text
dxspots.com:7300
```

Les filtres incluent :

- band,
- continent,
- mode,
- age/time.

| Touche | Action |
|---|---|
| Enter | Remplir le QSO form, régler le rig et revenir à QSO |
| Space | Régler le rig et rester sur DX Cluster |
| Backspace | Effacer les filtres |

### PSK Reporter

PSK Reporter nécessite internet.

Il fournit :

- spots de propagation,
- filtres band/time/mode,
- ASCII world map sur **F5**.

### APRS

CQOps prend en charge trois types de service APRS — choisissez celui qui
correspond à votre configuration de station :

| Service | Connexion | Internet requis |
|---|---|---|
| **APRS-IS** | TCP vers un serveur APRS-IS | Oui |
| **KISS** | Port série vers un TNC KISS matériel | Non |
| **KISS Server** | TCP vers un serveur TNC KISS (ex. Dire Wolf) | Non (réseau local) |

Sélectionnez le type de service dans le menu Integrations :

```text
F9 → Integrations → APRS → Service (Espace pour changer)
```

Les trois services prennent en charge la réception de rapports de
position APRS des stations proches et leur affichage sur la carte
locale CQOps Live avec :

- symboles APRS standards,
- popups callsign,
- auto-fit view,
- range circle configurable.

Tous les services prennent également en charge le **beaconing périodique
de position**. CQOps transmet votre grid locator à l'intervalle configuré.
Lorsque le GPS est actif et **Grid from GPS** est activé, le beacon
utilise automatiquement la position GPS — idéal pour l'opération
portable et mobile.

#### APRS-IS

Se connecte au réseau mondial APRS-IS via internet. Nécessite :

- un indicatif radioamateur valide,
- un passcode APRS-IS (généré à partir de votre indicatif),
- une connexion internet.

Serveur par défaut :

```text
euro.aprs2.net:14580
```

APRS-IS est configuré globalement sous **F9 → Integrations → APRS**.
Indicatif, SSID, symbole, commentaire, intervalle de beacon et filtre
de portée par logbook sous **F9 → Logbooks → [logbook actif] → APRS**.

#### KISS (série)

> ⚠️ **Expérimental.** Le support KISS TNC est expérimental. Testez
> soigneusement avant d'en dépendre pour l'opération.

Se connecte directement à un TNC KISS matériel via un port série. Aucune
connexion internet n'est requise — les trames APRS sont envoyées et
reçues via votre radio.

Configurez le port série, le baud rate, les data bits, la parité, les
stop bits et DTR/RTS dans le menu Integrations :

```text
F9 → Integrations → APRS → Service: KISS
```

Lorsque KISS est sélectionné, les champs série (Port, Baud, Data bits,
Parity, Stop bits, DTR, RTS) deviennent visibles.

Le bouton **Test** ouvre le port série pour vérifier que le TNC est
accessible.

#### KISS Server (TCP)

> ⚠️ **Expérimental.** Le support KISS Server est expérimental. Testez
> soigneusement avant d'en dépendre pour l'opération.

Se connecte à un TNC KISS accessible via TCP — par exemple, une instance
[Dire Wolf](https://github.com/wb2osz/direwolf) exécutée sur la même
machine ou sur le réseau local. Aucune connexion internet n'est requise.

Entrez l'hôte et le port dans le menu Integrations :

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Par défaut : `127.0.0.1:8001`

#### Beaconing

Les beacons sont envoyés à l'intervalle configuré par logbook.
L'intervalle minimum est de 1 minute. Le beacon inclut :

- indicatif de station avec SSID,
- grid locator (basé sur le GPS lorsque disponible),
- symbole APRS,
- commentaire optionnel.

Lorsque le **GPS** est actif et **Grid from GPS** est activé dans les
paramètres Station, le beacon utilise automatiquement le grid locator
GPS — aucune mise à jour manuelle du grid n'est nécessaire en
déplacement.

Intervalle de beacon et autres paramètres par logbook :

```text
F9 → Logbooks → [logbook actif] → APRS
```

#### Réception

Les rapports de position APRS reçus sont mis en cache localement et
affichés sur la carte du dashboard CQOps Live. Les stations sont
montrées avec leurs symboles APRS et peuvent être cliquées pour plus
de détails. L'affichage s'auto-ajuste pour montrer toutes les stations
visibles dans la portée configurée.

La réception APRS est indépendante de la transmission de beacon — vous
pouvez recevoir sans envoyer de beacon, et vice-versa. Activez
simplement APRS dans le menu Integrations et définissez le type de
service.

### Solar Data

Solar data provient de hamqsl.com et inclut :

- SFI,
- sunspot number,
- A/K indices,
- band-by-band conditions.

Les mises à jour live nécessitent internet. Les données en cache restent disponibles offline après un fetch réussi.

---

## CQOps Live Dashboard

CQOps Live est un dashboard intégré dans le navigateur pour l'activité de station en temps réel.

Il est utile pour :

- affichages publics de field day,
- écrans de station de club,
- monitoring de concours,
- suivi de la station depuis une autre pièce,
- stands d'événement ou de salon.

### Activer le dashboard

1. Appuyez sur **F9**.
2. Ouvrez **Integrations**.
3. Allez à **HTTP Server**.
4. Activez **HTTP server**.
5. Définissez optionnellement address et port.
6. Appuyez sur **Ctrl+S** pour sauvegarder.
7. Ouvrez le dashboard dans un navigateur.

Paramètres par défaut :

| Paramètre | Défaut |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

Le serveur démarre immédiatement après sauvegarde.

### Modes d'affichage

CQOps Live a deux modes d'affichage.

#### Overview mode

Affiché lorsqu'aucun indicatif actif n'est en cours.

Il affiche :

- live Leaflet map,
- marqueurs QSO du jour,
- great-circle paths,
- table recent QSOs,
- informations station,
- statistiques,
- suivi du rythme 5 minutes, 15 minutes et 1 heure,
- top operators,
- QSOs les plus longs.

#### Active / Now Working mode

Affiché lorsqu'un indicatif est en cours de travail.

Il affiche :

- grand callsign,
- submode indicator,
- photo QRZ si disponible,
- badges band et mode,
- indicateurs DUPE / NEW CALL / NEW DXCC,
- distance et bearing,
- chemin cartographique pointillé mis en évidence du grid station au grid partenaire.

### Info box

L'info box au-dessus de la local map alterne toutes les 5 secondes entre modules :

- band conditions,
- solar activity,
- geomagnetic field,
- dernier spot DX Cluster,
- compteurs de rapports PSK Reporter par bande.

Band conditions est toujours rendu en pleine largeur.

### Weather row

La weather row affiche les conditions Open-Meteo actuelles pour le grid locator de la station :

- temperature,
- wind,
- humidity,
- icon.

Les données météo sont récupérées côté navigateur et se dégradent proprement hors-ligne.

### Local map

La local map à droite peut afficher :

- stations APRS,
- symboles APRS standards,
- range circle,
- popups callsign,
- day/night terminator overlay optionnel,
- RainViewer weather radar overlay optionnel.

### Mises à jour temps réel et performance

CQOps Live se met à jour via Server-Sent Events (SSE). Aucun rafraîchissement de page n'est nécessaire.

Le dashboard est conçu pour du matériel peu puissant :

- le navigateur rend la carte,
- le navigateur calcule les distances,
- le navigateur calcule les statistiques,
- CQOps pousse de légères mises à jour JSON,
- quand le HTTP server est désactivé, aucun port n'est ouvert et aucune goroutine du dashboard ne tourne.

### Personnalisation du dashboard

Dans le formulaire d'intégration HTTP Server, vous pouvez configurer :

| Champ | Description |
|---|---|
| Header 1 | Titre principal dans le page header et la hero area. Revient à « CQOps Live » par défaut. |
| Header 2 | Sous-titre sous le titre. Revient à « Fast, portable ham radio logger » par défaut. |
| Logo URL | URL d'image publiquement accessible affichée en haut à gauche. Revient au logo CQOps par défaut. |
| Event Start | Date au format `YYYY-MM-DD`. Filtre les statistiques et listes QSO à partir de cette date. |

---

## Configuration

Ouvrez la configuration avec **F9**.

### Fichiers de configuration

| Plateforme | Chemin config |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Les identifiants sensibles sont stockés séparément dans `secrets.enc` dans le même répertoire de configuration.

Les secrets sont chiffrés avec une clé liée à la machine. Lors du déplacement d'une configuration vers une autre machine, les identifiants doivent être saisis de nouveau.

### Menus de configuration

| Menu | Configure |
|---|---|
| Station | Callsign, grid, CQ/ITU zone, IARU region, references |
| Rig | Rig presets, model, antenna, power, backend, rotor, WSJT-X |
| Wavelog | URL, API key, station profile ID |
| QRZ | Username et password |
| DX Cluster | Host, port, login |
| Operators | Profils opérateur |
| Logbooks | Paramètres station, Wavelog, contest, operator et APRS par logbook |
| Integrations | Type de service APRS (APRS-IS, KISS, KISS Server), GPS, serveur HTTP, DXC, QRZ |
| Notifications | QSO saved alerts, Wavelog status, dupe beep, error sounds |
| General | Timezone, distance units, map, debug mode |

### Multi-logbook

Utilisez plusieurs logbooks pour la maison, le portable, les concours et le club.

**Ctrl+L** change le logbook actif.

Chaque logbook conserve ses propres :

- station details,
- Wavelog settings,
- contest settings,
- operator settings.

### Multi-operator

Les profils opérateur contiennent :

- indicatif opérateur,
- nom opérateur.

**Ctrl+O** change l'opérateur actif.

L'opérateur actif est enregistré dans le champ ADIF `OPERATOR` et suit les envois Wavelog.

### Multi-rig

Les presets de rig stockent :

- backend,
- model,
- antenna,
- power,
- rotor settings,
- WSJT-X settings.

**Ctrl+R** change le rig actif.

### Secrets chiffrés

Depuis v0.8.7, les identifiants sont stockés chiffrés.

| Élément | Valeur |
|---|---|
| Secrets file | `secrets.enc` |
| Location | Même répertoire que `config.yaml` |
| Unix permissions | `0600` lorsque supporté |
| Encryption | AES-256-GCM avec une clé liée à la machine |
| Protected data | QRZ password, DX Cluster login, Wavelog API keys |

Les plaintext secrets des anciennes configurations migrent au premier lancement.

Si `secrets.enc` est corrompu, CQOps démarre avec un avertissement et demande de ressaisir les identifiants.

---

## Raccourcis clavier

### Global

| Touche | Action |
|---|---|
| F1 | QSO form et Recent QSOs |
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
| Ctrl+L | Changer le logbook actif |
| Ctrl+R | Changer le rig actif |
| Ctrl+C | Changer le concours actif |
| Ctrl+O | Changer l'opérateur actif |
| Esc | Retour à l'écran précédent |

### QSO form

| Touche | Action |
|---|---|
| Tab | Champ suivant |
| Shift+Tab | Champ précédent |
| ↑ / ↓ | Se déplacer dans la colonne |
| Enter | Sauvegarder le QSO, avec confirmation de doublon si nécessaire |
| Ctrl+S | Sauvegarder le QSO depuis n'importe quel champ |
| Del | Effacer tous les champs du formulaire |
| Ins | Lookup : QRZ, Wavelog, DXCC et duplicate check |
| PgUp / PgDn | Changer band, mode ou submode |
| Ctrl+D | Ouvrir spot dialog |
| Ctrl+T | Basculer Keep Comment |
| Alt+, | Ajuster l'azimuth du rotor de −5° |
| Alt+. | Ajuster l'azimuth du rotor de +5° |
| Alt+; | Ajuster l'elevation du rotor de +5° |
| Alt+' | Ajuster l'elevation du rotor de −5° |
| Alt+\ | Pointer le rotor vers le bearing du grid local au grid partenaire |
| Alt+/ | Arrêter le rotor |
| Alt+0–9 | Rappeler un favorite |
| Alt+Shift+0–9 | Sauvegarder frequency, mode et band actuels comme favorite |

### Logbook Editor

| Touche | Action |
|---|---|
| ↑ / ↓ | Naviguer dans les lignes |
| PgUp / PgDn | Page précédente ou suivante |
| Home / End | Première ou dernière ligne |
| Enter / e | Modifier le QSO sélectionné |
| Delete | Supprimer le QSO sélectionné |
| p | Purge de tous les QSOs |
| Ctrl+C | Changer le filtre concours |
| Ctrl+E | Export ADIF |
| Ctrl+I / Tab | Import ADIF |
| w | Envoyer les QSOs non envoyés vers Wavelog |
| Ctrl+W | Télécharger les contacts depuis Wavelog |
| Esc / F6 | Fermer l'éditeur et revenir au QSO form |

### DX Cluster

| Touche | Action |
|---|---|
| ↑ / ↓ | Naviguer dans les spots |
| Enter | Remplir le QSO form, régler le rig et revenir à QSO |
| Space | Régler le rig sur le spot sélectionné et rester sur DX Cluster |
| Home | Avancer le filtre band |
| End | Reculer le filtre band |
| `\` | Changer le filtre continent |
| Ins | Avancer le filtre mode |
| Del | Reculer le filtre mode |
| PgUp | Avancer le filtre time |
| PgDn | Reculer le filtre time |
| Backspace | Effacer tous les filtres |
| Esc / F4 | Revenir au QSO form |

### Partner view

| Touche | Action |
|---|---|
| F2 | Parcourir Partner view → Photo → Back |
| Esc / F1 | Revenir au QSO form |

---

## Dépannage

### CQOps ne démarre pas

Vérifiez :

- la taille du terminal est au moins 80×24,
- les utilisateurs Windows utilisent Windows Terminal,
- le démarrage réseau ne bloque pas en essayant :

  ```bash
  cqops --offline
  ```

Vérifiez les logs :

| Plateforme | Chemin des logs |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### Le rig ne se connecte pas

Pour flrig :

- vérifiez que flrig tourne,
- vérifiez le port dans le preset de rig actif,
- le port par défaut est `12345`.

Pour Hamlib :

- vérifiez que `rigctld` tourne,
- vérifiez host et port,
- vérifiez que votre radio/backend supporte les données demandées.

Les status labels aident au diagnostic :

| Couleur | Signification |
|---|---|
| Blanc/défaut | Connecté |
| Jaune | Désactivé ou connexion en cours |
| Rouge | Échec |

Les reconnect toasts peuvent être supprimés. CQOps peut réessayer silencieusement.

### WSJT-X ne journalise pas automatiquement

Vérifiez :

- WSJT-X **Settings → Reporting → UDP Server**,
- host et port UDP correspondent au preset de rig actif dans CQOps,
- WSJT-X 2.6 ou plus récent est utilisé,
- le status label WSJT est actif,
- le logbook actif est correct,
- l'opérateur actif est correct.

### L'envoi Wavelog échoue

Vérifiez :

- Wavelog URL,
- API key,
- station profile ID,
- status label **WL**.

Les erreurs d'envoi sont affichées comme toasts. Les QSOs restent sauvegardés localement même si l'envoi échoue. Les échecs individuels de QSO ne bloquent pas le reste du lot.

### Problèmes de config file

Config file :

| Plateforme | Chemin |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets file :

```text
secrets.enc
```

Le secrets file est stocké dans le même répertoire que `config.yaml`.

Si la configuration est corrompue, déplacez-la ou supprimez-la puis redémarrez CQOps. Le setup wizard créera une configuration fraîche.

Le champ `last_fetched_id` n'apparaît qu'après un téléchargement Wavelog réussi.

### Problèmes de performance

Essayez :

- désactiver map rendering dans General settings,
- désactiver le Solar panel si inutile,
- éviter les écrans lourds en réseau comme DX Cluster et PSK Reporter en offline,
- utiliser `cqops --offline` lorsque le réseau est peu fiable.

---

## Signaler des bugs

Avant de signaler un bug :

1. Activez **Debug mode** dans **F9 → General → Debug**, ou définissez :

   ```yaml
   debug: true
   ```

   dans `config.yaml`.

2. Reproduisez le problème.
3. Joignez le log pertinent.

Signalez les problèmes sur GitHub :

<https://github.com/szporwolik/cqops/issues>

Incluez :

- version de CQOps depuis `cqops --version`,
- système d'exploitation,
- terminal emulator,
- étapes pour reproduire,
- debug log pertinent.
