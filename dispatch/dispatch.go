package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	dc "hybridpipe.io"
)

// Test message formats
const (
	SimpleMessage string = `Simple test message from HybridPipe`
	JSONMessage   string = `{
		"id": "msg-123",
		"timestamp": "2023-06-15T12:34:56Z",
		"payload": {
			"type": "notification",
			"priority": "high",
			"content": "This is a test message"
		}
	}`
)

// Person struct for serialization testing
type Person struct {
	Name    string
	Age     int
	NextGen []string
	CAge    []int
	Next    *Person
}

// SensorData represents IoT sensor data for testing
type SensorData struct {
	DeviceID    string    `json:"device_id"`
	Timestamp   time.Time `json:"timestamp"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Pressure    float64   `json:"pressure"`
	Battery     int       `json:"battery"`
	Location    struct {
		Latitude  float64 `json:"lat"`
		Longitude float64 `json:"lng"`
	} `json:"location"`
}

// Command line flags
var (
	messageCount    int
	intervalMs      int
	protocolsFlag   string
	topicPrefix     string
	messageTypeFlag string
)

// setupProducer sets up message producers for all supported protocols
func setupProducer(wg *sync.WaitGroup) {
	defer wg.Done()

	// Enable serialization for custom types
	dc.Enable(Person{})
	dc.Enable(SensorData{})

	// Parse protocol list
	var selectedProtocols []string
	if protocolsFlag == "all" {
		selectedProtocols = []string{"AMQP", "NATS", "Kafka", "MQTT", "Qpid", "RabbitMQ", "NSQ", "TCP", "Redis", "ZeroMQ", "Mock"}
	} else {
		// Parse comma-separated list (simplified for this example)
		selectedProtocols = []string{protocolsFlag}
	}

	// Map protocol names to constants
	protocolMap := map[string]int{
		"AMQP":     dc.AMQP1,
		"NATS":     dc.NATS,
		"Kafka":    dc.KAFKA,
		"MQTT":     dc.MQTT,
		"Qpid":     dc.QPID,
		"RabbitMQ": dc.RABBITMQ,
		"NSQ":      dc.NSQ,
		"TCP":      dc.TCP,
		"Redis":    dc.REDIS,
		// "ZeroMQ":   dc.RESERVED1, // Removed ZeroMQ support
		"Mock": dc.MOCK,
	}

	// Deploy routers for selected protocols
	routers := make(map[string]dc.HybridPipe)
	for _, name := range selectedProtocols {
		protocolID, exists := protocolMap[name]
		if !exists {
			log.Printf("Unknown protocol: %s, skipping\n", name)
			continue
		}

		log.Printf("Deploying router for %s protocol\n", name)
		router, err := dc.DeployRouter(protocolID)
		if err != nil {
			log.Printf("Error deploying router for %s: %v\n", name, err)
			continue
		}

		routers[name] = router
		log.Printf("Successfully deployed %s router\n", name)
	}

	// Prepare test data
	var messageData interface{}
	switch messageTypeFlag {
	case "simple":
		messageData = SimpleMessage

	case "json":
		messageData = JSONMessage

	case "person":
		// Create nested Person objects
		child := Person{
			Name:    "Alex Smith",
			Age:     28,
			NextGen: []string{"Emma", "James", "Olivia"},
			CAge:    []int{5, 7, 3},
			Next:    nil,
		}

		messageData = Person{
			Name:    "John Doe",
			Age:     52,
			NextGen: []string{"Alex", "Sarah", "Michael"},
			CAge:    []int{28, 26, 24},
			Next:    &child,
		}

	case "sensor":
		// Create sensor data
		sensor := SensorData{
			DeviceID:    fmt.Sprintf("sensor-%d", time.Now().UnixNano()%1000),
			Timestamp:   time.Now(),
			Temperature: 22.5 + float64(time.Now().UnixNano()%10)/10.0,
			Humidity:    45.2 + float64(time.Now().UnixNano()%15)/10.0,
			Pressure:    1013.25 + float64(time.Now().UnixNano()%20)/10.0,
			Battery:     85 - int(time.Now().UnixNano()%15),
		}
		sensor.Location.Latitude = 37.7749 + float64(time.Now().UnixNano()%100)/10000.0
		sensor.Location.Longitude = -122.4194 + float64(time.Now().UnixNano()%100)/10000.0
		messageData = sensor

	default:
		log.Fatalf("Unknown message type: %s\n", messageTypeFlag)
	}

	// Print message data for reference
	jsonData, _ := json.MarshalIndent(messageData, "", "  ")
	log.Printf("Sending message data:\n%s\n", string(jsonData))

	// Send messages
	interval := time.Duration(intervalMs) * time.Millisecond
	log.Printf("Sending %d messages with %v interval\n", messageCount, interval)

	for i := 1; i <= messageCount; i++ {
		// Send to each protocol
		for name, router := range routers {
			topic := fmt.Sprintf("%s.test.%s", topicPrefix, name)
			err := router.Dispatch(topic, messageData)
			if err != nil {
				log.Printf("Error sending message #%d to %s: %v\n", i, name, err)
			} else if i%10 == 0 || i == 1 || i == messageCount {
				// Only log every 10th message to reduce output
				log.Printf("Sent message #%d to %s topic: %s\n", i, name, topic)
			}
		}

		// Wait for the specified interval
		if i < messageCount && interval > 0 {
			time.Sleep(interval)
		}
	}

	// Close all connections
	log.Println("All messages sent, closing connections...")
	for name, router := range routers {
		log.Printf("Closing %s connection\n", name)
		router.Close()
	}
}

func main() {
	// Parse command line flags
	flag.IntVar(&messageCount, "count", 100, "Number of messages to send")
	flag.IntVar(&intervalMs, "interval", 100, "Interval between messages in milliseconds")
	flag.StringVar(&protocolsFlag, "protocols", "all", "Comma-separated list of protocols to use (AMQP,NATS,Kafka,MQTT,Qpid,RabbitMQ,NSQ,TCP,Redis,Mock) or 'all'")
	flag.StringVar(&topicPrefix, "topic", "hybridpipe", "Topic prefix for messages")
	flag.StringVar(&messageTypeFlag, "type", "person", "Message type (simple, json, person, sensor)")
	flag.Parse()

	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Println("Starting HybridPipe producer test")

	// Start producer in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go setupProducer(&wg)

	// Wait for producer to complete
	wg.Wait()
	log.Println("HybridPipe producer test completed")
}
