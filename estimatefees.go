package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/btcsuite/btcd/btcjson"
)

type EstimatedFees struct {
	FeeRateFloor int       `json:"feerate_floor"`
	FeeRates     []FeeRate `json:"feerates"`
}

type FeeRate struct {
	Blocks  int `json:"blocks"`
	FeeRate int `json:"feerate"`
}

func getFeeRates() (*EstimatedFees, error) {
	// try bitcoind first
	if bitcoind != nil {
		in2, err2 := bitcoind.EstimateSmartFee(2, &btcjson.EstimateModeConservative)
		in6, err6 := bitcoind.EstimateSmartFee(6, &btcjson.EstimateModeEconomical)
		in12, err12 := bitcoind.EstimateSmartFee(12, &btcjson.EstimateModeEconomical)
		in100, err100 := bitcoind.EstimateSmartFee(100, &btcjson.EstimateModeEconomical)
		if err2 == nil && err6 == nil && err12 == nil && err100 == nil &&
			in2.FeeRate != nil && in6.FeeRate != nil && in12.FeeRate != nil && in100.FeeRate != nil {

			satPerKbP := func(r *btcjson.EstimateSmartFeeResult) int {
				return int(*r.FeeRate * float64(100000000))
			}

			return &EstimatedFees{
				FeeRateFloor: satPerKbP(in100),
				FeeRates: []FeeRate{
					{Blocks: 2, FeeRate: satPerKbP(in2)},
					{Blocks: 6, FeeRate: satPerKbP(in6)},
					{Blocks: 12, FeeRate: satPerKbP(in12)},
					{Blocks: 100, FeeRate: satPerKbP(in100)},
				},
			}, nil
		}
	}

	// then try explorers
	// (just copy sauron here)
	feerates, err := getFeeRatesFromEsplora()
	if err != nil {
		return nil, err
	}

	// actually let's be a little more patient here than sauron is
	slow := int(feerates["504"] * 1000)
	normal := int(feerates["10"] * 1000)
	urgent := int(feerates["5"] * 1000)
	veryUrgent := int(feerates["2"] * 1000)

	return &EstimatedFees{
		FeeRateFloor: slow,
		FeeRates: []FeeRate{
			{Blocks: 2, FeeRate: veryUrgent},
			{Blocks: 5, FeeRate: urgent},
			{Blocks: 10, FeeRate: normal},
			{Blocks: 504, FeeRate: slow},
		},
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
