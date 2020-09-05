package hybridpipe

// NOTE for Maintenance: Please exceed the line limit of column 140

import (
	"bytes"
	"encoding/gob"
	"log"
	"reflect"
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
)

// RoutersMap defines Broker or Router Registry for Pipe creation
var RoutersMap = make(map[int]reflect.Type)

// Initialize the Register Map with all supported interfaces
func init() {
	RoutersMap[NATS] = reflect.TypeOf((*NatsPacket)(nil)).Elem()
	RoutersMap[KAFKA] = reflect.TypeOf((*KafkaPacket)(nil)).Elem()
	RoutersMap[RABBITMQ] = reflect.TypeOf((*RabbitPacket)(nil)).Elem()
	RoutersMap[ZEROMQ] = reflect.TypeOf((*ZMQPacket)(nil)).Elem()
	RoutersMap[AMQP1] = reflect.TypeOf((*AMQPPacket)(nil)).Elem()
	RoutersMap[MQTT] = reflect.TypeOf((*MQTTPacket)(nil)).Elem()
}

// HybridPipe defines the interface for HybridPipe Module
//	Connect - Would connect with right Router backend
//	Dispatch - publishes / broadcasts messages to defined Pipe
//	Accept - consumes the received message
//	Get - Pesudo synchronous call to get data from remote service (Works as RPC)
//	Close - closes the specified connection with Broker / Router
type HybridPipe interface {
	Connect() error
	Dispatch(pipe string, data interface{}) error
	Accept(pipe string, fn Process) error
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

// DeployRouter defines the Broker / Router to be used for the communication with optional
// parameter of Respond Function definition. This function will be applicable only
// for NATS (Only NATS supports pseudo synchronous communication)
//	br - BrokerType
//	fn - HandleRequest type (Function to handle messaging if user use this
//	     connection object as Consumer of messages from specific Pipe / Data stream.
func DeployRouter(bt int, fn RespondFn) (HybridPipe, error) {
	var p HybridPipe = reflect.New(RoutersMap[bt]).Interface().(HybridPipe)
	if e := p.Connect(); e != nil {
		return nil, e
	}
	return p, nil
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
