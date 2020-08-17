```Date: March 14th 2020```
# HybridPipe

### Tools / Platforms used
-   [Apache Kafka](https://kafka.apache.org/)
-   [NATS](https://nats.io/)
-   [RabbitMQ](https://www.rabbitmq.com/)
-   [NSQ](https://nsq.io/)
-   [TCP Library for Go](https://golang.org/pkg/net/)

### Design Goals
1. Components with-in Client Application systems, should be able to communicate between them in both Async and Pseudo Sync communication Model.
2. Users (Hybridpipe clients) should be able to select the Messaging platforms available (KAFKA & NATS). [Future
 Expansion may include NSQ/TCP]
3. Clients should be able to communicate with their targets via both KAFKA and NATS at the same time. There shouldn't be multiple clients involved just for Hybridpipe communications.
4. Switching between these message platforms should be very simple and shouldn't involve Hybridpipe re-build and re-deployment.
5. Messaging system should support extra features that may come with platforms like NATS / NSQ etc.
6. Messaging system should always be expandable for inclusion of any new platforms.
7. Expose very liitle set of functions and data structures to the Messaging system clients.
8. HybridPipe implementation will be done in Rust with November Release.

### Developer / User Guide
HybridPipe have 2 components to define the functionalities required.
1.  Platform
2.  Piping Library

Platform contains both KAFKA and NATS deployment files.  Kafka would be deployed in baremetal systems (Not as Container) because of the load and size of the system.  Kafka is having functionality dependence with Zookeeper (Installation is included along with Kafka)

#### Commands to Control the messaging platforms
```shell
## For Kafka
$ sudo systemctl <start/restart/status/stop> zookeeper.service
$ sudo systemctl <start/restart/status/stop> kafka.service
## For NATS
$ nats-server -p 4222 -D -T --user anands --pass this4now &
```

#### NOTE:
For now, since there is no deployment specification defined for App messaging, we are deploying platforms directly on top of OS to extract better performance (Not as Container, eventhough we have created the Dockerfile to make these systems as containers) - This would be taken in phase2 once we have clear understanding on deployments for overall App systems.

### TLS
Using OpenSSL For TLS Client and Server Certificates generation with Local Self signed CA
```shell
# Generate CA: This will generate Certificate Authority and Private Key Files.
$ openssl req -out App_CA.pem -new -x509

# Generate Server Certificate and Key Pair:
$ touch file.srl => update the file with a serial starting number eg: 01
$ openssl genrsa -out app-mq-server.key 2048 => 4096 Encryption would be an overkill
$ openssl req -key app-mq-server.key -new -out app-mq-server.req
$ openssl x509 -req -in app-mq-server.req -CA App_CA.pem -CAkey privkey.pem -CAserial file.srl -out app-mq-server.pem

# Generate Client Certificate and Key Pair:
$ openssl genrsa -out client.key 2048 => We are not encrypting the Key. Else we need to add option "-des3"
$ openssl req -key app-mq-client.key -new -out app-mq-client.req
$ openssl x509 -req -in app-mq-client.req -CA CA.pem -CAkey privkey.pem -CAserial file.srl -out app-mq-client.pem
```

Note: Please make sure we are using same CA.pem and PrivateKey Files generated in Step 1, when we generate both
 Client and Server Certificates along with keys.

### Messaging System Usage

Messaging system exposes following list of Constants and APIs (with Description).  Usage example section would
 explain each possible scenarios along with more description.

**BrokerType will allow user / client to select the underlying Messaging platform out of 4 which are defined.**
1.  NATS
2.  KAFKA
3.  NSQ - Future expansion possibility
4.  RABBITMQ - Future expansion possibility

##### API - **Medium (bt int, fn ResponseHandler) (Handle, error)**
​        bt - BrokerType
​        fn - ResponseHandler which would be called in case of any NATS Sync Request. If User wants their application
 to respond for "REQUEST" queries from other systems, they should enable this
function and they should handle the incoming data, parse and derive the response and return the same.
​        returns
​        MQBus - handle to Messaging system Connection
​        Error - can be nil if there is no error
This API works as Abstract factory to create the end object based on the information uses passes. This API will
 decide if user asked KAFKA / NATS as messaging system, decides if the connection created
for Sending messages or for message subscription and defines thru which channel the communication happens.

##### API - **Distribute (pipe string, data interface{})**
​        pipe - Topic / Subject name to which message to be sent
​        data - Data in interface format. Any data (User-defined or Primitive) can be sent
This API should be called to send message to any specific Pipe (Topic / Subject)

##### API - **Accept (pipe string, fn MessageHandler)) error**
​        pipe - Pipe name (Topic / Subject)
​		fn   - Handler for processing the incoming message, Callback function
This API should be called if user wants to accept message from a specific Pipe.

##### API - **Get(pipe string, data interface{}) interface{}
​		pipe - Pipe name (Topic / Subject)
​		data - Request Data to be sent to the Target.
This API should be used to get any information from another process with-in client space, if the target too uses
 Hybridpipe for transacting data. This will be pseudo synchronous call.

##### API - **Enable(<User-Defined Struct>)**
​		Any user defined Type like "Person" struct defined above, can be passed for this API. This API will enable
 and allow user to send any type of data over Hybridpipe.

##### API - **Close()**
This API would disconnect from the messaging platform and removes the mapping between Messaging channel and
 connection before removing the connection.

### USAGE EXAMPLES and Description
Please refer the files
1. mqconsumer/mqconsumer.go - Consumer implementations for Hybridpipe
2. mqproducer/mqproducer.go - Producer implementations for Hybridpipe

#### To Subscribe for Messages via KAFKA
```go
kafkaObj, err := dc.Medium(dc.KAFKA, nil)
defer kafkaObj.Close()
kafkaObject.Accept("App.iLO5.App1", HandleKafkaMessage)
```
During subscription, user needs to pass their own function pointer to "" API to handle the incoming data. In this
 case, "HandleKafkaMessage" would be user defined one.

#### To Send Messages via KAFKA
```go
kafkaObj, err := dc.Medium(dc.KAFKA, nil)
defer kafkaObj.Close()
kafkaObj.Distribute("App.iLO5.APP1", Data)

// Here P is a struct defined in the Producer side and they want to send it to Consumer
Data := &Person {
        Name: "David Gower",
        Age: 75,
        NextGen: []string{"Pringle", "NH Fairbrother", "Wasim"},
        CAge: []int{45, 37, 39},
        Next:  Nx,
}
```
Here APIs are same. But instead of handler function, we are having struct P. Before sending any data of type Person
 struct, Client need to make messaging system to learn about this data type using API
"LearnDataType (&Person{})".

#### To Subscribe for messages via NATS
```go
NatsObj, err := dc.Medium(dc.NATS, HandleRequestQueriesFromAll)
defer NatsObj.Close()
NatsObj.Accept("App.iLO5.Nats.APP2", HandleNatsMessage)
```
It works similar to Kafka. When the message arrives, that will be passed to Callback function, in this case
 "HandleNatsMessage".

#### To Send messages via NATS
```go
NatsObj, err := dc.Medium(dc.NATS, nil)
defer NatsObj.Close()
NatsObj.Distribute("App.iLO5.Nats.APP2", data)

// In the case, user sends the message in any format.  If user sends a specfic user-defined struct object, that
 should be enabled with Hybridpipe first using API "Enable" liek below

dc.Enable(Person{})  // In this case, Person is a struct definition.  These struct members should be exported already
 and available across these 2 target systems for enabling / sending these user-defined objects.

jd string = `{
    "fruit": "Apple",
    "size": "Large",
    "color": "Red"
}`
```

##### It is now working on RabbitMQ, TCP and Redis. Embedded system deployable Hybridpipe is under development in Rust (Socket Based).
