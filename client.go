package main

import "github.com/google/uuid"

func newClient(name string) Client {
	uuid := uuid.NewString()
	// 4 * 13 + 3 * 13 + 2 * 13 + 1
	// first player plays 1 card each turn, other players all pass regardless if they have cards
	// 2nd player plays 1 card each turn etc etc
	// last player gets a loss game event
	return Client{events: make(chan GameEvent, 118), id: uuid, name: name}
}

// Fetches messages from hub and sends via sse
func (c *Client) writePump() {
}

type Client struct {
	name   string
	id     string
	events chan GameEvent
}
