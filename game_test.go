package main

import (
	"fmt"
	"testing"
	"time"
)

func TestGameStart(t *testing.T) {
	app := newApp()

	room := newRoom()
	roomId := room.id
	app.Hub.rooms[room.id] = make([]string, 0, 4)

	roomChannel := make(chan GameEvent, 118)
	app.Hub.roomChannels[room.id] = roomChannel
	client1 := newClient("a")
	client2 := newClient("b")
	client3 := newClient("c")
	client4 := newClient("d")

	gi := newGameInstance()
	app.GameInstances[room.id] = &gi

	go app.spawnGameLoop(roomId)

	app.Hub.rooms[roomId] = append(app.Hub.rooms[roomId], client1.id)
	fmt.Printf("adding client id %s to room %s", client1.id, roomId)
	app.Hub.clients[client1.id] = client1
	connectEvent := newConnectEvent(client1.name, roomId, client1.id, app.Hub.rooms[roomId])
	app.Hub.roomChannels[roomId] <- connectEvent
	time.Sleep(1 * time.Second)
	if len(client1.events) != 1 {
		t.Errorf("invalid %d", len(client1.events))
	}

	app.Hub.rooms[roomId] = append(app.Hub.rooms[roomId], client2.id)
	fmt.Printf("adding client id %s to room %s", client2.id, roomId)
	app.Hub.clients[client2.id] = client2
	connectEvent = newConnectEvent(client2.name, roomId, client2.id, app.Hub.rooms[roomId])
	app.Hub.roomChannels[roomId] <- connectEvent
	time.Sleep(1 * time.Second)
	if len(client1.events) != 2 {
		t.Errorf("invalid %d", len(client1.events))
	}
	if len(client2.events) != 1 {
		t.Errorf("invalid %d", len(client1.events))
	}

	app.Hub.rooms[roomId] = append(app.Hub.rooms[roomId], client3.id)
	fmt.Printf("adding client id %s to room %s", client3.id, roomId)
	app.Hub.clients[client3.id] = client3
	connectEvent = newConnectEvent(client3.name, roomId, client3.id, app.Hub.rooms[roomId])
	app.Hub.roomChannels[roomId] <- connectEvent
	time.Sleep(1 * time.Second)
	if len(client1.events) != 3 {
		t.Errorf("invalid %d", len(client1.events))
	}
	if len(client2.events) != 2 {
		t.Errorf("invalid %d", len(client1.events))
	}
	if len(client3.events) != 1 {
		t.Errorf("invalid %d", len(client1.events))
	}

	app.Hub.rooms[roomId] = append(app.Hub.rooms[roomId], client4.id)
	fmt.Printf("adding client id %s to room %s", client4.id, roomId)
	app.Hub.clients[client4.id] = client4
	connectEvent = newConnectEvent(client4.name, roomId, client4.id, app.Hub.rooms[roomId])
	app.Hub.roomChannels[roomId] <- connectEvent
	time.Sleep(1 * time.Second)
	if len(client1.events) != 5 {
		t.Errorf("invalid %d", len(client1.events))
	}
	if len(client2.events) != 4 {
		t.Errorf("invalid %d", len(client1.events))
	}
	if len(client3.events) != 3 {
		t.Errorf("invalid %d", len(client1.events))
	}
	if len(client4.events) != 2 {
		t.Errorf("invalid %d", len(client1.events))
	}
}
