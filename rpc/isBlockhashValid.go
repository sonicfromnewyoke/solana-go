package rpc

import (
	"context"

	"github.com/gagliardetto/solana-go"
)

// IsBlockhashValid returns whether a blockhash is still valid or not.
// Supported CallOptions: WithCommitment, WithMinContextSlot.
func (cl *Client) IsBlockhashValid(
	ctx context.Context,
	blockHash solana.Hash,
	calls ...CallOption,
) (out *IsValidBlockhashResult, err error) {
	resolved := cl.resolveCallConfig(callConfig{}, calls)

	obj := M{}
	if resolved.commitment != "" {
		obj["commitment"] = string(resolved.commitment)
	}
	if resolved.minContextSlot != nil {
		obj["minContextSlot"] = *resolved.minContextSlot
	}

	params := []any{blockHash}
	if len(obj) > 0 {
		params = append(params, obj)
	}

	err = cl.rpcClient.CallForInto(ctx, &out, "isBlockhashValid", params)
	return
}
