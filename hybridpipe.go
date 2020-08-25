package hybridpipe

// NOTE for Maintenance: Please exceed the line limit of column 140

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
)

// Broker ID - User can use this ID to select their communication medium
// User should use this ID when calling API - "Medium"
// 	NATS     - https://nats.io/
//	KAFKA    - https://kafka.apache.org/
//	RabbitMQ - https://www.rabbitmq.com/
//	ZeroMQ   - https://zeromq.org/
//	AMQP 1.0 - https://www.amqp.org
//	MQTT     - http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html
const (
	NATS = iota
	KAFKA
	RABBITMQ
	ZEROMQ
	AMQP1
	MQTT
	NSQ
)

// HybridPipe defines the interface for HybridPipe Module
//	Distribute - publishes / broadcasts messages to defined Pipe
//	Accept - consumes the received message
//	Get - Pesudo synchronous call to get data from remote service (Works as RPC)
//	Close - closes the specified connection with Broker / Router
type HybridPipe interface {
	Dispatch(pipe string, data interface{}) error
	Distribute(pipe string, data interface{}) error
	Accept(pipe string, fn Process) error
	Get(pipe string, d interface{}) interface{}
	Remove(pipe string) error
	Close()
}

// RespondFn allows the users to response for the requests it receives from
// remote Applications / systems / micro services.
//	d - Takes the input in any format.
// This API is used for getting some specific data from remote microservice with-in the
// application system. This is applicable only for NATS router for now.
type RespondFn = func(d interface{}) interface{}

// Process defines the callback function that should be called when client
// receives the message
//	d - Take the input in any data format.
// Note: This is not applicable for TCP Mode.
type Process = func(d interface{})

// Packet defines Broker / Router which is to be used for data routing & defines
// the response function when "Get" interface is called for this service.
// 	BrokerType - Refer above defined Constants for possible values
// 	DataResponder - Refer HandleResponse Type description
type Packet struct {
	BrokerType    int
	DataResponder RespondFn
}

// Medium defines the Broker / Router to be used for the communication with optional
// parameter of Respond Function definition. This function will be applicable only
// for NATS (Only NATS supports pseudo synchronous communication)
//	br - BrokerType
//	fn - HandleRequest type (Function to handle messaging if user use this
//	     connection object as Consumer of messages from specific Pipe / Data stream.
func Medium(bt int, fn RespondFn) (HybridPipe, error) {
	switch bt {
	case NATS:
		np := new(NatsPacket)
		np.BrokerType = bt
		np.DataResponder = fn
		if e := NATSConnect(np); e != nil {
			return nil, e
		}
		return np, nil
	case KAFKA:
		kp := new(KafkaPacket)
		kp.BrokerType = bt
		kp.DataResponder = fn
		if e := KafkaConnect(kp); e != nil {
			return nil, e
		}
		return kp, nil
	case RABBITMQ:
		rp := new(RabbitPacket)
		rp.BrokerType = bt
		rp.DataResponder = fn
		if e := RabbitConnect(rp); e != nil {
			return nil, e
		}
		return rp, nil
	case ZEROMQ:
		zp := new(ZMQPacket)
		zp.BrokerType = bt
		zp.DataResponder = fn
		if e := ZMQConnect(zp); e != nil {
			return nil, e
		}
		return zp, nil
	case AMQP1:
		ap := new(AMQPPacket)
		ap.BrokerType = bt
		ap.DataResponder = fn
		if e := AMQPConnect(ap); e != nil {
			return nil, e
		}
		return ap, nil
	case MQTT:
		mp := new(MQTTPacket)
		mp.BrokerType = bt
		mp.DataResponder = fn
		if e := MQTTConnect(mp); e != nil {
			return nil, e
		}
		return mp, nil
	default:
		return nil, fmt.Errorf("Broker: \"Broker Type\" is not supported - %d", bt)
	}
}

// Encode would user defined data (To be transmitted) into Byte stream.
// Please look into Enable API
func Encode(d interface{}) ([]byte, error) {

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if e := enc.Encode(&d); e != nil {
		log.Printf("ToBytes: Encoding error. Reason : %v", e)
		return nil, e
	}
	return b.Bytes(), nil
}

// Decode would convert the received Byte stream back into user defined interface data
func Decode(d []byte, a interface{}) error {

	N := bytes.NewReader(d)
	dec := gob.NewDecoder(N)
	if e := dec.Decode(a); e != nil {
		log.Printf("ToData: Error Decoding error. Reason : %v", e)
		return e
	}

	return nil
}

// Enable would enable a specific user-defined data to be passed via HybridPipe.
// Before user calls "Encode" and "Decode", they need to enable the user-defined
// data to be learned by HybridPipe using "gob" package module
func Enable(DT interface{}) {
	gob.Register(DT)
}
