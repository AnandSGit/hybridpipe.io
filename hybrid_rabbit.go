package hybridpipe

import (
	"log"

	ampq "github.com/streadway/amqp"
)

// RabbitPacket - Please refer KafkaPacket
type RabbitPacket struct {
	HandleConn *ampq.Connection
	RChannel *ampq.Channel
}

// Connect defines the connection procedure for RabbitMQ Server.
func (rp *RabbitPacket) Connect() error {

	rp.HandleConn, e := ampq.Dial(HPipeConfig.RServerPort)
	if e != nil {
		log.Printf("%v", e)
		return e
	}
	return nil
}

// Dispatch defines the Produce or Publisher Function for RabbitMQ Medium.
func (rp *RabbitPacket) Dispatch(pipe string, d interface{}) error {
	var q ampq.Queue
	var e error

	rp.RChannel, _ = rp.HandleConn.Channel()
	if q, e = rp.RChannel.QueueDeclare(pipe, false, false, false, false, nil); e != nil {
		log.Printf("%v", e)
		return e
	}
	b, ei := Encode(d)
	if ei != nil {
		log.Printf("%v", ei)
		return ei
	}
	e = rp.RChannel.Publish("", q.Name, false, false, ampq.Publishing{
		ContentType: "text/plain",
		Body:        b,
	})
	return nil
}

// Accept defines the Subscription / Consume procedure.
func (rp *RabbitPacket) Accept(pipe string, fn Process) error {
	
	var q ampq.Queue

	rp.RChannel, e := rp.HandleConn.Channel()
	if e != nil {
		log.Printf("%v", e)
		return e
	}
	if q, e = rp.RChannel.QueueDeclare(pipe, false, false, false, false, nil); e != nil {
		log.Printf("%v", e)
		return e
	}
	m, ei := rp.RChannel.Consume(q.Name, pipe, true, false, false, false, nil)
	if ei != nil {
		log.Printf("%v", e)
		return e
	}
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

// Remove would cancel the Queue name gracefully so that it doesn't affect the
// goroutine which is already handling messages from Producer side of this Pipe
func (rp *RabbitPacket) Remove(pipe string) error {

	if e := rp.RChannel.Cancel(pipe, false); e != nil {
		return e
	}
	return nil
}

// Close will close RabbitMQ connection.
func (rp *RabbitPacket) Close() {
	rp.HandleConn.Close()
}
