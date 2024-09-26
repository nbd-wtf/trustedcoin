package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

var heightCache = make(map[int64]string)

func getBlock(height int64) (block, hash string, err error) {
	hash, err = getHash(height)
	if err != nil {
		return
	}
	if hash == "" /* block unavailable */ {
		return
	}

	// try bitcoind first
	if bitcoind != nil {
		var decodedChainHash chainhash.Hash
		if err := chainhash.Decode(&decodedChainHash, hash); err == nil {
			if block, err := bitcoind.GetBlock(&decodedChainHash); err == nil {
				raw := &bytes.Buffer{}
				if err := block.BtcEncode(raw, wire.ProtocolVersion, wire.WitnessEncoding); err == nil {
					return hex.EncodeToString(raw.Bytes()), hash, nil
				}
			}
		}
	}

	// then try explorers
	var blockFetchFunctions []func(string) ([]byte, error)
	switch network {
	case "bitcoin":
		blockFetchFunctions = append(blockFetchFunctions, blockFromBlockchainInfo)
		blockFetchFunctions = append(blockFetchFunctions, blockFromBlockchair)
		blockFetchFunctions = append(blockFetchFunctions, blockFromEsplora)
	case "testnet":
		blockFetchFunctions = append(blockFetchFunctions, blockFromEsplora)
		blockFetchFunctions = append(blockFetchFunctions, blockFromBlockchair)
	case "signet", "liquid":
		blockFetchFunctions = append(blockFetchFunctions, blockFromEsplora)
	}

	for _, try := range blockFetchFunctions {
		block, errW := try(hash)
		if errW != nil || block == nil {
			err = errW
			continue
		}

		// verify and hash, but only on mainnet, the others we trust even more blindly
		if network == "bitcoin" {
			blockparsed, errW := btcutil.NewBlockFromBytes(block)
			if errW != nil {
				err = errW
				continue
			}
			header := blockparsed.MsgBlock().Header

			blockhash := hex.EncodeToString(reverseHash(blockparsed.Hash()))
			if blockhash != hash {
				err = fmt.Errorf("fetched block hash %s doesn't match expected %s",
					blockhash, hash)
				continue
			}

			prevHash := hex.EncodeToString(reverseHash(&header.PrevBlock))
			if cachedPrevHash, ok := heightCache[height-1]; ok {
				if prevHash != cachedPrevHash {
					// something is badly wrong with this block
					err = fmt.Errorf("block %d (%s): prev block hash %d (%s) doesn't match what we know from previous block %d (%s)", height, blockhash, height-1, prevHash, height-1, cachedPrevHash)
					continue
				}
			}
		}

		blockhex := hex.EncodeToString(block)
		return blockhex, hash, nil
	}

	return
}

func getHash(height int64) (hash string, err error) {
	// try bitcoind first
	if bitcoind != nil {
		if hash, err := bitcoind.GetBlockHash(height); err == nil {
			return hash.String(), nil
		}
	}

	// then try explorers
	for _, endpoint := range esploras(network) {
		w, errW := http.Get(fmt.Sprintf(endpoint+"/block-height/%d", height))
		if errW != nil {
			err = errW
			continue
		}
		defer w.Body.Close()

		if w.StatusCode >= 404 {
			continue
		}

		data, errW := io.ReadAll(w.Body)
		if errW != nil {
			err = errW
			continue
		}

		hash = strings.TrimSpace(string(data))

		if len(hash) > 64 {
			err = errors.New("got something that isn't a block hash: " + hash[:64])
			continue
		}

		return hash, nil
	}

	return "", err
}

func reverseHash(hash *chainhash.Hash) []byte {
	r := make([]byte, chainhash.HashSize)
	for i, b := range hash {
		r[chainhash.HashSize-i-1] = b
	}
	return r
}

func blockFromBlockchainInfo(hash string) ([]byte, error) {
	w, err := http.Get(fmt.Sprintf("https://blockchain.info/rawblock/%s?format=hex", hash))
	if err != nil {
		return nil, fmt.Errorf("failed to get raw block %s from blockchain.info: %s", hash, err.Error())
	}
	defer w.Body.Close()

	block, _ := io.ReadAll(w.Body)
	if len(block) < 100 {
		// block not available here yet
		return nil, nil
	}

	blockbytes, err := hex.DecodeString(string(block))
	if err != nil {
		return nil, fmt.Errorf("block from blockchain.info is invalid hex: %w", err)
	}

	return blockbytes, nil
}

func blockFromBlockchair(hash string) ([]byte, error) {
	var url string
	switch network {
	case "bitcoin":
		url = "https://api.blockchair.com/bitcoin/raw/block/"
	case "testnet":
		url = "https://api.blockchair.com/bitcoin/testnet/raw/block/"
	default:
		return nil, nil
	}
	w, err := http.Get(url + hash)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get raw block %s from blockchair.com: %s", hash, err.Error())
	}
	defer w.Body.Close()

	var data struct {
		Data map[string]struct {
			RawBlock string `json:"raw_block"`
		} `json:"data"`
	}
	err = json.NewDecoder(w.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	if bdata, ok := data.Data[hash]; ok {
		blockbytes, err := hex.DecodeString(bdata.RawBlock)
		if err != nil {
			return nil, fmt.Errorf("block from blockchair is invalid hex: %w", err)
		}

		return blockbytes, nil
	} else {
		// block not available here yet
		return nil, nil
	}
}

func blockFromEsplora(hash string) ([]byte, error) {
	var err error
	var block []byte

	for _, endpoint := range esploras(network) {
		w, errW := http.Get(fmt.Sprintf(endpoint+"/block/%s/raw", hash))
		if errW != nil {
			err = errW
			continue
		}

		defer w.Body.Close()
		block, _ = io.ReadAll(w.Body)

		if len(block) < 200 {
			// block not available yet
			return nil, nil
		}

		break
	}

	return block, err
}
