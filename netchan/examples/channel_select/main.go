package main

import (
	"fmt"
	"log"
	"time"

	"hybridpipe.io/netchan"
)

func main() {
	// Example of using NetChan with Go channel syntax and select statements
	fmt.Println("NetChan with select statements example")

	// Create multiple servers in separate goroutines
	go runServer("A", ":8081")
	go runServer("B", ":8082")

	// Wait for servers to start
	time.Sleep(500 * time.Millisecond)

	// Run client with multiple connections
	runMultiClient()
}

func runServer(name string, addr string) {
	// Create a NetChan listener
	netChan, err := netchan.Listen[string](addr, "")
	if err != nil {
		log.Fatalf("Server %s: Failed to create listener: %v", name, err)
	}

	// Create a channel wrapper
	wrapper := netchan.NewChannelWrapper(netChan, 10)
	if err := wrapper.Initialize(); err != nil {
		log.Fatalf("Server %s: Failed to initialize wrapper: %v", name, err)
	}

	fmt.Printf("Server %s: Listening on %s\n", name, addr)

	// Get the receive channel
	recvChan := wrapper.RecvChan()

	// Use Go channel syntax to receive messages
	for {
		// Use native channel syntax to receive
		msg, ok := <-recvChan
		if !ok {
			fmt.Printf("Server %s: Channel closed\n", name)
			break
		}

		fmt.Printf("Server %s received: %s\n", name, msg)

		// Send a response using the send channel
		wrapper.SendChan() <- fmt.Sprintf("From Server %s: %s", name, msg)
	}
}

func runMultiClient() {
	// Connect to multiple servers
	netChanA, err := netchan.Dial[string]("localhost:8081", "")
	if err != nil {
		log.Fatalf("Failed to connect to server A: %v", err)
	}

	netChanB, err := netchan.Dial[string]("localhost:8082", "")
	if err != nil {
		log.Fatalf("Failed to connect to server B: %v", err)
	}

	// Create channel wrappers
	wrapperA := netchan.NewChannelWrapper(netChanA, 10)
	if err := wrapperA.Initialize(); err != nil {
		log.Fatalf("Failed to initialize wrapper A: %v", err)
	}

	wrapperB := netchan.NewChannelWrapper(netChanB, 10)
	if err := wrapperB.Initialize(); err != nil {
		log.Fatalf("Failed to initialize wrapper B: %v", err)
	}

	// Get the send and receive channels
	sendChanA := wrapperA.SendChan()
	recvChanA := wrapperA.RecvChan()

	sendChanB := wrapperB.SendChan()
	recvChanB := wrapperB.RecvChan()

	// Create a ticker for regular messages
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Create a timeout channel
	timeout := time.After(10 * time.Second)

	// Use select to handle multiple channels
	counter := 0
	for {
		select {
		case <-timeout:
			fmt.Println("Client: Timeout reached, exiting")
			wrapperA.Close()
			wrapperB.Close()
			return

		case <-ticker.C:
			// Alternate between servers
			if counter%2 == 0 {
				sendChanA <- fmt.Sprintf("Message %d to Server A", counter)
				fmt.Printf("Client sent message %d to Server A\n", counter)
			} else {
				sendChanB <- fmt.Sprintf("Message %d to Server B", counter)
				fmt.Printf("Client sent message %d to Server B\n", counter)
			}
			counter++

		case msg, ok := <-recvChanA:
			if !ok {
				fmt.Println("Client: Channel A closed")
				recvChanA = nil // Disable this case
				continue
			}
			fmt.Printf("Client received from Server A: %s\n", msg)

		case msg, ok := <-recvChanB:
			if !ok {
				fmt.Println("Client: Channel B closed")
				recvChanB = nil // Disable this case
				continue
			}
			fmt.Printf("Client received from Server B: %s\n", msg)
		}

		// If both channels are closed, exit
		if recvChanA == nil && recvChanB == nil {
			fmt.Println("Client: All channels closed, exiting")
			return
		}
	}
}
