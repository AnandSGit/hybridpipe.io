package hybridpipe

// -----------------------------------------------------------------------------
// IMPORT Section
// -----------------------------------------------------------------------------
import (
	"log"

	ampq "github.com/streadway/amqp"
)

// RabbitPacket defines the RabbitMQ Message Packet Object. Apart from Base Packet, it
// will contain RabbitMQ Connection Object and identifiers related to RMQ. The Main
// connection handle and Channel handle would be stored as part of RabbitPacket to
// enable individual Channel Un-subscription operations from already existing or
// channel based on Pipe name.
type RabbitPacket struct {

	// Packet Include all the common parameters across KAFKA and NATS. BrokerType,
	// ActionType and Pipe would be mandatory. But in case of MsgProcess Handler
	// Function, this function reference would be passed to this object only when
	// ActionType is SUBSCRIBE.  Otherwise this function reference can be passed as "nil".
	Packet

	// HandleConn holds the connection object to RabbitMQ Server
	HandleConn *ampq.Connection

	// RChannel holds the RabbitMQ Connection Object locally. Single connection would
	// be used for handling all the connections from any single process that includes
	// responding for requests.
	RChannel *ampq.Channel
}

// RabbitConnect defines the connection procedure for RabbitMQ Server. As part
// of Connect procedure we would be reading the RabbitMQ configuration from the
// Config handler and would dial the connection towards RabbitMQ Server.
func RabbitConnect(rp *RabbitPacket) error {

	// Read the RabbitMQ Server information from Config File
	var e error
	var mq = new(MQF)
	ReadConfig(&mq)

	// Get the Connection Handle to RabbitMQ and Store in Packet Object
	rp.HandleConn, e = ampq.Dial(mq.RServerPort)
	if e != nil {
		log.Printf("%v", e)
		return e
	}
	return nil
}

// Distribute defines the Produce or Publisher Function for RabbitMQ Medium. User just
// needs to pass to which Pipe message needs to be passed and Message itself.
func (rp *RabbitPacket) Distribute(pipe string, d interface{}) error {

	var q ampq.Queue
	var e error

	// Get Channel object for RabbitMQ Connection. Ignoring the errors for now.
	rp.RChannel, _ = rp.HandleConn.Channel()

	// Declare the Queue based on Pipe name provided for this API
	if q, e = rp.RChannel.QueueDeclare(pipe, false, false, false, false, nil); e != nil {
		log.Printf("%v", e)
		return e
	}

	// Encode the message before appending into KAFKA Message struct
	b, ei := Encode(d)
	if ei != nil {
		log.Printf("%v", ei)
		return ei
	}

	// Send the message to the specified Pipe
	e = rp.RChannel.Publish("", q.Name, false, false, ampq.Publishing{
		ContentType: "text/plain",
		Body:        b,
	})
	return nil
}

// Accept defines the Subscription / Consume procedure. Again same connection would
// be used for handling all the communication with RabbitMQ as it is goroutine safe.
func (rp *RabbitPacket) Accept(pipe string, fn Process) error {

	var q ampq.Queue
	var e error

	// Getting the Consume Channel
	rp.RChannel, e = rp.HandleConn.Channel()
	if e != nil {
		log.Printf("%v", e)
		return e
	}

	// Define the Queue with Pipe name passed.
	if q, e = rp.RChannel.QueueDeclare(pipe, false, false, false, false, nil); e != nil {
		log.Printf("%v", e)
		return e
	}

	// Initiate the consume channel. Returned "m" is channel.
	m, ei := rp.RChannel.Consume(q.Name, pipe, true, false, false, false, nil)
	if ei != nil {
		log.Printf("%v", e)
		return e
	}

	// Initiate a infinite loop in reading incoming messages.
	inFinity := make(chan bool)
	go func() {
		for ms := range m {
			var rm interface{}
			if e := Decode(ms.Body, &rm); e != nil {
				log.Printf("%v", e)
				return
			}
			fn(rm)
		}
	}()
	<-inFinity
	return nil
}

// Get would initiate a Request from this object. Not supported in RabbitMQ now
func (rp *RabbitPacket) Get(pipe string, d interface{}) interface{} {

	return nil
}

// Remove would cancel the Queue name gracefully so that it doesn't affect the
// goroutine which is already handling messages from Producer side of this Pipe
func (rp *RabbitPacket) Remove(pipe string) error {

	if e := rp.RChannel.Cancel(pipe, false); e != nil {
		return e
	}
	return nil
}

// Close will close RabbitMQ connection. After this call, this Object will
// become un-usable. Unexpected behavior will occur if user tries to use
// the NPacket object post "Disconnect" call.
func (rp *RabbitPacket) Close() {

	rp.HandleConn.Close()
}
