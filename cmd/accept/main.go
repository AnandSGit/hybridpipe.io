// Command accept is a utility for subscribing to messages from various messaging systems.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	pipe      = flag.String("pipe", "test", "Pipe name to subscribe to")
	address   = flag.String("address", "", "Address of the messaging server")
	format    = flag.String("format", "json", "Output format (json, text)")
	monitor   = flag.Bool("monitor", false, "Enable monitoring")
	monitorAddr = flag.String("monitor-addr", ":8080", "Address for the monitoring server")
	timeout   = flag.Int("timeout", 0, "Timeout in seconds (0 for no timeout)")
	verbose   = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up timeout if specified
	if *timeout > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, time.Duration(*timeout)*time.Second)
		defer timeoutCancel()
	}

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

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
				config.SubscriberEndpoint = parts[0]
			}
			if len(parts) > 1 {
				config.PublisherEndpoint = parts[1]
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

		log.Printf("Monitoring server started on %s", *monitorAddr)
	}

	// Connect to the messaging system
	if err := router.Connect(); err != nil {
		log.Fatalf("Failed to connect to %s: %v", *protocol, err)
	}
	defer router.Close()

	log.Printf("Connected to %s", *protocol)

	// Subscribe to the pipe
	if err := router.Subscribe(*pipe, func(data []byte) error {
		// Process the message
		switch *format {
		case "json":
			var obj interface{}
			if err := json.Unmarshal(data, &obj); err != nil {
				// If it's not valid JSON, just print the raw data
				fmt.Printf("%s\n", data)
			} else {
				// Pretty print the JSON
				jsonData, err := json.MarshalIndent(obj, "", "  ")
				if err != nil {
					fmt.Printf("%s\n", data)
				} else {
					fmt.Printf("%s\n", jsonData)
				}
			}
		case "text":
			fmt.Printf("%s\n", data)
		default:
			fmt.Printf("%s\n", data)
		}

		if *verbose {
			log.Printf("Received message on pipe %s", *pipe)
		}

		return nil
	}); err != nil {
		log.Fatalf("Failed to subscribe to pipe %s: %v", *pipe, err)
	}

	log.Printf("Subscribed to pipe %s", *pipe)

	// Wait for context to be done
	<-ctx.Done()

	log.Printf("Shutting down")
}
