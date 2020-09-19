package hybridpipe

import (
	"fmt"
	"log"
	"os"

	nats "github.com/nats-io/nats.go"
)

// NatsPacket defines the NATS Message Packet Object. Apart from Base Packet, it
// will contain NATS Connection Object and identifiers related to NATS.
// 	HandleConn - NATS Connection Object
// 	PipeHandle - Create the map between Subject name and NATS Subscription
type NatsPacket struct {
	HandleConn *nats.Conn
	PipeHandle map[string]*nats.Subscription
}

// Connect - Similar to KafkaConnect in NATS context.
func (np *NatsPacket) Connect() error {
	nc, e := nats.Connect(HPipeConfig.NServer)
	/**
	nc, e := nats.Connect(HPipeConfig.NServer,
		nats.Secure(),
		nats.ClientCert(HPipeConfig.NATSCertFile, HPipeConfig.NATSKeyFile),
		nats.RootCAs(HPipeConfig.NATSCAFile),
	)
	**/
	if e != nil {
		er := fmt.Errorf("NATS Connect Error: %#v, %s", e, HPipeConfig.NServer)
		return er
	}
	np.HandleConn = nc
	if np.PipeHandle == nil {
		np.PipeHandle = make(map[string]*nats.Subscription)
	}
	return nil
}

// Dispatch defines the Produce or Publisher Function for NATS Medium. User
// just needs to pass to which Pipe message needs to be passed and Message itself
// This is non blocking call. So once Publish done, client would get the control
// back for their next procedure. By default, we have defined the Consume Timeout
// as "2 Seconds". If Consumer is not able to handle the incoming message with-in,
// this timeout period, That message is lost. NATS works in "Shoot & Forget" model
func (np *NatsPacket) Dispatch(pipe string, d interface{}) error {
	// Encode the message
	b, e := Encode(d)
	if e != nil {
		log.Printf("%v", e)
		return e
	}
	// Distribute / Publish / Produce the message to the specified Pipe
	if e = np.HandleConn.Publish(pipe, b); e != nil {
		log.Printf("%v", e)
		return e
	}
	np.HandleConn.Flush()
	return nil
}

// Accept defines the Subscription / Consume procedure. Again same connection would
// be used for handling all the communication with NATS as it is goroutine safe.
// Same as Request Response Model, in case of Consuming messages, we have used
// Queue Subscription to enable load balancing in NATS Server end.
func (np *NatsPacket) Accept(pipe string, fn Process) error {
	// Queue subscribe for message to enable Load balancing as well.
	s, e := np.HandleConn.QueueSubscribe(pipe, os.Args[0], func(m *nats.Msg) {
		var d interface{}
		if e := Decode(m.Data, &d); e != nil {
			log.Printf("%v", e)
			return
		}
		fn(d)
	})
	if e != nil {
		log.Printf("%v", e)
		return e
	}

	np.PipeHandle[pipe] = s
	return nil
}

// Remove will close a specific Subscription not the connection with NATS. This
// API should be called when user wants to just un-subscribe for some specific
// Pipes (Topic or Subject).
func (np *NatsPacket) Remove(pipe string) error {
	es, ok := np.PipeHandle[pipe]
	if ok == false {
		e := fmt.Errorf("specified pipe is not subscribed yet. please check the pipe name passed")
		return e
	}
	if e := es.Unsubscribe(); e != nil {
		return e
	}
	return nil
}

// Close will close NATS connection. After this call, this Object will
// become un-usable. Unexpected behavior will occur if user tries to use
// the NPacket object post "Disconnect" call.
func (np *NatsPacket) Close() {
	np.HandleConn.Close()
}
