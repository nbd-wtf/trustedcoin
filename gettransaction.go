package main

import (
	"encoding/json"
	"net/http"
)

type UTXOResponse struct {
	Amount *int64  `json:"amount"`
	Script *string `json:"script"`
}

func getTransaction(txid string) (tx struct {
	TXID string `json:"txid"`
	Vout []struct {
		ScriptPubKey string `json:"scriptPubKey"`
		Value        int64  `json:"value"`
	} `json:"vout"`
}, err error) {
	for _, endpoint := range esploras() {
		w, errW := http.Get(endpoint + "/tx/" + txid)
		if errW != nil {
			err = errW
			continue
		}
		defer w.Body.Close()

		errW = json.NewDecoder(w.Body).Decode(&tx)
		if errW != nil {
			err = errW
			continue
		}

		return tx, nil
	}

	return
}
