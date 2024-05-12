package main

import (
	"encoding/json"
	"testing"
)

func TestInitConnection(t *testing.T) {
	r, err := initRabbitMQ()

	if err != nil {
		t.Errorf(err.Error())
	}

	err = r.sendMessage("my_room", RMQMessage{OperationType: "connect", User: "jolai"})
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestConsumeMessage(t *testing.T) {
	r, err := initRabbitMQ()

	if err != nil {
		t.Errorf(err.Error())
	}
	err = r.addListenRoom("my_room")
	if err != nil {
		t.Errorf(err.Error())
	}
	err = r.sendMessage("my_room", RMQMessage{OperationType: "connect", User: "jolai"})
	if err != nil {
		t.Errorf(err.Error())
	}
	err, msgs := r.consume()
	if err != nil {
		t.Errorf(err.Error())
	}
	for d := range msgs {
		var msg RMQMessage
		err := json.Unmarshal(d.Body, &msg)
		if err != nil {
			t.Errorf(err.Error())
		}
		if msg.User == "jolai" {
			break
		}
		t.Errorf("invalid message")
	}
}
