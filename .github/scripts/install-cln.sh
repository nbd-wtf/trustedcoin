#!/bin/sh

set -xeu

DIRNAME="usr"
FILENAME="clightning-v${CORE_LIGHTNING_VERSION}-Ubuntu-24.04.tar.xz"

cd "${HOME}"
wget -q "https://github.com/ElementsProject/lightning/releases/download/v${CORE_LIGHTNING_VERSION}/${FILENAME}"
tar -xf "${FILENAME}"
sudo cp -r "${DIRNAME}"/bin     /usr/local
sudo cp -r "${DIRNAME}"/libexec /usr/local/libexec
rm -rf "${FILENAME}" "${DIRNAME}"
