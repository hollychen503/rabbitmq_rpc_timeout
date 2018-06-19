# Purpose:
demo rabbitmq RPC pattern with timeout functionality

# Files:
### auth.demo.go
- Demo an authentication service.
- It will only check user name and ID.
- Leave the access privilege checking to ac.demo
- auth.demo will send ac info to RabbitMQ with routing key ac_demo_rpc_queue.

### ac.demo.go
- demo the  access privilege checking
- receiving ac request from routing key ac_demo_rpc_queue and response to msg.ReplyTo queue


### setup.test.env.yml
- boot up RabbitMQ server

~~~
cd /path/to/rabbitmq_rpc_timeout
docker-compose -f setup.test.env.yml up
~~~

# Test
- single or multiple ac.demo
~~~
curl localhost:23450/users/holly/1234 &
curl localhost:23450/users/holly/1234 &
curl localhost:23450/users/holly/1234 &
curl localhost:23450/users/holly/1234 &
~~~

- stress test (you can run as much as possible of  ac.demo instance to share loading)
~~~
go run ac.demo/ac.demo.go -sleep=false
go run ac.demo/ac.demo.go -sleep=false
...
~~~

~~~
ab -c 100 -n 1000  localhost:23450/users/holly/1234
...
...
Concurrency Level:      100
Time taken for tests:   9.877 seconds
Complete requests:      10000
Failed requests:        0
Total transferred:      1670000 bytes
HTML transferred:       500000 bytes
Requests per second:    1012.50 [#/sec] (mean)
Time per request:       98.765 [ms] (mean)
Time per request:       0.988 [ms] (mean, across all concurrent requests)
Transfer rate:          165.12 [Kbytes/sec] received
~~~

- multiple auth.demo

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

- demo 1/4  requests timeout 

~~~
go run ac.demo/ac.demo.go -n=4
~~~

~~~
ab -c 100 -n 1000  localhost:23451/users/holly/1234

...

Concurrency Level:      100
Time taken for tests:   15.340 seconds
Complete requests:      1000
Failed requests:        250       894
~~~
Failed requests = 250 = 1000/4



# TODO
- what if ac.demo is not up and running? will RabbitMQ clean ac_demo_rpc_queue queue?




