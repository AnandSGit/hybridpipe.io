package main

import (
	"fmt"
	dc "hybridpipe"
	"sync"
	"time"
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
	}

	N, _ := dc.Medium(dc.NATS, nil)
	R, _ := dc.Medium(dc.RABBITMQ, nil)
	K, _ := dc.Medium(dc.KAFKA, nil)
	defer N.Close()
	defer R.Close()
	defer K.Close()

	for i := 1; i <= 3; i++ {
		N.Distribute("Server.iLO.Low", P)
		R.Distribute("Server.iLO.Med", jd)
		fmt.Printf("%v", N.Get("mqconsumer", jd))
		K.Distribute("Server.iLO.High", d)
		time.Sleep(2 * time.Second)
	}
}

func main() {

	var w sync.WaitGroup
	w.Add(1)
	dc.Enable(Person{})
	go produce(&w)
	w.Wait()
}
