package examples

import (
	"fmt"
	"os"
)

// Main function to run the examples
func Main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [basic|distributed|dashboard]")
		return
	}

	example := os.Args[1]
	switch example {
	case "basic":
		fmt.Println("Running basic tracing example...")
		BasicTracingExample()
	case "distributed":
		fmt.Println("Running distributed tracing example...")
		DistributedTracingExample()
	case "dashboard":
		fmt.Println("Running dashboard example...")
		DashboardExample()
	default:
		fmt.Printf("Unknown example: %s\n", example)
		fmt.Println("Available examples: basic, distributed, dashboard")
	}
}
