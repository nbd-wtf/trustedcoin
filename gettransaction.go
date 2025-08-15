package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type UTXOResponse struct {
	Amount *int64  `json:"amount"`
	Script *string `json:"script"`
}

type TxResponse struct {
	TXID string   `json:"txid"`
	Vout []TxVout `json:"vout"`
}

type TxVout struct {
	ScriptPubKey string `json:"scriptPubKey"`
	Value        int64  `json:"value"`
}

func getTransaction(txid string) (tx TxResponse, err error) {
	// try bitcoind first
	if bitcoind != nil {
		var decodedChainHash chainhash.Hash
		if err := chainhash.Decode(&decodedChainHash, txid); err == nil {
			if tx, err := bitcoind.GetRawTransaction(&decodedChainHash); err == nil {
				outputs := tx.MsgTx().TxOut
				vout := make([]TxVout, len(outputs))
				for i, out := range outputs {
					vout[i] = TxVout{
						ScriptPubKey: hex.EncodeToString(out.PkScript),
						Value:        out.Value,
					}
				}

				return TxResponse{
					TXID: txid,
					Vout: vout,
				}, nil
			}
		}
	}

	// then try explorers
	for _, endpoint := range esploras(network) {
		w, errW := http.Get(endpoint + "/tx/" + txid)
		if errW != nil {
			err = errW
			continue
		}
		defer w.Body.Close()

		if w.StatusCode >= 400 {
			data := make([]byte, 99)
			n, _ := w.Body.Read(data)
			message := string(data[0:n])
			if n >= 99 {
				message += "…"
			}
			err = fmt.Errorf("unexpected response: '%s'", message)
			continue
		}

		errW = json.NewDecoder(w.Body).Decode(&tx)
		if errW != nil {
			err = errW
			continue
		}

		return tx, nil
	}

	return TxResponse{}, fmt.Errorf("couldn't find the transaction anywhere (last error: %w)", err)
}
