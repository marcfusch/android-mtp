#!/bin/zsh
# Installe les dépendances, compile android puis lance android install.
set -eu

[[ $(uname) == Darwin ]] || { echo "macOS uniquement" >&2; exit 1; }
command -v brew >/dev/null || { echo "Homebrew est requis : https://brew.sh" >&2; exit 1; }

brew list --cask macfuse >/dev/null 2>&1 || brew install --cask macfuse
for f in go libusb pkg-config; do
  brew list $f >/dev/null 2>&1 || brew install $f
done

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
(cd "${0:A:h}" && PKG_CONFIG_PATH=$(brew --prefix libusb)/lib/pkgconfig go build -o "$tmp/android" .)
"$tmp/android" install
