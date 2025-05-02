package examples

import (
	"fmt"
	"log"
	"time"

	hp "hybridpipe.io"
)

// Define a custom type
type Person struct {
	Name      string
	Age       int
	BirthDate time.Time
	Address   Address
	Tags      []string
	Scores    map[string]float64
	Active    bool
}

type Address struct {
	Street     string
	City       string
	PostalCode string
	Country    string
}

func SerializationExample() {
	// Register types for gob encoding
	hp.Enable(Person{})
	hp.Enable(Address{})

	// Create a test person
	person := Person{
		Name:      "John Doe",
		Age:       30,
		BirthDate: time.Date(1993, 1, 1, 0, 0, 0, 0, time.UTC),
		Address: Address{
			Street:     "123 Main St",
			City:       "Anytown",
			PostalCode: "12345",
			Country:    "USA",
		},
		Tags:   []string{"developer", "golang", "hybridpipe"},
		Scores: map[string]float64{"math": 95.5, "science": 92.0, "history": 88.5},
		Active: true,
	}

	// Demonstrate different serialization formats
	demonstrateGob(person)
	demonstrateJSON(person)
	demonstrateMessagePack(person)
	demonstrateProtobuf(person)
	demonstrateCompression(person)
	demonstrateCrossLanguageCompatibility(person)
}

func demonstrateGob(person Person) {
	fmt.Println("\n=== Gob Serialization (Go-specific) ===")

	// Create serialization options for Gob
	options := hp.SerializationOptions{
		Format: hp.FormatGob,
	}

	// Serialize the person
	bytes, err := hp.EncodeWithOptions(person, options)
	if err != nil {
		log.Fatalf("Failed to serialize with Gob: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(bytes))

	// Deserialize the person
	var result Person
	err = hp.DecodeWithOptions(bytes, &result, options)
	if err != nil {
		log.Fatalf("Failed to deserialize with Gob: %v", err)
	}

	fmt.Printf("Deserialized person: %s, %d years old\n", result.Name, result.Age)
	fmt.Printf("Birth date: %s\n", result.BirthDate.Format("2006-01-02"))
	fmt.Printf("Address: %s, %s, %s\n", result.Address.Street, result.Address.City, result.Address.Country)
}

func demonstrateJSON(person Person) {
	fmt.Println("\n=== JSON Serialization (Cross-language compatible) ===")

	// Create serialization options for JSON
	options := hp.SerializationOptions{
		Format: hp.FormatJSON,
	}

	// Serialize the person
	bytes, err := hp.EncodeWithOptions(person, options)
	if err != nil {
		log.Fatalf("Failed to serialize with JSON: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(bytes))
	fmt.Printf("JSON content: %s\n", string(bytes[hp.HeaderSize:]))

	// Deserialize the person
	var result Person
	err = hp.DecodeWithOptions(bytes, &result, options)
	if err != nil {
		log.Fatalf("Failed to deserialize with JSON: %v", err)
	}

	fmt.Printf("Deserialized person: %s, %d years old\n", result.Name, result.Age)
}

func demonstrateMessagePack(person Person) {
	fmt.Println("\n=== MessagePack Serialization (Efficient and cross-language compatible) ===")

	// Create serialization options for MessagePack
	options := hp.SerializationOptions{
		Format: hp.FormatMsgPack,
	}

	// Serialize the person
	bytes, err := hp.EncodeWithOptions(person, options)
	if err != nil {
		log.Fatalf("Failed to serialize with MessagePack: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(bytes))

	// Deserialize the person
	var result Person
	err = hp.DecodeWithOptions(bytes, &result, options)
	if err != nil {
		log.Fatalf("Failed to deserialize with MessagePack: %v", err)
	}

	fmt.Printf("Deserialized person: %s, %d years old\n", result.Name, result.Age)
}

func demonstrateProtobuf(person Person) {
	fmt.Println("\n=== Protocol Buffers Serialization (Most efficient) ===")
	fmt.Println("Note: This example is simplified as it requires generated Protocol Buffer code.")
	fmt.Println("In a real application, you would define a .proto file and generate the Go code.")
}

func demonstrateCompression(person Person) {
	fmt.Println("\n=== Compression ===")

	// Create serialization options with compression
	options := hp.SerializationOptions{
		Format:           hp.FormatJSON,
		Compression:      true,
		CompressionLevel: 9, // Best compression
	}

	// Serialize the person without compression
	bytesUncompressed, err := hp.EncodeWithOptions(person, hp.SerializationOptions{Format: hp.FormatJSON})
	if err != nil {
		log.Fatalf("Failed to serialize without compression: %v", err)
	}

	// Serialize the person with compression
	bytesCompressed, err := hp.EncodeWithOptions(person, options)
	if err != nil {
		log.Fatalf("Failed to serialize with compression: %v", err)
	}

	fmt.Printf("Uncompressed size: %d bytes\n", len(bytesUncompressed))
	fmt.Printf("Compressed size: %d bytes\n", len(bytesCompressed))
	fmt.Printf("Compression ratio: %.2f%%\n", 100.0*(1.0-float64(len(bytesCompressed))/float64(len(bytesUncompressed))))

	// Deserialize the person
	var result Person
	err = hp.DecodeWithOptions(bytesCompressed, &result, options)
	if err != nil {
		log.Fatalf("Failed to deserialize with compression: %v", err)
	}

	fmt.Printf("Deserialized person: %s, %d years old\n", result.Name, result.Age)
}

func demonstrateCrossLanguageCompatibility(person Person) {
	fmt.Println("\n=== Cross-Language Compatibility ===")

	// Create serialization options for cross-language compatibility
	options := hp.SerializationOptions{
		Format:                  hp.FormatJSON,
		CrossLanguageCompatible: true,
	}

	// Serialize the person
	bytes, err := hp.EncodeWithOptions(person, options)
	if err != nil {
		log.Fatalf("Failed to serialize with cross-language compatibility: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(bytes))
	fmt.Printf("This data can be deserialized in other languages like Python, JavaScript, etc.\n")
}
