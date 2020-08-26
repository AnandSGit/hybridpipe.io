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

	/**
	go func() {
		sender, err := session.NewSender(
			amqp.LinkTargetAddress("/ServerIO"),
		)
		if err != nil {
			log.Fatal("Creating sender link:", err)
		}
		for {
			err = sender.Send(ctx, amqp.NewMessage([]byte("Hello!")))
			if err != nil {
				log.Fatal("Sending message:", err)
			}
		}
		sender.Close(ctx)
	}()
	**/

	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress("/ServerIO"),
		amqp.LinkCredit(10),
	)
	if err != nil {
		log.Fatal("Creating receiver link:", err)
	}
	ctx := context.Background()
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
