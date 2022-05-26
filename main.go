package main

import (
	"fmt"
	"math/rand"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

var (
	network string
	esplora = map[string][]string{
		"bitcoin": {
			"https://mempool.space/api",
			"https://blockstream.info/api",
			"https://mempool.emzy.de/api",
		},
		"testnet": {
			"https://mempool.space/testnet/api",
			"https://blockstream.info/testnet/api",
		},
		"signet": {
			"https://mempool.space/signet/api",
		},
		"liquid": {
			"https://liquid.network/api",
			"https://blockstream.info/liquid/api",
		},
	}
	defaultBitcoindRPCPorts = map[string]string{
		"bitcoin": "8332",
		"testnet": "18332",
		"signet":  "38332",
		"regtest": "18443",
	}
	bitcoind *rpcclient.Client
)

func esploras(network string) (ss []string) {
	n := len(esplora[network])
	index := rand.Intn(10)
	ss = make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = esplora[network][index%n]
	}
	return ss
}

func main() {
	p := plugin.Plugin{
		Name:    "trustedcoin",
		Version: "v0.5.0",
		Options: []plugin.Option{
			{Name: "bitcoin-rpcconnect", Type: "string", Description: "Hostname (IP) to bitcoind RPC (optional).", Default: ""},
			{Name: "bitcoin-rpcport", Type: "string", Description: "Port to bitcoind RPC (optional).", Default: ""},
			{Name: "bitcoin-rpcuser", Type: "string", Description: "Username to bitcoind RPC (optional).", Default: ""},
			{Name: "bitcoin-rpcpassword", Type: "string", Description: "Password to bitcoind RPC (optional).", Default: ""},
		},
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

					var bip70network string
					switch network {
					case "bitcoin":
						bip70network = "main"
					case "testnet":
						bip70network = "test"
					case "liquid":
						bip70network = "liquidv1"
					}

					return struct {
						Chain       string `json:"chain"`
						HeaderCount int64  `json:"headercount"`
						BlockCount  int64  `json:"blockcount"`
						IBD         bool   `json:"ibd"`
					}{bip70network, tip, tip, false}, 0, nil
				},
			}, {
				"estimatefees",
				"",
				"Get the Bitcoin feerate in sat/kilo-vbyte.",
				"",
				func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
					estfees, err := getFeeRates()
					if err != nil {
						p.Logf("estimatefees error: %s", err.Error())
						estfees = &EstimatedFees{}
					}

					return *estfees, 0, nil
				},
			}, {
				"sendrawtransaction",
				"tx",
				"Send a raw transaction to the Bitcoin network.",
				"",
				func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
					hex := params.Get("tx").String()

					srtresp, err := sendRawTransaction(hex)
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
		OnInit: func(p *plugin.Plugin) {
			network = p.Network

			// we will try to use a local bitcoind
			user := p.Args.Get("bitcoin-rpcuser").String()
			pass := p.Args.Get("bitcoin-rpcpassword").String()
			if user != "" && pass != "" {
				hostname := p.Args.Get("bitcoin-rpcconnect").String()
				if hostname == "" {
					hostname = "127.0.0.1"
				}
				port := p.Args.Get("bitcoin-rpcport").String()
				if port == "" {
					port = defaultBitcoindRPCPorts[network]
					if port == "" {
						port = "8332"
					}
				}

				client, err := rpcclient.New(&rpcclient.ConnConfig{
					Host:         hostname + ":" + port,
					User:         user,
					Pass:         pass,
					HTTPPostMode: true,
					DisableTLS:   true,
				}, nil)
				if err != nil {
					p.Log("bitcoind RPC backend settings detected but invalid, will only use block explorers.")
					return
				}

				bitcoind = client
				if _, err := bitcoind.GetBlockChainInfo(); err == nil {
					p.Log("bitcoind RPC working, will use that with highest priority and fall back to block explorers if it fails.")
				} else {
					p.Log("bitcoind RPC backend settings detected, but failed to connect, will keep trying to use it though.")
				}
				return
			}

			p.Log("bitcoind RPC settings not detected (looked for 'bitcoin-rpcuser', 'bitcoin-rpcpassword' and optionally 'bitcoin-rpcconnect' and 'bitcoin-rpcport'), will only use block explorers.")
		},
	}

	p.Run()
}
