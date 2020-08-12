## The `trustedcoin` plugin

A plugin that uses block explorers (blockstream.info, mempool.space, blockchair.com and blockchain.info) as backends instead of your own Bitcoin node.

This isn't what you should be doing, but sometimes you may need it.

(Remember this will download all blocks c-lightning needs from blockchain.info or blockchair.com in raw, hex format.)

## How to install

This is distributed as a single binary for your delight (or you can compile it yourself with `go get`, or ask me for binaries for other systems if you need them).

[Download it](https://github.com/fiatjaf/trustedcoin/releases), call `chmod +x <binary>` and put it in `~/.lightning/plugins` (create that directory if it doesn't exist).

You only need the binary you can get in [the releases page](https://github.com/fiatjaf/trustedcoin/releases), nothing else.

Then add the following line to your `~/.lightning/config` file:

```
disable-plugin=bcli
```

This disables the default Bitcoin backend plugin so `trustedcoin` can take its place.

### Extra: how to bootstrap a Lightning node from scratch, without Bitcoin Core, on Ubuntu amd64

```
add-apt-repository ppa:lightningnetwork/ppa
apt update
apt install lightningd
mkdir -p ~/.lightning/plugins
echo 'disable-plugin=bcli' >> .lightning/config
cd ~/.lightning/plugins
wget https://github.com/fiatjaf/lightningd-gjson-rpc/releases/download/trustedcoin-v0.2.4/trustedcoin_linux_amd64
chmod +x trustedcoin_linux_amd64
cd
lightningd
```

## Built with [![GoDoc](https://godoc.org/github.com/fiatjaf/lightningd-gjson-rpc?status.svg)](https://godoc.org/github.com/fiatjaf/lightningd-gjson-rpc)
