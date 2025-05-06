package core

// ProcessWrapper wraps a Process function to handle interface{} to []byte conversion.
// This is needed for backward compatibility with the old protocol implementations.
func ProcessWrapper(fn func(any)) Process {
	return func(data []byte) error {
		var decoded any
		if err := Decode(data, &decoded); err != nil {
			return err
		}
		fn(decoded)
		return nil
	}
}

// LegacyProcessWrapper wraps a Process function to handle []byte to interface{} conversion.
// This is needed for backward compatibility with the old protocol implementations.
func LegacyProcessWrapper(fn Process) func(any) {
	return func(data any) {
		// Convert data to []byte if needed
		var bytes []byte
		switch v := data.(type) {
		case []byte:
			bytes = v
		default:
			var err error
			bytes, err = Encode(v)
			if err != nil {
				return
			}
		}

		// Call the process function
		fn(bytes)
	}
}
