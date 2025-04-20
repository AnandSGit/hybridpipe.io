package hybridpipe

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Azure/go-amqp"
)

// AMQPPacket implements the HybridPipe interface for AMQP 1.0 messaging system.
type AMQPPacket struct {
	// HandleConn is the connection to the AMQP server
	HandleConn *amqp.Client
	// receivers maps pipe names to their receivers
	receivers map[string]*amqp.Receiver
	// sessions maps pipe names to their sessions
	sessions map[string]*amqp.Session
	// cancelFuncs maps pipe names to their context cancel functions
	cancelFuncs map[string]context.CancelFunc
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
}

// Connect establishes a connection to the AMQP server using the configuration in HPipeConfig.
func (ap *AMQPPacket) Connect() error {
	// Initialize maps if needed
	ap.mutex.Lock()
	if ap.receivers == nil {
		ap.receivers = make(map[string]*amqp.Receiver)
	}
	if ap.sessions == nil {
		ap.sessions = make(map[string]*amqp.Session)
	}
	if ap.cancelFuncs == nil {
		ap.cancelFuncs = make(map[string]context.CancelFunc)
	}
	ap.mutex.Unlock()

	// Connect to AMQP server if not already connected
	if ap.HandleConn == nil {
		// Set connection options
		options := []amqp.ConnOption{
			amqp.ConnIdleTimeout(30 * time.Second),
			amqp.ConnMaxFrameSize(4294967295),
		}

		// Connect to the server
		conn, err := amqp.Dial(HPipeConfig.AMQPServer, options...)
		if err != nil {
			return fmt.Errorf("failed to connect to AMQP server %s: %w", HPipeConfig.AMQPServer, err)
		}

		ap.HandleConn = conn
		log.Printf("Connected to AMQP server at %s", HPipeConfig.AMQPServer)
	}

	return nil
}

// Dispatch publishes a message to the specified pipe.
func (ap *AMQPPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection
	if ap.HandleConn == nil {
		return fmt.Errorf("not connected to AMQP server")
	}

	// Create a new session
	session, err := ap.HandleConn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create AMQP session: %w", err)
	}
	defer session.Close(context.Background())

	// Create a sender for the pipe
	sender, err := session.NewSender(
		amqp.LinkTargetAddress(pipe),
	)
	if err != nil {
		return fmt.Errorf("failed to create sender for pipe %s: %w", pipe, err)
	}
	defer sender.Close(context.Background())

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send the message
	err = sender.Send(ctx, amqp.NewMessage(bytes))
	if err != nil {
		return fmt.Errorf("failed to send message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (ap *AMQPPacket) Accept(pipe string, fn Process) error {
	// Validate connection
	if ap.HandleConn == nil {
		return fmt.Errorf("not connected to AMQP server")
	}

	// Lock for thread safety
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	// Cancel existing receiver if any
	if cancel, exists := ap.cancelFuncs[pipe]; exists {
		cancel()
		delete(ap.cancelFuncs, pipe)
	}

	// Close existing session if any
	if session, exists := ap.sessions[pipe]; exists {
		session.Close(context.Background())
		delete(ap.sessions, pipe)
	}

	// Create a new session
	session, err := ap.HandleConn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create AMQP session for pipe %s: %w", pipe, err)
	}

	// Create a receiver for the pipe
	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress(pipe),
		amqp.LinkCredit(10),
	)
	if err != nil {
		session.Close(context.Background())
		return fmt.Errorf("failed to create receiver for pipe %s: %w", pipe, err)
	}

	// Store the session and receiver
	ap.sessions[pipe] = session
	ap.receivers[pipe] = receiver

	// Create a context with cancel function for the receiver goroutine
	ctx, cancel := context.WithCancel(context.Background())
	ap.cancelFuncs[pipe] = cancel

	// Start a goroutine to handle messages
	go ap.readMessages(ctx, pipe, receiver, fn)

	log.Printf("Subscribed to AMQP pipe %s", pipe)
	return nil
}

// readMessages reads messages from AMQP in a loop until the context is canceled.
func (ap *AMQPPacket) readMessages(ctx context.Context, pipe string, receiver *amqp.Receiver, fn Process) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in AMQP reader for pipe %s: %v", pipe, r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping AMQP reader for pipe %s", pipe)
			return
		default:
			// Create a context with timeout for receiving messages
			receiveCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
			message, err := receiver.Receive(receiveCtx)
			cancel()

			// Check for context cancellation
			if ctx.Err() != nil {
				return
			}

			// Handle timeout and continue
			if err == context.DeadlineExceeded {
				continue
			}

			// Handle other errors
			if err != nil {
				log.Printf("Error receiving message from pipe %s: %v", pipe, err)
				// If the error is fatal, break the loop
				if err != context.Canceled {
					time.Sleep(1 * time.Second) // Wait before retrying
				}
				continue
			}

			// Accept the message
			message.Accept(context.Background())

			// Decode the message
			var data interface{}
			if err := Decode(message.GetData(), &data); err != nil {
				log.Printf("Failed to decode message from pipe %s: %v", pipe, err)
				continue
			}

			// Process the message
			fn(data)
		}
	}
}

// Remove unsubscribes from the specified pipe.
func (ap *AMQPPacket) Remove(pipe string) error {
	// Validate connection
	if ap.HandleConn == nil {
		return fmt.Errorf("not connected to AMQP server")
	}

	// Lock for thread safety
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	// Check if the pipe is subscribed
	receiver, exists := ap.receivers[pipe]
	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Cancel the receiver goroutine
	if cancel, exists := ap.cancelFuncs[pipe]; exists {
		cancel()
		delete(ap.cancelFuncs, pipe)
	}

	// Close the receiver
	if err := receiver.Close(context.Background()); err != nil {
		log.Printf("Error closing receiver for pipe %s: %v", pipe, err)
	}
	delete(ap.receivers, pipe)

	// Close the session
	if session, exists := ap.sessions[pipe]; exists {
		if err := session.Close(context.Background()); err != nil {
			log.Printf("Error closing session for pipe %s: %v", pipe, err)
		}
		delete(ap.sessions, pipe)
	}

	log.Printf("Unsubscribed from AMQP pipe %s", pipe)
	return nil
}

// Close terminates the connection to the AMQP server.
func (ap *AMQPPacket) Close() {
	// Check if connected
	if ap.HandleConn == nil {
		return
	}

	// Lock for thread safety
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	// Cancel all receiver goroutines
	for pipe, cancel := range ap.cancelFuncs {
		cancel()
		delete(ap.cancelFuncs, pipe)
		log.Printf("Canceled receiver for pipe %s", pipe)
	}

	// Close all receivers
	for pipe, receiver := range ap.receivers {
		if err := receiver.Close(context.Background()); err != nil {
			log.Printf("Error closing receiver for pipe %s: %v", pipe, err)
		}
	}
	ap.receivers = make(map[string]*amqp.Receiver)

	// Close all sessions
	for pipe, session := range ap.sessions {
		if err := session.Close(context.Background()); err != nil {
			log.Printf("Error closing session for pipe %s: %v", pipe, err)
		}
	}
	ap.sessions = make(map[string]*amqp.Session)

	// Close the connection
	if err := ap.HandleConn.Close(); err != nil {
		log.Printf("Error closing AMQP connection: %v", err)
	}
	ap.HandleConn = nil

	log.Printf("Closed AMQP connection")
}
