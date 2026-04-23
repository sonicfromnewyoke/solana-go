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

package rpc

import (
	"context"
)

// GetTransactionCount returns the current Transaction count from the ledger.
// Supported CallOptions: WithCommitment, WithMinContextSlot.
func (cl *Client) GetTransactionCount(
	ctx context.Context,
	calls ...CallOption,
) (out uint64, err error) {
	resolved := cl.resolveCallConfig(callConfig{}, calls)

	obj := M{}
	if resolved.commitment != "" {
		obj["commitment"] = resolved.commitment
	}
	if resolved.minContextSlot != nil {
		obj["minContextSlot"] = *resolved.minContextSlot
	}

	params := []any{}
	if len(obj) > 0 {
		params = append(params, obj)
	}

	err = cl.rpcClient.CallForInto(ctx, &out, "getTransactionCount", params)
	return
}
