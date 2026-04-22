// Copyright 2026 github.com/gagliardetto
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rpc

import (
	"context"
	stdjson "encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

// Constructors ----------------------------------------------------------

func TestNew_DefaultsToFinalized(t *testing.T) {
	cl := New("http://localhost:0")
	assert.Equal(t, CommitmentFinalized, cl.Commitment())
}

func TestNewWithCommitment_StoresCommitment(t *testing.T) {
	cl := NewWithCommitment("http://localhost:0", CommitmentProcessed)
	assert.Equal(t, CommitmentProcessed, cl.Commitment())
}

func TestNewWithCommitment_EmptyFallsBackToFinalized(t *testing.T) {
	cl := NewWithCommitment("http://localhost:0", "")
	assert.Equal(t, CommitmentFinalized, cl.Commitment())
}

func TestNewWithTimeout_DefaultsToFinalized(t *testing.T) {
	cl := NewWithTimeout("http://localhost:0", 5*time.Second)
	assert.Equal(t, CommitmentFinalized, cl.Commitment())
}

func TestNewWithTimeoutAndCommitment_StoresCommitment(t *testing.T) {
	cl := NewWithTimeoutAndCommitment("http://localhost:0", 5*time.Second, CommitmentConfirmed)
	assert.Equal(t, CommitmentConfirmed, cl.Commitment())
}

func TestNewWithCustomRPCClient_DefaultsToFinalized(t *testing.T) {
	cl := NewWithCustomRPCClient(&nopJSONRPC{})
	assert.Equal(t, CommitmentFinalized, cl.Commitment())
}

func TestNewWithCustomRPCClientAndCommitment_StoresCommitment(t *testing.T) {
	cl := NewWithCustomRPCClientAndCommitment(&nopJSONRPC{}, CommitmentProcessed)
	assert.Equal(t, CommitmentProcessed, cl.Commitment())
}

func TestNewWithHeaders_DefaultsToFinalized(t *testing.T) {
	cl := NewWithHeaders("http://localhost:0", map[string]string{"X-Foo": "bar"})
	assert.Equal(t, CommitmentFinalized, cl.Commitment())
}

// Timeout wiring --------------------------------------------------------

// TestNewWithTimeout_AppliesTimeout verifies the HTTP timeout is actually
// wired into the underlying http.Client. We open a raw TCP listener that
// accepts the connection but never writes a response, so the only way the
// call returns is via the http.Client.Timeout firing.
func TestNewWithTimeout_AppliesTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Hold the connection open; closing the listener will end the
			// test and let these goroutines exit.
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(io.Discard, c)
			}(conn)
		}
	}()

	cl := NewWithTimeout("http://"+ln.Addr().String(), 50*time.Millisecond)
	start := time.Now()
	_, err = cl.GetSlot(context.Background(), CommitmentFinalized)
	elapsed := time.Since(start)

	require.Error(t, err, "expected a timeout error")
	assert.Less(t, elapsed, 2*time.Second, "timeout did not fire at the configured 50ms, took %s", elapsed)
}

// Commitment fallback: positional argument ------------------------------

func TestCommitmentFallback_PositionalExplicitWins(t *testing.T) {
	server, closer := mockJSONRPC(t, stdjson.RawMessage(wrapIntoRPC(`0`)))
	defer closer()

	cl := NewWithCommitment(server.URL, CommitmentFinalized)
	_, err := cl.GetSlot(context.Background(), CommitmentProcessed)
	require.NoError(t, err)

	assertCommitment(t, server, "processed")
}

func TestCommitmentFallback_PositionalEmptyUsesClientDefault(t *testing.T) {
	server, closer := mockJSONRPC(t, stdjson.RawMessage(wrapIntoRPC(`0`)))
	defer closer()

	cl := NewWithCommitment(server.URL, CommitmentConfirmed)
	_, err := cl.GetSlot(context.Background(), "")
	require.NoError(t, err)

	assertCommitment(t, server, "confirmed")
}

func TestCommitmentFallback_DefaultNewIsFinalized(t *testing.T) {
	server, closer := mockJSONRPC(t, stdjson.RawMessage(wrapIntoRPC(`0`)))
	defer closer()

	cl := New(server.URL)
	_, err := cl.GetSlot(context.Background(), "")
	require.NoError(t, err)

	assertCommitment(t, server, "finalized")
}

// Commitment fallback: opts struct --------------------------------------

func TestCommitmentFallback_OptsNilUsesClientDefault(t *testing.T) {
	responseBody := `{"blockhash":"5M77sHdwzH6rckuQwF8HL1w52n7hjrh4GVTFiF6T8QyB","previousBlockhash":"Aq9jSXe1jRzfiaBcRFLe4wm7j499vWVEeFQrq5nnXfZN","parentSlot":0,"rewards":[]}`
	server, closer := mockJSONRPC(t, stdjson.RawMessage(wrapIntoRPC(responseBody)))
	defer closer()

	cl := NewWithCommitment(server.URL, CommitmentConfirmed)
	_, err := cl.GetBlockWithOpts(context.Background(), 42, nil)
	require.NoError(t, err)

	assertCommitment(t, server, "confirmed")
}

func TestCommitmentFallback_OptsEmptyUsesClientDefault(t *testing.T) {
	responseBody := `{"blockhash":"5M77sHdwzH6rckuQwF8HL1w52n7hjrh4GVTFiF6T8QyB","previousBlockhash":"Aq9jSXe1jRzfiaBcRFLe4wm7j499vWVEeFQrq5nnXfZN","parentSlot":0,"rewards":[]}`
	server, closer := mockJSONRPC(t, stdjson.RawMessage(wrapIntoRPC(responseBody)))
	defer closer()

	cl := NewWithCommitment(server.URL, CommitmentProcessed)
	_, err := cl.GetBlockWithOpts(context.Background(), 42, &GetBlockOpts{})
	require.NoError(t, err)

	assertCommitment(t, server, "processed")
}

func TestCommitmentFallback_OptsExplicitWins(t *testing.T) {
	responseBody := `{"blockhash":"5M77sHdwzH6rckuQwF8HL1w52n7hjrh4GVTFiF6T8QyB","previousBlockhash":"Aq9jSXe1jRzfiaBcRFLe4wm7j499vWVEeFQrq5nnXfZN","parentSlot":0,"rewards":[]}`
	server, closer := mockJSONRPC(t, stdjson.RawMessage(wrapIntoRPC(responseBody)))
	defer closer()

	cl := NewWithCommitment(server.URL, CommitmentFinalized)
	_, err := cl.GetBlockWithOpts(context.Background(), 42, &GetBlockOpts{Commitment: CommitmentProcessed})
	require.NoError(t, err)

	assertCommitment(t, server, "processed")
}

// Commitment fallback: sendTransaction preflight ------------------------

func TestCommitmentFallback_SendTransactionPreflightUsesClientDefault(t *testing.T) {
	server, closer := mockJSONRPC(t, stdjson.RawMessage(wrapIntoRPC(`"1111111111111111111111111111111111111111111111111111111111111111"`)))
	defer closer()

	cl := NewWithCommitment(server.URL, CommitmentConfirmed)
	_, err := cl.SendEncodedTransactionWithOpts(context.Background(), "ignored", TransactionOpts{})
	require.NoError(t, err)

	obj := extractParamsObj(t, server, 1)
	assert.Equal(t, "confirmed", obj["preflightCommitment"], "expected client default on preflightCommitment")
}

func TestCommitmentFallback_SendTransactionPreflightExplicitWins(t *testing.T) {
	server, closer := mockJSONRPC(t, stdjson.RawMessage(wrapIntoRPC(`"1111111111111111111111111111111111111111111111111111111111111111"`)))
	defer closer()

	cl := NewWithCommitment(server.URL, CommitmentFinalized)
	_, err := cl.SendEncodedTransactionWithOpts(context.Background(), "ignored", TransactionOpts{PreflightCommitment: CommitmentProcessed})
	require.NoError(t, err)

	obj := extractParamsObj(t, server, 1)
	assert.Equal(t, "processed", obj["preflightCommitment"], "explicit preflight should win")
}

// Test helpers ----------------------------------------------------------

// nopJSONRPC is a stub JSONRPCClient used for constructor tests that never
// actually dispatch a request.
type nopJSONRPC struct{}

func (nopJSONRPC) CallForInto(context.Context, any, string, []any) error { return nil }
func (nopJSONRPC) CallWithCallback(context.Context, string, []any, func(*http.Request, *http.Response) error) error {
	return nil
}
func (nopJSONRPC) CallBatch(context.Context, jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error) {
	return nil, nil
}

// extractParamsObj returns the options object at params[idx] from the last
// recorded JSON-RPC request body.
func extractParamsObj(t *testing.T, server *mockJSONRPCServer, idx int) map[string]any {
	t.Helper()
	body := server.RequestBody(t)
	params, ok := body["params"].([]any)
	require.True(t, ok, "params is not a slice: %T", body["params"])
	require.Greater(t, len(params), idx, "params too short: %v", params)
	obj, ok := params[idx].(map[string]any)
	require.True(t, ok, "params[%d] is not an object: %T", idx, params[idx])
	return obj
}

// assertCommitment asserts that the last recorded JSON-RPC request carries
// the given commitment in its trailing options object. It searches params
// in reverse to accommodate methods that pass other arguments first.
func assertCommitment(t *testing.T, server *mockJSONRPCServer, want string) {
	t.Helper()
	body := server.RequestBody(t)
	params, ok := body["params"].([]any)
	require.True(t, ok, "params is not a slice: %T", body["params"])
	for i := len(params) - 1; i >= 0; i-- {
		obj, ok := params[i].(map[string]any)
		if !ok {
			continue
		}
		if got, present := obj["commitment"]; present {
			assert.Equal(t, want, got)
			return
		}
	}
	t.Fatalf("commitment not found in params: %v", params)
}

// Silence unused imports if solana import is not referenced elsewhere in
// this file. (Kept for use in future tests.)
var _ = solana.Signature{}
