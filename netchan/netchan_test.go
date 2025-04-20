package netchan

import (
	"reflect"
	"testing"
	"time"
)

func TestMockTransport(t *testing.T) {
	// Create two mock transports
	options := DefaultOptions()
	t1 := NewMockTransport(options)
	t2 := NewMockTransport(options)

	// Connect them
	t1.Connect(t2)

	// Set up message handlers
	msgCh := make(chan *Message, 10)
	t2.SetMessageHandler(func(msg *Message) error {
		msgCh <- msg
		return nil
	})

	// Send a message
	msg := NewMessage(MessageData, 1, FlagNone, []byte("hello"))
	if err := t1.SendMessage(msg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for the message
	select {
	case received := <-msgCh:
		if received.Type != MessageData {
			t.Errorf("Expected message type %d, got %d", MessageData, received.Type)
		}
		if received.Sequence != 1 {
			t.Errorf("Expected sequence %d, got %d", 1, received.Sequence)
		}
		if string(received.Payload) != "hello" {
			t.Errorf("Expected payload %q, got %q", "hello", string(received.Payload))
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for message")
	}

	// Close the transports
	t1.Close()
	t2.Close()
}

func TestSerializerBasicTypes(t *testing.T) {
	serializer := NewSerializer(false, 0)

	// Test serializing and deserializing basic types
	testCases := []interface{}{
		42,
		3.14,
		"hello",
		true,
		[]int{1, 2, 3},
		map[string]int{"a": 1, "b": 2},
	}

	for _, tc := range testCases {
		// Serialize
		data, flags, err := serializer.Serialize(tc)
		if err != nil {
			t.Errorf("Failed to serialize %v: %v", tc, err)
			continue
		}

		// Deserialize
		value, err := serializer.Deserialize(data, flags, reflect.TypeOf(tc))
		if err != nil {
			t.Errorf("Failed to deserialize %v: %v", tc, err)
			continue
		}

		// Compare
		if !reflect.DeepEqual(tc, value) {
			t.Errorf("Value mismatch: expected %v, got %v", tc, value)
		}
	}
}

func TestProtocolMessageEncoding(t *testing.T) {
	// Create a message
	msg := NewMessage(MessageData, 42, FlagCompressed, []byte("test data"))

	// Encode the message
	data, err := msg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	// Decode the message
	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("Failed to decode message: %v", err)
	}

	// Compare
	if decoded.Type != msg.Type {
		t.Errorf("Type mismatch: expected %d, got %d", msg.Type, decoded.Type)
	}
	if decoded.Sequence != msg.Sequence {
		t.Errorf("Sequence mismatch: expected %d, got %d", msg.Sequence, decoded.Sequence)
	}
	if decoded.Flags != msg.Flags {
		t.Errorf("Flags mismatch: expected %d, got %d", msg.Flags, decoded.Flags)
	}
	if string(decoded.Payload) != string(msg.Payload) {
		t.Errorf("Payload mismatch: expected %q, got %q", string(msg.Payload), string(decoded.Payload))
	}
}

func TestTypeInfoCompatibility(t *testing.T) {
	// Create type info for some types
	type Person struct {
		Name string
		Age  int
	}

	type Employee struct {
		Name   string
		Age    int
		Salary float64
	}

	personType := reflect.TypeOf(Person{})
	employeeType := reflect.TypeOf(Employee{})
	stringType := reflect.TypeOf("")
	intType := reflect.TypeOf(0)
	stringSliceType := reflect.TypeOf([]string{})
	intSliceType := reflect.TypeOf([]int{})

	personInfo := TypeInfoFromType(personType)
	employeeInfo := TypeInfoFromType(employeeType)
	stringInfo := TypeInfoFromType(stringType)
	intInfo := TypeInfoFromType(intType)
	stringSliceInfo := TypeInfoFromType(stringSliceType)
	intSliceInfo := TypeInfoFromType(intSliceType)

	// Test compatibility
	testCases := []struct {
		a, b     TypeInfo
		expected bool
	}{
		{stringInfo, stringInfo, true},
		{intInfo, intInfo, true},
		{stringInfo, intInfo, false},
		{personInfo, personInfo, true},
		{employeeInfo, employeeInfo, true},
		{personInfo, employeeInfo, false}, // Person doesn't have all fields of Employee
		{stringSliceInfo, stringSliceInfo, true},
		{intSliceInfo, intSliceInfo, true},
		{stringSliceInfo, intSliceInfo, false},
	}

	for i, tc := range testCases {
		result := tc.a.Compatible(tc.b)
		if result != tc.expected {
			t.Errorf("Case %d: Expected compatibility %v, got %v", i, tc.expected, result)
		}
	}
}

// TestMockNetChan tests the NetChan implementation using mock transports
func TestMockNetChan(t *testing.T) {
	// Create a mock transport factory
	transportFactory := func() (transportLayer, transportLayer) {
		t1 := NewMockTransport(DefaultOptions())
		t2 := NewMockTransport(DefaultOptions())
		t1.Connect(t2)
		return t1, t2
	}

	// Create two channels
	ch1 := Make(0)
	ch2 := Make(0)

	// Set up the transports
	t1, t2 := transportFactory()
	ch1.transport = t1
	ch2.transport = t2

	// Set up message handlers
	t1.SetMessageHandler(func(msg *Message) error {
		return ch1.handleMessage(msg)
	})
	t2.SetMessageHandler(func(msg *Message) error {
		return ch2.handleMessage(msg)
	})

	// Send a value from ch1 to ch2
	go func() {
		if err := ch1.Send(42); err != nil {
			t.Errorf("Failed to send: %v", err)
		}
	}()

	// Receive the value on ch2
	value, ok, err := ch2.Recv()
	if err != nil {
		t.Fatalf("Failed to receive: %v", err)
	}
	if !ok {
		t.Fatal("Channel closed unexpectedly")
	}
	if value != 42 {
		t.Errorf("Expected value 42, got %v", value)
	}

	// Close the channels
	ch1.Close()
	ch2.Close()
}

// TestNetChanOverTCP tests the NetChan implementation over TCP
func TestNetChanOverTCP(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping TCP test in short mode")
	}

	// Start a receiver
	receiver, err := Listen(":8080", 0, WithBufferSize(10))
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer receiver.Close()

	// Get the actual address
	addr := ":8080" // Use a fixed port for testing

	// Connect a sender
	sender, err := Dial(addr, 0, WithBufferSize(10))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer sender.Close()

	// Send a value
	go func() {
		if err := sender.Send(42); err != nil {
			t.Errorf("Failed to send: %v", err)
		}
	}()

	// Receive the value
	value, ok, err := receiver.Recv()
	if err != nil {
		t.Fatalf("Failed to receive: %v", err)
	}
	if !ok {
		t.Fatal("Channel closed unexpectedly")
	}
	if value != 42 {
		t.Errorf("Expected value 42, got %v", value)
	}
}
