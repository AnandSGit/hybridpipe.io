package hybridpipe

import (
	"sync/atomic"
	"time"
)

// MessageStats tracks statistics about messages processed by HybridPipe.
type MessageStats struct {
	// MessagesSent is the number of messages sent
	MessagesSent uint64
	// MessagesReceived is the number of messages received
	MessagesReceived uint64
	// BytesSent is the number of bytes sent
	BytesSent uint64
	// BytesReceived is the number of bytes received
	BytesReceived uint64
	// LastSentTime is the time of the last sent message
	LastSentTime int64
	// LastReceivedTime is the time of the last received message
	LastReceivedTime int64
	// ErrorCount is the number of errors encountered
	ErrorCount uint64
}

// NewMessageStats creates a new MessageStats instance.
func NewMessageStats() *MessageStats {
	return &MessageStats{}
}

// IncrementSent increments the sent message counter.
func (s *MessageStats) IncrementSent(bytes uint64) {
	atomic.AddUint64(&s.MessagesSent, 1)
	atomic.AddUint64(&s.BytesSent, bytes)
	atomic.StoreInt64(&s.LastSentTime, time.Now().UnixNano())
}

// IncrementReceived increments the received message counter.
func (s *MessageStats) IncrementReceived(bytes uint64) {
	atomic.AddUint64(&s.MessagesReceived, 1)
	atomic.AddUint64(&s.BytesReceived, bytes)
	atomic.StoreInt64(&s.LastReceivedTime, time.Now().UnixNano())
}

// IncrementError increments the error counter.
func (s *MessageStats) IncrementError() {
	atomic.AddUint64(&s.ErrorCount, 1)
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

// GetLastSentTime returns the time of the last sent message.
func (s *MessageStats) GetLastSentTime() time.Time {
	return time.Unix(0, atomic.LoadInt64(&s.LastSentTime))
}

// GetLastReceivedTime returns the time of the last received message.
func (s *MessageStats) GetLastReceivedTime() time.Time {
	return time.Unix(0, atomic.LoadInt64(&s.LastReceivedTime))
}

// GetErrorCount returns the number of errors encountered.
func (s *MessageStats) GetErrorCount() uint64 {
	return atomic.LoadUint64(&s.ErrorCount)
}

// Reset resets all counters to zero.
func (s *MessageStats) Reset() {
	atomic.StoreUint64(&s.MessagesSent, 0)
	atomic.StoreUint64(&s.MessagesReceived, 0)
	atomic.StoreUint64(&s.BytesSent, 0)
	atomic.StoreUint64(&s.BytesReceived, 0)
	atomic.StoreInt64(&s.LastSentTime, 0)
	atomic.StoreInt64(&s.LastReceivedTime, 0)
	atomic.StoreUint64(&s.ErrorCount, 0)
}
