module github.com/fiatjaf/trustedcoin

go 1.14

require (
	github.com/btcsuite/btcd v0.20.1-beta.0.20200515232429-9f0179fd2c46
	github.com/btcsuite/btcutil v1.0.2
	github.com/fiatjaf/lightningd-gjson-rpc v1.1.0
	github.com/mitchellh/gox v1.0.1 // indirect
)

replace github.com/fiatjaf/lightningd-gjson-rpc => /home/fiatjaf/comp/lightningd-gjson-rpc
