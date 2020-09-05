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

// AMQPHandler Procedure
func AMQPHandler(am interface{}) {
	fmt.Println("Message Sent via AMQP: ", am)
}

func consume(w *sync.WaitGroup) {
	defer w.Done()
	dc.Enable(Person{})
	N, _ := dc.DeployRouter(dc.NATS, nil)
	fmt.Println(N.Accept("ServerIO", AMQPHandler))
}

func main() {

	var w sync.WaitGroup
	w.Add(1)
	go consume(&w)
	runtime.Goexit()
	w.Wait()
}
