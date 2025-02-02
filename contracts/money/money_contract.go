package contracts

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/gas"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
)

const (
	contractOwnerKey = "o"
)

func _deploy(_ interface{}, isUpdate bool) {
	if isUpdate {
		return
	}
	var ctx = storage.GetContext()
	var owner = runtime.GetScriptContainer().Sender

	storage.Put(ctx, contractOwnerKey, owner)
}

// GetBalance allows the host to get the contract balance
func GetBalance() int {
	var contractHash = runtime.GetExecutingScriptHash()
	return gas.BalanceOf(contractHash)
}

func getOwner() interop.Hash160 {
	var ctx = storage.GetReadOnlyContext()
	var owner = storage.Get(ctx, contractOwnerKey)

	if owner == nil {
		panic("Owner not set")
	}

	ownerHash, ok := owner.(interop.Hash160)
	if !ok {
		panic("Stored owner is not a valid Hash160")
	}

	return ownerHash
}

// HostWithdrawal allows host to claim funds from played game
func HostWithdrawal(amount int) bool {
	var contractHash = runtime.GetExecutingScriptHash()
	var ownerHash = getOwner()

	if !gas.Transfer(contractHash, ownerHash, amount, nil) {
		runtime.Log("Failed to withdraw tokens")
		return false
	}

	return true
}

// Deposit transfers tokens to the contract balance from wallet
func Deposit(wallet interop.Hash160, amount int) bool {
	var contractHash = runtime.GetExecutingScriptHash()
	if !gas.Transfer(wallet, contractHash, amount, nil) {
		runtime.Log("Failed to deposit tokens")
		return false
	}

	runtime.Log("Successfully deposited tokens from wallet")
	return true
}

// RewardPlayer transfers tokens from the contract balance to the player's wallet
func RewardPlayer(wallet interop.Hash160, amount int) bool {
	var contractHash = runtime.GetExecutingScriptHash()
	if !gas.Transfer(contractHash, wallet, amount, nil) {
		runtime.Log("Failed to transfer tokens to player")
		return false
	}

	runtime.Log("Successfully transferred tokens to player")
	return true
}

// Transfer tokens between wallets
func Transfer(from interop.Hash160, to interop.Hash160, amount int) bool {
	if !gas.Transfer(from, to, amount, nil) {
		runtime.Log("Failed to transfer tokens")
		return false
	}

	runtime.Log("Successfully transferred tokens")
	return true
}
