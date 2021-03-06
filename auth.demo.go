package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/chilts/sid"
	"github.com/streadway/amqp"
	/*
		"os"
		"strconv"
		"strings"
		"time"

		"github.com/streadway/amqp"
	*///"github.com/gin-gonic/gin"
)

const (
	reqTimeout = 5
)

var (
	conn *amqp.Connection
	ch   *amqp.Channel
	rpcQ amqp.Queue
	//corrID = randomString(32)
	corrID = sid.Id()
	rspMap sync.Map
	port   int
)

type myParams struct {
	Name    string
	ID      string
	UID     string
	Begin   int64 //time.Now().Unix()
	Rsp     string
	RspCode int
}
type pMyParams *myParams

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func setupRPCQueue() (err error) {
	/*
			Non-Durable and Auto-Deleted queues will not be redeclared on server restart
		and will be deleted by the server after a short time when the last consumer is
		canceled or the last consumer's channel is closed.  Queues with this lifetime
		can also be deleted normally with QueueDelete.  These durable queues can only
		be bound to non-durable exchanges.
	*/
	rpcQ, err = ch.QueueDeclare(
		"",    // name  // 让系统随机取名
		false, // durable // 不需要durable。一旦进程退出，就允许系统把这个queue 销毁
		false, // autoDelete // 至少有一个消费者连接本队列，之后所有与这个队列连接的消费者都断开时，才会自动删除. （生产者客户端创建这个队列，或者没有消费者客户端与这个队列连接，都不会自动删除）
		true,  // exclusive  // 这里怎么解释？ 如果一个队列被声明为排他队列，该队列仅仅对首次声明他的连接可见。并在连接断开时自动删除。“首次”是指如果一个连接已经声明了一个排他队列，其他连接是不允许建立同名的排他队列（只能打开？）。（Channels on other connections will receive an error when attempting  to declare, bind, consume, purge or delete a queue with the same name.）
		false, // noWait //When noWait is true, the queue will assume to be declared on the server.
		nil,   // arguments
	)

	log.Println("rpcQ name:", rpcQ.Name)

	msgs, err := ch.Consume(
		rpcQ.Name, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	//failOnError(err, "Failed to register a consumer")
	if err != nil {
		log.Println("Failed to register a consumer,", err)
		return err
	}

	// 等待 对方 发过来的消息，并处理之
	go func() {
		for d := range msgs {
			if corrID == d.CorrelationId {
				//res, err = strconv.Atoi(string(d.Body))
				//failOnError(err, "Failed to convert body to integer")
				//break
				//log.Println(d)
				handleResponse(d.Body)
			}
		}

	}()

	return err

}

func handleResponse(msg []byte) {
	log.Println("receive response")

	my := myParams{}
	if err := json.Unmarshal(msg, &my); err != nil {
		log.Println("failed to unmarshal msg")
	}
	dif := time.Now().Unix() - my.Begin
	if dif > reqTimeout {
		log.Println(" *** this req is timeout.", my)
		return
	}

	log.Println("  response data:", my)

	c, ok := rspMap.Load(my.UID)
	if ok {
		log.Println("set rsp string to channel")
		cc := c.(chan string)
		cc <- my.Rsp
		return
	}

	log.Println("CAN NOT find channel for UID. drop it.")

	//rspMap.Store(my.UID, &my)
}

func init() {

	log.Println("init system...", corrID)
	myport := flag.Int("port", 23450, "listen port")
	flag.Parse()
	log.Println("listen port is:", *myport)
	port = *myport

	var err error

	conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/") // get params from config/env/...
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err = conn.Channel()
	if err != nil {
		conn.Close()
		failOnError(err, "Failed to open a channel")
	}

	// 建立一个匿名 queue，用来接收 ac.demo 的处理结果；
	err = setupRPCQueue()
	if err != nil {
		ch.Close()
		conn.Close()
		failOnError(err, "Failed to declare rpc queue")
	}

}

func main() {
	log.Println("--- auth demo ---")
	log.Println("q.name=", rpcQ.Name)
	defer ch.Close()
	defer conn.Close()

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)
	e.GET("/users/:name/:id", users)

	// Start server
	e.Logger.Fatal(e.Start(":" + strconv.Itoa(port)))

}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

// user auth

func users(c echo.Context) error {
	//return c.String(http.StatusOK, "Hello, "+c.Param("name")+" with ID "+c.Param("id"))
	my := &myParams{}
	my.Name = c.Param("name")
	my.ID = c.Param("id")
	my.UID = sid.Id()
	my.Begin = time.Now().Unix()

	// check jtw for username ane token
	log.Println("Hello, " + my.Name + " with ID " + my.ID)
	log.Println("  Your ID is OK. Now we will check your access privileges...")

	// 将user name & id 发送给 ac.demo
	dat, err := json.Marshal(my)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal error!"+err.Error())
	}
	log.Println("  will send dat", string(dat))
	err = ch.Publish(
		"",                  // exchange
		"ac_demo_rpc_queue", // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: corrID,
			ReplyTo:       rpcQ.Name,
			Body:          dat,
		})
	//failOnError(err, "Failed to publish a message")
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal error!")
	}

	ready := make(chan string)
	rspMap.Store(my.UID, ready)

	select {
	case <-time.After(5 * time.Second):
		// 如果 N 秒之内还没有响应，终止这个 request
		rspMap.Delete(my.UID)
		return c.String(http.StatusInternalServerError, "Internal error, maybe timeout!")
	case d := <-ready:
		// 扩展： 可以将 response 扩展成 json， 检查 允许/拒绝，。。。
		//   根据 response  的 http code , 设置 成功码/错误码
		//    d => json => http code => client
		log.Println("  receive rsp data:", d)
		return c.String(http.StatusOK, "Hello, "+my.Name+"@"+my.ID+", AC info:"+d)
	}

}
