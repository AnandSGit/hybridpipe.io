package main

// Just testing AMQP for now

import (
	"fmt"
	"runtime"
	"sync"

	dc "hybridpipe.io"
)

// Person struct
type Person struct {
	Name    string
	Age     int
	NextGen []string
	CAge    []int
	Next    *Person
}

var i, ni int

// AMQPHandler function
func AMQPHandler(am interface{}) {
	fmt.Println(i, " - Message Sent via AMQP: ", am)
	i++
}

// NATSHandler function
func NATSHandler(nm interface{}) {
	fmt.Println(ni, "Message sent via NATS: ", nm)
	ni++
}

func consume(w *sync.WaitGroup) {
	defer w.Done()
	dc.Enable(Person{})
	A, _ := dc.DeployRouter(dc.AMQP1, nil)
	N, _ := dc.DeployRouter(dc.NATS, nil)
	A.Accept("ServerOI", AMQPHandler)
	N.Accept("ServerIO", NATSHandler)
}

func main() {

	var w sync.WaitGroup
	w.Add(1)
	go consume(&w)
	runtime.Goexit()
	w.Wait()
}
