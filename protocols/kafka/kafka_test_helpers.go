package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
)

// createTempCertFiles creates temporary certificate files for testing
func createTempCertFiles(t *testing.T) (certFile, keyFile, caFile string, cleanup func()) {
	tempDir := t.TempDir()

	certFile = filepath.Join(tempDir, "cert.pem")
	keyFile = filepath.Join(tempDir, "key.pem")
	caFile = filepath.Join(tempDir, "ca.pem")

	// Create dummy certificate files
	err := os.WriteFile(certFile, []byte("DUMMY CERT"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(keyFile, []byte("DUMMY KEY"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(caFile, []byte("DUMMY CA"), 0644)
	require.NoError(t, err)

	cleanup = func() {
		// The temp dir will be cleaned up automatically by t.TempDir()
	}

	return
}

// createInvalidCertFiles creates invalid certificate files for testing
func createInvalidCertFiles(t *testing.T) (certFile, keyFile, caFile string, cleanup func()) {
	tempDir := t.TempDir()

	certFile = filepath.Join(tempDir, "cert.pem")
	keyFile = filepath.Join(tempDir, "key.pem")
	caFile = filepath.Join(tempDir, "ca.pem")

	// Create invalid certificate files
	err := os.WriteFile(certFile, []byte("INVALID CERT"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(keyFile, []byte("INVALID KEY"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(caFile, []byte("INVALID CA"), 0644)
	require.NoError(t, err)

	cleanup = func() {
		// The temp dir will be cleaned up automatically by t.TempDir()
	}

	return
}

// createNonExistentCertFiles returns paths to non-existent certificate files
func createNonExistentCertFiles(t *testing.T) (certFile, keyFile, caFile string) {
	tempDir := t.TempDir()

	certFile = filepath.Join(tempDir, "nonexistent-cert.pem")
	keyFile = filepath.Join(tempDir, "nonexistent-key.pem")
	caFile = filepath.Join(tempDir, "nonexistent-ca.pem")

	return
}

// createInvalidCAFile creates a valid cert/key but invalid CA file
func createInvalidCAFile(t *testing.T) (certFile, keyFile, caFile string, cleanup func()) {
	tempDir := t.TempDir()

	// Create a self-signed certificate for testing
	certFile = filepath.Join(tempDir, "cert.pem")
	keyFile = filepath.Join(tempDir, "key.pem")
	caFile = filepath.Join(tempDir, "ca.pem")

	// Create dummy certificate files
	err := os.WriteFile(certFile, []byte("DUMMY CERT"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(keyFile, []byte("DUMMY KEY"), 0644)
	require.NoError(t, err)

	// Create an invalid CA file
	err = os.WriteFile(caFile, []byte("INVALID CA"), 0644)
	require.NoError(t, err)

	cleanup = func() {
		// The temp dir will be cleaned up automatically by t.TempDir()
	}

	return
}

// createUnreadableCAFile creates a CA file that cannot be read
func createUnreadableCAFile(t *testing.T) (certFile, keyFile, caFile string, cleanup func()) {
	tempDir := t.TempDir()

	certFile = filepath.Join(tempDir, "cert.pem")
	keyFile = filepath.Join(tempDir, "key.pem")
	caFile = filepath.Join(tempDir, "ca.pem")

	// Create dummy certificate files
	err := os.WriteFile(certFile, []byte("DUMMY CERT"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(keyFile, []byte("DUMMY KEY"), 0644)
	require.NoError(t, err)

	// Create CA file with no read permissions
	err = os.WriteFile(caFile, []byte("UNREADABLE CA"), 0644)
	require.NoError(t, err)

	// Make the CA file unreadable
	err = os.Chmod(caFile, 0000)
	require.NoError(t, err)

	cleanup = func() {
		// Make the file readable again so it can be deleted
		os.Chmod(caFile, 0644)
	}

	return
}

// createTestPacket creates a test Packet with mocked components
func createTestPacket(t *testing.T) *Packet {
	config := &core.KafkaConfig{
		KServer:  "localhost",
		KLport:   9092,
		KTimeout: 1,
	}

	return New(config)
}

// setupConnectedPacket creates a Packet that is connected with mocked components
func setupConnectedPacket(t *testing.T) *Packet {
	packet := createTestPacket(t)

	// Set up the packet as connected
	packet.Server = "localhost:9092"
	packet.DialerConn = &kafka.Dialer{
		Timeout:   time.Second,
		DualStack: true,
	}

	return packet
}

// assertSubscriptionState checks if a subscription is properly set up
func assertSubscriptionState(t *testing.T, packet *Packet, pipe string, expectSubscribed bool) {
	packet.mutex.RLock()
	defer packet.mutex.RUnlock()

	reader, readerExists := packet.Readers[pipe]
	cancel, cancelExists := packet.readerCancel[pipe]

	if expectSubscribed {
		assert.True(t, readerExists, "Reader should exist for pipe %s", pipe)
		assert.NotNil(t, reader, "Reader should not be nil for pipe %s", pipe)
		assert.True(t, cancelExists, "Cancel function should exist for pipe %s", pipe)
		assert.NotNil(t, cancel, "Cancel function should not be nil for pipe %s", pipe)
	} else {
		assert.False(t, readerExists, "Reader should not exist for pipe %s", pipe)
		assert.False(t, cancelExists, "Cancel function should not exist for pipe %s", pipe)
	}
}

// waitForCondition waits for a condition to be true with timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Timed out waiting for condition: %s", message)
}

// mockOsHostname mocks os.Hostname for testing
func mockOsHostname(hostname string, err error) func() {
	original := osHostname
	osHostname = func() (string, error) {
		return hostname, err
	}
	return func() {
		osHostname = original
	}
}

// Variables to allow mocking
var (
	osHostname = os.Hostname
)

// mockFileReadError mocks os.ReadFile to return an error
func mockFileReadError() func() {
	original := osReadFile
	osReadFile = func(name string) ([]byte, error) {
		return nil, errors.New("mock read error")
	}
	return func() {
		osReadFile = original
	}
}

// Variables to allow mocking
var (
	osReadFile = os.ReadFile
)

// mockTLSLoadX509KeyPair mocks tls.LoadX509KeyPair to return an error
func mockTLSLoadX509KeyPair() func() {
	original := tlsLoadX509KeyPair
	tlsLoadX509KeyPair = func(certFile, keyFile string) (tls.Certificate, error) {
		return tls.Certificate{}, errors.New("mock load key pair error")
	}
	return func() {
		tlsLoadX509KeyPair = original
	}
}

// Variables to allow mocking
var (
	tlsLoadX509KeyPair = tls.LoadX509KeyPair
)

// mockAppendCertsFromPEM mocks the AppendCertsFromPEM method to return false
func mockAppendCertsFromPEM(returnValue bool) func() {
	original := appendCertsFromPEM
	appendCertsFromPEM = func(pool *x509.CertPool, pemCerts []byte) bool {
		return returnValue
	}
	return func() {
		appendCertsFromPEM = original
	}
}

// Variables to allow mocking
var (
	appendCertsFromPEM = func(pool *x509.CertPool, pemCerts []byte) bool {
		return pool.AppendCertsFromPEM(pemCerts)
	}
)

// mockGetProtocolConfig mocks core.GetProtocolConfig
func mockGetProtocolConfig(config interface{}, err error) func() {
	original := getProtocolConfig
	getProtocolConfig = func(protocol int) (interface{}, error) {
		return config, err
	}
	return func() {
		getProtocolConfig = original
	}
}

// Variables to allow mocking
var (
	getProtocolConfig = core.GetProtocolConfig
)

// mockEncode mocks core.Encode
func mockEncode(mockData []byte, err error) func() {
	original := encode
	encode = func(data interface{}) ([]byte, error) {
		return mockData, err
	}
	return func() {
		encode = original
	}
}

// Variables to allow mocking
var (
	encode = core.Encode
)

// mockDecode mocks core.Decode
func mockDecode(err error) func() {
	original := decode
	decode = func(data []byte, v interface{}) error {
		return err
	}
	return func() {
		decode = original
	}
}

// Variables to allow mocking
var (
	decode = core.Decode
)

// mockFilePermissionError mocks os.ReadFile to return a permission error
func mockFilePermissionError() func() {
	original := osReadFile
	osReadFile = func(name string) ([]byte, error) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: os.ErrPermission}
	}
	return func() {
		osReadFile = original
	}
}
