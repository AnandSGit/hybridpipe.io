package hybridpipe

import (
	"github.com/pebbe/zmq4"
)

// ZMQPacket defines the ZeroMQ Message Packet Object. Apart from Base Packet, it
// will contain Connection Object and identifiers related to ZeroMQ.
// 	rMQ - ZeroMQ - Data Sending Channel Stream
// 	dMQ - Data Receiving Channel Stream
type ZMQPacket struct {
	dConn *zmq4.Socket
	aConn *zmq4.Socket
}

// Connect - Similar to KafkaConnect in NATS context.
func (zp *ZMQPacket) Connect() error {

	return nil
}

// Dispatch would be implemented only for AMQP 1.0 medium
func (zp *ZMQPacket) Dispatch(pipe string, d interface{}) error {

	return nil
}

// Accept defines the Subscription / Consume procedure. Again same connection would
// be used for handling all the communication with NATS as it is goroutine safe.
// Same as Request Response Model, in case of Consuming messages, we have used
// Queue Subscription to enable load balancing in NATS Server end.
func (zp *ZMQPacket) Accept(pipe string, fn Process) error {

	return nil
}

// Remove will close a specific Subscription not the connection with NATS. This
// API should be called when user wants to just un-subscribe for some specific
// Pipes (Topic or Subject).
func (zp *ZMQPacket) Remove(pipe string) error {
	return nil
}

// Close will close NATS connection. After this call, this Object will
// become un-usable. Unexpected behavior will occur if user tries to use
// the NPacket object post "Disconnect" call.
func (zp *ZMQPacket) Close() {

}
