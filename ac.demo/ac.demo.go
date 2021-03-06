package main

import (
	"encoding/json"
	"flag"
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
	total   int
)

const (
	reqTimeout = 5 // 请求超时时间
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
	useSleep := flag.Bool("sleep", true, "use sleep to demo timeout")
	nPart := flag.Int("n", 1, "1/n will timeout")

	flag.Parse()
	log.Println("use sleep:", *useSleep)

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
		false,               // autoDelete：delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	failOnError(err, "Failed to declare a ac_demo_rpc_queue queue")

	err = ch.Qos(
		1,     // prefetch count  （1: 在 worker 回复 ack 之前，不要给他新msg。这样就可以把msg 给其他闲人。） （don't dispatch a new message to a worker until it has processed and acknowledged the previous one.）
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		//true,  // auto-ack  //自动 ack。不要手动，让 client timeout 机制发挥它应有的作用??
		false, // 手动 ack. 必须手动ack，否则 consumer会一直接消息，不管自己能不能处理得过来！
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d0 := range msgs {
			fmt.Println("d0.ReplyTo=", d0.ReplyTo)
			go func(d amqp.Delivery) {
				//n, err := strconv.Atoi(string(d.Body))
				fmt.Println("d1.ReplyTo=", d.ReplyTo)

				fmt.Println(" body: ", string(d.Body))
				my := &myParams{}
				if err := json.Unmarshal(d.Body, my); err != nil {
					log.Println("failed to unmarshal msg")
					d.Ack(false)
					return
				}
				//failOnError(err, "Failed to convert body to integer")
				log.Println("Receive message:", *my)
				if time.Now().Unix()-my.Begin > reqTimeout {
					log.Println("This request is timeout. Just drop it.")
					d.Ack(false)
					return
				}

				if *nPart > 1 {
					// n 分之一 将 sleep, 模拟 timeout
					total++
					if total%(*nPart) == 0 {
						fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
						log.Println("sleep 6 second. demo timeout.")
						time.Sleep(6 * time.Second)
						fmt.Println("----------------------------------------------------------------")
					}
				}

				if *useSleep && *nPart <= 1 {
					// 随机休息 N 秒，造成 部分请求 timeout 效果
					n := 2 + rand.Intn(4)
					log.Println("sleep ", n, "seconds.")
					time.Sleep(time.Duration(n) * time.Second)
				}

				my.Rsp = "your access code is OK."
				dat, err := json.Marshal(my)
				if err != nil {
					log.Println(" failed to marshal.")
				}
				log.Println("send response back to ", d.ReplyTo)

				// test error marshal
				//dat = nil

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

				d.Ack(false)

			}(d0) // 注意这里不能使用指针传递或引用传递，否则会出错。尽快copy走数据！
		}
	}()

	log.Printf(" [*] Awaiting RPC requests")
	<-forever
}
