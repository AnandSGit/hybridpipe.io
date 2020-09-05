package hybridpipe

// ZMQPacket defines the ZeroMQ Message Packet Object. Apart from Base Packet, it
// will contain Connection Object and identifiers related to ZeroMQ.
// 	HandleConn - ZeroMQ Connection Object
// 	PipeHandle - Create the map between Pipe name and NATS Subscription
type ZMQPacket struct {
}

// Connect - Similar to KafkaConnect in NATS context.
func (zp *ZMQPacket) Connect() error {
	return nil
}

// initResponder defines the implicit local function that would respond for any incoming "Get" requests.
// The Pipe name defined for the subscription would be local process name. Because we use Queue Subscription,
// Load balancing would be handled from NATS end. So even if all the instances of this application running
// in parallel in different / same nodes, only one of the instance would really receive this Get calls.
// This function would give complete control to the user on how they wants to handle their request and
// response data. The Request and Response data types and formats should be decided by Interface definition
// between those 2 systems, those uses HybridPipe for communication.
func (zp *ZMQPacket) initResponder() error {
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
