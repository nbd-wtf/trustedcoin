package main

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"net/http"

	"github.com/btcsuite/btcd/wire"
)

type RawTransactionResponse struct {
	Success bool   `json:"success"`
	ErrMsg  string `json:"errmsg"`
}

func sendRawTransaction(txHex string) (resp RawTransactionResponse) {
	// try bitcoind first
	if bitcoind != nil {
		tx := &wire.MsgTx{}
		if txBytes, err := hex.DecodeString(txHex); err == nil {
			txBuf := bytes.NewBuffer(txBytes)
			if err := tx.BtcDecode(txBuf, wire.ProtocolVersion, wire.WitnessEncoding); err == nil {
				if _, err := bitcoind.SendRawTransaction(tx, true); err == nil {
					return RawTransactionResponse{true, ""}
				}
			}
		}
	}

	// then try explorers
	tx := bytes.NewBufferString(txHex)
	for _, endpoint := range esploras(network) {
		w, err := http.Post(endpoint+"/tx", "text/plain", tx)
		if err != nil {
			resp = RawTransactionResponse{false, err.Error()}
			continue
		}
		defer w.Body.Close()

		if w.StatusCode >= 300 {
			msg, _ := ioutil.ReadAll(w.Body)
			resp = RawTransactionResponse{false, string(msg)}
			err = nil
			continue
		}

		return RawTransactionResponse{true, ""}
	}

	return resp
}
