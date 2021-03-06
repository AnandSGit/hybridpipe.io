/*
Package hybridpipe Enables communication between 2 micro services / individual processes via
Brokers or Routers like KAFKA / NATS / RabbitMQ (AMQP) / NSQ / ZeroMQ / default NET package
from Golang. This system provides common interfacing APIs for all above mentioned Routers.
Based on user requirement, user can select the Router/Broker as underlying Message Q. Sample
applications are implemented to behave as HybridPipe Producer and Consumer.

NOTE: ONLY KAFKA, NATS and RABBITMQ interfacing are released. Will be updating the notes once
other brokers support is added.

Usage Example:

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
*/
package hybridpipe
