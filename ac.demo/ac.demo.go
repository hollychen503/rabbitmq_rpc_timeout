package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/streadway/amqp"
)

var (
	conn    *amqp.Connection
	ch      *amqp.Channel
	acQueue amqp.Queue
)

const (
	reqTimeout = 5
)

type myParams struct {
	Name  string
	ID    string
	UID   string
	Begin int64 //time.Now().Unix()
	Rsp   string
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func init() {
	log.Println("init system...")

}

func main() {

	var err error

	conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/") // get params from config/env/...
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err = conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	//
	q, err := ch.QueueDeclare(
		"ac_demo_rpc_queue", // name
		false,               // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	failOnError(err, "Failed to declare a ac_demo_rpc_queue queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		//false,  // auto-ack
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			go func() {
				//n, err := strconv.Atoi(string(d.Body))
				fmt.Println(" body: ", string(d.Body))
				my := &myParams{}
				if err := json.Unmarshal(d.Body, my); err != nil {
					log.Println("failed to unmarshal msg")
					//d.Ack(false)
					return
				}
				//failOnError(err, "Failed to convert body to integer")
				log.Println("Receive message:", *my)
				if time.Now().Unix()-my.Begin > reqTimeout {
					log.Println("This request is timeout. Just drop it.")
					//d.Ack(false)
					return
				}

				// 随机休息 N 秒，造成 response timeout
				n := 2 + rand.Intn(4)
				log.Println("sleep ", n, "seconds.")
				time.Sleep(time.Duration(n) * time.Second)
				my.Rsp = "your access code is OK."
				dat, err := json.Marshal(my)
				if err != nil {
					log.Println(" failed to marshal.")
				}
				log.Println("send back response")

				err = ch.Publish(
					"",        // exchange
					d.ReplyTo, // routing key
					false,     // mandatory
					false,     // immediate
					amqp.Publishing{
						ContentType:   "application/json",
						CorrelationId: d.CorrelationId,
						Body:          dat,
					})
				failOnError(err, "Failed to publish a message")

				//d.Ack(false)

			}()

		}
	}()

	log.Printf(" [*] Awaiting RPC requests")
	<-forever
}
