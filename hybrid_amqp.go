package hybridpipe

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/go-amqp"
)

// AMQPPacket defines the Qpid Message Packet
// 	HandleConn - AMQP Connection Object
type AMQPPacket struct {
	HandleConn *amqp.Client
}

// Connect - Defines the procedure to connect to AMQP Qpid server.
func (ap *AMQPPacket) Connect() error {
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

// Dispatch would send a user defined message using Qpid server
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

// Accept defines the Subscription / Consume procedure. The Message processing
// will be done in separate Go routine.
func (ap *AMQPPacket) Accept(pipe string, fn Process) error {
	session, e := ap.HandleConn.NewSession()
	if e != nil {
		er := fmt.Errorf("AMQP Accept Session Error - %#v", e)
		return er
	}
	receiver, e := session.NewReceiver(
		amqp.LinkSourceAddress(pipe),
		amqp.LinkCredit(10),
	)
	if e != nil {
		er := fmt.Errorf("Creating Receiver Link - FAILED : %#v", e)
		return er
	}
	go ap.read(receiver, fn)
	return nil
}

// Get the message using receiver call.
func (ap *AMQPPacket) read(r *amqp.Receiver, fn Process) error {
	ctx := context.Background()
	var d interface{}

	defer r.Close(ctx)
	for {
		m, e := r.Receive(ctx)
		if e != nil {
			er := fmt.Errorf("Message Receive Error - %#v", e)
			return er
		}
		m.Accept(ctx)
		if e = Decode(m.GetData(), &d); e != nil {
			return e
		}
		fn(d)
	}
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
