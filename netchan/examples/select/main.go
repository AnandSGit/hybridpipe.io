package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"hybridpipe.io/netchan"
)

func main() {
	// Parse command line flags
	mode := flag.String("mode", "hub", "Mode: 'hub', 'producer', or 'consumer'")
	hubAddr := flag.String("hub", ":8080", "Hub address")
	producerAddr := flag.String("producer", ":8081", "Producer address")
	consumerAddr := flag.String("consumer", ":8082", "Consumer address")
	flag.Parse()

	switch *mode {
	case "hub":
		runHub(*hubAddr, *producerAddr, *consumerAddr)
	case "producer":
		runProducer(*hubAddr)
	case "consumer":
		runConsumer(*consumerAddr)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runHub(hubAddr, producerAddr, consumerAddr string) {
	fmt.Printf("Starting hub on %s\n", hubAddr)
	fmt.Printf("Producer address: %s\n", producerAddr)
	fmt.Printf("Consumer address: %s\n", consumerAddr)

	// Create channels for producer and consumer
	producerCh, err := netchan.Listen(producerAddr, 0, netchan.WithBufferSize(10))
	if err != nil {
		log.Fatalf("Failed to listen for producers: %v", err)
	}
	defer producerCh.Close()

	consumerCh, err := netchan.Listen(consumerAddr, 0, netchan.WithBufferSize(10))
	if err != nil {
		log.Fatalf("Failed to listen for consumers: %v", err)
	}
	defer consumerCh.Close()

	// Create a channel for the hub control
	hubCh, err := netchan.Listen(hubAddr, "", netchan.WithBufferSize(10))
	if err != nil {
		log.Fatalf("Failed to listen for hub control: %v", err)
	}
	defer hubCh.Close()

	fmt.Println("Hub started, waiting for messages...")

	// Buffer for messages
	buffer := make([]int, 0, 100)

	for {
		// Create select cases
		var cases []netchan.Case
		var producerCase, consumerCase, hubCase, timeoutCase netchan.Case

		// Always listen for producer messages
		producerCase = netchan.RecvCase(producerCh)
		cases = append(cases, producerCase)

		// Only listen for consumer messages if we have something in the buffer
		if len(buffer) > 0 {
			consumerCase = netchan.SendCase(consumerCh, buffer[0])
			cases = append(cases, consumerCase)
		}

		// Always listen for hub control messages
		hubCase = netchan.RecvCase(hubCh)
		cases = append(cases, hubCase)

		// Add a timeout
		timeoutCase = netchan.TimeoutCase(5 * time.Second)
		cases = append(cases, timeoutCase)

		// Perform select
		chosen, recv, recvOK, err := netchan.Select(cases)
		if err != nil {
			log.Printf("Select error: %v", err)
			continue
		}

		// Handle the chosen case
		switch {
		case chosen == 0: // Producer
			if recvOK {
				value := recv.(int)
				fmt.Printf("Received from producer: %d\n", value)
				buffer = append(buffer, value)
			} else {
				fmt.Println("Producer channel closed")
			}

		case chosen == 1 && len(buffer) > 0: // Consumer
			fmt.Printf("Sent to consumer: %d\n", buffer[0])
			buffer = buffer[1:] // Remove the sent item from the buffer

		case chosen == len(cases)-2: // Hub control
			if recvOK {
				cmd := recv.(string)
				fmt.Printf("Received hub command: %s\n", cmd)
				switch cmd {
				case "status":
					fmt.Printf("Buffer size: %d\n", len(buffer))
				case "clear":
					buffer = buffer[:0]
					fmt.Println("Buffer cleared")
				case "quit":
					fmt.Println("Shutting down hub")
					return
				}
			} else {
				fmt.Println("Hub control channel closed")
			}

		case chosen == len(cases)-1: // Timeout
			fmt.Printf("Select timed out. Buffer size: %d\n", len(buffer))
		}
	}
}

func runProducer(hubAddr string) {
	fmt.Printf("Starting producer, connecting to hub at %s\n", hubAddr)

	// Connect to the hub for control messages
	hubCh, err := netchan.Dial(hubAddr, "", netchan.WithBufferSize(10))
	if err != nil {
		log.Fatalf("Failed to connect to hub: %v", err)
	}
	defer hubCh.Close()

	// Connect to the producer channel
	producerAddr := hubAddr[:len(hubAddr)-1] + "1" // Change last digit to 1
	producerCh, err := netchan.Dial(producerAddr, 0, netchan.WithBufferSize(10))
	if err != nil {
		log.Fatalf("Failed to connect to producer channel: %v", err)
	}
	defer producerCh.Close()

	fmt.Println("Connected to hub, sending messages...")

	// Send values
	for i := 0; i < 20; i++ {
		if err := producerCh.Send(i); err != nil {
			log.Printf("Error sending to producer channel: %v", err)
			continue
		}
		fmt.Printf("Sent: %d\n", i)
		time.Sleep(500 * time.Millisecond)

		// Every 5 messages, send a status request to the hub
		if i > 0 && i%5 == 0 {
			if err := hubCh.Send("status"); err != nil {
				log.Printf("Error sending status request: %v", err)
			} else {
				fmt.Println("Sent status request to hub")
			}
		}
	}

	// Send quit command to hub
	if err := hubCh.Send("quit"); err != nil {
		log.Printf("Error sending quit command: %v", err)
	} else {
		fmt.Println("Sent quit command to hub")
	}

	fmt.Println("Done sending messages")
}

func runConsumer(consumerAddr string) {
	fmt.Printf("Starting consumer, connecting to %s\n", consumerAddr)

	// Connect to the consumer channel
	consumerCh, err := netchan.Dial(consumerAddr, 0, netchan.WithBufferSize(10))
	if err != nil {
		log.Fatalf("Failed to connect to consumer channel: %v", err)
	}
	defer consumerCh.Close()

	fmt.Println("Connected, receiving messages...")

	// Receive values
	for {
		value, ok, err := consumerCh.Recv()
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		if !ok {
			log.Println("Channel closed")
			break
		}
		fmt.Printf("Received: %d\n", value)

		// Simulate processing time
		time.Sleep(1 * time.Second)
	}
}
