package main

import (
	"fmt"
	dc "hybridpipe"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// Person struct
type Person struct {
	Name    string
	Age     int
	NextGen []string
	CAge    []int
	Next    *Person
}

// KafkaHandler Procedure
func KafkaHandler(k interface{}) {
	fmt.Println("Message Sent via KAFKA: ", k)
}

// RespondHandler Procedure
func RespondHandler(nr interface{}) interface{} {
	return nr
}

// NatsHandler Procedure
func NatsHandler(n interface{}) {
	fmt.Println("Message Sent via NATS: ", n)
}

// RabbitHandler Procedure
func RabbitHandler(r interface{}) {
	fmt.Println("Message Sent via RabbitMQ: ", r)
}

var clear map[string]func() //create a map for storing clear funcs

func initClear() {

	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func doEvery(d time.Duration) {
	for range time.Tick(d) {
		clear[runtime.GOOS]()
	}
}

func consume() {

	dc.Enable(Person{})
	// initClear()
	// doEvery(15 * time.Second)

	N, _ := dc.Medium(dc.NATS, RespondHandler)
	R, _ := dc.Medium(dc.RABBITMQ, nil)
	K, _ := dc.Medium(dc.KAFKA, nil)

	defer N.Close()
	defer K.Close()
	defer R.Close()

	N.Accept("Server.iLO.Low", NatsHandler)
	N.Remove("Server.iLO.Low")
	K.Accept("Server.iLO.High", KafkaHandler)
	K.Remove("Server.iLO.High")
	R.Accept("Server.iLO.Med", RabbitHandler)
	R.Remove("Server.iLO.Mid")
}

func main() {

	var w sync.WaitGroup
	go consume()
	runtime.Goexit()
	w.Wait()
}
