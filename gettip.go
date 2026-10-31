package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

func getTip() (int64, error) {
	var errs []error

	// try bitcoind first
	if bitcoind != nil {
		info, err := bitcoind.GetBlockChainInfo()
		if err == nil {
			return int64(info.Headers), nil
		}
		errs = append(errs, fmt.Errorf("bitcoind: %w", err))
	}

	// then try explorers
	for _, endpoint := range esploras(network) {
		tip, err := getExplorerTip(endpoint)
		if err == nil {
			return tip, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", endpoint, err))
	}

	if len(errs) == 0 {
		return 0, fmt.Errorf("no block source: bitcoind is not configured and there are no block explorers for network %q", network)
	}
	return 0, errors.Join(errs...)
}

func getExplorerTip(endpoint string) (int64, error) {
	w, err := http.Get(endpoint + "/blocks/tip/height")
	if err != nil {
		return 0, err
	}
	defer w.Body.Close()

	if w.StatusCode >= 400 {
		return 0, fmt.Errorf("status %d", w.StatusCode)
	}

	data, err := io.ReadAll(w.Body)
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(string(data), 10, 64)
}
