package hybridpipe

// MQTTPacket defines the ZeroMQ Message Packet Object. Apart from Base Packet, it
// will contain Connection Object and identifiers related to ZeroMQ.
// 	HandleConn - ZeroMQ Connection Object
// 	PipeHandle - Create the map between Pipe name and NATS Subscription
type MQTTPacket struct {
}

// Connect - Similar to KafkaConnect in NATS context.
func (mp *MQTTPacket) Connect() error {
	return nil
}

// Dispatch will be implemented only for AMQP 1.0 medium
func (mp *MQTTPacket) Dispatch(pipe string, d interface{}) error {
	return nil
}

// Accept defines the Subscription / Consume procedure.
func (mp *MQTTPacket) Accept(pipe string, fn Process) error {
	return nil
}

// Remove will close a specific Subscription.
func (mp *MQTTPacket) Remove(pipe string) error {
	return nil
}

// Close will close NATS connection.
func (mp *MQTTPacket) Close() {

}
