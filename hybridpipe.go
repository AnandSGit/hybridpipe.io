package hybridpipe

// -----------------------------------------------------------------------------
// IMPORT Section
// -----------------------------------------------------------------------------
import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
)

// Middleware ID would be used to define / declare which Middleware would be used
// for communication.
// 	NATS     - Messaging system completely written in Go. Shoot and forget model
//	KAFKA    - Typically used for large data / file transfers
//	RbbbitMQ - Lot of Legacy Java based systems developed using RabbitMQ
//	ZeroMQ   - Very light-weight Communication system. Work in Progress
//	TCP      - Basic TCP Sockets for communication. Work in Progress
// When user is calling "Router" API, He / She needs to specify which Middleware
// to be used for communication. That input should come from one of this ID
const (
	NATS = iota
	KAFKA
	RABBITMQ
	ZEROMQ
	TCP
)

// HybridPipe Interface defines the Process interface function (Only function user
// should call). These functions are implemented in Router struct family.
//	Distribute - API to Publish Messages into specified Pipe (Topic / Subject)
//	Accept - Consume the incoming message if subscribed by that component
//	Get - Would initiate blocking call to remote process to get response
//	Close - Would disconnect the connection with Middleware.
type HybridPipe interface {
	Distribute(pipe string, data interface{}) error
	Accept(pipe string, fn Process) error
	Get(pipe string, d interface{}) interface{}
	Remove(pipe string) error
	Close()
}

// RespondFn allows the users to response for the requests it receives from
// remote Applications / systems / micro services.
//	d - Takes the input in any format.
// Response can be in any data format. This data would be derived from the data
// received as parameter in req. We recommend JSON format to be defined for request
// so that it can be parsed with ease for generating response.
// Note: This feature is applicable only for NATS, ZeroMQ and TCP.
type RespondFn = func(d interface{}) interface{}

// Process would handle the received messages, which are subscribed by the
// user.
//	d - Take the input in any data format.
// This function (Callback) will be called whenever there is a message received
// by dRouter lib. User should define this function and implement to process the
// incoming data / message.
// Note: This is not applicable for TCP Mode.
type Process = func(d interface{})

// Packet defines all the message related information that Producer or Consumer
// should know for message transactions. Both Producer and Consumer use this same
// structure for message transactions.
// 	BrokerType - Refer above defined Constants for possible values
// 	DataResponder - Refer HandleResponse Type description
type Packet struct {
	BrokerType    int
	DataResponder RespondFn
}

// Medium will enable user to define which Communication platform to be used from
// the list of supported Middlewares. It will also create / initiate the communication
//	br - BrokerType
//	fn - HandleRequest type (Function to handle messaging if user use this
//	     connection object as Consumer of messages from specific Pipe / Data stream.
//	Response - Would respond with specific Middeware connection Object
// TODO: ZeroMQ and TCP implementations under progress
func Medium(bt int, fn RespondFn) (HybridPipe, error) {

	var np *NatsPacket
	var kp *KafkaPacket
	var rp *RabbitPacket

	switch bt {
	case NATS:
		np = new(NatsPacket)
		np.BrokerType = bt
		np.DataResponder = fn
		if e := NATSConnect(np); e != nil {
			return nil, e
		}
		return np, nil
	case KAFKA:
		kp = new(KafkaPacket)
		kp.BrokerType = bt
		kp.DataResponder = fn
		if e := KafkaConnect(kp); e != nil {
			return nil, e
		}
		return kp, nil
	case RABBITMQ:
		rp = new(RabbitPacket)
		rp.BrokerType = bt
		rp.DataResponder = fn
		if e := RabbitConnect(rp); e != nil {
			return nil, e
		}
		return rp, nil
	default:
		return nil, fmt.Errorf("Broker: \"Broker Type\" is not supported - %d", bt)
	}
}

// Encode would convert the incoming data into Bytes. Please refer "Enable" API
func Encode(d interface{}) ([]byte, error) {

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if e := enc.Encode(&d); e != nil {
		log.Printf("ToBytes: Encoding error. Reason : %v", e)
		return nil, e
	}
	return b.Bytes(), nil
}

// Decode would convert the incoming Bytes into data. Please refer "Enable" API
func Decode(d []byte, a interface{}) error {

	N := bytes.NewReader(d)
	dec := gob.NewDecoder(N)
	if e := dec.Decode(a); e != nil {
		log.Printf("ToData: Error Decoding error. Reason : %v", e)
		return e
	}

	return nil
}

// Enable would enable a specific user-defined data to be passed via dRouter.
// Before user calls "ToBytes" and "ToData", they need to enable the user-defined
// data to be learned by dRouter system using "gob" package module
func Enable(DT interface{}) {
	gob.Register(DT)
}
