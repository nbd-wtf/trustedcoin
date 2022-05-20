package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/btcsuite/btcd/btcjson"
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

var intp = func(x int) *int { return &x }

func getFeeRates() (*EstimatedFees, error) {
	// try bitcoind first
	if bitcoind != nil {
		in2, err2 := bitcoind.EstimateSmartFee(2, &btcjson.EstimateModeConservative)
		in6, err6 := bitcoind.EstimateSmartFee(6, &btcjson.EstimateModeEconomical)
		in12, err12 := bitcoind.EstimateSmartFee(12, &btcjson.EstimateModeEconomical)
		in100, err100 := bitcoind.EstimateSmartFee(100, &btcjson.EstimateModeEconomical)
		if err2 == nil && err6 == nil && err12 == nil && err100 == nil &&
			in2.FeeRate != nil && in6.FeeRate != nil && in12.FeeRate != nil && in100.FeeRate != nil {

			satPerKbP := func(r *btcjson.EstimateSmartFeeResult) *int {
				x := int(*r.FeeRate * float64(100000000))
				return &x
			}

			estimated := &EstimatedFees{
				Opening:         satPerKbP(in12),
				MutualClose:     satPerKbP(in100),
				UnilateralClose: satPerKbP(in2),
				DelayedToUs:     satPerKbP(in12),
				HTLCResolution:  satPerKbP(in12),
				Penalty:         satPerKbP(in12),
			}
			estimated.MinAcceptable = intp(*estimated.MutualClose / 2)
			estimated.MaxAcceptable = intp(*estimated.UnilateralClose * 100)
		}
	}

	// then try explorers
	// (just copy sauron here)
	feerates, err := getFeeRatesFromEsplora()
	if err != nil {
		return nil, err
	}

	var (
		slow        = int(feerates["504"] * 1000)
		normal      = int(feerates["10"] * 1000)
		urgent      = int(feerates["5"] * 1000)
		very_urgent = int(feerates["2"] * 1000)
	)

	// actually let's be a little more patient here than sauron is
	return &EstimatedFees{
		Opening:         intp(slow),
		MutualClose:     intp(normal),
		UnilateralClose: intp(very_urgent),
		DelayedToUs:     intp(slow),
		HTLCResolution:  intp(normal),
		Penalty:         intp(urgent),
		MinAcceptable:   intp(slow / 2),
		MaxAcceptable:   intp(very_urgent * 100),
	}, nil
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
