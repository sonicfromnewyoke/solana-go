// Copyright 2022 github.com/gagliardetto
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

// GetFeeForMessage returns the fee the network will charge for a particular
// base64-encoded Message. Supported CallOptions: WithCommitment,
// WithMinContextSlot.
func (cl *Client) GetFeeForMessage(
	ctx context.Context,
	message string,
	calls ...CallOption,
) (out *GetFeeForMessageResult, err error) {
	resolved := cl.resolveCallConfig(callConfig{}, calls)

	obj := M{}
	if resolved.commitment != "" {
		obj["commitment"] = resolved.commitment
	}
	if resolved.minContextSlot != nil {
		obj["minContextSlot"] = *resolved.minContextSlot
	}

	params := []any{message}
	if len(obj) > 0 {
		params = append(params, obj)
	}

	err = cl.rpcClient.CallForInto(ctx, &out, "getFeeForMessage", params)
	return
}

// GetFeeForMessageResult.Value is the fee corresponding to the message at
// the specified blockhash.
type GetFeeForMessageResult = RPCResponse[*uint64]
