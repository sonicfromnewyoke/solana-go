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

// GetBlocks returns a list of confirmed blocks between two slots.
// The result is an array of u64 integers listing confirmed blocks between
// startSlot and either endSlot (if provided) or the latest confirmed block.
// Max range allowed is 500,000 slots.
//
// Supported CallOptions: WithCommitment. "processed" commitment is not
// supported by this method.
func (cl *Client) GetBlocks(
	ctx context.Context,
	startSlot uint64,
	endSlot *uint64,
	calls ...CallOption,
) (out BlocksResult, err error) {
	resolved := cl.resolveCallConfig(callConfig{}, calls)

	params := []any{startSlot}
	if endSlot != nil {
		params = append(params, endSlot)
	}
	if resolved.commitment != "" {
		params = append(params, M{"commitment": resolved.commitment})
	}

	err = cl.rpcClient.CallForInto(ctx, &out, "getBlocks", params)
	return
}

type BlocksResult []uint64
