package hybridpipe

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Azure/go-amqp"
)

// AMQPPacket defines the ZeroMQ Message Packet Object. Apart from Base Packet, it
// will contain Connection Object and identifiers related to ZeroMQ.
// 	HandleConn - ZeroMQ Connection Object
// 	PipeHandle - Create the map between Pipe name and NATS Subscription
type AMQPPacket struct {
	Packet
	HandleConn *amqp.Client
	MSender    *amqp.Session
	MReceiver  *amqp.Session
}

// AMQPConnect - Similar to KafkaConnect in AMQP context.
func AMQPConnect(ap *AMQPPacket) error {
	// In case where HandleConn is already created, we don't recreate the connection again.
	if ap.HandleConn == nil {
		conn, e := amqp.Dial(HPipeConfig.AMQPServer)
		if e != nil {
			_, er := fmt.Println("AMQP Connect Error: %#v, %s", e, HPipeConfig.AMQPServer)
			return er
		}
		ap.HandleConn = conn
	}
	return nil
}

// Dispatch would send a user defined message to another microservice or service or application.
func (ap *AMQPPacket) Dispatch(pipe string, d interface{}) error {
	var e error
	if ap.MSender, e = ap.HandleConn.NewSession(); e != nil {
		_, er := fmt.Println("AMQP Dispatch Session Error - %#v", e)
		return er
	}

	Ctx := context.Background()
	S, e := ap.MSender.NewSender(amqp.LinkTargetAddress("/" + pipe))
	if e != nil {
		_, er := fmt.Println("Creating Sender Link - FAILED : %#v", e)
		return er
	}
	// Message Encoding done
	b, e := Encode(d)
	if e != nil {
		log.Printf("%v", e)
		return e
	}

	// Dispatching the message.
	Ctx, Cancel := context.WithTimeout(Ctx, 5*time.Second)
	defer Cancel()
	defer S.Close(Ctx)
	if e = S.Send(Ctx, amqp.NewMessage(b)); e != nil {
		_, er := fmt.Println("Message dispatch Error - %#v", e)
		return er
	}
	return nil
}

// Distribute - Will call Dispatch with the same parameter. Implemented just
// to support other model of communications. (NATS, KAFKA etc)
func (ap *AMQPPacket) Distribute(pipe string, d interface{}) error {
	return ap.Dispatch(pipe, d)
}

// Accept defines the Subscription / Consume procedure. The Message processing
// will be done in separate Go routine.
func (ap *AMQPPacket) Accept(pipe string, fn Process) error {
	var e error
	if ap.MReceiver, e = ap.HandleConn.NewSession(); e != nil {
		_, er := fmt.Println("AMQP Accept Session Error - %#v", e)
		return er
	}

	Ctx := context.Background()
	R, e := ap.MReceiver.NewReceiver(
		amqp.LinkTargetAddress("/"+pipe),
		amqp.LinkCredit(1))
	if e != nil {
		_, er := fmt.Println("Creating Receiver Link - FAILED : %#v", e)
		return er
	}
	Ctx, Cancel := context.WithTimeout(Ctx, 1*time.Second)
	defer R.Close(Ctx)
	defer Cancel()

	// Receive & Accept the message.
	m, e := R.Receive(Ctx)
	if e != nil {
		_, er := fmt.Println("Message Accept Error - %#v", e)
		return er
	}
	m.Accept(Ctx)
	var d interface{}
	fmt.Println(m.GetData())
	if e = Decode(m.GetData(), &d); e != nil {
		_, er := fmt.Println("Received Message parsing error - %#v", e)
		return er
	}
	go fn(d)
	return nil
}

// Get - Not Supported in AMQP.
func (ap *AMQPPacket) Get(pipe string, d interface{}) interface{} {
	return nil
}

// Remove will close a specific session from the local stored map
func (ap *AMQPPacket) Remove(pipe string) error {
	return nil
}

// Close will close AMQP connection. Repeated calls would result in
// unexpected error.
func (ap *AMQPPacket) Close() {
}
