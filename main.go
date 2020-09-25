package main

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

var (
	esplora = []string{
		"https://mempool.space/api",
		"https://blockstream.info/api",
		"https://explorer.bullbitcoin.com/api",
		"https://mempool.emzy.de/api",
	}
)

func esploras() (ss []string) {
	n := len(esplora)
	index := rand.Intn(10)
	ss = make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = esplora[index%n]
	}
	return ss
}

func main() {
	p := plugin.Plugin{
		Name:    "trustedcoin",
		Version: "v0.2.5",
		RPCMethods: []plugin.RPCMethod{
			{
				"getrawblockbyheight",
				"height",
				"Get the bitcoin block at a given height",
				"",
				func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
					height := params.Get("height").Int()

					blockUnavailable := map[string]interface{}{
						"blockhash": nil,
						"block":     nil,
					}

					block, hash, err := getBlock(height)
					if err != nil {
						p.Logf("getblock error: %s", err.Error())
						return blockUnavailable, 0, nil
					}
					if block == "" {
						return blockUnavailable, 0, nil
					}

					p.Logf("returning block %d, %s…, %d bytes",
						height, string(hash[:26]), len(block)/2)

					return struct {
						BlockHash string `json:"blockhash"`
						Block     string `json:"block"`
					}{hash, string(block)}, 0, nil
				},
			}, {
				"getchaininfo",
				"",
				"Get the chain id, the header count, the block count and whether this is IBD.",
				"",
				func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
					tip, err := getTip()
					if err != nil {
						return nil, 20, fmt.Errorf("failed to get tip: %s", err.Error())
					}

					p.Logf("tip: %d", tip)

					return struct {
						Chain       string `json:"chain"`
						HeaderCount int64  `json:"headercount"`
						BlockCount  int64  `json:"blockcount"`
						IBD         bool   `json:"ibd"`
					}{"main", tip, tip, false}, 0, nil
				},
			}, {
				"estimatefees",
				"",
				"Get the Bitcoin feerate in btc/kilo-vbyte.",
				"",
				func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
					// just copy sauron here
					feerates, err := getFeeRatesFromEsplora()
					if err != nil {
						p.Logf("estimatefees error: %s", err.Error())
						return EstimatedFees{nil, nil, nil, nil, nil, nil, nil, nil},
							0, nil
					}

					var (
						slow        = int(feerates["504"] * 1000)
						normal      = int(feerates["10"] * 1000)
						urgent      = int(feerates["5"] * 1000)
						very_urgent = int(feerates["2"] * 1000)

						intp = func(x int) *int { return &x }
					)

					// actually let's be a little more patient here than sauron is
					return EstimatedFees{
						Opening:         intp(slow),
						MutualClose:     intp(normal),
						UnilateralClose: intp(very_urgent),
						DelayedToUs:     intp(slow),
						HTLCResolution:  intp(normal),
						Penalty:         intp(urgent),
						MinAcceptable:   intp(slow / 2),
						MaxAcceptable:   intp(very_urgent * 100),
					}, 0, nil
				},
			}, {
				"sendrawtransaction",
				"tx",
				"Send a raw transaction to the Bitcoin network.",
				"",
				func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
					hex := params.Get("tx").String()
					tx := bytes.NewBufferString(hex)

					srtresp, err := sendRawTransaction(tx)
					if err != nil {
						p.Logf("failed to publish transaction %s: %s", hex, err.Error())
						return nil, 21, err
					}

					return srtresp, 0, nil
				},
			}, {
				"getutxout",
				"txid vout",
				"Get informations about an output, identified by a {txid} an a {vout}",
				"",
				func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
					txid := params.Get("txid").String()
					vout := params.Get("vout").Int()

					tx, err := getTransaction(txid)
					if err != nil {
						p.Logf("failed to get tx %s: %s", txid, err.Error())
						return UTXOResponse{nil, nil}, 0, nil
					}

					output := tx.Vout[vout]

					return UTXOResponse{&output.Value, &output.ScriptPubKey}, 0, nil
				},
			},
		},
	}

	p.Run()
}
