package hybridpipe

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaPacket defines the data for handling KAFKA connections for Broker function.
// 	Packet - defines the backbone broker and response handling function. (Nil)
//	Readers - map between Pipe name and Kafka Reader objects
// 	Writers - map between Pipe name and Kafka writer objects
//	DialerConn - Kafka Connection Object
//	Server - KAFKA Server Target name
type KafkaPacket struct {
	Readers    map[string]*kafka.Reader
	Writers    map[string]*kafka.Writer
	DialerConn *kafka.Dialer
	Server     string
}

// TLS creates the TLS configuration objects to be used by KAFKA (For both
// Auth and Encryption)
func TLS(cCert, cKey, caCert string) (*tls.Config, error) {

	tlsConfig := tls.Config{}

	// Load client cert
	cert, e1 := tls.LoadX509KeyPair(cCert, cKey)
	if e1 != nil {
		return &tlsConfig, e1
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	// Load CA cert
	caCertR, e2 := ioutil.ReadFile(caCert)
	if e2 != nil {
		return &tlsConfig, e2
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertR)
	tlsConfig.RootCAs = caCertPool

	tlsConfig.BuildNameToCertificate()
	return &tlsConfig, e2
}

// Connect creates the Kafka connection & both Reader and Writer stream objects
// based on the pipe name
func (kp *KafkaPacket) Connect() error {
	kp.Server = HPipeConfig.KServer + ":" + strconv.Itoa(HPipeConfig.KLport)
	tls, e := TLS(HPipeConfig.KAFKACertFile, HPipeConfig.KAFKAKeyFile, HPipeConfig.KAFKACAFile)
	if e != nil {
		log.Printf("%v", e)
		return e
	}
	kp.DialerConn = &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS:       tls,
	}
	if kp.Readers == nil {
		kp.Readers = make(map[string]*kafka.Reader)
	}
	if kp.Writers == nil {
		kp.Writers = make(map[string]*kafka.Writer)
	}
	return nil
}

// Dispatch will be implemented only for AMQP 1.0 medium
func (kp *KafkaPacket) Dispatch(pipe string, d interface{}) error {
	return nil
}

// Distribute defines the Producer / Publisher role and functionality. Writer would be
// created for each Pipe comes-in for communication. If Writer already exists, that connection
// would be used for this call. Before publishing the message in the specified Pipe, it will be
// converted into Byte stream using "Encode" API. Encryption is enabled for the message via TLS.
func (kp *KafkaPacket) Distribute(pipe string, d interface{}) error {
	// Check for existing Writers. If not existing for this specific Pipe,
	// then we would create this Writer object for sending the message.
	if _, a := kp.Writers[pipe]; a == false {

		kp.Writers[pipe] = kafka.NewWriter(kafka.WriterConfig{
			Brokers:       []string{kp.Server},
			Topic:         pipe,
			Balancer:      &kafka.LeastBytes{},
			BatchSize:     1,
			QueueCapacity: 1,
			Async:         true,
			Dialer:        kp.DialerConn,
		})
	}
	b, e := Encode(d)
	if e != nil {
		log.Printf("%v", e)
		return e
	}
	km := kafka.Message{
		Key:   []byte(pipe),
		Value: b,
	}
	if e = kp.Writers[pipe].WriteMessages(context.Background(), km); e != nil {
		log.Printf("%v", e)
		return e
	}
	return nil
}

// Accept function defines the Consumer or Subscriber functionality for KAFKA. If Reader object
// for the specified Pipe is not available, New Reader Object would be created. From this
// function Goroutine "Read" will be invoked to handle the incoming messages.
func (kp *KafkaPacket) Accept(pipe string, fn Process) error {
	// If for the Reader Object for pipe and create one if required.
	if _, a := kp.Readers[pipe]; a == false {
		kp.Readers[pipe] = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        []string{kp.Server},
			GroupID:        pipe,
			Topic:          pipe,
			MinBytes:       10e1,
			MaxBytes:       10e6,
			CommitInterval: 1 * time.Second,
			Dialer:         kp.DialerConn,
		})
	}
	go kp.Read(pipe, fn)
	return nil
}

// Read would access the KAFKA messages in a infinite loop.
func (kp *KafkaPacket) Read(p string, fn Process) error {
	// This interface should be defined outside the inner level to make sure
	// we are making the ToData API to work.
	var d interface{}
	c := context.Background()

	// Infinite loop to make sure we are constantly reading the messages
	// from KAFKA.
	for {
		m, e := kp.Readers[p].ReadMessage(c)
		if e != nil {
			log.Printf("%v", e)
			return e
		}
		if e = Decode(m.Value, &d); e != nil {
			return e
		}
		fn(d)
	}
}

// Get - Not supported for now in Kafka from Message Bus side due to limitations
// on the quality of the go library implementation. Will be taken-up in future.
func (kp *KafkaPacket) Get(pipe string, d interface{}) interface{} {
	return nil
}

// Remove will just remove the existing subscription.
func (kp *KafkaPacket) Remove(pipe string) error {
	es, ok := kp.Readers[pipe]
	if ok == false {
		e := fmt.Errorf("specified pipe is not subscribed yet. please check the pipe name passed")
		return e
	}
	es.Close()
	delete(kp.Readers, pipe)
	return nil
}

// Close will disconnect KAFKA Connection. This API should be called when client
// is completely closing Kafka connection, we would called "Remove" for just removing subscription.
func (kp *KafkaPacket) Close() {
	// Closing all opened Readers Connections
	for rp, rc := range kp.Readers {
		rc.Close()
		delete(kp.Readers, rp)
	}

	// Closing all opened Writers Connections
	for wp, wc := range kp.Writers {
		wc.Close()
		delete(kp.Writers, wp)
	}
}
