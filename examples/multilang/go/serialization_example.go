package main

import (
	"fmt"
	"log"
	"time"

	hp "hybridpipe.io"
)

// Define a message type
type Message struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
	Tags      []string  `json:"tags"`
}

func main() {
	// Register the message type for gob encoding
	hp.Enable(Message{})

	// Create a message
	message := Message{
		Text:      "Hello, world!",
		Timestamp: time.Now(),
		Count:     42,
		Tags:      []string{"example", "serialization", "hybridpipe"},
	}

	fmt.Println("Original message:", message)

	// Demonstrate different serialization formats
	demonstrateGob(message)
	demonstrateJSON(message)
	demonstrateMessagePack(message)
	demonstrateProtobuf(message)
}

func demonstrateGob(message Message) {
	fmt.Println("\n=== GOB Serialization ===")

	// Create serialization options for GOB
	gobOptions := hp.SerializationOptions{
		Format: hp.FormatGob,
	}

	// Serialize the message
	gobBytes, err := hp.EncodeWithOptions(message, gobOptions)
	if err != nil {
		log.Fatalf("Failed to serialize with GOB: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(gobBytes))

	// Deserialize the message
	var gobResult Message
	err = hp.DecodeWithOptions(gobBytes, &gobResult)
	if err != nil {
		log.Fatalf("Failed to deserialize with GOB: %v", err)
	}

	fmt.Println("Deserialized message:", gobResult)
}

func demonstrateJSON(message Message) {
	fmt.Println("\n=== JSON Serialization ===")

	// Create serialization options for JSON
	jsonOptions := hp.SerializationOptions{
		Format:      hp.FormatJSON,
		Compression: true, // Enable compression
	}

	// Serialize the message
	jsonBytes, err := hp.EncodeWithOptions(message, jsonOptions)
	if err != nil {
		log.Fatalf("Failed to serialize with JSON: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(jsonBytes))

	// Deserialize the message
	var jsonResult Message
	err = hp.DecodeWithOptions(jsonBytes, &jsonResult)
	if err != nil {
		log.Fatalf("Failed to deserialize with JSON: %v", err)
	}

	fmt.Println("Deserialized message:", jsonResult)

	// Show the JSON representation
	jsonOptions.Compression = false
	plainJsonBytes, _ := hp.EncodeWithOptions(message, jsonOptions)
	fmt.Printf("JSON representation: %s\n", plainJsonBytes[hp.HeaderSize:])
}

func demonstrateMessagePack(message Message) {
	fmt.Println("\n=== MessagePack Serialization ===")

	// Create serialization options for MessagePack
	msgpackOptions := hp.SerializationOptions{
		Format: hp.FormatMsgPack,
	}

	// Serialize the message
	msgpackBytes, err := hp.EncodeWithOptions(message, msgpackOptions)
	if err != nil {
		log.Fatalf("Failed to serialize with MessagePack: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(msgpackBytes))

	// Deserialize the message
	var msgpackResult Message
	err = hp.DecodeWithOptions(msgpackBytes, &msgpackResult)
	if err != nil {
		log.Fatalf("Failed to deserialize with MessagePack: %v", err)
	}

	fmt.Println("Deserialized message:", msgpackResult)
}

func demonstrateProtobuf(message Message) {
	fmt.Println("\n=== Protocol Buffers Serialization ===")
	fmt.Println("Note: This example requires generating Protocol Buffer code from the message.proto file.")
	fmt.Println("For a real implementation, you would use the generated Protocol Buffer types.")
	fmt.Println("See the proto/message.proto file for the message definitions.")
}
