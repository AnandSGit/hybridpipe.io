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

// initResponder defines the implicit local function that would respond for any incoming "Get" requests.
// No Implementation required.
func (mp *MQTTPacket) initResponder() error {
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

// Get would initiate a Request a Sync request from remote process.
func (mp *MQTTPacket) Get(pipe string, d interface{}) interface{} {
	return nil
}

// Remove will close a specific Subscription not the connection with NATS. This
// API should be called when user wants to just un-subscribe for some specific
// Pipes (Topic or Subject).
func (mp *MQTTPacket) Remove(pipe string) error {
	return nil
}

// Close will close NATS connection. After this call, this Object will
// become un-usable. Unexpected behavior will occur if user tries to use
// the mpacket object post "Disconnect" call.
func (mp *MQTTPacket) Close() {

}
