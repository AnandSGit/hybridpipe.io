package hybridpipe

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
type HybridPipe interface {
	Connect() error
	Dispatch(pipe string, data interface{}) error
	Accept(pipe string, fn Process) error
	Remove(pipe string) error
	Close()
}

// Process defines the callback function that should be called when client
// receives the message
//	d - Take the input in any data format.
// Note: This is not applicable for TCP Mode.
type Process = func(d interface{})

// DeployRouter defines the Broker / Router to be used for the communication
func DeployRouter(bt int) (HybridPipe, error) {

	p := reflect.New(RoutersMap[bt]).Interface().(HybridPipe)
	if e := p.Connect(); e != nil {
		return nil, e
	}
	return p, nil
}

// Encode would user defined data (To be transmitted) into Byte stream.
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
func Enable(DT interface{}) {
	gob.Register(DT)
}
