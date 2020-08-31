package hybridpipe

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/go-amqp"
)

// AMQPPacket defines the ZeroMQ Message Packet Object. Apart from Base Packet, it
// will contain Connection Object and identifiers related to ZeroMQ.
// 	HandleConn - ZeroMQ Connection Object
// 	PipeHandle - Create the map between Pipe name and NATS Subscription
type AMQPPacket struct {
	Packet
	HandleConn *amqp.Client
}

// AMQPConnect - Similar to KafkaConnect in AMQP context.
func AMQPConnect(ap *AMQPPacket) error {
	// In case where HandleConn is already created, we don't recreate the connection again.
	if ap.HandleConn == nil {
		conn, e := amqp.Dial(HPipeConfig.AMQPServer)
		if e != nil {
			er := fmt.Errorf("AMQP Connect Error: %#v, %s", e, HPipeConfig.AMQPServer)
			return er
		}
		ap.HandleConn = conn
	}
	return nil
}

// Dispatch would send a user defined message to another microservice or service or application.
func (ap *AMQPPacket) Dispatch(pipe string, d interface{}) error {
	var e error
	session, e := ap.HandleConn.NewSession()
	if e != nil {
		er := fmt.Errorf("AMQP Dispatch Session Error - %#v", e)
		return er
	}
	sender, e := session.NewSender(amqp.LinkTargetAddress(pipe))
	if e != nil {
		er := fmt.Errorf("Creating Sender Link - FAILED : %#v", e)
		return er
	}
	// Message Encoding done
	b, e := Encode(d)
	if e != nil {
		log.Printf("%v", e)
		return e
	}
	Ctx := context.Background()
	defer sender.Close(Ctx)
	if e = sender.Send(Ctx, amqp.NewMessage(b)); e != nil {
		er := fmt.Errorf("Message dispatch Error - %#v", e)
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
	session, e := ap.HandleConn.NewSession()
	if e != nil {
		er := fmt.Errorf("AMQP Accept Session Error - %#v", e)
		return er
	}
	receiver, e := session.NewReceiver(
		amqp.LinkTargetAddress(pipe),
		amqp.LinkCredit(1),
	)
	if e != nil {
		er := fmt.Errorf("Creating Receiver Link - FAILED : %#v", e)
		return er
	}
	ap.read(receiver, pipe, fn)
	return nil
}

func (ap *AMQPPacket) read(r *amqp.Receiver, p string, fn Process) error {
	c := context.Background()
	defer r.Close(c)
	fmt.Println("Just before FOR loop")
	for {
		m, e := r.Receive(c)
		if e != nil {
			er := fmt.Errorf("Message Receive Error - %#v", e)
			return er
		} else {
			fmt.Println("OK")
		}
		m.Accept(c)
		fn(m.GetData())
	}
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
	ap.HandleConn.Close()
}
