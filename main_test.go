package main_test

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const executable = "./trustedcoin"

const getManifestRequest = `{"jsonrpc":"2.0","id":"getmanifest","method":"getmanifest","params":{}}`
const getManifestExpectedResponse = `{"jsonrpc":"2.0","id":"getmanifest","result":{"options":[{"name":"bitcoin-rpcconnect","type":"string","default":"","description":"Hostname (IP) to bitcoind RPC (optional)."},{"name":"bitcoin-rpcport","type":"string","default":"","description":"Port to bitcoind RPC (optional)."},{"name":"bitcoin-rpcuser","type":"string","default":"","description":"Username to bitcoind RPC (optional)."},{"name":"bitcoin-rpcpassword","type":"string","default":"","description":"Password to bitcoind RPC (optional)."},{"name":"bitcoin-datadir","type":"string","default":"","description":"-datadir arg for bitcoin-cli. For compatibility with bcli, not actually used."}],"rpcmethods":[{"name":"getrawblockbyheight","usage":"height","description":"Get the bitcoin block at a given height","long_description":""},{"name":"getchaininfo","usage":"","description":"Get the chain id, the header count, the block count and whether this is IBD.","long_description":""},{"name":"estimatefees","usage":"","description":"Get the Bitcoin feerate in sat/kilo-vbyte.","long_description":""},{"name":"sendrawtransaction","usage":"tx","description":"Send a raw transaction to the Bitcoin network.","long_description":""},{"name":"getutxout","usage":"txid vout","description":"Get informations about an output, identified by a {txid} an a {vout}","long_description":""}],"subscriptions":[],"hooks":[],"featurebits":{"features":"","channel":"","init":"","invoice":""},"dynamic":false,"notifications":[]}}`

const initRequest = `{"jsonrpc":"2.0","id":"init","method":"init","params":{"options":{},"configuration":{"network":"bitcoin","lightning-dir":"/tmp","rpc-file":"foo"}}}`
const regtestInitRequest = `{"jsonrpc":"2.0","id":"init","method":"init","params":{"options":{},"configuration":{"network":"regtest","lightning-dir":"/tmp","rpc-file":"foo"}}}`
const initExpectedResponse = `{"jsonrpc":"2.0","id":"init"}`

const getchaininfoRequest = `{"jsonrpc":"2.0","id":"ci","method":"getchaininfo","params":{}}`

const shutdownNotification = `{"jsonrpc":"2.0","method":"shutdown","params":{}}`

// regtest has no block explorers, so without bitcoind there is no source.
const noSourceMaxDuration = 5 * time.Second

func TestInitAndShutdown(t *testing.T) {
	cmd, stdin, stdout, stderr := start(t, initRequest)
	stop(t, cmd, stdin, stdout, stderr)
}

func TestGetchaininfoWithoutSourcesFails(t *testing.T) {
	cmd, stdin, stdout, stderr := start(t, regtestInitRequest)

	_, _ = io.WriteString(stdin, getchaininfoRequest)

	// Read with a deadline, so a stuck handler fails the test instead of
	// hanging it. Without bitcoind there must be no retry either.
	responses := make(chan string, 1)
	go func() { responses <- readline(stdout) }()
	var response string
	select {
	case response = <-responses:
	case <-time.After(noSourceMaxDuration):
		_ = cmd.Process.Kill()
		t.Fatalf("getchaininfo did not answer within %s", noSourceMaxDuration)
	}

	if !strings.Contains(response, `"error"`) {
		t.Fatalf("expected an error response, got %s", response)
	}
	if strings.Contains(response, `"headercount":0`) {
		t.Fatalf("expected no headercount 0, got %s", response)
	}

	stop(t, cmd, stdin, stdout, stderr)
}

func start(t *testing.T, init string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser) {
	cmd := exec.Command(executable)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		t.Fatalf("expected trustedcoin to start, got %v", err)
	}

	_, _ = io.WriteString(stdin, getManifestRequest)
	if response := readline(stdout); response != getManifestExpectedResponse {
		t.Fatalf("unexpected manifest response: %s", response)
	}

	_, _ = io.WriteString(stdin, init)
	if response := readline(stdout); response != initExpectedResponse {
		t.Fatalf("unexpected init response: %s", response)
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
