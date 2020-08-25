package hybridpipe

// ZMQPacket defines the ZeroMQ Message Packet Object. Apart from Base Packet, it
// will contain Connection Object and identifiers related to ZeroMQ.
// 	HandleConn - ZeroMQ Connection Object
// 	PipeHandle - Create the map between Pipe name and NATS Subscription
type ZMQPacket struct {
	Packet
}

// ZMQConnect - Similar to KafkaConnect in NATS context.
func ZMQConnect(zp *ZMQPacket) error {
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

// Distribute defines the Produce or Publisher Function for NATS Medium. User
// just needs to pass to which Pipe message needs to be passed and Message itself
// This is non blocking call. So once Publish done, client would get the control
// back for their next procedure. By default, we have defined the Consume Timeout
// as "2 Seconds". If Consumer is not able to handle the incoming message with-in,
// this timeout period, That message is lost. NATS works in "Shoot & Forget" model
func (zp *ZMQPacket) Distribute(pipe string, d interface{}) error {
	return nil
}

// Accept defines the Subscription / Consume procedure. Again same connection would
// be used for handling all the communication with NATS as it is goroutine safe.
// Same as Request Response Model, in case of Consuming messages, we have used
// Queue Subscription to enable load balancing in NATS Server end.
func (zp *ZMQPacket) Accept(pipe string, fn Process) error {
	return nil
}

// Get would initiate a Request a Sync request from remote process. Here Pipe name
// should be the remote process name. If the Sync Request Response Facility enabled
// for HybridPipe Connection object, That would create a Channel with that Client
// process name and any other process can communicate with this client process via
// that newly created Pipe (Topic / Subject). This procedure call is a blocking call
func (zp *ZMQPacket) Get(pipe string, d interface{}) interface{} {
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
