package examples

import (
	"fmt"
	"log"
	"time"

	hp "hybridpipe.io"
)

func NetChanAdapterExample() {
	// Create a new NetChanAdapter
	adapter, err := hp.NewNetChanAdapter(10)
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Close()

	// Get send and receive channels for a pipe
	sendChan, err := adapter.GetSendChan("example-pipe")
	if err != nil {
		log.Fatalf("Failed to get send channel: %v", err)
	}

	recvChan, err := adapter.GetRecvChan("example-pipe")
	if err != nil {
		log.Fatalf("Failed to get receive channel: %v", err)
	}

	// Start a goroutine to receive messages
	go func() {
		for msg := range recvChan {
			fmt.Printf("Received: %v\n", msg)
		}
	}()

	// Send messages using Go channel syntax
	messages := []string{
		"Hello, NetChan!",
		"This is using Go channel syntax",
		"With HybridPipe integration",
		"Isn't that cool?",
	}

	for _, msg := range messages {
		// Send using native channel syntax
		sendChan <- msg
		fmt.Printf("Sent: %s\n", msg)
		time.Sleep(500 * time.Millisecond)
	}

	// Wait a bit to ensure all messages are processed
	time.Sleep(1 * time.Second)

	// Print statistics
	router := adapter.GetRouter()
	if packet, ok := router.(*hp.NetChanPacket); ok {
		stats := packet.GetStats()
		fmt.Printf("\nStatistics:\n")
		fmt.Printf("Messages sent: %d\n", stats.GetMessagesSent())
		fmt.Printf("Messages received: %d\n", stats.GetMessagesReceived())
		fmt.Printf("Bytes sent: %d\n", stats.GetBytesSent())
		fmt.Printf("Bytes received: %d\n", stats.GetBytesReceived())
		fmt.Printf("Last sent time: %v\n", stats.GetLastSentTime())
		fmt.Printf("Last received time: %v\n", stats.GetLastReceivedTime())
		fmt.Printf("Error count: %d\n", stats.GetErrorCount())
	}
}
