package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/AnandSGit/hybridpipe.io/netchan"
)

func main() {
	// Parse command line flags
	mode := flag.String("mode", "receiver", "Mode: 'receiver' or 'sender'")
	addr := flag.String("addr", ":8080", "Address to listen on or connect to")
	flag.Parse()

	switch *mode {
	case "receiver":
		runReceiver(*addr)
	case "sender":
		runSender(*addr)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runReceiver(addr string) {
	fmt.Printf("Starting receiver on %s\n", addr)

	// Create a receiver for int values
	receiver, err := netchan.Listen(addr, 0, netchan.WithBufferSize(10))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer receiver.Close()

	fmt.Println("Listening for messages...")

	// Receive values
	for {
		value, ok, err := receiver.Recv()
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		if !ok {
			log.Println("Channel closed")
			break
		}
		fmt.Printf("Received: %d\n", value)
	}
}

func runSender(addr string) {
	fmt.Printf("Starting sender, connecting to %s\n", addr)

	// Connect to the receiver
	sender, err := netchan.Dial(addr, 0, netchan.WithBufferSize(10))
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer sender.Close()

	fmt.Println("Connected, sending messages...")

	// Send values
	for i := 0; i < 10; i++ {
		if err := sender.Send(i); err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		fmt.Printf("Sent: %d\n", i)
		time.Sleep(time.Second)
	}

	fmt.Println("Done sending messages")
}
