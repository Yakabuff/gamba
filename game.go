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
		timeout := time.After(10 * time.Second)
		select {
		case <-timeout:
			// timeout: skip current user's turn
			// increment round num
			// notify next user that it's their turn
			// if turn 1, must not pass; if user doesn't play card, play 3 of diamonds for them
			// if 3 timeouts, play lowest card
			currUserTurn := a.getCurrentUserTurn(roomId, gameInstance.currentUserTurnIndex)
			currUserName := a.Hub.clients[currUserTurn].name
			if gameInstance.turnNumber == 1 {
				user := gameInstance.findThreeOfDiamonds()
				removeCardsFromHand(gameInstance.playerHands[user], []Card{NewCard(Three, Diamonds)})
				gameInstance.lastPlayed = []Card{NewCard(Three, Diamonds)}
			} else if gameInstance.numSkips == 3 {
				lowest := slices.Min(gameInstance.playerHands[currUserTurn])
				removeCardsFromHand(gameInstance.playerHands[currUserTurn], []Card{lowest})
				gameInstance.lastPlayed = []Card{lowest}
			}
			if gameInstance.currentUserTurnIndex == 3 {
				gameInstance.currentUserTurnIndex = 0
			} else {
				gameInstance.currentUserTurnIndex++
			}
			nextUserTurn := a.getCurrentUserTurn(roomId, gameInstance.currentUserTurnIndex)
			skipMessage := newSkipEvent(currUserName, "", roomId, nextUserTurn, gameInstance.turnNumber+1, gameInstance.lastPlayed)
			a.notifyRoomMembers(skipMessage)
		case msg := <-channel:
			fmt.Println(msg)
			if !gameInstance.gameStarted && msg.OperationType == CONNECT {
				// connect: if player count < 4, notify room
				// if player count == 4, emit game start message
				// set game instance status to game started and current user's turn
				// current user is user with 3 of diamonds
				// notify room who's starting
				fmt.Println("connect")
				gameInstance.connect(msg.UserId)

				a.notifyRoomMembers(msg)
				if gameInstance.playerCount == 4 {
					gameInstance.gameStarted = true
					currUserTurn := gameInstance.findThreeOfDiamonds()
					gameStartEvent := newGameStartEvent(msg.Username, "", msg.RoomId, currUserTurn, 1)
					a.notifyRoomMembers(gameStartEvent)
				}

			} else if msg.OperationType == DISCONNECT {
				// disconnect: remove user from game and server
				// notify room
				// Note: doesnt matter if user disconnects mid turn, let it timeout
				fmt.Println("disconnect")
				gameInstance.disconnect(msg.UserId)
				a.disconnectRoomMember(msg)
				disconnectEvent := newDisconnectEvent(msg.Username, msg.RoomId, "")
				a.notifyRoomMembers(disconnectEvent)
			} else if gameInstance.gameStarted && msg.OperationType == MOVE {
				fmt.Println("action")
				removeCardsFromHand(gameInstance.playerHands[msg.UserId], msg.Cards)
				gameInstance.lastPlayed = msg.Cards
				a.notifyRoomMembers(msg)
				if len(gameInstance.playerHands[msg.UserId]) == 0 {
					gameInstance.gameStarted = false
				}
			} else if gameInstance.gameStarted && msg.OperationType == SKIP {
				// if skip, update last player
				// numSkips == 3, we know everyone skipped; wipe last played
				gameInstance.numSkips++
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

func (g *GameInstance) connect(user string) {
	cards := dealCards(g.deck)
	g.playerHands[user] = cards
	g.playerCount++
}

func (g *GameInstance) validateMove(c []Card) error {

	// Check if first turn has a 3 of diamonds played
	if g.turnNumber == 1 && !slices.Contains(c, NewCard(Three, Diamonds)) {
		return errors.New("must have 3 of diamonds")
	}
	// Check if valid number of cards eg: should not play full house on a single 3
	if (len(c) != len(g.lastPlayed)) && (len(c) != 1 && GetRank(c[0]) != Two) {
		return errors.New("invalid play")
	}

	if len(c) == 1 && c[0] < g.lastPlayed[0] {
		return errors.New("card smaller than last played")
	}
	return nil
}

func (g *GameInstance) disconnect(user string) {
	delete(g.playerHands, user)
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
}

func newConnectEvent(username string, roomId string, userId string) GameEvent {
	return GameEvent{CONNECT, roomId, username, userId, TurnData{}, CardData{}}
}

func newDisconnectEvent(username string, roomId string, userId string) GameEvent {
	return GameEvent{DISCONNECT, roomId, username, userId, TurnData{}, CardData{}}
}

// turn info
type TurnData struct {
	CurrentUserTurn string `json:currentturn,omitempty`
	TurnNumber      int    `json:turnnumber`
}

func newSkipEvent(username string, userId string, roomId string, currUser string, turnNumber int, lastPlayedCards []Card) GameEvent {
	return GameEvent{SKIP, roomId, username, userId, TurnData{CurrentUserTurn: currUser, TurnNumber: turnNumber}, CardData{lastPlayedCards}}
}

func newGameStartEvent(username string, userId string, roomId string, currUser string, turnNumber int) GameEvent {
	return GameEvent{GAMESTART, roomId, username, userId, TurnData{CurrentUserTurn: currUser, TurnNumber: turnNumber}, CardData{}}
}

// card info
type CardData struct {
	Cards []Card `json:"cards"`
}

func newPlayEvent(username string, roomId string, currUser string, turnNumber int, cards []Card, userId string) GameEvent {
	return GameEvent{GAMESTART, roomId, username, userId, TurnData{CurrentUserTurn: currUser, TurnNumber: turnNumber}, CardData{cards}}
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
}

func newGameInstance() GameInstance {
	shuffledDeck := shuffleDeck()
	playerHands := make(map[string][]Card)
	return GameInstance{nil, playerHands, shuffledDeck, 0, false, 0, 0, 0}
}

func (a App) getCurrentUserTurn(roomId string, userIndex int) string {
	return a.Hub.rooms[roomId][userIndex]

}

// Remove cards from hand
// Returns updated hand
func removeCardsFromHand(hand []Card, cards []Card) []Card {
	for _, j := range cards {
		i := slices.Index(hand, j)
		hand[i] = hand[len(hand)-1]
		hand = hand[:len(hand)-1]
	}
	return hand
}

const GAMESTART = "game_start"
const CONNECT = "connect"
const DISCONNECT = "disconnect"
const MOVE = "move"
const SKIP = "skip"
const ERROR = "error"
