// Package proto contains Protocol Buffer definitions for HybridPipe.
//
// This file is used to generate Go code from the Protocol Buffer definitions.
// Run the following command to generate the code:
//
//   go generate ./proto
//
//go:generate protoc --go_out=. --go_opt=paths=source_relative message.proto
package proto
