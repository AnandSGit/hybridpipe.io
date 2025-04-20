package netchan

// transportLayer defines the interface for the transport layer.
type transportLayer interface {
	// SendMessage sends a message.
	SendMessage(msg *Message) error
	// Close closes the transport.
	Close() error
	// SetMessageHandler sets the message handler.
	SetMessageHandler(handler func(*Message) error)
	// SetCloseHandler sets the close handler.
	SetCloseHandler(handler func(error))
}
