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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

// TestClassifyRPCError_MapsKnownCodes table-drives every known custom
// error code and asserts that the classifier wraps it with the right
// sentinel so errors.Is finds it.
func TestClassifyRPCError_MapsKnownCodes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code     int
		sentinel error
	}{
		{ErrorCodeBlockCleanedUp, ErrBlockCleanedUp},
		{ErrorCodeSendTransactionPreflightFailure, ErrSendTransactionPreflightFailure},
		{ErrorCodeTransactionSignatureVerificationFailure, ErrTransactionSignatureVerificationFailure},
		{ErrorCodeBlockNotAvailable, ErrBlockNotAvailable},
		{ErrorCodeNodeUnhealthy, ErrNodeUnhealthy},
		{ErrorCodeTransactionPrecompileVerificationFailure, ErrTransactionPrecompileVerificationFailure},
		{ErrorCodeSlotSkipped, ErrSlotSkipped},
		{ErrorCodeNoSnapshot, ErrNoSnapshot},
		{ErrorCodeLongTermStorageSlotSkipped, ErrLongTermStorageSlotSkipped},
		{ErrorCodeKeyExcludedFromSecondaryIndex, ErrKeyExcludedFromSecondaryIndex},
		{ErrorCodeTransactionHistoryNotAvailable, ErrTransactionHistoryNotAvailable},
		{ErrorCodeScan, ErrScan},
		{ErrorCodeTransactionSignatureLenMismatch, ErrTransactionSignatureLenMismatch},
		{ErrorCodeBlockStatusNotAvailableYet, ErrBlockStatusNotAvailableYet},
		{ErrorCodeUnsupportedTransactionVersion, ErrUnsupportedTransactionVersion},
		{ErrorCodeMinContextSlotNotReached, ErrMinContextSlotNotReached},
		{ErrorCodeEpochRewardsPeriodActive, ErrEpochRewardsPeriodActive},
		{ErrorCodeSlotNotEpochBoundary, ErrSlotNotEpochBoundary},
		{ErrorCodeLongTermStorageUnreachable, ErrLongTermStorageUnreachable},
		{ErrorCodeFilterTransactionNotFound, ErrFilterTransactionNotFound},
	}
	require.Len(t, cases, 20, "every custom error code must have a test row")

	for _, tc := range cases {
		t.Run(fmt.Sprintf("code_%d", tc.code), func(t *testing.T) {
			t.Parallel()
			raw := &jsonrpc.RPCError{Code: tc.code, Message: "server message"}
			got := ClassifyRPCError(raw)
			require.Error(t, got)
			assert.ErrorIs(t, got, tc.sentinel,
				"classified error does not match sentinel for code %d", tc.code)
		})
	}
}

// TestClassifyRPCError_PreservesOriginalRPCError confirms that after
// classification the original *jsonrpc.RPCError is still reachable via
// errors.As, so callers can read the server-supplied Message and Data.
func TestClassifyRPCError_PreservesOriginalRPCError(t *testing.T) {
	t.Parallel()
	raw := &jsonrpc.RPCError{
		Code:    ErrorCodeMinContextSlotNotReached,
		Message: "Minimum context slot has not been reached",
		Data:    map[string]any{"contextSlot": float64(12345)},
	}
	classified := ClassifyRPCError(raw)
	require.Error(t, classified)

	assert.ErrorIs(t, classified, ErrMinContextSlotNotReached)

	var rpcErr *jsonrpc.RPCError
	require.True(t, errors.As(classified, &rpcErr), "original *jsonrpc.RPCError not reachable via errors.As")
	assert.Equal(t, ErrorCodeMinContextSlotNotReached, rpcErr.Code)
	assert.Equal(t, "Minimum context slot has not been reached", rpcErr.Message)
	data, ok := rpcErr.Data.(map[string]any)
	require.True(t, ok, "Data payload should still be accessible")
	assert.EqualValues(t, 12345, data["contextSlot"])
}

// TestClassifyRPCError_UnknownCodePassesThrough verifies that codes
// outside the custom-error range (standard JSON-RPC or future codes) are
// returned unchanged.
func TestClassifyRPCError_UnknownCodePassesThrough(t *testing.T) {
	t.Parallel()
	raw := &jsonrpc.RPCError{Code: -32000, Message: "generic server error"}
	got := ClassifyRPCError(raw)
	require.Same(t, any(raw), any(got),
		"unknown-code error should be returned unchanged (same pointer)")
}

// TestClassifyRPCError_NonRPCError passes through non-jsonrpc errors.
func TestClassifyRPCError_NonRPCError(t *testing.T) {
	t.Parallel()
	raw := errors.New("some other error")
	got := ClassifyRPCError(raw)
	assert.Same(t, raw, got)
}

// TestClassifyRPCError_NilInput returns nil for a nil input.
func TestClassifyRPCError_NilInput(t *testing.T) {
	t.Parallel()
	assert.Nil(t, ClassifyRPCError(nil))
}

// TestClassifyRPCError_WrappedInput handles inputs where *jsonrpc.RPCError
// is reachable via errors.As but not the direct value.
func TestClassifyRPCError_WrappedInput(t *testing.T) {
	t.Parallel()
	raw := &jsonrpc.RPCError{Code: ErrorCodeNodeUnhealthy, Message: "unhealthy"}
	wrapped := fmt.Errorf("call failed: %w", raw)

	classified := ClassifyRPCError(wrapped)
	assert.ErrorIs(t, classified, ErrNodeUnhealthy)

	var rpcErr *jsonrpc.RPCError
	require.True(t, errors.As(classified, &rpcErr))
	assert.Equal(t, ErrorCodeNodeUnhealthy, rpcErr.Code)
}

// TestSentinels_AreDistinct sanity-checks that every sentinel is a
// distinct error value, i.e. no two constants were accidentally aliased.
func TestSentinels_AreDistinct(t *testing.T) {
	t.Parallel()
	sentinels := []error{
		ErrBlockCleanedUp,
		ErrSendTransactionPreflightFailure,
		ErrTransactionSignatureVerificationFailure,
		ErrBlockNotAvailable,
		ErrNodeUnhealthy,
		ErrTransactionPrecompileVerificationFailure,
		ErrSlotSkipped,
		ErrNoSnapshot,
		ErrLongTermStorageSlotSkipped,
		ErrKeyExcludedFromSecondaryIndex,
		ErrTransactionHistoryNotAvailable,
		ErrScan,
		ErrTransactionSignatureLenMismatch,
		ErrBlockStatusNotAvailableYet,
		ErrUnsupportedTransactionVersion,
		ErrMinContextSlotNotReached,
		ErrEpochRewardsPeriodActive,
		ErrSlotNotEpochBoundary,
		ErrLongTermStorageUnreachable,
		ErrFilterTransactionNotFound,
	}
	seen := make(map[error]struct{}, len(sentinels))
	for _, s := range sentinels {
		_, dup := seen[s]
		require.Falsef(t, dup, "duplicate sentinel: %v", s)
		seen[s] = struct{}{}
	}
	assert.Len(t, seen, 20, "expected 20 distinct sentinel values")
}
