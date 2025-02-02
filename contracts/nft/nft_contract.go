package nft

import (
	"fmt"
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/iterator"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/crypto"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/gas"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/std"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
	"github.com/nspcc-dev/neo-go/pkg/interop/util"
)

// CONSTANTS

const (
	balancePrefix  = "b"
	accountPrefix  = "a"
	tokenPrefix    = "t"
	totalSupplyKey = "s"

	symbol        = "QUESTIONS"
	decimals      = 0
	questionPrice = 10_0000_0000
	linkPrice     = 5_0000_0000
)

// STRUCTS

type QuestionNFT struct {
	ID         []byte
	Owner      interop.Hash160
	Question   string
	SourceLink string
	PrevOwners int
}

func _deploy(_ interface{}, isUpdate bool) {
	if isUpdate {
		return
	}
}

// GLOBAL PRIVATE METHODS FOR NFT

func makeAccountKey(accountId interop.Hash160) []byte {
	return append([]byte(accountPrefix), accountId...)
}

func makeBalanceKey(balanceId interop.Hash160) []byte {
	return append([]byte(balancePrefix), balanceId...)
}

func makeTokenKey(tokenID []byte) []byte {
	return append([]byte(tokenPrefix), tokenID...)
}

func getNFT(ctx storage.Context, token []byte) QuestionNFT {
	var key = makeTokenKey(token)
	var nft = storage.Get(ctx, key)
	if nft == nil {
		panic(fmt.Sprintf("Token with key:%s not found", key))
	}

	return std.Deserialize(nft.([]byte)).(QuestionNFT)
}

func setNFT(ctx storage.Context, token []byte, nft QuestionNFT) {
	var key = makeTokenKey(token)
	var value = std.Serialize(nft)
	storage.Put(ctx, key, value)
}

func nftExists(ctx storage.Context, token []byte) bool {
	return storage.Get(ctx, makeTokenKey(token)) != nil
}

func getBalanceOf(ctx storage.Context, balanceKey []byte) int {
	var balance = storage.Get(ctx, balanceKey)
	if balance != nil {
		return balance.(int)
	}

	return 0
}

func addToBalance(ctx storage.Context, balanceId interop.Hash160, amount int) {
	var key = makeBalanceKey(balanceId)
	var balance = getBalanceOf(ctx, key) + amount
	if balance > 0 {
		storage.Put(ctx, key, balance)
	} else {
		storage.Delete(ctx, key)
	}
}

func addToken(ctx storage.Context, tokenId interop.Hash160, token []byte) {
	var key = makeAccountKey(tokenId)
	storage.Put(ctx, append(key, token...), token)
}

func removeToken(ctx storage.Context, tokenId interop.Hash160, token []byte) {
	var key = makeAccountKey(tokenId)
	storage.Delete(ctx, append(key, token...))
}

// MAIN PUBLIC METHODS

func Symbol() string {
	return symbol
}

func Decimals() int {
	return decimals
}

func TotalSupply() int {
	return storage.Get(storage.GetReadOnlyContext(), totalSupplyKey).(int)
}

func BalanceOf(holder interop.Hash160) int {
	if len(holder) != 20 {
		panic("bad owner address")
	}

	return getBalanceOf(storage.GetReadOnlyContext(), makeBalanceKey(holder))
}

func OwnerOf(token []byte) interop.Hash160 {
	return getNFT(storage.GetReadOnlyContext(), token).Owner
}

func Properties(token []byte) map[string]string {
	var ctx = storage.GetReadOnlyContext()
	var nft = getNFT(ctx, token)

	var result = map[string]string{
		"id":         string(nft.ID),
		"owner":      string(nft.Owner),
		"question":   nft.Question,
		"sourceLink": nft.SourceLink,
		"prevOwners": std.Itoa10(nft.PrevOwners),
	}

	return result
}

func Tokens() iterator.Iterator {
	var ctx = storage.GetReadOnlyContext()
	var key = []byte(tokenPrefix)

	return storage.Find(ctx, key, storage.RemovePrefix|storage.KeysOnly)
}

func TokensList() []string {
	var ctx = storage.GetReadOnlyContext()
	var key = []byte(tokenPrefix)
	var iter = storage.Find(ctx, key, storage.RemovePrefix|storage.KeysOnly)

	var keys []string
	for iterator.Next(iter) {
		var k = iterator.Value(iter)
		keys = append(keys, k.(string))
	}

	return keys
}

func TokensOf(wallet interop.Hash160) iterator.Iterator {
	if len(wallet) != 20 {
		panic(fmt.Sprintf("Owner wallet:%s is not valid", wallet))
	}
	var ctx = storage.GetReadOnlyContext()
	var key = makeAccountKey(wallet)

	return storage.Find(ctx, key, storage.ValuesOnly)
}

func TokensOfList(wallet interop.Hash160) [][]byte {
	if len(wallet) != 20 {
		panic(fmt.Sprintf("Owner wallet:%s is not valid", wallet))
	}
	var ctx = storage.GetReadOnlyContext()
	var key = makeAccountKey(wallet)

	var result [][]byte
	var iter = storage.Find(ctx, key, storage.ValuesOnly)
	for iterator.Next(iter) {
		result = append(result, iterator.Value(iter).([]byte))
	}

	return result
}

func Transfer(to interop.Hash160, token []byte) bool {
	if len(to) != 20 {
		panic(fmt.Sprintf("To wallet:%s is not valid", to))
	}
	var ctx = storage.GetContext()
	var nft = getNFT(ctx, token)
	var from = nft.Owner

	if !runtime.CheckWitness(from) {
		runtime.Log("Unauthorized transfer attempt")
		return false
	}

	if !from.Equals(to) {
		nft.Owner = to
		nft.PrevOwners++
		setNFT(ctx, token, nft)

		addToBalance(ctx, from, -1)
		removeToken(ctx, from, token)

		addToBalance(ctx, to, 1)
		addToken(ctx, to, token)
	}

	runtime.Notify("Transfer", from, to, 1, token)
	return true
}

func Burn(token []byte) bool {
	var ctx = storage.GetContext()
	var nft = getNFT(ctx, token)

	storage.Delete(ctx, makeTokenKey(nft.ID))

	var total = storage.Get(ctx, totalSupplyKey).(int) - 1
	storage.Put(ctx, totalSupplyKey, total)

	runtime.Notify("Burn", token)
	return true
}

// data format '{"question":"What is Neo?", "data":"link<optional>"}'
func parseData(input any) (string, string) {
	var data = std.JSONDeserialize(input.([]byte)).(map[string]any)

	var question = ""
	if q, exists := data["question"]; exists {
		question, _ = q.(string)
	} else {
		panic("Missing or invalid 'question' field")
	}

	var sourceLink = ""
	if l, exists := data["link"]; exists {
		sourceLink, _ = l.(string)
	}

	return question, sourceLink
}

func OnNEP17Payment(from interop.Hash160, amount int, data any) {
	defer func() {
		if r := recover(); r != nil {
			runtime.Log(r.(string))
			util.Abort()
		}
	}()

	var callingHash = runtime.GetCallingScriptHash()
	if !callingHash.Equals(gas.Hash) {
		panic("Only GAS is accepted")
	}

	var question, sourceLink = parseData(data)

	var price = questionPrice

	if sourceLink != "" {
		price += linkPrice
	}

	if amount < price {
		panic("Insufficient GAS for minting NFT")
	}

	var ctx = storage.GetContext()
	var tokenID = crypto.Sha256([]byte(question))
	if nftExists(ctx, tokenID) {
		panic("Token already exists")
	}

	var nft = QuestionNFT{
		ID:         tokenID,
		Owner:      from,
		Question:   question,
		SourceLink: sourceLink,
		PrevOwners: 0,
	}

	setNFT(ctx, tokenID, nft)
	addToBalance(ctx, from, 1)
	addToken(ctx, from, tokenID)

	var total = storage.Get(ctx, totalSupplyKey).(int) + 1
	storage.Put(ctx, totalSupplyKey, total)

	runtime.Notify("Create", from, tokenID)
}
