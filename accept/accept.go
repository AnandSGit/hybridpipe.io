package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	dc "hybridpipe.io"
)

// Person struct for serialization testing
type Person struct {
	Name    string
	Age     int
	NextGen []string
	CAge    []int
	Next    *Person
}

// Message counters for each protocol
var (
	amqpCount  int
	natsCount  int
	kafkaCount int
	mqttCount  int
	qpidCount  int
	rmqCount   int
	nsqCount   int
	tcpCount   int
	redisCount int
	// zmqCount   int // Removed ZeroMQ support
	mockCount int
	mutex     sync.Mutex
)

// AMQPHandler processes messages received via AMQP
func AMQPHandler(data interface{}) {
	mutex.Lock()
	amqpCount++
	count := amqpCount
	mutex.Unlock()
	fmt.Printf("[AMQP] Message #%d: %v\n", count, data)
}

// NATSHandler processes messages received via NATS
func NATSHandler(data interface{}) {
	mutex.Lock()
	natsCount++
	count := natsCount
	mutex.Unlock()
	fmt.Printf("[NATS] Message #%d: %v\n", count, data)
}

// KafkaHandler processes messages received via Kafka
func KafkaHandler(data interface{}) {
	mutex.Lock()
	kafkaCount++
	count := kafkaCount
	mutex.Unlock()
	fmt.Printf("[Kafka] Message #%d: %v\n", count, data)
}

// MQTTHandler processes messages received via MQTT
func MQTTHandler(data interface{}) {
	mutex.Lock()
	mqttCount++
	count := mqttCount
	mutex.Unlock()
	fmt.Printf("[MQTT] Message #%d: %v\n", count, data)
}

// QpidHandler processes messages received via Qpid
func QpidHandler(data interface{}) {
	mutex.Lock()
	qpidCount++
	count := qpidCount
	mutex.Unlock()
	fmt.Printf("[Qpid] Message #%d: %v\n", count, data)
}

// RabbitMQHandler processes messages received via RabbitMQ
func RabbitMQHandler(data interface{}) {
	mutex.Lock()
	rmqCount++
	count := rmqCount
	mutex.Unlock()
	fmt.Printf("[RabbitMQ] Message #%d: %v\n", count, data)
}

// NSQHandler processes messages received via NSQ
func NSQHandler(data interface{}) {
	mutex.Lock()
	nsqCount++
	count := nsqCount
	mutex.Unlock()
	fmt.Printf("[NSQ] Message #%d: %v\n", count, data)
}

// TCPHandler processes messages received via TCP
func TCPHandler(data interface{}) {
	mutex.Lock()
	tcpCount++
	count := tcpCount
	mutex.Unlock()
	fmt.Printf("[TCP] Message #%d: %v\n", count, data)
}

// RedisHandler processes messages received via Redis
func RedisHandler(data interface{}) {
	mutex.Lock()
	redisCount++
	count := redisCount
	mutex.Unlock()
	fmt.Printf("[Redis] Message #%d: %v\n", count, data)
}

// ZeroMQHandler was removed as ZeroMQ support was removed
// func ZeroMQHandler(data interface{}) {
// 	mutex.Lock()
// 	zmqCount++
// 	count := zmqCount
// 	mutex.Unlock()
// 	fmt.Printf("[ZeroMQ] Message #%d: %v\n", count, data)
// }

// MockHandler processes messages received via the mock implementation
func MockHandler(data interface{}) {
	mutex.Lock()
	mockCount++
	count := mockCount
	mutex.Unlock()
	fmt.Printf("[Mock] Message #%d: %v\n", count, data)
}

// setupConsumer sets up message consumers for all supported protocols
func setupConsumer(wg *sync.WaitGroup) {
	defer wg.Done()

	// Enable serialization for Person struct
	dc.Enable(Person{})

	// Deploy routers for all supported protocols
	routers := make(map[string]dc.HybridPipe)
	protocols := []struct {
		name     string
		protocol int
		handler  dc.Process
		topic    string
	}{
		{"AMQP", dc.AMQP1, AMQPHandler, "hybridpipe.test.amqp"},
		{"NATS", dc.NATS, NATSHandler, "hybridpipe.test.nats"},
		{"Kafka", dc.KAFKA, KafkaHandler, "hybridpipe.test.kafka"},
		{"MQTT", dc.MQTT, MQTTHandler, "hybridpipe.test.mqtt"},
		{"Qpid", dc.QPID, QpidHandler, "hybridpipe.test.qpid"},
		{"RabbitMQ", dc.RABBITMQ, RabbitMQHandler, "hybridpipe.test.rabbitmq"},
		{"NSQ", dc.NSQ, NSQHandler, "hybridpipe.test.nsq"},
		{"TCP", dc.TCP, TCPHandler, "hybridpipe.test.tcp"},
		{"Redis", dc.REDIS, RedisHandler, "hybridpipe.test.redis"},
		// {"ZeroMQ", dc.RESERVED1, ZeroMQHandler, "hybridpipe.test.zmq"}, // Removed ZeroMQ support
		{"Mock", dc.MOCK, MockHandler, "hybridpipe.test.mock"},
	}

	// Connect to each protocol and subscribe to topics
	for _, p := range protocols {
		log.Printf("Setting up consumer for %s protocol\n", p.name)

		router, err := dc.DeployRouter(p.protocol)
		if err != nil {
			log.Printf("Error deploying router for %s: %v\n", p.name, err)
			continue
		}

		// Store router for cleanup
		routers[p.name] = router

		// Subscribe to topic
		err = router.Accept(p.topic, p.handler)
		if err != nil {
			log.Printf("Error subscribing to topic for %s: %v\n", p.name, err)
			continue
		}

		log.Printf("Successfully subscribed to %s topic: %s\n", p.name, p.topic)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigCh
	log.Println("Shutdown signal received, closing connections...")

	// Close all connections
	for name, router := range routers {
		log.Printf("Closing %s connection\n", name)
		router.Close()
	}

	// Print final message counts
	log.Printf("Final message counts:\n")
	log.Printf("AMQP: %d\n", amqpCount)
	log.Printf("NATS: %d\n", natsCount)
	log.Printf("Kafka: %d\n", kafkaCount)
	log.Printf("MQTT: %d\n", mqttCount)
	log.Printf("Qpid: %d\n", qpidCount)
	log.Printf("RabbitMQ: %d\n", rmqCount)
	log.Printf("NSQ: %d\n", nsqCount)
	log.Printf("TCP: %d\n", tcpCount)
	log.Printf("Redis: %d\n", redisCount)
	// log.Printf("ZeroMQ: %d\n", zmqCount) // Removed ZeroMQ support
	log.Printf("Mock: %d\n", mockCount)
}

func main() {
	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Println("Starting HybridPipe consumer test")

	// Start consumer in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go setupConsumer(&wg)

	// Print status every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			mutex.Lock()
			log.Printf("Current message counts - AMQP: %d, NATS: %d, Kafka: %d, MQTT: %d, Qpid: %d, RabbitMQ: %d, NSQ: %d, TCP: %d, Redis: %d, Mock: %d\n",
				amqpCount, natsCount, kafkaCount, mqttCount, qpidCount, rmqCount, nsqCount, tcpCount, redisCount, mockCount)
			mutex.Unlock()
		}
	}()

	// Wait for goroutines to complete
	runtime.Goexit()
	wg.Wait()
	ticker.Stop()
	log.Println("HybridPipe consumer test completed")
}
