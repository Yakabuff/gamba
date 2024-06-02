package main

import (
	"errors"
	"fmt"
	"log"
	"slices"
	"time"
)

func (a App) spawnGameLoop(roomId string) {
	log.Printf("spawning game loop. Room ID:%s", roomId)
	gameInstance := a.GameInstances[roomId]
	// Map {player1: [deck]}
	channel := a.Hub.roomChannels[roomId]
	for {
		gameInstance.roundNumber++
		for {
			timeout := time.After(1 * time.Second)
			select {
			case <-timeout:
				if !gameInstance.gameStarted {
					continue
				}
				log.Println("timeout!")
				// timeout: skip current user's turn
				// increment round num
				// notify next user that it's their turn
				// if turn 1, must not pass; if user doesn't play card, play 3 of diamonds for them
				// if 3 timeouts, play lowest card
				currUserTurn := a.getCurrentUserTurn(roomId, gameInstance.currentUserTurnIndex)
				currUserName := a.Hub.clients[currUserTurn].name
				gameInstance.turnNumber = gameInstance.turnNumber + 1
				gameInstance.numSkips = gameInstance.numSkips + 1
				fmt.Printf("NUM SKIPS %d \n", gameInstance.numSkips)
				if gameInstance.turnNumber == 1 {
					user := gameInstance.findThreeOfDiamonds()
					gameInstance.removeCardsFromHand([]Card{NewCard(Three, Diamonds)}, user)
					gameInstance.lastPlayed = []Card{NewCard(Three, Diamonds)}
					//if auto played card, update hand
					updateHandEvent := newUpdateHandEvent(a.Hub.clients[currUserTurn].name, currUserTurn, roomId, gameInstance.playerHands[currUserTurn], a.getUsernamesRoom(roomId), cardsToString(gameInstance.playerHands[currUserTurn]))
					a.notifyUser(updateHandEvent)
					gameInstance.numSkips = 0
				} else if gameInstance.numSkips == len(a.Hub.rooms[roomId]) {
					// p1 plays, p2 skips, p3 skips, p4 skips, p1 skips/times out -> automatically play lowest card in p1s hand
					// 4 skips required
					lowest := slices.Min(gameInstance.playerHands[currUserTurn])
					gameInstance.removeCardsFromHand([]Card{lowest}, currUserTurn)
					gameInstance.lastPlayed = []Card{lowest}
					//if auto played card, update hand
					updateHandEvent := newUpdateHandEvent(a.Hub.clients[currUserTurn].name, currUserTurn, roomId, gameInstance.playerHands[currUserTurn], a.getUsernamesRoom(roomId), cardsToString(gameInstance.playerHands[currUserTurn]))
					a.notifyUser(updateHandEvent)
					gameInstance.numSkips = 0
					// if player has no more cards notify win
					if len(gameInstance.playerHands[currUserTurn]) == 0 {
						// Update leaderboard
						_, ok := gameInstance.leaderBoard[currUserName]
						if ok {
							gameInstance.leaderBoard[currUserName] = gameInstance.leaderBoard[currUserName] + 1
						} else {
							gameInstance.leaderBoard[currUserName] = 1
						}
						gameFinishEvent := newGameFinishEvent(currUserName, "", roomId, gameInstance.turnNumber, a.getUsernamesRoom(roomId), gameInstance.leaderBoard)
						a.notifyRoomMembers(gameFinishEvent)
						time.Sleep(5 * time.Second)
						gameInstance.resetGameInstance()
						gameInstance.gameStarted = true
						currUserTurn := gameInstance.findThreeOfDiamonds()
						currUserTurnName := a.Hub.clients[currUserTurn].name
						gameInstance.currentUserTurnIndex = slices.Index(a.Hub.rooms[roomId], currUserTurn)
						for _, j := range a.Hub.rooms[roomId] {
							cardsToString := cardsToString(gameInstance.playerHands[j])
							usernames := a.getUsernamesRoom(roomId)
							gameStartEvent := newGameStartEvent("", j, roomId, currUserTurnName, 1, usernames, gameInstance.playerHands[j], cardsToString, gameInstance.currentUserTurnIndex, gameInstance.leaderBoard)
							a.notifyUser(gameStartEvent)
						}
						break
					}
				}
				if gameInstance.currentUserTurnIndex == 3 {
					gameInstance.currentUserTurnIndex = 0
				} else {
					gameInstance.currentUserTurnIndex++
				}
				nextUserTurn := a.getCurrentUserTurn(roomId, gameInstance.currentUserTurnIndex)
				cardsToString := cardsToString(gameInstance.lastPlayed)
				skipMessage := newSkipEvent(currUserName, "", roomId, nextUserTurn, gameInstance.turnNumber, gameInstance.lastPlayed, a.getUsernamesRoom(roomId), cardsToString, gameInstance.currentUserTurnIndex)
				a.notifyRoomMembers(skipMessage)
			case msg := <-channel:
				fmt.Println(msg)
				if !gameInstance.gameStarted && msg.OperationType == CONNECT {
					// connect: if player count < 4, notify room
					// if player count == 4, emit game start message
					// set game instance status to game started and current user's turn
					// current user is user with 3 of diamonds
					// notify room who's starting
					gameInstance.connect(msg.UserId)
					gameInstance.leaderBoard[msg.Username] = 0
					usernames := a.getUsernamesRoom(roomId)
					gameConnectEvent := newConnectEvent(msg.Username, roomId, "", usernames, gameInstance.leaderBoard)
					a.notifyRoomMembers(gameConnectEvent)
					log.Println(gameInstance.playerCount)
					if gameInstance.playerCount == 4 {
						gameInstance.gameStarted = true
						currUserTurn := gameInstance.findThreeOfDiamonds()
						currUserTurnName := a.Hub.clients[currUserTurn].name
						gameInstance.currentUserTurnIndex = slices.Index(a.Hub.rooms[roomId], currUserTurn)
						for _, j := range a.Hub.rooms[roomId] {
							cardsToString := cardsToString(gameInstance.playerHands[j])
							gameStartEvent := newGameStartEvent("", j, msg.RoomId, currUserTurnName, 1, usernames, gameInstance.playerHands[j], cardsToString, gameInstance.currentUserTurnIndex, gameInstance.leaderBoard)
							a.notifyUser(gameStartEvent)
						}
					}

				} else if msg.OperationType == DISCONNECT {
					// disconnect: remove user from game and server
					// notify room
					// Note: doesnt matter if user disconnects mid turn, let it timeout
					gameInstance.disconnect(msg.UserId)
					a.disconnectRoomMember(msg)
					usernames := a.getUsernamesRoom(roomId)
					disconnectEvent := newDisconnectEvent(msg.Username, msg.RoomId, "", usernames)
					a.notifyRoomMembers(disconnectEvent)
				} else if gameInstance.gameStarted && msg.OperationType == ACTION {
					// update hand of player
					// notify all players of action
					gameInstance.removeCardsFromHand(msg.Cards, msg.UserId)
					gameInstance.lastPlayed = msg.Cards
					gameInstance.numSkips = 0
					gameInstance.turnNumber++
					// find next user
					if gameInstance.currentUserTurnIndex == 3 {
						gameInstance.currentUserTurnIndex = 0
					} else {
						gameInstance.currentUserTurnIndex++
					}
					currUser := a.Hub.rooms[roomId][gameInstance.currentUserTurnIndex]
					actionEvent := newPlayEvent(msg.Username, roomId, currUser, gameInstance.turnNumber, msg.Cards, "", a.getUsernamesRoom(roomId), cardsToString(msg.Cards))
					a.notifyRoomMembers(actionEvent)
					cardsHandString := cardsToString(gameInstance.playerHands[msg.UserId])
					usernames := a.getUsernamesRoom(roomId)
					updateHandEvent := newUpdateHandEvent(msg.Username, msg.UserId, roomId, gameInstance.playerHands[msg.UserId], usernames, cardsHandString)
					a.notifyUser(updateHandEvent)
					if len(gameInstance.playerHands[msg.UserId]) == 0 {
						gameFinishEvent := newGameFinishEvent(msg.Username, "", roomId, gameInstance.turnNumber, a.Hub.rooms[roomId], gameInstance.leaderBoard)
						a.notifyRoomMembers(gameFinishEvent)
						gameInstance.resetGameInstance()
						break
					}
					continue
				} else if gameInstance.gameStarted && msg.OperationType == SKIP {
					// if skip, update last player
					// numSkips == 3, we know everyone skipped; wipe last played
					gameInstance.numSkips = gameInstance.numSkips + 1
					if gameInstance.numSkips == 3 {
						gameInstance.lastPlayed = gameInstance.lastPlayed[:0]
					}
					if gameInstance.currentUserTurnIndex == 3 {
						gameInstance.currentUserTurnIndex = 0
					} else {
						gameInstance.currentUserTurnIndex++
					}
				} else {
					log.Println("invalid operation")
				}
			}
		}
	}
}

func (g *GameInstance) connect(user string) {
	g.dealCards(user)
	g.playerCount = g.playerCount + 1
}

func (g *GameInstance) validateMove(c []Card) error {
	// TODO: Check if cards exist in hand
	// Check if first turn has a 3 of diamonds played
	if g.turnNumber == 1 && !slices.Contains(c, NewCard(Three, Diamonds)) {
		return errors.New("must have 3 of diamonds")
	}
	if len(g.lastPlayed) == 0 {
		return nil
	}
	// Check if valid number of cards eg: should not play full house on a single 3
	if (len(c) != len(g.lastPlayed)) && (len(c) != 1 && GetRank(c[0]) != Two) {
		return errors.New("invalid play")
	}
	// Compare high card
	if len(c) == 1 && c[0] < g.lastPlayed[0] {
		return errors.New("card smaller than last played")
	}
	// Compare poker hands
	if len(c) > 1 {
		ph, phr := getPokerHand(c)
		phlp, phrlp := getPokerHand(g.lastPlayed)

		if ph < phlp {
			return errors.New("poker hand smaller than last played")
		}

		if ph == phrlp && phr < phrlp {
			return errors.New("pokerhand rank smaller than last played")
		}
	}
	return nil
}

func (g *GameInstance) disconnect(user string) {
	delete(g.playerHands, user)
}

func (g *GameInstance) resetGameInstance() {
	g.turnNumber = 0
	g.numSkips = 0
	g.gameStarted = false
	shuffledDeck := shuffleDeck()
	g.deck = shuffledDeck
	for i := range g.playerHands {
		g.dealCards(i)
	}
	g.lastPlayed = nil
}

func (g *GameInstance) findThreeOfDiamonds() string {
	for key, j := range g.playerHands {
		if slices.Contains(j, NewCard(Three, Diamonds)) {
			return key
		}
	}
	return ""
}

type GameEvent struct {
	OperationType string `json:operation`
	RoomId        string `json:roomid`
	Username      string `json:username`
	UserId        string
	TurnData
	CardData
	Players     []string
	Leaderboard map[string]int
}

func newConnectEvent(username string, roomId string, userId string, players []string, leaderboard map[string]int) GameEvent {
	return GameEvent{CONNECT, roomId, username, userId, TurnData{}, CardData{}, players, leaderboard}
}

func newDisconnectEvent(username string, roomId string, userId string, players []string) GameEvent {
	return GameEvent{DISCONNECT, roomId, username, userId, TurnData{}, CardData{}, players, nil}
}

// turn info
type TurnData struct {
	CurrentUserTurn      string `json:currentturn,omitempty`
	TurnNumber           int    `json:turnnumber`
	CurrentUserTurnIndex int
}

type PlayerData struct {
	Players []string
}

func newSkipEvent(username string, userId string, roomId string, currUser string, turnNumber int, lastPlayedCards []Card, players []string, lastPlayedCardsString []string, currentUserTurnIndex int) GameEvent {
	return GameEvent{SKIP, roomId, username, userId, TurnData{CurrentUserTurn: currUser, TurnNumber: turnNumber, CurrentUserTurnIndex: currentUserTurnIndex}, CardData{lastPlayedCards, lastPlayedCardsString}, players, nil}
}

func newGameStartEvent(username string, userId string, roomId string, currUser string, turnNumber int, players []string, hand []Card, handString []string, currUserTurnIndex int, leaderboard map[string]int) GameEvent {
	return GameEvent{GAMESTART, roomId, username, userId, TurnData{CurrentUserTurn: currUser, TurnNumber: turnNumber, CurrentUserTurnIndex: currUserTurnIndex}, CardData{hand, handString}, players, leaderboard}
}
func newGameFinishEvent(username string, userId string, roomId string, turnNumber int, players []string, leaderboard map[string]int) GameEvent {
	return GameEvent{GAMEFINISH, roomId, username, userId, TurnData{TurnNumber: turnNumber}, CardData{}, players, leaderboard}
}

func newUpdateHandEvent(username string, userId string, roomId string, cards []Card, players []string, cardsString []string) GameEvent {
	return GameEvent{UPDATE_HAND, roomId, username, userId, TurnData{}, CardData{Cards: cards, CardString: cardsString}, players, nil}
}

// card info
type CardData struct {
	Cards      []Card
	CardString []string
}

func newPlayEvent(username string, roomId string, currUser string, turnNumber int, cards []Card, userId string, players []string, cardsString []string) GameEvent {
	return GameEvent{ACTION, roomId, username, userId, TurnData{CurrentUserTurn: currUser, TurnNumber: turnNumber}, CardData{cards, cardsString}, players, nil}
}

type GameInstance struct {
	lastPlayed           []Card
	playerHands          map[string][]Card
	deck                 []Card
	playerCount          int
	gameStarted          bool
	currentUserTurnIndex int
	turnNumber           int
	numSkips             int
	roundNumber          int
	leaderBoard          map[string]int
}

func newGameInstance() GameInstance {
	shuffledDeck := shuffleDeck()
	playerHands := make(map[string][]Card)
	leaderBoard := make(map[string]int)
	return GameInstance{nil, playerHands, shuffledDeck, 0, false, 0, 0, 0, 0, leaderBoard}
}

func (a App) getCurrentUserTurn(roomId string, userIndex int) string {
	return a.Hub.rooms[roomId][userIndex]

}

// Remove cards from hand
// Returns updated hand
// TODO: Fix
func (g *GameInstance) removeCardsFromHand(cards []Card, userId string) {
	hand := g.playerHands[userId]
	for _, j := range cards {
		i := slices.Index(hand, j)
		tmp := hand[i]
		hand[i] = hand[len(hand)-1]
		hand[len(hand)-1] = tmp
		hand = hand[:len(hand)-1]
	}
	g.playerHands[userId] = hand
}

func (a App) getUsernamesRoom(roomId string) []string {

	var usernames []string
	for _, j := range a.Hub.rooms[roomId] {
		usernames = append(usernames, a.Hub.clients[j].name)
	}
	return usernames
}

func (g *GameInstance) dealCards(user string) {
	x := g.deck[0:13]
	if len(g.deck) == 13 {
		g.deck = []Card{}
	} else {
		g.deck = g.deck[13:]
	}
	g.playerHands[user] = x
}

const GAMESTART = "game_start"
const CONNECT = "connect"
const DISCONNECT = "disconnect"
const ACTION = "action"
const SKIP = "skip"
const ERROR = "error"
const GAMEFINISH = "game_finish"
const UPDATE_HAND = "update_hand"
