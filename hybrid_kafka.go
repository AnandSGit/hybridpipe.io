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
type KafkaPacket struct {
	Readers    map[string]*kafka.Reader
	Writers    map[string]*kafka.Writer
	DialerConn *kafka.Dialer
	Server     string
}

// TLS creates the TLS configuration objects to be used by KAFKA
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

// Dispatch defines the Producer / Publisher role and functionality.
func (kp *KafkaPacket) Dispatch(pipe string, d interface{}) error {

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

// Accept function defines the Consumer or Subscriber functionality for KAFKA.
func (kp *KafkaPacket) Accept(pipe string, fn Process) error {

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

	var d interface{}
	c := context.Background()
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

// Close will disconnect KAFKA Connection.
func (kp *KafkaPacket) Close() {

	for rp, rc := range kp.Readers {
		rc.Close()
		delete(kp.Readers, rp)
	}
	for wp, wc := range kp.Writers {
		wc.Close()
		delete(kp.Writers, wp)
	}
}
