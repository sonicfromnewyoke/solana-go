// Copyright 2021 github.com/gagliardetto
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

// Solana JSON-RPC server error codes and sentinels.
//
// The Solana RPC server returns errors with codes in the -32001 .. -32020
// range; these are documented in agave's rpc-client-api/src/custom_error.rs
// and kept in sync with @solana/web3.js. This file exposes them as Go
// constants plus matching sentinel errors so callers can branch on
// well-known conditions without hard-coding integer codes:
//
//	err := client.GetBlock(ctx, slot)
//	if errors.Is(err, rpc.ErrBlockCleanedUp) { ... }
//	if errors.Is(err, rpc.ErrNodeUnhealthy)  { /* maybe retry */ }
//
// To convert a raw *jsonrpc.RPCError into a sentinel-wrapped form, call
// ClassifyRPCError. The wrapped error chain preserves the original
// *jsonrpc.RPCError so callers can still use errors.As to read Data.
//
// References (stable):
//   - custom error codes:  https://github.com/anza-xyz/agave/blob/master/rpc-client-api/src/custom_error.rs
//   - transaction errors:  https://github.com/anza-xyz/agave/blob/master/sdk/transaction-error/src/lib.rs
//   - instruction errors:  https://github.com/anza-xyz/agave/blob/master/sdk/instruction/src/error.rs

package rpc

import (
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

// JSON-RPC server error codes. Mirrors the JSON_RPC_SERVER_ERROR_*
// constants in agave's rpc-client-api/src/custom_error.rs.
const (
	ErrorCodeBlockCleanedUp                          = -32001
	ErrorCodeSendTransactionPreflightFailure         = -32002
	ErrorCodeTransactionSignatureVerificationFailure = -32003
	ErrorCodeBlockNotAvailable                       = -32004
	ErrorCodeNodeUnhealthy                           = -32005
	ErrorCodeTransactionPrecompileVerificationFailure = -32006
	ErrorCodeSlotSkipped                             = -32007
	ErrorCodeNoSnapshot                              = -32008
	ErrorCodeLongTermStorageSlotSkipped              = -32009
	ErrorCodeKeyExcludedFromSecondaryIndex           = -32010
	ErrorCodeTransactionHistoryNotAvailable          = -32011
	ErrorCodeScan                                    = -32012
	ErrorCodeTransactionSignatureLenMismatch         = -32013
	ErrorCodeBlockStatusNotAvailableYet              = -32014
	ErrorCodeUnsupportedTransactionVersion           = -32015
	ErrorCodeMinContextSlotNotReached                = -32016
	ErrorCodeEpochRewardsPeriodActive                = -32017
	ErrorCodeSlotNotEpochBoundary                    = -32018
	ErrorCodeLongTermStorageUnreachable              = -32019
	ErrorCodeFilterTransactionNotFound               = -32020
)

// Sentinel errors for the Solana RPC server error codes. Compare with
// errors.Is after calling ClassifyRPCError (or directly on errors returned
// by methods that already pass the response through the classifier).
var (
	ErrBlockCleanedUp                           = errors.New("block cleaned up")
	ErrSendTransactionPreflightFailure          = errors.New("send transaction preflight failure")
	ErrTransactionSignatureVerificationFailure  = errors.New("transaction signature verification failure")
	ErrBlockNotAvailable                        = errors.New("block not available")
	ErrNodeUnhealthy                            = errors.New("node unhealthy")
	ErrTransactionPrecompileVerificationFailure = errors.New("transaction precompile verification failure")
	ErrSlotSkipped                              = errors.New("slot skipped")
	ErrNoSnapshot                               = errors.New("no snapshot")
	ErrLongTermStorageSlotSkipped               = errors.New("slot skipped in long-term storage")
	ErrKeyExcludedFromSecondaryIndex            = errors.New("key excluded from secondary index")
	ErrTransactionHistoryNotAvailable           = errors.New("transaction history not available")
	ErrScan                                     = errors.New("scan error")
	ErrTransactionSignatureLenMismatch          = errors.New("transaction signature length mismatch")
	ErrBlockStatusNotAvailableYet               = errors.New("block status not available yet")
	ErrUnsupportedTransactionVersion            = errors.New("unsupported transaction version")
	ErrMinContextSlotNotReached                 = errors.New("minimum context slot not reached")
	ErrEpochRewardsPeriodActive                 = errors.New("epoch rewards period active")
	ErrSlotNotEpochBoundary                     = errors.New("slot is not an epoch boundary")
	ErrLongTermStorageUnreachable               = errors.New("long-term storage unreachable")
	ErrFilterTransactionNotFound                = errors.New("filter transaction not found")
)

// sentinelByCode maps a JSON-RPC server error code to its matching sentinel.
// Returns nil for codes outside the custom-error range (standard JSON-RPC
// codes like -32700, server-defined -32000, or future additions).
func sentinelByCode(code int) error {
	switch code {
	case ErrorCodeBlockCleanedUp:
		return ErrBlockCleanedUp
	case ErrorCodeSendTransactionPreflightFailure:
		return ErrSendTransactionPreflightFailure
	case ErrorCodeTransactionSignatureVerificationFailure:
		return ErrTransactionSignatureVerificationFailure
	case ErrorCodeBlockNotAvailable:
		return ErrBlockNotAvailable
	case ErrorCodeNodeUnhealthy:
		return ErrNodeUnhealthy
	case ErrorCodeTransactionPrecompileVerificationFailure:
		return ErrTransactionPrecompileVerificationFailure
	case ErrorCodeSlotSkipped:
		return ErrSlotSkipped
	case ErrorCodeNoSnapshot:
		return ErrNoSnapshot
	case ErrorCodeLongTermStorageSlotSkipped:
		return ErrLongTermStorageSlotSkipped
	case ErrorCodeKeyExcludedFromSecondaryIndex:
		return ErrKeyExcludedFromSecondaryIndex
	case ErrorCodeTransactionHistoryNotAvailable:
		return ErrTransactionHistoryNotAvailable
	case ErrorCodeScan:
		return ErrScan
	case ErrorCodeTransactionSignatureLenMismatch:
		return ErrTransactionSignatureLenMismatch
	case ErrorCodeBlockStatusNotAvailableYet:
		return ErrBlockStatusNotAvailableYet
	case ErrorCodeUnsupportedTransactionVersion:
		return ErrUnsupportedTransactionVersion
	case ErrorCodeMinContextSlotNotReached:
		return ErrMinContextSlotNotReached
	case ErrorCodeEpochRewardsPeriodActive:
		return ErrEpochRewardsPeriodActive
	case ErrorCodeSlotNotEpochBoundary:
		return ErrSlotNotEpochBoundary
	case ErrorCodeLongTermStorageUnreachable:
		return ErrLongTermStorageUnreachable
	case ErrorCodeFilterTransactionNotFound:
		return ErrFilterTransactionNotFound
	}
	return nil
}

// ClassifyRPCError converts a raw JSON-RPC error into a sentinel-wrapped
// form so callers can branch on well-known error kinds with errors.Is.
//
// If err wraps a *jsonrpc.RPCError whose Code matches one of the known
// Solana server error codes, the returned error is a chain that:
//   - satisfies errors.Is(err, ErrXxx) for the matching sentinel, and
//   - preserves the original *jsonrpc.RPCError (accessible via errors.As)
//     so callers can still read rpcErr.Data for structured details.
//
// If err is not a *jsonrpc.RPCError, or its Code is outside the known
// range, the input is returned unchanged.
func ClassifyRPCError(err error) error {
	if err == nil {
		return nil
	}
	var rpcErr *jsonrpc.RPCError
	if !errors.As(err, &rpcErr) {
		return err
	}
	sentinel := sentinelByCode(rpcErr.Code)
	if sentinel == nil {
		return err
	}
	return fmt.Errorf("%w: %w", sentinel, rpcErr)
}
