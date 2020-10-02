package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

type EstimatedFees struct {
	Opening         *int `json:"opening"`
	MutualClose     *int `json:"mutual_close"`
	UnilateralClose *int `json:"unilateral_close"`
	DelayedToUs     *int `json:"delayed_to_us"`
	HTLCResolution  *int `json:"htlc_resolution"`
	Penalty         *int `json:"penalty"`
	MinAcceptable   *int `json:"min_acceptable"`
	MaxAcceptable   *int `json:"max_acceptable"`
}

func getFeeRatesFromEsplora() (feerates map[string]float64, err error) {
	for _, endpoint := range esploras(network) {
		w, errW := http.Get(endpoint + "/fee-estimates")
		if errW != nil {
			err = errW
			continue
		}
		defer w.Body.Close()

		if w.StatusCode >= 300 {
			err = errors.New(endpoint + " error: " + w.Status)
			return
		}

		err = json.NewDecoder(w.Body).Decode(&feerates)
		return
	}

	err = errors.New("none of the esploras returned usable responses")
	return
}
