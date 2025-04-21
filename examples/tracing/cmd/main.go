package main

import (
	"fmt"
	"os"

	"github.com/AnandSGit/hybridpipe.io/examples/tracing"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [basic|distributed|dashboard]")
		return
	}

	example := os.Args[1]
	switch example {
	case "basic":
		fmt.Println("Running basic tracing example...")
		examples.BasicTracingExample()
	case "distributed":
		fmt.Println("Running distributed tracing example...")
		examples.DistributedTracingExample()
	case "dashboard":
		fmt.Println("Running dashboard example...")
		examples.DashboardExample()
	default:
		fmt.Printf("Unknown example: %s\n", example)
		fmt.Println("Available examples: basic, distributed, dashboard")
	}
}
