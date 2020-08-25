package main

import (
	"fmt"
	"sync"
	"time"

	dc "hybridpipe.io"
)

const (
	d  string = `SMALL Message`
	jd string = `{
		"fruit": "Apple",
		"size": "Lage",
		"color": "Red"
	}`
)

// Person struct
type Person struct {
	Name    string
	Age     int
	NextGen []string
	CAge    []int
	Next    *Person
}

func produce(w *sync.WaitGroup) {

	defer w.Done()
	dc.Enable(Person{})

	/**
	// Nx - Linked with P
	Nx := Person{
		Name:    "Wasim",
		Age:     39,
		NextGen: []string{"Root", "Ponting", "Sachin"},
		CAge:    []int{26, 32, 33},
		Next:    nil,
	}
	// P - Data to sent
	P := Person{
		Name:    "David Gower",
		Age:     75,
		NextGen: []string{"Pringle", "NH Fairbrother", "Wasim"},
		CAge:    []int{45, 37, 39},
		Next:    &Nx,
	}**/

	fmt.Println ("Medium is called....")
	// N, _ := dc.Medium(dc.NATS, nil)
	// R, _ := dc.Medium(dc.RABBITMQ, nil)
	// K, _ := dc.Medium(dc.KAFKA, nil)
	A, _ := dc.Medium(dc.AMQP1, nil)

	fmt.Println("Close - Defer call is placed")
	// defer N.Close()
	// defer R.Close()
	// defer K.Close()
	defer A.Close()

	fmt.Println("Dispatch is called")
	for i := 1; i <= 10; i++ {
		// N.Distribute("Server.iLO.Low", P)
		// R.Distribute("Server.iLO.Med", jd)
		// fmt.Printf("%v", N.Get("mqconsumer", jd))
		// K.Distribute("Server.iLO.High", d)
		A.Dispatch("Server.iLO.Mod", d)
		time.Sleep(1 * time.Second)
	}
}

func main() {

	var w sync.WaitGroup
	w.Add(1)
	dc.Enable(Person{})
	go produce(&w)
	w.Wait()
}
