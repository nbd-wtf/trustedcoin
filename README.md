<a href="https://nbd.wtf"><img align="right" height="196" src="https://user-images.githubusercontent.com/1653275/194609043-0add674b-dd40-41ed-986c-ab4a2e053092.png" /></a>

## The `trustedcoin` plugin

A plugin that uses block explorers (blockstream.info, mempool.space, blockchair.com and blockchain.info) as backends instead of your own Bitcoin node.

This isn't what you should be doing, but sometimes you may need it.

(Remember this will download all blocks Core Lightning needs from blockchain.info or blockchair.com in raw, hex format.)

## How to install

This is distributed as a single binary for your delight (or you can compile it yourself with `go get`, or ask me for binaries for other systems if you need them).

[Download it](https://github.com/fiatjaf/trustedcoin/releases), call `chmod +x <binary>` and put it in `~/.lightning/plugins` (create that directory if it doesn't exist).

You only need the binary you can get in [the releases page](https://github.com/fiatjaf/trustedcoin/releases), nothing else.

Then add the following line to your `~/.lightning/config` file:

```
disable-plugin=bcli
```

This disables the default Bitcoin backend plugin so `trustedcoin` can take its place.

If you're running on `testnet`, `signet` or `liquid` trustedcoin will also work automatically.

## Using `bitcoind`

If you have `bitcoind` available and start `lightningd` with the settings `bitcoin-rpcuser`, `bitcoin-rpcpassword`, and optionally `bitcoin-rpcconnect` (defaults to 127.0.0.1) and `bitcoin-rpcport` (defaults to 8332 on mainnet etc.), then `trustedcoin` will try to use that and fall back to the explorers if it is not available.

### Extra: how to bootstrap a Lightning node from scratch, without Bitcoin Core, on Ubuntu amd64

```
add-apt-repository ppa:lightningnetwork/ppa
apt update
apt install lightningd
mkdir -p ~/.lightning/plugins
echo 'disable-plugin=bcli' >> .lightning/config
cd ~/.lightning/plugins
wget https://github.com/nbd-wtf/trustedcoin/releases/download/v0.8.1/trustedcoin-v0.8.1-linux-amd64.tar.gz
tar -xvf trustedcoin-v0.8.1-linux-amd64.tar.gz
cd
lightningd
```
