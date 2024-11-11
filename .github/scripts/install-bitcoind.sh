#!/bin/sh

set -xeu

DIRNAME="bitcoin-${BITCOIN_CORE_VERSION}"
FILENAME="${DIRNAME}-x86_64-linux-gnu.tar.gz"

cd "${HOME}"
wget -q "https://bitcoincore.org/bin/bitcoin-core-${BITCOIN_CORE_VERSION}/${FILENAME}"
tar -xf "${FILENAME}"
sudo mv "${DIRNAME}"/bin/* "/usr/local/bin"
rm -rf "${FILENAME}" "${DIRNAME}"
