package main

import (
	"bytes"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/wire"
)

type RawTransactionResponse struct {
	Success bool   `json:"success"`
	ErrMsg  string `json:"errmsg"`
}

func sendRawTransaction(txHex string) RawTransactionResponse {
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
	var errs []string
	for _, endpoint := range esploras(network) {
		tx := bytes.NewBufferString(txHex)

		w, err := http.Post(endpoint+"/tx", "text/plain", tx)
		if err != nil {
			errs = append(errs, endpoint+": "+err.Error())
			continue
		}

		if w.StatusCode >= 300 {
			msg, _ := io.ReadAll(w.Body)
			w.Body.Close()
			errs = append(errs, endpoint+": "+string(msg))
			continue
		}
		w.Body.Close()

		return RawTransactionResponse{true, ""}
	}

	return RawTransactionResponse{false, strings.Join(errs, "; ")}
}
