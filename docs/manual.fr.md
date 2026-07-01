---
title: Manuel utilisateur CQOps
description: Guide complet pour installer, configurer et utiliser CQOps — un logger radioamateur rapide, orienté terminal
---

> **Note :** Cette traduction a été générée avec un modèle LLM. Les corrections sont les bienvenues — veuillez les proposer sous forme de Pull Request vers la branche `dev`.

# Manuel utilisateur CQOps

## Table des matières

1. [À propos de CQOps](#à-propos-de-cqops)
2. [Téléchargement et Installation](#téléchargement-et-installation)
3. [Premier lancement — assistant de configuration](#premier-lancement--assistant-de-configuration)
4. [Démarrage rapide : enregistrer votre premier QSO](#démarrage-rapide--enregistrer-votre-premier-qso)
5. [Aperçu de l'écran principal](#aperçu-de-lécran-principal)
6. [Flux de travail courants](#flux-de-travail-courants)
7. [Fonctionnalités principales](#fonctionnalités-principales)
8. [Intégrations](#intégrations)
9. [Référence de configuration](#référence-de-configuration)
10. [Raccourcis Clavier](#raccourcis-clavier)
11. [Dépannage](#dépannage)

---

## À propos de CQOps

CQOps est un logger radioamateur rapide, orienté terminal, conçu pour les opérateurs qui ont besoin de vitesse, de fiabilité et d'une faible charge système — au shack, sur un sommet, lors d'un field day ou dans une station de club partagée.

**Priorité au hors-ligne.** Le logging local des QSO ne nécessite pas d'accès internet. Les données de référence, les données solaires et les préfixes DXCC mis en cache restent disponibles après un premier téléchargement. Les intégrations réseau telles que Wavelog, QRZ.com, DX Cluster et PSK Reporter nécessitent une connexion et sont ignorées en mode `--offline`.

**Conçu pour le terrain.** CQOps est prêt pour le QRP, adapté au SOTA/POTA et fonctionne confortablement sur des machines de classe Raspberry Pi, de vieux portables et des systèmes sans environnement de bureau.

**Prêt pour les stations de club.** CQOps prend en charge plusieurs logbooks, profils d'opérateur et préréglages de rig. Le logbook actif, l'opérateur actif ou le rig actif se changent d'une seule touche.

**Portable par conception.** CQOps est un binaire unique écrit en Go. Il n'a aucune dépendance CGO et ne nécessite aucun service système.

**Multi-plateforme.** Windows, Linux et macOS sont pris en charge sur amd64 et arm64.

### À qui s'adresse CQOps

- Opérateurs portables ayant besoin d'un logging rapide au clavier sur du matériel basse consommation.
- Activateurs SOTA et POTA qui loguent hors-ligne et téléversent plus tard.
- Stations de club avec plusieurs opérateurs partageant la même station.
- Équipes de field day utilisant des machines partagées ou du matériel de classe Raspberry Pi.
- Opérateurs préférant un flux de travail en terminal à une interface graphique.

CQOps n'est pas destiné à remplacer les loggers de bureau complets ou les plateformes de logbook web. Il se concentre sur le logging rapide en terminal, l'utilisation sur le terrain, l'utilisation hors-ligne et les flux de travail en station partagée.

---

## Téléchargement et installation

> [Parcourir toutes les versions →](https://github.com/szporwolik/cqops/releases)

### Windows

| Paquet | Lien | Notes |
|---------|------|-------|
| **Installateur** | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recommandé pour la plupart des utilisateurs. Ajoute CQOps au Menu Démarrer et au PATH. |
| ZIP Portable | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extraire et exécuter sans installation. |

### Linux — Debian / Ubuntu

| Architecture | Lien | Pour |
|-------------|------|---------|
| **amd64** | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La plupart des PC Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Systèmes ARM 64 bits |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS 32 bits |

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Tarball Portable

| Architecture | Lien | Pour |
|-------------|------|---------|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La plupart des PC Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Systèmes ARM 64 bits |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS 32 bits |

### macOS

| Architecture | Lien | Pour |
|-------------|------|---------|
| **Apple Silicon** | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Macs M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Macs Intel |

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Depuis les Sources

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build        # Construction seule ; le binaire est dans build/
make install      # Construction et installation système
```

La compilation depuis les sources nécessite Go 1.26 ou plus récent.

### Prérequis

- Taille du terminal : minimum 80×24 caractères.
- Taille recommandée : 80×43 ou plus.
- Un émulateur de terminal moderne est recommandé. Sous Windows, utilisez Windows Terminal plutôt que la console héritée.

### Options de ligne de commande

```bash
cqops              # Lancer l'interface TUI
cqops --offline    # Lancer sans activité réseau
cqops --version    # Afficher la version et quitter
cqops --help       # Afficher l'aide
```

---

## Premier lancement — assistant de configuration

Au premier lancement, CQOps ouvre un assistant de configuration pour les paramètres essentiels de la station. Les intégrations réseau peuvent être ignorées ; le logging local fonctionne sans elles.

1. **Station & Logbook** — Configurer le logbook initial, l'indicatif de station, l'opérateur et le grid locator. Champs optionnels : références SOTA/POTA/WWFF, région IARU, zone CQ/ITU, DXCC et SIG/SIG Info. La configuration Wavelog est également disponible ici : URL, clé API, ID de profil de station, Update et Test.

2. **Rig** — Configurer un préréglage de rig : nom, modèle, antenne, puissance et backend radio. Backends pris en charge : None, flrig et Hamlib rigctld. Paramètres optionnels : contrôle de rotor Hamlib et intégration UDP WSJT-X.

3. **Intégrations** — Configurer la recherche callbook QRZ.com : option d'activation, nom d'utilisateur, mot de passe masqué et Test.

4. **Général** — Sélectionner le fuseau horaire IANA. CQOps détecte le fuseau horaire système par défaut et fournit également une liste déroulante.

5. **Résumé** — Vérifier la configuration. Appuyez sur **Ctrl+S** pour enregistrer et démarrer CQOps.

**Navigation de l'assistant :** **Ctrl+S** avance après validation. **Esc** recule. **F10** quitte. La barre d'espace bascule les cases à cocher. Tab et Shift+Tab se déplacent entre les champs.

Tous les paramètres de l'assistant peuvent être modifiés ultérieurement depuis le menu de configuration avec **F9**.

---

## Démarrage rapide : enregistrer votre premier QSO

1. **Installer et lancer CQOps.** Téléchargez le paquet pour votre plateforme, lancez `cqops` et terminez l'assistant de configuration avec au moins votre indicatif et grid locator.

2. **Utiliser le QSO Form.** Le QSO Form s'ouvre sur **F1**. Entrez un indicatif ; CQOps le met automatiquement en majuscules. Si le rig actif est connecté via flrig ou Hamlib, la fréquence, la bande, le mode et le submode sont remplis automatiquement. La date et l'heure sont définies sur l'UTC actuel.

3. **Naviguer entre les champs.** Utilisez **Tab**, **Shift+Tab** et **↑/↓**.

4. **Enregistrer le QSO.** Appuyez sur **Enter** ou **Ctrl+S**. Si un avertissement **DUPE!** apparaît, appuyez à nouveau sur **Enter** pour enregistrer quand même, ou **Esc** pour annuler.

Le nouveau QSO apparaît immédiatement dans le tableau Recent QSOs sous le formulaire.

---

## Aperçu de l'écran principal

```text
┌─ Status Bar ──────────────────────────────────────────────────────────────────┐
│  CQOps v0.8.8  Log Portable  Rig FTDx10  Call SP9MOA/P                          │
│  Net WSJT Hamlib DXC WL                                            23:00L 2100Z │
├─ Tab Bar ────────────────────────────────────────────────────────────────┤
│ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮         │
│ │F1 QSO│ │F2 QRZ│ │F4 DXC│ │F5 HRD│ │F6 REF│ │F7 BPL│ │F8 LOG│ │F9 CFG│         │
├─ Main Content Area ──────────────────────────────────────────────────────┤
│                                                                                  │
│  QSO Form, tableau, carte, éditeur ou contenu de l'écran actif                   │
│                                                                                  │
├─ Help Bar ───────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Status Bar

Status Bar affiche la version de CQOps, le logbook actif, le rig actif, l'indicatif de station et l'opérateur actif. À droite figurent les étiquettes de statut d'intégration et l'heure locale (`L`) et UTC (`Z`).

**Couleurs des étiquettes :**

| Couleur | Signification |
|-------|---------|
| Blanc/défaut | Connecté ou actif |
| Jaune | Désactivé, connexion en cours ou hors-ligne attendu |
| Rouge | Erreur ou déconnecté |
| Accent + gras | WSJT-X est en transmission |

Étiquettes possibles : **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC** et **WL**.

### Tab Bar

| Touche | Onglet | Écran |
|-----|-----|--------|
| F1 | QSO | QSO Form et tableau Recent QSOs |
| F2 | QRZ | Partner view : données callbook, carte, stats, photo |
| F4 | DXC | Spots DX Cluster et filtres |
| F5 | HRD | Spots PSK Reporter et carte de propagation |
| F6 | REF | Recherche de références SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Navigateur de plans de bande |
| F8 | LOG | Logbook Editor, ADIF, synchronisation Wavelog |
| F9 | CFG | Menus de configuration |

### Help Bar

La ligne inférieure affiche les raccourcis les plus pertinents pour l'écran actif. Appuyez sur **?** pour la superposition d'aide complète.

---

## Flux de travail courants

### Opération portable / SOTA / POTA

1. **Avant de quitter la maison**, lancez CQOps une fois avec un accès internet pour remplir les caches : données solaires, données REF et préfixes DXCC.
2. **Vérifiez le cache** avant de passer hors-ligne. Le panneau Solar doit afficher des données et la recherche REF sur **F6** doit retourner des résultats.
3. **Sur le terrain**, lancez CQOps avec `cqops --offline`. L'activité réseau est ignorée, évitant les délais des services inaccessibles.
4. **Journalisez normalement.** Le logging local fonctionne sans internet.
5. **Téléversez plus tard.** De retour en ligne, ouvrez le Logbook Editor avec **F8** et appuyez sur **w** pour téléverser les QSO non envoyés vers Wavelog.

### Station de club partagée et hot-seat

1. **Ajoutez des profils d'opérateur :** ouvrez **F9 → Operators**, puis appuyez sur **Ins** pour chaque opérateur. Entrez son indicatif et son nom.
2. **Changez d'opérateur actif :** sur le QSO Form, appuyez sur **Ctrl+O**. L'opérateur actif est affiché dans la Status Bar et enregistré dans le champ `OPERATOR` des QSO sauvegardés.
3. **Utilisez le logging hot-seat :** l'opérateur A logue un QSO, l'opérateur B appuie sur **Ctrl+O**, puis logue sous son propre profil.
4. **Utilisez Retain si nécessaire :** activez **Retain** si plusieurs opérateurs doivent journaliser le même contact sans retaper le formulaire complet.

Avant d'enregistrer sur une station partagée, vérifiez le logbook actif et l'opérateur actif dans la Status Bar.

### Logbook privé + club

Beaucoup d'opérateurs maintiennent un logbook personnel et un ou plusieurs logbooks de club.

1. **Créez des logbooks :** ouvrez **F9 → Logbooks**, puis appuyez sur **Ins** pour chaque logbook.
2. **Changez de logbook actif :** appuyez sur **Ctrl+L** sur le QSO Form. Status Bar affiche le logbook actif.
3. **Gardez les données de station séparées :** chaque logbook peut avoir son propre indicatif, paramètres Wavelog, paramètres de concours et opérateurs.
4. **Double logging rapide :** activez **Retain**, enregistrez le QSO dans un logbook, appuyez sur **Ctrl+L**, puis enregistrez-le dans l'autre logbook si approprié.

### Plusieurs rigs

1. **Créez des préréglages de rig :** ouvrez **F9 → Rigs**, puis appuyez sur **Ins** pour chaque rig.
2. **Définissez le backend :** utilisez flrig ou Hamlib pour les rigs contrôlés par CAT. Utilisez None pour les rigs accordés manuellement.
3. **Changez de rig actif :** appuyez sur **Ctrl+R** sur le QSO Form.
4. **Opérez des stations mixtes :** par exemple, un rig HF contrôlé par CAT et un rig VHF/UHF manuel dans la même session.
5. **Configurez WSJT-X par rig :** chaque préréglage de rig peut avoir ses propres paramètres UDP WSJT-X.

Lorsque le rig actif a le contrôle CAT, CQOps peut remplir automatiquement la fréquence, la bande, le mode et le submode. Pour les rigs manuels, saisissez-les vous-même.

### FT8 / logging automatique WSJT-X

Lorsque WSJT-X est connecté via UDP, CQOps peut journaliser automatiquement les QSO numériques à partir des messages ADIF de WSJT-X.

- Les QSO auto-journalisés sont enregistrés dans le logbook actif.
- Les QSO auto-journalisés en double sont ignorés.
- Les QSO auto-journalisés héritent de l'ID de concours actif.
- Les QSO apparaissent immédiatement dans Recent QSOs.
- Si Wavelog est configuré et accessible, les QSO auto-journalisés peuvent être téléversés automatiquement.
- Si l'opérateur WSJT-X ne correspond pas à l'opérateur actif, CQOps affiche un avertissement.

Vérifiez le logbook actif, l'opérateur actif et le concours actif avant les longues sessions numériques.

### Synchronisation Wavelog

La synchronisation Wavelog est optionnelle. CQOps enregistre toujours les QSO localement en premier.

**Upload :** appuyez sur **w** dans le Logbook Editor (**F8**). CQOps téléverse les QSO non envoyés par lots de 50 et suit le statut par QSO : non envoyé, envoyé ou erreur.

**Téléchargement :** appuyez sur **Ctrl+W** dans le Logbook Editor. Les téléchargements sont incrémentiels. CQOps récupère les QSO plus récents que le `last_fetched_id` enregistré pour le logbook actif. Les doublons sont ignorés.

Si un téléversement Wavelog échoue, le QSO reste dans le logbook local et peut être réessayé plus tard. Vider un logbook réinitialise l'ID de récupération à `0`, permettant un re-téléchargement complet.

---

## Fonctionnalités principales

### Logging QSO

Le QSO Form (**F1**) est l'écran principal de logging. Il utilise une disposition à trois colonnes et peut remplir automatiquement les champs depuis le contrôle de rig, QRZ.com, la recherche Wavelog, les données DXCC/préfixes et les bases de données REF.

**Champs du formulaire :**

| Colonne Gauche | Colonne Centrale | Colonne Droite |
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

⚠️ Les champs d'exchange n'apparaissent que lorsqu'un concours est actif. Les champs marqués **(▼)** se parcourent avec **PgUp/PgDn**.

La ligne inférieure contient :

- **Comment**
- **Keep** — préserve le champ Comment entre les QSO ; basculer avec **Ctrl+T**
- **Retain** — préserve tout le formulaire après l'enregistrement

La ligne de route/azimut affiche la distance et l'azimut lorsque les deux locators de grille sont connus. Elle peut également afficher des badges comme **DUPE!**, **New Call!** et **New DXCC!**.

### Sources d'autoremplissage

| Source | Champs |
|--------|--------|
| flrig / Hamlib | Frequency, Freq RX si split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| Base REF | Références SOTA, POTA, WWFF, IOTA |
| Recherche Wavelog | Statut worked/confirmed si configuré |

### Logging de concours

Les concours ajoutent des champs d'exchange et la gestion des numéros de série au QSO Form.

Créez ou configurez un concours dans le Logbook Editor (**F8**) avec **Ins**. Définissez le nom du concours, la date, l'ID de concours ADIF et les modèles d'exchange.

Marqueurs de modèle supportés :

| Marqueur | Remplacé Par |
|--------|---------------|
| `@rst` | RST envoyé ou reçu |
| `@serial` | Numéro de série auto-incrémenté |
| `@call` | Votre indicatif |
| `@grid` | Votre grid locator |
| `@name` | Nom de l'opérateur depuis le profil d'opérateur |

Appuyez sur **Ctrl+C** pour parcourir le concours actif. Lorsqu'un concours est actif :

- le QSO Form affiche les champs d'exchange,
- les numéros de série s'incrémentent automatiquement,
- Recent QSOs peut filtrer sur les QSO de concours,
- l'export ADIF préserve `CONTEST_ID`.

### Logbook Editor

Le Logbook Editor (**F8**) sert à la gestion des QSO, l'import/export ADIF, la synchronisation Wavelog et les opérations liées aux concours.

**Édition en ligne :** sélectionnez une ligne avec **↑/↓**, appuyez sur **Enter** ou **e**, modifiez le QSO, puis enregistrez avec **Ctrl+S**. Les modifications sont reflétées immédiatement dans Recent QSOs.

### Import et Export ADIF

CQOps prend en charge l'import et l'export ADIF 3.1.7.

- **Ctrl+I** importe un fichier ADIF, valide les enregistrements, ignore les doublons et affiche un résumé.
- **Ctrl+E** exporte les QSOs. L'export peut inclure tous les QSOs ou les QSOs filtrés par concours.
- Les QSOs importés sont marqués pour le téléversement Wavelog si la synchronisation Wavelog est configurée.

### Favoris

Les favoris stockent des préréglages de fréquence, mode et bande dans 10 emplacements.

| Raccourci | Action |
|----------|--------|
| Alt+0–9 | Rappeler l'emplacement favori |
| Alt+Shift+0–9 | Enregistrer fréquence/mode/bande actuels dans l'emplacement |

Les favoris sont stockés dans la configuration et partagés entre les logbooks.

Exemple : pour une configuration d'appel SOTA FM polonaise, entrez `145.55`, mode `FM`, bande `2m`, puis appuyez sur **Alt+Shift+1**. Plus tard, appuyez sur **Alt+1** pour le rappeler.

### Recherche REF

L'écran REF (**F6**) recherche les références SOTA, POTA, WWFF et IOTA. Recherchez par préfixe, nom ou désignateur de référence. Les références sélectionnées peuvent remplir le QSO Form.

### Band Plan Browser

Le Band Plan Browser (**F7**) donne un accès rapide aux bandes amateur, gammes VHF/UHF, CB, PMR446 et préréglages de diffusion. Une fréquence sélectionnée peut être utilisée pour accorder le rig actif. Les données de plans de bande peuvent également être exportées en Markdown.

---

## Intégrations

### QRZ.com

La recherche QRZ.com nécessite un accès internet et un abonnement QRZ XML.

Appuyez sur **Ins** dans le QSO Form pour remplir les champs callbook : nom, QTH, grid, pays, zones CQ/ITU, DXCC et continent. La Partner view (**F2**) peut afficher la photo de l'opérateur si disponible.

### Wavelog

L'intégration Wavelog nécessite un accès internet. Elle prend en charge le téléversement, le téléchargement incrémentiel et la recherche worked/confirmed.

Wavelog est configuré par logbook actif avec l'URL, la clé API et l'ID de profil de station. CQOps enregistre toujours les QSOs localement en premier ; l'échec du téléversement Wavelog n'entraîne pas de perte de données.

Voir [Synchronisation Wavelog](#synchronisation-wavelog).

### flrig

L'intégration flrig utilise XML-RPC sur HTTP. Le point de terminaison par défaut est `localhost:12345`.

CQOps peut lire la fréquence, le mode et la puissance depuis flrig. L'opération split est mappée comme VFO A → Frequency et VFO B → Freq RX.

### Hamlib / rigctld

Le contrôle de rig Hamlib utilise le démon TCP `rigctld`. CQOps peut interroger la fréquence, le mode, le VFO, le split et la puissance selon le support du rig.

Certains rigs ou backends Hamlib ne prennent pas en charge toutes les requêtes. CQOps gère l'absence de support de nom VFO de manière sécurisée lorsque c'est possible.

### Hamlib Rotor / rotctld

Le contrôle de rotor utilise Hamlib `rotctld`. CQOps prend en charge les commandes d'azimut, d'élévation et d'arrêt.

Raccourcis utiles :

| Raccourci | Action |
|----------|--------|
| Ctrl+←/→ | Ajuster l'azimut de 5° |
| Ctrl+↑/↓ | Ajuster l'élévation de 5° |
| Ctrl+A | Pointer le rotor vers l'azimut de route calculé |
| Ctrl+F1 | Arrêter le rotor |

### WSJT-X

L'intégration WSJT-X utilise les messages UDP de WSJT-X. CQOps analyse les messages ADIF et peut journaliser automatiquement les QSOs terminés.

L'étiquette du rig devient de couleur accent lorsque WSJT-X transmet. Si l'opérateur rapporté par WSJT-X ne correspond pas à l'opérateur actif, CQOps affiche un avertissement.

Voir [FT8 / logging automatique WSJT-X](#ft8--logging-automatique-wsjt-x).

### DX Cluster

L'intégration DX Cluster utilise une connexion telnet et nécessite un accès internet. Le serveur par défaut est `dxspots.com:7300`.

Les filtres incluent la bande, le continent, le mode et l'âge/temps. Appuyez sur **Enter** sur un spot pour remplir le QSO Form, accorder le rig actif et revenir à l'écran QSO. Appuyez sur **Space** pour accorder sans remplir le formulaire. Appuyez sur **Backspace** pour effacer les filtres.

### PSK Reporter

L'intégration PSK Reporter nécessite un accès internet. Elle fournit des spots de propagation, des filtres bande/temps/mode et une carte du monde ASCII sur **F5**.

### APRS

L'intégration APRS utilise une connexion TCP vers un serveur APRS-IS et nécessite un accès internet. Le serveur par défaut est `euro.aprs2.net:14580`.

CQOps reçoit les rapports de position des stations à proximité et les affiche sur la carte locale du tableau de bord CQOps Live avec des symboles standard, des popups d'indicatif et une vue à ajustement automatique. Un cercle de portée configurable montre la zone de couverture de la balise. Une balise périodique avec l'indicatif de la station, le SSID, le locator de grille et un commentaire optionnel peut être envoyée.

APRS est configuré par logbook dans les paramètres de station (**F9 → Logbooks → [logbook actif] → APRS**).

### Données solaires

Les données solaires incluent le SFI, le nombre de taches solaires, les indices A/K et les conditions bande par bande depuis hamqsl.com. Les mises à jour en direct nécessitent un accès internet. Les données mises en cache restent disponibles hors-ligne après une récupération réussie.

### CQOps Live — Tableau de bord navigateur

CQOps Live est un tableau de bord web intégré qui affiche l'activité de votre station en temps réel dans n'importe quel navigateur — parfait pour les présentations Field Day, les écrans de radio-club, la surveillance de concours ou pour garder un œil sur la station depuis une autre pièce.

**Activation**

1. Appuyez sur **F9** pour ouvrir le menu principal, puis sélectionnez **Integrations**.
2. Faites défiler jusqu'à la section **HTTP Server** et cochez **Enable HTTP server**.
3. Configurez éventuellement l'adresse (par défaut `0.0.0.0`) et le port (par défaut `8073`).
4. Appuyez sur **Ctrl+S** pour enregistrer. Le serveur démarre immédiatement.
5. Ouvrez `http://localhost:8073` (ou l'adresse configurée) dans n'importe quel navigateur.

**Ce que montre le tableau de bord**

Le tableau de bord a deux modes qui changent automatiquement :

- **Mode aperçu** (aucun indicatif actif) : une carte Leaflet en direct avec les marqueurs QSO du jour et les tracés orthodromiques, un tableau des QSO récents, les infos de station, les statistiques, les meilleurs opérateurs et les QSO les plus longs.
- **Mode Actif / Now Working** (indicatif en cours) : un affichage proéminent de l'indicatif, photo QRZ (si disponible), badges bande/mode, indicateurs DUPE/NEW CALL/NEW DXCC, distance et azimut, et une ligne pointillée mise en évidence sur la carte de votre station à l'emplacement du correspondant.

Tous les panneaux sont mis à jour en temps réel via Server-Sent Events (SSE) — aucun rechargement de page nécessaire.

**Personnalisation**

Dans le formulaire d'intégration du serveur HTTP, vous pouvez configurer :

| Champ | Description |
|-------|-------------|
| Header 1 | Titre principal affiché dans l'en-tête et la zone hero. Valeur par défaut : « CQOps Live ». |
| Header 2 | Sous-titre sous le titre. Valeur par défaut : « Fast, portable ham radio logger ». |
| Logo URL | URL d'image accessible publiquement affichée en haut à gauche. Valeur par défaut : logo CQOps. |
| Event Start | Date au format `YYYY-MM-DD`. Lorsqu'elle est définie, les statistiques et les listes QSO sont filtrées à partir de cette date — utile pour les événements de plusieurs jours. |

**Performance**

Le tableau de bord est conçu pour du matériel à faible consommation. Le navigateur gère tout le rendu cartographique, les calculs de distance et les statistiques. L'application terminal CQOps envoie uniquement des mises à jour JSON légères via SSE. Lorsque le serveur HTTP est désactivé, il n'y a aucune surcharge.

**Cas d'usage typiques**

- **Field Day / présentation publique** : connectez un grand écran ou projecteur pour afficher la carte en direct et les QSO récents.
- **Écran d'information du radio-club** : moniteur dédié montrant l'activité aux visiteurs.
- **Surveillance à distance** : ouvrez le tableau de bord sur une tablette ou un téléphone pour suivre l'activité depuis une autre pièce.
- **Stand salon / événement** : configurez Header 1/2 et le logo du club pour une présentation professionnelle.

---

## Référence de configuration

La configuration de CQOps est stockée dans :

| Plateforme | Chemin de configuration |
|----------|-------------|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Les informations d'identification sensibles sont stockées séparément dans `secrets.enc` dans le même répertoire de configuration. Les identifiants sont chiffrés avec une clé liée à la machine, donc les informations d'identification doivent être ressaisies lors du déplacement d'une configuration vers une autre machine.

Ouvrez la configuration avec **F9**.

| Menu | Configure |
|------|------------|
| Station | Indicatif, grid, zone CQ/ITU, région IARU, références |
| Rig | Préréglages de rig, modèle, antenne, puissance, backend, rotor, WSJT-X |
| Wavelog | URL, clé API, ID de profil de station |
| QRZ | Nom d'utilisateur et mot de passe |
| DX Cluster | Hôte, port, login |
| Operators | Profils d'opérateur : indicatif et nom |
| Logbooks | Paramètres de station, Wavelog, concours et opérateur par logbook |
| Notifications | Comportement des toasts et notifications |
| General | Fuseau horaire, unités de distance, carte, mode debug |

### Multi-Logbook

Utilisez plusieurs logbooks pour les opérations à domicile, portables, concours et club. Appuyez sur **Ctrl+L** pour parcourir le logbook actif. Chaque logbook conserve ses propres détails de station, paramètres Wavelog, paramètres de concours et opérateurs.

### Multi-Opérateur

Les profils d'opérateur contiennent l'indicatif et le nom de l'opérateur. Appuyez sur **Ctrl+O** pour parcourir l'opérateur actif. L'opérateur actif est enregistré dans le champ ADIF `OPERATOR` et utilisé lors des téléversements Wavelog.

### Multi-Rig

Les préréglages de rig stockent le backend, le modèle, l'antenne, la puissance, le rotor et les paramètres WSJT-X. Appuyez sur **Ctrl+R** pour parcourir le rig actif.

### Identifiants chiffrés

Depuis la v0.8.7, les informations d'identification sont stockées chiffrées.

- **Fichier d’identifiants :** `secrets.enc`
- **Emplacement :** même répertoire que `config.yaml`
- **Permissions Unix :** `0600` lorsque supporté
- **Chiffrement :** AES-256-GCM avec clé liée à la machine
- **Données protégées :** mot de passe QRZ, login DX Cluster, clés API Wavelog
- **Migration :** les identifiants en texte clair des anciennes configurations migrent au premier lancement
- **Récupération :** si `secrets.enc` est corrompu, CQOps démarre avec un avertissement et demande de ressaisir les informations d'identification

---

## Raccourcis clavier

### Global

| Touche | Action |
|-----|--------|
| F1 | QSO Form et Recent QSOs |
| F2 | Partner view |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | Recherche REF |
| F7 | Navigateur de plans de bande |
| F8 | Logbook Editor |
| F9 | Configuration / menu principal |
| F10 | Quitter |
| Ctrl+F9 | Visualiseur de logs |
| ? | Superposition d'aide |
| Ctrl+L | Parcourir le logbook actif |
| Ctrl+R | Parcourir le rig actif |
| Ctrl+C | Parcourir le concours actif |
| Ctrl+O | Parcourir l'opérateur actif |
| Esc | Revenir à l'écran précédent |

### QSO Form — F1

| Touche | Action |
|-----|--------|
| Tab | Champ suivant |
| Shift+Tab | Champ précédent |
| ↑ / ↓ | Se déplacer dans la colonne |
| Enter | Enregistrer le QSO, avec confirmation de doublon si nécessaire |
| Ctrl+S | Enregistrer le QSO depuis n'importe quel champ |
| Del | Effacer tous les champs du formulaire |
| Ins | Recherche : QRZ, Wavelog, DXCC et vérification de doublon |
| PgUp / PgDn | Parcourir bande, mode ou submode |
| Ctrl+D | Ouvrir la boîte de dialogue spot |
| Ctrl+T | Basculer Keep Comment |
| Ctrl+←/→ | Ajuster l'azimut du rotor de 5° |
| Ctrl+↑/↓ | Ajuster l'élévation du rotor de 5° |
| Ctrl+A | Pointer le rotor vers l'azimut du grid propre au grid partenaire |
| Ctrl+F1 | Arrêter le rotor |
| Alt+0–9 | Rappeler l'emplacement favori |
| Alt+Shift+0–9 | Enregistrer fréquence/mode/bande actuels dans l'emplacement favori |

### Logbook Editor — F8

| Touche | Action |
|-----|--------|
| ↑ / ↓ | Naviguer dans les lignes |
| PgUp / PgDn | Page précédente ou suivante |
| Home / End | Première ou dernière ligne |
| Enter / e | Modifier le QSO sélectionné |
| Delete | Supprimer le QSO sélectionné |
| p | Vider tous les QSOs |
| Ctrl+C | Basculer le filtre de concours |
| Ctrl+E | Exporter ADIF |
| Ctrl+I / Tab | Importer ADIF |
| w | Uploader les QSOs non envoyés vers Wavelog |
| Ctrl+W | Télécharger les contacts depuis Wavelog |
| Esc / F6 | Fermer l'éditeur, revenir à QSO |

### DX Cluster — F4

| Touche | Action |
|-----|--------|
| ↑ / ↓ | Naviguer dans les spots |
| Enter | Remplir le formulaire + accorder le rig + aller à QSO |
| Space | Accorder le rig sur le spot (rester sur DXC) |
| Home | Filtre de bande avant |
| End | Filtre de bande arrière |
| \\ | Filtre de continent |
| Ins | Filtre de mode avant |
| Del | Filtre de mode arrière |
| PgUp | Filtre de temps avant |
| PgDn | Filtre de temps arrière |
| Backspace | Effacer tous les filtres |
| Esc / F4 | Revenir au QSO Form |

### Partner View — F2

| Touche | Action |
|-----|--------|
| F2 | Cycle : Partner view → Photo → Retour |
| Esc / F1 | Revenir au QSO Form |

---

## Dépannage

### L'application ne démarre pas

- Le terminal doit avoir au moins 80×24 caractères.
- Sous Windows, utilisez Windows Terminal, pas la console héritée `cmd.exe`.
- Essayez `cqops --offline` pour exclure les problèmes réseau.
- Vérifiez les logs : `~/.local/share/cqops/logs/` (Linux), `~/Library/Application Support/cqops/logs/` (macOS) ou `%APPDATA%\cqops\logs\` (Windows).

### Le rig ne se connecte pas

- **flrig :** vérifiez que flrig est en cours d'exécution et que le port correspond (défaut `12345`).
- **Hamlib :** vérifiez que rigctld est en cours d'exécution et que le port TCP est correct.
- Couleur de l'étiquette de statut : blanc = connecté, jaune = connexion/désactivé, rouge = erreur.
- Les toasts de reconnexion supprimés sont normaux — CQOps réessaie en arrière-plan.

### WSJT-X ne journalise pas automatiquement

- Vérifiez les paramètres UDP de WSJT-X : Settings → Reporting → UDP Server.
- WSJT-X doit être en version 2.6 ou plus récente.
- L'étiquette de statut doit être blanche (défaut) lorsque WSJT-X est en cours d'exécution.

### Le téléversement Wavelog échoue

- Vérifiez l'URL, la clé API et l'ID de profil de station dans la configuration.
- Étiquette de statut : blanc = accessible, jaune = désactivé/pas d'internet, rouge = erreur.
- Les erreurs de téléversement sont affichées sous forme de toasts ; les QSOs restent enregistrés localement.
- Les échecs individuels de QSO ne bloquent pas le reste du lot.

### Problèmes de fichier de configuration

- Configuration : `~/.config/cqops/config.yaml` (Linux/macOS) ou `%APPDATA%\cqops\config.yaml` (Windows).
- Identifiants : `secrets.enc` dans le même répertoire.
- Si la configuration est corrompue, supprimez-la et redémarrez — l'assistant en créera une nouvelle.
- Le champ `last_fetched_id` n'apparaît qu'après un téléchargement Wavelog réussi.

### Performances

- Désactivez le rendu de carte et le panneau solaire dans les paramètres General.
- Fermez les onglets inutilisés (DXC, PSK).
- Lancez avec `--offline` si le réseau n'est pas fiable.

### Signaler des bugs

Activez le **mode Debug** avant de reproduire un problème — F9 → General → Debug, ou définissez `debug: true` dans la configuration. Les logs complets sont écrits dans le répertoire de logs spécifique à la plateforme.

Signalez les problèmes sur [GitHub Issues](https://github.com/szporwolik/cqops/issues) avec :
- Version de CQOps (`cqops --version`)
- Système d'exploitation et émulateur de terminal
- Étapes pour reproduire
- Log de debug
