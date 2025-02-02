package room

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/std"
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
	"sort"
)

// CONSTANTS

const (
	StatusWaiting   = "waiting"   // Waiting for the game to start, players are joining
	StatusGaming    = "gaming"    // In this phase, players are ready, but the question hasn't been asked yet
	StatusAnswering = "answering" // Phase when the round has started and the question has been asked, players can submit answers
	StatusVoting    = "voting"    // Voting phase, where players select the best answer from the options
	StatusFinished  = "finished"  // Game is finished, and results have been determined
)

const (
	moneyContractKey = "m"
	nftContractKey   = "n"
)

const (
	createRoomCommission    = 4_0000_0000
	joinRoomCommission      = 2_0000_0000
	sendAnswerCommission    = 1_0000_0000
	gamePrizePoolCommission = 2000_0000
	userCommission          = 7000_0000
	oneGas                  = 1_0000_0000
)

// STRUCTS

type Room struct {
	Id                string
	Host              interop.Hash160
	Status            string
	GamePrizePool     int
	RoundPrizePool    int
	RoundWinnersCount int
	GameWinnersCount  int
	Players           []Player
	Rounds            []Round
}

type Round struct {
	TokenId  []byte // NFT token id, checking for uniqueness of questions
	Question string
	Answers  []Answer
}

type Answer struct {
	Wallet  interop.Hash160
	Content string
	Votes   []interop.Hash160 // Wallets who voted for answer
}

type Player struct {
	Wallet          interop.Hash160
	RoundsWon       int
	IsReady         bool
	IsVotedToFinish bool
	isActive        bool
}

// --data '{"m": "0xabc123...", "n": "0xdef456..."}'
func _deploy(data interface{}, isUpdate bool) {
	if isUpdate {
		return
	}

	var args = std.JSONDeserialize(data.([]byte)).(map[string]any)

	var moneyContractHash interop.Hash160
	if hash, exist := args[moneyContractKey]; exist && len(hash.(interop.Hash160)) == interop.Hash160Len {
		moneyContractHash = hash.(interop.Hash160)
	} else {
		panic("Missing or invalid 'm' field - money contract hash or invalid hash contract")
	}

	var nftContractHash interop.Hash160
	if hash, exist := args[nftContractKey]; exist && len(hash.(interop.Hash160)) == interop.Hash160Len {
		nftContractHash = hash.(interop.Hash160)
	} else {
		panic("Missing or invalid 'n' field - nft contract hash or invalid hash contract")
	}

	var ctx = storage.GetContext()
	storage.Put(ctx, moneyContractKey, moneyContractHash)
	storage.Put(ctx, nftContractKey, nftContractHash)
}

// GLOBAL PRIVATE METHODS FOR ROOM

func getSender() interop.Hash160 {
	return runtime.GetScriptContainer().Sender
}

// Function to set room in storage with serialize
func setRoom(ctx storage.Context, room *Room) {
	var serializedRoom = std.Serialize(room)
	storage.Put(ctx, "room:"+room.Id, serializedRoom)
}

// Function to get room from storage with deserialize
func getRoom(ctx storage.Context, roomId string) Room {
	var roomData = storage.Get(ctx, "room:"+roomId)

	if roomData == nil {
		panic(fmt.Sprintf("Room with roomId=%s not found", roomId))
	}

	var room = std.Deserialize(roomData.([]byte)).(Room)
	return room
}

func getNftContractHash(ctx storage.Context) interop.Hash160 {
	return storage.Get(ctx, nftContractKey).(interop.Hash160)
}

func getMoneyContractHash(ctx storage.Context) interop.Hash160 {
	return storage.Get(ctx, moneyContractKey).(interop.Hash160)
}

// Function to send message to players, event is recorded in blockchain
// Could be read through getapplicationlog or RPC call
func sendMessageToPlayers(notificationName string, message string) {
	runtime.Notify(notificationName, message)
}

func isPlayerDeactivate(room Room, wallet interop.Hash160) bool {
	for _, player := range room.Players {
		if player.Wallet.Equals(wallet) {
			return !player.isActive
		}
	}

	return true // Player was not found
}

func sendRewardRoundWinners(ctx storage.Context, room *Room, wonAnswers []Answer) {
	var pool = room.RoundPrizePool
	room.GamePrizePool += pool * gamePrizePoolCommission / oneGas // Increase GamePrizePool by 20% of the RoundPrizePool
	pool -= pool * gamePrizePoolCommission / oneGas

	/*
		Total votes = 9 4 1 1 = 15
		Weights to send reward = 9/15 4/15 1/15 1/15, all * userCommission
		because 1 - userCommission is commission of host for game
	*/
	var totalVotes = 0
	for _, answer := range wonAnswers {
		totalVotes += len(answer.Votes)
	}

	if totalVotes == 0 {
		runtime.Log("No votes, skipping reward distribution")
		return
	}

	for _, answer := range wonAnswers {
		// reward = pool * weight * userCommission / oneGas
		var reward = (pool * (len(answer.Votes) / totalVotes) * userCommission) / oneGas

		var result = contract.Call(getMoneyContractHash(ctx), "RewardPlayer", contract.All, answer.Wallet, reward).(bool)
		sendMessageToPlayers(
			"RewardResult",
			fmt.Sprintf("player:%s, rewarded:%t", string(answer.Wallet), result))
	}

	// After receiving all the rewards, the remaining part of the pool remains with the host.
	// Reward for host = pool - pool * 70% (and the integer precision inaccuracy)
	room.RoundPrizePool = 0
	setRoom(ctx, room)
}

func sendRewardGameWinners(ctx storage.Context, room *Room, wonPlayers []Player) {
	var pool = room.GamePrizePool
	var totalRounds = len(room.Rounds)

	if totalRounds == 0 {
		runtime.Log("No rounds played, skipping reward distribution")
		return
	}

	for _, player := range wonPlayers {
		// reward = pool * weight * userCommission / oneGas
		var reward = (pool * (player.RoundsWon / totalRounds) * userCommission) / oneGas

		var result = contract.Call(getMoneyContractHash(ctx), "RewardPlayer", contract.All, player.Wallet, reward).(bool)
		sendMessageToPlayers(
			"RewardResult",
			fmt.Sprintf("player:%s, rewarded:%t", string(player.Wallet), result))
	}

	// After receiving all the rewards, the remaining part of the pool remains with the host.
	// Reward for host = pool - pool * 70% (and the integer precision inaccuracy)
	room.GamePrizePool = 0
	setRoom(ctx, room)
}

// MAIN METHODS TO PLAY IN GAME

func CreateRoom(RoundWinnersCount int, GameWinnersCount int) string {
	var ctx = storage.GetContext()
	var id = uuid.NewString()
	var host = getSender()

	var withdraw = contract.Call(getMoneyContractHash(ctx), "Deposit", contract.All, host, createRoomCommission).(bool)
	if !withdraw {
		panic("Host does not have enough tokens to create a room")
	}

	var room = Room{
		Id:                id,
		Host:              host,
		Status:            StatusWaiting,
		GamePrizePool:     createRoomCommission,
		RoundPrizePool:    0,
		RoundWinnersCount: RoundWinnersCount,
		GameWinnersCount:  GameWinnersCount,
		Players:           []Player{},
		Rounds:            []Round{},
	}

	setRoom(ctx, &room)
	return id
}

func JoinRoom(roomId string) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)
	var wallet = getSender()

	if room.Host.Equals(wallet) || room.Status != StatusWaiting {
		return false // Host can not be player, player cannot join started room
	}

	for _, player := range room.Players {
		if player.Wallet.Equals(wallet) {
			return false // Player already joined room
		}
	}

	var withdraw = contract.Call(getMoneyContractHash(ctx), "Deposit", contract.All, wallet, joinRoomCommission).(bool)
	if !withdraw {
		panic("Player does not have enough tokens to join in room")
	}
	room.GamePrizePool += joinRoomCommission

	var player = Player{
		Wallet:          wallet,
		RoundsWon:       0,
		IsReady:         false,
		IsVotedToFinish: false,
		isActive:        true,
	}

	room.Players = append(room.Players, player)
	setRoom(ctx, &room)
	return true
}

func ConfirmReadiness(roomId string) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)
	var wallet = getSender()

	for i, p := range room.Players {
		if p.Wallet.Equals(wallet) {
			if p.IsReady || !p.isActive {
				return false // Player is already ready, player must be active
			}
			room.Players[i].IsReady = true
			setRoom(ctx, &room)
			return true
		}
	}

	return false // Player not found in the room
}

func StartGame(roomId string) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)

	if !room.Host.Equals(getSender()) || room.Status != StatusWaiting || len(room.Players) <= room.RoundWinnersCount {
		return false // Only host can start game, room status must be waiting and players count must be > count winners
	}

	for _, player := range room.Players {
		if !player.IsReady {
			return false // If any player is not ready, the game can not be started
		}
	}

	room.Status = StatusGaming
	setRoom(ctx, &room)
	return true
}

func checkingForUniqueness(rounds []Round, tokenId []byte) bool {
	for _, round := range rounds {
		if bytes.Equal(round.TokenId, tokenId) {
			return false
		}
	}

	return true
}

func AskQuestion(roomId string, tokenId []byte) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)
	var wallet = getSender()

	if !room.Host.Equals(wallet) || room.Status != StatusGaming {
		return false // Only host can ask question, room status must be gaming
	}

	// Get token properties from nft contract
	var tokenProperties = contract.Call(getNftContractHash(ctx), "Properties", contract.All, tokenId).(map[string]string)
	if tokenProperties == nil || tokenProperties["owner"] != string(wallet) || checkingForUniqueness(room.Rounds, tokenId) {
		return false // NFT was not found, host is not the owner of question, round must contain unique questions
	}

	var question = tokenProperties["question"]
	var round = Round{
		TokenId:  tokenId,
		Question: question,
		Answers:  []Answer{},
	}
	room.Rounds = append(room.Rounds, round)
	room.Status = StatusAnswering

	sendMessageToPlayers("RoundQuestion", question)

	setRoom(ctx, &room)
	return true
}

func roomContainsPlayer(players []Player, wallet interop.Hash160) bool {
	for _, player := range players {
		if player.Wallet.Equals(wallet) {
			return true
		}
	}
	return false
}

func SendAnswer(roomId string, text string) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)
	var wallet = getSender()

	if !roomContainsPlayer(room.Players, wallet) || isPlayerDeactivate(room, wallet) || room.Status != StatusAnswering {
		return false // Only player can send content, player must be active, room status must be answering
	}

	var withdraw = contract.Call(getMoneyContractHash(ctx), "Deposit", contract.All, wallet, sendAnswerCommission).(bool)
	if !withdraw {
		panic("Player does not have enough tokens to send answer")
	}
	room.RoundPrizePool += sendAnswerCommission

	var round = room.Rounds[len(room.Rounds)-1]

	for _, answer := range round.Answers {
		if answer.Wallet.Equals(wallet) {
			return false // Player cannot send answer twice
		}
	}

	var answer = Answer{
		Wallet:  wallet,
		Content: text,
		Votes:   []interop.Hash160{},
	}

	round.Answers = append(round.Answers, answer)
	room.Rounds[len(room.Rounds)-1] = round
	setRoom(ctx, &room)
	return true
}

func deactivatingPlayers(rounds []Round, players []Player) []Player {
	var previous, current = rounds[len(rounds)-2], rounds[len(rounds)-1]
	for _, player := range players {
		var isActive = false
		for _, answer := range previous.Answers {
			if answer.Wallet.Equals(player.Wallet) {
				isActive = true
				break
			}
		}

		for _, answer := range current.Answers {
			if answer.Wallet.Equals(player.Wallet) {
				isActive = true
			}
		}
		player.isActive = isActive
	}

	return players
}

func EndQuestion(roomId string) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)

	if !room.Host.Equals(getSender()) || room.Status != StatusAnswering {
		return false // Only host can end question, room status must be answering
	}

	room.Status = StatusVoting

	var rounds = room.Rounds
	var round = rounds[len(rounds)-1]
	var result string
	for i, answer := range round.Answers {
		result += fmt.Sprintf("index:%d, player:%s, answer:%s\n", i, answer.Wallet, answer.Content)
	}
	sendMessageToPlayers("RoundAnswers", result)

	if len(rounds) > 1 {
		room.Players = deactivatingPlayers(rounds, room.Players)
	}

	setRoom(ctx, &room)
	return true
}

func VoteAnswer(roomId string, answerIdx int) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)
	var wallet = getSender()

	if room.Host.Equals(wallet) || isPlayerDeactivate(room, wallet) || room.Status != StatusVoting {
		return false // Only player can choose answer, player must be active, room status must be voting
	}

	var round = room.Rounds[len(room.Rounds)-1]
	if !(0 <= answerIdx && answerIdx < len(round.Answers)) || round.Answers[answerIdx].Wallet.Equals(wallet) {
		return false // answerIdx is incorrect and player cannot vote for himself
	}

	for _, votedWallet := range round.Answers[answerIdx].Votes {
		if votedWallet.Equals(wallet) {
			return false // Player cannot vote twice for one answer
		}
	}

	round.Answers[answerIdx].Votes = append(round.Answers[answerIdx].Votes, wallet)
	setRoom(ctx, &room)
	return true
}

func chooseWonAnswers(round Round, RoundWinnersCount int) []Answer {
	var wonAnswers []Answer
	var answers = round.Answers
	sort.Slice(answers, func(a, b int) bool {
		return len(answers[a].Votes) > len(answers[b].Votes)
	})

	if RoundWinnersCount > len(round.Answers) {
		return round.Answers
	}

	if RoundWinnersCount == 1 {
		return []Answer{answers[0]}
	}

	// Choose wonAnswers from sorted answers. If the current answer has the same number of votes as the previous one,
	// we add it to the wonAnswers list. We increase the number of wonAnswers if there are multiple answers with the same
	// number of votes, as in cases where there are 5 answers with equal votes and RoundWinnersCount is 3, all should be included.
	var lastVote = len(answers[0].Votes)
	wonAnswers = append(wonAnswers, answers[0])
	for i := 1; i < len(answers) && len(wonAnswers) < RoundWinnersCount; i++ {
		var currentVote = len(answers[i].Votes)
		if lastVote == currentVote {
			RoundWinnersCount++ // todo: Могут возникнуть проблемы, надо протестировать
		}
		lastVote = currentVote
		wonAnswers = append(wonAnswers, answers[i])
	}

	return wonAnswers
}

func GetRoundWinner(roomId string) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)

	if !room.Host.Equals(getSender()) || room.Status != StatusVoting {
		return false // Only host can get winner, room status must be voting
	}

	var round = room.Rounds[len(room.Rounds)-1]
	if len(round.Answers) == 0 {
		return false // Zero winners, because no answer
	}

	var wonAnswers = chooseWonAnswers(round, room.RoundWinnersCount)
	var players = room.Players
	for _, answer := range wonAnswers {
		for _, player := range players {
			if answer.Wallet.Equals(player.Wallet) {
				player.RoundsWon++
				break
			}
		}
	}
	room.Players = players

	var result string
	for i, answer := range wonAnswers {
		result += fmt.Sprintf("place:%d, winner:%s, votes:%s\n", i, answer.Wallet, answer.Votes)
	}

	sendMessageToPlayers("RoundWinners", result)

	sendRewardRoundWinners(ctx, &room, wonAnswers)

	room.Status = StatusGaming // Next game cycle available to AskQuestion
	setRoom(ctx, &room)
	return true
}

func VoteToFinishGame(roomId string) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)
	var wallet = getSender()

	if room.Host.Equals(wallet) {
		return false // Host cannot vote to finish game
	}

	var voted = 0
	var isFound = false
	for i, p := range room.Players {
		if p.Wallet.Equals(wallet) {
			if p.IsVotedToFinish {
				return false // Player has already voted to finish the game
			}
			room.Players[i].IsVotedToFinish = true
			isFound = true
		}

		if p.IsVotedToFinish {
			voted++ // Count voted players to finish the game
		}
	}

	if !isFound {
		return false // Player was not found in the room
	}

	var result = fmt.Sprintf("voted to finish game:%d, need votes:%d", voted, len(room.Players))
	sendMessageToPlayers("FinishVote", result)

	return automaticFinishGame(ctx, &room, voted)
}

func automaticFinishGame(ctx storage.Context, room *Room, voted int) bool {
	if voted != len(room.Players) {
		return false // All players must have voted to finish the game
	}

	return finishGame(ctx, room)
}

func ManuallyFinishGame(roomId string) bool {
	var ctx = storage.GetContext()
	var room = getRoom(ctx, roomId)

	if !room.Host.Equals(getSender()) || room.Status != StatusGaming {
		return false // Only host can finish game, room status must be gaming
	}

	return finishGame(ctx, &room)
}

func chooseWonPlayers(room *Room, GameWinnersCount int) []Player {
	var players = room.Players
	var winners []Player
	sort.Slice(players, func(a, b int) bool {
		return players[a].RoundsWon > players[b].RoundsWon
	})

	if GameWinnersCount > len(room.Players) {
		return room.Players
	}

	if GameWinnersCount == 1 {
		return []Player{players[0]}
	}

	var lastWinner = players[0].RoundsWon
	winners = append(winners, players[0])
	for i := 1; i < len(players) && len(winners) < GameWinnersCount; i++ {
		var currentVote = players[i].RoundsWon
		if lastWinner == currentVote {
			GameWinnersCount++ // todo: Могут возникнуть проблемы, надо протестировать
		}
		lastWinner = currentVote
		winners = append(winners, players[i])
	}

	return winners
}

func finishGame(ctx storage.Context, room *Room) bool {
	var winners = chooseWonPlayers(room, room.GameWinnersCount)

	var result = fmt.Sprintf("finish the game! count winners:%d\n", len(winners))
	for i, player := range winners {
		result += fmt.Sprintf("place:%d, player:%s, score:%d\n", i, player.Wallet, player.RoundsWon)
	}
	sendMessageToPlayers("FinishGame", result)

	sendRewardGameWinners(ctx, room, winners)
	// Host reward remains on the money_contract.go wallet, from which he can withdraw money to his personal wallet.

	room.Status = StatusFinished
	setRoom(ctx, room)
	return true
}
