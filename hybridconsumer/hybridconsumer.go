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

var i int = 0

// AMQPHandler Procedure
func AMQPHandler(am interface{}) {
	fmt.Println(i, " - Message Sent via AMQP: ", am)
	i++
}

func consume(w *sync.WaitGroup) {
	defer w.Done()
	dc.Enable(Person{})
	A, _ := dc.DeployRouter(dc.AMQP1, nil)
	A.Accept("/ServerOI", AMQPHandler)
}

func main() {

	var w sync.WaitGroup
	w.Add(1)
	go consume(&w)
	runtime.Goexit()
	w.Wait()
}
