// Copyright 2021 github.com/gagliardetto
// This file has been modified by github.com/gagliardetto
//
// Copyright 2020 dfuse Platform Inc.
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
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/klauspost/compress/gzhttp"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNotConfirmed = errors.New("not confirmed")
)

type Client struct {
	rpcURL            string
	rpcClient         JSONRPCClient
	defaultCommitment CommitmentType
}

type JSONRPCClient interface {
	CallForInto(ctx context.Context, out any, method string, params []any) error
	CallWithCallback(ctx context.Context, method string, params []any, callback func(*http.Request, *http.Response) error) error
	CallBatch(ctx context.Context, requests jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error)
}

// New creates a new Solana JSON RPC client with the default HTTP timeout
// (5 minutes) and a default commitment of "finalized".
// Client is safe for concurrent use by multiple goroutines.
func New(rpcEndpoint string) *Client {
	return NewWithTimeoutAndCommitment(rpcEndpoint, defaultTimeout, CommitmentFinalized)
}

// NewWithHeaders creates a new Solana JSON RPC client with the provided
// custom headers. The provided headers will be added to each RPC request.
// The HTTP timeout is the default (5 minutes) and the default commitment
// is "finalized".
func NewWithHeaders(rpcEndpoint string, headers map[string]string) *Client {
	opts := &jsonrpc.RPCClientOpts{
		HTTPClient:    newHTTP(defaultTimeout),
		CustomHeaders: headers,
	}
	rpcClient := jsonrpc.NewClientWithOpts(rpcEndpoint, opts)
	return newClientWithCommitment(rpcClient, CommitmentFinalized)
}

// NewWithCommitment creates a new Solana JSON RPC client with the default
// HTTP timeout and a caller-specified default commitment level. RPC methods
// that accept a commitment argument will fall back to this value when the
// caller passes the zero value.
func NewWithCommitment(rpcEndpoint string, commitment CommitmentType) *Client {
	return NewWithTimeoutAndCommitment(rpcEndpoint, defaultTimeout, commitment)
}

// NewWithTimeout creates a new Solana JSON RPC client with a caller-specified
// HTTP timeout and a default commitment of "finalized".
func NewWithTimeout(rpcEndpoint string, timeout time.Duration) *Client {
	return NewWithTimeoutAndCommitment(rpcEndpoint, timeout, CommitmentFinalized)
}

// NewWithTimeoutAndCommitment creates a new Solana JSON RPC client with a
// caller-specified HTTP timeout and default commitment level. Mirrors
// RpcClient::new_with_timeout_and_commitment in the Rust solana-rpc-client.
func NewWithTimeoutAndCommitment(rpcEndpoint string, timeout time.Duration, commitment CommitmentType) *Client {
	opts := &jsonrpc.RPCClientOpts{
		HTTPClient: newHTTP(timeout),
	}
	rpcClient := jsonrpc.NewClientWithOpts(rpcEndpoint, opts)
	return newClientWithCommitment(rpcClient, commitment)
}

// NewWithCustomRPCClient creates a new Solana RPC client with the provided
// RPC client. The default commitment is "finalized"; use
// NewWithCustomRPCClientAndCommitment to override.
func NewWithCustomRPCClient(rpcClient JSONRPCClient) *Client {
	return newClientWithCommitment(rpcClient, CommitmentFinalized)
}

// NewWithCustomRPCClientAndCommitment creates a new Solana RPC client with
// the provided RPC client and a caller-specified default commitment level.
func NewWithCustomRPCClientAndCommitment(rpcClient JSONRPCClient, commitment CommitmentType) *Client {
	return newClientWithCommitment(rpcClient, commitment)
}

func newClientWithCommitment(rpcClient JSONRPCClient, commitment CommitmentType) *Client {
	if commitment == "" {
		commitment = CommitmentFinalized
	}
	return &Client{
		rpcClient:         rpcClient,
		defaultCommitment: commitment,
	}
}

// Commitment returns the default commitment level that this client applies
// when a method is called without an explicit commitment argument.
func (cl *Client) Commitment() CommitmentType {
	return cl.defaultCommitment
}

// commitmentOrDefault returns c when it is non-empty, otherwise the client's
// default commitment. Used internally by every RPC method that accepts a
// commitment argument.
func (cl *Client) commitmentOrDefault(c CommitmentType) CommitmentType {
	if c != "" {
		return c
	}
	return cl.defaultCommitment
}

// Close closes the client.
func (cl *Client) Close() error {
	if cl.rpcClient == nil {
		return nil
	}
	if c, ok := cl.rpcClient.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

var (
	defaultMaxIdleConnsPerHost = 9
	defaultTimeout             = 5 * time.Minute
	defaultKeepAlive           = 180 * time.Second
)

func newHTTPTransport(timeout time.Duration) *http.Transport {
	return &http.Transport{
		IdleConnTimeout:     timeout,
		MaxConnsPerHost:     defaultMaxIdleConnsPerHost,
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		Proxy:               http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: defaultKeepAlive,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
	}
}

// newHTTP returns a new http.Client configured with the given total-request
// timeout. Safe for concurrent use by multiple goroutines.
func newHTTP(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: gzhttp.Transport(newHTTPTransport(timeout)),
	}
}

// RPCCallForInto allows to access the raw RPC client and send custom requests.
func (cl *Client) RPCCallForInto(ctx context.Context, out any, method string, params []any) error {
	return cl.rpcClient.CallForInto(ctx, out, method, params)
}

func (cl *Client) RPCCallWithCallback(
	ctx context.Context,
	method string,
	params []any,
	callback func(*http.Request, *http.Response) error,
) error {
	return cl.rpcClient.CallWithCallback(ctx, method, params, callback)
}

func (cl *Client) RPCCallBatch(
	ctx context.Context,
	requests jsonrpc.RPCRequests,
) (jsonrpc.RPCResponses, error) {
	return cl.rpcClient.CallBatch(ctx, requests)
}

func NewBoolean(b bool) *bool {
	return &b
}

func NewTransactionVersion(v uint64) *uint64 {
	return &v
}
