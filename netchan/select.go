package netchan

import (
	"reflect"
	"time"
)

// CaseType represents the type of a select case.
type CaseType int

const (
	// CaseSend represents a send case.
	CaseSend CaseType = iota
	// CaseRecv represents a receive case.
	CaseRecv
	// CaseTimeout represents a timeout case.
	CaseTimeout
)

// Case represents a select case.
type Case struct {
	// Type is the type of the case.
	Type CaseType
	// Chan is the channel for the case.
	Chan interface{}
	// SendValue is the value to send (for send cases).
	SendValue interface{}
	// Timeout is the timeout duration (for timeout cases).
	Timeout time.Duration
}

// SendCase creates a new select case for sending.
func SendCase[T any](ch *Chan[T], value T) Case {
	return Case{
		Type:      CaseSend,
		Chan:      ch,
		SendValue: value,
	}
}

// RecvCase creates a new select case for receiving.
func RecvCase[T any](ch *Chan[T]) Case {
	return Case{
		Type: CaseRecv,
		Chan: ch,
	}
}

// TimeoutCase creates a new select case for a timeout.
func TimeoutCase(duration time.Duration) Case {
	return Case{
		Type:    CaseTimeout,
		Timeout: duration,
	}
}

// Select performs a select operation on the given cases.
func Select(cases []Case) (chosen int, recv interface{}, recvOK bool, err error) {
	if len(cases) == 0 {
		// No cases, block forever
		<-make(chan struct{}) // This will block forever
	}

	// Create a channel for results
	type result struct {
		index  int
		value  interface{}
		ok     bool
		err    error
		chosen bool
	}
	resultCh := make(chan result, len(cases))

	// Start a goroutine for each case
	for i, c := range cases {
		go func(i int, c Case) {
			switch c.Type {
			case CaseSend:
				// Handle send case
				if ch, ok := c.Chan.(*Chan[interface{}]); ok {
					err := ch.Send(c.SendValue)
					resultCh <- result{i, nil, false, err, err == nil}
				} else {
					// Use reflection to call the Send method
					chVal := reflect.ValueOf(c.Chan)
					sendMethod := chVal.MethodByName("Send")
					if !sendMethod.IsValid() {
						resultCh <- result{i, nil, false, ErrInvalidType, false}
						return
					}

					args := []reflect.Value{reflect.ValueOf(c.SendValue)}
					ret := sendMethod.Call(args)
					if len(ret) > 0 && !ret[0].IsNil() {
						resultCh <- result{i, nil, false, ret[0].Interface().(error), false}
					} else {
						resultCh <- result{i, nil, false, nil, true}
					}
				}

			case CaseRecv:
				// Handle receive case
				if ch, ok := c.Chan.(*Chan[interface{}]); ok {
					val, ok, err := ch.Recv()
					resultCh <- result{i, val, ok, err, err == nil}
				} else {
					// Use reflection to call the Recv method
					chVal := reflect.ValueOf(c.Chan)
					recvMethod := chVal.MethodByName("Recv")
					if !recvMethod.IsValid() {
						resultCh <- result{i, nil, false, ErrInvalidType, false}
						return
					}

					ret := recvMethod.Call(nil)
					if len(ret) >= 3 && !ret[2].IsNil() {
						resultCh <- result{i, nil, false, ret[2].Interface().(error), false}
					} else if len(ret) >= 2 {
						resultCh <- result{i, ret[0].Interface(), ret[1].Bool(), nil, true}
					} else {
						resultCh <- result{i, nil, false, ErrInvalidType, false}
					}
				}

			case CaseTimeout:
				// Handle timeout case
				timer := time.NewTimer(c.Timeout)
				<-timer.C
				resultCh <- result{i, nil, false, nil, true}
			}
		}(i, c)
	}

	// Create a counter to track received results
	resultsReceived := 0

	// Process results as they arrive
	for resultsReceived < len(cases) {
		r := <-resultCh
		resultsReceived++

		if r.chosen {
			return r.index, r.value, r.ok, r.err
		}
	}

	// No case was chosen
	return -1, nil, false, ErrTimeout
}

// TrySelect performs a non-blocking select operation on the given cases.
func TrySelect(cases []Case) (chosen int, recv interface{}, recvOK bool, ok bool, err error) {
	if len(cases) == 0 {
		return -1, nil, false, false, nil
	}

	// Try each case in order
	for i, c := range cases {
		switch c.Type {
		case CaseSend:
			// Try to send
			if ch, ok := c.Chan.(*Chan[interface{}]); ok {
				ok, err := ch.TrySend(c.SendValue)
				if err != nil {
					return i, nil, false, false, err
				}
				if ok {
					return i, nil, false, true, nil
				}
			} else {
				// Use reflection to call the TrySend method
				chVal := reflect.ValueOf(c.Chan)
				trySendMethod := chVal.MethodByName("TrySend")
				if !trySendMethod.IsValid() {
					continue
				}

				args := []reflect.Value{reflect.ValueOf(c.SendValue)}
				ret := trySendMethod.Call(args)
				if len(ret) >= 2 && ret[0].Bool() {
					if !ret[1].IsNil() {
						return i, nil, false, false, ret[1].Interface().(error)
					}
					return i, nil, false, true, nil
				}
			}

		case CaseRecv:
			// Try to receive
			if ch, ok := c.Chan.(*Chan[interface{}]); ok {
				val, ok, err := ch.TryRecv()
				if err != nil {
					return i, nil, false, false, err
				}
				if ok {
					return i, val, true, true, nil
				}
			} else {
				// Use reflection to call the TryRecv method
				chVal := reflect.ValueOf(c.Chan)
				tryRecvMethod := chVal.MethodByName("TryRecv")
				if !tryRecvMethod.IsValid() {
					continue
				}

				ret := tryRecvMethod.Call(nil)
				if len(ret) >= 3 && ret[1].Bool() {
					if !ret[2].IsNil() {
						return i, nil, false, false, ret[2].Interface().(error)
					}
					return i, ret[0].Interface(), ret[1].Bool(), true, nil
				}
			}

		case CaseTimeout:
			// Timeout cases are always ready in non-blocking mode
			return i, nil, false, true, nil
		}
	}

	// No case was ready
	return -1, nil, false, false, nil
}
