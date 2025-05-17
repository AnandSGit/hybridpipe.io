// Command dispatch is a utility for sending messages to various messaging systems.
package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"hybridpipe.io/core"
	"hybridpipe.io/monitoring"
	"hybridpipe.io/protocols/amqp"
	"hybridpipe.io/protocols/kafka"
	"hybridpipe.io/protocols/mqtt"
	"hybridpipe.io/protocols/nats"
	"hybridpipe.io/protocols/netchan"
	"hybridpipe.io/protocols/nsq"
	"hybridpipe.io/protocols/qpid"
	"hybridpipe.io/protocols/rabbitmq"
	"hybridpipe.io/protocols/redis"
	"hybridpipe.io/protocols/tcp"
	"hybridpipe.io/protocols/zeromq"
)

var (
	protocol  = flag.String("protocol", "nats", "Messaging protocol to use (nats, kafka, rabbitmq, amqp, mqtt, qpid, nsq, tcp, redis, zeromq, netchan)")
	pipe      = flag.String("pipe", "test", "Pipe name to send to")
	address   = flag.String("address", "", "Address of the messaging server")
	message   = flag.String("message", "", "Message to send")
	file      = flag.String("file", "", "File containing message to send")
	monitor   = flag.Bool("monitor", false, "Enable monitoring")
	monitorAddr = flag.String("monitor-addr", ":8080", "Address for the monitoring server")
	timeout   = flag.Int("timeout", 5, "Timeout in seconds")
	verbose   = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	// Get the message to send
	var messageData string
	if *message != "" {
		messageData = *message
	} else if *file != "" {
		data, err := ioutil.ReadFile(*file)
		if err != nil {
			log.Fatalf("Failed to read file %s: %v", *file, err)
		}
		messageData = string(data)
	} else {
		// Read from stdin
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Failed to read from stdin: %v", err)
		}
		messageData = string(data)
	}

	// Create a router based on the protocol
	var router core.HybridPipe
	var err error

	switch strings.ToLower(*protocol) {
	case "nats":
		config := &core.NATSConfig{}
		if *address != "" {
			config.URL = *address
		}
		router = nats.New(config)
	case "kafka":
		config := &core.KafkaConfig{}
		if *address != "" {
			config.Brokers = []string{*address}
		}
		router = kafka.New(config)
	case "rabbitmq":
		config := &core.RabbitMQConfig{}
		if *address != "" {
			config.URL = *address
		}
		router = rabbitmq.New(config)
	case "amqp":
		config := &core.AMQPConfig{}
		if *address != "" {
			config.URL = *address
		}
		router = amqp.New(config)
	case "mqtt":
		config := &core.MQTTConfig{}
		if *address != "" {
			config.Broker = *address
		}
		router = mqtt.New(config)
	case "qpid":
		config := &core.QpidConfig{}
		if *address != "" {
			config.URL = *address
		}
		router = qpid.New(config)
	case "nsq":
		config := &core.NSQConfig{}
		if *address != "" {
			config.NsqdAddress = *address
		}
		router = nsq.New(config)
	case "tcp":
		config := &core.TCPConfig{}
		if *address != "" {
			config.Address = *address
		}
		router = tcp.New(config)
	case "redis":
		config := &core.RedisConfig{}
		if *address != "" {
			config.Address = *address
		}
		router = redis.New(config)
	case "zeromq":
		config := &core.ZeroMQConfig{}
		if *address != "" {
			parts := strings.Split(*address, ",")
			if len(parts) > 0 {
				config.PublisherEndpoint = parts[0]
			}
			if len(parts) > 1 {
				config.SubscriberEndpoint = parts[1]
			}
		}
		router = zeromq.New(config)
	case "netchan":
		config := &core.NetChanConfig{}
		if *address != "" {
			config.Address = *address
		}
		router = netchan.New(config)
	default:
		log.Fatalf("Unsupported protocol: %s", *protocol)
	}

	// Add monitoring if enabled
	if *monitor {
		tracer := monitoring.NewMessageTracer(1000)
		router = monitoring.NewTracingMiddleware(router, tracer, *protocol)

		// Start the monitoring server
		if err := monitoring.StartDefaultMonitoringServer(*monitorAddr); err != nil {
			log.Fatalf("Failed to start monitoring server: %v", err)
		}
		defer monitoring.StopDefaultMonitoringServer()

		if *verbose {
			log.Printf("Monitoring server started on %s", *monitorAddr)
		}
	}

	// Connect to the messaging system
	if err := router.Connect(); err != nil {
		log.Fatalf("Failed to connect to %s: %v", *protocol, err)
	}
	defer router.Close()

	if *verbose {
		log.Printf("Connected to %s", *protocol)
	}

	// Dispatch the message
	if err := router.DispatchWithContext(ctx, *pipe, messageData); err != nil {
		log.Fatalf("Failed to dispatch message to pipe %s: %v", *pipe, err)
	}

	if *verbose {
		log.Printf("Message dispatched to pipe %s", *pipe)
	}
}
