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

// GetStakeMinimumDelegation returns the stake minimum delegation, in lamports.
// Supported CallOptions: WithCommitment.
func (cl *Client) GetStakeMinimumDelegation(
	ctx context.Context,
	calls ...CallOption,
) (out *GetStakeMinimumDelegationResult, err error) {
	resolved := cl.resolveCallConfig(callConfig{}, calls)

	params := []any{}
	if resolved.commitment != "" {
		params = append(params, M{"commitment": string(resolved.commitment)})
	}

	err = cl.rpcClient.CallForInto(ctx, &out, "getStakeMinimumDelegation", params)
	return
}
