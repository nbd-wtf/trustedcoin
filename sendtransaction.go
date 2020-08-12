package main

import (
	"io"
	"io/ioutil"
	"net/http"
)

type RawTransactionResponse struct {
	Success bool   `json:"success"`
	ErrMsg  string `json:"errmsg"`
}

func sendRawTransaction(tx io.Reader) (resp RawTransactionResponse, err error) {
	for _, endpoint := range esploras() {
		w, errW := http.Post(endpoint+"/tx", "text/plain", tx)
		if errW != nil {
			err = errW
			continue
		}
		defer w.Body.Close()

		if w.StatusCode >= 300 {
			msg, _ := ioutil.ReadAll(w.Body)
			return RawTransactionResponse{false, string(msg)}, nil
		}

		return RawTransactionResponse{true, ""}, nil
	}

	return resp, err
}
