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

package solana

// Typed transaction and instruction errors.
//
// This is a port of the upstream anza-xyz/solana-sdk `transaction-error`
// (TransactionError) and `instruction-error` (InstructionError) crates: the
// full variant taxonomy plus the two encodings it travels in:
//
//   - bincode: u32 little-endian variant tag followed by the payload, the
//     form stored on-chain infrastructure and produced by
//     bincode::serialize(&TransactionError) in Rust;
//   - serde JSON: unit variants as strings ("AccountInUse"), payload
//     variants as single-key objects ({"InstructionError":[0,{"Custom":6}]}),
//     the form returned by JSON-RPC endpoints such as getTransaction and
//     simulateTransaction.
//
// Both types implement the error interface with messages matching the Rust
// Display impls, and DebugString matches Rust's {:?} rendering.

import (
	"bytes"
	"encoding/binary"
	"fmt"

	bin "github.com/gagliardetto/binary"
	gojson "github.com/goccy/go-json"
)

// TransactionErrorType identifies a TransactionError variant. Its numeric
// value is the serde/bincode variant tag of the upstream Rust enum.
type TransactionErrorType uint32

const (
	TxErrAccountInUse TransactionErrorType = iota
	TxErrAccountLoadedTwice
	TxErrAccountNotFound
	TxErrProgramAccountNotFound
	TxErrInsufficientFundsForFee
	TxErrInvalidAccountForFee
	TxErrAlreadyProcessed
	TxErrBlockhashNotFound
	TxErrInstructionError
	TxErrCallChainTooDeep
	TxErrMissingSignatureForFee
	TxErrInvalidAccountIndex
	TxErrSignatureFailure
	TxErrInvalidProgramForExecution
	TxErrSanitizeFailure
	TxErrClusterMaintenance
	TxErrAccountBorrowOutstanding
	TxErrWouldExceedMaxBlockCostLimit
	TxErrUnsupportedVersion
	TxErrInvalidWritableAccount
	TxErrWouldExceedMaxAccountCostLimit
	TxErrWouldExceedAccountDataBlockLimit
	TxErrTooManyAccountLocks
	TxErrAddressLookupTableNotFound
	TxErrInvalidAddressLookupTableOwner
	TxErrInvalidAddressLookupTableData
	TxErrInvalidAddressLookupTableIndex
	TxErrInvalidRentPayingAccount
	TxErrWouldExceedMaxVoteCostLimit
	TxErrWouldExceedAccountDataTotalLimit
	TxErrDuplicateInstruction
	TxErrInsufficientFundsForRent
	TxErrMaxLoadedAccountsDataSizeExceeded
	TxErrInvalidLoadedAccountsDataSizeLimit
	TxErrResanitizationNeeded
	TxErrProgramExecutionTemporarilyRestricted
	TxErrUnbalancedTransaction
	TxErrProgramCacheHitMaxLimit
	TxErrCommitCancelled
)

// String returns the Rust variant name.
func (t TransactionErrorType) String() string {
	if int(t) < len(transactionErrorNames) {
		return transactionErrorNames[t]
	}
	return fmt.Sprintf("TransactionErrorType(%d)", uint32(t))
}

// InstructionErrorType identifies an InstructionError variant. Its numeric
// value is the serde/bincode variant tag of the upstream Rust enum.
type InstructionErrorType uint32

const (
	InstrErrGenericError InstructionErrorType = iota
	InstrErrInvalidArgument
	InstrErrInvalidInstructionData
	InstrErrInvalidAccountData
	InstrErrAccountDataTooSmall
	InstrErrInsufficientFunds
	InstrErrIncorrectProgramId
	InstrErrMissingRequiredSignature
	InstrErrAccountAlreadyInitialized
	InstrErrUninitializedAccount
	InstrErrUnbalancedInstruction
	InstrErrModifiedProgramId
	InstrErrExternalAccountLamportSpend
	InstrErrExternalAccountDataModified
	InstrErrReadonlyLamportChange
	InstrErrReadonlyDataModified
	InstrErrDuplicateAccountIndex
	InstrErrExecutableModified
	InstrErrRentEpochModified
	InstrErrNotEnoughAccountKeys
	InstrErrAccountDataSizeChanged
	InstrErrAccountNotExecutable
	InstrErrAccountBorrowFailed
	InstrErrAccountBorrowOutstanding
	InstrErrDuplicateAccountOutOfSync
	InstrErrCustom
	InstrErrInvalidError
	InstrErrExecutableDataModified
	InstrErrExecutableLamportChange
	InstrErrExecutableAccountNotRentExempt
	InstrErrUnsupportedProgramId
	InstrErrCallDepth
	InstrErrMissingAccount
	InstrErrReentrancyNotAllowed
	InstrErrMaxSeedLengthExceeded
	InstrErrInvalidSeeds
	InstrErrInvalidRealloc
	InstrErrComputationalBudgetExceeded
	InstrErrPrivilegeEscalation
	InstrErrProgramEnvironmentSetupFailure
	InstrErrProgramFailedToComplete
	InstrErrProgramFailedToCompile
	InstrErrImmutable
	InstrErrIncorrectAuthority
	InstrErrBorshIoError
	InstrErrAccountNotRentExempt
	InstrErrInvalidAccountOwner
	InstrErrArithmeticOverflow
	InstrErrUnsupportedSysvar
	InstrErrIllegalOwner
	InstrErrMaxAccountsDataAllocationsExceeded
	InstrErrMaxAccountsExceeded
	InstrErrMaxInstructionTraceLengthExceeded
	InstrErrBuiltinProgramsMustConsumeComputeUnits
)

// String returns the Rust variant name.
func (t InstructionErrorType) String() string {
	if int(t) < len(instructionErrorNames) {
		return instructionErrorNames[t]
	}
	return fmt.Sprintf("InstructionErrorType(%d)", uint32(t))
}

// InstructionError is the reason an instruction failed. Port of the upstream
// solana_instruction_error::InstructionError enum.
type InstructionError struct {
	Type InstructionErrorType

	// Custom is the program-defined error code; meaningful only when Type
	// is InstrErrCustom.
	Custom uint32

	// BorshIoMessage holds the string payload that BorshIoError carried in
	// SDKs before instruction-error 2.2.0 removed it. It is populated only
	// when decoding legacy JSON of the form {"BorshIoError":"..."}; current
	// encoders never produce it.
	BorshIoMessage string
}

// Error returns the upstream Rust Display message.
func (e InstructionError) Error() string {
	if e.Type == InstrErrCustom {
		return fmt.Sprintf("custom program error: %#x", e.Custom)
	}
	if int(e.Type) < len(instructionErrorDisplays) {
		return instructionErrorDisplays[e.Type]
	}
	return e.Type.String()
}

// DebugString returns the upstream Rust {:?} rendering, e.g. "Custom(42)".
func (e InstructionError) DebugString() string {
	if e.Type == InstrErrCustom {
		return fmt.Sprintf("Custom(%d)", e.Custom)
	}
	return e.Type.String()
}

func (e InstructionError) MarshalWithEncoder(encoder *bin.Encoder) error {
	if err := encoder.WriteUint32(uint32(e.Type), binary.LittleEndian); err != nil {
		return err
	}
	if e.Type == InstrErrCustom {
		return encoder.WriteUint32(e.Custom, binary.LittleEndian)
	}
	return nil
}

func (e *InstructionError) UnmarshalWithDecoder(decoder *bin.Decoder) error {
	tag, err := decoder.ReadUint32(binary.LittleEndian)
	if err != nil {
		return err
	}
	if int(tag) >= len(instructionErrorNames) {
		return fmt.Errorf("instruction error: unknown variant tag %d", tag)
	}
	e.Type = InstructionErrorType(tag)
	if e.Type == InstrErrCustom {
		e.Custom, err = decoder.ReadUint32(binary.LittleEndian)
		return err
	}
	return nil
}

// MarshalJSON produces the serde JSON form: variant-name string for unit
// variants, {"Custom": n} for Custom.
func (e InstructionError) MarshalJSON() ([]byte, error) {
	if e.Type == InstrErrCustom {
		return json.Marshal(map[string]uint32{"Custom": e.Custom})
	}
	return json.Marshal(e.Type.String())
}

// UnmarshalJSON accepts the serde JSON form. For BorshIoError it also
// accepts the pre-2.2.0 {"BorshIoError":"..."} object, preserving the
// message in BorshIoMessage.
func (e *InstructionError) UnmarshalJSON(data []byte) error {
	*e = InstructionError{}
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		for i, n := range instructionErrorNames {
			if n == name {
				e.Type = InstructionErrorType(i)
				return nil
			}
		}
		return fmt.Errorf("instruction error: unknown variant %q", name)
	}
	var obj map[string]gojson.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil || len(obj) != 1 {
		return fmt.Errorf("instruction error: unrecognized JSON form %s", string(data))
	}
	if raw, ok := obj["Custom"]; ok {
		e.Type = InstrErrCustom
		return json.Unmarshal(raw, &e.Custom)
	}
	if raw, ok := obj["BorshIoError"]; ok {
		e.Type = InstrErrBorshIoError
		return json.Unmarshal(raw, &e.BorshIoMessage)
	}
	return fmt.Errorf("instruction error: unrecognized JSON form %s", string(data))
}

// TransactionError is the reason a transaction was rejected. Port of the
// upstream solana_transaction_error::TransactionError enum.
type TransactionError struct {
	Type TransactionErrorType

	// InstructionIndex is the index of the failing instruction; meaningful
	// only when Type is TxErrInstructionError or TxErrDuplicateInstruction.
	InstructionIndex uint8

	// InstructionError is the nested instruction failure; non-nil exactly
	// when Type is TxErrInstructionError.
	InstructionError *InstructionError

	// AccountIndex identifies the offending account; meaningful only when
	// Type is TxErrInsufficientFundsForRent or
	// TxErrProgramExecutionTemporarilyRestricted.
	AccountIndex uint8
}

// Error returns the upstream Rust Display message.
func (e *TransactionError) Error() string {
	switch e.Type {
	case TxErrInstructionError:
		inner := "nil"
		if e.InstructionError != nil {
			inner = e.InstructionError.Error()
		}
		return fmt.Sprintf("Error processing Instruction %d: %s", e.InstructionIndex, inner)
	case TxErrDuplicateInstruction:
		return fmt.Sprintf("Transaction contains a duplicate instruction (%d) that is not allowed", e.InstructionIndex)
	case TxErrInsufficientFundsForRent:
		return fmt.Sprintf("Transaction results in an account (%d) with insufficient funds for rent", e.AccountIndex)
	case TxErrProgramExecutionTemporarilyRestricted:
		return fmt.Sprintf("Execution of the program referenced by account at index %d is temporarily restricted.", e.AccountIndex)
	}
	if int(e.Type) < len(transactionErrorDisplays) {
		return transactionErrorDisplays[e.Type]
	}
	return e.Type.String()
}

// DebugString returns the upstream Rust {:?} rendering, e.g.
// "InstructionError(0, Custom(42))".
func (e *TransactionError) DebugString() string {
	switch e.Type {
	case TxErrInstructionError:
		inner := "nil"
		if e.InstructionError != nil {
			inner = e.InstructionError.DebugString()
		}
		return fmt.Sprintf("InstructionError(%d, %s)", e.InstructionIndex, inner)
	case TxErrDuplicateInstruction:
		return fmt.Sprintf("DuplicateInstruction(%d)", e.InstructionIndex)
	case TxErrInsufficientFundsForRent:
		return fmt.Sprintf("InsufficientFundsForRent { account_index: %d }", e.AccountIndex)
	case TxErrProgramExecutionTemporarilyRestricted:
		return fmt.Sprintf("ProgramExecutionTemporarilyRestricted { account_index: %d }", e.AccountIndex)
	}
	return e.Type.String()
}

func (e TransactionError) MarshalWithEncoder(encoder *bin.Encoder) error {
	if err := encoder.WriteUint32(uint32(e.Type), binary.LittleEndian); err != nil {
		return err
	}
	switch e.Type {
	case TxErrInstructionError:
		if err := encoder.WriteUint8(e.InstructionIndex); err != nil {
			return err
		}
		inner := InstructionError{}
		if e.InstructionError != nil {
			inner = *e.InstructionError
		}
		return inner.MarshalWithEncoder(encoder)
	case TxErrDuplicateInstruction:
		return encoder.WriteUint8(e.InstructionIndex)
	case TxErrInsufficientFundsForRent, TxErrProgramExecutionTemporarilyRestricted:
		return encoder.WriteUint8(e.AccountIndex)
	}
	return nil
}

func (e *TransactionError) UnmarshalWithDecoder(decoder *bin.Decoder) error {
	tag, err := decoder.ReadUint32(binary.LittleEndian)
	if err != nil {
		return err
	}
	if int(tag) >= len(transactionErrorNames) {
		return fmt.Errorf("transaction error: unknown variant tag %d", tag)
	}
	*e = TransactionError{Type: TransactionErrorType(tag)}
	switch e.Type {
	case TxErrInstructionError:
		if e.InstructionIndex, err = decoder.ReadUint8(); err != nil {
			return err
		}
		var inner InstructionError
		if err := inner.UnmarshalWithDecoder(decoder); err != nil {
			return err
		}
		e.InstructionError = &inner
	case TxErrDuplicateInstruction:
		e.InstructionIndex, err = decoder.ReadUint8()
	case TxErrInsufficientFundsForRent, TxErrProgramExecutionTemporarilyRestricted:
		e.AccountIndex, err = decoder.ReadUint8()
	}
	return err
}

// MarshalBinary bincode-encodes the error, matching the Rust serde encoding
// of solana_transaction_error::TransactionError.
func (e TransactionError) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := e.MarshalWithEncoder(bin.NewBinEncoder(buf)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary decodes a bincode-encoded transaction error.
func (e *TransactionError) UnmarshalBinary(data []byte) error {
	return e.UnmarshalWithDecoder(bin.NewBinDecoder(data))
}

// DecodeTransactionError decodes a bincode-encoded
// solana_transaction_error::TransactionError, rejecting trailing bytes.
func DecodeTransactionError(data []byte) (*TransactionError, error) {
	var e TransactionError
	dec := bin.NewBinDecoder(data)
	if err := e.UnmarshalWithDecoder(dec); err != nil {
		return nil, fmt.Errorf("failed to decode transaction error: %w", err)
	}
	if dec.Remaining() != 0 {
		return nil, fmt.Errorf("transaction error: %d trailing byte(s)", dec.Remaining())
	}
	return &e, nil
}

// MarshalJSON produces the serde JSON form: variant-name string for unit
// variants, single-key objects for payload variants (the shape JSON-RPC
// returns in transaction meta).
func (e TransactionError) MarshalJSON() ([]byte, error) {
	switch e.Type {
	case TxErrInstructionError:
		inner := InstructionError{}
		if e.InstructionError != nil {
			inner = *e.InstructionError
		}
		innerJSON, err := inner.MarshalJSON()
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string][2]gojson.RawMessage{
			"InstructionError": {gojson.RawMessage(fmt.Sprintf("%d", e.InstructionIndex)), innerJSON},
		})
	case TxErrDuplicateInstruction:
		return json.Marshal(map[string]uint8{"DuplicateInstruction": e.InstructionIndex})
	case TxErrInsufficientFundsForRent:
		return json.Marshal(map[string]map[string]uint8{
			"InsufficientFundsForRent": {"account_index": e.AccountIndex},
		})
	case TxErrProgramExecutionTemporarilyRestricted:
		return json.Marshal(map[string]map[string]uint8{
			"ProgramExecutionTemporarilyRestricted": {"account_index": e.AccountIndex},
		})
	}
	return json.Marshal(e.Type.String())
}

// UnmarshalJSON accepts the serde JSON form.
func (e *TransactionError) UnmarshalJSON(data []byte) error {
	*e = TransactionError{}
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		for i, n := range transactionErrorNames {
			if n == name {
				e.Type = TransactionErrorType(i)
				return nil
			}
		}
		return fmt.Errorf("transaction error: unknown variant %q", name)
	}
	var obj map[string]gojson.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil || len(obj) != 1 {
		return fmt.Errorf("transaction error: unrecognized JSON form %s", string(data))
	}
	if raw, ok := obj["InstructionError"]; ok {
		var pair [2]gojson.RawMessage
		if err := json.Unmarshal(raw, &pair); err != nil {
			return fmt.Errorf("transaction error: bad InstructionError payload: %w", err)
		}
		e.Type = TxErrInstructionError
		if err := json.Unmarshal(pair[0], &e.InstructionIndex); err != nil {
			return err
		}
		var inner InstructionError
		if err := inner.UnmarshalJSON(pair[1]); err != nil {
			return err
		}
		e.InstructionError = &inner
		return nil
	}
	if raw, ok := obj["DuplicateInstruction"]; ok {
		e.Type = TxErrDuplicateInstruction
		return json.Unmarshal(raw, &e.InstructionIndex)
	}
	for name, typ := range map[string]TransactionErrorType{
		"InsufficientFundsForRent":              TxErrInsufficientFundsForRent,
		"ProgramExecutionTemporarilyRestricted": TxErrProgramExecutionTemporarilyRestricted,
	} {
		if raw, ok := obj[name]; ok {
			var payload struct {
				AccountIndex uint8 `json:"account_index"`
			}
			if err := json.Unmarshal(raw, &payload); err != nil {
				return err
			}
			e.Type = typ
			e.AccountIndex = payload.AccountIndex
			return nil
		}
	}
	return fmt.Errorf("transaction error: unrecognized JSON form %s", string(data))
}

// Variant name and Display-string tables, derived from
// solana-instruction-error 2.2.0 and solana-transaction-error 3.1.0
// (pinned by testdata/transaction_error_fixtures.json). Payload-carrying
// variants have an empty Displays entry; their message is composed in
// Error().

var instructionErrorNames = [...]string{
	"GenericError",
	"InvalidArgument",
	"InvalidInstructionData",
	"InvalidAccountData",
	"AccountDataTooSmall",
	"InsufficientFunds",
	"IncorrectProgramId",
	"MissingRequiredSignature",
	"AccountAlreadyInitialized",
	"UninitializedAccount",
	"UnbalancedInstruction",
	"ModifiedProgramId",
	"ExternalAccountLamportSpend",
	"ExternalAccountDataModified",
	"ReadonlyLamportChange",
	"ReadonlyDataModified",
	"DuplicateAccountIndex",
	"ExecutableModified",
	"RentEpochModified",
	"NotEnoughAccountKeys",
	"AccountDataSizeChanged",
	"AccountNotExecutable",
	"AccountBorrowFailed",
	"AccountBorrowOutstanding",
	"DuplicateAccountOutOfSync",
	"Custom",
	"InvalidError",
	"ExecutableDataModified",
	"ExecutableLamportChange",
	"ExecutableAccountNotRentExempt",
	"UnsupportedProgramId",
	"CallDepth",
	"MissingAccount",
	"ReentrancyNotAllowed",
	"MaxSeedLengthExceeded",
	"InvalidSeeds",
	"InvalidRealloc",
	"ComputationalBudgetExceeded",
	"PrivilegeEscalation",
	"ProgramEnvironmentSetupFailure",
	"ProgramFailedToComplete",
	"ProgramFailedToCompile",
	"Immutable",
	"IncorrectAuthority",
	"BorshIoError",
	"AccountNotRentExempt",
	"InvalidAccountOwner",
	"ArithmeticOverflow",
	"UnsupportedSysvar",
	"IllegalOwner",
	"MaxAccountsDataAllocationsExceeded",
	"MaxAccountsExceeded",
	"MaxInstructionTraceLengthExceeded",
	"BuiltinProgramsMustConsumeComputeUnits",
}

var instructionErrorDisplays = [...]string{
	"generic instruction error",
	"invalid program argument",
	"invalid instruction data",
	"invalid account data for instruction",
	"account data too small for instruction",
	"insufficient funds for instruction",
	"incorrect program id for instruction",
	"missing required signature for instruction",
	"instruction requires an uninitialized account",
	"instruction requires an initialized account",
	"sum of account balances before and after instruction do not match",
	"instruction illegally modified the program id of an account",
	"instruction spent from the balance of an account it does not own",
	"instruction modified data of an account it does not own",
	"instruction changed the balance of a read-only account",
	"instruction modified data of a read-only account",
	"instruction contains duplicate accounts",
	"instruction changed executable bit of an account",
	"instruction modified rent epoch of an account",
	"insufficient account keys for instruction",
	"program other than the account's owner changed the size of the account data",
	"instruction expected an executable account",
	"instruction tries to borrow reference for an account which is already borrowed",
	"instruction left account with an outstanding borrowed reference",
	"instruction modifications of multiply-passed account differ",
	"",
	"program returned invalid error code",
	"instruction changed executable accounts data",
	"instruction changed the balance of an executable account",
	"executable accounts must be rent exempt",
	"Unsupported program id",
	"Cross-program invocation call depth too deep",
	"An account required by the instruction is missing",
	"Cross-program invocation reentrancy not allowed for this instruction",
	"Length of the seed is too long for address generation",
	"Provided seeds do not result in a valid address",
	"Failed to reallocate account data",
	"Computational budget exceeded",
	"Cross-program invocation with unauthorized signer or writable account",
	"Failed to create program execution environment",
	"Program failed to complete",
	"Program failed to compile",
	"Account is immutable",
	"Incorrect authority provided",
	"Failed to serialize or deserialize account data",
	"An account does not have enough lamports to be rent-exempt",
	"Invalid account owner",
	"Program arithmetic overflowed",
	"Unsupported sysvar",
	"Provided owner is not allowed",
	"Accounts data allocations exceeded the maximum allowed per transaction",
	"Max accounts exceeded",
	"Max instruction trace length exceeded",
	"Builtin programs must consume compute units",
}

var transactionErrorNames = [...]string{
	"AccountInUse",
	"AccountLoadedTwice",
	"AccountNotFound",
	"ProgramAccountNotFound",
	"InsufficientFundsForFee",
	"InvalidAccountForFee",
	"AlreadyProcessed",
	"BlockhashNotFound",
	"InstructionError",
	"CallChainTooDeep",
	"MissingSignatureForFee",
	"InvalidAccountIndex",
	"SignatureFailure",
	"InvalidProgramForExecution",
	"SanitizeFailure",
	"ClusterMaintenance",
	"AccountBorrowOutstanding",
	"WouldExceedMaxBlockCostLimit",
	"UnsupportedVersion",
	"InvalidWritableAccount",
	"WouldExceedMaxAccountCostLimit",
	"WouldExceedAccountDataBlockLimit",
	"TooManyAccountLocks",
	"AddressLookupTableNotFound",
	"InvalidAddressLookupTableOwner",
	"InvalidAddressLookupTableData",
	"InvalidAddressLookupTableIndex",
	"InvalidRentPayingAccount",
	"WouldExceedMaxVoteCostLimit",
	"WouldExceedAccountDataTotalLimit",
	"DuplicateInstruction",
	"InsufficientFundsForRent",
	"MaxLoadedAccountsDataSizeExceeded",
	"InvalidLoadedAccountsDataSizeLimit",
	"ResanitizationNeeded",
	"ProgramExecutionTemporarilyRestricted",
	"UnbalancedTransaction",
	"ProgramCacheHitMaxLimit",
	"CommitCancelled",
}

var transactionErrorDisplays = [...]string{
	"Account in use",
	"Account loaded twice",
	"Attempt to debit an account but found no record of a prior credit.",
	"Attempt to load a program that does not exist",
	"Insufficient funds for fee",
	"This account may not be used to pay transaction fees",
	"This transaction has already been processed",
	"Blockhash not found",
	"",
	"Loader call chain is too deep",
	"Transaction requires a fee but has no signature present",
	"Transaction contains an invalid account reference",
	"Transaction did not pass signature verification",
	"This program may not be used for executing instructions",
	"Transaction failed to sanitize accounts offsets correctly",
	"Transactions are currently disabled due to cluster maintenance",
	"Transaction processing left an account with an outstanding borrowed reference",
	"Transaction would exceed max Block Cost Limit",
	"Transaction version is unsupported",
	"Transaction loads a writable account that cannot be written",
	"Transaction would exceed max account limit within the block",
	"Transaction would exceed account data limit within the block",
	"Transaction locked too many accounts",
	"Transaction loads an address table account that doesn't exist",
	"Transaction loads an address table account with an invalid owner",
	"Transaction loads an address table account with invalid data",
	"Transaction address table lookup uses an invalid index",
	"Transaction leaves an account with a lower balance than rent-exempt minimum",
	"Transaction would exceed max Vote Cost Limit",
	"Transaction would exceed total account data limit",
	"",
	"",
	"Transaction exceeded max loaded accounts data size cap",
	"LoadedAccountsDataSizeLimit set for transaction must be greater than 0.",
	"ResanitizationNeeded",
	"",
	"Sum of account balances before and after transaction do not match",
	"Program cache hit max limit",
	"CommitCancelled",
}
