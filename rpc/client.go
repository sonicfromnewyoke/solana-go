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
	"go.uber.org/ratelimit"
	"golang.org/x/time/rate"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNotConfirmed = errors.New("not confirmed")
)

// ============================================================================
// Types
// ============================================================================

type Client struct {
	rpcURL                string
	rpcClient             JSONRPCClient
	defaultCommitment     CommitmentType
	defaultMinContextSlot *uint64
}

type JSONRPCClient interface {
	CallForInto(ctx context.Context, out any, method string, params []any) error
	CallWithCallback(ctx context.Context, method string, params []any, callback func(*http.Request, *http.Response) error) error
	CallBatch(ctx context.Context, requests jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error)
}

// Option configures a Client built by NewClient or NewWithCustomRPCClient.
//
// Options are applied in order; later options override earlier ones with the
// same target field. Options that target HTTP-layer behavior (WithTimeout,
// WithHeaders, WithHTTPClient) are ignored when the Client is constructed
// from a caller-supplied JSONRPCClient, since that client owns its own
// transport.
//
// Some options (notably WithCommitment) also satisfy CallOption and can be
// passed to individual RPC methods to override the client-level default for
// a single call.
type Option interface {
	applyClient(*clientConfig)
}

// CallOption overrides Client-level defaults for a single RPC call. Options
// that make sense in both contexts (WithCommitment) implement both Option
// and CallOption; construction-only options (WithTimeout, WithHeaders,
// WithHTTPClient, WithRateLimit, WithLimiter) implement only Option and
// cannot be passed to RPC methods — the compiler rejects them.
type CallOption interface {
	applyCall(*callConfig)
}

// callConfig is the resolved per-call state for an RPC method. A method
// starts from the client's defaults, layers on any struct-based Opts, then
// applies CallOption overrides last.
type callConfig struct {
	commitment     CommitmentType
	minContextSlot *uint64
}

// dualOption implements both Option and CallOption; used for options whose
// semantics are meaningful at both the client level (sets default) and the
// call level (overrides the default for one call).
type dualOption struct {
	clientFn func(*clientConfig)
	callFn   func(*callConfig)
}

func (d dualOption) applyClient(cfg *clientConfig) { d.clientFn(cfg) }
func (d dualOption) applyCall(cfg *callConfig)     { d.callFn(cfg) }

// clientOnlyOption implements only Option; used for options that configure
// construction-time transport behavior and have no meaning for a single
// RPC call (timeout, headers, rate limiting, etc.).
type clientOnlyOption struct {
	fn func(*clientConfig)
}

func (c clientOnlyOption) applyClient(cfg *clientConfig) { c.fn(cfg) }

// ============================================================================
// Constructors
// ============================================================================

// NewClient creates a new Solana JSON RPC client configured by the given
// options. This is the primary constructor; the New / NewWith* variants are
// thin delegators preserved for API compatibility and Rust-SDK parity.
//
// Defaults: 5-minute HTTP timeout, "finalized" default commitment, no custom
// headers. Client is safe for concurrent use by multiple goroutines.
//
//	client := rpc.NewClient("https://api.mainnet-beta.solana.com",
//	    rpc.WithCommitment(rpc.CommitmentConfirmed),
//	    rpc.WithTimeout(30*time.Second),
//	)
func NewClient(rpcEndpoint string, opts ...Option) *Client {
	cfg := defaultClientConfig()
	for _, o := range opts {
		o.applyClient(&cfg)
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = newHTTP(cfg.timeout)
	}
	var rpcClient JSONRPCClient = jsonrpc.NewClientWithOpts(rpcEndpoint, &jsonrpc.RPCClientOpts{
		HTTPClient:    httpClient,
		CustomHeaders: cfg.headers,
	})
	for _, mw := range cfg.middlewares {
		rpcClient = mw(rpcClient)
	}
	return &Client{
		rpcURL:                rpcEndpoint,
		rpcClient:             rpcClient,
		defaultCommitment:     cfg.commitment,
		defaultMinContextSlot: cfg.minContextSlot,
	}
}

// NewWithCustomRPCClient creates a new Solana RPC client backed by a
// caller-supplied JSONRPCClient. Use this when you need a non-HTTP transport
// (mocks in tests, batching middleware, rate limiting, etc.).
//
// HTTP-layer options (WithTimeout, WithHeaders, WithHTTPClient) are ignored
// because the supplied client owns its transport. WithCommitment,
// WithRateLimit, and WithLimiter are honored — the rate-limit middleware
// wraps the supplied client.
func NewWithCustomRPCClient(rpcClient JSONRPCClient, opts ...Option) *Client {
	cfg := defaultClientConfig()
	for _, o := range opts {
		o.applyClient(&cfg)
	}
	for _, mw := range cfg.middlewares {
		rpcClient = mw(rpcClient)
	}
	return &Client{
		rpcClient:             rpcClient,
		defaultCommitment:     cfg.commitment,
		defaultMinContextSlot: cfg.minContextSlot,
	}
}

// New creates a new Solana JSON RPC client with default configuration
// (5-minute HTTP timeout, "finalized" commitment). Equivalent to
// NewClient(rpcEndpoint). Client is safe for concurrent use by multiple
// goroutines.
func New(rpcEndpoint string) *Client {
	return NewClient(rpcEndpoint)
}

// NewWithHeaders creates a new Solana JSON RPC client with the given custom
// HTTP headers. Equivalent to NewClient(url, WithHeaders(h)).
func NewWithHeaders(rpcEndpoint string, headers map[string]string) *Client {
	return NewClient(rpcEndpoint, WithHeaders(headers))
}

// NewWithCommitment creates a new Solana JSON RPC client with a
// caller-specified default commitment level. Equivalent to
// NewClient(url, WithCommitment(c)). Mirrors
// RpcClient::new_with_commitment in the Rust solana-rpc-client.
func NewWithCommitment(rpcEndpoint string, commitment CommitmentType) *Client {
	return NewClient(rpcEndpoint, WithCommitment(commitment))
}

// NewWithTimeout creates a new Solana JSON RPC client with a caller-specified
// HTTP timeout. Equivalent to NewClient(url, WithTimeout(d)). Mirrors
// RpcClient::new_with_timeout in the Rust solana-rpc-client.
func NewWithTimeout(rpcEndpoint string, timeout time.Duration) *Client {
	return NewClient(rpcEndpoint, WithTimeout(timeout))
}

// NewWithTimeoutAndCommitment creates a new Solana JSON RPC client with a
// caller-specified HTTP timeout and default commitment level. Equivalent to
// NewClient(url, WithTimeout(d), WithCommitment(c)). Mirrors
// RpcClient::new_with_timeout_and_commitment in the Rust solana-rpc-client.
func NewWithTimeoutAndCommitment(rpcEndpoint string, timeout time.Duration, commitment CommitmentType) *Client {
	return NewClient(rpcEndpoint, WithTimeout(timeout), WithCommitment(commitment))
}

// NewWithCustomRPCClientAndCommitment creates a new Solana RPC client with
// the provided RPC client and a caller-specified default commitment level.
// Equivalent to NewWithCustomRPCClient(rpcClient, WithCommitment(c)).
func NewWithCustomRPCClientAndCommitment(rpcClient JSONRPCClient, commitment CommitmentType) *Client {
	return NewWithCustomRPCClient(rpcClient, WithCommitment(commitment))
}

// NewWithRateLimit creates a rate-limited *Client using a leaky-bucket
// limiter (go.uber.org/ratelimit).
//
// Deprecated: prefer NewClient(rpcEndpoint, WithRateLimit(rps)), which wires
// the same middleware alongside the other client options in one place.
func NewWithRateLimit(rpcEndpoint string, rps int) *Client {
	return NewClient(rpcEndpoint, WithRateLimit(rps))
}

// NewWithLimiter creates a rate-limited *Client using a token-bucket limiter
// (golang.org/x/time/rate).
// Example: NewWithLimiter(url, rate.Every(time.Second), 1).
//
// Deprecated: prefer NewClient(rpcEndpoint, WithLimiter(every, burst)), which
// wires the same middleware alongside the other client options in one place.
func NewWithLimiter(rpcEndpoint string, every rate.Limit, b int) *Client {
	return NewClient(rpcEndpoint, WithLimiter(every, b))
}

// ============================================================================
// Options
// ============================================================================

// WithCommitment sets the commitment level. Applied at construction time
// (via NewClient), it becomes the client-wide default. Applied at call time
// (via any method that takes CallOption), it overrides the default for a
// single call.
//
// Passing the empty string is a no-op at both layers: the existing default
// is preserved. This is deliberate so that passing a caller-held variable
// that happens to be empty falls back to the client default instead of
// producing a request with no commitment.
func WithCommitment(c CommitmentType) dualOption {
	apply := func(assign func()) {
		if c != "" {
			assign()
		}
	}
	return dualOption{
		clientFn: func(cfg *clientConfig) { apply(func() { cfg.commitment = c }) },
		callFn:   func(cfg *callConfig) { apply(func() { cfg.commitment = c }) },
	}
}

// WithMinContextSlot sets the minimum slot at which the node must have
// already processed the request. The RPC returns an error if the node is
// behind this slot. Applied at construction time (via NewClient) it becomes
// a client-wide floor; applied at call time it overrides the floor for a
// single call. Valid for the subset of RPC methods that accept
// minContextSlot on the wire (getBalance, getAccountInfo, getSlot, etc.);
// ignored on methods that do not.
//
// There is no way to "clear" a client-level default at call time — the
// minimum value is 0 (genesis), which is a valid slot. If you need a call
// that ignores the client's minContextSlot, construct a second Client
// without the default.
func WithMinContextSlot(slot uint64) dualOption {
	s := slot
	return dualOption{
		clientFn: func(cfg *clientConfig) { cfg.minContextSlot = &s },
		callFn:   func(cfg *callConfig) { cfg.minContextSlot = &s },
	}
}

// WithTimeout sets the HTTP request timeout applied to the underlying
// http.Client. The timeout bounds the entire request lifecycle (dial, TLS,
// write, read). A zero duration means "no timeout" per http.Client semantics
// — rarely what callers want; prefer an explicit duration. Ignored when
// WithHTTPClient or NewWithCustomRPCClient is also used. Construction-only.
func WithTimeout(timeout time.Duration) clientOnlyOption {
	return clientOnlyOption{fn: func(cfg *clientConfig) { cfg.timeout = timeout }}
}

// WithHeaders adds HTTP headers to every RPC request. The given map is
// used as-is; callers should not mutate it after construction. A nil or
// empty map is treated as "no override", so layering headers across
// multiple options (or falling back to defaults) is safe.
//
// Ignored when NewWithCustomRPCClient is used, since the custom client
// owns its transport. Construction-only.
func WithHeaders(headers map[string]string) clientOnlyOption {
	return clientOnlyOption{fn: func(cfg *clientConfig) {
		if len(headers) > 0 {
			cfg.headers = headers
		}
	}}
}

// WithHTTPClient replaces the default http.Client with a caller-supplied
// one. Use this to install custom transports (proxies, TLS configs,
// middleware). When set, WithTimeout is ignored; the caller is responsible
// for configuring the timeout on their http.Client.
//
// Ignored when NewWithCustomRPCClient is used. Construction-only.
func WithHTTPClient(client *http.Client) clientOnlyOption {
	return clientOnlyOption{fn: func(cfg *clientConfig) { cfg.httpClient = client }}
}

// WithRateLimit throttles calls with a leaky-bucket limiter
// (go.uber.org/ratelimit) configured for rps requests per second. Waits do
// not honor context cancellation; a caller in a hot loop will block on
// Take() until the next token is available. Use WithLimiter if you need
// context-aware waits.
//
// Multiple rate-limit options stack as nested middleware in the order they
// appear. Construction-only.
func WithRateLimit(rps int) clientOnlyOption {
	return clientOnlyOption{fn: func(cfg *clientConfig) {
		cfg.middlewares = append(cfg.middlewares, func(c JSONRPCClient) JSONRPCClient {
			return &clientWithRateLimiting{
				rpcClient:   c,
				rateLimiter: ratelimit.New(rps),
			}
		})
	}}
}

// WithLimiter throttles calls with a token-bucket limiter
// (golang.org/x/time/rate), honoring context cancellation on Wait. The first
// argument is the token arrival rate (e.g. rate.Every(time.Second)); the
// second is the burst size.
//
// Multiple rate-limit options stack as nested middleware in the order they
// appear. Construction-only.
func WithLimiter(every rate.Limit, burst int) clientOnlyOption {
	return clientOnlyOption{fn: func(cfg *clientConfig) {
		cfg.middlewares = append(cfg.middlewares, func(c JSONRPCClient) JSONRPCClient {
			return &clientWithLimiter{
				rpcClient: c,
				limiter:   rate.NewLimiter(every, burst),
			}
		})
	}}
}

// ============================================================================
// Client methods
// ============================================================================

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

// resolveCallConfig builds the per-call state for an RPC method. Pure
// function: takes the client's defaults, layers the method's struct-based
// opts seed on top (non-zero fields override), then applies each CallOption
// in order. Precedence: client default < seed (opts struct) < CallOption.
//
// A zero-value field in seed is treated as "not set" and leaves the client
// default in place — empty CommitmentType, nil *uint64.
func resolveCallConfig(defaults, seed callConfig, calls []CallOption) callConfig {
	resolved := defaults
	if seed.commitment != "" {
		resolved.commitment = seed.commitment
	}
	if seed.minContextSlot != nil {
		resolved.minContextSlot = seed.minContextSlot
	}
	for _, o := range calls {
		o.applyCall(&resolved)
	}
	return resolved
}

// clientDefaults returns the client-wide callConfig defaults used to seed
// per-call state. Exists so RPC methods don't need to know about the two
// default fields on Client individually.
func (cl *Client) clientDefaults() callConfig {
	return callConfig{
		commitment:     cl.defaultCommitment,
		minContextSlot: cl.defaultMinContextSlot,
	}
}

// resolveCallConfig is a method-wrapped shortcut over the pure
// resolveCallConfig function, seeding from the client's defaults.
func (cl *Client) resolveCallConfig(seed callConfig, calls []CallOption) callConfig {
	return resolveCallConfig(cl.clientDefaults(), seed, calls)
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

// ============================================================================
// Internals
// ============================================================================

// clientConfig is the internal configuration accumulator used by the
// functional-options constructors. It is never exposed to callers; options
// mutate it before NewClient materializes the resulting Client.
type clientConfig struct {
	commitment     CommitmentType
	minContextSlot *uint64
	timeout        time.Duration
	headers        map[string]string
	httpClient     *http.Client
	middlewares    []func(JSONRPCClient) JSONRPCClient
}

func defaultClientConfig() clientConfig {
	return clientConfig{
		commitment: CommitmentFinalized,
		timeout:    defaultTimeout,
	}
}

const (
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

// clientWithRateLimiting throttles calls to the wrapped JSONRPCClient using a
// leaky-bucket limiter (go.uber.org/ratelimit). Take() blocks until the next
// token is available; it does not honor context cancellation mid-wait.
type clientWithRateLimiting struct {
	rpcClient   JSONRPCClient
	rateLimiter ratelimit.Limiter
}

var _ JSONRPCClient = (*clientWithRateLimiting)(nil)

func (wr *clientWithRateLimiting) CallForInto(ctx context.Context, out any, method string, params []any) error {
	wr.rateLimiter.Take()
	return wr.rpcClient.CallForInto(ctx, out, method, params)
}

func (wr *clientWithRateLimiting) CallWithCallback(
	ctx context.Context,
	method string,
	params []any,
	callback func(*http.Request, *http.Response) error,
) error {
	wr.rateLimiter.Take()
	return wr.rpcClient.CallWithCallback(ctx, method, params, callback)
}

func (wr *clientWithRateLimiting) CallBatch(
	ctx context.Context,
	requests jsonrpc.RPCRequests,
) (jsonrpc.RPCResponses, error) {
	wr.rateLimiter.Take()
	return wr.rpcClient.CallBatch(ctx, requests)
}

func (wr *clientWithRateLimiting) Close() error {
	if c, ok := wr.rpcClient.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// clientWithLimiter throttles calls to the wrapped JSONRPCClient using a
// token-bucket limiter (golang.org/x/time/rate). Wait honors context
// cancellation, so callers can bound total wait time with ctx.
type clientWithLimiter struct {
	rpcClient JSONRPCClient
	limiter   *rate.Limiter
}

var _ JSONRPCClient = (*clientWithLimiter)(nil)

func (wr *clientWithLimiter) CallForInto(ctx context.Context, out any, method string, params []any) error {
	if err := wr.limiter.Wait(ctx); err != nil {
		return err
	}
	return wr.rpcClient.CallForInto(ctx, out, method, params)
}

func (wr *clientWithLimiter) CallWithCallback(
	ctx context.Context,
	method string,
	params []any,
	callback func(*http.Request, *http.Response) error,
) error {
	if err := wr.limiter.Wait(ctx); err != nil {
		return err
	}
	return wr.rpcClient.CallWithCallback(ctx, method, params, callback)
}

func (wr *clientWithLimiter) CallBatch(
	ctx context.Context,
	requests jsonrpc.RPCRequests,
) (jsonrpc.RPCResponses, error) {
	if err := wr.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return wr.rpcClient.CallBatch(ctx, requests)
}

func (wr *clientWithLimiter) Close() error {
	if c, ok := wr.rpcClient.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// ============================================================================
// Misc helpers
// ============================================================================

// Ptr returns a pointer to a copy of v. It's the generic replacement for
// the old NewBoolean / NewTransactionVersion helpers and is handy for
// populating the *T optional fields on Opts structs without a temporary
// variable:
//
//	opts := &GetAccountInfoOpts{MinContextSlot: rpc.Ptr(uint64(12345))}
//	opts := &GetBlockOpts{Rewards: rpc.Ptr(false)}
func Ptr[T any](v T) *T {
	return &v
}
