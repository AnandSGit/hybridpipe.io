package netchan

import (
	"context"
	"fmt"
	"sync"
)

// ChannelWrapper provides a Go channel-like interface for network channels
type ChannelWrapper[T any] struct {
	sendChan    chan T
	recvChan    chan T
	netChan     *Chan[T]
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	initialized bool
	mutex       sync.Mutex
}

// NewChannelWrapper creates a new wrapper around a NetChan that allows using
// native Go channel syntax
func NewChannelWrapper[T any](netChan *Chan[T], bufferSize int) *ChannelWrapper[T] {
	ctx, cancel := context.WithCancel(context.Background())
	
	wrapper := &ChannelWrapper[T]{
		sendChan:    make(chan T, bufferSize),
		recvChan:    make(chan T, bufferSize),
		netChan:     netChan,
		ctx:         ctx,
		cancel:      cancel,
		initialized: false,
	}
	
	return wrapper
}

// Initialize starts the background goroutines that connect the local channels
// to the network channel
func (cw *ChannelWrapper[T]) Initialize() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	
	if cw.initialized {
		return nil
	}
	
	// Start goroutine to forward from sendChan to network
	cw.wg.Add(1)
	go func() {
		defer cw.wg.Done()
		for {
			select {
			case <-cw.ctx.Done():
				return
			case val, ok := <-cw.sendChan:
				if !ok {
					// Channel closed
					return
				}
				
				// Forward to network
				if err := cw.netChan.Send(val); err != nil {
					// Log error but continue
					fmt.Printf("Error sending to network: %v\n", err)
				}
			}
		}
	}()
	
	// Start goroutine to forward from network to recvChan
	cw.wg.Add(1)
	go func() {
		defer cw.wg.Done()
		for {
			select {
			case <-cw.ctx.Done():
				return
			default:
				// Try to receive from network
				val, ok, err := cw.netChan.Recv()
				if err != nil {
					// Log error but continue
					fmt.Printf("Error receiving from network: %v\n", err)
					continue
				}
				
				if !ok {
					// Channel closed
					close(cw.recvChan)
					return
				}
				
				// Forward to local channel
				select {
				case <-cw.ctx.Done():
					return
				case cw.recvChan <- val:
					// Successfully forwarded
				}
			}
		}
	}()
	
	cw.initialized = true
	return nil
}

// Send sends a value to the network channel using Go's channel syntax
func (cw *ChannelWrapper[T]) Send(val T) {
	cw.sendChan <- val
}

// Recv receives a value from the network channel using Go's channel syntax
func (cw *ChannelWrapper[T]) Recv() (T, bool) {
	val, ok := <-cw.recvChan
	return val, ok
}

// SendChan returns a channel that can be used to send values using the <- operator
func (cw *ChannelWrapper[T]) SendChan() chan<- T {
	return cw.sendChan
}

// RecvChan returns a channel that can be used to receive values using the <- operator
func (cw *ChannelWrapper[T]) RecvChan() <-chan T {
	return cw.recvChan
}

// Close closes the wrapper and all associated channels
func (cw *ChannelWrapper[T]) Close() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	
	if !cw.initialized {
		return nil
	}
	
	// Signal goroutines to stop
	cw.cancel()
	
	// Close the send channel
	close(cw.sendChan)
	
	// Wait for goroutines to finish
	cw.wg.Wait()
	
	// Close the underlying network channel
	return cw.netChan.Close()
}
