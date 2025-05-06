// Package monitoring provides monitoring functionality for the HybridPipe system.
package monitoring

import (
	"sync"
	"sync/atomic"
	"time"
)

// MessageStats tracks message statistics for a HybridPipe implementation.
type MessageStats struct {
	// MessagesSent is the number of messages sent.
	MessagesSent uint64
	// MessagesReceived is the number of messages received.
	MessagesReceived uint64
	// BytesSent is the number of bytes sent.
	BytesSent uint64
	// BytesReceived is the number of bytes received.
	BytesReceived uint64
	// Errors is the number of errors encountered.
	Errors uint64
	// StartTime is the time when statistics collection started.
	StartTime time.Time
	// LastMessageTime is the time when the last message was sent or received.
	LastMessageTime time.Time
	// mutex protects LastMessageTime and other non-atomic fields.
	mutex sync.RWMutex
}

// NewMessageStats creates a new MessageStats instance.
func NewMessageStats() *MessageStats {
	return &MessageStats{
		StartTime:      time.Now(),
		LastMessageTime: time.Now(),
	}
}

// IncrementSent increments the number of messages sent.
func (s *MessageStats) IncrementSent(bytes uint64) {
	atomic.AddUint64(&s.MessagesSent, 1)
	atomic.AddUint64(&s.BytesSent, bytes)
	s.updateLastMessageTime()
}

// IncrementReceived increments the number of messages received.
func (s *MessageStats) IncrementReceived(bytes uint64) {
	atomic.AddUint64(&s.MessagesReceived, 1)
	atomic.AddUint64(&s.BytesReceived, bytes)
	s.updateLastMessageTime()
}

// IncrementError increments the number of errors.
func (s *MessageStats) IncrementError() {
	atomic.AddUint64(&s.Errors, 1)
}

// GetMessagesSent returns the number of messages sent.
func (s *MessageStats) GetMessagesSent() uint64 {
	return atomic.LoadUint64(&s.MessagesSent)
}

// GetMessagesReceived returns the number of messages received.
func (s *MessageStats) GetMessagesReceived() uint64 {
	return atomic.LoadUint64(&s.MessagesReceived)
}

// GetBytesSent returns the number of bytes sent.
func (s *MessageStats) GetBytesSent() uint64 {
	return atomic.LoadUint64(&s.BytesSent)
}

// GetBytesReceived returns the number of bytes received.
func (s *MessageStats) GetBytesReceived() uint64 {
	return atomic.LoadUint64(&s.BytesReceived)
}

// GetErrors returns the number of errors.
func (s *MessageStats) GetErrors() uint64 {
	return atomic.LoadUint64(&s.Errors)
}

// GetStartTime returns the time when statistics collection started.
func (s *MessageStats) GetStartTime() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.StartTime
}

// GetLastMessageTime returns the time when the last message was sent or received.
func (s *MessageStats) GetLastMessageTime() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.LastMessageTime
}

// GetUptime returns the time elapsed since statistics collection started.
func (s *MessageStats) GetUptime() time.Duration {
	return time.Since(s.GetStartTime())
}

// GetIdleTime returns the time elapsed since the last message was sent or received.
func (s *MessageStats) GetIdleTime() time.Duration {
	return time.Since(s.GetLastMessageTime())
}

// Reset resets all statistics.
func (s *MessageStats) Reset() {
	atomic.StoreUint64(&s.MessagesSent, 0)
	atomic.StoreUint64(&s.MessagesReceived, 0)
	atomic.StoreUint64(&s.BytesSent, 0)
	atomic.StoreUint64(&s.BytesReceived, 0)
	atomic.StoreUint64(&s.Errors, 0)
	
	s.mutex.Lock()
	s.StartTime = time.Now()
	s.LastMessageTime = time.Now()
	s.mutex.Unlock()
}

// updateLastMessageTime updates the last message time to the current time.
func (s *MessageStats) updateLastMessageTime() {
	s.mutex.Lock()
	s.LastMessageTime = time.Now()
	s.mutex.Unlock()
}

// GetStats returns a map of statistics.
func (s *MessageStats) GetStats() map[string]interface{} {
	uptime := s.GetUptime()
	idleTime := s.GetIdleTime()
	
	return map[string]interface{}{
		"messages_sent":     s.GetMessagesSent(),
		"messages_received": s.GetMessagesReceived(),
		"bytes_sent":        s.GetBytesSent(),
		"bytes_received":    s.GetBytesReceived(),
		"errors":            s.GetErrors(),
		"start_time":        s.GetStartTime().Format(time.RFC3339),
		"last_message_time": s.GetLastMessageTime().Format(time.RFC3339),
		"uptime_seconds":    uptime.Seconds(),
		"idle_seconds":      idleTime.Seconds(),
	}
}
