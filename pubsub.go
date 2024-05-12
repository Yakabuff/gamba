package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func initRabbitMQ() (RMQ, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return RMQ{}, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return RMQ{}, err
	}
	err = ch.ExchangeDeclare(
		"gamba",  // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return RMQ{}, err
	}
	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return RMQ{}, err
	}
	return RMQ{ch: ch, queueName: q.Name}, nil

}

type RMQ struct {
	ch        *amqp.Channel
	queueName string
}

func (r *RMQ) sendMessage(key string, message RMQMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	err = r.ch.PublishWithContext(ctx,
		"gamba", // exchange
		key,     // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	if err != nil {
		return err
	}

	log.Printf(" [x] Sent %s", body)
	return nil
}

func (r *RMQ) consume() (error, <-chan amqp.Delivery) {
	msgs, err := r.ch.Consume(
		r.queueName, // queue
		"",          // consumer
		true,        // auto ack
		false,       // exclusive
		false,       // no local
		false,       // no wait
		nil,         // args
	)
	if err != nil {
		return err, nil
	}
	return nil, msgs
}

func (r *RMQ) addListenRoom(roomId string) error {
	err := r.ch.QueueBind(
		r.queueName, // queue name
		roomId,      // routing key
		"gamba",     // exchange
		false,
		nil)
	if err != nil {
		return err
	}
	return nil
}
func (r *RMQ) removeListenRoom(roomId string) error {
	err := r.ch.QueueUnbind(r.queueName, roomId, "gamba", nil)
	if err != nil {
		return err
	}
	return nil
}

type RMQMessage struct {
	OperationType string `json:operation`
	RoomId        string `json:roomid`
	Cards         []Card `json:cards`
	User          string `json:user`
	TurnNumber    int    `json:turnnumber`
}
