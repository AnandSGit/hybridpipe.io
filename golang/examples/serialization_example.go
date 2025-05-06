package examples

import (
	"fmt"
	"log"
	"time"

	"hybridpipe.io/core"
)

// Define a message type
type Message struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
	Tags      []string  `json:"tags"`
}

func SerializationExample() {
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
}

func demonstrateGob(message Message) {
	fmt.Println("\n=== GOB Serialization ===")

	// Create serialization options for GOB
	gobOptions := &core.SerializationOptions{
		Format: core.FormatGOB,
	}

	// Serialize the message
	gobBytes, err := core.EncodeWithOptions(message, gobOptions)
	if err != nil {
		log.Fatalf("Failed to serialize with GOB: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(gobBytes))

	// Deserialize the message
	var gobResult Message
	err = core.Decode(gobBytes, &gobResult)
	if err != nil {
		log.Fatalf("Failed to deserialize with GOB: %v", err)
	}

	fmt.Println("Deserialized message:", gobResult)
}

func demonstrateJSON(message Message) {
	fmt.Println("\n=== JSON Serialization ===")

	// Create serialization options for JSON
	jsonOptions := &core.SerializationOptions{
		Format:      core.FormatJSON,
		Compression: true, // Enable compression
	}

	// Serialize the message
	jsonBytes, err := core.EncodeWithOptions(message, jsonOptions)
	if err != nil {
		log.Fatalf("Failed to serialize with JSON: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(jsonBytes))

	// Deserialize the message
	var jsonResult Message
	err = core.Decode(jsonBytes, &jsonResult)
	if err != nil {
		log.Fatalf("Failed to deserialize with JSON: %v", err)
	}

	fmt.Println("Deserialized message:", jsonResult)

	// Show the JSON representation
	jsonOptions.Compression = false
	plainJsonBytes, _ := core.EncodeWithOptions(message, jsonOptions)
	fmt.Printf("JSON representation: %s\n", string(plainJsonBytes))
}

func demonstrateMessagePack(message Message) {
	fmt.Println("\n=== MessagePack Serialization ===")

	// Create serialization options for MessagePack
	msgpackOptions := &core.SerializationOptions{
		Format: core.FormatMessagePack,
	}

	// Serialize the message
	msgpackBytes, err := core.EncodeWithOptions(message, msgpackOptions)
	if err != nil {
		log.Fatalf("Failed to serialize with MessagePack: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(msgpackBytes))

	// Deserialize the message
	var msgpackResult Message
	err = core.Decode(msgpackBytes, &msgpackResult)
	if err != nil {
		log.Fatalf("Failed to deserialize with MessagePack: %v", err)
	}

	fmt.Println("Deserialized message:", msgpackResult)
}
