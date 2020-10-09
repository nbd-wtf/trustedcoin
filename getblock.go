package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
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

	api_range := []func(string) (string, error){
		blockFromBlockchainInfo,
		blockFromBlockchair,
	}

	if network == "test" {
		api_range = []func(string) (string, error){
			blockFromBlockchair,
		}
	}

	for _, try := range api_range {
		blockhex, errW := try(hash)
		if errW != nil || blockhex == "" {
			err = errW
			continue
		}

		// verify and hash
		blockbytes, errW := hex.DecodeString(blockhex)
		if errW != nil {
			err = errW
			continue
		}

		blockparsed, errW := btcutil.NewBlockFromBytes(blockbytes)
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

		prevHash := hex.EncodeToString(header.PrevBlock[:])
		if cachedPrevHash, ok := heightCache[height-1]; ok {
			if prevHash != cachedPrevHash {
				// something is badly wrong with this block
				err = fmt.Errorf("block %d (%s): prev block hash %d (%s) doesn't match what we know from previous block %d (%s)", height, blockhash, height-1, prevHash, height-1, cachedPrevHash)
				continue
			}
		}

		delete(heightCache, height)
		return blockhex, hash, nil
	}

	return
}

func getHash(height int64) (hash string, err error) {
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

		data, errW := ioutil.ReadAll(w.Body)
		if errW != nil {
			err = errW
			continue
		}

		hash = strings.TrimSpace(string(data))

		if len(hash) > 64 {
			err = errors.New("got something that isn't a block hash: " + hash[:64])
			continue
		}

		heightCache[height] = hash

		return hash, nil
	}

	return "", err
}

func blockFromBlockchainInfo(hash string) (string, error) {
	w, err := http.Get(fmt.Sprintf("https://blockchain.info/block/%s?format=hex", hash))
	if err != nil {
		return "", fmt.Errorf("failed to get raw block %s from blockchain.info: %s", hash, err.Error())
	}
	defer w.Body.Close()

	block, _ := ioutil.ReadAll(w.Body)
	if len(block) < 100 {
		// block not available here yet
		return "", nil
	}

	return string(block), nil
}

func blockFromBlockchair(hash string) (string, error) {
	var url string
	if network == "main" {
		url = "https://api.blockchair.com/bitcoin/raw/block/"
	} else {
		url = "https://api.blockchair.com/bitcoin/testnet/raw/block/"
	}
	w, err := http.Get(url + hash)
	if err != nil {
		return "", fmt.Errorf("failed to get raw block %s from blockchair.com: %s", hash, err.Error())
	}
	defer w.Body.Close()

	var data struct {
		Data map[string]struct {
			RawBlock string `json:"raw_block"`
		} `json:"data"`
	}
	err = json.NewDecoder(w.Body).Decode(&data)
	if err != nil {
		return "", err
	}

	if bdata, ok := data.Data[hash]; ok {
		return bdata.RawBlock, nil
	} else {
		// block not available here yet
		return "", nil
	}
}

func reverseHash(hash *chainhash.Hash) []byte {
	r := make([]byte, chainhash.HashSize)
	for i, b := range hash {
		r[chainhash.HashSize-i-1] = b
	}
	return r
}
