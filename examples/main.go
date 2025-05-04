package examples

import (
	"fmt"
	"os"
)

func main() {
	// Check if an example name was provided
	if len(os.Args) < 2 {
		fmt.Println("Please specify an example to run:")
		fmt.Println("  mock - Run the mock messaging example")
		fmt.Println("  netchan - Run the NetChan example")
		fmt.Println("  serialization - Run the serialization example")
		os.Exit(1)
	}

	// Run the specified example
	switch os.Args[1] {
	case "mock":
		fmt.Println("Running Mock Example...")
		MockExample()
	case "netchan":
		fmt.Println("Running NetChan Example...")
		NetChanExample()
	case "serialization":
		fmt.Println("Running Serialization Example...")
		SerializationExample()
	default:
		fmt.Printf("Unknown example: %s\n", os.Args[1])
		fmt.Println("Available examples: mock, netchan, serialization")
		os.Exit(1)
	}
}
