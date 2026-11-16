# android-mtp

Monte un téléphone Android dans `~/Android` sur macOS, via MTP et macFUSE. Le téléphone apparaît dans la barre latérale du Finder dès qu'il est branché en mode Transfert de fichiers, et se démonte quand on le débranche.

## Installation

```sh
git clone https://github.com/marcfusch/android-mtp ~/Documents/git/android-mtp
~/Documents/git/android-mtp/install.sh
```

`install.sh` installe macFUSE, Go, libusb et pkg-config avec Homebrew, compile la commande puis lance `android install`. Celle ci copie le binaire dans `~/.local/bin` et charge un agent launchd qui surveille l'USB. Au premier lancement, macFUSE doit être autorisé dans Réglages Système, Confidentialité et sécurité.

## Utilisation

| Commande | Effet |
| :-- | :-- |
| `android` | ouvre `~/Android`, en remontant le téléphone s'il a été éjecté |
| `android off` | éjecte ; le téléphone n'est remonté qu'au prochain branchement |
| `android install` | installe la commande et l'agent |
| `android uninstall` | retire la commande et l'agent |
| `android watch` | surveille l'USB (lancé par l'agent) |

Éjecter depuis le Finder revient à `android off`. Le journal est dans `~/Library/Logs/android-mtp.log`.

## Fonctionnement

| Fichier | Rôle |
| :-- | :-- |
| `main.go` | les commandes |
| `watch.go` | la boucle de l'agent et `tracker`, qui décide à chaque tour s'il faut monter, démonter ou attendre |
| `phone.go` | la session MTP et le serveur FUSE d'un téléphone monté |
| `usb.go` | l'état de l'interface MTP lu dans IOKit |
| `ptpcamerad.go` | la mise à l'écart de Transfert d'images |
| `mounts.go` | la table des montages |
| `rawfs.go` | l'adaptation à macOS : noms Unicode et attributs étendus |
| `launchd.go` | l'installation de l'agent |

Le protocole MTP et le système de fichiers viennent de [go-mtpfs](https://github.com/hanwen/go-mtpfs) v1.0.1, la même base qu'OpenMTP, copié dans `third_party/go-mtpfs` avec ces correctifs :

| Correctif | Effet |
| :-- | :-- |
| attente de 10 ms au lieu du délai complet pour le paquet USB vide envoyé après chaque envoi | création d'un fichier en 0,02 s au lieu de 6 s |
| écritures consécutives regroupées en envois de 1 Mio, transferts USB de 256 Kio | environ 28 Mo/s au lieu de 4, le plafond de l'USB 2 |
| dates lues dans le fuseau local, celui qu'envoie Android | plus de décalage horaire sur les fichiers |

macOS passe les noms en Unicode décomposé alors que le téléphone les garde précomposés : `rawfs.go` les convertit, sinon tout nom accentué est introuvable. Il accepte aussi les attributs étendus sans les garder, car MTP n'en a pas et le Finder abandonne une copie s'il ne peut pas les écrire.

À la connexion, chaque transfert USB a un délai de 3 s : un téléphone verrouillé renvoie une erreur au lieu de bloquer. Une session restée ouverte est refermée, et l'USB est réinitialisé si besoin. Une fois connecté, le délai passe à 30 s, car un transfert qui expire en plein envoi fige le téléphone jusqu'au prochain débranchement. Pour la même raison, l'agent ne tue jamais de session en cours : il démonte proprement, y compris quand launchd l'arrête, et réessaie toutes les 2 s tant que le téléphone ne répond pas.

macOS lance `ptpcamerad` (Transfert d'images) au branchement : il prend le téléphone pour lui seul et ne le rend jamais. launchd le relance aussitôt tué, même désactivé. L'agent le gèle avec SIGSTOP : il reste en place sans rien prendre. S'il tient déjà le téléphone, l'agent doit le tuer et demande de rebrancher. Transfert d'images ne voit plus d'appareil photo tant que l'agent tourne ; `android uninstall` le libère.

## Tests

```sh
PKG_CONFIG_PATH=$(brew --prefix libusb)/lib/pkgconfig go test ./... github.com/hanwen/go-mtpfs/...
```
