package main

import (
	"io/ioutil"
	"net/http"
	"strconv"
)

func getTip() (tip int64, err error) {
	// try bitcoind first
	if bitcoind != nil {
		if info, err := bitcoind.GetBlockChainInfo(); err == nil {
			return int64(info.Headers), nil
		}
	}

	// then try explorers
	for _, endpoint := range esploras(network) {
		w, errW := http.Get(endpoint + "/blocks/tip/height")
		if errW != nil {
			err = errW
			continue
		}
		defer w.Body.Close()

		data, errW := ioutil.ReadAll(w.Body)
		if errW != nil {
			err = errW
			continue
		}

		tip, errW = strconv.ParseInt(string(data), 10, 64)
		if errW != nil {
			err = errW
			continue
		}

		return tip, nil
	}

	return 0, err
}
