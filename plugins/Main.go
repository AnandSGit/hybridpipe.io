package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/go-amqp"
)

func main() {
	// Create client
	client, err := amqp.Dial("amqp://0.0.0.0/")
	if err != nil {
		log.Fatal("Dialing AMQP server:", err)
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		log.Fatal("Creating AMQP session:", err)
	}
	ctx := context.Background()

	/**
	go func() {
		sender, err := session.NewSender(
			amqp.LinkTargetAddress("/ServerOI"),
		)
		if err != nil {
			log.Fatal("Creating sender link:", err)
		}
		for {
			err = sender.Send(ctx, amqp.NewMessage([]byte("Hello!")))
			if err != nil {
				log.Fatal("Sending message:", err)
			} else {
				fmt.Println("Message sent successfully!")
			}
		}
		sender.Close(ctx)
	}()
	**/

	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress("/ServerOI"),
		amqp.LinkCredit(10),
	)
	if err != nil {
		log.Fatal("Creating receiver link:", err)
	}
	for {
		msg, err := receiver.Receive(ctx)
		if err != nil {
			log.Fatal("Reading message from AMQP:", err)
		}
		// Accept message
		msg.Accept(ctx)
		fmt.Printf("Message received: %s\n", msg.GetData())
	}

}
