package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Hub *Hub
	// // Rmq      *RMQ
	// Msgs     <-chan amqp091.Delivery
	Redis         *redis.Client
	NodeName      string
	GameInstances map[string]*GameInstance
}
type Hub struct {
	clients      map[string]Client
	rooms        map[string][]string
	roomChannels map[string]chan GameEvent
}

func newApp() App {
	hub := newHub()
	redis := initRedis()
	// r, err := initRabbitMQ()
	// if err != nil {
	// 	log.Panic(err)
	// }
	// err, msgs := r.consume()
	// if err != nil {
	// log.Panic(err)
	// }
	// return App{&hub, &r, msgs, redis, "asdf"}
	gameInstances := make(map[string]*GameInstance)
	return App{&hub, redis, "asdf", gameInstances}
}

// func (a App) listenPubsub() {
// 	var forever chan struct{}
// 	go func() {
// 		for d := range a.Msgs {
// 			log.Printf(" [x] %s", d.Body)
// 			var msg RMQMessage
// 			err := json.Unmarshal(d.Body, &msg)
// 			if err != nil {
// 				log.Println(err.Error())
// 				continue
// 			}
// 			roomId := msg.RoomId
// 			val, ok := a.Hub.roomChannels[roomId]
// 			if ok {
// 				val <- msg
// 			}
// 		}
// 	}()
// 	<-forever
// }

func newHub() Hub {
	clients := make(map[string]Client)
	rooms := make(map[string][]string)
	roomChannels := make(map[string]chan GameEvent)
	return Hub{clients: clients, rooms: rooms, roomChannels: roomChannels}
}

func newRoom() Room {
	uuid := uuid.NewString()
	deck := make([]Card, 54)
	playerHands := make(map[string][]Card)
	return Room{id: uuid, deck: deck, playerHands: playerHands}
}

type Room struct {
	id          string
	deck        []Card
	playerHands map[string][]Card
}

func (e App) notifyRoomMembers(g GameEvent) {
	members := e.Hub.rooms[g.RoomId]
	fmt.Println(members)
	for _, j := range members {
		fmt.Println("Notifying" + j)
		client := e.Hub.clients[j]
		fmt.Println(client)
		client.events <- g
		fmt.Println("finished notifying" + j)
	}
}

func (e App) notifyUser(g GameEvent) {
	client := e.Hub.clients[g.UserId]
	client.events <- g
}

// Delete client from map
// Delete client from room
func (e App) disconnectRoomMember(g GameEvent) {
	delete(e.Hub.clients, g.UserId)
	var index int
	for i, j := range e.Hub.rooms[g.RoomId] {
		if j == g.UserId {
			index = i
		}
	}
	x := e.Hub.rooms[g.RoomId][:index]
	y := e.Hub.rooms[g.RoomId][index+1:]
	final := append(x, y...)
	e.Hub.rooms[g.RoomId] = final
}
