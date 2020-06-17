package hybridpipe

// -----------------------------------------------------------------------------
// IMPORT Section
// -----------------------------------------------------------------------------
import (
	"fmt"
	"log"
	"os"
	"time"

	PS "github.com/mitchellh/go-ps"
	nats "github.com/nats-io/nats.go"
)

// NatsPacket defines the NATS Message Packet Object. Apart from Base Packet, it
// will contain NATS Connection Object and identifiers related to NATS.
// 	HandleConn - NATS Connection Object
// 	PipeHandle - Create the map between Subject name and NATS Subscription
type NatsPacket struct {
	Packet
	HandleConn *nats.Conn
	PipeHandle map[string]*nats.Subscription
}

// NATSConnect defines the connection procedure for NATS Server. This API would
// access NATS related configurations from the specified configuration file and
// load the same into MQF struct object. NATS uses the TLS Certificates and Keys
// for Authentication and Encryption. TLS object handling is done inside NATS. So
// We just pass only the files to NATS. Once connection with NATS established, we
// would initate the Responder function if user wants to respond for Sync queries
// comes from different subsystems.
func NATSConnect(np *NatsPacket) error {

	// Create both NatsF and NatsPacket Objects. NatsF will be used to store
	// all config information including Server URL, Port, User credentials
	// and other configuration information, which is for Future Expansion.
	var mq = new(MQF)

	// Configuration File read and updating NatsF Object.
	ReadConfig(&mq)

	// Using NatsF details, connecting to the NATS Server. Unlike Kafka or RabbitMQ
	// NATS accepts the Client Cert, CA and Key files directly instead of TLS object
	nc, e := nats.Connect(mq.NServer,
		nats.Secure(),
		nats.ClientCert(mq.NATSCertFile, mq.NATSKeyFile),
		nats.RootCAs(mq.NATSCAFile),
	)
	if e != nil {
		er := fmt.Errorf("NATS Connect Error: %#v, %s", e, mq.NServer)
		return er
	}

	// Store the NATS connection in the Packet Object
	np.HandleConn = nc

	// Initialize the PipeHandle Map
	if np.PipeHandle == nil {
		np.PipeHandle = make(map[string]*nats.Subscription)
	}

	// We are initializing the ResponseHandler API
	np.initResponder()
	return nil
}

// initResponder defines the implicit local function that would respond for any
// incoming "Get" requests. The Pipe name defined for the subscription would be
// local process name. Because we use Queue Subscription, Load balancing would be
// handled from NATS end. So even if all the instances of this application
// running in parallel in different / same nodes, only one of the instance would
// really receive this Get calls. This function would give complete control to
// the user on how they wants to handle their request and response data. The Request
// and Response data types and formats should be decided by Interface definition
// between those 2 systems, those uses HybridPipe for communication.
func (np *NatsPacket) initResponder() error {

	myPID := os.Getpid()
	pName, _ := PS.FindProcess(myPID)

	// Subscribe for the Requests coming to the current running process.
	if _, e := np.HandleConn.QueueSubscribe(pName.Executable(), pName.Executable(), func(m *nats.Msg) {

		// Accept the incoming message in bytes. Decode the data and call the
		// client / user provided Response handling Callback. Get the Callback
		// response Encode the data again and respond for the same NATS message
		// object. This callback doesn't return any data. So handle all the errors
		// inside this inline callback.
		var idata interface{}
		Decode(m.Data, &idata)

		if np.DataResponder == nil {
			log.Printf("response object is not initialized with full capacity by the user")
			return
		}
		d := np.DataResponder(idata)

		// Encode the response before replying
		b, er := Encode(d)
		if er != nil {
			log.Printf("%v", er)
			return
		}

		// Use the message object to send the respond
		if er = m.Respond(b); er != nil {

			log.Printf("%v", er)
			return
		}
	}); e != nil {

		log.Printf("%v", e)
		return e
	}
	return nil
}

// Distribute defines the Produce or Publisher Function for NATS Medium. User
// just needs to pass to which Pipe message needs to be passed and Message itself
// This is non blocking call. So once Publish done, client would get the control
// back for their next procedure. By default, we have defined the Consume Timeout
// as "2 Seconds". If Consumer is not able to handle the incoming message with-in,
// this timeout period, That message is lost. NATS works in "Shoot & Forget" model
func (np *NatsPacket) Distribute(pipe string, d interface{}) error {

	// Encode the message
	b, e := Encode(d)
	if e != nil {
		log.Printf("%v", e)
		return e
	}

	// Distribute / Publish / Produce the message to the specified Pipe
	if e = np.HandleConn.Publish(pipe, b); e != nil {
		log.Printf("%v", e)
		return e
	}
	np.HandleConn.Flush()
	return nil
}

// Accept defines the Subscription / Consume procedure. Again same connection would
// be used for handling all the communication with NATS as it is goroutine safe.
// Same as Request Response Model, in case of Consuming messages, we have used
// Queue Subscription to enable load balancing in NATS Server end.
func (np *NatsPacket) Accept(pipe string, fn Process) error {

	// Queue subscribe for message to enable Load balancing as well.
	s, e := np.HandleConn.QueueSubscribe(pipe, os.Args[0], func(m *nats.Msg) {

		var d interface{}
		if e := Decode(m.Data, &d); e != nil {
			log.Printf("%v", e)
			return
		}
		fn(d)
	})
	if e != nil {
		log.Printf("%v", e)
		return e
	}

	np.PipeHandle[pipe] = s
	return nil
}

// Get would initiate a Request a Sync request from remote process. Here Pipe name
// should be the remote process name. If the Sync Request Response Facility enabled
// for HybridPipe Connection object, That would create a Channel with that Client
// process name and any other process can communicate with this client process via
// that newly created Pipe (Topic / Subject). This procedure call is a blocking call
func (np *NatsPacket) Get(pipe string, d interface{}) interface{} {

	var data interface{}

	//Encode the message
	b, e := Encode(d)
	if e != nil {
		log.Printf("%v", e)
		return e
	}

	// Calling the Request API from NATS. Response would be stored in the an
	// interface and same would be passed back to the caller
	response, er := np.HandleConn.Request(pipe, b, 4*time.Second)
	if er != nil {
		log.Printf("%v", er)
		return er
	}

	// Decode the response from BYTE stream and pass the same back to caller
	if er = Decode(response.Data, &data); e != nil {
		log.Printf("%v", er)
		return er
	}
	return data
}

// Remove will close a specific Subscription not the connection with NATS. This
// API should be called when user wants to just un-subscribe for some specific
// Pipes (Topic or Subject).
func (np *NatsPacket) Remove(pipe string) error {

	es, ok := np.PipeHandle[pipe]
	if ok == false {
		e := fmt.Errorf("specified pipe is not subscribed yet. please check the pipe name passed")
		return e
	}
	if e := es.Unsubscribe(); e != nil {
		return e
	}
	return nil
}

// Close will close NATS connection. After this call, this Object will
// become un-usable. Unexpected behavior will occur if user tries to use
// the NPacket object post "Disconnect" call.
func (np *NatsPacket) Close() {

	np.HandleConn.Close()
}
