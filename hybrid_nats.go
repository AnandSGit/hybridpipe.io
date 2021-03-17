package hybridpipe

import (
	"fmt"
	"log"
	"os"

	nats "github.com/nats-io/nats.go"
)

// NatsPacket defines the NATS Message Packet Object.
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

// Dispatch defines the Produce or Publisher Function for NATS Medium.
func (np *NatsPacket) Dispatch(pipe string, d interface{}) error {

	b, e := Encode(d)
	if e != nil {
		log.Printf("%v", e)
		return e
	}
	if e = np.HandleConn.Publish(pipe, b); e != nil {
		log.Printf("%v", e)
		return e
	}
	np.HandleConn.Flush()
	return nil
}

// Accept defines the Subscription / Consume procedure.
func (np *NatsPacket) Accept(pipe string, fn Process) error {

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

// Remove will close a specific Subscription not the connection with NATS.
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

// Close will close NATS connection.
func (np *NatsPacket) Close() {
	np.HandleConn.Close()
}
