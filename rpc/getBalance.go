// Copyright 2021 github.com/gagliardetto
// This file has been modified by github.com/gagliardetto
//
// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package rpc

import (
	"context"

	"github.com/gagliardetto/solana-go"
)

// GetBalance returns the balance of the account of provided publicKey.
// Supported CallOptions: WithCommitment, WithMinContextSlot.
func (cl *Client) GetBalance(
	ctx context.Context,
	publicKey solana.PublicKey,
	calls ...CallOption,
) (out *GetBalanceResult, err error) {
	resolved := cl.resolveCallConfig(callConfig{}, calls)

	obj := M{}
	if resolved.commitment != "" {
		obj["commitment"] = resolved.commitment
	}
	if resolved.minContextSlot != nil {
		obj["minContextSlot"] = *resolved.minContextSlot
	}

	params := []any{publicKey}
	if len(obj) > 0 {
		params = append(params, obj)
	}

	err = cl.rpcClient.CallForInto(ctx, &out, "getBalance", params)
	return
}
