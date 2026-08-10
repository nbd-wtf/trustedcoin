// Regression tests for the trustedcoin broadcast bug: sendRawTransaction shared
// one bytes.Buffer across the whole explorer loop, so every endpoint after the
// first got an empty body and returned "-22 TX decode failed" for a valid
// transaction.
//
// Run with `go test -run TestBroadcast -v` (not `-run BroadcastFailover`, which
// skips the second test). These set package-level vars from main.go, so they
// must live in package main.
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A confirmed mainnet transaction (txid 62f42127...0fce51c, block 961810) used
// only as an opaque request body -- the loop never decodes it, so any 444-char
// hex payload would work; the byte-count assertions below depend on that length.
const testTxHex = "02000000000101899bb5a4848105a9a8ac5325bf392a83553e57f00d498a1b15b1cc5e4296f1240000000000fdffffff025e41000000000000160014aa75e46ae192c16bd49ab140f3418eca6fa5a750dc05000000000000160014df48604c3e3cffc56e44951988a3778162cb810102473044022048011be500c651d0730a739ffc42eb11452c5c4a42c67da55cf4e2da964f85e50220497b3fc66405a57c4ca2a63dcda3600681c05afd54aa2f9f9e70bed2c171fcc1012103a86b1c0f1a563299eee7dee9c59a3cbda9a8957586efcb4fb32c9d373ff7b3b100000000"

// useEndpoints points trustedcoin at the given fake explorers and guarantees no
// bitcoind shortcut is taken.
func useEndpoints(t *testing.T, urls ...string) {
	t.Helper()
	bitcoind = nil
	network = "regtest"
	esplora["regtest"] = urls
	t.Cleanup(func() { delete(esplora, "regtest") })
}

// TestBroadcastFailoverSendsFullBody: every endpoint the loop reaches must
// receive the whole transaction, not just the first one.
func TestBroadcastFailoverSendsFullBody(t *testing.T) {
	var seen []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = append(seen, len(b))
		w.WriteHeader(400) // reject, forcing the loop to try the next endpoint
		io.WriteString(w, "rejected")
	}))
	defer srv.Close()

	useEndpoints(t, srv.URL, srv.URL, srv.URL)
	sendRawTransaction(testTxHex)

	if len(seen) != 3 {
		t.Fatalf("expected 3 attempts, got %d: %v", len(seen), seen)
	}
	for i, n := range seen {
		if n != len(testTxHex) {
			t.Errorf("attempt %d delivered %d bytes, want %d -- the request body buffer was drained by an earlier attempt",
				i+1, n, len(testTxHex))
		}
	}
}

// TestBroadcastSucceedsAfterFirstEndpointErrors: a rate limit on the first
// explorer must not poison the ones after it.
func TestBroadcastSucceedsAfterFirstEndpointErrors(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(429)
		io.WriteString(w, "Too Many Requests")
	}))
	defer bad.Close()

	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if len(b) == 0 {
			w.WriteHeader(400)
			io.WriteString(w, "TX decode failed. Make sure the tx has at least one input.")
			return
		}
		w.WriteHeader(200)
		io.WriteString(w, "accepted")
	}))
	defer good.Close()

	useEndpoints(t, bad.URL, bad.URL, good.URL)

	// esploras() shuffles the order, so a single run can pass by luck; 30 reps make a false pass statistically negligible.
	const runs = 30
	for i := 1; i <= runs; i++ {
		resp := sendRawTransaction(testTxHex)
		if !resp.Success {
			t.Fatalf("run %d/%d: broadcast failed although a healthy explorer was reachable: %s",
				i, runs, resp.ErrMsg)
		}
		if strings.Contains(resp.ErrMsg, "at least one input") {
			t.Fatalf("run %d/%d: an empty body reached an explorer: %s", i, runs, resp.ErrMsg)
		}
	}
}

// TestBroadcastSurvivesAnUnreachableEndpoint: an explorer that never answers
// must not consume the attempt of the ones after it. This is the other half of
// the loop's error handling -- a transport failure rather than an HTTP status.
func TestBroadcastSurvivesAnUnreachableEndpoint(t *testing.T) {
	// a server that is closed right away leaves an address that refuses connections.
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	dead := closed.URL
	closed.Close()

	var seen []int
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = append(seen, len(b))
		w.WriteHeader(200)
		io.WriteString(w, "accepted")
	}))
	defer good.Close()

	useEndpoints(t, dead, good.URL)

	// two endpoints, shuffled: repeating covers both orders.
	const runs = 20
	for i := 1; i <= runs; i++ {
		if resp := sendRawTransaction(testTxHex); !resp.Success {
			t.Fatalf("run %d/%d: broadcast failed although a healthy explorer was reachable: %s",
				i, runs, resp.ErrMsg)
		}
	}
	for i, n := range seen {
		if n != len(testTxHex) {
			t.Errorf("the healthy explorer got %d bytes on hit %d, want %d", n, i+1, len(testTxHex))
		}
	}
}

// TestBroadcastReportsEveryEndpointError: every endpoint's message must reach
// the caller when all reject the transaction -- keeping only the last one is
// what made the original incident look like a malformed transaction.
func TestBroadcastReportsEveryEndpointError(t *testing.T) {
	messages := []string{"Too Many Requests", "bad-txns-inputs-missingorspent", "sendrawtransaction is disabled"}

	var urls []string
	for _, msg := range messages {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.Copy(io.Discard, r.Body)
			w.WriteHeader(400)
			io.WriteString(w, msg)
		}))
		defer srv.Close()
		urls = append(urls, srv.URL)
	}

	useEndpoints(t, urls...)
	resp := sendRawTransaction(testTxHex)

	if resp.Success {
		t.Fatalf("broadcast reported success although every explorer rejected it")
	}
	for _, msg := range messages {
		if !strings.Contains(resp.ErrMsg, msg) {
			t.Errorf("%q is missing from the reported error -- an endpoint's message was discarded: %s",
				msg, resp.ErrMsg)
		}
	}
}
