# android-mtp

Monte un téléphone Android dans `~/Android` sur macOS, via MTP et macFUSE. Le téléphone apparaît dans la barre latérale du Finder dès qu'il est branché en mode Transfert de fichiers, et se démonte quand on le débranche.

## Installation

Il faut macOS, [Homebrew](https://brew.sh) et un téléphone Android.

```sh
git clone https://github.com/marcfusch/android-mtp
android-mtp/install.sh
```

`install.sh` installe macFUSE, Go, libusb et pkg-config, compile la commande `android` puis lance `android install`, qui la copie dans `~/.local/bin` et charge un agent launchd. Au premier lancement, macFUSE doit être autorisé dans Réglages Système, Confidentialité et sécurité. Pour mettre à jour : `git pull`, puis `install.sh` de nouveau.

## Utilisation

Branche le téléphone, déverrouille le et choisis Transfert de fichiers : il est monté en quelques secondes, avec une notification. Verrouillé, il ne répond pas en MTP ; l'agent réessaie toutes les 2 s et le monte dès qu'il est déverrouillé.

| Commande | Effet |
| :-- | :-- |
| `android` | ouvre `~/Android`, en remontant le téléphone s'il a été éjecté |
| `android off` | éjecte, comme le Finder ; le téléphone n'est remonté qu'au prochain branchement |
| `android install` | installe la commande et l'agent |
| `android uninstall` | retire la commande et l'agent |
| `android watch` | la boucle de l'agent, lancée par launchd |

Le journal est dans `~/Library/Logs/android-mtp.log`.

## Limites

* Un seul téléphone à la fois. Chaque stockage du téléphone est un dossier à la racine de `~/Android`.
* Tant que l'agent tourne, Transfert d'images ne voit plus d'appareil photo branché en USB (voir plus bas). `android uninstall` le rétablit.
* Le débit plafonne vers 30 Mo/s, la limite de l'USB 2 en MTP. Chaque fichier coûte en plus environ 0,1 s, surtout le temps que le téléphone l'enregistre.

## Fonctionnement

| Fichier | Rôle |
| :-- | :-- |
| `main.go` | les commandes |
| `watch.go` | la boucle de l'agent : état USB, montage, notifications, signaux |
| `tracker.go` | la décision de chaque tour, sans effet de bord : monter, démonter ou attendre |
| `phone.go` | la session MTP et le serveur FUSE d'un téléphone monté |
| `rawfs.go` | l'adaptation à macOS : noms Unicode et attributs étendus |
| `usb.go` | l'interface MTP du téléphone et son propriétaire, lus dans IOKit |
| `ptpcamerad.go` | la mise à l'écart de Transfert d'images |
| `mounts.go` | la table des montages |
| `launchd.go` | l'installation de l'agent |

**Connexion.** À la connexion, chaque transfert USB a un délai de 3 s, pour qu'un téléphone verrouillé renvoie une erreur au lieu de bloquer ; une session restée ouverte est refermée et l'USB réinitialisé si besoin. Une fois connecté, le délai passe à 30 s : un transfert qui expire en plein envoi fige le téléphone jusqu'au prochain débranchement. Pour la même raison, l'agent ne coupe jamais une session en cours et démonte proprement, y compris quand launchd l'arrête.

**Transfert d'images.** macOS lance `ptpcamerad` au branchement : il prend le téléphone pour lui seul et ne le rend jamais. launchd le relance aussitôt tué, même désactivé. L'agent le gèle donc avec SIGSTOP : il reste en place sans rien prendre. S'il tient déjà le téléphone, l'agent doit le tuer et demande de rebrancher.

**macOS.** macOS passe les noms en Unicode décomposé alors que le téléphone les garde précomposés : `rawfs.go` les convertit, sinon tout nom accentué est introuvable. Il accepte aussi les attributs étendus sans les garder, car MTP n'en a pas et le Finder abandonne une copie s'il ne peut pas les écrire.

**go-mtpfs.** Le protocole MTP et le système de fichiers viennent de [go-mtpfs](https://github.com/hanwen/go-mtpfs) v1.0.1, copié dans `third_party/go-mtpfs` avec ces correctifs :

| Correctif | Pourquoi |
| :-- | :-- |
| le paquet USB vide qui suit un envoi n'attend que 10 ms quand il est en trop | Android ne le lit pas, et chaque création de fichier attendait le délai complet : 6 s au lieu de 0,02 s |
| ce paquet garde le délai complet quand il termine un envoi d'un multiple de 512 octets | le téléphone en a besoin pour finir de lire ; sans lui, l'envoi bloque |
| les écritures consécutives sont regroupées en envois de 1 Mio, par transferts USB de 256 Kio | chaque envoi a un coût fixe : 28 Mo/s au lieu de 4 |
| un fichier est agrandi avant une écriture au delà de sa fin | macOS livre parfois la fin d'un gros fichier en premier, et Android refuse cette écriture sans lire ses données, ce qui fige le téléphone |
| seul un descripteur qui a écrit termine l'édition, et les écritures en attente partent aussi à la libération | macOS ouvre en lecture un fichier en cours de copie, pour sa miniature ou Spotlight |
| une réponse du téléphone arrivée sans lecture des données est lue et journalisée | la session reste utilisable et le journal dit pourquoi |
| les dates sont lues dans le fuseau local | Android envoie l'heure locale sans fuseau ; elles étaient décalées |

## Développement

```sh
export PKG_CONFIG_PATH=$(brew --prefix libusb)/lib/pkgconfig
go test ./... github.com/hanwen/go-mtpfs/...
```

Pour tracer chaque opération FUSE et MTP, arrête l'agent (`launchctl bootout gui/$(id -u)/local.android-mtp`) puis lance `ANDROID_MTP_DEBUG=1 android watch`.
