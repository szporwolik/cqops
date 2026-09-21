---
title: Manuel utilisateur de CQOps
description: Guide pratique pour configurer CQOps et enregistrer les contacts en station ou sur le terrain
---

# Manuel utilisateur de CQOps

CQOps est un logiciel de carnet de trafic radioamateur piloté au clavier, adapté à la station fixe, au portable, au radioclub et à la participation occasionnelle aux concours. Les QSO sont d’abord enregistrés sur votre ordinateur ; les services Internet sont facultatifs. Commencez par la saisie manuelle, puis ajoutez la commande du poste et les services en ligne selon vos besoins.

Les noms des menus et champs ci-dessous correspondent à l’interface anglaise. Les raccourcis dépendent de l’écran actif : consultez la barre d’aide ou **?**.

## Sommaire

1. [Installation](#installation)
2. [Configuration initiale](#setup)
3. [Premier QSO](#first-qso)
4. [Écrans et état](#screens)
5. [Saisie quotidienne](#logging)
6. [Profils de station](#profiles)
7. [Carnet de trafic et sauvegardes](#logbook)
8. [Poste et modes numériques](#radio)
9. [Services en ligne](#online)
10. [GPS et APRS](#position)
11. [Opération en portable](#portable)
12. [Concours](#contests)
13. [CQOps Live](#dashboard)
14. [Raccourcis clavier](#keys)
15. [Dépannage et assistance](#help)

<a id="installation"></a>

## Installation

Téléchargez CQOps depuis la [page des versions](https://github.com/szporwolik/cqops/releases). Le terminal doit mesurer au moins 75 × 24 caractères ; 80 × 43 ou plus est préférable.

| Système | Installation |
|---|---|
| Windows | Téléchargez `cqops-setup.exe` ou décompressez `cqops-windows-portable.zip` pour une utilisation sans installation. Windows Terminal est recommandé. |
| Debian, Ubuntu, Linux Mint, Pop!_OS | Prenez le `.deb` adapté : `amd64` pour la plupart des PC Intel/AMD, `arm64` pour ARM 64 bits, `armhf` pour Raspberry Pi OS 32 bits. Ouvrez-le avec l’installateur de paquets. |
| Fedora, RHEL, Rocky, AlmaLinux | Utilisez les commandes du dépôt ci-dessous. |
| Arch, Manjaro, CachyOS | Installez le paquet AUR avec `paru -S cqops-bin` ou `yay -S cqops-bin`. |
| Autres systèmes Linux | Téléchargez et décompressez l’archive Linux `.tar.gz` correspondant au processeur. |
| macOS | Prenez `cqops-darwin-arm64` pour Apple Silicon ou `cqops-darwin-amd64` pour Intel. Utilisez les commandes ci-dessous. |

Sur les systèmes Debian, l’installation depuis le dépôt est également possible :

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

Sur les systèmes Fedora :

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

Sur macOS, exécutez ces commandes dans le dossier de téléchargement en remplaçant `FILE` par le nom exact du fichier :

```bash
chmod +x FILE
sudo mv FILE /usr/local/bin/cqops
```

Lancez `cqops` ou le programme portable décompressé. `cqops --offline` démarre hors ligne, `cqops --version` affiche la version et `cqops --help` les options de démarrage. Exportez vos carnets avant une mise à jour.

<a id="setup"></a>

## Configuration initiale

L’assistant demande le nom du carnet, l’indicatif de station, le locator Maidenhead et le continent. L’**indicatif de station** est celui utilisé en émission ; le **profil opérateur** identifie la personne aux commandes.

**Ctrl+A** affiche les références de station facultatives et les zones CQ/ITU. Votre référence SOTA/POTA/WWFF va dans les paramètres de station/carnet ; les références du formulaire QSO concernent le correspondant. Réglez ensuite la région IARU dans **F9 → Logbooks**.

Créez un profil de poste avec nom, antenne et puissance. Choisissez **None** pour saisir fréquence et mode manuellement, **flrig** ou **Hamlib**. Ajoutez les connexions facultatives après avoir vérifié la saisie de base.

**Tab / Shift+Tab** change de champ, **Space** modifie les options et **Save & Next** poursuit l’assistant. **Esc** revient en arrière ; **F10** quitte. Vérifiez le résumé et enregistrez. CQOps détecte le fuseau du système ; les dates et heures des QSO sont en UTC. Vérifiez l’horloge avant de trafiquer.

<a id="first-qso"></a>

## Premier QSO

1. Appuyez sur **F1**. Vérifiez carnet, indicatif de station, opérateur, poste et concours actifs.
2. Saisissez l’indicatif du correspondant. **Ins** lance la recherche si elle est configurée.
3. Vérifiez date/heure UTC, fréquence en MHz, bande, mode et reports émis/reçus.
4. Ajoutez si utile nom, QTH, locator, référence ou commentaire.
5. Appuyez sur **Enter**. Le contact apparaît dans Recent QSOs.

Si **DUPE!** demande confirmation, appuyez à nouveau sur **Enter** pour conserver le QSO, ou sur **Esc** pour annuler la confirmation. L’avertissement invite à vérifier le contact, pas nécessairement à le supprimer.

<a id="screens"></a>

## Écrans et état

| Touche | Écran | Utilisation |
|---|---|---|
| F1 | QSO | Saisie et derniers contacts |
| F2 | Partner | Informations du correspondant, carte, statistiques, photo |
| F3 | APRS | Stations proches |
| F4 | DX Cluster | Spots et filtres |
| F5 | PSK Reporter | Rapports de réception numérique |
| F6 | References | Recherche SOTA, POTA, WWFF, IOTA |
| F7 | Band Plan | Fréquences et préréglages |
| F8 | Logbook | Modification, import, export, synchronisation |
| F9 | Configuration | Paramètres de station et services |
| F10 | Quit | Quitter CQOps |

La barre supérieure indique la configuration active, l’heure locale (**L**) et UTC (**Z**). Le blanc signifie normalement actif, le jaune désactivé/connexion/attente, le rouge une erreur. WSJT est mis en évidence pendant l’émission. **WL!** signale une ancienne clé Wavelog non prise en charge.

<a id="logging"></a>

## Saisie quotidienne

Utilisez **Tab / Shift+Tab** entre les champs et **PgUp / PgDn** pour changer bande, mode ou sous-mode. **Shift+Backspace** efface le champ ; **Del** efface le formulaire. En split, vérifiez **Freq RX**.

**Keep** conserve le commentaire après enregistrement. **Retain** conserve tout le formulaire : vérifiez indicatif, heure, reports et références avant le QSO suivant. Les champs d’échange apparaissent seulement avec un concours actif. **SIG / SIG Info** permet d’indiquer d’autres groupes d’intérêt particulier.

Avec deux locators connus, CQOps affiche distance et azimut. Une adresse de callbook peut être celle du domicile plutôt que du site portable actuel : vérifiez-la. Les indicateurs de nouvel indicatif, nouveau DXCC et doublon facilitent l’évaluation.

**F6** recherche les références par nom ou identifiant et peut renseigner celle du correspondant. **F7** affiche les plans de bandes et peut accorder un poste connecté. Ces entrées sont des aides, pas une autorisation d’émettre : respectez vos droits et le plan local.

Trois favoris communs enregistrent fréquence, mode et bande :

| Emplacement | Rappeler | Enregistrer les valeurs |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

<a id="profiles"></a>

## Profils de station

Créez carnets, opérateurs, postes et concours dans leurs menus **F9** ; **Ins** ajoute une entrée. Depuis l’écran QSO :

| Raccourci | Changement |
|---|---|
| Ctrl+L | Carnet |
| Ctrl+O | Opérateur |
| Ctrl+R | Poste |
| Ctrl+C | Concours |

Les carnets gardent des données de station et réglages Wavelog/APRS distincts. Le profil opérateur identifie la personne aux commandes ; son indicatif figure dans ADIF `OPERATOR`. Les profils de poste gardent matériel, puissance, commande radio/rotor et réglages WSJT-X. Contrôlez la barre d’état après chaque changement, surtout en saisie numérique automatique.

Les autres menus **F9** couvrent affichage, unités, fuseau horaire, callbooks, intégrations et notifications sonores.

<a id="logbook"></a>

## Carnet de trafic et sauvegardes

Dans **F8**, sélectionnez un QSO et appuyez sur **Enter** ou **e** pour le modifier. Enregistrez avec **Enter** et confirmez. **Delete** supprime le contact sélectionné. Sauvegardez avant une opération globale ; **Ctrl+P** efface tous les QSO, ce n’est pas une recherche.

| Raccourci F8 | Action |
|---|---|
| Ctrl+I | Importer ADIF, vérifier les enregistrements, ignorer les doublons |
| Ctrl+E | Exporter tous les contacts ou une sélection filtrée par concours |
| Ctrl+W | Envoyer les contacts non transmis à Wavelog |
| Alt+W | Télécharger depuis Wavelog |

Vérifiez le bilan d’import et la sélection d’export. Les contacts importés pourront ensuite être envoyés à Wavelog. CQOps prend en charge ADIF 3.1.7 et conserve identifiants de concours et échanges. Gardez une sauvegarde par carnet, de préférence sur un autre appareil. ADIF sauvegarde les contacts, pas tous les paramètres ni les identifiants d’accès.

La configuration est dans `~/.config/cqops/config.yaml` sous Linux/macOS, et `%APPDATA%\cqops\config.yaml` sous Windows. Les accès sont stockés séparément dans `secrets.enc` ; ressaisissez-les après un changement d’ordinateur. Ne commencez pas un dépannage en supprimant la configuration.

<a id="radio"></a>

## Poste et modes numériques

### Commande du poste

Dans **F9 → Rigs**, choisissez flrig ou Hamlib et faites correspondre les réglages. Lancez d’abord flrig ou `rigctld`. flrig utilise généralement `localhost:12345`. Les lectures disponibles de fréquence, mode, split et puissance dépendent du poste. Avec **None**, saisissez-les manuellement.

### WSJT-X

Utilisez WSJT-X 2.6 ou ultérieur. Faites correspondre **Settings → Reporting → UDP Server** aux réglages UDP du profil de poste CQOps actif. Enregistrez un QSO d’essai terminé dans WSJT-X et vérifiez son apparition dans CQOps.

Les QSO reçus utilisent le carnet et le concours actifs ; les doublons sont ignorés. Avant la session, contrôlez opérateur et indicateur WSJT. CQOps avertit si les opérateurs diffèrent. Un envoi Wavelog peut suivre s’il est configuré. Choisissez le bon Mode/Submode ; FT8 est exporté en FT8, FT4/FT2 en MFSK avec le sous-mode correspondant.

### Commande du rotor

La commande Hamlib `rotctld` est expérimentale. Vérifiez sens et limites physiques avant utilisation. Prévoyez un arrêt sûr : des réglages incorrects peuvent endommager antenne, rotor ou ligne d’alimentation.

| Raccourci | Action |
|---|---|
| Alt+, / Alt+. | Azimut −5° / +5° |
| Alt+' / Alt+; | Élévation −5° / +5° |
| Alt+\ | Orienter vers l’azimut calculé |
| Alt+/ | Arrêter le mouvement |

<a id="online"></a>

## Services en ligne

### Callbooks

Réglez fournisseurs et priorité dans **F9 → Callbook**, puis utilisez **Ins** sur le formulaire QSO. CQOps les consulte dans l’ordre. La recherche de l’indicatif de base peut retirer préfixes/suffixes portables ; vérifiez le lieu renvoyé.

| Fournisseur | Accès |
|---|---|
| QRZ.com | Abonnement XML et identifiants |
| HamQTH | Compte gratuit |
| QRZ.RU | Accès API distinct du compte du site |
| Callook.info | Indicatifs américains ; sans compte |

**F2** affiche le correspondant. Les photos dépendent du fournisseur et du terminal ; **Kitty Graphics**, expérimental dans General, nécessite un terminal compatible tel que Kitty, Ghostty ou WezTerm.

### Wavelog

Définissez URL, jeton API v2 (`wl2_…`) et profil de station par carnet. Les anciennes clés v1 sont refusées. La sélection d’une station Wavelog peut remplir les données locales : vérifiez indicatif, locator et références avant d’enregistrer.

Les QSO sont d’abord sauvegardés localement. Réessayez un envoi échoué avec **F8 → Ctrl+W** ; **Alt+W** télécharge. L’ouverture d’un QSO lié pour modification peut actualiser ses données depuis Wavelog. Les modifications et suppressions en ligne affectent aussi la copie distante. Lisez la confirmation, surtout hors ligne ; une modification locale n’est pas nécessairement parvenue à Wavelog.

### DX Cluster et propagation

Configurez DX Cluster dans Integrations, puis ouvrez **F4**. **b / c / m / t** filtrent bande, continent du spotteur, mode et ancienneté. **Backspace** efface les filtres. **Enter** remplit le formulaire QSO, accorde le poste connecté et revient à F1 ; **Space** accorde sans quitter le cluster.

Sur F1, **Ctrl+S** ouvre l’envoi de spot et **Ctrl+P** reprend l’indicatif du spot affiché le plus proche. Vérifiez-le avant l’envoi. **F5** montre des rapports PSK Reporter, pas une garantie de propagation actuelle. Solar présente les conditions HamQSL ; les valeurs en cache peuvent être anciennes.

<a id="position"></a>

## GPS et APRS

### GPS

Configurez un GPS série ou GPSD dans Integrations. Activez **Grid from GPS** dans les réglages station/carnet pour utiliser ce locator pour QSO, azimuts, APRS et tableau de bord. Rouge signifie erreur, jaune absence de position, blanc position acquise. Vérifiez le locator avant de trafiquer. Choisissez 6, 8 ou 10 caractères ; davantage de caractères ne garantit pas une meilleure précision du récepteur.

### APRS

| Service | Connexion |
|---|---|
| APRS-IS | Serveur APRS Internet |
| KISS | TNC matériel série et poste |
| KISS Server | TNC TCP tel que Dire Wolf ; utilisable localement |

Choisissez le service dans **F9 → Integrations → APRS**. Réglez indicatif/SSID, symbole, commentaire, portée et intervalle dans **F9 → Logbooks → [active logbook] → APRS**. N’activez **APRS TX** et **Send beacons** que si vous souhaitez émettre. La réception seule affiche **APRS-RX**. Les balises révèlent votre position : vérifiez-la et prenez en compte leurs destinataires.

L’intervalle automatique est d’au moins cinq minutes. **F3** affiche les stations entendues récemment : flèches pour sélectionner, **Enter** pour remplir QSO, **d / t / s** pour filtrer distance/ancienneté/type, **Backspace** pour effacer les filtres et **b** pour envoyer immédiatement une balise configurée. Les balises GPS nécessitent **Grid from GPS** et une position valide.

<a id="portable"></a>

## Opération en portable

Avant le départ, sélectionnez le carnet portable ; vérifiez indicatif, locator, référence d’activation, poste, antenne et puissance. Testez toute la station, puis lancez CQOps en ligne pour actualiser références et préfixes. Vérifiez que **F6** trouve les références utiles. Exportez une sauvegarde.

La saisie locale fonctionne sans Internet. `cqops --offline` ignore les fonctions réseau ; ne comptez pas sur les recherches en direct ni la synchronisation. Testez vos équipements en réseau local avec ce mode de démarrage avant de partir. Les données en cache peuvent être périmées.

Au retour, vérifiez total des QSO et références, exportez ADIF, gardez une sauvegarde et envoyez les contacts en attente à Wavelog si utilisé. Vérifiez le format demandé par chaque programme de diplômes ; convertissez si nécessaire.

<a id="contests"></a>

## Concours

CQOps gère les concours occasionnels, échanges, numéros progressifs et cadence. Ce n’est pas un système complet de calcul des scores ou de soumission. Préférez un logiciel spécialisé pour une activité avancée.

Dans **F9 → Contests**, appuyez sur **Ins**, puis réglez nom, date, identifiant ADIF du concours, numéro de départ et modèles d’échange émis/reçu.

| Marqueur | Valeur |
|---|---|
| `@rst` | Report émis ou reçu |
| `@serial` | Numéro progressif |
| `@cqz` / `@mycqz` | Zone CQ du correspondant / la vôtre |
| `@itu` / `@myitu` | Zone ITU du correspondant / la vôtre |
| `@grid` / `@mygrid` | Locator du correspondant / le vôtre |

Sur **F1**, **Ctrl+C** change de concours. Vérifiez échange et prochain numéro avant d’émettre. La barre d’état montre total, prochain numéro et durées ; une fenêtre plus large montre davantage de statistiques de cadence. Revenez au mode sans concours après la fin.

Pour exporter, ouvrez **F8**, choisissez le filtre concours avec **Ctrl+C**, puis **Ctrl+E** et vérifiez la sélection. Le fichier est en ADIF, pas en Cabrillo. Respectez les règles de format et de soumission de l’organisateur.

<a id="dashboard"></a>

## CQOps Live

Activez **F9 → Integrations → HTTP Server**, puis enregistrez avec **Ctrl+S**. Sur l’ordinateur CQOps, ouvrez `http://localhost:8073`.

L’adresse par défaut `0.0.0.0` permet l’accès sur le réseau local, sous réserve du pare-feu. Depuis un autre appareil, utilisez l’IP du PC CQOps et le port `8073`. `127.0.0.1` limite l’accès au PC CQOps. Restez sur un réseau de confiance ; ne redirigez pas le port vers Internet.

Le tableau de bord actualise automatiquement contact en cours, cartes QSO, derniers contacts, cadences, opérateurs, APRS et données de propagation/météo disponibles. Les couches Internet peuvent manquer hors ligne. Header 1, Header 2, Logo URL et Event Start personnalisent l’affichage ; la date de début filtre statistiques et listes de QSO.

<a id="keys"></a>

## Raccourcis clavier

| Écran | Touches | Action |
|---|---|---|
| Général | ? / Esc / F10 | Aide / retour / quitter |
| QSO | Tab / Shift+Tab | Champ suivant / précédent |
| QSO | Enter / Ins | Enregistrer / rechercher |
| QSO | Shift+Backspace / Del | Effacer champ / formulaire |
| QSO | Ctrl+L / Ctrl+O / Ctrl+R / Ctrl+C | Changer carnet / opérateur / poste / concours |
| Carnet | ↑ / ↓, PgUp / PgDn, Home / End | Sélection, page, première / dernière ligne |
| Carnet | Enter ou e / Delete | Modifier / supprimer le QSO sélectionné |
| Carnet | Ctrl+I / Ctrl+E | Importer / exporter ADIF |
| Carnet | Ctrl+W / Alt+W | Envoyer / télécharger Wavelog |
| Carnet | Ctrl+C / Backspace | Filtre concours / effacer recherche |

Les raccourcis dépendent de l’écran : **Ctrl+C ne quitte pas CQOps**. Sur un portable, les touches de fonction peuvent nécessiter **Fn**. Si le terminal intercepte un raccourci, vérifiez ses réglages clavier et la barre d’aide CQOps.

<a id="help"></a>

## Dépannage et assistance

| Problème | Premières vérifications |
|---|---|
| Démarrage ou écran incomplet | Taille du terminal, Windows Terminal sous Windows, essai `cqops --offline` |
| Poste déconnecté | Profil actif, flrig/rigctld lancé, modèle, port série, débit, hôte/port, port série occupé par un autre logiciel |
| QSO WSJT-X absent | Réglages UDP, indicateur WSJT, QSO réellement enregistré dans WSJT-X, carnet actif |
| Erreur Wavelog | URL, jeton `wl2_`, profil de station, Internet ; les QSO locaux sont conservés |
| Pas de position GPS | Port/débit ou adresse GPSD, ciel dégagé, position valide, Grid from GPS activé |
| APRS sans balise | Carnet, APRS TX et Send beacons, indicatif/SSID, TNC/poste ou Internet |
| Tableau de bord inaccessible | Serveur activé, IP/port corrects, pare-feu ; localhost désigne l’appareil où tourne le navigateur |

Utilisez **F9** avant de modifier des fichiers de paramètres. Ressaisissez les accès si CQOps signale un problème de secrets ou après un changement de PC.

Si nécessaire, activez **F9 → General → Debug**, reproduisez le problème sans risque et récupérez le journal de diagnostic pertinent ; désactivez ensuite le débogage.

| Système | Journaux de diagnostic |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

Signalez les problèmes dans [GitHub Issues](https://github.com/szporwolik/cqops/issues), avec version CQOps, système, terminal, étapes et journal pertinent. Retirez mots de passe, jetons API et informations privées avant tout partage.
