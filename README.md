# purpose:
demo rabbitmq RPC pattern with timeout

# auth.demo.go
- Demo an authentication service.
- It will only check user name and ID.
- Leave the access privilege checking to ac.demo
- auth.demo will send ac info to RabbitMQ with routing key ac_demo_rpc_queue.

# ac.demo.go
- demo the  access privilege checking
- receiving ac request from routing key ac_demo_rpc_queue and response to msg.ReplyTo queue


# RabbitMQ server
run the following commands to bootup rabbitMQ 

~~~
cd /path/to/rabbitmq_rpc_timeout
docker-compose -f setup.test.env.yml up
~~~

# test
## single or multiple ac.demo
~~~
curl localhost:23450/users/holly/1234 &
curl localhost:23450/users/holly/1234 &
curl localhost:23450/users/holly/1234 &
curl localhost:23450/users/holly/1234 &
~~~

## multiple auth.demo

~~~
go run auth.demo.go -port=23450 &
go run auth.demo.go -port=23451 &
go run auth.demo.go -port=23452 &
~~~

~~~
curl localhost:23450/users/holly/1234 &
curl localhost:23451/users/holly/1234 &
curl localhost:23452/users/holly/1234 &
~~~