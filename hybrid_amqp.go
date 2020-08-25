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
	MSenders   map[string]*amqp.Session
	MReceivers map[string]*amqp.Session
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

	// Initialize the session maps for both dispatcher and receiver.
	if ap.MSenders == nil {
		ap.MSenders = make(map[string]*amqp.Session)
	}
	if ap.MReceivers == nil {
		ap.MReceivers = make(map[string]*amqp.Session)
	}
	return nil
}

// Dispatch would send a user defined message to another microservice or service or application.
func (ap *AMQPPacket) Dispatch(pipe string, d interface{}) error {
	_, isAvail := ap.MSenders[pipe]
	if !isAvail {
		var e error
		ap.MSenders[pipe], e = ap.HandleConn.NewSession()
		if e != nil {
			er := fmt.Errorf("AMQP Session Error: %#v", e)
			return er
		}
	}
	Ctx := context.Background()
	S, e := ap.MSenders[pipe].NewSender(amqp.LinkTargetAddress("/" + pipe))
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

	// Dispatching the message.
	Ctx, Cancel := context.WithTimeout(Ctx, 5*time.Second)
	defer Cancel()
	if e = S.Send(Ctx, amqp.NewMessage(b)); e != nil {
		er := fmt.Errorf("Message dispatch Error - %#v", e)
		return er
	}
	return nil
}

// Distribute defines the Produce or Publisher Function for NATS Medium. User
// just needs to pass to which Pipe message needs to be passed and Message itself
// This is non blocking call. So once Publish done, client would get the control
// back for their next procedure. By default, we have defined the Consume Timeout
// as "2 Seconds". If Consumer is not able to handle the incoming message with-in,
// this timeout period, That message is lost. NATS works in "Shoot & Forget" model
func (ap *AMQPPacket) Distribute(pipe string, d interface{}) error {
	return ap.Dispatch(pipe, d)
}

// Accept defines the Subscription / Consume procedure. Again same connection would
// be used for handling all the communication with NATS as it is goroutine safe.
// Same as Request Response Model, in case of Consuming messages, we have used
// Queue Subscription to enable load balancing in NATS Server end.
func (ap *AMQPPacket) Accept(pipe string, fn Process) error {
	return nil
}

// Get would initiate a Request a Sync request from remote process. Here Pipe name
// should be the remote process name. If the Sync Request Response Facility enabled
// for HybridPipe Connection object, That would create a Channel with that Client
// process name and any other process can communicate with this client process via
// that newly created Pipe (Topic / Subject). This procedure call is a blocking call
func (ap *AMQPPacket) Get(pipe string, d interface{}) interface{} {
	return nil
}

// Remove will close a specific Subscription not the connection with NATS. This
// API should be called when user wants to just un-subscribe for some specific
// Pipes (Topic or Subject).
func (ap *AMQPPacket) Remove(pipe string) error {
	return nil
}

// Close will close NATS connection. After this call, this Object will
// become un-usable. Unexpected behavior will occur if user tries to use
// the apacket object post "Disconnect" call.
func (ap *AMQPPacket) Close() {
}
