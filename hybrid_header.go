/*

Package hybridpipe facilitates communication between different individual processes
or microservices or components. It allows user to select the underlying communication
mediums. For example, user can select one of Kafka, NATS and RabbitMQ as the main
underlying communication middleware.

This component provides the basic building block for enabling communication by allowing
users / clients to send any user defined information over to the remote applications
or systems

This system also enables clients or users to make synchronous calls to the remote
running microservices and get the response as though it is getting response from
local function call. This feature is enabled only for subsystems which uses NATS
as communication middleware. This can be called as "Pseudo Synchronous call".

There are two small sample applications implemented to behave as HybridPipe Producer and
Consumer. Both are existing under "hybridpipe/hybridproducer" and "hybridpipe/hybridcomsumer"
directories defined under "hybridpipe".

Usage Example

NATS Producer Client :
	dc.Enable(Person{})
	N, _ := dc.Medium(dc.NATS, nil)		        // This obj won't handle Sync call
	defer N.Close()
	N.Distribute("Apps.iLO.Low", P)		// P object of Person{}
	B := N.Get("EventServer", "{userid:43245}")	// Response would be stored in B. This is blocking call

NATS Consumer Client :
	dc.Enable(Person{})
	N, _ := dc.Medium(dc.NATS, RespondHandler)      // Respond will be handled in this API
	N.Accept("Apps.iLO.Low", NatsHandler)          // Incoming Message would be handled in "NatsHandler" API in
	client side

KAFKA Producer Client :
	dc.Enable(Person{})
	K, _ := dc.Medium(dc.KAFKA, nil)                // This obj won't handle Sync call
	defer K.Close()
	K.Distribute("Apps.iLO.Med", P)                // P object of Person{}

KAFKA Consumer Client :
	dc.Enable(Person{})
	K, _ := dc.Medium(dc.KAFKA, nil)                // KAFKA doesn't support Sync calls
	N.Accept("Apps.iLO.Med", KAfkaHandler)         // Incoming Message would be handled in "KafkaHandler" API in
	client side

RABBITMQ Producer Client :
	R, _ := dc.Medium(dc.RABBITMQ, nil)
	defer R.Close()
	R.Distribute("Apps.iLO.Med", jd)               // Sending JSON Content

RABBITMQ Consumer Client :
	R, _ := dc.Medium(dc.RABBITMQ, nil)
	defer R.Close()
	R.Accept("Apps.iLO.Med", RabbitHandler)        // Incoming Message would be handled in "RabbitHandler" API in
	client side

*/
package hybridpipe
