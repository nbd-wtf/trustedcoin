package main_test

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"testing"
)

const executable = "./trustedcoin"

const manifestRequest = `{"jsonrpc":"2.0","id":"manifest","method":"init","params":{"options":{},"configuration":{"network":"bitcoin","lightning-dir":"/tmp","rpc-file":"foo"}}}`
const manifestExpectedResponse = `{"jsonrpc":"2.0","id":"manifest"}`

const initRequest = `{"jsonrpc":"2.0","id":"init","method":"getmanifest","params":{}}`
const initExpectedResponse = `{"jsonrpc":"2.0","id":"init","result":{"options":[{"name":"bitcoin-rpcconnect","type":"string","default":"","description":"Hostname (IP) to bitcoind RPC (optional)."},{"name":"bitcoin-rpcport","type":"string","default":"","description":"Port to bitcoind RPC (optional)."},{"name":"bitcoin-rpcuser","type":"string","default":"","description":"Username to bitcoind RPC (optional)."},{"name":"bitcoin-rpcpassword","type":"string","default":"","description":"Password to bitcoind RPC (optional)."}],"rpcmethods":[{"name":"getrawblockbyheight","usage":"height","description":"Get the bitcoin block at a given height","long_description":""},{"name":"getchaininfo","usage":"","description":"Get the chain id, the header count, the block count and whether this is IBD.","long_description":""},{"name":"estimatefees","usage":"","description":"Get the Bitcoin feerate in sat/kilo-vbyte.","long_description":""},{"name":"sendrawtransaction","usage":"tx","description":"Send a raw transaction to the Bitcoin network.","long_description":""},{"name":"getutxout","usage":"txid vout","description":"Get informations about an output, identified by a {txid} an a {vout}","long_description":""}],"subscriptions":[],"hooks":[],"featurebits":{"features":"","channel":"","init":"","invoice":""},"dynamic":false,"notifications":[]}}`

const shutdownNotification = `{"jsonrpc":"2.0","method":"shutdown","params":{}}`

func TestInitAndShutdown(t *testing.T) {
	cmd, stdin, stdout, stderr := start(t)
	stop(t, cmd, stdin, stdout, stderr)
}

func start(t *testing.T) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser) {
	cmd := exec.Command(executable)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		t.Fatalf("expected trustedcoin to start, got %v", err)
	}

	_, _ = io.WriteString(stdin, manifestRequest)
	if response := readline(stdout); response != manifestExpectedResponse {
		t.Fatalf("unexpected RPC response: %s", response)
	}

	_, _ = io.WriteString(stdin, initRequest)
	if response := readline(stdout); response != initExpectedResponse {
		t.Fatalf("unexpected RPC response: %s", response)
	}

	if response := readline(stderr); !strings.Contains(response, "initialized plugin") {
		t.Fatalf("unexpected output in stderr: %s", response)
	}

	return cmd, stdin, stdout, stderr
}

func stop(t *testing.T, cmd *exec.Cmd, stdin io.WriteCloser, stdout, stderr io.ReadCloser) {
	_, _ = io.WriteString(stdin, shutdownNotification)
	_ = stdin.Close()
	_ = stdout.Close()
	_ = stderr.Close()

	if err := cmd.Wait(); err != nil {
		t.Fatalf("expected process to exit cleanly, got %v", err)
	}
}

func readline(r io.Reader) string {
	line, _ := bufio.NewReader(r).ReadString('\n')

	return strings.TrimSuffix(line, "\n")
}
