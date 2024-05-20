package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	app := newApp()
	app.Redis = initRedis()
	err := app.addSelfToServerList(app.NodeName)
	if err != nil {
		log.Panic(err)
	}
	// go app.listenPubsub()
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "home.html")
	})
	r.Post("/create", app.postCreateRoom)
	r.Post("/join", app.postJoinRoom)
	r.Get("/{room_id}", app.getJoinRoom)
	r.Get("/events", app.roomStateSSE)
	r.Post("/move", app.postPlayCards)

	http.ListenAndServe(":3333", r)
}

func (e App) getJoinRoom(w http.ResponseWriter, r *http.Request) {
	roomId := strings.Split(r.URL.Path, "/")[1]
	id, _ := r.Cookie("uuid")
	fmt.Println(e.Hub.rooms[roomId])
	t, err := template.ParseFiles("room.html")
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	data := struct {
		RoomID string
		Uuid   string
	}{
		RoomID: roomId,
		Uuid:   id.Value,
	}
	err = t.Execute(w, data)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
}
func (e App) postJoinRoom(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	id := r.Form.Get("room_id")
	name := r.Form.Get("name")
	if name == "" || id == "" {
		w.WriteHeader(400)
		return
	}
	// Verify room exists in node.  If it doesn't, try to find in redis
	// If exist on node, update node state and redirect to room page
	// If not exist on node but exist on redis, redirect to room page on that node

	_, ok := e.Hub.rooms[id]

	if !ok {
		res, err := e.findRoom(id)
		if err != nil || res == "" {
			w.WriteHeader(500)
			return
		}
		http.Redirect(w, r, res+"/join", http.StatusTemporaryRedirect)
	}

	client := newClient(name)
	e.Hub.rooms[id] = append(e.Hub.rooms[id], client.id)
	fmt.Println(e.Hub.rooms[id])
	fmt.Printf("adding client id %s to room %s", client.id, id)
	e.Hub.clients[client.id] = client
	connectEvent := newConnectEvent(client.name, id, client.id, e.Hub.rooms[id])
	e.Hub.roomChannels[id] <- connectEvent
	cookie := http.Cookie{
		Name:     "uuid",
		Value:    client.id,
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/"+id, http.StatusSeeOther)
}
func (e App) postCreateRoom(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	name := r.Form.Get("name")
	if name == "" {
		w.WriteHeader(400)
		return
	}
	// Check if node is full
	// If full, find server on redis that isn't full
	// Call create room endpoint on that server
	// If not full, proceed with room initialization

	numRooms := len(e.Hub.rooms)

	if numRooms > 50 {
		s, err := e.findServer()
		if err != nil {
			w.WriteHeader(500)
			return
		}
		http.Redirect(w, r, s+"/create", http.StatusTemporaryRedirect)
	}

	room := newRoom()
	client := newClient(name)
	e.Hub.rooms[room.id] = make([]string, 0, 4)
	e.Hub.rooms[room.id] = append(e.Hub.rooms[room.id], client.id)
	e.Hub.clients[client.id] = client

	roomChannel := make(chan GameEvent, 118)
	e.Hub.roomChannels[room.id] = roomChannel
	gi := newGameInstance()
	e.GameInstances[room.id] = &gi
	go e.spawnGameLoop(room.id)
	connectEvent := newConnectEvent(client.name, room.id, client.id, nil)
	e.Hub.roomChannels[room.id] <- connectEvent
	cookie := http.Cookie{
		Name:     "uuid",
		Value:    client.id,
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	// Add room to cache
	err := e.setRoomInfo(room.id, 0)
	err2 := e.updateNumRooms(room.id, 0)
	if err != nil {
		delete(e.Hub.rooms, room.id)
		delete(e.Hub.clients, client.id)
		e.deleteKey(room.id)
		w.WriteHeader(500)
		return
	}
	if err2 != nil {
		delete(e.Hub.rooms, room.id)
		delete(e.Hub.clients, client.id)
		w.WriteHeader(500)
		return
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/"+room.id, http.StatusSeeOther)
}

func (a App) roomCleanup(roomId string, name string) {

}

func (a App) roomStateSSE(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	room_id := r.Form.Get("room_id")
	uuid := r.Form.Get("uuid")
	if room_id == "" || uuid == "" {
		w.WriteHeader(400)
		return
	}
	client := a.Hub.clients[uuid]
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	for {
		// timeout := time.After(5 * time.Second)
		select {
		case ev := <-client.events:
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.Encode(ev)
			fmt.Fprintf(w, "data: %v\n\n", buf.String())
			fmt.Printf("data: %v\n", buf.String())
			// case <-timeout:
			// 	fmt.Println("timeout")
			// 	fmt.Fprintf(w, ": nothing to sent\n\n")
		}
		fmt.Println("here")
		if f, ok := w.(http.Flusher); ok {
			fmt.Println("flushing")
			f.Flush()
		}
		fmt.Println("finished flushing")
	}
}

// func updateGameState(client *Client) {
// 	for {
// 		gs := GameState{
// 			User: uint(rand.Uint32()),
// 		}
// 		time.Sleep(8 * time.Second)
// 		client.events <- gs
// 	}
// }

type GameState struct {
	User uint
}

// Check if valid move
// If valid move, send to game instance
func (e App) postPlayCards(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	data := &playCardRequest{}
	err := decoder.Decode(&data)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	uuid := data.uuid
	roomId := data.roomId
	cards := data.play
	if uuid == "" || roomId == "" || len(cards) == 0 {
		w.WriteHeader(400)
		return
	}
	_, ok := e.Hub.rooms[roomId]

	if !ok {
		w.WriteHeader(400)
		return
	}
	validUUID := slices.Contains(e.Hub.rooms[roomId], uuid)
	if !validUUID {
		w.WriteHeader(400)
		return
	}
	game := e.GameInstances[roomId]
	err = game.validateMove(cards)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	username := e.Hub.clients[uuid].name
	playEvent := newPlayEvent(username, roomId, "", 0, cards, uuid, e.Hub.rooms[roomId], nil)
	e.Hub.roomChannels[uuid] <- playEvent
}

type playCardRequest struct {
	uuid   string
	roomId string
	play   []Card
}
