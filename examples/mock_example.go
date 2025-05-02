package examples

import (
	"fmt"
	"log"
	"time"

	hp "hybridpipe.io"
)

// Define a custom data type
type SensorData struct {
	DeviceID    string
	Temperature float64
	Humidity    float64
	Timestamp   time.Time
}

func MockExample() {
	// Register the custom data type
	hp.Enable(SensorData{})

	// Create a mock router
	router, err := hp.DeployRouter(hp.MOCK)
	if err != nil {
		log.Fatalf("Failed to create mock router: %v", err)
	}
	defer router.Close()

	fmt.Println("Connected to mock messaging system")

	// Subscribe to a topic
	err = router.Accept("sensors/temperature", func(data interface{}) {
		if sensorData, ok := data.(SensorData); ok {
			fmt.Printf("Received sensor data: Device %s, Temperature %.1f°C, Humidity %.1f%%, Time %s\n",
				sensorData.DeviceID, sensorData.Temperature, sensorData.Humidity, sensorData.Timestamp.Format(time.RFC3339))
		}
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	fmt.Println("Subscribed to sensors/temperature")

	// Publish some sensor data
	for i := 0; i < 5; i++ {
		sensorData := SensorData{
			DeviceID:    fmt.Sprintf("device-%d", i+1),
			Temperature: 20.0 + float64(i),
			Humidity:    50.0 + float64(i*2),
			Timestamp:   time.Now(),
		}

		err = router.Dispatch("sensors/temperature", sensorData)
		if err != nil {
			log.Printf("Failed to publish sensor data: %v", err)
			continue
		}

		fmt.Printf("Published sensor data for device-%d\n", i+1)
		time.Sleep(500 * time.Millisecond)
	}

	// Unsubscribe from the topic
	err = router.Remove("sensors/temperature")
	if err != nil {
		log.Fatalf("Failed to unsubscribe: %v", err)
	}

	fmt.Println("Unsubscribed from sensors/temperature")

	// Try to publish after unsubscribing (should still work but no handler will process it)
	sensorData := SensorData{
		DeviceID:    "device-final",
		Temperature: 30.0,
		Humidity:    70.0,
		Timestamp:   time.Now(),
	}

	err = router.Dispatch("sensors/temperature", sensorData)
	if err != nil {
		log.Printf("Failed to publish final sensor data: %v", err)
	} else {
		fmt.Println("Published final sensor data (no subscribers)")
	}

	fmt.Println("Mock example completed successfully")
}
