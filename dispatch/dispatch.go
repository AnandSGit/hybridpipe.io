package main

import (
	"sync"

	dc "hybridpipe.io"
)

const (
	d  string = `SMALL Message`
	jd string = `{
		"fruit": "Apple",
		"size": "Large",
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
	N, _ := dc.DeployRouter(dc.NATS, nil)
	A, _ := dc.DeployRouter(dc.AMQP1, nil)
	for i := 1; i <= 500; i++ {
		A.Dispatch("ServerOI", P)
		N.Dispatch("ServerIO", jd)
	}
}

func main() {

	var w sync.WaitGroup
	w.Add(1)
	dc.Enable(Person{})
	go produce(&w)
	w.Wait()
}
