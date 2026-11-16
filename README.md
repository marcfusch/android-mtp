# android-mtp

Mounts an Android phone at `~/Android` on macOS over MTP, using macFUSE. The phone is mounted when plugged in with File transfer enabled, and unmounted when unplugged.

## Install

Requires macOS and [Homebrew](https://brew.sh).

```sh
git clone https://github.com/marcfusch/android-mtp
android-mtp/install.sh
```

The script installs macFUSE, Go, libusb and pkg-config, builds the `android` binary into `~/.local/bin` and loads a launchd agent. macFUSE must be allowed once in System Settings, Privacy & Security.

## Usage

| Command | Description |
| :-- | :-- |
| `android` | Open `~/Android`, remounting the phone if it was ejected |
| `android off` | Eject until the next plug |
| `android install` | Install the binary and the agent |
| `android uninstall` | Remove the binary and the agent |

Logs: `~/Library/Logs/android-mtp.log`.

## Limitations

* One phone at a time.
* A locked phone does not answer MTP; it is mounted once unlocked.
* While the agent runs, Image Capture cannot access USB cameras. `android uninstall` restores it.
* Throughput is about 30 MB/s on USB 2.

## How it works

The agent watches IOKit for the phone's MTP interface and serves it through [go-mtpfs](https://github.com/hanwen/go-mtpfs), vendored in `third_party/go-mtpfs` with fixes for transfer speed, out of order writes, zero length packets and timezones. It suspends `ptpcamerad`, which otherwise claims the phone on plug, and normalizes file names to NFC.

## Development

```sh
export PKG_CONFIG_PATH=$(brew --prefix libusb)/lib/pkgconfig
go test ./... github.com/hanwen/go-mtpfs/...
```

Set `ANDROID_MTP_DEBUG=1` to trace FUSE and MTP operations.
